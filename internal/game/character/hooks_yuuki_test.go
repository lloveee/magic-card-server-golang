package character_test

import (
	"testing"

	"echo/internal/game/character"
)

// TestYuukiSealsByNewSuitNames 验证结城的 OnCardPlayed 钩子在收到
// 新花色名（红桃/方片/梅花/黑桃，由 card.Suit.String() 输出）+ cardType="攻击"
// 时，能正确设置对应的 seal_* 标志。
//
// 这是 Phase 1.1（重命名 card 派系→花色）的回归测试：旧版 case 用的是
// 子系名（梦境/虚幻/重组/轮回），重命名后若 case 未同步则四印将永远无法封印。
func TestYuukiSealsByNewSuitNames(t *testing.T) {
	def := character.MustGet("yuuki")
	if def.Hooks == nil || def.Hooks.OnCardPlayed == nil {
		t.Fatal("yuuki must have OnCardPlayed hook")
	}

	cases := []struct {
		suit    string
		sealKey string
	}{
		{"红桃", "seal_dream"},
		{"方片", "seal_illusion"},
		{"梅花", "seal_reconstruct"},
		{"黑桃", "seal_cycle"},
	}

	for _, c := range cases {
		t.Run(c.suit, func(t *testing.T) {
			es := map[string]any{}
			// 引擎调用形式：OnCardPlayed(cardType, points, faction, es)
			// 其中 faction 实参为 card.Suit.String()（新花色名）。
			def.Hooks.OnCardPlayed("攻击", 0, c.suit, es)

			v, ok := es[c.sealKey]
			if !ok {
				t.Fatalf("suit=%s: expected key %q in ExtraState, got keys=%v",
					c.suit, c.sealKey, keysOf(es))
			}
			b, ok := v.(bool)
			if !ok || !b {
				t.Fatalf("suit=%s: expected %q == true, got %v", c.suit, c.sealKey, v)
			}
		})
	}
}

// TestYuukiIgnoresNonAttackCardType 验证 cardType != "攻击" 时不会设置任何封印。
func TestYuukiIgnoresNonAttackCardType(t *testing.T) {
	def := character.MustGet("yuuki")
	es := map[string]any{}
	def.Hooks.OnCardPlayed("技能", 0, "红桃", es)
	if _, ok := es["seal_dream"]; ok {
		t.Fatalf("non-attack cardType should not set seal_dream; es=%v", es)
	}
}

// TestYuukiFourSealsActivateLiberation 验证四花色攻击牌全部出过后 lib_active 自动开启。
func TestYuukiFourSealsActivateLiberation(t *testing.T) {
	def := character.MustGet("yuuki")
	es := map[string]any{}
	for _, suit := range []string{"红桃", "方片", "梅花", "黑桃"} {
		def.Hooks.OnCardPlayed("攻击", 0, suit, es)
		// IsAttackUndefendable 在引擎流程里会消费 _seal_triggered；
		// 测试里手动清掉以模拟引擎已读取的状态，避免影响后续断言。
		delete(es, "_seal_triggered")
	}
	if v, _ := es["lib_active"].(bool); !v {
		t.Fatalf("after sealing all four suits, lib_active should be true; es=%v", es)
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
