package certificate

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"syscall"
	"unsafe"
)

var (
	modcrypt32                           = syscall.NewLazyDLL("crypt32.dll")
	procCertAddEncodedCertificateToStore = modcrypt32.NewProc("CertAddEncodedCertificateToStore")
	procCertCloseStore                   = modcrypt32.NewProc("CertCloseStore")
	procCertDeleteCertificateFromStore   = modcrypt32.NewProc("CertDeleteCertificateFromStore")
	procCertDuplicateCertificateContext  = modcrypt32.NewProc("CertDuplicateCertificateContext")
	procCertEnumCertificatesInStore      = modcrypt32.NewProc("CertEnumCertificatesInStore")
	procCertOpenSystemStoreW             = modcrypt32.NewProc("CertOpenSystemStoreW")
)

type windowsRootStore uintptr
type WindowsCertInstaller struct{}

func init() {
	installer = WindowsCertInstaller{}
	slog.Debug("证书安装器创建成功")
}

func (w WindowsCertInstaller) Install(_ string, cert []byte) bool {
	// Decode PEM
	if certBlock, _ := pem.Decode(cert); certBlock == nil || certBlock.Type != "CERTIFICATE" {
		slog.Error("invalid PEM data")
	} else {
		cert = certBlock.Bytes
	}
	// Open root store
	store, err := openWindowsRootStore()
	if err != nil {
		slog.Error("open root store")
		return false
	}
	defer store.close()
	// Add cert
	if err := store.addCert(cert); err != nil {
		slog.Error("安装证书失败", "err", err.Error())
		return false
	} else {
		slog.Info("证书安装成功")
	}
	return true
}

func (w WindowsCertInstaller) Uninstall(serial *big.Int) bool {
	store, err := openWindowsRootStore()
	if err != nil {
		slog.Error("open root store", "err", err.Error())
		return false
	}
	defer store.close()
	// Do the deletion
	deletedAny, err := store.deleteCertsWithSerial(serial)
	if err == nil && !deletedAny {
		err = fmt.Errorf("no certs found")
	}
	if err != nil {
		slog.Error("删除证书失败", "err", err.Error())
		return false
	}
	return true
}

func openWindowsRootStore() (windowsRootStore, error) {
	rootStr, err := syscall.UTF16PtrFromString("ROOT")
	if err != nil {
		return 0, err
	}
	store, _, err := procCertOpenSystemStoreW.Call(0, uintptr(unsafe.Pointer(rootStr)))
	if store != 0 {
		return windowsRootStore(store), nil
	}
	return 0, fmt.Errorf("failed to open windows root store: %v", err)
}

func (w windowsRootStore) close() error {
	ret, _, err := procCertCloseStore.Call(uintptr(w), 0)
	if ret != 0 {
		return nil
	}
	return fmt.Errorf("failed to close windows root store: %v", err)
}

func (w windowsRootStore) addCert(cert []byte) error {
	ret, _, err := procCertAddEncodedCertificateToStore.Call(
		uintptr(w), // HCERTSTORE hCertStore
		uintptr(syscall.X509_ASN_ENCODING|syscall.PKCS_7_ASN_ENCODING), // DWORD dwCertEncodingType
		uintptr(unsafe.Pointer(&cert[0])),                              // const BYTE *pbCertEncoded
		uintptr(len(cert)),                                             // DWORD cbCertEncoded
		3,                                                              // DWORD dwAddDisposition (CERT_STORE_ADD_REPLACE_EXISTING is 3)
		0,                                                              // PCCERT_CONTEXT *ppCertContext
	)
	if ret != 0 {
		return nil
	}
	return fmt.Errorf("failed adding cert: %v", err)
}

func (w windowsRootStore) deleteCertsWithSerial(serial *big.Int) (bool, error) {
	// Go over each, deleting the ones we find
	var cert *syscall.CertContext
	deletedAny := false
	for {
		// Next enum
		certPtr, _, err := procCertEnumCertificatesInStore.Call(uintptr(w), uintptr(unsafe.Pointer(cert)))
		if cert = (*syscall.CertContext)(unsafe.Pointer(certPtr)); cert == nil {
			var errno syscall.Errno
			if errors.As(err, &errno) && errno == 0x80092004 {
				break
			}
			return deletedAny, fmt.Errorf("failed enumerating certs: %v", err)
		}
		// Parse cert
		certBytes := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:cert.Length]
		parsedCert, err := x509.ParseCertificate(certBytes)
		// We'll just ignore parse failures for now
		if err == nil && parsedCert.SerialNumber != nil && parsedCert.SerialNumber.Cmp(serial) == 0 {
			// Duplicate the context so it doesn't stop the enum when we delete it
			dupCertPtr, _, err := procCertDuplicateCertificateContext.Call(uintptr(unsafe.Pointer(cert)))
			if dupCertPtr == 0 {
				return deletedAny, fmt.Errorf("failed duplicating context: %v", err)
			}
			if ret, _, err := procCertDeleteCertificateFromStore.Call(dupCertPtr); ret == 0 {
				return deletedAny, fmt.Errorf("failed deleting certificate: %v", err)
			}
			deletedAny = true
		}
	}
	return deletedAny, nil
}
