package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const (
	configFileName = "config.json"
)

// ConfigStorage 配置存储管理器
type ConfigStorage struct {
	configPath string
	mu         sync.RWMutex
}

// NewConfigStorage 创建新的配置存储管理器
func NewConfigStorage() *ConfigStorage {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("无法获取工作目录: %v", err)
		wd = "."
	}

	configPath := filepath.Join(wd, configFileName)
	return &ConfigStorage{
		configPath: configPath,
	}
}

// LoadFromFile 从文件加载配置
func (cs *ConfigStorage) LoadFromFile() (*Config, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// 检查文件是否存在
	if _, err := os.Stat(cs.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", cs.configPath)
	}

	// 读取文件
	data, err := os.ReadFile(cs.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	log.Printf("从文件加载配置成功: %s", cs.configPath)
	return &cfg, nil
}

// SaveToFile 保存配置到文件
func (cs *ConfigStorage) SaveToFile(cfg *Config) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(cs.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	log.Printf("配置已保存到文件: %s", cs.configPath)
	return nil
}

// FileExists 检查配置文件是否存在
func (cs *ConfigStorage) FileExists() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	_, err := os.Stat(cs.configPath)
	return err == nil
}

// GetConfigPath 获取配置文件路径
func (cs *ConfigStorage) GetConfigPath() string {
	return cs.configPath
}

// MergeWithEnv 合并环境变量配置
func (cs *ConfigStorage) MergeWithEnv(fileCfg *Config) *Config {
	// 创建环境变量配置
	envCfg, err := LoadConfigFromEnv()
	if err != nil {
		log.Printf("从环境变量加载配置失败: %v", err)
		return fileCfg
	}

	// 合并配置（环境变量优先级更高）
	merged := *fileCfg

	// 只有在环境变量设置了值时才覆盖
	if envCfg.HTTPPort != "" && envCfg.HTTPPort != "8088" {
		merged.HTTPPort = envCfg.HTTPPort
	}
	if envCfg.QQWSURL != "" && envCfg.QQWSURL != "ws://127.0.0.1:3009" {
		merged.QQWSURL = envCfg.QQWSURL
	}
	if envCfg.QQHTTPURL != "" && envCfg.QQHTTPURL != "http://127.0.0.1:6700" {
		merged.QQHTTPURL = envCfg.QQHTTPURL
	}
	if envCfg.QQReverseWS != "" {
		merged.QQReverseWS = envCfg.QQReverseWS
	}
	if envCfg.QQAccessToken != "" {
		merged.QQAccessToken = envCfg.QQAccessToken
	}
	if len(envCfg.QQGroupID) > 0 {
		merged.QQGroupID = envCfg.QQGroupID
	}
	if envCfg.ConnectionMode != "" && envCfg.ConnectionMode != "websocket" {
		merged.ConnectionMode = envCfg.ConnectionMode
	}

	return &merged
}

// LoadConfigFromEnv 仅从环境变量加载配置（用于合并）
func LoadConfigFromEnv() (*Config, error) {
	var cfg Config

	// 从环境变量加载配置
	if port := os.Getenv("HTTP_PORT"); port != "" {
		cfg.HTTPPort = port
	}
	if wsURL := os.Getenv("QQ_WS_URL"); wsURL != "" {
		cfg.QQWSURL = wsURL
	}
	if httpURL := os.Getenv("QQ_HTTP_URL"); httpURL != "" {
		cfg.QQHTTPURL = httpURL
	}
	if reverseWS := os.Getenv("QQ_REVERSE_WS"); reverseWS != "" {
		cfg.QQReverseWS = reverseWS
	}
	if token := os.Getenv("QQ_ACCESS_TOKEN"); token != "" {
		cfg.QQAccessToken = token
	}
	if mode := os.Getenv("CONNECTION_MODE"); mode != "" {
		cfg.ConnectionMode = mode
	}

	// 清理URL
	cfg.QQWSURL = TrimSpace(cfg.QQWSURL)
	cfg.QQHTTPURL = TrimSpace(cfg.QQHTTPURL)
	cfg.QQReverseWS = TrimSpace(cfg.QQReverseWS)

	return &cfg, nil
}

// GetDefaultCommandOutputSettings 获取默认的指令输出设置
func GetDefaultCommandOutputSettings() CommandOutputSettings {
	return CommandOutputSettings{
		RollCommand:    "掷骰结果: {dice} = {total} ({values})",
		CocCommand:     "COC角色生成: 力量{str} 体质{con} 体型{siz} 敏捷{dex} 外貌{app} 智力{int} 意志{pow} 教育{edu} 幸运{luck}",
		DndCommand:     "DND角色生成: 力量{str} 敏捷{dex} 体质{con} 智力{int} 感知{wis} 魅力{cha}",
		HelpCommand:    "可用命令: .r [骰子表达式] - 掷骰子, .coc - COC角色生成, .dnd - DND角色生成, .help - 显示帮助",
		UnknownCommand: "未知命令: {command}",
	}
}

// TrimSpace 辅助函数，处理空字符串
func TrimSpace(s string) string {
	if s == "" {
		return ""
	}
	return s
}
