package character

import "fmt"

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

// rokkaEvaluateGeometry 占位，Task 3.4 实现真正逻辑
func rokkaEvaluateGeometry(_ []int) *SkillResult {
	return &SkillResult{Desc: "六華大阵：三眼结算 (TODO Task 3.4)"}
}
