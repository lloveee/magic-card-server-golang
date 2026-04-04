package character

import (
	"fmt"
	"math"
)

func init() {
	// 建造者：召唤小人盖房子的经营型角色。
	// 被动：技能牌点数 = 能量消耗。根据房子数量解锁新被动。
	//   阶段一(>1房子)：小人工作效率 +2
	//   阶段二(>2房子)：受伤 -2
	//   阶段三(>=3房子)：每回合回血 +5
	//   受到单次 >20 伤害：房子数量减半（向上取整）
	// 普通技能：按技能牌点数召唤等量小人，消耗等量能量。
	// 解放技（>=5房子，20-25点技能牌）：扣除所有房子，小人翻倍+效率翻倍。
	//
	// ExtraState:
	//   "workers"        int - 小人数量
	//   "build_progress" int - 当前房子进度(0-99)
	//   "houses"         int - 已完成房子数
	//   "eff_mult"       int - 效率倍率(默认1，解放后2)

	registry["jianzao"] = &CharDef{
		ID: "jianzao",
		Hooks: &CharHooks{
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				if phase != "action" {
					return 0, ""
				}
				cfg := HooksConfig("jianzao")
				workers := esInt(es, "workers", 0)
				if workers == 0 {
					return 0, ""
				}

				houses := esInt(es, "houses", 0)
				progress := esInt(es, "build_progress", 0)
				effMult := esInt(es, "eff_mult", 1)

				// 计算每工人效率：基础1 + 房子奖励
				baseEff := hcInt(cfg, "base_worker_eff", 1)
				if houses > 1 {
					baseEff += hcInt(cfg, "house1_eff_bonus", 2)
				}
				totalWork := workers * baseEff * effMult
				progress += totalWork

				// 检查是否完成新房子
				newHouses := 0
				for progress >= 100 {
					progress -= 100
					houses++
					newHouses++
				}
				es["build_progress"] = progress
				es["houses"] = houses

				// 阶段三被动：>=3 房子回血
				healDelta := 0
				if houses >= 3 {
					healDelta = hcInt(cfg, "house3_heal", 5)
				}

				var msg string
				if newHouses > 0 {
					msg = fmt.Sprintf("小人建造报告：%d工人产出%d进度，完成%d栋新房（共%d栋），剩余进度%d/100",
						workers, totalWork, newHouses, houses, progress)
				} else {
					msg = fmt.Sprintf("小人建造报告：%d工人产出%d进度（%d/100），共%d栋房子",
						workers, totalWork, progress, houses)
				}

				// healDelta 通过返回的 energyDelta 间接实现不方便（那是能量不是HP）
				// 回血需要通过 SkillResult 或直接 ExtraState 标记，由引擎处理
				// 简化方案：将 heal 存入 ExtraState，引擎在 callPhaseStartHooks 后处理
				if healDelta > 0 {
					es["pending_heal"] = healDelta
					msg += fmt.Sprintf("，回复%d血", healDelta)
				}

				return 0, msg
			},

			ModifyIncomingDamage: func(damage int, damageType string, es map[string]any) (int, int) {
				cfg := HooksConfig("jianzao")
				houses := esInt(es, "houses", 0)
				finalDamage := damage

				// 阶段二被动：>2 房子减伤
				if houses > 2 {
					reduction := hcInt(cfg, "house2_dmg_reduction", 2)
					finalDamage -= reduction
					if finalDamage < 1 {
						finalDamage = 1
					}
				}

				return finalDamage, 0
			},

			OnDamageReceived: func(finalDamage int, es map[string]any) {
				cfg := HooksConfig("jianzao")
				threshold := hcInt(cfg, "damage_halve_threshold", 20)
				houses := esInt(es, "houses", 0)

				// 受到单次 > threshold 伤害：房子减半（向上取整）
				if finalDamage > threshold && houses > 0 {
					es["houses"] = int(math.Ceil(float64(houses) / 2.0))
				}
			},

			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				cfg := HooksConfig("jianzao")

				houses := esInt(es, "houses", 0)
				libHouseThreshold := hcInt(cfg, "lib_house_threshold", 5)
				libPtsMin := hcInt(cfg, "lib_pts_min", 20)
				libPtsMax := hcInt(cfg, "lib_pts_max", 25)

				// 解放判定：房子够 + 点数在范围内
				if houses >= libHouseThreshold && pts >= libPtsMin && pts <= libPtsMax {
					workers := esInt(es, "workers", 0)
					// 扣除所有房子
					sacrificed := houses
					es["houses"] = 0
					es["build_progress"] = 0
					// 翻倍工人和效率
					es["workers"] = workers * 2
					es["eff_mult"] = esInt(es, "eff_mult", 1) * 2
					cost := pts // 能量消耗 = 点数
					return &SkillResult{
						Tier: TierLiberation,
						Desc: fmt.Sprintf("建造者解放：献祭%d栋房子，小人翻倍至%d，工作效率翻倍",
							sacrificed, workers*2),
					}, cost, true
				}

				// 普通技能：召唤小人，消耗 = 点数
				workers := esInt(es, "workers", 0)
				es["workers"] = workers + pts
				cost := pts
				return &SkillResult{
					Tier: TierNormal,
					Desc: fmt.Sprintf("召唤小人：召唤了%d名小人（现有%d名），开始建造", pts, workers+pts),
				}, cost, true
			},

			BuildExtraInfo: func(es map[string]any) map[string]any {
				workers := esInt(es, "workers", 0)
				houses := esInt(es, "houses", 0)
				progress := esInt(es, "build_progress", 0)
				effMult := esInt(es, "eff_mult", 1)
				if workers == 0 && houses == 0 {
					return nil
				}
				return map[string]any{
					"workers":        workers,
					"houses":         houses,
					"build_progress": progress,
					"eff_mult":       effMult,
				}
			},
		},
	}
}
