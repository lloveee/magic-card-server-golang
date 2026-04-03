package character

import "fmt"

func init() {
	// 时空裂缝者：HP/能量共享，通过裂缝被动积累能量，攻击可消耗超量能量强化伤害。
	// 数值参数从 hooks_config（JSON）读取，行为逻辑保留在 Go 代码中。
	registry["liewen"] = &CharDef{
		ID: "liewen",
		Hooks: &CharHooks{
			HPEnergyShared: true,
			InitHP:         60,
			InitEnergy:     60,

			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				if phase != "action" {
					return 0, ""
				}
				rifts := esInt(es, "rifts", 0)
				if rifts == 0 {
					return 0, ""
				}
				cfg := HooksConfig("liewen")
				bonus := esInt(es, "rift_bonus", hcInt(cfg, "default_rift_bonus", 3))
				delta := rifts * bonus
				return delta, fmt.Sprintf("时空裂缝（%d条）提供 %d 点能量", rifts, delta)
			},

			OnDamageReceived: func(dmg int, es map[string]any) {
				rifts := esInt(es, "rifts", 0)
				if rifts > 0 {
					es["rifts"] = rifts - 1
				}
			},

			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				cfg := HooksConfig("liewen")
				threshold := hcInt(cfg, "enhanced_skill_pts_threshold", 3)

				if pts < threshold {
					// 普通技能：开启一道裂缝
					es["rifts"] = esInt(es, "rifts", 0) + 1
					cur := esInt(es, "rifts", 0)
					cost := hcInt(cfg, "normal_skill_cost", 15)
					return &SkillResult{
						Tier: TierNormal,
						Desc: fmt.Sprintf("开启裂缝：开启一道时空裂缝（现有 %d 条）", cur),
					}, cost, true
				}
				// 强化技能：提升裂缝产出
				defaultBonus := hcInt(cfg, "default_rift_bonus", 3)
				increment := hcInt(cfg, "rift_bonus_increment", 2)
				cur := esInt(es, "rift_bonus", defaultBonus)
				es["rift_bonus"] = cur + increment
				cost := hcInt(cfg, "enhanced_skill_cost", 30)
				return &SkillResult{
					Tier: TierEnhanced,
					Desc: fmt.Sprintf("强化裂缝：每条裂缝每阶段能量产出提升至 %d", cur+increment),
				}, cost, true
			},

			OnAttackLaunched: func(attackPoints int, energy int, es map[string]any) (int, int) {
				cfg := HooksConfig("liewen")
				threshold := hcInt(cfg, "liberation_energy_threshold", 100)
				if energy < threshold {
					return 0, 0
				}
				excess := energy - threshold
				return excess, excess
			},
		},
	}
}
