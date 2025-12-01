package config

// Config 配置结构体
type Config struct {
	General *GeneralConfig `toml:"general" comment:"通用配置"`
	UI      *UIConfig      `toml:"ui" comment:"UI配置"`
	Proxy   *ProxyConfig   `toml:"proxy" comment:"代理配置"`
	Log     *LogConfig     `toml:"log" comment:"日志配置"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Mode          string `toml:"mode" comment:"代理模式：http|host"` // 代理模式：http/socks/host
	Port          uint16 `toml:"port" comment:"http代理服务器监听端口"`  // 服务器监听端口，非host模式可配置
	CertInstalled bool   `toml:"cert_installed" comment:"CA 证书安装状态"`
}

// GeneralConfig 通用配置
type GeneralConfig struct {
	AutoStart      bool `toml:"auto_start" comment:"开机自动启动"`     // 自动启动
	StartMinimized bool `toml:"start_minimized" comment:"最小化启动"` // 最小化启动
	//StartAsAdmin   bool   `toml:"start_as_admin" comment:"以管理员权限启动"`             // 以管理员权限启动
	CloseAction string `toml:"close_action" comment:"关闭动作：ask|minimize|exit"` // 关闭动作
}

// UIConfig UI配置
type UIConfig struct {
	Language string `toml:"language" comment:"语言"` // 语言
	Theme    string `toml:"theme" comment:"主题"`    // 主题
	Width    uint16 `toml:"width" comment:"窗口宽度"`  // 宽度
	Height   uint16 `toml:"height" comment:"窗口高度"` // 高度
}

// LogConfig 日志配置
type LogConfig struct {
	Level   string `toml:"level" comment:"日志等级"`       // 日志等级
	File    bool   `toml:"file" comment:"是否保存到文件"`     // 是否保存到文件
	Console bool   `toml:"console" comment:"是否输出到控制台"` // 是否输出到控制台
}

// Manager 配置管理器
type Manager struct {
	configPath string
	config     *Config
}

// getDefaultConfig 返回默认配置
func getDefaultConfig() *Config {
	return &Config{
		General: &GeneralConfig{
			AutoStart:      false,
			StartMinimized: false,
			CloseAction:    "ask",
			//StartAsAdmin:   false,
		},
		UI: &UIConfig{
			Language: "zh-CN",
			Theme:    "light",
			Width:    1200,
			Height:   800,
		},
		Proxy: &ProxyConfig{
			Mode: "http",
			Port: 9527,
		},
		Log: &LogConfig{
			Level:   "debug",
			File:    true,
			Console: true,
		},
	}
}
