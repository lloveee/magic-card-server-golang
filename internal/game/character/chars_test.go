package character_test

import (
	"testing"

	"echo/internal/game/character"
)

// TestXuemoSkillCostZero 血魔者所有技能档位的能量消耗必须为 0。
func TestXuemoSkillCostZero(t *testing.T) {
	inst, err := character.NewInstance("xuemo")
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}

	cases := []struct {
		pts     int
		label   string
	}{
		{1, "普通技能(pts=1)"},
		{3, "强化技能(pts=3)"},
		{25, "吸血激活(pts=25)"},
	}

	for _, tc := range cases {
		_, cost, err := inst.UseSkill(tc.pts)
		if err != nil {
			t.Errorf("%s: UseSkill error: %v", tc.label, err)
			continue
		}
		if cost != 0 {
			t.Errorf("%s: cost = %d, want 0", tc.label, cost)
		}
	}
}
