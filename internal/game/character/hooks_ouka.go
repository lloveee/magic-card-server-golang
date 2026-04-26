package character

import (
	"fmt"
	"math"
)

func init() {
	// 建造者：召唤人偶盖房子的经营型角色。
	// 被动：技能牌点数 = 能量消耗。根据房子数量解锁并叠加被动。
	//   阶段一(>1房子)：每层房子每回合增加 +2 名工人数量
	//   阶段二(>2房子)：每层房子增加 +2 减伤
	//   阶段三(>=3房子)：每层房子每回合回血 +5
	//   受到单次 >20 伤害：房子数量减半（向上取整）
	// 普通技能：按技能牌点数召唤等量人偶，消耗等量能量。
	// 解放技（>=5房子，20-25点技能牌）：扣除所有房子，人偶翻倍+效率翻倍。
	//
	// ExtraState:
	//   "workers"        int - 人偶数量
	//   "build_progress" int - 当前房子进度(0-99)
	//   "houses"         int - 已完成房子数
	//   "eff_mult"       int - 效率倍率(默认1，解放后2)

	registry["ouka"] = &CharDef{
		ID: "ouka",
		Hooks: &CharHooks{
			OnPhaseStart: func(phase string, es map[string]any) (int, string) {
				if phase != "action" {
					return 0, ""
				}
				cfg := HooksConfig("ouka")
				workers := esInt(es, "workers", 0)
				if workers == 0 {
					return 0, ""
				}

				houses := esInt(es, "houses", 0)
				progress := esInt(es, "build_progress", 0)
				effMult := esInt(es, "eff_mult", 1)

				// >1房子：每回合增加工人数量
				if houses > 1 {
					workerBonus := hcInt(cfg, "house1_worker_bonus", 2)
					newWorkers := houses * workerBonus
					es["workers"] = workers + newWorkers
					workers = workers + newWorkers
				}
				// 基础效率计算
				baseEff := hcInt(cfg, "base_worker_eff", 1)
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

				// 阶段三被动：>=3 房子，每层房子回血
				healDelta := 0
				if houses >= 3 {
					healPerHouse := hcInt(cfg, "house3_heal", 5)
					healDelta = houses * healPerHouse
				}

				var msg string
				if newHouses > 0 {
					msg = fmt.Sprintf("人偶建造报告：%d工人产出%d进度，完成%d栋新房（共%d栋），剩余进度%d/100",
						workers, totalWork, newHouses, houses, progress)
				} else {
					msg = fmt.Sprintf("人偶建造报告：%d工人产出%d进度（%d/100），共%d栋房子",
						workers, totalWork, progress, houses)
				}

				if healDelta > 0 {
					es["pending_heal"] = healDelta
					msg += fmt.Sprintf("，每层房回血5（共%d栋→回复%d血）", houses, healDelta)
				}

				return 0, msg
			},

			ModifyIncomingDamage: func(damage int, damageType string, es map[string]any) (int, int) {
				cfg := HooksConfig("ouka")
				houses := esInt(es, "houses", 0)
				finalDamage := damage

				// 阶段二被动：>2 房子，每层房子减伤
				if houses > 2 {
					reductionPerHouse := hcInt(cfg, "house2_dmg_reduction", 2)
					reduction := houses * reductionPerHouse
					finalDamage -= reduction
					if finalDamage < 1 {
						finalDamage = 1
					}
				}

				return finalDamage, 0
			},

			OnDamageReceived: func(finalDamage int, es map[string]any) {
				cfg := HooksConfig("ouka")
				threshold := hcInt(cfg, "damage_halve_threshold", 20)
				houses := esInt(es, "houses", 0)

				// 受到单次 > threshold 伤害：房子减半（向上取整）
				if finalDamage > threshold && houses > 0 {
					es["houses"] = int(math.Ceil(float64(houses) / 2.0))
				}
			},

			UseSkillOverride: func(pts int, es map[string]any) (*SkillResult, int, bool) {
				cfg := HooksConfig("ouka")

				houses := esInt(es, "houses", 0)
				libHouseThreshold := hcInt(cfg, "lib_house_threshold", 5)
				libPtsMin := hcInt(cfg, "lib_pts_min", 20)
				libPtsMax := hcInt(cfg, "lib_pts_max", 25)

				// 解放判定：房子够 + 点数在范围内
				if houses >= libHouseThreshold && pts >= libPtsMin && pts <= libPtsMax {
					workers := esInt(es, "workers", 0)
					sacrificed := houses
					es["houses"] = 0
					es["build_progress"] = 0
					es["workers"] = workers * 2
					es["eff_mult"] = esInt(es, "eff_mult", 1) * 2
					cost := pts
					return &SkillResult{
						Tier: TierLiberation,
						Desc: fmt.Sprintf("创世解放：献祭%d栋房子，人偶翻倍至%d，工作效率翻倍",
							sacrificed, workers*2),
					}, cost, true
				}

				// 普通技能：召唤人偶，消耗 = 点数
				workers := esInt(es, "workers", 0)
				es["workers"] = workers + pts
				cost := pts
				return &SkillResult{
					Tier: TierNormal,
					Desc: fmt.Sprintf("人偶召唤：召唤了%d名人偶（现有%d名），开始建造", pts, workers+pts),
				}, cost, true
			},

			BuildPublicExtra: func(es map[string]any) map[string]any {
				houses := esInt(es, "houses", 0)
				workers := esInt(es, "workers", 0)
				if houses == 0 && workers == 0 {
					return nil
				}
				return map[string]any{
					"houses":  houses,
					"workers": workers,
				}
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
