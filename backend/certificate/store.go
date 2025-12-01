package certificate

import (
	"crypto/tls"
	"log/slog"
	"sync"
)

type Storage struct {
	Cache map[string]*tls.Certificate
	mu    sync.Mutex
}

var Store = &Storage{Cache: make(map[string]*tls.Certificate)}

func (s *Storage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cert, ex := s.Cache[hostname]; ex {
		slog.Debug("[cert-store] 证书缓存命中", "host", hostname)
		return cert, nil
	}

	if cert, err := gen(); err != nil {
		slog.Debug("[cert-store] 证书生成失败", "host", hostname)
		return nil, err
	} else {
		slog.Debug("[cert-store] 证书生成成功", "host", hostname)
		s.Cache[hostname] = cert
		return cert, nil
	}
}
