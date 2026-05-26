package character

import "testing"

// ════════════════════════════════════════════════════════════════
//  Phase 4.6 — 蘇芳钩子单元测试
// ════════════════════════════════════════════════════════════════

// TestSuouAttackResetEachPhase: action phase 开始时 suou_attacked_this_phase 被重置为 false。
func TestSuouAttackResetEachPhase(t *testing.T) {
	def := MustGet("suou")
	if def.Hooks == nil || def.Hooks.OnPhaseStart == nil {
		t.Fatal("suou must have OnPhaseStart hook")
	}
	es := map[string]any{"suou_attacked_this_phase": true}
	_, _ = def.Hooks.OnPhaseStart("action", es)
	if esBool(es, "suou_attacked_this_phase", true) {
		t.Fatalf("suou_attacked_this_phase should be reset to false at action phase, es=%v", es)
	}
}

// TestSuouSkipCleanupIfNoAttack: 本回合未出攻击 → SkipCleanup 返回 true 且写入 suou_skip_pending。
func TestSuouSkipCleanupIfNoAttack(t *testing.T) {
	def := MustGet("suou")
	if def.Hooks == nil || def.Hooks.SkipCleanup == nil {
		t.Fatal("suou must have SkipCleanup hook")
	}
	es := map[string]any{"suou_attacked_this_phase": false}
	if !def.Hooks.SkipCleanup(es) {
		t.Fatal("SkipCleanup should return true when no attack played")
	}
	if !esBool(es, "suou_skip_pending", false) {
		t.Fatal("suou_skip_pending should be set after SkipCleanup")
	}
}

// TestSuouNoSkipIfAttacked: 本回合出过攻击 → SkipCleanup 返回 false，不设置 skip_pending。
func TestSuouNoSkipIfAttacked(t *testing.T) {
	def := MustGet("suou")
	es := map[string]any{"suou_attacked_this_phase": true}
	if def.Hooks.SkipCleanup(es) {
		t.Fatal("SkipCleanup should return false when attack was played")
	}
	if esBool(es, "suou_skip_pending", false) {
		t.Fatal("suou_skip_pending should NOT be set when attacking")
	}
}

// TestSuouSkipCleanupConsumesFlag: 模拟 OnCardPlayed 攻击 → SkipCleanup 第二次调用不再 skip。
// 验证攻击 flag 被 OnCardPlayed 正确写入并影响 SkipCleanup。
func TestSuouSkipCleanupConsumesFlag(t *testing.T) {
	def := MustGet("suou")
	es := map[string]any{}
	def.Hooks.OnCardPlayed("攻击", 3, "", es)
	if !esBool(es, "suou_attacked_this_phase", false) {
		t.Fatal("OnCardPlayed(攻击) should set suou_attacked_this_phase=true")
	}
	if def.Hooks.SkipCleanup(es) {
		t.Fatal("After attack, SkipCleanup must NOT skip")
	}
}

// TestSuouBonusDrawConsumesValue: skip_pending=true → BonusFillDraw 返回 4 且消费 flag；
// 再次调用返回 0。
func TestSuouBonusDrawConsumesValue(t *testing.T) {
	def := MustGet("suou")
	if def.Hooks == nil || def.Hooks.BonusFillDraw == nil {
		t.Fatal("suou must have BonusFillDraw hook")
	}
	es := map[string]any{"suou_skip_pending": true}
	if got := def.Hooks.BonusFillDraw(es); got != 4 {
		t.Fatalf("BonusFillDraw(skip_pending=true) = %d, want 4", got)
	}
	if _, has := es["suou_skip_pending"]; has {
		t.Fatal("suou_skip_pending should be consumed after BonusFillDraw")
	}
	if got := def.Hooks.BonusFillDraw(es); got != 0 {
		t.Fatalf("BonusFillDraw(no flag) = %d, want 0", got)
	}
}

// TestSuouFillTargetSize: FillTargetSize 永远返回 4。
func TestSuouFillTargetSize(t *testing.T) {
	def := MustGet("suou")
	if def.Hooks == nil || def.Hooks.FillTargetSize == nil {
		t.Fatal("suou must have FillTargetSize hook")
	}
	if got := def.Hooks.FillTargetSize(map[string]any{}); got != 4 {
		t.Fatalf("FillTargetSize = %d, want 4", got)
	}
}

// TestSuouLethalCheckSurvivePending: OnLethalCheck 返回 (true, 0, 0)，让引擎在 handleHPZero
// 不走默认 triggerDeath，而是后续 enterAwaitingRevive。
func TestSuouLethalCheckSurvivePending(t *testing.T) {
	def := MustGet("suou")
	if def.Hooks == nil || def.Hooks.OnLethalCheck == nil {
		t.Fatal("suou must have OnLethalCheck hook")
	}
	survive, hpAfter, reflectDmg := def.Hooks.OnLethalCheck(99, map[string]any{}, 30)
	if !survive {
		t.Fatal("OnLethalCheck must return survive=true so engine routes to revive dialog")
	}
	if hpAfter != 0 {
		t.Fatalf("hpAfter = %d, want 0 (HP stays at 0 until revive resolves)", hpAfter)
	}
	if reflectDmg != 0 {
		t.Fatalf("reflectDmg = %d, want 0", reflectDmg)
	}
}
