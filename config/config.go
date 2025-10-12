package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Config 应用配置
type Config struct {
	HTTPPort      string  `env:"HTTP_PORT" envDefault:"8088"`
	QQWSURL       string  `env:"QQ_WS_URL" envDefault:"ws://127.0.0.1:3009"`
	QQHTTPURL     string  `env:"QQ_HTTP_URL" envDefault:"http://127.0.0.1:6700"`
	QQReverseWS   string  `env:"QQ_REVERSE_WS" envDefault:""`
	QQAccessToken string  `env:"QQ_ACCESS_TOKEN" envDefault:""`
	QQGroupID     []int64 `env:"QQ_GROUP_ID" envSeparator:","`
	ConnectionMode string `env:"CONNECTION_MODE" envDefault:"websocket"` // websocket, http, reverse_websocket
}

// ConnectionMode 连接模式枚举
const (
	ModeWebSocket      = "websocket"
	ModeHTTP           = "http" 
	ModeReverseWebSocket = "reverse_websocket"
)

// Validate 验证配置
func (c *Config) Validate() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT 不能为空")
	}

	switch c.ConnectionMode {
	case ModeWebSocket:
		if c.QQWSURL == "" {
			return fmt.Errorf("WebSocket模式需要配置 QQ_WS_URL")
		}
	case ModeHTTP:
		if c.QQHTTPURL == "" {
			return fmt.Errorf("HTTP模式需要配置 QQ_HTTP_URL")
		}
	case ModeReverseWebSocket:
		if c.QQReverseWS == "" {
			return fmt.Errorf("反向WebSocket模式需要配置 QQ_REVERSE_WS")
		}
	default:
		return fmt.Errorf("不支持的连接模式: %s", c.ConnectionMode)
	}

	return nil
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("配置加载失败: %w", err)
	}

	// 清理URL
	cfg.QQWSURL = strings.TrimSpace(cfg.QQWSURL)
	cfg.QQHTTPURL = strings.TrimSpace(cfg.QQHTTPURL)
	cfg.QQReverseWS = strings.TrimSpace(cfg.QQReverseWS)

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if len(cfg.QQGroupID) == 0 {
		log.Println("未配置QQ_GROUP_ID，将处理所有群组消息")
	}

	log.Printf("配置加载成功 - 连接模式: %s", cfg.ConnectionMode)
	return &cfg, nil
}
