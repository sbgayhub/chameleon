package certificate

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// https://github.com/golang/go/issues/24652#issuecomment-399826583
var trustSettings []interface{}

// getPasswordWithGUI 使用 GUI 弹框获取用户密码
func getPasswordWithGUI(title, prompt string) (string, error) {
	script := fmt.Sprintf(`
		tell application "System Events"
			activate
		end tell

		set userChoice to display dialog "%s" default answer "" buttons {"取消", "安装"} default button "安装" with title "%s" with hidden answer

		if button returned of userChoice is "安装" then
			return text returned of userChoice
		else
			error "用户取消"
		end if
	`, prompt, title)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "用户取消") {
			return "", fmt.Errorf("用户取消了密码输入")
		}
		return "", fmt.Errorf("获取密码失败: %v", err)
	}

	password := strings.TrimSpace(string(output))
	if password == "" {
		return "", fmt.Errorf("密码不能为空")
	}

	return password, nil
}

// executeCommandWithGUIPassword 使用 GUI 密码输入执行需要 sudo 权限的命令
func executeCommandWithGUIPassword(cmdName string, args ...string) error {
	// 首先尝试不使用 sudo
	cmd := exec.Command(cmdName, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		slog.Info("命令执行成功（无需管理员权限）")
		return nil
	}

	// 如果失败，尝试使用 GUI 获取密码并执行 sudo 命令
	slog.Info("需要管理员权限，显示密码输入对话框")
	prompt := fmt.Sprintf("github.com/sbgayhub/chameleon 需要管理员权限来安装证书。\n\n请输入您的系统密码以继续安装。\n\n将要执行的命令: %s %s", cmdName, strings.Join(args, " "))

	password, err := getPasswordWithGUI("github.com/sbgayhub/chameleon 证书安装", prompt)
	if err != nil {
		return fmt.Errorf("获取密码失败: %v", err)
	}

	// 使用密码执行 sudo 命令
	sudoArgs := []string{"-S"}
	sudoArgs = append(sudoArgs, cmdName)
	sudoArgs = append(sudoArgs, args...)

	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdin = strings.NewReader(password + "\n")
	sudoCmd.Stdout = nil
	sudoCmd.Stderr = nil

	output, err = sudoCmd.CombinedOutput()
	if err != nil {
		// 检查是否是密码错误
		if strings.Contains(string(output), "incorrect password") || strings.Contains(string(output), "Authentication failure") {
			return fmt.Errorf("密码错误，请重试")
		}
		return fmt.Errorf("命令执行失败: %v, 输出: %s", err, string(output))
	}

	slog.Info("命令执行成功（使用管理员权限）")
	return nil
}

var _ = xml.Unmarshal(trustSettingsData, &trustSettings)
var trustSettingsData = []byte(`
<array>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAED
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>sslServer</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAEC
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>basicX509</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
</array>
`)

type DarwinCertInstaller struct{}

func init() {
	installer = DarwinCertInstaller{}
	slog.Debug("证书安装器创建成功")
}

func (d DarwinCertInstaller) Install(dir string, cert []byte) bool {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("创建目录失败", "dir", dir, "err", err.Error())
		return false
	}

	path := filepath.Join(dir, "chameleon.pem")

	// 创建或打开文件进行写入
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error("打开证书文件失败", "path", path, "err", err.Error())
		return false
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.Error("关闭文件失败", "path", path, "err", closeErr.Error())
		}
	}()

	if _, err := f.Write(cert); err != nil {
		slog.Error("证书文件写入失败", "path", path, "err", err.Error())
		return false
	}

	slog.Info("证书文件已保存", "path", path)
	return d.installCertificate(path)
}

// installCertificate 安装证书到系统钥匙串
func (d DarwinCertInstaller) installCertificate(certPath string) bool {
	// 首先尝试不使用 sudo 安装到用户钥匙串
	userCmd := exec.Command("security", "add-trusted-cert", "-d", "-k", filepath.Join(os.Getenv("HOME"), "Library/Keychains/login.keychain"), certPath)
	if out, err := userCmd.CombinedOutput(); err != nil {
		slog.Warn("用户钥匙串安装失败，尝试系统钥匙串", "err", err.Error(), "out", string(out))

		// 如果用户钥匙串安装失败，使用 GUI 密码输入尝试系统钥匙串
		err = executeCommandWithGUIPassword("security", "add-trusted-cert", "-d", "-k", "/Library/Keychains/System.keychain", certPath)
		if err != nil {
			slog.Error("系统钥匙串安装失败", "err", err.Error())

			// 如果是用户取消或密码错误，提供手动安装指导
			if strings.Contains(err.Error(), "用户取消") {
				slog.Info("用户取消了证书安装")
				return false
			}

			return d.handleInstallFailure(certPath, err.Error())
		} else {
			slog.Info("证书已成功安装到系统钥匙串")
			return true
		}
	} else {
		slog.Info("证书已成功安装到用户钥匙串")
		return true
	}
}

// handleInstallFailure 处理安装失败的情况，提供用户友好的指导
func (d DarwinCertInstaller) handleInstallFailure(certPath, output string) bool {
	if strings.Contains(output, "password") || strings.Contains(output, "sudo") {
		slog.Error("需要管理员权限来安装证书")
		slog.Error("请选择以下方式之一手动安装：")
		slog.Error("1. 在终端中运行：")
		slog.Error(fmt.Sprintf("   sudo security add-trusted-cert -d -k /Library/Keychains/System.keychain %s", certPath))
		slog.Error("2. 使用图形界面：")
		slog.Error("   - 双击证书文件打开钥匙串访问")
		slog.Error("   - 展开 '信任' 部分")
		slog.Error("   - 将 '使用此证书时' 设置为 '始终信任'")
		slog.Error("3. 打开证书文件所在目录进行手动安装：")
		slog.Error(fmt.Sprintf("   open %s", filepath.Dir(certPath)))

		// 尝试打开 Finder 显示证书文件
		if runtime.GOOS == "darwin" {
			openCmd := exec.Command("open", filepath.Dir(certPath))
			if err := openCmd.Run(); err != nil {
				slog.Warn("无法打开 Finder", "err", err.Error())
			}
		}
	}
	return false
}

func (d DarwinCertInstaller) Uninstall(serial *big.Int) bool {
	// 使用数据目录路径
	path := filepath.Join("data", "chameleon.pem")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("证书文件不存在", "path", path)
		return true // 如果文件不存在，认为已经卸载
	}

	// 使用 GUI 密码输入来卸载证书
	err := executeCommandWithGUIPassword("security", "remove-trusted-cert", "-d", path)
	if err != nil {
		slog.Error("证书卸载失败", "err", err.Error())
		return false
	}

	slog.Info("证书卸载成功", "path", path)
	return true
}
