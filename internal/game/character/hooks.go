package character

// CharHooks 包含角色的可选行为钩子，允许特殊角色覆盖或扩展引擎的标准逻辑。
// 所有函数字段均为可选（nil = 使用默认行为），引擎在调用前检查 nil。
type CharHooks struct {

	// ── 属性标志 ─────────────────────────────────────────────────

	// HPEnergyShared：HP 和能量共享同一数值。
	// 能量增加时 HP 同步增加，受到伤害时能量同步减少。
	HPEnergyShared bool

	// AllCardsAsAttack：所有手牌均视为攻击牌出牌，优先级高于场地效果。
	AllCardsAsAttack bool

	// LibRepeatable：解放技能可多次触发（不受 LibUsed 限制）。
	LibRepeatable bool

	// InitHP / InitEnergy：覆盖选角后的初始值（0 = 使用 MaxHP / MaxEnergy）。
	InitHP     int
	InitEnergy int

	// ── 阶段钩子 ─────────────────────────────────────────────────

	// OnPhaseStart 在每个阶段开始时对角色拥有者调用。
	// 返回能量变化量（正=获得，负=消耗）和可选日志文本。
	// 若 HPEnergyShared，引擎同步修改 HP。
	OnPhaseStart func(phase string, es map[string]any) (energyDelta int, msg string)

	// ── 伤害钩子 ─────────────────────────────────────────────────

	// OnDamageReceived 在此玩家受到任意来源的最终伤害后调用。
	OnDamageReceived func(finalDamage int, es map[string]any)

	// OnDamageDealt 在此玩家对对手造成伤害后调用（不含自伤）。
	OnDamageDealt func(finalDamage int, es map[string]any)

	// OnDamageLanded 在此玩家对对手造成伤害后调用，返回自身回复量（0=不回复）。
	// 用于吸血效果。
	OnDamageLanded func(finalDamage int, es map[string]any) int

	// ── 攻击修正钩子 ─────────────────────────────────────────────

	// ModifyCardPoints 在 BonusOutgoing 和场地加成之前修改牌面点数。
	ModifyCardPoints func(pts int, es map[string]any) int

	// ModifyOutgoingAttack 在 BonusOutgoing 和场地加成之后对最终攻击点数进行修正。
	// 应返回修正后的最终值（而非增量）。
	ModifyOutgoingAttack func(pts int, energy int, es map[string]any) int

	// OnAttackLaunched 在创建 PendingAttack 之前调用。
	// 返回 (额外攻击点数, 消耗能量)，引擎将额外点数加入攻击并扣除能量。
	OnAttackLaunched func(attackPoints int, energy int, es map[string]any) (extraPoints int, energySpent int)

	// ── 技能覆盖 ─────────────────────────────────────────────────

	// UseSkillOverride 替换默认的技能档位判定逻辑（若设置）。
	// 返回 (result, cost, handled)。handled=false 则回退到普通/强化默认逻辑。
	UseSkillOverride func(cardPoints int, es map[string]any) (result *SkillResult, cost int, handled bool)
}

// esInt 从 ExtraState 读取 int 值，键不存在或类型不符时返回 defVal。
func esInt(es map[string]any, key string, defVal int) int {
	if v, ok := es[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return defVal
}

// esBool 从 ExtraState 读取 bool 值，键不存在或类型不符时返回 defVal。
func esBool(es map[string]any, key string, defVal bool) bool {
	if v, ok := es[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defVal
}
