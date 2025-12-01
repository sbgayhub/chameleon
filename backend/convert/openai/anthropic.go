package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/statistics"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
)

type AnthropicConverter struct {
}

var finishReason = map[string]string{
	"end_turn":      "stop",
	"max_tokens":    "length",
	"stop_sequence": "stop",
	"tool_use":      "tool_calls",
}

func RegistryAnthropicConverter() {
	if err := convert.GetRegistry().Register(&AnthropicConverter{}); err != nil {
		slog.Warn(err.Error())
	}
}

func (a *AnthropicConverter) Name() string {
	return convert.OPENAI2ANTHROPIC
}

// ConvertRequest 转换OpenAI请求到Anthropic格式
func (a *AnthropicConverter) ConvertRequest(request *http.Request, channel channel.Channel) (result *http.Request, err error) {
	result = &http.Request{}
	// 1、处理url、path、header
	var u *url.URL
	if strings.HasSuffix(channel.URL, "/") {
		u, err = url.Parse(channel.URL + "messages")
	} else {
		u, err = url.Parse(channel.URL + "/v1/messages")
	}
	if err != nil {
		slog.Warn("url 解析失败", "channel", channel.Name, "err", err.Error())
		return nil, err
	}
	result.URL = u
	result.Host = u.Host
	result.Method = request.Method
	result.Header = http.Header{}

	result.Header.Set("x-api-key", channel.ApiKey)
	result.Header.Set("anthropic-version", "2023-06-01")
	result.Header.Set("Content-Type", "application/json")

	// 2、处理body，进行格式转换
	var requestData = make(map[string]any)
	var resultData = make(map[string]any)
	// 将请求body读取到data中
	all, _ := io.ReadAll(request.Body)
	defer func(Body io.ReadCloser) { _ = Body.Close() }(request.Body)
	_ = json.Unmarshal(all, &requestData)

	// 模型替换
	result.Header.Set("original_model", strutil.StringOr(requestData["model"], ""))
	resultData["model"] = channel.ModelMapper.MapModel(requestData["model"].(string))

	// 处理消息和系统消息
	if messages, ex := requestData["messages"]; ex {
		for _, message := range messages.([]any) {
			message := message.(map[string]any)
			// 提取系统消息
			if message["role"] == "system" || message["role"] == "developer" {
				resultData["system"] = message["content"]
			} else {
				//chatMessages = append(chatMessages, message)
				var chatMessages []map[string]any

				switch message["role"] {
				case "user": // 用户消息，正常转换
					chatMessages = append(chatMessages, map[string]any{
						"role":    "user",
						"content": message["content"], // todo 暂时直接使用content
					})

				case "assistant": // 助手消息
					if toolCalls, ex := message["tool_calls"]; ex { // tool_calls处理
						var content = make([]any, len(toolCalls.([]any)))
						if toolCalls, ok := toolCalls.([]map[string]any); ok {
							for _, tc := range toolCalls {
								if tc != nil && tc["type"] == "function" {
									if f, ex := tc["function"]; ex {
										if f, ok := f.(map[string]any); ok {
											content = append(content, map[string]any{
												"type":  "tool_use",
												"id":    tc["id"],
												"name":  f["name"],
												"input": f["arguments"],
											})
										}
									}
								}
							}
						}
						chatMessages = append(chatMessages, map[string]any{
							"role":    "assistant",
							"content": content,
						})
					} else { // 普通消息
						chatMessages = append(chatMessages, map[string]any{
							"role":    "assistant",
							"content": message["content"], // todo 暂时直接使用content
						})
					}

				case "tool": // 工具消息
					chatMessages = append(chatMessages, map[string]any{
						"role": "user",
						"content": []map[string]any{
							{
								"type":        "tool_result",
								"tool_use_id": message["tool_call_id"],
								"content":     message["content"],
							},
						},
					})

				}
				resultData["messages"] = chatMessages
			}
		}

		//// 将messages从any转为[]map[string]any
		//if messages, ok := messages.([]map[string]any); ok {
		//	var chatMessages = make([]map[string]any, len(messages))
		//	for _, message := range messages {
		//		// 提取系统消息
		//		if message["role"] == "system" || message["role"] == "developer" {
		//			resultData["system"] = message["content"]
		//		} else {
		//			chatMessages = append(chatMessages, message)
		//
		//			switch message["role"] {
		//			case "user": // 用户消息，正常转换
		//				chatMessages = append(chatMessages, map[string]any{
		//					"role":    "user",
		//					"content": message["content"], // todo 暂时直接使用content
		//				})
		//
		//			case "assistant": // 助手消息
		//				if toolCalls, ex := message["tool_calls"]; ex { // tool_calls处理
		//					var content = make([]any, len(toolCalls.([]any)))
		//					if toolCalls, ok := toolCalls.([]map[string]any); ok {
		//						for _, tc := range toolCalls {
		//							if tc != nil && tc["type"] == "function" {
		//								if f, ex := tc["function"]; ex {
		//									if f, ok := f.(map[string]any); ok {
		//										content = append(content, map[string]any{
		//											"type":  "tool_use",
		//											"id":    tc["id"],
		//											"name":  f["name"],
		//											"input": f["arguments"],
		//										})
		//									}
		//								}
		//							}
		//						}
		//					}
		//					chatMessages = append(chatMessages, map[string]any{
		//						"role":    "assistant",
		//						"content": content,
		//					})
		//				} else { // 普通消息
		//					chatMessages = append(chatMessages, map[string]any{
		//						"role":    "assistant",
		//						"content": message["content"], // todo 暂时直接使用content
		//					})
		//				}
		//
		//			case "tool": // 工具消息
		//				chatMessages = append(chatMessages, map[string]any{
		//					"role": "user",
		//					"content": []map[string]any{
		//						{
		//							"type":        "tool_result",
		//							"tool_use_id": message["tool_call_id"],
		//							"content":     message["content"],
		//						},
		//					},
		//				})
		//
		//			}
		//		}
		//	}
		//	resultData["messages"] = chatMessages
		//}
	}

	// Anthropic 要求必须有 max_tokens
	if maxTokens, ex := requestData["max_tokens"]; ex {
		resultData["max_tokens"] = maxTokens
	} else {
		resultData["max_tokens"] = 32000
	}

	if temp, ex := requestData["temperature"]; ex {
		resultData["temperature"] = temp
	}

	if topP, ex := requestData["top_p"]; ex {
		resultData["top_p"] = topP
	}

	if stop, ex := requestData["stop"]; ex {
		if stop, ok := stop.([]any); ok {
			resultData["stop_sequences"] = stop
		} else {
			resultData["stop_sequences"] = []any{stop}
		}
	}

	if stream, ex := requestData["stream"]; ex {
		resultData["stream"] = stream
	}

	if tools, ex := requestData["tools"]; ex {
		if tools, ok := tools.([]map[string]any); ok {
			var tmp = make([]map[string]any, len(tools))
			for _, tool := range tools {
				if tool["type"] == "function" {
					if f, ex := tool["function"]; ex {
						if f, ok := f.(map[string]any); ok {
							tmp = append(tmp, map[string]any{
								"name":         f["name"],
								"description":  f["description"],
								"input_schema": f["parameters"],
							})
						}
					}
				}
			}
			resultData["tools"] = tmp
		}
	}

	// 处理思考预算，通过max_completion_tokens判断是否为思考模式
	if _, ex := requestData["max_completion_tokens"]; ex {
		reasoningEffort := strutil.StringOrDefault(requestData["reasoning_effort"], "medium")
		var thinkBudget = 0
		switch reasoningEffort {
		case "low":
			thinkBudget = 2048
		case "medium":
			thinkBudget = 8192
		case "high":
			thinkBudget = 16384
		}
		resultData["thinking"] = map[string]any{
			"type":          "enabled",
			"budget_tokens": thinkBudget,
		}
	}

	if body, err := json.Marshal(resultData); err != nil {
		return nil, err
	} else {
		result.Body = io.NopCloser(bytes.NewReader(body))
		return result, nil
	}
}

func (a *AnthropicConverter) ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) {
	var tokenUsage convert.TokenUsage
	var model = response.Request.Header.Get("original_model")

	// 解析json数据
	var data = make(map[string]any)
	if b, err := io.ReadAll(response.Body); err != nil {
		return nil, err
	} else {
		_ = json.Unmarshal(b, &data)
	}

	var result = map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%s", strutil.RandomChars(12)),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{}},
		"usage":   map[string]any{},
	}

	var content = ""
	var toolCalls []any
	var thinkingContent = ""
	if temp, ex := data["content"]; ex && reflect.TypeOf(temp).Kind() == reflect.Slice {
		for _, item := range temp.([]any) {
			item := item.(map[string]any)
			if item["type"] == "text" {
				content += strutil.StringOr(item["text"], "")
			} else if item["type"] == "thinking" {
				thinkingContent += strutil.StringOr(item["thinking"], "")
			} else if item["type"] == "tool_use" {
				marshal, _ := json.Marshal(strutil.StringOr(item["input"], "{}"))
				toolCalls = append(toolCalls, map[string]any{
					"id":   strutil.StringOr(item["id"], ""),
					"type": "function",
					"function": map[string]any{
						"name":      strutil.StringOr(item["name"], ""),
						"arguments": marshal,
					},
				})
			}
		}
	}
	// 如果有thinking内容，将其作为前缀添加到content中
	if strutil.IsNotBlank(thinkingContent) {
		content = fmt.Sprintf("<thinking>\n%s\n</thinking>\n\n%s", thinkingContent, content)
	}

	var message = map[string]any{"role": "assistant", "content": content}
	var finish string
	if len(toolCalls) != 0 {
		message["tool_calls"] = toolCalls
		finish = "tool_calls"
	} else {
		finish = finishReason[strutil.StringOr(data["stop_reason"], "")]
	}

	result["choices"] = []map[string]any{{
		"index":         0,
		"message":       message,
		"finish_reason": finish,
	}}

	if usage, ex := data["usage"]; ex && usage != nil {
		usage := usage.(map[string]any)
		tokenUsage.InputTokens = uint64(usage["input_tokens"].(float64))
		tokenUsage.OutputTokens = uint64(usage["output_tokens"].(float64))
		result["usage"] = map[string]any{
			"prompt_tokens":     usage["input_tokens"],
			"completion_tokens": usage["output_tokens"],
			"total_tokens":      usage["input_tokens"].(float64) + usage["output_tokens"].(float64),
		}
	}

	if body, err := json.Marshal(result); err != nil {
		statistics.UpdateStatistics(channel.Name, false, tokenUsage.InputTokens, tokenUsage.OutputTokens)
		return response, err
	} else {
		statistics.UpdateStatistics(channel.Name, true, tokenUsage.InputTokens, tokenUsage.OutputTokens)
		response.Body = io.NopCloser(bytes.NewReader(body))
		return response, nil
	}
}

// ConvertStream 将Anthropic格式的流式响应数据转换回OpenAI格式
func (a *AnthropicConverter) ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error) {
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
		var id = strutil.RandomChars(12)
		var endMarker = "data: [DONE]\n\n"
		var scanner = bufio.NewScanner(body)

		for scanner.Scan() {
			line := scanner.Text()

			// 跳过空行和非数据行
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			slog.Debug("接收到anthropic stream响应数据\n" + line)
			line = strings.TrimPrefix(line, "data: ")
			count += 1

			// 处理结束数据
			if slices.Contains([]string{"[DONE]", ""}, strings.TrimSpace(line)) {
				if _, err := writer.Write([]byte(endMarker)); err != nil {
					return
				}
			}

			// 解析json数据
			var data = make(map[string]any)
			_ = json.Unmarshal([]byte(line), &data)

			var result = map[string]any{
				"id":      fmt.Sprintf("chatcmpl-%s", id),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   model,
				"choices": []map[string]any{{
					"index":         0,
					"delta":         map[string]any{},
					"finish_reason": nil,
				}},
				"usage": map[string]any{},
			}

			// 解析sse事件
			var toolState = map[float64]map[string]string{}
			switch t := data["type"].(string); t {
			case "message_start":
				result["choices"].([]map[string]any)[0]["delta"].(map[string]any)["role"] = "assistant"
			case "content_block_start": // 内容块开始
				if block, ok := data["content_block"].(map[string]any); ok {
					if t, ok := maputil.GetFromAny("type", block); ok && t == "tool_use" {
						index := data["index"].(float64)
						toolState[index] = map[string]string{
							"id":        strutil.StringOr(block["id"], ""),
							"name":      strutil.StringOr(block["name"], ""),
							"arguments": "",
						}
						result["choices"].([]map[string]any)[0]["delta"].(map[string]any)["tool_calls"] = []map[string]any{{
							"id":       strutil.StringOr(block["id"], ""),
							"index":    index,
							"type":     "function",
							"function": map[string]any{"name": strutil.StringOr(block["name"], "")},
						}}
					}
				}
			case "content_block_delta": // 内容增量
				delta := data["delta"].(map[string]any)
				index := data["index"].(float64)

				if delta["type"] == "text_delta" {
					result["choices"].([]map[string]any)[0]["delta"].(map[string]any)["content"] = strutil.StringOr(delta["text"], "")
				} else if delta["type"] == "thinking_delta" {
					result["choices"].([]map[string]any)[0]["delta"].(map[string]any)["reasoning_content"] = strutil.StringOr(delta["thinking"], "")
				} else if delta["type"] == "input_json_delta" {
					if maputil.HasKey(toolState, index) {
						partialJson := strutil.StringOr(delta["partial_json"], "")
						toolState[index]["argument"] += partialJson

						result["choices"].([]map[string]any)[0]["delta"].(map[string]any)["tool_calls"] = []map[string]any{{
							"index":    index,
							"function": map[string]any{"arguments": partialJson},
						}}
					}
				}
			case "content_block_stop": // 内容块结束
			case "message_delta": // 消息结束
				delta := data["delta"].(map[string]any)
				stop := delta["stop_reason"].(string)
				result["choices"].([]map[string]any)[0]["finish_reason"] = finishReason[stop]
				result["usage"] = map[string]any{
					"prompt_token":      data["usage"].(map[string]any)["input_tokens"],
					"completion_tokens": data["usage"].(map[string]any)["output_tokens"],
					"total_tokens":      data["usage"].(map[string]any)["input_tokens"].(float64) + data["usage"].(map[string]any)["output_tokens"].(float64),
				}
			case "message_stop": // 消息流结束
				result["choices"].([]map[string]any)[0]["finish_reason"] = "stop"
			default:
			}

			// 序列化数据
			if b, err := json.Marshal(result); err != nil {
				slog.Warn(fmt.Sprintf("[%s] 数据序列化失败", a.Name()))
				return
			} else {
				if _, err := writer.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
					slog.Warn(fmt.Sprintf("[%s] 数据写入失败", a.Name()))
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
				slog.Warn(fmt.Sprintf("[%s] 数据序列化失败", a.Name()))
				return
			} else {
				if _, err := writer.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
					slog.Warn(fmt.Sprintf("[%s] 数据写入失败", a.Name()))
					return
				}
			}
		}
	}()

	return response, nil
}
