package game

import (
	"testing"
	"time"

	"echo/internal/game/character"
	"echo/internal/player"
	"echo/internal/protocol"
	"echo/internal/room"
)

// ════════════════════════════════════════════════════════════════
//  Phase 4.7 — 蘇芳复活对话框流程测试
//
//  沿用 handler_surrender_test.go 的最小 Engine harness 模式：
//  直接构造 *Engine + 离线 AIPlayer（无 Session，Send 自动静默丢弃），
//  同步驱动 processAction，断言 state 变化和 ctx 是否被取消。
// ════════════════════════════════════════════════════════════════

// newSuouTestEngine 构造一个用于蘇芳测试的最小 Engine。
// seat0 选蘇芳；seat1 选 rokka（任意角色均可）。
// reviveTimeoutSec 可覆盖（测试需要快速到期时传更小值）。
func newSuouTestEngine(t *testing.T, reviveTimeoutSec int) *Engine {
	t.Helper()
	p0 := player.NewAIPlayer("test-p0")
	p1 := player.NewAIPlayer("test-p1")
	r := &room.Room{
		ID:      "test-room-suou",
		Players: [2]*player.Player{p0, p1},
		AISeat:  -1,
	}
	e := NewEngine(r)
	if reviveTimeoutSec > 0 {
		e.reviveTimeoutSec = reviveTimeoutSec
	}

	// seat 0：蘇芳
	suou, err := character.NewInstance("suou")
	if err != nil {
		t.Fatalf("create suou instance: %v", err)
	}
	e.state.Players[0].Char = suou
	e.state.Players[0].CharacterID = "suou"
	e.state.Players[0].MaxHP = 30
	e.state.Players[0].HP = 30
	e.state.Players[0].MaxEnergy = 30

	// seat 1：任意已注册角色作为对手
	opp, err := character.NewInstance("rokka")
	if err != nil {
		t.Fatalf("create opponent instance: %v", err)
	}
	e.state.Players[1].Char = opp
	e.state.Players[1].CharacterID = "rokka"

	e.state.Phase = PhaseAction
	e.state.Round = 1
	return e
}

// TestSuouEnterReviveOnLethal: 蘇芳 HP 归零 → 进入 AwaitingRevive，不调用 triggerDeath。
func TestSuouEnterReviveOnLethal(t *testing.T) {
	e := newSuouTestEngine(t, 60) // 长超时避免在断言期间触发 sentinel

	// 把 HP 砍到 0：直接调用 applyDamage（绕过攻击牌路径，无 PendingAttack 干扰）
	e.applyDamage(0, 100, "test lethal")

	if e.state.AwaitingRevive != 0 {
		t.Fatalf("AwaitingRevive = %d, want 0 (seat 0 should be awaiting revive)", e.state.AwaitingRevive)
	}
	if e.state.isOver() {
		t.Fatal("game should NOT be over yet: suou enters revive dialog instead of dying")
	}
	if e.state.Phase == PhaseGameOver {
		t.Fatalf("Phase = %q, must NOT be game_over while revive dialog is open", e.state.Phase)
	}
	if e.state.ReviveDeadline.IsZero() {
		t.Fatal("ReviveDeadline must be set")
	}
	if !e.state.ReviveDeadline.After(time.Now()) {
		t.Fatal("ReviveDeadline must be in the future")
	}
}

// TestSuouReviveTimeout: 进入复活对话框后 timeout 触发 → finalizeRealDeath，
// 对手获胜，Reason="suou_revive_timeout"，ctx 被取消。
func TestSuouReviveTimeout(t *testing.T) {
	// 用 1 秒超时让 sentinel 快速到达；测试主体内通过 processAction 同步驱动。
	e := newSuouTestEngine(t, 1)

	// 触发 enterAwaitingRevive：HP 归零
	e.applyDamage(0, 100, "test lethal")
	if e.state.AwaitingRevive != 0 {
		t.Fatalf("setup: AwaitingRevive = %d, want 0", e.state.AwaitingRevive)
	}

	// 等待 sentinel 到达 actionCh（goroutine 1s 后投递）。
	// 用 select 限定最长等待 3s 防止 hang。
	var act action
	select {
	case act = <-e.actionCh:
	case <-time.After(3 * time.Second):
		t.Fatal("revive timeout sentinel did not arrive within 3s")
	}
	if act.MsgID != msgReviveTimeoutSentinel {
		t.Fatalf("first action MsgID = %d, want %d (sentinel)", act.MsgID, msgReviveTimeoutSentinel)
	}

	// 同步驱动：将 sentinel 喂给 processAction
	e.processAction(act)

	if e.state.Winner != 1 {
		t.Fatalf("Winner = %d, want 1 (opponent wins after suou revive timeout)", e.state.Winner)
	}
	if e.state.Phase != PhaseGameOver {
		t.Fatalf("Phase = %q, want %q", e.state.Phase, PhaseGameOver)
	}
	select {
	case <-e.ctx.Done():
		// 期望：finalizeRealDeath 调用了 e.Stop()
	default:
		t.Fatal("engine ctx not cancelled after revive timeout")
	}
	_ = protocol.MsgDeathDialogEv // keep protocol import referenced
}
