package character

import (
	"fmt"
	"sort"
)

func init() {
	// 六華：六角大阵——按技能牌使用顺序激活六眼，激活后凭点数匹配点亮三眼触发效果。
	//
	// ExtraState:
	//   rokka_eyes_points [6]int   — 六眼点数（顺时针，下标 0=正上）
	//   rokka_activation_idx int   — 已存入眼数 (0..6)；达 6 时大阵锁定
	//   rokka_lit_eyes []int       — 当前已点亮眼下标（达 3 时清空）
	registry["rokka"] = &CharDef{
		ID: "rokka",
		Hooks: &CharHooks{
			MaxHandSize: func(_ map[string]any, _ int) int {
				return 15
			},
			PreUseSkillCheck: func(pts int, es map[string]any) error {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx < 6 {
					return nil
				}
				eyes := rokkaGetEyes(es)
				lit := rokkaGetLit(es)
				if rokkaFindEye(eyes, lit, pts) < 0 {
					return fmt.Errorf("六華：没有点数为 %d 的未点亮眼", pts)
				}
				return nil
			},
			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx < 6 {
					eyes := rokkaGetEyes(es)
					eyes[idx] = pts
					es["rokka_eyes_points"] = eyes
					es["rokka_activation_idx"] = idx + 1
					return &SkillResult{Desc: "六華大阵：眼位定义"}, 0, true
				}
				eyes := rokkaGetEyes(es)
				lit := rokkaGetLit(es)
				candidate := rokkaFindEye(eyes, lit, pts)
				if candidate < 0 {
					return &SkillResult{}, 0, true
				}
				lit = append(lit, candidate)
				if len(lit) < 3 {
					es["rokka_lit_eyes"] = lit
					return &SkillResult{Desc: fmt.Sprintf("六華大阵：第 %d 眼点亮", len(lit))}, 0, true
				}
				es["rokka_lit_eyes"] = lit
				result := rokkaEvaluateGeometry(lit)
				es["rokka_lit_eyes"] = []int{}
				return result, 0, true
			},
		},
	}
}

// rokkaGetEyes 返回 ExtraState 中存储的六眼点数副本。
func rokkaGetEyes(es map[string]any) [6]int {
	if v, ok := es["rokka_eyes_points"].([6]int); ok {
		return v
	}
	return [6]int{}
}

// rokkaGetLit 返回 ExtraState 中存储的已点亮眼下标列表副本。
func rokkaGetLit(es map[string]any) []int {
	if v, ok := es["rokka_lit_eyes"].([]int); ok {
		return v
	}
	return []int{}
}

// rokkaFindEye 在 eyes 中查找点数为 pts、且不在 lit 中的最小下标，返回 -1 表示无匹配。
func rokkaFindEye(eyes [6]int, lit []int, pts int) int {
	for i := 0; i < 6; i++ {
		if eyes[i] != pts {
			continue
		}
		alreadyLit := false
		for _, l := range lit {
			if l == i {
				alreadyLit = true
				break
			}
		}
		if !alreadyLit {
			return i
		}
	}
	return -1
}

// rokkaEquilateralSets 是等边三角形眼位集合（已排序）。
var rokkaEquilateralSets = [][3]int{
	{0, 2, 4},
	{1, 3, 5},
}

// rokkaAdjacentSets 是三个相邻位（含跨界）的眼位集合（未排序，按顺序定义）。
var rokkaAdjacentSets = [][3]int{
	{0, 1, 2}, {1, 2, 3}, {2, 3, 4}, {3, 4, 5},
	{4, 5, 0}, {5, 0, 1},
}

// rokkaEvaluateGeometry 根据三个点亮眼位的几何形状返回结算结果。
//   - 等边三角形（{0,2,4} 或 {1,3,5}）：回血 5、抽 3
//   - 三连邻接（含跨界）：抽 8
//   - 其他：造成 5 点直接伤害 + 抽 5
func rokkaEvaluateGeometry(lit []int) *SkillResult {
	if len(lit) != 3 {
		return &SkillResult{}
	}
	s := append([]int(nil), lit...)
	sort.Ints(s)
	key := [3]int{s[0], s[1], s[2]}

	for _, eq := range rokkaEquilateralSets {
		if key == eq {
			return &SkillResult{
				Tier:      TierEnhanced,
				HealSelf:  5,
				DrawCards: 3,
				Desc:      "六華·等边：回复 5 血并补 3 张牌",
			}
		}
	}
	for _, adj := range rokkaAdjacentSets {
		var sa [3]int
		copy(sa[:], adj[:])
		sort.Ints(sa[:])
		if key == sa {
			return &SkillResult{
				Tier:      TierEnhanced,
				DrawCards: 8,
				Desc:      "六華·邻接：补 8 张牌",
			}
		}
	}
	return &SkillResult{
		Tier:             TierEnhanced,
		DealDirectDamage: 5,
		DrawCards:        5,
		Desc:             "六華·杂阵：造成 5 伤害并补 5 张牌",
	}
}
