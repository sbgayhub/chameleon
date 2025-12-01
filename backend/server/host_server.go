package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/sbgayhub/chameleon/backend/certificate"
	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/config"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/host"
	"github.com/sbgayhub/chameleon/backend/statistics"

	"github.com/gookit/goutil/errorx"
)

// HostServer Host代理服务器
type HostServer struct {
	server     *http.Server
	client     *http.Client
	config     *config.ProxyConfig
	hostMgr    *host.Manager
	channelMgr *channel.Manager
	statsMgr   *statistics.Manager
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
}

// NewHostServer 创建Host服务器
func NewHostServer(config *config.ProxyConfig, hostMgr *host.Manager, channelMgr *channel.Manager, statsMgr *statistics.Manager) *HostServer {
	slog.Info("创建Host代理服务器")
	ctx, cancel := context.WithCancel(context.Background())
	return &HostServer{
		server:     nil,
		client:     &http.Client{Timeout: 3 * time.Minute},
		config:     config,
		hostMgr:    hostMgr,
		channelMgr: channelMgr,
		statsMgr:   statsMgr,
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}
}

// Start 启动服务器
func (s *HostServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("代理服务器已在运行")
	}

	// 先创建host
	hosts := lo.Map(s.channelMgr.List(), func(item *channel.Group, _ int) string { return item.Endpoint })
	if err := s.hostMgr.AddHosts(hosts); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.proxyHandler)

	// 创建Host服务器
	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", "127.0.0.1", 443),
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				slog.Debug("[host-proxy] 签发证书", "host", hello.ServerName)
				return certificate.Store.Fetch(hello.ServerName, func() (*tls.Certificate, error) {
					return certificate.SignHost(certificate.CA, []string{hello.ServerName})
				})
			},
		},
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Host 代理服务器启动失败", "error", err)
		}
	}()

	s.running = true
	slog.Info("Host 代理服务器启动成功")
	return nil
}

// Stop 停止服务器
func (s *HostServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("服务器未运行")
	}

	slog.Info("正在停止Host服务器")

	// 取消上下文
	s.cancel()

	// 关闭Host服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			slog.Error("Host服务器停止失败", "error", err)
			return err
		}
	}

	// 等待goroutine结束
	s.wg.Wait()

	// 移除hosts
	if err := s.hostMgr.RemoveHosts(); err != nil {
		return err
	}

	s.running = false
	slog.Info("Host服务器已停止")
	return nil
}

// IsRunning 检查服务器是否在运行
func (s *HostServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig 获取服务器配置
func (s *HostServer) GetConfig() *config.ProxyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// UpdateConfig 更新服务器配置
func (s *HostServer) UpdateConfig(proxyConfig *config.ProxyConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果服务器正在运行且端口发生变化，需要重启
	if s.running && s.config.Port != proxyConfig.Port {
		slog.Warn("端口更改，需要重启服务器", "old", s.config.Port, "new", proxyConfig.Port)
		// 这里可以实现平滑重启逻辑
	}

	s.config = proxyConfig
	slog.Info("服务器配置已更新", "port", proxyConfig.Port, "mode", proxyConfig.Mode)
	return nil
}

// proxyHandler 代理处理函数
func (s *HostServer) proxyHandler(writer http.ResponseWriter, request *http.Request) {
	// 处理请求
	req, p, err := s.handleRequest(request)
	if err != nil {
		slog.Error(err.Error())
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// 发送请求
	response, err := s.client.Do(req)
	if err != nil {
		slog.Error("请求出现错误", "host", req.Host, "err", err.Error())
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// 处理响应
	resp, err := s.handleResponse(p, response)
	if err != nil {
		slog.Error(err.Error())
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			writer.Header().Set(key, value)
		}
	}
	_, _ = io.Copy(writer, resp.Body)
	_ = resp.Body.Close()
}

func (s *HostServer) handleRequest(request *http.Request) (*http.Request, *channel.Channel, error) {
	// 复制request
	newRequest, _ := http.NewRequest(request.Method, "https://"+request.Host+request.URL.Path, request.Body)
	newRequest.Header = request.Header
	newRequest.Host = request.Host
	// 如果请求在渠道组中，则进行代理处理，否则直接转发
	groups := s.channelMgr.List()
	ex := slices.ContainsFunc(groups, func(group *channel.Group) bool {
		return group.Endpoint == request.Host && group.Enabled && len(group.Channels) != 0
	})
	if !ex {
		return newRequest, nil, nil
	}

	// 获取一个可用的渠道节点
	p, err := s.channelMgr.SelectChannel(newRequest.Host)
	if err != nil {
		slog.Error("获取代理失败", "error", err.Error())
		return newRequest, nil, errorx.E("获取代理失败")
	}
	newRequest = newRequest.WithContext(context.WithValue(request.Context(), "proxy", p))

	// 转换请求
	converter, err := convert.Get(p.ConverterName)
	if err != nil {
		slog.Error("获取代理失败", "error", err.Error())
		return nil, nil, err
	}
	if request, err := converter.ConvertRequest(newRequest, *p); err != nil {
		return request, p, err
	} else {
		return request, p, nil
	}
}

func (s *HostServer) handleResponse(p *channel.Channel, response *http.Response) (*http.Response, error) {
	// 如果请求在渠道组中，则进行代理处理，否则直接转发
	//groups := s.channelMgr.List()

	//endpoints := lo.FlatMap[*channel.Group, string](groups, func(group *channel.Group, index int) []string {
	//	t := lo.Map[*channel.Channel, string](maputil.Values(group.Channels), func(item *channel.Channel, index int) string {
	//		u, _ := url.Parse(item.URL)
	//		return u.Host
	//	})
	//	return t
	//})
	//
	//if !slices.Contains(endpoints, response.Request.Host) {
	//	return response, nil
	//}

	slog.Info(fmt.Sprintf("[%s] 开始处理响应", p.Name), "status", response.StatusCode, "url", response.Request.URL)
	if response.StatusCode != http.StatusOK {
		statistics.UpdateStatistics(p.Name, false, 0, 0)
		return response, nil
	}

	converter, err := convert.Get(p.ConverterName)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s] 获取转换器失败", p.Name), "name", p.ConverterName, "error", err)
		return nil, err
	}

	// 检查是否是 SSE 流
	if strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		response, err = converter.ConvertStream(response, *p)
	} else {
		response, err = converter.ConvertResponse(response, *p)
	}

	if err != nil {
		slog.Error(fmt.Sprintf("[%s] 转换响应失败", p.Name), "name", p.ConverterName, "error", err)
		return nil, err
	} else {
		slog.Info(fmt.Sprintf("[%s] 处理响应成功", p.Name), "status", response.StatusCode, "url", response.Request.URL)
		return response, nil
	}
}

// loggingMiddleware 日志中间件
func (s *HostServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建响应写入器包装器来捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 处理请求
		next.ServeHTTP(wrapped, r)

		// 记录日志
		duration := time.Since(start)
		slog.Info("HTTP请求",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"status", wrapped.statusCode,
			"duration", duration,
		)
	})
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
