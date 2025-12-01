package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/convert"
)

type GeminiConverter struct {
}

func RegistryGeminiConverter() {
	if err := convert.GetRegistry().Register(&GeminiConverter{}); err != nil {
		slog.Error(err.Error())
		return
	}
}

func (g *GeminiConverter) Name() string {
	return convert.ANTHROPIC2GEMINI
}

func (g *GeminiConverter) ConvertRequest(request *http.Request, channel channel.Channel) (*http.Request, error) {
	// 替换url及path、host

	// 替换header（api-key）

	// 转换body、替换模型

	return request, nil
}

func (g *GeminiConverter) ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) {
	return response, nil
}
func (g *GeminiConverter) ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error) {
	body := response.Body
	model := response.Request.Header.Get("original_model")
	reader, writer := io.Pipe()
	response.Body = reader
	response.ContentLength = -1
	response.Header.Del("Content-Length")
	response.Header.Set("Transfer-Encoding", "chunked")

	go func() {
		defer func(body io.ReadCloser) { _ = body.Close() }(body)
		defer func(writer *io.PipeWriter) { _ = writer.Close() }(writer)

		var count = 0
		//var id = fmt.Sprintf("msg_%d", time.Now().Unix())
		var endMarker = "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		var scanner = bufio.NewScanner(body)

		for scanner.Scan() {
			line := scanner.Text()

			// 跳过空行和非数据行
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			slog.Debug("接收到openai stream响应数据\n" + line)
			count += 1
			line = strings.TrimPrefix(line, "data: ")

			// 处理结束数据

			if slices.Contains([]string{"[DONE]", ""}, strings.TrimSpace(line)) {
				if _, err := writer.Write([]byte(endMarker)); err != nil {
					return
				}
			}

			// 解析json数据
			var data map[string]any
			_ = json.Unmarshal([]byte(line), &data)

			var result = map[string]any{}

			// todo 格式转换

			// 序列化数据
			if b, err := json.Marshal(result); err != nil {
				slog.Warn(fmt.Sprintf("[%s] 数据序列化失败", g.Name()))
				return
			} else {
				if _, err := writer.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
					slog.Warn(fmt.Sprintf("[%s] 数据写入失败", g.Name()))
					return
				}
			}
		}
		slog.Debug("stream 处理完成", "count", count)
		// 如果没有处理任何chunks，发送一个错误响应
		if count == 0 {
			var result = map[string]any{
				"id":      "chatcmpl-error",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   model,
				"choices": []map[string]any{{
					"index":         0,
					"delta":         map[string]any{"content": "Error: No response received from AI service."},
					"finish_reason": "stop",
				}},
			}
			// 序列化数据
			if b, err := json.Marshal(result); err != nil {
				slog.Warn(fmt.Sprintf("[%s] 数据序列化失败", g.Name()))
				return
			} else {
				if _, err := writer.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
					slog.Warn(fmt.Sprintf("[%s] 数据写入失败", g.Name()))
					return
				}
			}
		}
	}()

	return response, nil
}
