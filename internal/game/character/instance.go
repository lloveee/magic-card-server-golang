package character

import "errors"

// ════════════════════════════════════════════════════════════════
//  CharInstance — 角色的运行时状态
// ════════════════════════════════════════════════════════════════

// CharInstance 持有一名玩家已选角色在一局游戏中的可变状态。
//
// 不可变的定义（技能效果、初始属性）放在 CharDef，
// 只有"已用过"类的状态放在此结构。
type CharInstance struct {
	Def *CharDef

	// LibUsed：解放技能是否已触发过（LibRepeatable=true 时此字段不阻止再次触发）
	LibUsed bool

	// InterceptUsed：殉道者被动"二次死亡拦截"是否已用过
	InterceptUsed bool

	// ExtraState：角色特定的跨阶段持久状态（钩子函数用）
	ExtraState map[string]any
}

// NewInstance 根据角色 ID 创建运行时实例。
// 角色 ID 未注册时返回错误。
func NewInstance(charID string) (*CharInstance, error) {
	def, ok := Get(charID)
	if !ok {
		return nil, errors.New("未知角色 ID: " + charID)
	}
	return &CharInstance{
		Def:        def,
		ExtraState: make(map[string]any),
	}, nil
}

// ════════════════════════════════════════════════════════════════
//  技能激活
// ════════════════════════════════════════════════════════════════

// UseSkill 根据技能牌点数决定激活哪个档位的技能。
// 返回：(技能结果, 能量消耗, 错误)
//
// 优先检查 Hooks.UseSkillOverride；若未处理则回退默认逻辑：
//
//	cardPoints ≤ 2 → TierNormal
//	cardPoints ≥ 3 → TierEnhanced
func (ci *CharInstance) UseSkill(cardPoints int) (*SkillResult, int, error) {
	// 先尝试钩子覆盖
	if ci.Def.Hooks != nil && ci.Def.Hooks.UseSkillOverride != nil {
		if result, cost, handled := ci.Def.Hooks.UseSkillOverride(cardPoints, ci.ExtraState); handled {
			return result, cost, nil
		}
	}
	// 默认档位逻辑
	var skill SkillDef
	if cardPoints <= 2 {
		skill = ci.Def.Normal
	} else {
		skill = ci.Def.Enhanced
	}
	result := skill.Result // 值拷贝，防止外部意外修改定义
	return &result, skill.EnergyCost, nil
}

// TriggerLiberation 触发解放技能。
// LibRepeatable=true 时可重复触发；否则每局只能触发一次。
// 能量消耗由调用方（engine）在调用前检查并扣除。
func (ci *CharInstance) TriggerLiberation() (*SkillResult, error) {
	repeatable := ci.Def.Hooks != nil && ci.Def.Hooks.LibRepeatable
	if ci.LibUsed && !repeatable {
		return nil, errors.New("解放技能每局只能使用一次")
	}
	if !repeatable {
		ci.LibUsed = true
	}
	result := ci.Def.Lib.Result // 值拷贝
	return &result, nil
}

// CanLiberate 检查是否可以手动触发解放（能量足够，且若不可重复则未用过）。
func (ci *CharInstance) CanLiberate(energy int) bool {
	if ci.Def.LibThreshold <= 0 {
		return false // 阈值为 0 表示该角色不使用传统解放按钮
	}
	repeatable := ci.Def.Hooks != nil && ci.Def.Hooks.LibRepeatable
	if !repeatable && ci.LibUsed {
		return false
	}
	return energy >= ci.Def.LibThreshold
}

// ════════════════════════════════════════════════════════════════
//  被动钩子
// ════════════════════════════════════════════════════════════════

// ModifyOutgoing 将被动"攻击加成"应用到攻击伤害上。
// engine 在每次结算攻击牌伤害时调用此方法。
func (ci *CharInstance) ModifyOutgoing(damage int) int {
	return damage + ci.Def.Passive.BonusOutgoing
}

// ModifyIncoming 将被动"伤害减免"应用到受到的伤害上。
// 最终伤害最低为 1（不能被减免到 0）。
func (ci *CharInstance) ModifyIncoming(damage int) int {
	d := damage - ci.Def.Passive.IncomingReduction
	if d < 1 {
		d = 1
	}
	return d
}

// InterceptSecondDeath 尝试用被动拦截二次死亡（殉道者）。
// 返回 true 表示成功拦截，此后 LibUsed 和 InterceptUsed 均设为 true。
// 每局只能拦截一次。
func (ci *CharInstance) InterceptSecondDeath() bool {
	if !ci.Def.Passive.InterceptNearDeath {
		return false
	}
	if ci.InterceptUsed {
		return false
	}
	ci.InterceptUsed = true
	ci.LibUsed = true // 自动触发解放，标记解放已用
	return true
}
