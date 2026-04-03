//go:build ignore

// gen_template.go 生成预填数据的 Excel 模板文件。
//
// 用法：go run gen_template.go
package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	defer f.Close()

	// ── 角色表 ─────────────────────────────────────────────────
	charSheet := "角色"
	f.SetSheetName("Sheet1", charSheet)

	charHeaders := []string{
		"id", "name", "max_hp", "max_energy", "lib_threshold", "manual_lib",
		"passive_bonus_outgoing", "passive_incoming_reduction", "passive_intercept_near_death",
		"normal_name", "normal_energy_cost", "normal_damage", "normal_heal", "normal_energy_gain", "normal_draw", "normal_self_damage", "normal_desc",
		"enhanced_name", "enhanced_energy_cost", "enhanced_damage", "enhanced_heal", "enhanced_energy_gain", "enhanced_draw", "enhanced_self_damage", "enhanced_desc",
		"lib_name", "lib_energy_cost", "lib_damage", "lib_heal", "lib_energy_gain", "lib_draw", "lib_self_damage", "lib_desc",
		"hooks_config",
	}
	for i, h := range charHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(charSheet, cell, h)
	}

	// 角色数据行
	charData := [][]any{
		{"licai", "力裁者", 100, 100, 80, "是", 1, "", "",
			"力裁斩击", 10, 8, "", "", "", "", "力裁斩击：对对手造成8点直接伤害",
			"强化裁决", 20, 16, "", "", "", "", "强化裁决：对对手造成16点直接伤害",
			"绝对裁决", 80, 30, 20, "", "", "", "绝对裁决：造成30点直接伤害并回复20点生命",
			""},
		{"jinghuan", "镜换者", 90, 100, 80, "是", "", 1, "",
			"镜像映射", 8, "", 5, "", 2, "", "镜像映射：摸2张牌并回复5点生命",
			"镜像反击", 16, 10, "", "", 3, "", "镜像反击：摸3张牌并造成10点直接伤害",
			"镜换轮转", 80, 20, 20, "", 2, "", "镜换轮转：造成20点直接伤害，回复20点生命，摸2张牌",
			""},
		{"kongshou", "空手者", 95, 100, 60, "是", "", "", "",
			"虚拳引气", 5, "", "", 20, "", "", "虚拳引气：获得20点能量",
			"引气冲拳", 10, 8, "", 30, "", "", "引气冲拳：获得30点能量并造成8点直接伤害",
			"空手相搏", 60, 20, 20, 20, "", "", "空手相搏：造成20点直接伤害，回复20点生命，获得20点能量",
			""},
		{"shiyuan", "噬渊者", 95, 100, 80, "是", "", "", "",
			"噬渊之触", 10, 6, 6, "", "", "", "噬渊之触：造成6点直接伤害并汲取6点生命",
			"深渊噬魂", 20, 14, 10, "", "", "", "深渊噬魂：造成14点直接伤害并汲取10点生命",
			"渊噬万物", 80, 28, 20, "", "", "", "渊噬万物：造成28点直接伤害并汲取20点生命",
			""},
		{"zhuoxue", "灼血者", 85, 100, 80, "是", 2, "", "",
			"灼血冲击", 10, 10, "", "", "", "", "灼血冲击：造成10点直接伤害",
			"烈焰灼血", 20, 20, "", "", "", "", "烈焰灼血：造成20点直接伤害",
			"血焰爆发", 80, 40, "", "", "", "", "血焰爆发：造成40点直接伤害",
			""},
		{"xundao", "殉道者", 110, 100, 80, "否", "", "", "是",
			"殉道之愿", 8, "", 10, "", "", "", "殉道之愿：回复10点生命",
			"殉道之力", 16, "", 20, 10, "", "", "殉道之力：回复20点生命并获得10点能量",
			"殉道解放", 0, "", 30, 10, "", "", "殉道解放：自动触发，濒死时回复30点生命并获得10点能量",
			""},
		{"liewen", "时空裂缝者", 150, 150, 0, "否", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			`{"hp_energy_shared":true,"init_hp":60,"init_energy":60,"default_rift_bonus":3,"normal_skill_cost":15,"enhanced_skill_cost":30,"enhanced_skill_pts_threshold":3,"rift_bonus_increment":2,"liberation_energy_threshold":100}`},
		{"wanneng", "万能者", 80, 100, 0, "否", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			`{"all_cards_as_attack":true,"phase_thresholds":[10,50,100],"phase2_card_bonus":2,"phase1_attack_bonus":2,"phase3_attack_multiplier":2}`},
		{"xuemo", "血魔", 90, 100, 0, "否", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			"", "", "", "", "", "", "", "",
			`{"dmg_received_threshold":50,"dmg_received_card_bonus":3,"lifesteal_damage_threshold":30,"lifesteal_activate_pts":25,"enhanced_pts_threshold":3,"enhanced_self_damage":20,"enhanced_atk_bonus":10,"normal_self_damage":10,"normal_draw_cards":2}`},
	}

	for r, data := range charData {
		for c, val := range data {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(charSheet, cell, val)
		}
	}

	// 设置列宽
	f.SetColWidth(charSheet, "A", "A", 12)
	f.SetColWidth(charSheet, "B", "B", 14)
	f.SetColWidth(charSheet, "Q", "Q", 40)
	f.SetColWidth(charSheet, "Y", "Y", 40)
	f.SetColWidth(charSheet, "AG", "AG", 50)
	f.SetColWidth(charSheet, "AH", "AH", 80)

	// ── 场地效果表 ─────────────────────────────────────────────
	fieldSheet := "场地效果"
	f.NewSheet(fieldSheet)

	fieldHeaders := []string{
		"id", "name", "illusion_bonus", "allow_same_type", "reincarn_rule",
		"hide_drawn_cards", "bonus_attack", "near_death_drain",
	}
	for i, h := range fieldHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(fieldSheet, cell, h)
	}

	fieldData := [][]any{
		{"clear", "空旷之地", "", "", "", "", "", ""},
		{"illusion_real", "虚幻之境·实", "是", "", "", "", "", ""},
		{"illusion_void", "虚幻之境·虚", "", "", "", "是", "", ""},
		{"reinc_base", "轮回之境·实", "", "", 1, "", "", ""},
		{"reinc_other", "轮回之境·虚", "", "", 2, "", "", ""},
		{"chaos", "混沌之域", "", "是", "", "", "", ""},
		{"echo", "回响之地", "", "", "", "", 1, ""},
		{"protect", "守护之光", "", "", "", "", "", 15},
	}

	for r, data := range fieldData {
		for c, val := range data {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(fieldSheet, cell, val)
		}
	}

	f.SetColWidth(fieldSheet, "A", "A", 16)
	f.SetColWidth(fieldSheet, "B", "B", 16)

	// 保存
	outPath := "data/游戏配置表.xlsx"
	if err := f.SaveAs(outPath); err != nil {
		log.Fatalf("保存失败: %v", err)
	}
	fmt.Printf("✓ 模板已生成 → %s\n", outPath)
}
