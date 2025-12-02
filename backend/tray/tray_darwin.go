//go:build darwin

package tray

import (
	"context"
	_ "embed"

	"github.com/progrium/darwinkit/dispatch"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed icon.png
var iconData []byte

// Manager 托盘管理器 (macOS 使用 darwinkit 实现)
type Manager struct {
	ctx        context.Context
	statusItem appkit.StatusItem
	proxyItem  appkit.MenuItem
	isRunning  bool
}

// NewManager 创建托盘管理器
func NewManager() *Manager {
	return &Manager{}
}

// Setup 初始化托盘
func (m *Manager) Setup(ctx context.Context) {
	m.ctx = ctx
	dispatch.MainQueue().DispatchAsync(func() {
		m.setupStatusItem()
	})
}

// setupStatusItem 创建状态栏图标和菜单
func (m *Manager) setupStatusItem() {
	m.statusItem = appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
	objc.Retain(&m.statusItem)

	// 设置图标
	imgData := foundation.NewDataWithBytesLength(iconData, uint(len(iconData)))
	img := appkit.NewImageWithData(imgData)
	img.SetSize(foundation.Size{Width: 18, Height: 18})
	img.SetTemplate(true)
	m.statusItem.Button().SetImage(img)
	m.statusItem.Button().SetToolTip("Chameleon")

	// 创建菜单
	menu := appkit.NewMenu()

	showItem := appkit.NewMenuItemWithTitleActionKeyEquivalent("显示窗口", objc.Sel("showWindow:"), "")
	showItem.SetTarget(objc.WrapAsProtocol[objc.PObject](m))
	menu.AddItem(showItem)

	m.proxyItem = appkit.NewMenuItemWithTitleActionKeyEquivalent("启动代理", objc.Sel("toggleProxy:"), "")
	m.proxyItem.SetTarget(objc.WrapAsProtocol[objc.PObject](m))
	menu.AddItem(m.proxyItem)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	quitItem := appkit.NewMenuItemWithTitleActionKeyEquivalent("退出", objc.Sel("quitApp:"), "q")
	quitItem.SetTarget(objc.WrapAsProtocol[objc.PObject](m))
	menu.AddItem(quitItem)

	m.statusItem.SetMenu(menu)
}

// Selector 实现 objc 选择器
func (m *Manager) Selector_showWindow_(sender objc.Object) {
	m.ShowWindow()
}

func (m *Manager) Selector_toggleProxy_(sender objc.Object) {
	if m.isRunning {
		runtime.EventsEmit(m.ctx, "stop_proxy")
	} else {
		runtime.EventsEmit(m.ctx, "start_proxy")
	}
}

func (m *Manager) Selector_quitApp_(sender objc.Object) {
	m.Quit()
}

// ShowWindow 显示窗口
func (m *Manager) ShowWindow() {
	if m.ctx != nil {
		runtime.WindowShow(m.ctx)
		runtime.WindowUnminimise(m.ctx)
	}
}

// UpdateProxyStatus 更新代理状态
func (m *Manager) UpdateProxyStatus(running bool) {
	m.isRunning = running
	dispatch.MainQueue().DispatchAsync(func() {
		if m.proxyItem.IsNil() {
			return
		}
		if running {
			m.proxyItem.SetTitle("停止代理")
		} else {
			m.proxyItem.SetTitle("启动代理")
		}
	})
}

// HideWindow 隐藏窗口
func (m *Manager) HideWindow() {
	if m.ctx != nil {
		runtime.WindowHide(m.ctx)
	}
}

// Quit 退出应用
func (m *Manager) Quit() {
	if m.ctx != nil {
		runtime.Quit(m.ctx)
	}
}
