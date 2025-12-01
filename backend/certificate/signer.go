/**
dynamic sign tls cert, copy from 'github.com/qlazarl/goproxy'
*/

package certificate

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"net"
	"runtime"
	"sort"
	"strings"
	"time"
)

const signerVersion = ":chameleon"

type CounterEncryptorRand struct {
	cipher  cipher.Block
	counter []byte
	rand    []byte
	ix      int
}

func SignHost(ca tls.Certificate, hosts []string) (cert *tls.Certificate, err error) {
	// 使用提供的 CA 生成证书，使用已解析的叶证书（如果存在）。
	x509ca := ca.Leaf
	if x509ca == nil {
		if x509ca, err = x509.ParseCertificate(ca.Certificate[0]); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	start := now.Add(-30 * 24 * time.Hour) // -30 days
	end := now.Add(365 * 24 * time.Hour)   // 365 days

	// 始终生成正 int 值（当第一位为 0 时，不启用二补码）
	generated := rand.Uint64()
	generated >>= 1

	template := x509.Certificate{
		SerialNumber: big.NewInt(int64(generated)),
		Issuer:       x509ca.Subject,
		Subject: pkix.Name{
			Organization:       []string{"github.com/sbgayhub/chameleon untrusted MITM proxy Inc"},
			OrganizationalUnit: []string{"github.com/sbgayhub/chameleon Proxy"},
		},
		NotBefore: start,
		NotAfter:  end,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
			template.Subject.CommonName = h
		}
	}

	hash := hashSorted(append(hosts, signerVersion, ":"+runtime.Version()))
	var csprng CounterEncryptorRand
	if csprng, err = NewCounterEncryptorRandFromKey(ca.PrivateKey, hash); err != nil {
		return nil, err
	}

	var certpriv crypto.Signer
	switch ca.PrivateKey.(type) {
	case *rsa.PrivateKey:
		if certpriv, err = rsa.GenerateKey(&csprng, 2048); err != nil {
			return nil, err
		}
	case *ecdsa.PrivateKey:
		if certpriv, err = ecdsa.GenerateKey(elliptic.P256(), &csprng); err != nil {
			return nil, err
		}
	case ed25519.PrivateKey:
		if _, certpriv, err = ed25519.GenerateKey(&csprng); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported key type %T", ca.PrivateKey)
	}

	derBytes, err := x509.CreateCertificate(&csprng, &template, x509ca, certpriv.Public(), ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	// 保存已解析的叶证书以使用更少的 CPU
	leafCert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, err
	}

	certBytes := [][]byte{derBytes}
	certBytes = append(certBytes, ca.Certificate...)
	return &tls.Certificate{
		Certificate: certBytes,
		PrivateKey:  certpriv,
		Leaf:        leafCert,
	}, nil
}

func hashSorted(lst []string) []byte {
	c := make([]string, len(lst))
	copy(c, lst)
	sort.Strings(c)
	h := sha256.New()
	h.Write([]byte(strings.Join(c, ",")))
	return h.Sum(nil)
}

func NewCounterEncryptorRandFromKey(key any, seed []byte) (r CounterEncryptorRand, err error) {
	var keyBytes []byte
	switch key := key.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
	case *ecdsa.PrivateKey:
		if keyBytes, err = x509.MarshalECPrivateKey(key); err != nil {
			return
		}
	case ed25519.PrivateKey:
		if keyBytes, err = x509.MarshalPKCS8PrivateKey(key); err != nil {
			return
		}
	default:
		return r, errors.New("only RSA, ED25519 and ECDSA keys supported")
	}
	h := sha256.New()
	if r.cipher, err = aes.NewCipher(h.Sum(keyBytes)[:aes.BlockSize]); err != nil {
		return r, err
	}
	r.counter = make([]byte, r.cipher.BlockSize())
	if seed != nil {
		copy(r.counter, h.Sum(seed)[:r.cipher.BlockSize()])
	}
	r.rand = make([]byte, r.cipher.BlockSize())
	r.ix = len(r.rand)
	return r, nil
}

func (c *CounterEncryptorRand) Seed(b []byte) {
	if len(b) != len(c.counter) {
		panic("SetCounter: wrong counter size")
	}
	copy(c.counter, b)
}

func (c *CounterEncryptorRand) refill() {
	c.cipher.Encrypt(c.rand, c.counter)
	for i := 0; i < len(c.counter); i++ {
		if c.counter[i]++; c.counter[i] != 0 {
			break
		}
	}
	c.ix = 0
}

func (c *CounterEncryptorRand) Read(b []byte) (n int, err error) {
	if c.ix == len(c.rand) {
		c.refill()
	}
	if n = len(c.rand) - c.ix; n > len(b) {
		n = len(b)
	}
	copy(b, c.rand[c.ix:c.ix+n])
	c.ix += n
	return
}
