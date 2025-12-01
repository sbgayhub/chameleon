package anthropic

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/convert"

	"github.com/gookit/goutil/strutil"
)

type NilConverter struct{}

func RegistryAnthropicConverter() {
	if err := convert.GetRegistry().Register(&NilConverter{}); err != nil {
		slog.Warn(err.Error())
	}
}

func (n *NilConverter) Name() string {
	return convert.ANTHROPIC2ANTHROPIC
}

func (n *NilConverter) ConvertRequest(request *http.Request, channel channel.Channel) (result *http.Request, err error) {
	result = &http.Request{}
	// 1、处理url、path、header
	var u *url.URL
	if request.URL.Path == "/v1/messages" {
		if strings.HasSuffix(channel.URL, "/") {
			u, err = url.Parse(channel.URL + "messages")
		} else {
			u, err = url.Parse(channel.URL + "/v1/messages")
		}
		if err != nil {
			slog.Warn("url 解析失败", "channel", channel.Name, "err", err.Error())
			return nil, err
		}
	} else {
		if strings.HasSuffix(channel.URL, "/") {
			u, err = url.Parse(channel.URL + strings.TrimPrefix(request.URL.Path, "/v1/"))
		} else {
			u, err = url.Parse(channel.URL + request.URL.Path)
		}
		if err != nil {
			slog.Warn("url 解析失败", "channel", channel.Name, "err", err.Error())
			return nil, err
		}
	}

	result.URL = u
	result.Host = u.Host
	result.Method = request.Method
	result.Header = http.Header{}

	result.Header.Set("x-api-key", channel.ApiKey)
	result.Header.Set("Authorization", "Bearer "+channel.ApiKey)
	result.Header.Set("anthropic-version", "2023-06-01")
	result.Header.Set("Content-Type", "application/json")

	if request.Method == http.MethodGet {
		result.Body = request.Body
		return
	}

	// 2、处理body，进行格式转换
	var requestData = make(map[string]any)
	//var resultData = make(map[string]any)
	// 将请求body读取到data中
	all, _ := io.ReadAll(request.Body)
	defer func(Body io.ReadCloser) { _ = Body.Close() }(request.Body)
	_ = json.Unmarshal(all, &requestData)

	// 模型替换
	result.Header.Set("original_model", strutil.StringOr(requestData["model"], ""))
	requestData["model"] = channel.ModelMapper.MapModel(requestData["model"].(string))

	// 序列化数据
	if body, err := json.Marshal(requestData); err != nil {
		return nil, err
	} else {
		result.Body = io.NopCloser(bytes.NewReader(body))
		return result, nil
	}
}

func (n *NilConverter) ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) {
	return response, nil
}

func (n *NilConverter) ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error) {
	return response, nil
}
