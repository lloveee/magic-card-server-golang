package character

import (
	"fmt"
	"sort"
)

func init() {
	// 六華：六角大阵
	//
	// 玩法（本版改动）：
	//   1. 预定：用任意牌（能耗牌或技能牌，只看点数）依次为六眼盖印点数；
	//      点数可重复（牌面点数仅 1~5 共 5 种，6 眼必然出现重复），无类型/唯一性要求。
	//   2. 锁定：存满 6 眼后大阵锁定（activation_idx==6）。
	//   3. 点亮：再用任意牌（能耗/技能，按点数匹配某未点亮眼）点亮眼位。
	//   4. 待激活：点亮第 3 眼后进入「待激活」态（不立即结算），
	//      玩家点击「激活大阵」按钮、弃置任意一张手牌（纯触发，不结算该牌效果）后，
	//      按 3 个点亮眼位的几何形状结算效果，并清空点亮状态以便重新循环。
	//
	// ExtraState:
	//   rokka_eyes_points [6]int        — 六眼点数（顺时针，下标 0=正上）
	//   rokka_activation_idx int        — 已存入眼数 (0..6)；达 6 时大阵锁定
	//   rokka_lit_eyes []int            — 当前已点亮眼下标（达 3 进入待激活；激活后清空）
	//   rokka_pending_activation bool   — 已集齐 3 眼，等待玩家点击激活按钮
	registry["rokka"] = &CharDef{
		ID: "rokka",
		Hooks: &CharHooks{
			MaxHandSize: func(_ map[string]any, _ int) int {
				return 15
			},
			BuildExtraInfo:   rokkaBuildInfo,
			BuildPublicExtra: rokkaBuildInfo,

			// 能耗牌与技能牌共用同一套大阵校验/结算逻辑（只看点数）。
			PreUseEnergyCheck: rokkaPreCheck,
			PreUseSkillCheck:  rokkaPreCheck,

			// 能耗牌：消耗为大阵动作（预定/点亮），不产生能量。
			UseEnergyOverride: func(pts int, es map[string]any) bool {
				rokkaApply(pts, es)
				return true
			},
			// 技能牌：同样用于预定/点亮；几何结算延后到「激活大阵」按钮，
			// 因此点亮本身不产生即时效果，只回展示文案。
			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				desc := rokkaApply(pts, es)
				return &SkillResult{Desc: desc}, 0, true
			},

			// 「激活大阵」按钮：仅待激活态可用，结算几何并清空点亮状态。
			ResolveFormationActivation: func(es map[string]any) (*SkillResult, string) {
				if !esBool(es, "rokka_pending_activation", false) {
					return nil, "六華：大阵尚未集齐三眼，无法激活"
				}
				lit := rokkaGetLit(es)
				result := rokkaEvaluateGeometry(lit)
				es["rokka_lit_eyes"] = []int{}
				es["rokka_pending_activation"] = false
				return result, ""
			},
		},
	}
}

// rokkaBuildInfo 构建发送给客户端（己方 BuildExtraInfo 与对手 BuildPublicExtra 共用）的大阵状态。
func rokkaBuildInfo(es map[string]any) map[string]any {
	eyes := rokkaGetEyes(es)
	idx := esInt(es, "rokka_activation_idx", 0)
	return map[string]any{
		"rokka_eyes_points":        eyes[:],
		"rokka_activation_idx":     idx,
		"rokka_lit_eyes":           rokkaGetLit(es),
		"rokka_locked":             idx >= 6,
		"rokka_pending_activation": esBool(es, "rokka_pending_activation", false),
	}
}

// rokkaPreCheck 校验一张牌（能耗或技能）能否参与大阵；不修改状态。
//   - 初始化阶段(idx<6)：任意点数均可（用于预定眼位）。
//   - 待激活态：拒绝出牌，提示先点击「激活大阵」按钮。
//   - 锁定后(idx==6)：必须能点亮某个未点亮眼（点数匹配），否则拒绝并放回。
func rokkaPreCheck(pts int, es map[string]any) error {
	idx := esInt(es, "rokka_activation_idx", 0)
	if idx < 6 {
		return nil
	}
	if esBool(es, "rokka_pending_activation", false) {
		return fmt.Errorf("六華：已集齐三眼，请点击「激活大阵」按钮弃牌触发")
	}
	eyes := rokkaGetEyes(es)
	lit := rokkaGetLit(es)
	if rokkaFindEye(eyes, lit, pts) < 0 {
		return fmt.Errorf("六華：没有点数为 %d 的未点亮眼", pts)
	}
	return nil
}

// rokkaApply 执行预定/点亮（假定 rokkaPreCheck 已通过），返回用于展示的文案。
func rokkaApply(pts int, es map[string]any) string {
	idx := esInt(es, "rokka_activation_idx", 0)
	if idx < 6 {
		eyes := rokkaGetEyes(es)
		eyes[idx] = pts
		es["rokka_eyes_points"] = eyes
		es["rokka_activation_idx"] = idx + 1
		if idx+1 >= 6 {
			return "六華大阵：六眼集齐，大阵锁定"
		}
		return fmt.Sprintf("六華大阵：第 %d 眼已预定（点数 %d）", idx+1, pts)
	}

	// 锁定后：点亮匹配点数的未点亮眼。
	eyes := rokkaGetEyes(es)
	lit := rokkaGetLit(es)
	candidate := rokkaFindEye(eyes, lit, pts)
	if candidate < 0 {
		return "六華：无可点亮的眼" // 理论上 rokkaPreCheck 已拦截
	}
	lit = append(lit, candidate)
	es["rokka_lit_eyes"] = lit
	if len(lit) >= 3 {
		es["rokka_pending_activation"] = true
		return "六華大阵：三眼齐亮，请点击「激活大阵」弃牌触发"
	}
	return fmt.Sprintf("六華大阵：第 %d 眼点亮", len(lit))
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
