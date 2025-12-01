package certificate

import (
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"os/user"
	"sync"
)

var sudoWarningOnce sync.Once
var installer CertInstaller = DefaultInstaller{}

type CertInstaller interface {
	Install(dir string, cert []byte) bool
	Uninstall(serial *big.Int) bool
}

type DefaultInstaller struct{}

func (d DefaultInstaller) Install(string, []byte) bool {
	//TODO implement me
	panic("implement me")
}

func (d DefaultInstaller) Uninstall(*big.Int) bool {
	//TODO implement me
	panic("implement me")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func commandWithSudo(cmd ...string) *exec.Cmd {
	if u, err := user.Current(); err == nil && u.Uid == "0" {
		return exec.Command(cmd[0], cmd[1:]...)
	}
	if !binaryExists("sudo") {
		sudoWarningOnce.Do(func() {
			slog.Warn(`Warning: "sudo" is not available, and mkcert is not running as root. The (un)install operation might fail. ⚠️`)
		})
		return exec.Command(cmd[0], cmd[1:]...)
	}
	return exec.Command("sudo", append([]string{"--prompt=Sudo password:", "--"}, cmd...)...)
}
