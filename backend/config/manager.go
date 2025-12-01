package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// NewManager 创建配置管理器
func NewManager(dataDir string) *Manager {
	configPath := filepath.Join(dataDir, "config.toml")

	manager := &Manager{
		configPath: configPath,
		config:     getDefaultConfig(),
	}

	// 启动时加载配置，失败则创建默认配置
	if err := manager.Load(); err != nil {
		slog.Warn("加载配置失败，使用默认配置", "error", err)
		_ = manager.Save()
	}

	InitLogger(dataDir, manager.config.Log)

	return manager
}

// Load 加载配置文件
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	return toml.Unmarshal(data, m.config)
}

// Save 保存配置文件
func (m *Manager) Save() error {
	_ = os.MkdirAll(filepath.Dir(m.configPath), 0755)

	data, err := toml.Marshal(m.config)
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// UpdateConfig 更新配置
func (m *Manager) UpdateConfig(config *Config) error {
	m.config = config
	return m.Save()
}

// UpdateProxyConfig 更新代理配置
func (m *Manager) UpdateProxyConfig(proxy *ProxyConfig) error {
	m.config.Proxy = proxy
	return m.Save()
}

// UpdateGeneralConfig 更新通用配置
func (m *Manager) UpdateGeneralConfig(general *GeneralConfig) error {
	m.config.General = general
	return m.Save()
}

// UpdateUIConfig 更新UI配置
func (m *Manager) UpdateUIConfig(ui *UIConfig) error {
	m.config.UI = ui
	return m.Save()
}

// UpdateLogConfig 更新日志配置
func (m *Manager) UpdateLogConfig(log *LogConfig) error {
	m.config.Log = log
	return m.Save()
}

//
//// LoadConfig 加载配置文件 (包级别函数)
//func LoadConfig() (*Config, error) {
//	dataDir := getConfigDataDir()
//	if dataDir == "" {
//		return nil, fmt.Errorf("无法获取数据目录")
//	}
//
//	configPath := filepath.Join(dataDir, "config.toml")
//
//	data, err := os.ReadFile(configPath)
//	if err != nil {
//		return nil, err
//	}
//
//	var config Config
//	if err := toml.Unmarshal(data, &config); err != nil {
//		return nil, err
//	}
//
//	return &config, nil
//}
//
//// SaveConfig 保存配置文件 (包级别函数)
//func SaveConfig(config *Config) error {
//	// 使用本地getConfigDataDir函数确保路径一致性
//	dataDir := getConfigDataDir()
//	if dataDir == "" {
//		return fmt.Errorf("无法获取数据目录")
//	}
//
//	configPath := filepath.Join(dataDir, "config.toml")
//
//	_ = os.MkdirAll(filepath.Dir(configPath), 0755)
//
//	data, err := toml.Marshal(config)
//	if err != nil {
//		return err
//	}
//
//	return os.WriteFile(configPath, data, 0644)
//}
//
//// GetDefaultConfig 获取默认配置 (包级别函数)
//func GetDefaultConfig() *Config {
//	return getDefaultConfig()
//}
