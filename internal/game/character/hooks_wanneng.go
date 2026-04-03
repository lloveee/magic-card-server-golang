package character

func init() {
	// 万能者：所有手牌均视为攻击牌；无技能；造成累积伤害后被动逐步强化。
	registry["wanneng"] = &CharDef{
		ID: "wanneng",
		Hooks: &CharHooks{
			AllCardsAsAttack: true,

			OnDamageDealt: func(dmg int, es map[string]any) {
				cfg := HooksConfig("wanneng")
				thresholds := hcIntSlice(cfg, "phase_thresholds", []int{10, 50, 100})

				total := esInt(es, "total_damage", 0) + dmg
				es["total_damage"] = total

				phase := 0
				// 从高到低检查阈值
				for i := len(thresholds) - 1; i >= 0; i-- {
					if total >= thresholds[i] {
						phase = i + 1
						break
					}
				}
				es["phase"] = phase
			},

			ModifyCardPoints: func(pts int, es map[string]any) int {
				cfg := HooksConfig("wanneng")
				bonus := hcInt(cfg, "phase2_card_bonus", 2)
				if esInt(es, "phase", 0) >= 2 {
					return pts + bonus
				}
				return pts
			},

			ModifyOutgoingAttack: func(pts int, energy int, es map[string]any) int {
				cfg := HooksConfig("wanneng")
				phase := esInt(es, "phase", 0)
				if phase >= 1 {
					pts += hcInt(cfg, "phase1_attack_bonus", 2)
				}
				if phase >= 3 {
					pts *= hcInt(cfg, "phase3_attack_multiplier", 2)
				}
				return pts
			},
		},
	}
}
