// cmd/exceltojson 是一个小工具，将策划填写的 Excel 配置表导出为服务端/客户端使用的 JSON 数据文件。
//
// 用法：
//
//	go run ./cmd/exceltojson -i data/游戏配置表.xlsx -o data/
//
// 会在 -o 目录下生成：
//   - characters.json （角色配置）
//   - fields.json     （场地效果配置）
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	input := flag.String("i", "data/游戏配置表.xlsx", "输入 Excel 文件路径")
	output := flag.String("o", "data/", "输出目录")
	flag.Parse()

	f, err := excelize.OpenFile(*input)
	if err != nil {
		log.Fatalf("打开 Excel 失败: %v", err)
	}
	defer f.Close()

	// 导出角色表
	chars, err := parseCharacters(f)
	if err != nil {
		log.Fatalf("解析[角色]表失败: %v", err)
	}
	if err := writeJSON(filepath.Join(*output, "characters.json"), chars); err != nil {
		log.Fatalf("写入 characters.json 失败: %v", err)
	}
	fmt.Printf("✓ 已导出 %d 个角色 → %s\n", len(chars), filepath.Join(*output, "characters.json"))

	// 导出场地效果表
	fields, err := parseFields(f)
	if err != nil {
		log.Fatalf("解析[场地效果]表失败: %v", err)
	}
	if err := writeJSON(filepath.Join(*output, "fields.json"), fields); err != nil {
		log.Fatalf("写入 fields.json 失败: %v", err)
	}
	fmt.Printf("✓ 已导出 %d 个场地效果 → %s\n", len(fields), filepath.Join(*output, "fields.json"))
}

// ════════════════════════════════════════════════════════════════
//  角色表解析
// ════════════════════════════════════════════════════════════════

// Excel "角色" 表的列顺序（A-Z）：
// A: id
// B: name
// C: max_hp
// D: max_energy
// E: lib_threshold
// F: manual_lib (是/否)
// G: passive_bonus_outgoing
// H: passive_incoming_reduction
// I: passive_intercept_near_death (是/否)
// J: normal_name
// K: normal_energy_cost
// L: normal_damage
// M: normal_heal
// N: normal_energy_gain
// O: normal_draw
// P: normal_self_damage
// Q: normal_desc
// R: enhanced_name
// S: enhanced_energy_cost
// T: enhanced_damage
// U: enhanced_heal
// V: enhanced_energy_gain
// W: enhanced_draw
// X: enhanced_self_damage
// Y: enhanced_desc
// Z: lib_name
// AA: lib_energy_cost
// AB: lib_damage
// AC: lib_heal
// AD: lib_energy_gain
// AE: lib_draw
// AF: lib_self_damage
// AG: lib_desc
// AH: hooks_config (JSON字符串，可选)

func parseCharacters(f *excelize.File) ([]map[string]any, error) {
	const sheet = "角色"
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表[%s]: %w", sheet, err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("工作表[%s]至少需要表头+1行数据", sheet)
	}

	var result []map[string]any
	for i, row := range rows[1:] { // 跳过表头
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue // 跳过空行
		}
		ch, err := rowToCharacter(row, i+2) // i+2 = Excel行号（1-based + 表头）
		if err != nil {
			return nil, fmt.Errorf("第%d行: %w", i+2, err)
		}
		result = append(result, ch)
	}
	return result, nil
}

func rowToCharacter(row []string, lineNum int) (map[string]any, error) {
	get := func(col int) string {
		if col < len(row) {
			return strings.TrimSpace(row[col])
		}
		return ""
	}
	getInt := func(col int) int {
		s := get(col)
		if s == "" {
			return 0
		}
		n, _ := strconv.Atoi(s)
		return n
	}
	getBool := func(col int) bool {
		s := get(col)
		return s == "是" || s == "true" || s == "TRUE" || s == "1"
	}

	id := get(0)
	if id == "" {
		return nil, fmt.Errorf("id 不能为空")
	}

	ch := map[string]any{
		"id":            id,
		"name":          get(1),
		"max_hp":        getInt(2),
		"max_energy":    getInt(3),
		"lib_threshold": getInt(4),
		"manual_lib":    getBool(5),
	}

	// passive
	passive := map[string]any{}
	if v := getInt(6); v != 0 {
		passive["bonus_outgoing"] = v
	}
	if v := getInt(7); v != 0 {
		passive["incoming_reduction"] = v
	}
	if getBool(8) {
		passive["intercept_near_death"] = true
	}
	ch["passive"] = passive

	// normal skill (J-Q, col 9-16)
	ch["normal"] = buildSkill(get(9), getInt(10), getInt(11), getInt(12), getInt(13), getInt(14), getInt(15), get(16))

	// enhanced skill (R-Y, col 17-24)
	ch["enhanced"] = buildSkill(get(17), getInt(18), getInt(19), getInt(20), getInt(21), getInt(22), getInt(23), get(24))

	// lib skill (Z-AG, col 25-32)
	ch["lib"] = buildSkill(get(25), getInt(26), getInt(27), getInt(28), getInt(29), getInt(30), getInt(31), get(32))

	// hooks_config (AH, col 33) — 可选JSON字符串
	if hc := get(33); hc != "" {
		var hooksConfig map[string]any
		if err := json.Unmarshal([]byte(hc), &hooksConfig); err != nil {
			return nil, fmt.Errorf("hooks_config JSON 解析失败 (行%d): %w", lineNum, err)
		}
		ch["hooks_config"] = hooksConfig
	}

	return ch, nil
}

func buildSkill(name string, cost, damage, heal, energy, draw, selfDmg int, desc string) map[string]any {
	skill := map[string]any{}
	if name != "" {
		skill["name"] = name
	}
	if cost != 0 {
		skill["energy_cost"] = cost
	}
	result := map[string]any{}
	if damage != 0 {
		result["deal_direct_damage"] = damage
	}
	if heal != 0 {
		result["heal_self"] = heal
	}
	if energy != 0 {
		result["gain_energy"] = energy
	}
	if draw != 0 {
		result["draw_cards"] = draw
	}
	if selfDmg != 0 {
		result["damage_self"] = selfDmg
	}
	if desc != "" {
		result["desc"] = desc
	}
	if len(result) > 0 {
		skill["result"] = result
	}
	return skill
}

// ════════════════════════════════════════════════════════════════
//  场地效果表解析
// ════════════════════════════════════════════════════════════════

// Excel "场地效果" 表的列顺序（A-G）：
// A: id
// B: name
// C: illusion_bonus (是/否)
// D: allow_same_type (是/否)
// E: reincarn_rule (0/1/2)
// F: hide_drawn_cards (是/否)
// G: bonus_attack
// H: near_death_drain

func parseFields(f *excelize.File) ([]map[string]any, error) {
	const sheet = "场地效果"
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表[%s]: %w", sheet, err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("工作表[%s]至少需要表头+1行数据", sheet)
	}

	var result []map[string]any
	for i, row := range rows[1:] {
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		get := func(col int) string {
			if col < len(row) {
				return strings.TrimSpace(row[col])
			}
			return ""
		}
		getInt := func(col int) int {
			s := get(col)
			if s == "" {
				return 0
			}
			n, _ := strconv.Atoi(s)
			return n
		}
		getBool := func(col int) bool {
			s := get(col)
			return s == "是" || s == "true" || s == "TRUE" || s == "1"
		}

		id := get(0)
		if id == "" {
			continue
		}

		fe := map[string]any{
			"id":   id,
			"name": get(1),
		}
		if getBool(2) {
			fe["illusion_bonus"] = true
		}
		if getBool(3) {
			fe["allow_same_type"] = true
		}
		if v := getInt(4); v != 0 {
			fe["reincarn_rule"] = v
		}
		if getBool(5) {
			fe["hide_drawn_cards"] = true
		}
		if v := getInt(6); v != 0 {
			fe["bonus_attack"] = v
		}
		if v := getInt(7); v != 0 {
			fe["near_death_drain"] = v
		}

		result = append(result, fe)
		_ = i // suppress unused
	}
	return result, nil
}

// ════════════════════════════════════════════════════════════════
//  工具函数
// ════════════════════════════════════════════════════════════════

func writeJSON(path string, data any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
