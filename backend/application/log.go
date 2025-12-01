package application

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GetLogs returns the latest log entries
func (app *App) GetLogs(lines int) ([]string, error) {
	// 获取应用同级目录下的 data 文件夹
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, "data", "logs", "app.log")

	// 如果日志文件不存在，返回空数组
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(logFile)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	// 读取所有行
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	// 返回最后的 N 行
	if lines <= 0 || lines >= len(allLines) {
		return allLines, nil
	}

	return allLines[len(allLines)-lines:], nil
}

// SearchLogs 在日志中搜索关键词
func (app *App) SearchLogs(keyword string, maxResults int) ([]string, error) {
	if strings.TrimSpace(keyword) == "" {
		return app.GetLogs(1000) // 返回最近的1000行
	}

	// 获取应用同级目录下的 data 文件夹
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, "data", "logs", "app.log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(logFile)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	var results []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			results = append(results, line)
			if len(results) >= maxResults {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	return results, nil
}

// ClearLogs 清空日志文件
func (app *App) ClearLogs() error {
	// 获取应用同级目录下的 data 文件夹
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, "data", "logs", "app.log")

	// 如果文件不存在，直接返回成功
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil
	}

	// 清空文件内容
	file, err := os.OpenFile(logFile, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("清空日志文件失败: %w", err)
	}
	defer file.Close()

	slog.Info("日志文件已清空", "file", logFile)
	return nil
}
