package card_test

// synthesis_test.go — 合成系统的表驱动单元测试
//
// 学习要点：
//   1. Go 表驱动测试（table-driven tests）的标准写法
//   2. 测试纯函数：相同输入一定给出相同输出，最易验证
//   3. 测试错误路径（sentinel error 比较）
//   4. 测试边界条件（点数上限截断）

import (
	"errors"
	"testing"

	"echo/internal/game/card"
)

// ════════════════════════════════════════════════════════════════
//  辅助工厂
// ════════════════════════════════════════════════════════════════

// atk / skl / eng 快速创建指定花色、指定牌型、指定点数的牌。
func atk(s card.Suit, pts int) *card.Card { return card.New(s, card.TypeAttack, pts) }
func skl(s card.Suit, pts int) *card.Card { return card.New(s, card.TypeSkill, pts) }
func eng(s card.Suit, pts int) *card.Card { return card.New(s, card.TypeEnergy, pts) }

// ════════════════════════════════════════════════════════════════
//  Combine 主规则测试
// ════════════════════════════════════════════════════════════════

func TestCombine_SameMajor_Multiplies(t *testing.T) {
	// 同大系（梦幻+梦幻）→ 点数相乘
	cases := []struct {
		name string
		base *card.Card
		ingr *card.Card
		want int
	}{
		{"梦境攻击×梦境技能 2×3=6", atk(card.SuitHeart, 2), skl(card.SuitHeart, 3), 6},
		{"梦境攻击×梦境技能 2×2=4", atk(card.SuitHeart, 2), skl(card.SuitHeart, 2), 4},
		{"虚幻攻击×虚幻能耗 1×5=5", atk(card.SuitDiamond, 1), eng(card.SuitDiamond, 5), 5},
		{"重组攻击×重组技能 3×2=6", atk(card.SuitClub, 3), skl(card.SuitClub, 2), 6},
	}

	for _, tc := range cases {
		tc := tc // 防止闭包捕获问题（Go 1.22 前需要）
		t.Run(tc.name, func(t *testing.T) {
			result, err := card.Combine(tc.base, tc.ingr, card.DefaultOpts())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Points != tc.want {
				t.Errorf("points = %d, want %d", result.Points, tc.want)
			}
			// 结果继承 base 的属性
			if result.CardType != tc.base.CardType {
				t.Errorf("CardType = %v, want %v", result.CardType, tc.base.CardType)
			}
			if result.Suit != tc.base.Suit {
				t.Errorf("Suit = %v, want %v", result.Suit, tc.base.Suit)
			}
		})
	}
}

func TestCombine_DifferentMajor_Adds(t *testing.T) {
	// 不同大系（梦幻+重回）→ 点数相加
	cases := []struct {
		name string
		base *card.Card
		ingr *card.Card
		want int
	}{
		{"梦境攻击+轮回技能 2+3=5", atk(card.SuitHeart, 2), skl(card.SuitSpade, 3), 5},
		{"虚幻攻击+重组技能 3+4=7", atk(card.SuitDiamond, 3), skl(card.SuitClub, 4), 7},
		{"梦境技能+重组攻击 1+1=2", skl(card.SuitHeart, 1), atk(card.SuitClub, 1), 2},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := card.Combine(tc.base, tc.ingr, card.DefaultOpts())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Points != tc.want {
				t.Errorf("points = %d, want %d", result.Points, tc.want)
			}
		})
	}
}

// ════════════════════════════════════════════════════════════════
//  错误路径：同类型禁止
// ════════════════════════════════════════════════════════════════

func TestCombine_SameType_ReturnsErrSameCardType(t *testing.T) {
	pairs := [][2]*card.Card{
		{atk(card.SuitHeart, 1), atk(card.SuitClub, 2)},           // 攻击+攻击
		{skl(card.SuitHeart, 1), skl(card.SuitHeart, 2)},            // 技能+技能
		{eng(card.SuitDiamond, 3), eng(card.SuitSpade, 1)}, // 能耗+能耗
	}

	for _, pair := range pairs {
		_, err := card.Combine(pair[0], pair[1], card.DefaultOpts())
		if !errors.Is(err, card.ErrSameCardType) {
			t.Errorf("Combine(%v, %v) = %v, want ErrSameCardType", pair[0], pair[1], err)
		}
	}
}

// ════════════════════════════════════════════════════════════════
//  场地效果：混沌之域 AllowSameType
// ════════════════════════════════════════════════════════════════

func TestCombine_AllowSameType_NoError(t *testing.T) {
	opts := card.DefaultOpts()
	opts.AllowSameType = true

	base := atk(card.SuitHeart, 2)
	ingr := atk(card.SuitClub, 3) // 不同大系，相加 = 5

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error with AllowSameType: %v", err)
	}
	// 相加：梦幻+重回 → 2+3=5
	if result.Points != 5 {
		t.Errorf("points = %d, want 5", result.Points)
	}
}

func TestCombine_AllowSameType_SameMajorMultiplies(t *testing.T) {
	opts := card.DefaultOpts()
	opts.AllowSameType = true

	// 同大系同类型：2×3=6（无上限）
	base := atk(card.SuitHeart, 2)
	ingr := atk(card.SuitDiamond, 3)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 6 {
		t.Errorf("points = %d, want 6", result.Points)
	}
}

// ════════════════════════════════════════════════════════════════
//  场地效果：方片之境·实 IllusionBonus
// ════════════════════════════════════════════════════════════════

func TestCombine_IllusionBonus_CapAt7ForIllusion(t *testing.T) {
	opts := card.DefaultOpts()
	opts.IllusionBonus = true

	// base 是方片牌，同色乘法：3×3=9 → 上限提升至7
	base := atk(card.SuitDiamond, 3)
	ingr := skl(card.SuitDiamond, 3)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 7 {
		t.Errorf("IllusionBonus: points = %d, want 7", result.Points)
	}
}

func TestCombine_IllusionBonus_NoCapForNonIllusion(t *testing.T) {
	opts := card.DefaultOpts()
	opts.IllusionBonus = true

	// base 是红桃牌（非方片），IllusionBonus 不生效，无上限：3×3=9
	base := atk(card.SuitHeart, 3)
	ingr := skl(card.SuitHeart, 3)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 9 {
		t.Errorf("non-illusion base: points = %d, want 9", result.Points)
	}
}

// ════════════════════════════════════════════════════════════════
//  场地效果：黑桃之境·实 ReincarnationAsBase
// ════════════════════════════════════════════════════════════════

func TestCombine_ReincarnAsBase_UsesReincarnPoints(t *testing.T) {
	opts := card.DefaultOpts()
	opts.ReincarnationRule = card.ReincarnationAsBase

	// 黑桃牌点数4，另一张点数2 → 结果 = 黑桃牌自身 = 4
	base := atk(card.SuitSpade, 4)
	ingr := skl(card.SuitHeart, 2)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 4 {
		t.Errorf("ReincarnAsBase: points = %d, want 4", result.Points)
	}
}

func TestCombine_ReincarnAsBase_IngredientIsReinc(t *testing.T) {
	opts := card.DefaultOpts()
	opts.ReincarnationRule = card.ReincarnationAsBase

	// ingredient 是黑桃牌
	base := atk(card.SuitHeart, 3)
	ingr := skl(card.SuitSpade, 2)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 2 { // 黑桃牌自身点数2
		t.Errorf("ReincarnAsBase (ingr): points = %d, want 2", result.Points)
	}
}

// ════════════════════════════════════════════════════════════════
//  场地效果：黑桃之境·虚 ReincarnationAsOther
// ════════════════════════════════════════════════════════════════

func TestCombine_ReincarnAsOther_UsesOtherPoints(t *testing.T) {
	opts := card.DefaultOpts()
	opts.ReincarnationRule = card.ReincarnationAsOther

	// 黑桃牌4 + 红桃牌3 → 结果 = 红桃牌 = 3
	base := atk(card.SuitSpade, 4)
	ingr := skl(card.SuitHeart, 3)

	result, err := card.Combine(base, ingr, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Points != 3 {
		t.Errorf("ReincarnAsOther: points = %d, want 3", result.Points)
	}
}

// ════════════════════════════════════════════════════════════════
//  Validate 单独测试
// ════════════════════════════════════════════════════════════════

func TestValidate_NilCard(t *testing.T) {
	if err := card.Validate(nil, atk(card.SuitHeart, 1)); err == nil {
		t.Error("expected error for nil base")
	}
	if err := card.Validate(atk(card.SuitHeart, 1), nil); err == nil {
		t.Error("expected error for nil ingredient")
	}
}

func TestValidate_DifferentTypes_NoError(t *testing.T) {
	pairs := [][2]*card.Card{
		{atk(card.SuitHeart, 1), skl(card.SuitHeart, 1)},
		{atk(card.SuitHeart, 1), eng(card.SuitClub, 1)},
		{skl(card.SuitDiamond, 2), eng(card.SuitSpade, 3)},
	}
	for _, pair := range pairs {
		if err := card.Validate(pair[0], pair[1]); err != nil {
			t.Errorf("Validate(%v, %v) unexpected error: %v", pair[0], pair[1], err)
		}
	}
}
