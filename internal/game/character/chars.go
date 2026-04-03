package character

// LiewenDefaultRiftBonus 是裂缝者每条裂缝每阶段的初始能量产出。
// 此值也在 data/characters/liewen.json 的 hooks_config.default_rift_bonus 中配置。
const LiewenDefaultRiftBonus = 3

// 角色数据已迁移至 data/characters/*.json，通过 LoadFromDir() 加载。
// 行为钩子在 hooks_liewen.go、hooks_wanneng.go、hooks_xuemo.go 中通过 init() 注册。
