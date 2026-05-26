package character

import "testing"

// TestRokkaActivation 验证：6 张技能牌依次打出后，eyes_points 与 activation_idx 正确，
// 不产生技能效果（SkillResult 全零）。
func TestRokkaActivation(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks == nil || def.Hooks.UseSkillOverride == nil {
		t.Fatal("rokka must have UseSkillOverride hook")
	}

	es := map[string]any{}
	points := []int{3, 5, 2, 4, 2, 1}
	for i, p := range points {
		result, cost, handled := def.Hooks.UseSkillOverride(p, es)
		if !handled {
			t.Fatalf("step %d: not handled", i)
		}
		if cost != 0 {
			t.Fatalf("step %d: cost=%d, want 0", i, cost)
		}
		if result == nil || result.DealDirectDamage != 0 || result.HealSelf != 0 || result.DrawCards != 0 {
			t.Fatalf("step %d: result should be zero, got %+v", i, result)
		}
	}

	idx := esInt(es, "rokka_activation_idx", -1)
	if idx != 6 {
		t.Fatalf("activation_idx=%d, want 6", idx)
	}

	eyes, ok := es["rokka_eyes_points"].([6]int)
	if !ok {
		t.Fatalf("eyes_points type wrong: %T", es["rokka_eyes_points"])
	}
	for i, p := range points {
		if eyes[i] != p {
			t.Fatalf("eyes[%d]=%d, want %d", i, eyes[i], p)
		}
	}
}

// TestRokkaLightOneEye 验证：大阵锁定后，打出与某眼点数匹配的技能牌点亮该眼，
// 不产生技能效果（达到3眼前不结算）。
func TestRokkaLightOneEye(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 2, 1},
		"rokka_activation_idx": 6,
	}
	result, cost, handled := def.Hooks.UseSkillOverride(5, es)
	if !handled || cost != 0 {
		t.Fatalf("handled=%v cost=%d", handled, cost)
	}
	if result.DealDirectDamage != 0 || result.HealSelf != 0 || result.DrawCards != 0 {
		t.Fatalf("result should be zero, got %+v", result)
	}
	lit, _ := es["rokka_lit_eyes"].([]int)
	if len(lit) != 1 || lit[0] != 1 {
		t.Fatalf("lit_eyes=%v, want [1]", lit)
	}
}

// TestRokkaRejectNoMatch 验证：锁定后无匹配眼时 PreUseSkillCheck 拒绝。
func TestRokkaRejectNoMatch(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks.PreUseSkillCheck == nil {
		t.Fatal("rokka must have PreUseSkillCheck")
	}
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 2, 1},
		"rokka_activation_idx": 6,
	}
	err := def.Hooks.PreUseSkillCheck(99, es)
	if err == nil {
		t.Fatal("expected error for no-match skill play")
	}
}
