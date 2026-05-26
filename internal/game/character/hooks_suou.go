package character

// 蘇芳：30/30 HP，4 张基础手牌，无攻击则跳过清场并下回合补 +4；HP 归零进入 15s 复活对话框。
//
// ExtraState keys:
//   suou_attacked_this_phase bool  — 本回合是否出过攻击牌（每个行动阶段开始重置）
//   suou_skip_pending        bool  — 本回合清场触发了 skip_cleanup（供下回合 BonusFillDraw 读取）
//
// 注意：OnCardPlayed 的真实签名是 4 参数 (cardType, points, faction, es)，
// 我们只关心 cardType。faction 参数留作占位以匹配 hooks.go 中的定义。
func init() {
	registry["suou"] = &CharDef{
		ID: "suou",
		Hooks: &CharHooks{
			// 每个行动阶段开始：重置本回合的攻击标志。
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				if phase == "action" {
					es["suou_attacked_this_phase"] = false
				}
				return 0, ""
			},

			// 记录本回合是否出过攻击牌（OnCardPlayed 在 AllCardsAsAttack 转换前调用，
			// 所以这里读到的是原始牌型）。
			OnCardPlayed: func(cardType string, _ int, _ string, es map[string]any) {
				if cardType == "攻击" {
					es["suou_attacked_this_phase"] = true
				}
			},

			// 补牌阶段：始终补到 4 张。
			FillTargetSize: func(_ map[string]any) int {
				return 4
			},

			// 清场阶段：本回合未出过攻击牌 → 跳过清场并打上 suou_skip_pending=true，
			// 供下回合 BonusFillDraw 读取后 +4 抽牌。flag 消费在 BonusFillDraw 内执行。
			SkipCleanup: func(es map[string]any) bool {
				if esBool(es, "suou_attacked_this_phase", false) {
					return false
				}
				es["suou_skip_pending"] = true
				return true
			},

			// 补牌阶段（基础 Fill 之后）：若上回合 skip_pending=true 则额外补 4 张并消费 flag。
			BonusFillDraw: func(es map[string]any) int {
				if !esBool(es, "suou_skip_pending", false) {
					return 0
				}
				delete(es, "suou_skip_pending")
				return 4
			},

			// HP 归零：返回 (survive=true, hpAfter=0, reflectDmg=0)，把"是否真死"的判定推迟到引擎的
			// enterAwaitingRevive 流程——hooks 包没有 engine/protocol 的访问权，复活对话框由 engine
			// 在 handleHPZero 中根据角色 ID 开启。
			// 这里返回 survive=true 是必要的：让 handleHPZero 不走 triggerDeath 默认路径。
			OnLethalCheck: func(_ int, _ map[string]any, _ int) (bool, int, int) {
				return true, 0, 0
			},
		},
	}
}
