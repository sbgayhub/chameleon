package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sbgayhub/chameleon/backend/certificate"
	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/config"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/statistics"

	"github.com/elazarl/goproxy"
	"github.com/gookit/goutil/errorx"
)

type ProxyServer struct {
	config     *config.ProxyConfig
	server     *http.Server
	statsMgr   *statistics.Manager
	channelMgr *channel.Manager
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	mu         sync.Mutex
}

func NewProxyServer(config *config.ProxyConfig, channelMgr *channel.Manager, statsMgr *statistics.Manager) *ProxyServer {
	slog.Info("创建Http代理服务器")
	ctx, cancel := context.WithCancel(context.Background())
	return &ProxyServer{
		server:     nil,
		config:     config,
		statsMgr:   statsMgr,
		channelMgr: channelMgr,
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}
}

func (s *ProxyServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return errorx.E("代理服务器已在运行")
	}

	ps := goproxy.NewProxyHttpServer()
	ps.CertStore = certificate.Store
	// 中间人代理，处理https请求
	ps.OnRequest().HandleConnect(s.handleConnect())

	// 处理请求
	ps.OnRequest().Do(s.handRequest())

	// 处理响应
	ps.OnResponse().Do(s.handleResponse())

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", "127.0.0.1", s.config.Port),
		Handler: ps,
	}

	// 先设置运行状态
	s.running = true
	slog.Info("Http 代理服务器启动成功", "监听端口", s.config.Port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Http 代理服务器运行出错", "error", err)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}
	}()
	return nil
}

func (s *ProxyServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.server == nil {
		slog.Info("代理服务器未运行，无需停止")
		return nil
	}

	slog.Info("正在停止代理服务器")

	// 先设置运行状态为 false
	s.running = false

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("代理服务器停止失败", "error", err)
		return errorx.With(err, "代理服务器停止失败")
	}

	slog.Info("代理服务器已停止")
	return nil
}

func (s *ProxyServer) handleConnect() goproxy.FuncHttpsHandler {
	var hosts []string
	for _, group := range s.channelMgr.List() {
		hosts = append(hosts, group.Endpoint+":443")
	}
	// 只有在渠道组中的host才进行mitm中间人代理，其他直接放行
	return func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		if slices.Contains(hosts, host) {
			return &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
				return &tls.Config{GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					slog.Debug("[http-proxy] 签发证书", "host", hello.ServerName)
					return certificate.Store.Fetch(hello.ServerName, func() (*tls.Certificate, error) {
						return certificate.SignHost(certificate.CA, []string{hello.ServerName})
					})
				}}, nil
			}}, host
			//return goproxy.MitmConnect, host
		} else {
			return goproxy.OkConnect, host
		}
	}
}

func (s *ProxyServer) handRequest() goproxy.FuncReqHandler {
	return func(request *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {

		slog.Debug("请求进入", "method", request.Method, "host", request.Host)

		// 如果请求在渠道组中，则进行代理处理，否则直接转发
		groups := s.channelMgr.List()
		ex := slices.ContainsFunc(groups, func(group *channel.Group) bool {
			return group.Endpoint == request.Host && group.Enabled && len(group.Channels) != 0
		})
		if !ex {
			return request, nil
		}

		// 获取一个可用的渠道节点
		p, err := s.channelMgr.SelectChannel(request.Host)
		if err != nil {
			slog.Error("获取代理失败", "error", err.Error())
			return request, goproxy.NewResponse(request, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
		}

		ctx.UserData = p
		slog.Info(fmt.Sprintf("[%s] 开始处理请求", p.Name), "method", request.Method, "url", request.URL)

		// 转换请求
		converter, err := convert.Get(p.ConverterName)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] 获取转换器失败", p.Name), "name", p.ConverterName, "error", err)
			return nil, goproxy.NewResponse(request, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
		}

		if request, err := converter.ConvertRequest(request, *p); err != nil {
			slog.Error(fmt.Sprintf("[%s] 转换请求失败", p.Name), "name", p.ConverterName, "error", err)
			return request, goproxy.NewResponse(request, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
		} else {
			slog.Info(fmt.Sprintf("[%s] 处理请求成功", p.Name), "url", request.URL)
			ctx.Req = request
			return request, nil
		}
	}
}

func (s *ProxyServer) handleResponse() goproxy.FuncRespHandler {
	return func(response *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		request := ctx.Req

		// 如果handleRequest阶段获取到了channel，则需要进行处理，否则直接返回响应
		if ctx.UserData == nil {
			return response
		}
		p, ok := ctx.UserData.(*channel.Channel)
		if !ok {
			return response
		}

		slog.Info(fmt.Sprintf("[%s] 开始处理响应", p.Name), "status", response.StatusCode, "url", request.URL)
		if response.StatusCode != http.StatusOK {
			statistics.UpdateStatistics(p.Name, false, 0, 0)
			return response
		}

		converter, err := convert.Get(p.ConverterName)
		if err != nil {
			slog.Error(fmt.Sprintf("[%s] 获取转换器失败", p.Name), "name", p.ConverterName, "error", err)
			return goproxy.NewResponse(request, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
		}

		// 如果是GET请求或响应body为空，直接返回
		if response.Request.Method == http.MethodGet || response.Body == nil {
			return response
		}

		// 检查是否是 SSE 流
		if strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
			response, err = converter.ConvertStream(response, *p)
		} else {
			response, err = converter.ConvertResponse(response, *p)
		}

		if err != nil {
			slog.Error(fmt.Sprintf("[%s] 转换响应失败", p.Name), "name", p.ConverterName, "error", err)
			return goproxy.NewResponse(request, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
		} else {
			slog.Info(fmt.Sprintf("[%s] 处理响应成功", p.Name), "status", response.StatusCode, "url", request.URL)
			return response
		}
	}
}
