package character

import "testing"

// TestRokkaInitAnyCardAnyPoints 验证：初始化阶段任意点数（含重复）均可预定眼位，
// 能耗牌与技能牌行为一致；存满 6 眼后 activation_idx==6（大阵锁定）。
func TestRokkaInitAnyCardAnyPoints(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks == nil || def.Hooks.UseEnergyOverride == nil || def.Hooks.UseSkillOverride == nil {
		t.Fatal("rokka must have UseEnergyOverride and UseSkillOverride hooks")
	}
	if def.Hooks.PreUseEnergyCheck == nil || def.Hooks.PreUseSkillCheck == nil {
		t.Fatal("rokka must have PreUse*Check hooks")
	}

	es := map[string]any{}
	// 含重复点数（3 出现两次），且能耗/技能交替预定——均应被接受。
	points := []int{3, 3, 2, 4, 5, 1}
	for i, p := range points {
		if err := def.Hooks.PreUseEnergyCheck(p, es); err != nil {
			t.Fatalf("step %d: PreUseEnergyCheck unexpected error: %v", i, err)
		}
		if i%2 == 0 {
			if handled := def.Hooks.UseEnergyOverride(p, es); !handled {
				t.Fatalf("step %d: energy not handled", i)
			}
		} else {
			_, cost, handled := def.Hooks.UseSkillOverride(p, es)
			if !handled || cost != 0 {
				t.Fatalf("step %d: skill handled=%v cost=%d", i, handled, cost)
			}
		}
	}

	if idx := esInt(es, "rokka_activation_idx", -1); idx != 6 {
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

// TestRokkaDuplicatePointAllowed 验证：初始化阶段重复点数不再被拒绝。
func TestRokkaDuplicatePointAllowed(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{3, 5, 0, 0, 0, 0},
		"rokka_activation_idx": 2,
	}
	if err := def.Hooks.PreUseEnergyCheck(5, es); err != nil {
		t.Fatalf("duplicate point must now be allowed, got %v", err)
	}
	if err := def.Hooks.PreUseSkillCheck(3, es); err != nil {
		t.Fatalf("duplicate point must now be allowed, got %v", err)
	}
}

// TestRokkaLightThenPending 验证：锁定后点亮 3 眼进入待激活态，
// 第 3 眼不立即结算几何（待激活按钮触发）。
func TestRokkaLightThenPending(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{1, 2, 3, 4, 5, 1},
		"rokka_activation_idx": 6,
	}
	// 点亮位 0(点数1)、位 1(点数2)、位 2(点数3) → 邻接三连，但应延后结算。
	for _, p := range []int{1, 2, 3} {
		if err := def.Hooks.PreUseSkillCheck(p, es); err != nil {
			t.Fatalf("PreUseSkillCheck(%d) unexpected error: %v", p, err)
		}
		result, cost, handled := def.Hooks.UseSkillOverride(p, es)
		if !handled || cost != 0 {
			t.Fatalf("handled=%v cost=%d", handled, cost)
		}
		// 点亮过程本身不结算几何效果。
		if result.HealSelf != 0 || result.DrawCards != 0 || result.DealDirectDamage != 0 {
			t.Fatalf("lighting must not settle geometry immediately, got %+v", result)
		}
	}
	if !esBool(es, "rokka_pending_activation", false) {
		t.Fatal("after 3rd eye lit, pending_activation must be true")
	}
	lit, _ := es["rokka_lit_eyes"].([]int)
	if len(lit) != 3 {
		t.Fatalf("lit_eyes=%v, want length 3", lit)
	}
}

// TestRokkaPendingRejectsMoreCards 验证：待激活态下再出能耗/技能牌被拒绝（提示点按钮）。
func TestRokkaPendingRejectsMoreCards(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":        [6]int{1, 2, 3, 4, 5, 1},
		"rokka_activation_idx":     6,
		"rokka_lit_eyes":           []int{0, 1, 2},
		"rokka_pending_activation": true,
	}
	if err := def.Hooks.PreUseEnergyCheck(4, es); err == nil {
		t.Fatal("pending state must reject further energy cards")
	}
	if err := def.Hooks.PreUseSkillCheck(4, es); err == nil {
		t.Fatal("pending state must reject further skill cards")
	}
}

// TestRokkaResolveActivation 验证：激活钩子在待激活态结算几何并清空状态；
// 非待激活态返回 errMsg。
func TestRokkaResolveActivation(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks.ResolveFormationActivation == nil {
		t.Fatal("rokka must have ResolveFormationActivation hook")
	}

	// 非待激活态 → 报错。
	es := map[string]any{"rokka_activation_idx": 6}
	if _, errMsg := def.Hooks.ResolveFormationActivation(es); errMsg == "" {
		t.Fatal("expected errMsg when not pending")
	}

	// 待激活、点亮位 {0,2,4} → 等边三角：回血 5 + 摸 3。
	es = map[string]any{
		"rokka_eyes_points":        [6]int{1, 2, 3, 4, 5, 1},
		"rokka_activation_idx":     6,
		"rokka_lit_eyes":           []int{0, 2, 4},
		"rokka_pending_activation": true,
	}
	result, errMsg := def.Hooks.ResolveFormationActivation(es)
	if errMsg != "" {
		t.Fatalf("unexpected errMsg: %s", errMsg)
	}
	if result.HealSelf != 5 || result.DrawCards != 3 {
		t.Fatalf("equilateral → %+v, want heal=5 draw=3", result)
	}
	if esBool(es, "rokka_pending_activation", false) {
		t.Fatal("pending must be cleared after activation")
	}
	if lit, _ := es["rokka_lit_eyes"].([]int); len(lit) != 0 {
		t.Fatalf("lit_eyes must be cleared, got %v", lit)
	}
}

// TestRokkaRejectNoMatch 验证：锁定后，技能/能耗牌点数与所有未点亮眼都不匹配时 PreCheck 拒绝。
func TestRokkaRejectNoMatch(t *testing.T) {
	def := MustGet("rokka")
	es := map[string]any{
		"rokka_eyes_points":    [6]int{1, 2, 3, 4, 5, 1},
		"rokka_activation_idx": 6,
	}
	if err := def.Hooks.PreUseSkillCheck(99, es); err == nil {
		t.Fatal("expected error for no-match play")
	}
	if err := def.Hooks.PreUseEnergyCheck(99, es); err == nil {
		t.Fatal("expected error for no-match energy play")
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

// TestRokkaBuildExtraInfo 验证客户端可见字段（含 pending_activation / locked）。
func TestRokkaBuildExtraInfo(t *testing.T) {
	def := MustGet("rokka")
	if def.Hooks.BuildExtraInfo == nil {
		t.Fatal("rokka must have BuildExtraInfo")
	}
	es := map[string]any{
		"rokka_eyes_points":        [6]int{1, 2, 3, 4, 5, 1},
		"rokka_activation_idx":     6,
		"rokka_lit_eyes":           []int{1, 3},
		"rokka_pending_activation": false,
	}
	info := def.Hooks.BuildExtraInfo(es)
	if info["rokka_activation_idx"] != 6 {
		t.Errorf("activation_idx=%v", info["rokka_activation_idx"])
	}
	if !info["rokka_locked"].(bool) {
		t.Error("locked must be true")
	}
	if info["rokka_pending_activation"].(bool) {
		t.Error("pending must be false here")
	}
}
