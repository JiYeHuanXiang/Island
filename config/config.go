package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Config 应用配置
type Config struct {
	HTTPPort       string                `env:"HTTP_PORT" envDefault:"8088"`
	QQWSURL        string                `env:"QQ_WS_URL" envDefault:""`
	QQHTTPURL      string                `env:"QQ_HTTP_URL" envDefault:""`
	QQReverseWS    string                `env:"QQ_REVERSE_WS" envDefault:""`
	QQAccessToken  string                `env:"QQ_ACCESS_TOKEN" envDefault:""`
	QQGroupID      []int64               `env:"QQ_GROUP_ID" envSeparator:","`
	ConnectionMode string                `env:"CONNECTION_MODE" envDefault:"websocket"` // websocket, http, reverse_websocket
	CommandOutput  CommandOutputSettings `env:"COMMAND_OUTPUT" envDefault:""`
}

// CommandOutputSettings 指令输出文本设置
type CommandOutputSettings struct {
	RollCommand    string `json:"roll_command"`
	CocCommand     string `json:"coc_command"`
	DndCommand     string `json:"dnd_command"`
	HelpCommand    string `json:"help_command"`
	UnknownCommand string `json:"unknown_command"`
}

// ConnectionMode 连接模式枚举
const (
	ModeWebSocket        = "websocket"
	ModeHTTP             = "http"
	ModeReverseWebSocket = "reverse_websocket"
)

// Validate 验证配置（严格验证，用于运行时）
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

// ValidateForSave 验证配置（宽松验证，用于保存配置）
func (c *Config) ValidateForSave() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP端口不能为空")
	}

	// 只验证连接模式是否有效，不验证具体URL是否为空
	switch c.ConnectionMode {
	case ModeWebSocket, ModeHTTP, ModeReverseWebSocket:
		// 允许URL为空，因为用户可能稍后配置
		return nil
	default:
		return fmt.Errorf("不支持的连接模式: %s", c.ConnectionMode)
	}
}

// LoadConfig 加载配置（支持文件和环境变量）
func LoadConfig() (*Config, error) {
	storage := NewConfigStorage()

	// 优先从文件加载配置
	if storage.FileExists() {
		fileCfg, err := storage.LoadFromFile()
		if err != nil {
			log.Printf("从文件加载配置失败: %v，回退到环境变量", err)
		} else {
			// 合并环境变量配置（环境变量优先级更高）
			mergedCfg := storage.MergeWithEnv(fileCfg)

			if len(mergedCfg.QQGroupID) == 0 {
				log.Println("未配置QQ_GROUP_ID，将处理所有群组消息")
			}

			log.Printf("配置加载成功 - 连接模式: %s (来自文件和环境变量)", mergedCfg.ConnectionMode)
			return mergedCfg, nil
		}
	}

	// 回退到环境变量配置
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

	log.Printf("配置加载成功 - 连接模式: %s (来自环境变量)", cfg.ConnectionMode)
	return &cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg *Config) error {
	storage := NewConfigStorage()
	return storage.SaveToFile(cfg)
}

// GetConfigStorage 获取配置存储实例
func GetConfigStorage() *ConfigStorage {
	return NewConfigStorage()
}
