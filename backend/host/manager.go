package host

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type Manager struct{}

type Entity struct {
	ip     string // 固定为127.0.0.1
	host   string // 需要被劫持的host
	remark string // 备注：Chameleon-{host}
}

// getHostsPath 获取系统hosts文件路径
func getHostsPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\System32\\drivers\\etc\\hosts"
	}
	return "/etc/hosts"
}

// AddHosts 添加host集合
func (m *Manager) AddHosts(hosts []string) error {
	hostsPath := getHostsPath()
	file, err := os.OpenFile(hostsPath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开hosts文件失败: %w", err)
	}
	defer file.Close()

	for _, host := range hosts {
		entity := Entity{
			ip:     "127.0.0.1",
			host:   host,
			remark: fmt.Sprintf("Chameleon-%s", host),
		}
		line := fmt.Sprintf("%s\t%s\t# %s\n", entity.ip, entity.host, entity.remark)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("写入hosts失败: %w", err)
		}
	}
	return nil
}

// RemoveHosts 移除添加到系统host中的数据
func (m *Manager) RemoveHosts() error {
	hostsPath := getHostsPath()
	file, err := os.Open(hostsPath)
	if err != nil {
		return fmt.Errorf("打开hosts文件失败: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "Chameleon-") {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取hosts文件失败: %w", err)
	}

	return os.WriteFile(hostsPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
