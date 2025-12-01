package tray

import (
	"context"
	_ "embed"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed icon.ico
var icon []byte

// Manager 托盘管理器
type Manager struct {
	ctx       context.Context
	proxyItem *systray.MenuItem
	isRunning bool
}

// NewManager 创建托盘管理器
func NewManager() *Manager {
	return &Manager{}
}

// Setup 初始化托盘
func (m *Manager) Setup(ctx context.Context) {
	m.ctx = ctx
	go systray.Run(m.onReady, m.onExit)
}

// onReady 托盘就绪
func (m *Manager) onReady() {
	systray.SetIcon(icon)
	systray.SetTitle("Chameleon")
	systray.SetTooltip("点击显示窗口")

	mShow := systray.AddMenuItem("显示窗口", "显示主窗口")
	m.proxyItem = systray.AddMenuItem("启动代理", "启动代理")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出应用")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				m.ShowWindow()
			case <-m.proxyItem.ClickedCh:
				m.toggleProxy()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

// onExit 托盘退出
func (m *Manager) onExit() {
	if m.ctx != nil {
		runtime.Quit(m.ctx)
	}
}

// ShowWindow 显示窗口
func (m *Manager) ShowWindow() {
	if m.ctx != nil {
		runtime.WindowShow(m.ctx)
		runtime.WindowUnminimise(m.ctx)
	}
	//runtime.EventsEmit(m.ctx, "app:restore")
}

// toggleProxy 切换代理状态
func (m *Manager) toggleProxy() {
	if m.isRunning {
		runtime.EventsEmit(m.ctx, "stop_proxy")
	} else {
		runtime.EventsEmit(m.ctx, "start_proxy")
	}
}

// UpdateProxyStatus 更新代理状态
func (m *Manager) UpdateProxyStatus(running bool) {
	m.isRunning = running
	if m.proxyItem != nil {
		if running {
			m.proxyItem.SetTitle("停止代理")
			m.proxyItem.SetTooltip("停止代理")
		} else {
			m.proxyItem.SetTitle("启动代理")
			m.proxyItem.SetTooltip("启动代理")
		}
	}
}

// HideWindow 隐藏窗口到托盘
func (m *Manager) HideWindow() {
	if m.ctx != nil {
		runtime.WindowHide(m.ctx)
	}
	//runtime.EventsEmit(m.ctx, "app:minimize")
}

// Quit 退出应用
func (m *Manager) Quit() {
	systray.Quit()
}
