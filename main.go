package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"echo/internal/config"
	"echo/internal/game"
	"echo/internal/game/character"
	"echo/internal/game/field"
	"echo/internal/matchmaking"
	"echo/internal/network"
	"echo/internal/player"
	"echo/internal/protocol"
	"echo/internal/room"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
	slog.Info("config loaded", "addr", cfg.ListenAddr, "rateLimit", cfg.RateLimit, "turnDuration", cfg.TurnDuration)

	// ── 加载数据驱动配置 ──────────────────────────────────────
	dataDir := filepath.Join(".", "data")
	charFile := filepath.Join(dataDir, "characters.json")
	fieldFile := filepath.Join(dataDir, "fields.json")
	if err := character.LoadFromFile(charFile); err != nil {
		slog.Error("failed to load characters", "err", err)
		os.Exit(1)
	}
	if err := field.LoadFromFile(fieldFile); err != nil {
		slog.Error("failed to load field effects", "err", err)
		os.Exit(1)
	}
	configHash := computeConfigHash(charFile, fieldFile)
	slog.Info("game data loaded", "characters", len(character.All()), "fields", len(field.Pool), "configHash", configHash)

	// ── 依赖初始化（顺序有意义，被依赖的先初始化）──────────────
	playerMgr := player.NewManager()
	roomMgr := room.NewManager()
	queue := matchmaking.NewQueue(roomMgr)

	// 玩家断线时，从匹配队列中移除（防止匹配到已离线的玩家）
	playerMgr.OnDisconnect(func(p *player.Player) {
		queue.Dequeue(p.ID)
	})

	// ── 消息路由注册 ───────────────────────────────────────────
	router := network.NewRouter()

	// 匹配层：登录、入队、退队
	mmHandler := matchmaking.NewHandler(playerMgr, queue, roomMgr, configHash)
	mmHandler.RegisterAll(router)

	// 系统层：客户端 RTT 探测（原样回显时间戳）
	router.Register(protocol.MsgClientPingReq, func(s *network.Session, data []byte) {
		s.Send(protocol.MsgClientPingResp, data)
	})

	// 游戏层：角色选择、行动阶段所有操作
	gameHandler := game.NewHandler(playerMgr, roomMgr)
	gameHandler.RegisterAll(router)

	// 房间创建时，自动为该房间创建并启动游戏引擎
	roomMgr.OnRoomCreated(gameHandler.OnRoomCreated)

	// ── 启动服务器（支持 SIGINT/SIGTERM 优雅关停）──────────────
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := network.NewServer(cfg.ListenAddr, router)
	if err := srv.Start(ctx); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// computeConfigHash 对配置文件内容计算 SHA256 摘要（取前16字符），用于客户端缓存校验。
func computeConfigHash(files ...string) string {
	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			slog.Error("computeConfigHash: read failed", "file", f, "err", err)
			continue
		}
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
