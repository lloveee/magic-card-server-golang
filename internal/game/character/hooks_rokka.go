package character

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
			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				idx := esInt(es, "rokka_activation_idx", 0)
				if idx < 6 {
					eyes := rokkaGetEyes(es)
					eyes[idx] = pts
					es["rokka_eyes_points"] = eyes
					es["rokka_activation_idx"] = idx + 1
					return &SkillResult{Desc: "六華大阵：眼位定义"}, 0, true
				}
				// TODO Task 3.3 — locked 阶段
				return &SkillResult{}, 0, true
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
