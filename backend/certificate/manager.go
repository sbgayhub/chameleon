package certificate

type CertManager struct {
	dir       string
	installer CertInstaller
}

type Pair struct {
	CertPath string
	KeyPath  string
	CertPEM  []byte
	KeyPEM   []byte
}

func NewManager(dataDir string) *CertManager {
	return &CertManager{dir: dataDir, installer: installer}
}

func (c *CertManager) Install() bool {
	return c.installer.Install(c.dir, cert)
}

func (c *CertManager) Uninstall() bool {
	return c.installer.Uninstall(CA.Leaf.SerialNumber)
}
