package updater

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/creativeprojects/go-selfupdate"
)

const (
	repoOwner = "sbgayhub"
	repoName  = "chameleon"
)

// Version 编译时注入的版本号
var Version = "dev"

// UpdateInfo 更新信息
type UpdateInfo struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseURL"`
}

// Manager 更新管理器
type Manager struct {
	version string
}

// NewManager 创建更新管理器
func NewManager() *Manager {
	return &Manager{version: Version}
}

// CheckUpdate 检查更新
func (m *Manager) CheckUpdate() (*UpdateInfo, error) {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		slog.Error("检查更新失败", "error", err)
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: m.version,
		HasUpdate:      false,
	}

	if !found {
		slog.Info("未找到发布版本")
		return info, nil
	}

	info.LatestVersion = latest.Version()
	info.ReleaseNotes = latest.ReleaseNotes
	info.ReleaseURL = latest.URL

	if latest.GreaterThan(m.version) {
		info.HasUpdate = true
		slog.Info("发现新版本", "current", m.version, "latest", latest.Version())
	} else {
		slog.Info("当前已是最新版本", "version", m.version)
	}

	return info, nil
}

// DoUpdate 执行更新
func (m *Manager) DoUpdate() (bool, error) {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		return false, fmt.Errorf("检测版本失败: %w", err)
	}
	if !found {
		return false, fmt.Errorf("未找到适用于 %s/%s 的发布版本", runtime.GOOS, runtime.GOARCH)
	}

	if !latest.GreaterThan(m.version) {
		slog.Info("当前已是最新版本")
		return false, nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return false, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	slog.Info("开始更新", "from", m.version, "to", latest.Version())

	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return false, fmt.Errorf("更新失败: %w", err)
	}

	slog.Info("更新完成", "version", latest.Version())
	return true, nil
}

// GetVersion 获取当前版本
func (m *Manager) GetVersion() string {
	return m.version
}

// autoUpdate 自动更新
func (m *Manager) autoUpdate(ctx context.Context) {
	updated, err := m.DoUpdate()
	if err != nil {
		slog.Error("自动更新失败", "error", err)
		wailsruntime.EventsEmit(ctx, "update_error", err.Error())
		return
	}
	if updated {
		wailsruntime.EventsEmit(ctx, "update_completed", "更新完成，请重启应用以应用更新")
	}
}

// CheckUpdateOnStartup 启动时检查更新
func (m *Manager) CheckUpdateOnStartup(ctx context.Context, autoUpdate bool) {
	info, err := m.CheckUpdate()
	if err != nil {
		slog.Warn("启动时检查更新失败", "error", err)
		return
	}
	if info.HasUpdate {
		// 通知前端有更新
		wailsruntime.EventsEmit(ctx, "update_available", info)

		// 如果启用了自动更新，则自动执行更新
		if autoUpdate {
			go m.autoUpdate(ctx)
		}
	}
}
