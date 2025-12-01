package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/statistics"

	"github.com/gookit/goutil/errorx"
)

type NilConverter struct{}

func RegistryOpenAIConverter() {
	if err := convert.GetRegistry().Register(&NilConverter{}); err != nil {
		slog.Error(err.Error())
		return
	}
}

func (n NilConverter) Name() string {
	return convert.OPENAI2OPENAI
}

func (n NilConverter) ConvertRequest(request *http.Request, channel channel.Channel) (*http.Request, error) {
	address, _ := url.Parse(channel.URL + request.URL.Path)
	request.URL = address
	request.Host = address.Host
	request.Header.Set("Authorization", "Bearer "+channel.ApiKey)
	return request, nil
}

func (n NilConverter) ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) {

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errorx.With(err, "解析 OpenAI 响应失败")
	}
	var openaiResponse convert.OpenAIResponse
	if err := json.Unmarshal(body, &openaiResponse); err != nil {
		return nil, errorx.With(err, "解析 OpenAI 响应失败")
	}

	// 提取token使用信息
	usage := convert.TokenUsage{
		InputTokens:  uint64(openaiResponse.Usage.PromptTokens),
		OutputTokens: uint64(openaiResponse.Usage.CompletionTokens),
	}

	statistics.UpdateStatistics(channel.Name, true, usage.InputTokens, usage.OutputTokens)

	return response, nil
}

func (n NilConverter) ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error) {
	// 读取原始响应体
	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("读取响应失败", "err", err)
		return response, errorx.With(err, "读取响应失败")
	}
	_ = response.Body.Close()

	// 解析并记录每个 SSE 事件
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var event strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// 空行表示一个事件结束
		if line == "" && event.Len() > 0 {
			var body convert.OpenAIStreamResponse
			if err := json.Unmarshal([]byte(event.String()), &body); err != nil {
				return nil, errorx.With(err, "json解析失败")
			}

			event.Reset()
		} else {
			event.WriteString(line)
			event.WriteString("\n")
		}
	}

	// 恢复响应体
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}
