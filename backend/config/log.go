package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/phsym/console-slog"
)

// InitLogger 初始化日志系统
func InitLogger(dataDir string, config *LogConfig) {
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	if config.Console && config.File {
		// 同时输出到控制台和文件
		logFile := filepath.Join(dataDir, "logs", "app.log")
		_ = os.MkdirAll(filepath.Dir(logFile), 0755)

		// 打开日志文件
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("无法打开日志文件", "error", err)
			// 如果文件打开失败，回退到仅控制台输出
			slog.SetDefault(slog.New(console.NewHandler(os.Stdout, &console.HandlerOptions{Level: level})))
			return
		}

		// 使用多路输出（写入到同时包含控制台和文件的 writer）
		multiWriter := io.MultiWriter(os.Stdout, file)

		logger := slog.New(console.NewHandler(multiWriter, &console.HandlerOptions{
			Level:   level,
			NoColor: true,
			//AddSource: true,
			//TimeFormat: "",
			//Theme:      nil,
		}))
		slog.SetDefault(logger)

		// 为文件使用文本处理器，为控制台使用彩色处理器
		//textHandler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		//	Level: level,
		//	ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		//		if a.Key == slog.TimeKey {
		//			// 格式化时间为更友好的格式
		//			a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05"))
		//		}
		//		return a
		//	},
		//})

		//logger := slog.New(textHandler)
		//slog.SetDefault(logger)

		// 确保日志文件立即刷新
		logger.Info("日志系统初始化完成", "file", logFile, "level", config.Level)

	} else if config.Console {
		// 仅控制台输出
		slog.SetDefault(slog.New(console.NewHandler(os.Stdout, &console.HandlerOptions{Level: level})))

	} else {
		// 静默模式
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
	}
}
