package character

func init() {
	// 血魔：自伤换资源；被累积伤害 50 后攻击牌面 +3；25 点技能激活吸血被动。
	registry["xuemo"] = &CharDef{
		ID: "xuemo",
		Hooks: &CharHooks{
			OnDamageReceived: func(dmg int, es map[string]any) {
				es["dmg_received"] = esInt(es, "dmg_received", 0) + dmg
			},

			ModifyCardPoints: func(pts int, es map[string]any) int {
				cfg := HooksConfig("xuemo")
				threshold := hcInt(cfg, "dmg_received_threshold", 50)
				bonus := hcInt(cfg, "dmg_received_card_bonus", 3)
				if esInt(es, "dmg_received", 0) >= threshold {
					return pts + bonus
				}
				return pts
			},

			ModifyOutgoingAttack: func(pts int, energy int, es map[string]any) int {
				bonus := esInt(es, "next_atk_bonus", 0)
				if bonus > 0 {
					pts += bonus
					es["next_atk_bonus"] = 0
				}
				return pts
			},

			OnDamageLanded: func(dmg int, es map[string]any) int {
				cfg := HooksConfig("xuemo")
				threshold := hcInt(cfg, "lifesteal_damage_threshold", 30)
				if !esBool(es, "lifesteal", false) {
					return 0
				}
				if dmg > threshold {
					return dmg
				}
				return 0
			},

			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				cfg := HooksConfig("xuemo")
				activatePts := hcInt(cfg, "lifesteal_activate_pts", 25)
				enhancedPts := hcInt(cfg, "enhanced_pts_threshold", 3)

				if pts >= activatePts {
					if esBool(es, "lifesteal", false) {
						return nil, 0, false
					}
					es["lifesteal"] = true
					return &SkillResult{
						Tier: TierEnhanced,
						Desc: "血魔啸：吸血被动已激活（单次伤害 >30 时获得 100% 吸血）",
					}, 0, true
				}
				if pts >= enhancedPts {
					atkBonus := hcInt(cfg, "enhanced_atk_bonus", 10)
					selfDmg := hcInt(cfg, "enhanced_self_damage", 20)
					es["next_atk_bonus"] = esInt(es, "next_atk_bonus", 0) + atkBonus
					return &SkillResult{
						Tier:       TierEnhanced,
						DamageSelf: selfDmg,
						Desc:       "血誓：自身受到 20 点伤害，下次攻击总和 +10",
					}, 0, true
				}
				// 普通技能
				selfDmg := hcInt(cfg, "normal_self_damage", 10)
				draw := hcInt(cfg, "normal_draw_cards", 2)
				return &SkillResult{
					Tier:       TierNormal,
					DamageSelf: selfDmg,
					DrawCards:  draw,
					Desc:       "血祭：自身受到 10 点伤害，摸 2 张牌",
				}, 0, true
			},
		},
	}
}
