package character

import (
	"fmt"
	"sort"
)

func init() {
	// 六華：六角大阵——按能耗牌使用顺序依次定义六眼（点数必须互不相同），
	// 六眼集齐后用技能牌点亮三眼即触发几何结算效果。
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
			BuildExtraInfo: func(es map[string]any) map[string]any {
				eyes := rokkaGetEyes(es)
				idx := esInt(es, "rokka_activation_idx", 0)
				return map[string]any{
					"rokka_eyes_points":    eyes[:],
					"rokka_activation_idx": idx,
					"rokka_lit_eyes":       rokkaGetLit(es),
					"rokka_locked":         idx >= 6,
				}
			},
			BuildPublicExtra: func(es map[string]any) map[string]any {
				eyes := rokkaGetEyes(es)
				idx := esInt(es, "rokka_activation_idx", 0)
				return map[string]any{
					"rokka_eyes_points":    eyes[:],
					"rokka_activation_idx": idx,
					"rokka_lit_eyes":       rokkaGetLit(es),
					"rokka_locked":         idx >= 6,
				}
			},
			// 初始化阶段（idx<6）：能耗牌点数不得与已定义眼位重复。
			PreUseEnergyCheck: func(pts int, es map[string]any) error {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx >= 6 {
					return nil // 大阵已锁定，能耗牌恢复正常能量增益。
				}
				eyes := rokkaGetEyes(es)
				for i := 0; i < idx; i++ {
					if eyes[i] == pts {
						return fmt.Errorf("六華：点数 %d 已被第 %d 眼占用，请使用其他点数的能耗牌", pts, i+1)
					}
				}
				return nil
			},
			// 初始化阶段（idx<6）：消耗能耗牌点数定义本次眼位，不产生能量。
			// 大阵锁定后（idx==6）返回 false，让引擎按默认能量增益处理。
			UseEnergyOverride: func(pts int, es map[string]any) bool {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx >= 6 {
					return false
				}
				eyes := rokkaGetEyes(es)
				eyes[idx] = pts
				es["rokka_eyes_points"] = eyes
				es["rokka_activation_idx"] = idx + 1
				return true
			},
			// 技能牌只能在大阵锁定后使用：用于点亮眼位，结算几何效果。
			PreUseSkillCheck: func(pts int, es map[string]any) error {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx < 6 {
					return fmt.Errorf("六華大阵未初始化（已定义 %d/6 眼），请先使用能耗牌定义眼位", idx)
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
					// 理论上 PreUseSkillCheck 已拦截；此处安全兜底，返回 not handled
					// 让引擎走默认技能路径（无角色默认技能则无效果）。
					return nil, 0, false
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
