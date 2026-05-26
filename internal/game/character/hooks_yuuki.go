package character

func init() {
	// 结城：封印的魔法少女。
	// 四个封印分别对应四种花色的攻击牌，封印后该次伤害不可抵挡。
	// 四印全部封印后自动开启解放：每回合首次命中→20不可抵挡+后续命中→附加15恢复。
	//
	// ExtraState:
	//   seal_dream       bool - 红桃封印（红桃花色攻击牌）
	//   seal_illusion    bool - 方片封印（方片花色攻击牌）
	//   seal_reconstruct bool - 梅花封印（梅花花色攻击牌）
	//   seal_cycle       bool - 黑桃封印（黑桃花色攻击牌）
	//   lib_active       bool - 四印解放是否激活
	//   first_hit_used   bool - 本回合首次命中是否已用

	registry["yuuki"] = &CharDef{
		ID: "yuuki",
		Hooks: &CharHooks{
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				if phase == "action" {
					es["first_hit_used"] = false
				}
				return 0, ""
			},

			OnCardPlayed: func(cardType string, _ int, faction string, es map[string]any) {
				if cardType != "攻击" {
					return
				}
				switch faction {
				case "红桃":
					if !esBool(es, "seal_dream", false) {
						es["seal_dream"] = true
						es["_seal_triggered"] = "dream"
					}
				case "方片":
					if !esBool(es, "seal_illusion", false) {
						es["seal_illusion"] = true
						es["_seal_triggered"] = "illusion"
					}
				case "梅花":
					if !esBool(es, "seal_reconstruct", false) {
						es["seal_reconstruct"] = true
						es["_seal_triggered"] = "reconstruct"
					}
				case "黑桃":
					if !esBool(es, "seal_cycle", false) {
						es["seal_cycle"] = true
						es["_seal_triggered"] = "cycle"
					}
				}

				// 检查四印是否全部封印
				if esBool(es, "seal_dream", false) &&
					esBool(es, "seal_illusion", false) &&
					esBool(es, "seal_reconstruct", false) &&
					esBool(es, "seal_cycle", false) {
					if !esBool(es, "lib_active", false) {
						es["lib_active"] = true
					}
				}
			},

			IsAttackUndefendable: func(es map[string]any) bool {
				// 刚触发封印的攻击 → 不可抵挡
				if _, ok := es["_seal_triggered"]; ok {
					delete(es, "_seal_triggered")
					return true
				}
				// 解放激活后每回合首次攻击 → 不可抵挡
				if esBool(es, "lib_active", false) && !esBool(es, "first_hit_used", false) {
					return true
				}
				return false
			},

			ModifyOutgoingAttack: func(pts int, energy int, es map[string]any) int {
				// 解放首次命中：伤害固定20点
				if esBool(es, "lib_active", false) && !esBool(es, "first_hit_used", false) {
					return 20
				}
				return pts
			},

			OnDamageDealt: func(finalDamage int, es map[string]any) {
				if esBool(es, "lib_active", false) {
					es["first_hit_used"] = true
				}
			},

			OnDamageLanded: func(finalDamage int, es map[string]any) int {
				// 解放非首次命中：附加15点恢复
				if esBool(es, "lib_active", false) && esBool(es, "first_hit_used", true) {
					cfg := HooksConfig("yuuki")
					return hcInt(cfg, "lib_subsequent_heal", 15)
				}
				return 0
			},

			BuildExtraInfo: func(es map[string]any) map[string]any {
				info := map[string]any{}
				sealed := 0
				if esBool(es, "seal_dream", false) {
					info["seal_dream"] = true
					sealed++
				}
				if esBool(es, "seal_illusion", false) {
					info["seal_illusion"] = true
					sealed++
				}
				if esBool(es, "seal_reconstruct", false) {
					info["seal_reconstruct"] = true
					sealed++
				}
				if esBool(es, "seal_cycle", false) {
					info["seal_cycle"] = true
					sealed++
				}
				info["yuuki_sealed_count"] = sealed
				if esBool(es, "lib_active", false) {
					info["yuuki_lib_active"] = true
				}
				if len(info) == 0 {
					return nil
				}
				return info
			},

			BuildPublicExtra: func(es map[string]any) map[string]any {
				info := map[string]any{}
				sealed := 0
				if esBool(es, "seal_dream", false) {
					sealed++
				}
				if esBool(es, "seal_illusion", false) {
					sealed++
				}
				if esBool(es, "seal_reconstruct", false) {
					sealed++
				}
				if esBool(es, "seal_cycle", false) {
					sealed++
				}
				if sealed > 0 {
					info["yuuki_sealed_count"] = sealed
				}
				if esBool(es, "lib_active", false) {
					info["yuuki_lib_active"] = true
				}
				if len(info) == 0 {
					return nil
				}
				return info
			},
		},
	}
}
