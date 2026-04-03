package character

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// ════════════════════════════════════════════════════════════════
//  JSON 数据加载器
// ════════════════════════════════════════════════════════════════

// charJSON 是角色 JSON 文件的反序列化目标。
type charJSON struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	MaxHP        int            `json:"max_hp"`
	MaxEnergy    int            `json:"max_energy"`
	LibThreshold int            `json:"lib_threshold"`
	ManualLib    bool           `json:"manual_lib"`
	Passive      passiveJSON    `json:"passive"`
	Normal       skillJSON      `json:"normal"`
	Enhanced     skillJSON      `json:"enhanced"`
	Lib          skillJSON      `json:"lib"`
	HooksConfig  map[string]any `json:"hooks_config,omitempty"`
}

type passiveJSON struct {
	BonusOutgoing      int  `json:"bonus_outgoing,omitempty"`
	IncomingReduction  int  `json:"incoming_reduction,omitempty"`
	InterceptNearDeath bool `json:"intercept_near_death,omitempty"`
}

type skillJSON struct {
	Name       string     `json:"name,omitempty"`
	EnergyCost int        `json:"energy_cost,omitempty"`
	Result     resultJSON `json:"result,omitempty"`
}

type resultJSON struct {
	DealDirectDamage int    `json:"deal_direct_damage,omitempty"`
	HealSelf         int    `json:"heal_self,omitempty"`
	GainEnergy       int    `json:"gain_energy,omitempty"`
	DrawCards        int    `json:"draw_cards,omitempty"`
	DamageSelf       int    `json:"damage_self,omitempty"`
	Desc             string `json:"desc,omitempty"`
}

// LoadFromFile 从单个 JSON 文件加载所有角色定义并注册到 registry。
// 已有 hooks 注册的角色（来自 hooks_*.go init()）会自动合并。
func LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("character: read %s: %w", path, err)
	}
	var raws []charJSON
	if err := json.Unmarshal(data, &raws); err != nil {
		return fmt.Errorf("character: parse %s: %w", path, err)
	}
	if len(raws) == 0 {
		return fmt.Errorf("character: empty character list in %s", path)
	}

	for _, raw := range raws {
		def := raw.toCharDef()

		// 如果该角色已有 hooks 注册（来自 hooks_*.go init()），合并进来
		if existing, ok := registry[def.ID]; ok && existing.Hooks != nil {
			def.Hooks = existing.Hooks
		}
		// 保存 hooks_config 到包级变量，供 hooks 读取数值参数
		if raw.HooksConfig != nil {
			hooksConfigs[def.ID] = raw.HooksConfig
		}

		registry[def.ID] = def
		slog.Debug("character loaded", "id", def.ID, "name", def.Name)
	}
	return nil
}

// AllJSON 返回所有角色的 JSON 可序列化数据（供 GameConfigEv 下发给客户端）。
func AllJSON() []map[string]any {
	result := make([]map[string]any, 0, len(registry))
	for _, def := range registry {
		ch := map[string]any{
			"id":            def.ID,
			"name":          def.Name,
			"max_hp":        def.MaxHP,
			"max_energy":    def.MaxEnergy,
			"lib_threshold": def.LibThreshold,
			"manual_lib":    def.ManualLib,
		}
		// passive
		p := map[string]any{}
		if def.Passive.BonusOutgoing != 0 {
			p["bonus_outgoing"] = def.Passive.BonusOutgoing
		}
		if def.Passive.IncomingReduction != 0 {
			p["incoming_reduction"] = def.Passive.IncomingReduction
		}
		if def.Passive.InterceptNearDeath {
			p["intercept_near_death"] = true
		}
		ch["passive"] = p

		// skills
		ch["normal"] = skillToMap(def.Normal)
		ch["enhanced"] = skillToMap(def.Enhanced)
		ch["lib"] = skillToMap(def.Lib)

		// hooks flags (客户端需要知道的特殊标志)
		if def.Hooks != nil {
			flags := map[string]any{}
			if def.Hooks.HPEnergyShared {
				flags["hp_energy_shared"] = true
			}
			if def.Hooks.AllCardsAsAttack {
				flags["all_cards_as_attack"] = true
			}
			if def.Hooks.InitHP != 0 {
				flags["init_hp"] = def.Hooks.InitHP
			}
			if def.Hooks.InitEnergy != 0 {
				flags["init_energy"] = def.Hooks.InitEnergy
			}
			if len(flags) > 0 {
				ch["hooks_flags"] = flags
			}
		}
		// hooks_config（数值参数，客户端可选用于展示）
		if cfg, ok := hooksConfigs[def.ID]; ok {
			ch["hooks_config"] = cfg
		}

		result = append(result, ch)
	}
	return result
}

func skillToMap(s SkillDef) map[string]any {
	m := map[string]any{}
	if s.Name != "" {
		m["name"] = s.Name
	}
	if s.EnergyCost != 0 {
		m["energy_cost"] = s.EnergyCost
	}
	r := map[string]any{}
	if s.Result.DealDirectDamage != 0 {
		r["deal_direct_damage"] = s.Result.DealDirectDamage
	}
	if s.Result.HealSelf != 0 {
		r["heal_self"] = s.Result.HealSelf
	}
	if s.Result.GainEnergy != 0 {
		r["gain_energy"] = s.Result.GainEnergy
	}
	if s.Result.DrawCards != 0 {
		r["draw_cards"] = s.Result.DrawCards
	}
	if s.Result.DamageSelf != 0 {
		r["damage_self"] = s.Result.DamageSelf
	}
	if s.Result.Desc != "" {
		r["desc"] = s.Result.Desc
	}
	if len(r) > 0 {
		m["result"] = r
	}
	return m
}

// hooksConfigs 存储各角色的 hooks_config，供 hooks 代码读取数值参数。
var hooksConfigs = map[string]map[string]any{}

// HooksConfig 返回指定角色的 hooks_config（只读）。
func HooksConfig(id string) map[string]any {
	return hooksConfigs[id]
}

// hcInt 从 hooks_config 读取 int 值（支持 float64 → int 转换，JSON 默认数字类型）。
func hcInt(cfg map[string]any, key string, defVal int) int {
	if cfg == nil {
		return defVal
	}
	v, ok := cfg[key]
	if !ok {
		return defVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return defVal
}

// hcBool 从 hooks_config 读取 bool 值。
func hcBool(cfg map[string]any, key string, defVal bool) bool {
	if cfg == nil {
		return defVal
	}
	v, ok := cfg[key]
	if !ok {
		return defVal
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return defVal
}

// hcIntSlice 从 hooks_config 读取 []int（JSON 数组 → []float64 → []int）。
func hcIntSlice(cfg map[string]any, key string, defVal []int) []int {
	if cfg == nil {
		return defVal
	}
	v, ok := cfg[key]
	if !ok {
		return defVal
	}
	arr, ok := v.([]any)
	if !ok {
		return defVal
	}
	result := make([]int, 0, len(arr))
	for _, item := range arr {
		if f, ok := item.(float64); ok {
			result = append(result, int(f))
		}
	}
	if len(result) == 0 {
		return defVal
	}
	return result
}

// toCharDef 将 JSON 结构转换为 CharDef（不含 Hooks，Hooks 由单独的 RegisterHooks 设置）。
func (c *charJSON) toCharDef() *CharDef {
	tier := func(t SkillTier, s skillJSON) SkillDef {
		return SkillDef{
			Name:       s.Name,
			EnergyCost: s.EnergyCost,
			Result: SkillResult{
				Tier:             t,
				DealDirectDamage: s.Result.DealDirectDamage,
				HealSelf:         s.Result.HealSelf,
				GainEnergy:       s.Result.GainEnergy,
				DrawCards:        s.Result.DrawCards,
				DamageSelf:       s.Result.DamageSelf,
				Desc:             s.Result.Desc,
			},
		}
	}
	return &CharDef{
		ID:           c.ID,
		Name:         c.Name,
		MaxHP:        c.MaxHP,
		MaxEnergy:    c.MaxEnergy,
		LibThreshold: c.LibThreshold,
		ManualLib:    c.ManualLib,
		Passive: PassiveTraits{
			BonusOutgoing:      c.Passive.BonusOutgoing,
			IncomingReduction:  c.Passive.IncomingReduction,
			InterceptNearDeath: c.Passive.InterceptNearDeath,
		},
		Normal:   tier(TierNormal, c.Normal),
		Enhanced: tier(TierEnhanced, c.Enhanced),
		Lib:      tier(TierLiberation, c.Lib),
	}
}
