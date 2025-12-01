package certificate

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
)

var (
	SystemTrustFilename string
	SystemTrustCommand  []string
	CertutilInstallHelp string
)

type LinuxCertInstaller struct{}

func init() {
	installer = LinuxCertInstaller{}
	if pathExists("/etc/pki/ca-trust/source/anchors/") {
		SystemTrustFilename = "/etc/pki/ca-trust/source/anchors/%s.pem"
		SystemTrustCommand = []string{"update-ca-trust", "extract"}
	} else if pathExists("/usr/local/share/ca-certificates/") {
		SystemTrustFilename = "/usr/local/share/ca-certificates/%s.crt"
		SystemTrustCommand = []string{"update-ca-certificates"}
	} else if pathExists("/etc/ca-certificates/trust-source/anchors/") {
		SystemTrustFilename = "/etc/ca-certificates/trust-source/anchors/%s.crt"
		SystemTrustCommand = []string{"trust", "extract-compat"}
	} else if pathExists("/usr/share/pki/trust/anchors") {
		SystemTrustFilename = "/usr/share/pki/trust/anchors/%s.pem"
		SystemTrustCommand = []string{"update-ca-certificates"}
	}
	slog.Debug("证书安装器创建成功")
}

func (l LinuxCertInstaller) Install(_ string, cert []byte) bool {
	if SystemTrustCommand == nil {
		slog.Error("Installing to the system store is not yet supported on this Linux 😣.")
		return false
	}

	cmd := commandWithSudo("tee", fmt.Sprintf(SystemTrustFilename, "chameleon_proxy"))
	cmd.Stdin = bytes.NewReader(cert)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to execute", "cmd", "tee", "err", err, "out", out)
		return false
	}

	cmd = commandWithSudo(SystemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to execute", "cmd", strings.Join(SystemTrustCommand, " "), "err", err.Error(), "out", out)
		return false
	}
	return true
}

func (l LinuxCertInstaller) Uninstall(serial *big.Int) bool {
	if SystemTrustCommand == nil {
		return false
	}

	cmd := commandWithSudo("rm", "-f", fmt.Sprintf(SystemTrustFilename, "chameleon_proxy"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to execute", "cmd", "rm", "err", err, "out", out)
		return false
	}

	// We used to install under non-unique filenames.
	legacyFilename := fmt.Sprintf(SystemTrustFilename, "mkcert-rootCA")
	if pathExists(legacyFilename) {
		cmd := commandWithSudo("rm", "-f", legacyFilename)
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("failed to execute", "cmd", "rm (legacy filename)", "err", err, "out", out)
			return false
		}
	}

	cmd = commandWithSudo(SystemTrustCommand...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to execute", "cmd", strings.Join(SystemTrustCommand, " "), "err", err, "out", out)
		return false
	}

	return true
}
