package field

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// fieldJSON 是场地效果 JSON 的反序列化目标。
type fieldJSON struct {
	ID             EffectID `json:"id"`
	Name           string   `json:"name"`
	IllusionBonus  bool     `json:"illusion_bonus,omitempty"`
	AllowSameType  bool     `json:"allow_same_type,omitempty"`
	ReincarnRule   int8     `json:"reincarn_rule,omitempty"`
	HideDrawnCards bool     `json:"hide_drawn_cards,omitempty"`
	BonusAttack    int      `json:"bonus_attack,omitempty"`
	NearDeathDrain int      `json:"near_death_drain,omitempty"`
}

// LoadFromFile 从 JSON 文件加载场地效果，替换内置的 Pool。
func LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("field: read %s: %w", path, err)
	}
	var raws []fieldJSON
	if err := json.Unmarshal(data, &raws); err != nil {
		return fmt.Errorf("field: parse %s: %w", path, err)
	}
	if len(raws) == 0 {
		return fmt.Errorf("field: empty pool in %s", path)
	}

	newPool := make([]*FieldEffect, 0, len(raws))
	for _, r := range raws {
		newPool = append(newPool, &FieldEffect{
			ID:             r.ID,
			Name:           r.Name,
			IllusionBonus:  r.IllusionBonus,
			AllowSameType:  r.AllowSameType,
			ReincarnRule:   ReincarnHint(r.ReincarnRule),
			HideDrawnCards: r.HideDrawnCards,
			BonusAttack:    r.BonusAttack,
			NearDeathDrain: r.NearDeathDrain,
		})
		slog.Debug("field effect loaded", "id", r.ID, "name", r.Name)
	}
	Pool = newPool
	return nil
}

// AllJSON 返回所有场地效果的 JSON 可序列化数据（供 GameConfigEv 下发给客户端）。
func AllJSON() []map[string]any {
	result := make([]map[string]any, 0, len(Pool))
	for _, e := range Pool {
		m := map[string]any{
			"id":   string(e.ID),
			"name": e.Name,
		}
		if e.IllusionBonus {
			m["illusion_bonus"] = true
		}
		if e.AllowSameType {
			m["allow_same_type"] = true
		}
		if e.ReincarnRule != 0 {
			m["reincarn_rule"] = int(e.ReincarnRule)
		}
		if e.HideDrawnCards {
			m["hide_drawn_cards"] = true
		}
		if e.BonusAttack != 0 {
			m["bonus_attack"] = e.BonusAttack
		}
		if e.NearDeathDrain != 0 {
			m["near_death_drain"] = e.NearDeathDrain
		}
		result = append(result, m)
	}
	return result
}
