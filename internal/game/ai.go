package game

import (
	"log/slog"
	"time"

	"echo/internal/game/card"
	"echo/internal/protocol"
)

// ════════════════════════════════════════════════════════════════
//  AI 决策系统 — GOAP 评分驱动
//
//  架构：
//    1. buildAICtx   → 量化当前局面（HP、能量、手牌成分、合成机会）
//    2. scoreActions → 枚举并打分所有合法行动（解放/技能/合成/移牌/出牌/结束）
//    3. runAITurn    → 贪心迭代：每步选最高分行动，直到选出"结束"
//    4. runAIDefense → 防御阶段独立评估，权衡机会成本与安全
//
//  角色感知：
//    每个角色的策略权重通过 charID 和 ExtraState 在 scoreXxx 函数中体现，
//    无需外部配置，直接对号入座。
//
//  并发安全：
//    所有方法均在 Engine 的单一 goroutine 中同步调用，无需加锁。
//    AI 直接调用 handle* 内部处理函数，绕过 actionCh。
// ════════════════════════════════════════════════════════════════

// ────────────────────────────────────────────────────────────────
//  行动枚举
// ────────────────────────────────────────────────────────────────

type aiActionKind int

const (
	aiActEndTurn    aiActionKind = iota // 宣告结束行动（保底）
	aiActLiberate                       // 触发解放技能
	aiActUseSkill                       // 使用技能牌
	aiActSynthesize                     // 合成两张牌
	aiActMoveToSynth                    // 将手牌移入合成区
	aiActPlayCard                       // 出攻击牌或能耗牌
)

// aiAct 描述一个具体可执行的 AI 行动及其评分。
type aiAct struct {
	kind  aiActionKind
	zone  string // 主牌区域（"hand" / "synth"）
	slot  int    // 主牌槽位
	zone2 string // 合成第二张牌的区域
	slot2 int    // 合成第二张牌的槽位
	score float64
	desc  string
}

// ────────────────────────────────────────────────────────────────
//  局面上下文
// ────────────────────────────────────────────────────────────────

// cardEntry 是带有出牌有效伤害估算的手牌条目。
type cardEntry struct {
	zone   string
	slot   int
	c      *card.Card
	effPts int // 作为攻击牌出时预期造成的有效伤害（含被动、场地加成）
}

// synthOp 描述一次可执行的合成操作。
type synthOp struct {
	zone1, zone2 string
	slot1, slot2 int
	resultPts    int
	resultType   card.CardType
	gain         int // resultPts - max(input1.Points, input2.Points)，正值才有意义
}

// aiCtx 是 AI 每次决策时的局面快照，避免重复遍历手牌。
type aiCtx struct {
	seat, oppSeat int
	me, opp       *PlayerState

	charID       string  // AI 角色 ID
	passiveBonus int     // BonusOutgoing 被动加成（已含赐福角色）
	myHPRatio    float64 // me.HP / me.MaxHP
	oppHPRatio   float64 // opp.HP / opp.MaxHP
	myEnergy     int
	myLibThresh  int
	canLiberate  bool // ManualLib 且能量足够且未用过

	attackCards []cardEntry // 所有攻击牌（含万能者）
	energyCards []cardEntry // 所有能耗牌
	skillCards  []cardEntry // 所有技能牌（手牌区）
	synthCards  []cardEntry // 合成区中的牌（不含攻击类，供防御评估）

	synthOps []synthOp // 当前可立即执行的合成操作
}

// ────────────────────────────────────────────────────────────────
//  引擎入口
// ────────────────────────────────────────────────────────────────

// maybeRunAIAction 在行动阶段主循环的每次迭代中调用。
// 返回 true 表示 AI 已处理本次迭代（调用方应 continue）。
func (e *Engine) maybeRunAIAction() bool {
	if e.aiSeat < 0 {
		return false
	}
	aiP := e.state.Players[e.aiSeat]
	if aiP.ActionDone {
		return false
	}

	// 情形 A：防御窗口期，AI 是防御方
	if e.state.PendingAttack != nil {
		defSeat := 1 - e.state.PendingAttack.AttackerSeat
		if defSeat == e.aiSeat {
			e.runAIDefense()
			return true
		}
		// 防御窗口期，AI 是攻击方 → 等待人类防御
		return false
	}

	// 情形 B：行动阶段轮到 AI
	if e.state.ActiveSeat == e.aiSeat {
		e.runAITurn()
		return true
	}

	return false
}

// ────────────────────────────────────────────────────────────────
//  构建局面上下文
// ────────────────────────────────────────────────────────────────

// buildAICtx 扫描手牌并构建 AI 决策所需的局面快照。
func (e *Engine) buildAICtx(seat int) aiCtx {
	me := e.state.Players[seat]
	opp := e.state.Players[1-seat]

	ctx := aiCtx{
		seat:        seat,
		oppSeat:     1 - seat,
		me:          me,
		opp:         opp,
		myHPRatio:   float64(me.HP) / float64(me.MaxHP),
		oppHPRatio:  float64(opp.HP) / float64(opp.MaxHP),
		myEnergy:    me.Energy,
		myLibThresh: me.LibThreshold,
	}

	if me.Char != nil {
		ctx.charID = me.Char.Def.ID
		ctx.canLiberate = me.Char.Def.ManualLib && me.Char.CanLiberate(me.Energy)
		// passiveBonus 用于合成评分中估算攻击结果的有效伤害
		ctx.passiveBonus = me.Char.Def.Passive.BonusOutgoing
		if me.SecondChar != nil {
			ctx.passiveBonus += me.SecondChar.Def.Passive.BonusOutgoing
		}
	}

	allAsAttack := me.Char != nil && me.Char.Def.Hooks != nil && me.Char.Def.Hooks.AllCardsAsAttack

	// 手牌区
	for s := 1; s <= card.HandZoneSize; s++ {
		c := me.Hand.HandCard(s)
		if c == nil {
			continue
		}
		eff := e.aiEffPts(seat, c)
		entry := cardEntry{zone: "hand", slot: s, c: c, effPts: eff}
		switch {
		case c.CardType == card.TypeAttack || allAsAttack:
			ctx.attackCards = append(ctx.attackCards, entry)
		case c.CardType == card.TypeSkill:
			ctx.skillCards = append(ctx.skillCards, entry)
		case c.CardType == card.TypeEnergy:
			ctx.energyCards = append(ctx.energyCards, entry)
		}
	}

	// 合成区
	for s := 1; s <= card.SynthZoneSize; s++ {
		c := me.Hand.SynthCard(s)
		if c == nil {
			continue
		}
		eff := e.aiEffPts(seat, c)
		entry := cardEntry{zone: "synth", slot: s, c: c, effPts: eff}
		if c.CardType == card.TypeAttack || allAsAttack {
			ctx.attackCards = append(ctx.attackCards, entry)
		} else {
			ctx.synthCards = append(ctx.synthCards, entry)
		}
	}

	// 合成机会
	ctx.synthOps = e.findSynthOps(seat)
	return ctx
}

// aiEffPts 估算一张牌作为攻击牌出时的有效伤害，近似引擎实际计算流程。
// 含：ModifyCardPoints → BonusOutgoing → 场地加成 → ModifyOutgoingAttack。
func (e *Engine) aiEffPts(seat int, c *card.Card) int {
	pts := c.Points
	p := e.state.Players[seat]
	if p.Char == nil {
		return pts
	}
	if p.Char.Def.Hooks != nil && p.Char.Def.Hooks.ModifyCardPoints != nil {
		pts = p.Char.Def.Hooks.ModifyCardPoints(pts, p.Char.ExtraState)
	}
	pts = e.applyOutgoing(p, pts)
	if e.state.FieldEffect != nil {
		pts += e.state.FieldEffect.BonusAttack
	}
	if p.Char.Def.Hooks != nil && p.Char.Def.Hooks.ModifyOutgoingAttack != nil {
		pts = p.Char.Def.Hooks.ModifyOutgoingAttack(pts, p.Energy, p.Char.ExtraState)
	}
	return pts
}

// findSynthOps 枚举当前可直接执行的合成对：合成区×合成区、合成区×手牌区。
// 不含需要先 MoveToSynth 的手牌×手牌（那由 aiActMoveToSynth 配合下一轮处理）。
func (e *Engine) findSynthOps(seat int) []synthOp {
	me := e.state.Players[seat]
	opts := e.fieldSynthOpts()
	ops := make([]synthOp, 0, 8)

	synthSlots := me.Hand.SynthSlottedCards()
	handSlots := me.Hand.HandSlottedCards()

	// 合成区 × 合成区
	for i := 0; i < len(synthSlots); i++ {
		for j := i + 1; j < len(synthSlots); j++ {
			a, b := synthSlots[i], synthSlots[j]
			aiTryPair(a.Card, a.Slot, "synth", b.Card, b.Slot, "synth", opts, &ops)
		}
	}

	// 合成区 × 手牌区
	for _, sc := range synthSlots {
		for _, hc := range handSlots {
			if sc.Card.Synthesized || hc.Card.Synthesized {
				continue
			}
			aiTryPair(sc.Card, sc.Slot, "synth", hc.Card, hc.Slot, "hand", opts, &ops)
		}
	}

	return ops
}

// aiTryPair 尝试将两张牌合成，结果比输入更好时追加到 ops。
// 同时尝试正反两个方向（决定结果牌的功能类型）。
func aiTryPair(a *card.Card, aSlot int, aZone string,
	b *card.Card, bSlot int, bZone string,
	opts card.SynthesisOpts, ops *[]synthOp) {

	if a.Synthesized || b.Synthesized {
		return
	}
	maxInput := aiMax(a.Points, b.Points)

	// a 作 base
	if r, err := card.Combine(a, b, opts); err == nil && r.Points > maxInput {
		*ops = append(*ops, synthOp{
			zone1: aZone, slot1: aSlot,
			zone2: bZone, slot2: bSlot,
			resultPts: r.Points, resultType: r.CardType,
			gain: r.Points - maxInput,
		})
	}
	// b 作 base（结果类型不同时才追加，避免重复）
	if r2, err2 := card.Combine(b, a, opts); err2 == nil && r2.Points > maxInput && r2.CardType != a.CardType {
		*ops = append(*ops, synthOp{
			zone1: bZone, slot1: bSlot,
			zone2: aZone, slot2: aSlot,
			resultPts: r2.Points, resultType: r2.CardType,
			gain: r2.Points - maxInput,
		})
	}
}

// ────────────────────────────────────────────────────────────────
//  行动打分
// ────────────────────────────────────────────────────────────────

// scoreActions 枚举所有合法行动并打分，返回按分数降序排列的列表。
func (e *Engine) scoreActions(ctx aiCtx) []aiAct {
	acts := make([]aiAct, 0, 24)
	me := ctx.me
	opp := ctx.opp

	// ── 解放 ─────────────────────────────────────────────────────
	if ctx.canLiberate {
		libDmg := me.Char.Def.Lib.Result.DealDirectDamage
		libHeal := me.Char.Def.Lib.Result.HealSelf
		var s float64
		if libDmg >= opp.HP {
			s = 10000 // 解放必杀
		} else {
			s = float64(libDmg)*3.0 + float64(libHeal)*(2.0-ctx.myHPRatio)
			if float64(me.Energy) > float64(me.LibThreshold)*1.25 {
				s += 15 // 能量已大幅溢出，及早释放
			}
		}
		acts = append(acts, aiAct{kind: aiActLiberate, score: s, desc: "解放"})
	}

	// ── 技能 ─────────────────────────────────────────────────────
	for _, sk := range ctx.skillCards {
		s := e.scoreSkill(ctx, sk)
		if s > 0 {
			acts = append(acts, aiAct{
				kind: aiActUseSkill, slot: sk.slot,
				score: s, desc: "技能 " + sk.c.String(),
			})
		}
	}

	// ── 合成 ─────────────────────────────────────────────────────
	for _, op := range ctx.synthOps {
		s := e.scoreSynth(ctx, op)
		if s > 0 {
			acts = append(acts, aiAct{
				kind:  aiActSynthesize,
				zone:  op.zone1, slot: op.slot1,
				zone2: op.zone2, slot2: op.slot2,
				score: s, desc: "合成",
			})
		}
	}

	// ── 移入合成区 ────────────────────────────────────────────────
	if me.Hand.SynthCount() < card.SynthZoneSize {
		for s := 1; s <= card.HandZoneSize; s++ {
			c := me.Hand.HandCard(s)
			if c == nil || c.Synthesized {
				continue
			}
			sc := e.scoreMoveToSynth(ctx, s, c)
			if sc > 0 {
				acts = append(acts, aiAct{
					kind:  aiActMoveToSynth,
					slot:  s,
					score: sc, desc: "移入合成区",
				})
			}
		}
	}

	// ── 攻击牌 ───────────────────────────────────────────────────
	for _, ac := range ctx.attackCards {
		s := e.scoreAttack(ctx, ac)
		acts = append(acts, aiAct{
			kind: aiActPlayCard, zone: ac.zone, slot: ac.slot,
			score: s, desc: "出牌 " + ac.c.String(),
		})
	}

	// ── 能耗牌 ───────────────────────────────────────────────────
	for _, ec := range ctx.energyCards {
		s := e.scoreEnergy(ctx, ec)
		if s > 0 {
			acts = append(acts, aiAct{
				kind: aiActPlayCard, zone: ec.zone, slot: ec.slot,
				score: s, desc: "能耗牌 " + ec.c.String(),
			})
		}
	}

	// ── 结束行动（保底，分值 0）─────────────────────────────────────
	acts = append(acts, aiAct{kind: aiActEndTurn, score: 0, desc: "结束行动"})

	// 降序排列（简单插入排序，列表极短）
	for i := 1; i < len(acts); i++ {
		for j := i; j > 0 && acts[j].score > acts[j-1].score; j-- {
			acts[j], acts[j-1] = acts[j-1], acts[j]
		}
	}
	return acts
}

// scoreAttack 对打出一张攻击牌打分。
func (e *Engine) scoreAttack(ctx aiCtx, ac cardEntry) float64 {
	dmg := ac.effPts
	opp := ctx.opp

	score := float64(dmg) * 2.5

	// 致命奖励
	if dmg >= opp.HP {
		score += 5000
	}

	// 伤害占对手 HP 的比例
	score += float64(dmg) / float64(opp.MaxHP) * 20

	// 角色特殊加成 ─────────────────────────────────────────────
	switch ctx.charID {
	case "liewen":
		// 时空裂缝者：超量能量会转为攻击加成，当前能量越多越值得打
		if ctx.myEnergy >= 100 {
			excess := ctx.myEnergy - 100
			score += float64(excess) * 1.2
		}
	case "wanneng":
		// 万能者：打出后累积伤害，接近里程碑时加分
		if ctx.me.Char != nil {
			total := aiEsInt(ctx.me.Char.ExtraState, "total_damage")
			if next := wannengNextMilestone(total); next > 0 && total+dmg >= next {
				score += 30 // 里程碑解锁被动强化
			}
		}
	}

	// 己方 HP 危急时激进打法，争取速胜
	if ctx.myHPRatio < 0.3 {
		score *= 1.25
	}

	return score
}

// scoreEnergy 对打出一张能耗牌打分。
func (e *Engine) scoreEnergy(ctx aiCtx, ec cardEntry) float64 {
	me := ctx.me
	energyGain := ec.c.Points

	// 已能解放，不再蓄力
	if ctx.canLiberate {
		return -1
	}

	// 无标准解放阈值的角色
	if ctx.myLibThresh <= 0 {
		if ctx.charID == "liewen" {
			// 时空裂缝者：积累到 100 能量触发攻击加成
			if ctx.myEnergy >= 100 {
				return 0 // 已超量，不再囤积
			}
			deficit := 100 - ctx.myEnergy
			score := float64(energyGain) / float64(deficit) * 12.0
			if ctx.myEnergy+energyGain >= 100 {
				score += 15
			}
			return score
		}
		// wanneng / xuemo 等：能耗牌作为攻击处理，不走此分支
		return 0
	}

	// 标准解放流程
	deficit := ctx.myLibThresh - ctx.myEnergy
	if deficit <= 0 {
		return 0
	}

	score := float64(energyGain) / float64(deficit) * 15.0
	if me.Energy+energyGain >= ctx.myLibThresh {
		score += 20 // 能够踩线解放
	}

	// 出此能耗牌是否能解锁某个当前负担不起的技能
	for _, sk := range ctx.skillCards {
		cost := e.aiSkillCost(me, sk.c.Points)
		if cost > 0 && ctx.myEnergy < cost && ctx.myEnergy+energyGain >= cost {
			score += 8 // 解锁技能行动
			break
		}
	}

	// 空手者：蓄能是核心策略
	if ctx.charID == "kongshou" {
		score *= 1.4
	}

	return score
}

// scoreSkill 对使用一张技能牌打分。
func (e *Engine) scoreSkill(ctx aiCtx, sk cardEntry) float64 {
	me := ctx.me
	opp := ctx.opp

	if me.Char == nil {
		return 0
	}

	pts := sk.c.Points

	// UseSkillOverride 角色（时空裂缝者 / 血魔）特殊处理
	if me.Char.Def.Hooks != nil && me.Char.Def.Hooks.UseSkillOverride != nil {
		return e.scoreSpecialSkill(ctx, pts)
	}

	// 标准技能：先检查能量
	var cost, dmg, heal, draw, eGain, selfDmg int
	if pts <= 2 {
		cost = me.Char.Def.Normal.EnergyCost
		dmg = me.Char.Def.Normal.Result.DealDirectDamage
		heal = me.Char.Def.Normal.Result.HealSelf
		draw = me.Char.Def.Normal.Result.DrawCards
		eGain = me.Char.Def.Normal.Result.GainEnergy
		selfDmg = me.Char.Def.Normal.Result.DamageSelf
	} else {
		cost = me.Char.Def.Enhanced.EnergyCost
		dmg = me.Char.Def.Enhanced.Result.DealDirectDamage
		heal = me.Char.Def.Enhanced.Result.HealSelf
		draw = me.Char.Def.Enhanced.Result.DrawCards
		eGain = me.Char.Def.Enhanced.Result.GainEnergy
		selfDmg = me.Char.Def.Enhanced.Result.DamageSelf
	}

	if me.Energy < cost {
		return 0 // 能量不足，跳过
	}

	score := 0.0

	// 直接伤害
	if dmg > 0 {
		score += float64(dmg) * 2.0
		if dmg >= opp.HP {
			score += 5000 // 技能 KO
		}
	}

	// 治疗（HP 越低，回血价值越高）
	if heal > 0 {
		score += float64(heal) * (2.0 - ctx.myHPRatio)
	}

	// 摸牌（手牌匮乏时价值倍增）
	if draw > 0 {
		perCard := 2.0
		if me.Hand.HandCount() <= 3 {
			perCard = 5.0
		}
		score += float64(draw) * perCard
	}

	// 能量获取（相对于解放缺口评分）
	if eGain > 0 && ctx.myLibThresh > 0 {
		deficit := float64(ctx.myLibThresh - me.Energy)
		if deficit > 0 {
			score += float64(eGain) / deficit * 12.0
		}
	}

	// 自伤（减分，HP 越低越危险）
	if selfDmg > 0 {
		score -= float64(selfDmg) * (2.0 - ctx.myHPRatio)
	}

	// 角色专属调整 ─────────────────────────────────────────────
	switch ctx.charID {
	case "xundao":
		// 殉道者：低 HP 时强化技能（回血+能量）价值倍增
		if ctx.myHPRatio < 0.5 && pts >= 3 {
			score *= 2.0
		}
	case "shiyuan":
		// 噬渊者：伤害+吸血组合天然优先于纯攻击
		if dmg > 0 && heal > 0 {
			score *= 1.3
		}
	case "jinghuan":
		// 镜换者：手牌少时摸牌+回血技能极为宝贵
		if draw > 0 && me.Hand.HandCount() <= 4 {
			score *= 1.4
		}
	case "kongshou":
		// 空手者：技能能量收益是核心，大幅加权
		if eGain > 0 {
			score *= 1.5
		}
	}

	return score
}

// scoreSpecialSkill 打分 UseSkillOverride 角色（时空裂缝者 / 血魔）。
func (e *Engine) scoreSpecialSkill(ctx aiCtx, pts int) float64 {
	me := ctx.me

	switch ctx.charID {
	case "liewen":
		rifts := aiEsInt(me.Char.ExtraState, "rifts")
		riftBonus := aiEsInt(me.Char.ExtraState, "rift_bonus")
		if riftBonus == 0 {
			riftBonus = 3
		}
		if pts <= 2 {
			// 开一条裂缝：消耗 15 能量，长期每阶段产生额外能量
			if ctx.myEnergy < 15 {
				return 0
			}
			// 新裂缝每阶段贡献 riftBonus 能量，乘以预期剩余阶段数估算价值
			return float64((rifts+1)*riftBonus) * 2.0
		}
		// 强化裂缝产出：消耗 30 能量，每条裂缝每阶段 +2
		if ctx.myEnergy < 30 {
			return 0
		}
		return float64(rifts*2) * 3.0

	case "xuemo":
		if pts >= 25 {
			// 激活吸血被动：仅当攻击手段足够强且 HP 安全时才值得
			if !aiEsBool(me.Char.ExtraState, "lifesteal", false) && ctx.myHPRatio > 0.4 {
				maxAtk := 0
				for _, ac := range ctx.attackCards {
					if ac.effPts > maxAtk {
						maxAtk = ac.effPts
					}
				}
				if maxAtk > 30 {
					return 200 // 高攻击 + 吸血 = 复合收益
				}
			}
			return 0
		}
		if pts >= 3 {
			// 血誓：自伤 20，下次攻击 +10
			if float64(me.HP-20)/float64(me.MaxHP) < 0.2 {
				return 0 // HP 太低，不赌
			}
			return 15 + 20.0 - float64(20)*(2.0-ctx.myHPRatio)
		}
		// 血祭：自伤 10，摸 2 张
		if float64(me.HP-10)/float64(me.MaxHP) < 0.15 {
			return 0
		}
		selfPenalty := float64(10) * (2.0 - ctx.myHPRatio)
		drawVal := 6.0
		if me.Hand.HandCount() <= 3 {
			drawVal = 10.0
		}
		return drawVal - selfPenalty
	}

	return 0
}

// scoreSynth 对一次合成操作打分。
func (e *Engine) scoreSynth(ctx aiCtx, op synthOp) float64 {
	opp := ctx.opp

	// 合成结果不优于原材料，不做
	if op.gain <= 0 {
		return 0
	}

	// 已有致命攻击牌时不浪费时间合成
	for _, ac := range ctx.attackCards {
		if ac.effPts >= opp.HP {
			return 0
		}
	}

	score := float64(op.gain) * 3.0

	switch op.resultType {
	case card.TypeAttack:
		// 攻击合成：估算有效伤害，检查是否形成 KO
		effResult := op.resultPts + ctx.passiveBonus
		if e.state.FieldEffect != nil {
			effResult += e.state.FieldEffect.BonusAttack
		}
		score += float64(effResult) * 1.5
		if effResult >= opp.HP {
			score += 500 // 合成后可以 KO
		}
		if ctx.charID == "wanneng" {
			score *= 1.4 // 万能者：攻击合成价值最高
		}
	case card.TypeSkill:
		// 技能合成：高点数意味着可触发强化技能
		if ctx.me.Char != nil && op.resultPts >= 3 {
			cost := ctx.me.Char.Def.Enhanced.EnergyCost
			if cost > 0 && ctx.myEnergy >= cost {
				score += float64(ctx.me.Char.Def.Enhanced.Result.DealDirectDamage) * 0.6
			}
		}
	case card.TypeEnergy:
		// 能耗合成：仅在蓄力路线下有价值
		if ctx.myLibThresh > 0 && !ctx.canLiberate {
			deficit := ctx.myLibThresh - ctx.myEnergy
			if deficit > 0 {
				score += float64(op.resultPts) / float64(deficit) * 10.0
			}
		}
	}

	return score
}

// scoreMoveToSynth 评估把手牌槽 handSlot 的牌移入合成区是否有价值。
func (e *Engine) scoreMoveToSynth(ctx aiCtx, handSlot int, c *card.Card) float64 {
	me := ctx.me
	opts := e.fieldSynthOpts()
	maxGain := 0

	// 与合成区现有牌的合成潜力
	for s := 1; s <= card.SynthZoneSize; s++ {
		sc := me.Hand.SynthCard(s)
		if sc == nil || sc.Synthesized {
			continue
		}
		if r, err := card.Combine(c, sc, opts); err == nil {
			if g := r.Points - aiMax(c.Points, sc.Points); g > maxGain {
				maxGain = g
			}
		}
		if r, err := card.Combine(sc, c, opts); err == nil {
			if g := r.Points - aiMax(c.Points, sc.Points); g > maxGain {
				maxGain = g
			}
		}
	}

	// 与手牌区其他牌的合成潜力（两步链：本次 move，下轮 synth）
	for s := 1; s <= card.HandZoneSize; s++ {
		if s == handSlot {
			continue
		}
		hc := me.Hand.HandCard(s)
		if hc == nil || hc.Synthesized {
			continue
		}
		if r, err := card.Combine(c, hc, opts); err == nil {
			if g := r.Points - aiMax(c.Points, hc.Points); g > maxGain {
				maxGain = g
			}
		}
		if r, err := card.Combine(hc, c, opts); err == nil {
			if g := r.Points - aiMax(c.Points, hc.Points); g > maxGain {
				maxGain = g
			}
		}
	}

	if maxGain > 0 {
		return float64(maxGain) * 2.8
	}

	// 合成区为空时，低点数牌可预置，等待互补材料（保守策略）
	if me.Hand.SynthCount() == 0 && c.Points <= 2 && !c.Synthesized {
		return 1.5
	}

	return 0
}

// ────────────────────────────────────────────────────────────────
//  主行动循环
// ────────────────────────────────────────────────────────────────

// runAITurn 贪心迭代执行行动，每轮选择得分最高的行动直到选出"结束"。
func (e *Engine) runAITurn() {
	time.Sleep(600 * time.Millisecond) // 初始思考延迟

	seat := e.aiSeat
	const maxActions = 10 // 防止无限循环

	for i := 0; i < maxActions; i++ {
		if e.state.isOver() {
			return
		}

		ctx := e.buildAICtx(seat)
		acts := e.scoreActions(ctx)
		if len(acts) == 0 {
			break
		}

		best := acts[0]
		slog.Info("AI chose action", "seat", seat, "action", best.desc, "score", int(best.score))

		switch best.kind {

		case aiActEndTurn:
			e.aiEndAction(seat)
			return

		case aiActLiberate:
			e.handleTriggerLiberate(seat)
			time.Sleep(300 * time.Millisecond)
			if e.state.isOver() {
				return
			}

		case aiActUseSkill:
			payload := protocol.MustEncode(protocol.UseSkillReq{SkillCardSlot: best.slot})
			e.handleUseSkill(seat, payload)
			time.Sleep(300 * time.Millisecond)
			if e.state.isOver() {
				return
			}

		case aiActSynthesize:
			payload := protocol.MustEncode(protocol.SynthesizeReq{
				Zone1: best.zone, Slot1: best.slot,
				Zone2: best.zone2, Slot2: best.slot2,
			})
			e.handleSynthesize(seat, payload)
			time.Sleep(200 * time.Millisecond)

		case aiActMoveToSynth:
			payload := protocol.MustEncode(protocol.MoveToSynthReq{HandSlot: best.slot})
			e.handleMoveToSynth(seat, payload)
			time.Sleep(150 * time.Millisecond)

		case aiActPlayCard:
			payload := protocol.MustEncode(protocol.PlayCardReq{Zone: best.zone, Slot: best.slot})
			e.handlePlayCard(seat, payload)
			time.Sleep(300 * time.Millisecond)
			if e.state.PendingAttack != nil {
				// 攻击牌触发防御窗口，暂停 AI 行动等待对手响应
				return
			}
			if e.state.isOver() {
				return
			}
		}
	}

	// 行动轮次耗尽，强制结束
	e.aiEndAction(seat)
}

// ────────────────────────────────────────────────────────────────
//  防御决策
// ────────────────────────────────────────────────────────────────

// runAIDefense 权衡机会成本与安全，选择最优防御牌或放弃防御。
//
// 决策维度：
//  1. 不防御是否致命 → 致命时必须防御，优先用低价值牌
//  2. 完全格挡的浪费程度 → 浪费越多扣分越多
//  3. 防御牌的进攻机会成本 → 高价值攻击牌不应轻易用来防御
//  4. 当前 HP 状况 → HP 越低，防御价值越高
func (e *Engine) runAIDefense() {
	time.Sleep(400 * time.Millisecond)

	seat := e.aiSeat
	pending := e.state.PendingAttack
	atkPts := pending.AttackPoints
	me := e.state.Players[seat]
	opp := e.state.Players[1-seat]

	// 不防御时实际受到的伤害（含减伤被动）
	incoming := atkPts
	if me.Char != nil {
		incoming -= me.Char.Def.Passive.IncomingReduction
		if incoming < 1 {
			incoming = 1
		}
	}
	wouldDie := me.HP-incoming <= 0

	type candidate struct {
		zone  string
		slot  int
		c     *card.Card
		score float64
	}
	var cands []candidate

	evalCard := func(zone string, slot int, c *card.Card) {
		if c == nil {
			return
		}
		blocked := aiMin(c.Points, atkPts)
		score := float64(blocked) * 2.0

		// 致命时必须防御，优先用低价值牌（点数越低越省资源）
		if wouldDie && blocked > 0 {
			score += 8000 - float64(c.Points)*0.2
		}

		// 完全格挡奖励，但浪费越多扣分越重
		if c.Points >= atkPts {
			wastage := c.Points - atkPts
			score += 8 - float64(wastage)*1.5
		}

		// 机会成本：此牌作为攻击牌的价值
		allAsAttack := me.Char != nil && me.Char.Def.Hooks != nil && me.Char.Def.Hooks.AllCardsAsAttack
		if c.CardType == card.TypeAttack || allAsAttack {
			effAtkPts := e.aiEffPts(seat, c)
			atkValue := float64(effAtkPts) * 2.5
			if effAtkPts >= opp.HP {
				atkValue += 2000 // 这张牌能 KO 对手，绝不轻易用来防御
			}
			if !wouldDie {
				score -= atkValue * 0.55
			}
		}

		// HP 越低，防御价值越高
		hpFactor := 1.0 + (1.0 - float64(me.HP)/float64(me.MaxHP))
		score *= hpFactor

		if score > 0 {
			cands = append(cands, candidate{zone, slot, c, score})
		}
	}

	for s := 1; s <= card.HandZoneSize; s++ {
		evalCard("hand", s, me.Hand.HandCard(s))
	}
	for s := 1; s <= card.SynthZoneSize; s++ {
		evalCard("synth", s, me.Hand.SynthCard(s))
	}

	if len(cands) == 0 {
		slog.Info("AI passing defense (no cards)", "seat", seat)
		e.handleDefenseAction(seat, protocol.MustEncode(protocol.DefenseReq{Pass: true}))
		return
	}

	best := cands[0]
	for _, cd := range cands[1:] {
		if cd.score > best.score {
			best = cd
		}
	}

	// 最优牌评分为负（机会成本太高）且不是致命危机 → 放弃防御
	if best.score <= 0 && !wouldDie {
		slog.Info("AI passing defense (not worth it)", "seat", seat, "score", int(best.score))
		e.handleDefenseAction(seat, protocol.MustEncode(protocol.DefenseReq{Pass: true}))
		return
	}

	slog.Info("AI defending", "seat", seat,
		"atkPts", atkPts, "defPts", best.c.Points,
		"zone", best.zone, "slot", best.slot, "score", int(best.score))
	e.handleDefenseAction(seat, protocol.MustEncode(protocol.DefenseReq{
		Zone: best.zone, Slot: best.slot,
	}))
}

// ────────────────────────────────────────────────────────────────
//  结束行动
// ────────────────────────────────────────────────────────────────

// aiEndAction 宣告 AI 结束行动，将行动权切换给对手（若对手未结束）。
func (e *Engine) aiEndAction(seat int) {
	p := e.state.Players[seat]
	p.ActionDone = true
	slog.Info("AI ended action", "seat", seat, "round", e.state.Round)

	opp := e.state.Players[1-seat]
	if !opp.ActionDone {
		e.state.ActiveSeat = 1 - seat
	}
	e.broadcastState("ai ended action")
}

// ────────────────────────────────────────────────────────────────
//  辅助函数
// ────────────────────────────────────────────────────────────────

// aiSkillCost 返回使用 pts 点技能牌所需的能量消耗（0 = 无法使用）。
func (e *Engine) aiSkillCost(p *PlayerState, pts int) int {
	if p.Char == nil {
		return 0
	}
	// UseSkillOverride 角色（时空裂缝者/血魔）的固定成本
	if p.Char.Def.Hooks != nil && p.Char.Def.Hooks.UseSkillOverride != nil {
		// 血魔者的技能无能量消耗
		if p.Char.Def.ID == "xuemo" {
			return 0
		}
		// 时空裂缝者的固定成本
		if pts <= 2 {
			return 15
		}
		return 30
	}
	if pts <= 2 {
		return p.Char.Def.Normal.EnergyCost
	}
	return p.Char.Def.Enhanced.EnergyCost
}

func aiMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func aiMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// aiEsInt 从角色 ExtraState 读取 int 值，键不存在时返回 0。
func aiEsInt(es map[string]any, key string) int {
	if es == nil {
		return 0
	}
	v, ok := es[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

// aiEsBool 从角色 ExtraState 读取 bool 值，键不存在时返回 def。
func aiEsBool(es map[string]any, key string, def bool) bool {
	if es == nil {
		return def
	}
	v, ok := es[key]
	if !ok {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

// wannengNextMilestone 返回万能者累积伤害的下一个里程碑（0 = 已全部解锁）。
func wannengNextMilestone(total int) int {
	switch {
	case total < 10:
		return 10
	case total < 50:
		return 50
	case total < 100:
		return 100
	default:
		return 0
	}
}
