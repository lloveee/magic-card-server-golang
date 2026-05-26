package game

import (
	"testing"

	"echo/internal/player"
	"echo/internal/protocol"
	"echo/internal/room"
)

// ════════════════════════════════════════════════════════════════
//  Phase 2.2 — MsgSurrenderReq 处理器
//
//  这些是焦点单元测试，直接构造 Engine + 离线 Player（无 Session，
//  Player.Send 自动静默丢弃），观察 handleSurrender 对 GameState 的
//  影响以及 engine.ctx 是否被取消。
//
//  我们不启动 engine.run() goroutine，而是同步调用 handleSurrender，
//  这样无需任何 PvP 集成测试框架（项目当前没有这类框架）。
// ════════════════════════════════════════════════════════════════

// newSurrenderTestEngine 构造一个用于投降处理器测试的最小 Engine。
// 两名 Player 都没有 Session，因此 room.Broadcast 是 no-op，
// 但 GameState 与 ctx 的变化仍然可观察。
func newSurrenderTestEngine(t *testing.T) *Engine {
	t.Helper()
	p0 := player.NewAIPlayer("test-p0")
	p1 := player.NewAIPlayer("test-p1")
	r := &room.Room{
		ID:      "test-room-surrender",
		Players: [2]*player.Player{p0, p1},
		AISeat:  -1, // PvP，避免任何 AI 自动逻辑
	}
	e := NewEngine(r)
	// 模拟双方已选角并进入行动阶段（投降只关心 game 是否已结束，
	// 阶段名只用于断言转换）。
	e.state.Phase = PhaseAction
	e.state.Round = 1
	return e
}

// TestSurrenderEndsGame：seat 0 投降 → 对手（seat 1）获胜，
// 阶段进入 game_over，引擎被停止。
func TestSurrenderEndsGame(t *testing.T) {
	e := newSurrenderTestEngine(t)

	e.handleSurrender(0)

	if e.state.Winner != 1 {
		t.Fatalf("Winner = %d, want 1 (seat 1 should win when seat 0 surrenders)", e.state.Winner)
	}
	if e.state.Phase != PhaseGameOver {
		t.Fatalf("Phase = %q, want %q", e.state.Phase, PhaseGameOver)
	}
	if !e.state.isOver() {
		t.Fatal("isOver() = false, want true after surrender")
	}
	select {
	case <-e.ctx.Done():
		// 期望路径：投降后引擎应取消 ctx，停止主循环
	default:
		t.Fatal("engine ctx not cancelled after surrender")
	}
}

// TestSurrenderFromSeat1：对称用例，seat 1 投降 → seat 0 获胜。
func TestSurrenderFromSeat1(t *testing.T) {
	e := newSurrenderTestEngine(t)

	e.handleSurrender(1)

	if e.state.Winner != 0 {
		t.Fatalf("Winner = %d, want 0 (seat 0 should win when seat 1 surrenders)", e.state.Winner)
	}
	if e.state.Phase != PhaseGameOver {
		t.Fatalf("Phase = %q, want %q", e.state.Phase, PhaseGameOver)
	}
}

// TestSurrenderAfterGameOver_NoOp：游戏已经结束后再次收到投降请求，
// 应为 no-op（不覆盖既有胜者）。匹配 triggerDeath 的常规模式：
// 只要 state.isOver()，所有结束游戏的入口都是幂等的。
func TestSurrenderAfterGameOver_NoOp(t *testing.T) {
	e := newSurrenderTestEngine(t)
	// 先让 seat 1 死亡（seat 0 获胜）
	e.triggerDeath(1)
	if e.state.Winner != 0 {
		t.Fatalf("setup failed: Winner = %d, want 0", e.state.Winner)
	}

	// seat 0 再发投降请求 — 不应反转胜负
	e.handleSurrender(0)

	if e.state.Winner != 0 {
		t.Fatalf("Winner reverted to %d after redundant surrender; want 0", e.state.Winner)
	}
}

// TestMsgSurrenderReqEncodes：编码 / 解码 SurrenderReq（空结构）正常工作，
// 这是引擎层调用 protocol.Decode[SurrenderReq] 的最小先决条件。
func TestMsgSurrenderReqEncodes(t *testing.T) {
	data := protocol.MustEncode(protocol.SurrenderReq{})
	if _, err := protocol.Decode[protocol.SurrenderReq](data); err != nil {
		t.Fatalf("decode SurrenderReq: %v", err)
	}
}
