package application

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sbgayhub/chameleon/backend/certificate"
	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/config"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/convert/anthropic"
	"github.com/sbgayhub/chameleon/backend/convert/openai"
	"github.com/sbgayhub/chameleon/backend/host"
	"github.com/sbgayhub/chameleon/backend/server"
	"github.com/sbgayhub/chameleon/backend/statistics"
	"github.com/sbgayhub/chameleon/backend/tray"
	"github.com/wailsapp/wails/v2/pkg/options"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProxyStatus 代理服务状态信息
type ProxyStatus struct {
	IsRunning         bool   `json:"isRunning"`
	StartTime         int64  `json:"startTime"`
	Uptime            int64  `json:"uptime"`
	Port              uint16 `json:"port"`
	ActiveConnections int    `json:"activeConnections"`
	TotalRequests     int64  `json:"totalRequests"`
	Mode              string `json:"mode"`
}

// App struct
type App struct {
	running   bool
	startTime time.Time

	HostMgr    *host.Manager
	ConfigMgr  *config.Manager
	ChannelMgr *channel.Manager
	TrayMgr    *tray.Manager
	StatsMgr   *statistics.Manager
	CertMgr    *certificate.CertManager

	Server server.Server

	ctx context.Context
	mu  sync.RWMutex
}

// getDataDir 获取应用数据目录（exe同级的data文件夹）
func getDataDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// 如果获取失败，回退到用户目录
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" {
			homeDir = os.TempDir()
		}
		return filepath.Join(homeDir, "github.com/sbgayhub/chameleon")
	}

	// 获取可执行文件所在目录
	dir := filepath.Join(filepath.Dir(exePath), "data")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	return dir
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 获取数据目录
	dataDir := getDataDir()

	// 初始化配置管理器
	configMgr := config.NewManager(dataDir)
	trayMgr := tray.NewManager()

	// 初始化其他管理器
	channelMgr := channel.NewManager(dataDir)
	statsMgr := statistics.NewManager(dataDir)
	certMgr := certificate.NewManager(dataDir)

	// 加载代理配置
	if err := channelMgr.LoadFromFile(); err != nil {
		slog.Warn("加载代理配置失败", "error", err)
	}

	return &App{
		CertMgr:    certMgr,
		TrayMgr:    trayMgr,
		ConfigMgr:  configMgr,
		ChannelMgr: channelMgr,
		StatsMgr:   statsMgr,
		running:    false,
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (app *App) Startup(ctx context.Context) {
	app.ctx = ctx
	// 初始化托盘
	app.TrayMgr.Setup(ctx)

	// 订阅事件
	runtime.EventsOn(ctx, "start_proxy", func(data ...interface{}) {
		err := app.StartProxy()
		if err != nil {
			return
		}
		app.TrayMgr.UpdateProxyStatus(true)
	})

	runtime.EventsOn(ctx, "stop_proxy", func(data ...interface{}) {
		err := app.StopProxy()
		if err != nil {
			return
		}
		app.TrayMgr.UpdateProxyStatus(false)
	})

	// 注册转换器
	anthropic.RegistryOpenAIConverter()
	anthropic.RegistryGeminiConverter()
	anthropic.RegistryAnthropicConverter()
	openai.RegistryOpenAIConverter()
	openai.RegistryAnthropicConverter()

	slog.Info("app start")
}

// GetConverterNames 获取转换器名称列表
func (app *App) GetConverterNames() []string {
	return convert.GetRegistry().Names()
}

// HandleWindowClose 处理窗口关闭事件
func (app *App) HandleWindowClose() string {
	cfg := app.ConfigMgr.GetConfig()
	return cfg.General.CloseAction
}

// MinimizeToTray 最小化到托盘
func (app *App) MinimizeToTray() {
	app.TrayMgr.HideWindow()
}

// QuitApp 退出应用
func (app *App) QuitApp() {
	runtime.Quit(app.ctx)
}

// SaveCloseAction 保存关闭动作配置
func (app *App) SaveCloseAction(action string) error {
	cfg := app.ConfigMgr.GetConfig()
	cfg.General.CloseAction = action
	return app.ConfigMgr.Save()
}

func (app *App) OnSecondInstanceLaunch(data options.SecondInstanceData) {
	runtime.WindowUnminimise(app.ctx)
	runtime.Show(app.ctx)
}
