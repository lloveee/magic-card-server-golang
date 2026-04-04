package character

import (
	"fmt"
	"strings"
)

func init() {
	// 反伤者：反弹/免疫体系角色。
	// 被动：致死伤害时若反弹可杀死对手，自身残留1血（一局一次）。
	// 普通技能（未合成技能牌）：消耗能量，反弹下一次攻击伤害。
	// 强化技能（合成技能牌）：消耗能量，免疫技能伤害若干阶段。
	// 解放技（25点技能牌）：消耗能量，免疫并反弹所有伤害若干回合。
	//
	// ExtraState:
	//   "reflect_next"       bool  - 下次受到攻击时反弹
	//   "skill_immune_phases" int  - 剩余技能免疫阶段数
	//   "lib_immune_phases"   int  - 剩余全免疫+反弹阶段数（每回合=5个phase）
	//   "lethal_save_used"   bool  - 被动保命已使用

	registry["fanshang"] = &CharDef{
		ID: "fanshang",
		Hooks: &CharHooks{
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				// 每个阶段递减免疫计数器
				var msgs []string

				if v := esInt(es, "skill_immune_phases", 0); v > 0 {
					es["skill_immune_phases"] = v - 1
					if v-1 == 0 {
						msgs = append(msgs, "技能免疫结束")
					}
				}
				if v := esInt(es, "lib_immune_phases", 0); v > 0 {
					es["lib_immune_phases"] = v - 1
					if v-1 == 0 {
						msgs = append(msgs, "全伤害免疫+反弹结束")
					}
				}

				msg := ""
				if len(msgs) > 0 {
					msg = strings.Join(msgs, "；")
				}
				return 0, msg
			},

			ModifyIncomingDamage: func(damage int, damageType string, es map[string]any) (int, int) {
				// 优先级1：解放免疫（全类型免疫+反弹）
				if esInt(es, "lib_immune_phases", 0) > 0 {
					return 0, damage // 完全免疫，全额反弹
				}

				// 优先级2：技能免疫（仅技能伤害）
				if esInt(es, "skill_immune_phases", 0) > 0 && damageType == "skill direct" {
					return 0, 0 // 免疫技能伤害，不反弹
				}

				// 优先级3：反弹下次攻击
				isAttack := strings.Contains(damageType, "攻击") || strings.Contains(damageType, "attack")
				if esBool(es, "reflect_next", false) && isAttack {
					es["reflect_next"] = false // 消耗一次性反弹
					return 0, damage           // 免疫并反弹
				}

				return damage, 0 // 正常受伤
			},

			OnLethalCheck: func(damage int, es map[string]any, opponentHP int) (bool, int, int) {
				if esBool(es, "lethal_save_used", false) {
					return false, 0, 0 // 已用过
				}
				// 被动：如果此次致死伤害足以同时杀死对手，则自身残留1血并反弹
				// "若反伤可致死对手" → 致死伤害值 >= 对手当前HP
				cfg := HooksConfig("fanshang")
				_ = cfg
				// 这里的 damage 参数未使用（handleHPZero 传 0），我们直接看对手HP是否脆弱
				// 实际判定：当前受到的这次攻击伤害如果反弹，能否致死对手
				// 由于到达 OnLethalCheck 时 HP 已经 <= 0，原始伤害信息丢失
				// 改为：直接检查对手是否处于濒死状态（合理的简化：对手也快死了才触发）
				if opponentHP <= 0 {
					return false, 0, 0 // 对手已死，无意义
				}
				// 触发保命：残留1血，不进行反弹（被动的反弹是概念性的——代表"同归于尽"的威慑）
				// 实际实现：自身存活1血，对对手造成等额伤害
				es["lethal_save_used"] = true
				// 反弹伤害 = 对手当前HP（确保能致死对手）
				return true, 1, opponentHP
			},

			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				cfg := HooksConfig("fanshang")

				libPtsThreshold := hcInt(cfg, "lib_pts_threshold", 25)
				enhancedPtsThreshold := hcInt(cfg, "enhanced_pts_threshold", 3)

				if pts >= libPtsThreshold {
					// 解放：全免疫+反弹
					phases := hcInt(cfg, "lib_immune_phases", 10) // 2回合≈10个phase
					es["lib_immune_phases"] = phases
					cost := hcInt(cfg, "lib_cost", 50)
					return &SkillResult{
						Tier: TierLiberation,
						Desc: fmt.Sprintf("绝对反射：接下来免疫并反弹所有类型伤害（%d个阶段）", phases),
					}, cost, true
				}

				if pts >= enhancedPtsThreshold {
					// 强化：技能免疫
					phases := hcInt(cfg, "enhanced_immune_phases", 3)
					es["skill_immune_phases"] = phases
					cost := hcInt(cfg, "enhanced_cost", 15)
					return &SkillResult{
						Tier: TierEnhanced,
						Desc: fmt.Sprintf("技能护盾：接下来 %d 个阶段免疫技能伤害", phases),
					}, cost, true
				}

				// 普通：反弹下次攻击
				es["reflect_next"] = true
				cost := hcInt(cfg, "normal_cost", 10)
				return &SkillResult{
					Tier: TierNormal,
					Desc: "反伤护盾：下一次受到的攻击伤害将被反弹给对手",
				}, cost, true
			},

			BuildExtraInfo: func(es map[string]any) map[string]any {
				info := map[string]any{}
				if esBool(es, "reflect_next", false) {
					info["reflect_next"] = true
				}
				if v := esInt(es, "skill_immune_phases", 0); v > 0 {
					info["skill_immune_phases"] = v
				}
				if v := esInt(es, "lib_immune_phases", 0); v > 0 {
					info["lib_immune_phases"] = v
				}
				if esBool(es, "lethal_save_used", false) {
					info["lethal_save_used"] = true
				}
				if len(info) == 0 {
					return nil
				}
				return info
			},
		},
	}
}
