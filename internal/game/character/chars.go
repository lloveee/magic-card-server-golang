package character

import "fmt"

// chars.go 注册全部角色。
//
// 角色设计思路：
//   - 力裁者：纯攻击，靠直接伤害技能拿下对手
//   - 镜换者：防守反击，减免被动 + 摸牌型技能
//   - 空手者：能量流，技能获取大量能量，靠解放爆发
//   - 噬渊者：生命汲取，每次技能都自己回血
//   - 灼血者：高风险高回报，攻击加成最高但 HP 最低
//   - 殉道者：濒死专精，被动能拦截一次二次死亡并自动解放

func init() {
	register(&CharDef{
		ID:           "licai",
		Name:         "力裁者",
		MaxHP:        100,
		MaxEnergy:    100,
		LibThreshold: 80,
		ManualLib:    true,
		Passive: PassiveTraits{
			BonusOutgoing: 1, // 攻击牌额外 +1 伤害
		},
		Normal: SkillDef{
			Name:       "力裁斩击",
			EnergyCost: 10,
			Result: SkillResult{
				Tier:             TierNormal,
				DealDirectDamage: 8,
				Desc:             "力裁斩击：对对手造成8点直接伤害",
			},
		},
		Enhanced: SkillDef{
			Name:       "强化裁决",
			EnergyCost: 20,
			Result: SkillResult{
				Tier:             TierEnhanced,
				DealDirectDamage: 16,
				Desc:             "强化裁决：对对手造成16点直接伤害",
			},
		},
		Lib: SkillDef{
			Name:       "绝对裁决",
			EnergyCost: 80,
			Result: SkillResult{
				Tier:             TierLiberation,
				DealDirectDamage: 30,
				HealSelf:         20,
				Desc:             "绝对裁决：造成30点直接伤害并回复20点生命",
			},
		},
	})

	register(&CharDef{
		ID:           "jinghuan",
		Name:         "镜换者",
		MaxHP:        90,
		MaxEnergy:    100,
		LibThreshold: 80,
		ManualLib:    true,
		Passive: PassiveTraits{
			IncomingReduction: 1, // 每次受到的伤害 -1
		},
		Normal: SkillDef{
			Name:       "镜像映射",
			EnergyCost: 8,
			Result: SkillResult{
				Tier:      TierNormal,
				DrawCards: 2,
				HealSelf:  5,
				Desc:      "镜像映射：摸2张牌并回复5点生命",
			},
		},
		Enhanced: SkillDef{
			Name:       "镜像反击",
			EnergyCost: 16,
			Result: SkillResult{
				Tier:             TierEnhanced,
				DrawCards:        3,
				DealDirectDamage: 10,
				Desc:             "镜像反击：摸3张牌并造成10点直接伤害",
			},
		},
		Lib: SkillDef{
			Name:       "镜换轮转",
			EnergyCost: 80,
			Result: SkillResult{
				Tier:             TierLiberation,
				DealDirectDamage: 20,
				HealSelf:         20,
				DrawCards:        2,
				Desc:             "镜换轮转：造成20点直接伤害，回复20点生命，摸2张牌",
			},
		},
	})

	register(&CharDef{
		ID:           "kongshou",
		Name:         "空手者",
		MaxHP:        95,
		MaxEnergy:    100,
		LibThreshold: 60,
		ManualLib:    true,
		Passive:      PassiveTraits{}, // 无被动，靠技能补偿
		Normal: SkillDef{
			Name:       "虚拳引气",
			EnergyCost: 5,
			Result: SkillResult{
				Tier:       TierNormal,
				GainEnergy: 20,
				Desc:       "虚拳引气：获得20点能量",
			},
		},
		Enhanced: SkillDef{
			Name:       "引气冲拳",
			EnergyCost: 10,
			Result: SkillResult{
				Tier:             TierEnhanced,
				GainEnergy:       30,
				DealDirectDamage: 8,
				Desc:             "引气冲拳：获得30点能量并造成8点直接伤害",
			},
		},
		Lib: SkillDef{
			Name:       "空手相搏",
			EnergyCost: 60,
			Result: SkillResult{
				Tier:             TierLiberation,
				DealDirectDamage: 20,
				HealSelf:         20,
				GainEnergy:       20,
				Desc:             "空手相搏：造成20点直接伤害，回复20点生命，获得20点能量",
			},
		},
	})

	register(&CharDef{
		ID:           "shiyuan",
		Name:         "噬渊者",
		MaxHP:        95,
		MaxEnergy:    100,
		LibThreshold: 80,
		ManualLib:    true,
		Passive:      PassiveTraits{}, // 无被动，技能内置汲取
		Normal: SkillDef{
			Name:       "噬渊之触",
			EnergyCost: 10,
			Result: SkillResult{
				Tier:             TierNormal,
				DealDirectDamage: 6,
				HealSelf:         6,
				Desc:             "噬渊之触：造成6点直接伤害并汲取6点生命",
			},
		},
		Enhanced: SkillDef{
			Name:       "深渊噬魂",
			EnergyCost: 20,
			Result: SkillResult{
				Tier:             TierEnhanced,
				DealDirectDamage: 14,
				HealSelf:         10,
				Desc:             "深渊噬魂：造成14点直接伤害并汲取10点生命",
			},
		},
		Lib: SkillDef{
			Name:       "渊噬万物",
			EnergyCost: 80,
			Result: SkillResult{
				Tier:             TierLiberation,
				DealDirectDamage: 28,
				HealSelf:         20,
				Desc:             "渊噬万物：造成28点直接伤害并汲取20点生命",
			},
		},
	})

	register(&CharDef{
		ID:           "zhuoxue",
		Name:         "灼血者",
		MaxHP:        85,
		MaxEnergy:    100,
		LibThreshold: 80,
		ManualLib:    true,
		Passive: PassiveTraits{
			BonusOutgoing: 2, // 攻击牌额外 +2 伤害（最高加成）
		},
		Normal: SkillDef{
			Name:       "灼血冲击",
			EnergyCost: 10,
			Result: SkillResult{
				Tier:             TierNormal,
				DealDirectDamage: 10,
				Desc:             "灼血冲击：造成10点直接伤害",
			},
		},
		Enhanced: SkillDef{
			Name:       "烈焰灼血",
			EnergyCost: 20,
			Result: SkillResult{
				Tier:             TierEnhanced,
				DealDirectDamage: 20,
				Desc:             "烈焰灼血：造成20点直接伤害",
			},
		},
		Lib: SkillDef{
			Name:       "血焰爆发",
			EnergyCost: 80,
			Result: SkillResult{
				Tier:             TierLiberation,
				DealDirectDamage: 40,
				Desc:             "血焰爆发：造成40点直接伤害",
			},
		},
	})

	// ── 三个新角色 ───────────────────────────────────────────────

	// 时空裂缝者：HP/能量共享，通过裂缝被动积累能量，攻击可消耗超量能量强化伤害
	register(&CharDef{
		ID:           "liewen",
		Name:         "时空裂缝者",
		MaxHP:        150,
		MaxEnergy:    150,
		LibThreshold: 0,     // 无传统解放按钮，解放逻辑在 OnAttackLaunched 中处理
		ManualLib:    false,
		Passive:      PassiveTraits{},
		Normal:       SkillDef{}, // 技能通过 UseSkillOverride 处理
		Enhanced:     SkillDef{},
		Lib:          SkillDef{},
		Hooks: &CharHooks{
			HPEnergyShared: true,
			InitHP:         60,
			InitEnergy:     60,
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				// 每个阶段开始时，每条裂缝提供 rift_bonus 点能量
				rifts := esInt(es, "rifts", 0)
				if rifts == 0 {
					return 0, ""
				}
				bonus := esInt(es, "rift_bonus", 3)
				delta := rifts * bonus
				return delta, fmt.Sprintf("时空裂缝（%d条）提供 %d 点能量", rifts, delta)
			},
			OnDamageReceived: func(dmg int, es map[string]any) {
				// 每次受到伤害封印一道裂缝
				rifts := esInt(es, "rifts", 0)
				if rifts > 0 {
					es["rifts"] = rifts - 1
				}
			},
			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				if pts <= 2 {
					// 普通技能：消耗 15 能量，开启一道裂缝
					es["rifts"] = esInt(es, "rifts", 0) + 1
					cur := esInt(es, "rifts", 0)
					return &SkillResult{
						Tier: TierNormal,
						Desc: fmt.Sprintf("开启裂缝：开启一道时空裂缝（现有 %d 条）", cur),
					}, 15, true
				}
				// 强化技能：消耗 30 能量，每条裂缝每阶段能量产出 +2
				cur := esInt(es, "rift_bonus", 3)
				es["rift_bonus"] = cur + 2
				return &SkillResult{
					Tier: TierEnhanced,
					Desc: fmt.Sprintf("强化裂缝：每条裂缝每阶段能量产出提升至 %d", cur+2),
				}, 30, true
			},
			OnAttackLaunched: func(attackPoints int, energy int, es map[string]any) (int, int) {
				// 解放：攻击时若能量 ≥ 100，自动消耗超出 100 的部分强化伤害
				if energy < 100 {
					return 0, 0
				}
				excess := energy - 100
				return excess, excess
			},
		},
	})

	// 万能者：所有手牌均视为攻击牌；无技能；造成累积伤害后被动逐步强化
	register(&CharDef{
		ID:           "wanneng",
		Name:         "万能者",
		MaxHP:        80,
		MaxEnergy:    100,
		LibThreshold: 0,
		ManualLib:    false,
		Passive:      PassiveTraits{},
		Normal:       SkillDef{},
		Enhanced:     SkillDef{},
		Lib:          SkillDef{},
		Hooks: &CharHooks{
			AllCardsAsAttack: true,
			OnDamageDealt: func(dmg int, es map[string]any) {
				// 累积伤害里程碑解锁被动强化（阶段 0→1→2→3）
				total := esInt(es, "total_damage", 0) + dmg
				es["total_damage"] = total
				phase := 0
				switch {
				case total >= 100:
					phase = 3
				case total >= 50:
					phase = 2
				case total >= 10:
					phase = 1
				}
				es["phase"] = phase
			},
			ModifyCardPoints: func(pts int, es map[string]any) int {
				// 阶段 2+：攻击牌面点数 +2
				if esInt(es, "phase", 0) >= 2 {
					return pts + 2
				}
				return pts
			},
			ModifyOutgoingAttack: func(pts int, energy int, es map[string]any) int {
				// 阶段 1+：结算总和 +2；阶段 3+：结算总和 *2
				phase := esInt(es, "phase", 0)
				if phase >= 1 {
					pts += 2
				}
				if phase >= 3 {
					pts *= 2
				}
				return pts
			},
		},
	})

	// 血魔：自伤换资源；被累积伤害 50 后攻击牌面 +3；25 点技能激活吸血被动
	register(&CharDef{
		ID:           "xuemo",
		Name:         "血魔",
		MaxHP:        90,
		MaxEnergy:    100,
		LibThreshold: 0,
		ManualLib:    false,
		Passive:      PassiveTraits{},
		Normal:       SkillDef{},
		Enhanced:     SkillDef{},
		Lib:          SkillDef{},
		Hooks: &CharHooks{
			OnDamageReceived: func(dmg int, es map[string]any) {
				// 累积受到的伤害，达到 50 后开启攻击被动
				es["dmg_received"] = esInt(es, "dmg_received", 0) + dmg
			},
			ModifyCardPoints: func(pts int, es map[string]any) int {
				// 累积受到伤害 ≥ 50：攻击牌面点数 +3
				if esInt(es, "dmg_received", 0) >= 50 {
					return pts + 3
				}
				return pts
			},
			ModifyOutgoingAttack: func(pts int, energy int, es map[string]any) int {
				// 消耗"下次攻击加成"（血誓技能）
				bonus := esInt(es, "next_atk_bonus", 0)
				if bonus > 0 {
					pts += bonus
					es["next_atk_bonus"] = 0
				}
				return pts
			},
			OnDamageLanded: func(dmg int, es map[string]any) int {
				// 吸血被动：单次伤害超过 30 点时获得 100% 吸血（仅触发一次激活）
				if !esBool(es, "lifesteal", false) {
					return 0
				}
				if dmg > 30 {
					return dmg
				}
				return 0
			},
			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				if pts >= 25 {
					// 25 点技能：激活吸血被动（仅一次）
					if esBool(es, "lifesteal", false) {
						// 已激活，回退到强化技能逻辑
						return nil, 0, false
					}
					es["lifesteal"] = true
					return &SkillResult{
						Tier: TierEnhanced,
						Desc: "血魔啸：吸血被动已激活（单次伤害 >30 时获得 100% 吸血）",
					}, 0, true
				}
				if pts >= 3 {
					// 强化技能：自伤 20，下次攻击 +10
					es["next_atk_bonus"] = esInt(es, "next_atk_bonus", 0) + 10
					return &SkillResult{
						Tier:       TierEnhanced,
						DamageSelf: 20,
						Desc:       "血誓：自身受到 20 点伤害，下次攻击总和 +10",
					}, 0, true
				}
				// 普通技能：自伤 10，摸 2 张牌
				return &SkillResult{
					Tier:       TierNormal,
					DamageSelf: 10,
					DrawCards:  2,
					Desc:       "血祭：自身受到 10 点伤害，摸 2 张牌",
				}, 0, true
			},
		},
	})

	register(&CharDef{
		ID:           "xundao",
		Name:         "殉道者",
		MaxHP:        110,
		MaxEnergy:    100,
		LibThreshold: 80,
		ManualLib:    false, // 解放由引擎在二次死亡时自动触发
		Passive: PassiveTraits{
			InterceptNearDeath: true, // 被动：在二次死亡时自动触发解放（每局一次）
		},
		Normal: SkillDef{
			Name:       "殉道之愿",
			EnergyCost: 8,
			Result: SkillResult{
				Tier:     TierNormal,
				HealSelf: 10,
				Desc:     "殉道之愿：回复10点生命",
			},
		},
		Enhanced: SkillDef{
			Name:       "殉道之力",
			EnergyCost: 16,
			Result: SkillResult{
				Tier:       TierEnhanced,
				HealSelf:   20,
				GainEnergy: 10,
				Desc:       "殉道之力：回复20点生命并获得10点能量",
			},
		},
		Lib: SkillDef{
			Name:       "殉道解放",
			EnergyCost: 0, // 自动触发，不消耗能量
			Result: SkillResult{
				Tier:     TierLiberation,
				HealSelf: 30, // 引擎另外将 HP 设为60，此处额外回复由 applySkillResult 处理
				Desc:     "殉道解放：自动触发，濒死时回复30点生命并获得10点能量",
				GainEnergy: 10,
			},
		},
	})
}
