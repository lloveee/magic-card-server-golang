package character

import "testing"

// TestRokkaInitViaEnergy 验证：6 张能耗牌依次打出后，eyes_points 与 activation_idx 正确，
// UseEnergyOverride 返回 handled=true 表示不再走默认能量增益。
func TestRokkaInitViaEnergy(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks == nil || def.Hooks.UseEnergyOverride == nil {
		t.Fatal("rokka must have UseEnergyOverride hook")
	}
	if def.Hooks.PreUseEnergyCheck == nil {
		t.Fatal("rokka must have PreUseEnergyCheck hook")
	}

	es := map[string]any{}
	points := []int{3, 5, 2, 4, 6, 1} // 6 个互不相同的点数
	for i, p := range points {
		if err := def.Hooks.PreUseEnergyCheck(p, es); err != nil {
			t.Fatalf("step %d: PreUseEnergyCheck unexpected error: %v", i, err)
		}
		handled := def.Hooks.UseEnergyOverride(p, es)
		if !handled {
			t.Fatalf("step %d: UseEnergyOverride not handled", i)
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

// TestRokkaRejectDuplicatePoint 验证：在初始化阶段，重复点数被 PreUseEnergyCheck 拒绝。
func TestRokkaRejectDuplicatePoint(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 0, 0, 0, 0},
		"rokka_activation_idx": 2,
	}
	if err := def.Hooks.PreUseEnergyCheck(5, es); err == nil {
		t.Fatal("expected error: point 5 already used")
	}
	if err := def.Hooks.PreUseEnergyCheck(3, es); err == nil {
		t.Fatal("expected error: point 3 already used")
	}
	// 未使用过的点数应当通过。
	if err := def.Hooks.PreUseEnergyCheck(7, es); err != nil {
		t.Fatalf("unexpected error for fresh point 7: %v", err)
	}
}

// TestRokkaEnergyAfterLock 验证：大阵锁定后（idx==6），能耗牌恢复正常能量增益
// （UseEnergyOverride 返回 false，PreUseEnergyCheck 不再校验唯一性）。
func TestRokkaEnergyAfterLock(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 6, 1},
		"rokka_activation_idx": 6,
	}
	if err := def.Hooks.PreUseEnergyCheck(3, es); err != nil {
		t.Fatalf("after lock, PreUseEnergyCheck must allow any point; got %v", err)
	}
	if handled := def.Hooks.UseEnergyOverride(3, es); handled {
		t.Fatal("after lock, UseEnergyOverride must return false to delegate to default gainEnergy")
	}
}

// TestRokkaSkillBeforeInit 验证：大阵未初始化完成（idx<6）时，技能牌被 PreUseSkillCheck 拒绝。
func TestRokkaSkillBeforeInit(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_activation_idx": 3,
	}
	if err := def.Hooks.PreUseSkillCheck(4, es); err == nil {
		t.Fatal("expected error: skill card cannot be played before init done")
	}
}

// TestRokkaLightOneEye 验证：大阵锁定后，打出与某眼点数匹配的技能牌点亮该眼。
func TestRokkaLightOneEye(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 6, 1},
		"rokka_activation_idx": 6,
	}
	if err := def.Hooks.PreUseSkillCheck(5, es); err != nil {
		t.Fatalf("PreUseSkillCheck unexpected error: %v", err)
	}
	result, cost, handled := def.Hooks.UseSkillOverride(5, es)
	if !handled || cost != 0 {
		t.Fatalf("handled=%v cost=%d", handled, cost)
	}
	if result.DealDirectDamage != 0 || result.HealSelf != 0 || result.DrawCards != 0 {
		t.Fatalf("result should be zero (only 1 eye lit), got %+v", result)
	}
	lit, _ := es["rokka_lit_eyes"].([]int)
	if len(lit) != 1 || lit[0] != 1 {
		t.Fatalf("lit_eyes=%v, want [1]", lit)
	}
}

// TestRokkaRejectNoMatch 验证：锁定后，技能牌点数与所有眼都不匹配时 PreUseSkillCheck 拒绝。
func TestRokkaRejectNoMatch(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks.PreUseSkillCheck == nil {
		t.Fatal("rokka must have PreUseSkillCheck")
	}
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 6, 1},
		"rokka_activation_idx": 6,
	}
	err := def.Hooks.PreUseSkillCheck(99, es)
	if err == nil {
		t.Fatal("expected error for no-match skill play")
	}
}

// TestRokkaEquilateral 验证等边三角形 {0,2,4} 与 {1,3,5}。
func TestRokkaEquilateral(t *testing.T) {
	r := rokkaEvaluateGeometry([]int{0, 2, 4})
	if r.HealSelf != 5 || r.DrawCards != 3 {
		t.Fatalf("{0,2,4} → %+v, want heal=5 draw=3", r)
	}
	r = rokkaEvaluateGeometry([]int{1, 3, 5})
	if r.HealSelf != 5 || r.DrawCards != 3 {
		t.Fatalf("{1,3,5} → %+v, want heal=5 draw=3", r)
	}
}

// TestRokkaAdjacent 验证三个相邻位（含跨界）。
func TestRokkaAdjacent(t *testing.T) {
	cases := [][]int{
		{0, 1, 2}, {1, 2, 3}, {2, 3, 4}, {3, 4, 5},
		{4, 5, 0}, {5, 0, 1},
	}
	for _, c := range cases {
		r := rokkaEvaluateGeometry(c)
		if r.DrawCards != 8 || r.HealSelf != 0 || r.DealDirectDamage != 0 {
			t.Fatalf("%v → %+v, want draw=8 only", c, r)
		}
	}
}

// TestRokkaOther 验证其他情况：造成 5 伤害 + 抽 5 牌。
func TestRokkaOther(t *testing.T) {
	cases := [][]int{
		{0, 1, 3}, {0, 2, 3}, {0, 3, 5}, {1, 2, 4},
	}
	for _, c := range cases {
		r := rokkaEvaluateGeometry(c)
		if r.DealDirectDamage != 5 || r.DrawCards != 5 {
			t.Fatalf("%v → %+v, want dmg=5 draw=5", c, r)
		}
	}
}

// TestRokkaBuildExtraInfo 验证客户端可见字段
func TestRokkaBuildExtraInfo(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks.BuildExtraInfo == nil {
		t.Fatal("rokka must have BuildExtraInfo")
	}
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 2, 4, 6, 1},
		"rokka_activation_idx": 6,
		"rokka_lit_eyes":       []int{1, 3},
	}
	info := def.Hooks.BuildExtraInfo(es)
	if info["rokka_activation_idx"] != 6 {
		t.Errorf("activation_idx=%v", info["rokka_activation_idx"])
	}
	if !info["rokka_locked"].(bool) {
		t.Error("locked must be true")
	}
}
