package application

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sbgayhub/chameleon/backend/server"
)

// StartProxy 启动代理服务器
func (app *App) StartProxy() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.running {
		return fmt.Errorf("代理服务器已在运行")
	}

	// 检查代理管理器是否初始化
	if app.ChannelMgr == nil {
		return fmt.Errorf("代理管理器未初始化")
	}

	// 检查统计管理器是否初始化
	if app.StatsMgr == nil {
		return fmt.Errorf("统计管理器未初始化")
	}

	// 获取配置
	config := app.ConfigMgr.GetConfig()
	if config == nil {
		return fmt.Errorf("应用配置未初始化")
	}

	if config.Proxy.Mode == "host" {
		app.Server = server.NewHostServer(config.Proxy, app.HostMgr, app.ChannelMgr, app.StatsMgr)
	} else {
		app.Server = server.NewProxyServer(config.Proxy, app.ChannelMgr, app.StatsMgr)
	}
	if err := app.Server.Start(); err != nil {
		return fmt.Errorf("启动代理服务器失败: %w", err)
	}

	app.running = true
	app.startTime = time.Now()
	app.TrayMgr.UpdateProxyStatus(true)

	slog.Info("代理服务器启动成功", "mode", config.Proxy.Mode)
	return nil
}

// StopProxy 停止代理服务器
func (app *App) StopProxy() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.running {
		return fmt.Errorf("代理服务器未运行")
	}

	if err := app.Server.Stop(); err != nil {
		return fmt.Errorf("停止代理服务器失败: %w", err)
	}

	app.running = false
	app.TrayMgr.UpdateProxyStatus(false)
	slog.Info("代理服务器已停止")
	return nil
}

// GetProxyStatus 获取代理状态
func (app *App) GetProxyStatus() *ProxyStatus {
	app.mu.RLock()
	defer app.mu.RUnlock()

	// 获取配置信息，使用默认值
	var port uint16 = 8080
	var mode string = "http"
	config := app.ConfigMgr.GetConfig()
	if config != nil {
		port = config.Proxy.Port
		mode = config.Proxy.Mode
	}

	status := &ProxyStatus{
		IsRunning:         app.running,
		Port:              port,
		Mode:              mode,
		ActiveConnections: 0,
		TotalRequests:     0,
	}

	// 安全检查：如果统计管理器存在，获取总请求数
	if app.StatsMgr != nil {
		status.TotalRequests = app.StatsMgr.GetTotalRequests()
	}

	if app.running && !app.startTime.IsZero() {
		status.StartTime = app.startTime.Unix()
		status.Uptime = int64(time.Since(app.startTime).Seconds())
	}

	return status
}
