package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config 保存所有服务端可配置参数，用于替代散落在各模块的硬编码常量。
// 默认值适用于开发环境；生产部署通过环境变量覆盖。
type Config struct {
	// 网络
	ListenAddr string // TCP 监听地址，默认 "0.0.0.0:43966"
	RateLimit  int    // 每秒允许的最大消息数，默认 30

	// 心跳
	PingInterval time.Duration // Ping 发送间隔，默认 15s
	PongTimeout  time.Duration // Pong 超时判定，默认 35s

	// 游戏
	TurnDuration       time.Duration // 行动阶段每回合时长，默认 90s
	DisconnectTimerSec int           // 断线后回合缩短至（秒），默认 15

	// 玩家
	ReconnectTTL time.Duration // 断线重连凭证有效期，默认 3m

	// 日志
	LogLevel slog.Level // 日志级别，默认 Debug
}

// Load 加载配置：先填入默认值，再用环境变量覆盖。
func Load() *Config {
	c := &Config{
		ListenAddr:         "0.0.0.0:43966",
		RateLimit:          30,
		PingInterval:       15 * time.Second,
		PongTimeout:        35 * time.Second,
		TurnDuration:       90 * time.Second,
		DisconnectTimerSec: 15,
		ReconnectTTL:       3 * time.Minute,
		LogLevel:           slog.LevelDebug,
	}

	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		c.ListenAddr = v
	}
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.RateLimit = n
		}
	}
	if v := os.Getenv("TURN_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.TurnDuration = d
		}
	}
	if v := os.Getenv("RECONNECT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.ReconnectTTL = d
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		switch v {
		case "debug":
			c.LogLevel = slog.LevelDebug
		case "info":
			c.LogLevel = slog.LevelInfo
		case "warn":
			c.LogLevel = slog.LevelWarn
		case "error":
			c.LogLevel = slog.LevelError
		}
	}

	return c
}
