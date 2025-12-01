package anthropic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/strutil"
	"github.com/samber/lo"
	"github.com/sbgayhub/chameleon/backend/channel"
	"github.com/sbgayhub/chameleon/backend/convert"
	"github.com/sbgayhub/chameleon/backend/statistics"
	"github.com/tidwall/gjson"
)

type OpenAIConverter struct {
	reason map[string]string
}

func RegistryOpenAIConverter() {
	converter := OpenAIConverter{
		reason: map[string]string{
			"stop":           "end_turn",
			"length":         "max_tokens",
			"content_filter": "stop_sequence",
			"tool_calls":     "tool_use",
		},
	}
	if err := convert.GetRegistry().Register(&converter); err != nil {
		slog.Error(err.Error())
	}
}

func (o OpenAIConverter) Name() string {
	return convert.ANTHROPIC2OPENAI
}

func (o OpenAIConverter) ConvertRequest(request *http.Request, channel channel.Channel) (result *http.Request, err error) {
	slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理请求", channel.Name, o.Name()))
	// 1、处理url、method、header
	result, err = o.prepareUrl(request, channel)
	if err != nil {
		slog.Warn(fmt.Sprintf("[%s] [%s] URL 处理失败", channel.Name, o.Name()), "err", err.Error())
		return nil, err
	}
	slog.Debug(fmt.Sprintf("[%s] [%s] URL 处理成功", channel.Name, o.Name()))

	// 2、处理body，进行格式转换
	body, _ := io.ReadAll(request.Body)
	if body == nil || len(body) == 0 {
		slog.Debug(fmt.Sprintf("[%s] [%s] 请求体为空，处理完成", channel.Name, o.Name()))
		return
	}
	slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理请求体数据", channel.Name, o.Name()))
	model, body := o.convertRequestBody(body, channel)
	slog.Debug(fmt.Sprintf("[%s] [%s] 请求体数据处理完成", channel.Name, o.Name()))

	// 3、设置原始模型和请求体
	result.Header.Set("original_model", model)
	result.Body = io.NopCloser(bytes.NewReader(body))

	return
}

func (o OpenAIConverter) prepareUrl(request *http.Request, channel channel.Channel) (result *http.Request, err error) {
	var u *url.URL
	if request.URL.Path == "/v1/messages" {
		if strings.HasSuffix(channel.URL, "/") {
			u, err = url.Parse(channel.URL + "chat/completions")
		} else {
			u, err = url.Parse(channel.URL + "/v1/chat/completions")
		}
	} else {
		if strings.HasSuffix(channel.URL, "/") {
			u, err = url.Parse(channel.URL + strings.TrimPrefix(request.URL.Path, "/v1/"))
		} else {
			u, err = url.Parse(channel.URL + request.URL.Path)
		}
	}
	if err != nil {
		return nil, errorx.With(err, "url 解析失败")
	}

	result = &http.Request{}
	result.URL = u
	result.Host = u.Host
	result.Body = request.Body
	result.Method = request.Method
	result.Header = http.Header{}
	result.Header.Set("Authorization", "Bearer "+channel.ApiKey)
	result.Header.Set("Content-Type", "application/json")
	return
}

func (o OpenAIConverter) convertRequestBody(body []byte, channel channel.Channel) (string, []byte) {
	var result = map[string]any{}
	data := gjson.ParseBytes(body)

	// 模型替换
	originalModel := data.Get("model").String()
	result["model"] = channel.ModelMapper.MapModel(originalModel)

	// 处理系统消息和用户消息
	var messages []any
	if res := data.Get("system"); res.Exists() {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": res.String(),
		})
	}
	if res := data.Get("messages"); res.Exists() {
		for _, msg := range res.Array() {
			// 忽略user/assistant之外的消息
			if !slices.Contains([]string{"user", "assistant"}, msg.Get("role").String()) {
				continue
			}

			// 处理tool_result
			if msg.Get("role").String() == "user" && msg.Get("content").IsArray() {
				for _, item := range msg.Get("content").Array() {
					if item.Get("type").String() != "tool_result" {
						continue
					}
					var id string
					if r := item.Get("id"); r.Exists() {
						id = r.String()
					} else if r := item.Get("tool_use_id"); r.Exists() {
						id = r.String()
					}
					messages = append(messages, map[string]any{
						"role":         "tool",
						"tool_call_id": id,
						"content":      item.Get("content").String(),
					})
				}
			}

			// 处理tool_use
			if msg.Get("role").String() == "assistant" && msg.Get("content").IsArray() {
				first := msg.Get("content").Array()[0]
				if first.Get("type").String() != "tool_use" {
					continue
				}
				var id string
				if r := first.Get("id"); r.Exists() {
					id = r.String()
				} else if r := first.Get("tool_use_id"); r.Exists() {
					id = r.String()
				}
				messages = append(messages, map[string]any{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []map[string]any{{
						"id":   id,
						"type": "function",
						"function": map[string]any{
							"name":      first.Get("name").String(),
							"arguments": first.Get("input").String(),
						},
					}},
				})
			}

			// 普通文本消息
			if content := msg.Get("content"); content.Exists() {
				if content.IsArray() {
					var temp []map[string]any
					for _, item := range content.Array() {
						if item.Get("type").String() == "text" {
							temp = append(temp, map[string]any{
								"type": "text",
								"text": item.Get("text").String(),
							})
						} else if item.Get("type").String() == "image" {
							source := item.Get("source")
							if source.Get("type").String() == "base64" {
								temp = append(temp, map[string]any{
									"type": "image_url",
									"image_url": map[string]any{
										"url": fmt.Sprintf("data:%s;base64,%s",
											strutil.BlankOr(source.Get("media_type").String(), "image/jpeg"),
											source.Get("data").String(),
										),
									},
								})
							}
						}
					}

					if len(temp) > 1 {
						messages = append(messages, map[string]any{
							"role":    msg.Get("role").String(),
							"content": temp,
						})
					} else {
						messages = append(messages, map[string]any{
							"role":    msg.Get("role").String(),
							"content": temp[0]["text"],
						})
					}
				} else {
					messages = append(messages, map[string]any{
						"role":    msg.Get("role").String(),
						"content": content.String(),
					})
				}
			}
		}
	}
	result["messages"] = messages

	// 工具调用处理
	if res := data.Get("tools"); res.Exists() {
		result["tool_choice"] = "auto"
		result["tools"] = lo.Map(res.Array(), func(item gjson.Result, index int) any {
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        item.Get("name").String(),
					"description": item.Get("description").String(),
					"parameters":  item.Get("input_schema").Map(),
				},
			}
		})
	}

	// 思考预算处理
	if res := data.Get("thinking"); res.Exists() && res.Get("type").String() == "enable" {
		// 根据budget_tokens设置reasoning_effort等级
		budgetToken := res.Get("budget_tokens")
		if !budgetToken.Exists() {
			result["reasoning_effort"] = "high" // 如果未提供，默认未high
		}
		temp := budgetToken.Uint()
		if temp <= 2048 {
			result["reasoning_effort"] = "low"
		} else if temp <= 16384 {
			result["reasoning_effort"] = "medium"
		} else {
			result["reasoning_effort"] = "high"
		}

		// 处理max_completion_tokens
		if maxToken := data.Get("max_token"); maxToken.Exists() {
			// 移除max_tokens，使用max_completion_tokens
			delete(result, "max_token")
			result["max_completion_tokens"] = maxToken.Uint()
		}
	}

	// 其他参数处理
	if res := data.Get("max_token"); res.Exists() {
		result["max_token"] = res.Uint()
	}
	if res := data.Get("temperature"); res.Exists() {
		result["temperature"] = res.Float()
	}
	if res := data.Get("top_p"); res.Exists() {
		result["top_p"] = res.Float()
	}
	if res := data.Get("stop_sequences"); res.Exists() {
		result["stop_sequences"] = res.String()
	}
	if res := data.Get("stream"); res.Exists() {
		result["stream"] = res.Bool()
	}

	// 序列化数据
	bys, _ := json.Marshal(result)
	return originalModel, bys
}

func (o OpenAIConverter) ConvertResponse(response *http.Response, channel channel.Channel) (*http.Response, error) {
	slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理响应", channel.Name, o.Name()))
	// 从请求中获取原始模型
	var model = response.Request.Header.Get("original_model")

	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Debug(fmt.Sprintf("[%s] [%s] 读取响应体失败", channel.Name, o.Name()))
		return nil, err
	}

	if response.Request.URL.Path == "/v1/models" {
		// 处理模型列表转换、模型替换
		slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理模型列表转换", channel.Name, o.Name()))
		body = o.convertModels(body, model)
		slog.Debug(fmt.Sprintf("[%s] [%s] 模型列表转换处理完成", channel.Name, o.Name()))
	} else {
		// 处理对话消息转换
		slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理对话消息转换", channel.Name, o.Name()))
		body = o.convertMessages(body, model, channel.Name)
		slog.Debug(fmt.Sprintf("[%s] [%s] 对话消息转换处理完成", channel.Name, o.Name()))
	}

	// 设置请求体
	response.Header.Set("Content-Type", "application/json")
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

func (o OpenAIConverter) convertModels(body []byte, model string) []byte {
	return body
}

func (o OpenAIConverter) convertMessages(body []byte, model, name string) []byte {
	var data = gjson.ParseBytes(body)
	var usage = convert.TokenUsage{}
	var result = map[string]any{
		"id":          strutil.BlankOr(data.Get("id").String(), "msg_openai"),
		"type":        "message",
		"role":        "assistant",
		"content":     []map[string]any{},
		"model":       model,
		"stop_reason": "end_turn",
		"usage":       map[string]any{},
	}

	// 处理choices对话
	if res := data.Get("choices"); res.Exists() && res.IsArray() && res.Array()[0].Exists() {
		var messages []any
		var choice = res.Array()[0]
		var message = choice.Get("message")

		// 处理tool_calls
		if tools := message.Get("tool_calls"); tools.Exists() && tools.IsArray() {
			for _, tool := range tools.Array() {
				if fn := tool.Get("function"); fn.Exists() {
					argStr := strutil.StringOr(fn.Get("arguments").String(), "{}")
					//var arg = map[string]any{}
					arg, _ := convertor.StructToMap(argStr)
					messages = append(messages, map[string]any{
						"type":  "tool_use",
						"id":    tool.Get("id").String(),
						"name":  fn.Get("name").String(),
						"input": arg,
					})
				}
			}
		}

		// 处理思考内容
		if reason := message.Get("reasoning_content"); reason.Exists() {
			messages = append(messages, map[string]any{
				"type":     "thinking",
				"thinking": reason.String(),
			})
		}

		// 处理普通文本内容
		if content := message.Get("content"); content.Exists() {
			reg := regexp.MustCompile(`(?s)<thinking>\s*(.*?)\s*</thinking>`)

			// 查找所有匹配项，每个match包含[完整匹配, 捕获组]
			allMatches := reg.FindAllStringSubmatch(content.String(), -1)
			// 获取所有匹配的位置索引，每个index为[起始位置, 结束位置]
			allIndexes := reg.FindAllStringIndex(content.String(), -1)
			// 如果没有找到匹配项，返回原文本
			if len(allMatches) == 0 {
				messages = append(messages, map[string]any{
					"type": "text",
					"text": content.String(),
				})
			} else {
				// 不确定需不需要使用一个数组将thinking内容包裹起来
				var last int
				for i, match := range allMatches {
					// match[0]是完整匹配（含标签），match[1]是捕获组（thinking内容）
					thinkingContent := strings.TrimSpace(match[1])

					// 当前匹配的起止位置
					matchStart := allIndexes[i][0]
					matchEnd := allIndexes[i][1]

					// 添加thinking标签之前的文本
					beforeText := strings.TrimSpace(content.String()[last:matchStart])
					if beforeText != "" {
						messages = append(messages, map[string]any{
							"type": "text",
							"text": beforeText,
						})
					}

					// 添加thinking内容块
					if thinkingContent != "" {
						messages = append(messages, map[string]any{
							"type":     "thinking",
							"thinking": thinkingContent,
						})
					}

					last = matchEnd
				}
				// 添加最后一个thinking标签之后的文本
				afterText := strings.TrimSpace(content.String()[last:])
				if afterText != "" {
					messages = append(messages, map[string]any{
						"type": "text",
						"text": afterText,
					})
				}
			}
		}

		result["content"] = messages
		result["stop_reason"] = o.reason[strutil.BlankOr(choice.Get("finish_reason").String(), "stop")]
	}

	// 处理token使用数据
	if res := data.Get("usage"); res.Exists() {
		usage.InputTokens = res.Get("prompt_tokens").Uint()
		usage.OutputTokens = res.Get("completion_tokens").Uint()
		result["usage"] = map[string]uint64{
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
		}
	}

	// 处理error
	if res := data.Get("error"); res.Exists() {
		_ = json.Unmarshal(body, &result)
	}

	// 统计
	statistics.UpdateStatistics(name, true, usage.InputTokens, usage.OutputTokens)

	// 序列化数据
	bys, _ := json.Marshal(result)
	return bys
}

func (o OpenAIConverter) ConvertStream(response *http.Response, channel channel.Channel) (*http.Response, error) {
	var body = response.Body
	var reader, writer = io.Pipe()
	var model = response.Request.Header.Get("original_model")

	response.Body = reader
	response.ContentLength = -1
	response.Header.Del("Content-Length")
	response.Header.Set("Transfer-Encoding", "chunked")

	go func() {
		defer func(body io.ReadCloser) { _ = body.Close() }(body)
		defer func(writer *io.PipeWriter) { _ = writer.Close() }(writer)

		var count = 0
		var contentStart = false
		var thinkingStart = false
		var blockIndex = 0
		var contentIndex = 0
		var toolCallIndexes = map[int64]int{}
		var toolCallsInfo = map[int64]map[string]any{}
		var id = fmt.Sprintf("msg_%d", time.Now().Unix())
		var endMarker = "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
		var scanner = bufio.NewScanner(body)

		slog.Debug(fmt.Sprintf("[%s] [%s] 开始处理流式响应", channel.Name, o.Name()))
		for scanner.Scan() {
			line := scanner.Text()
			// 跳过空行和非数据行
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			slog.Debug(fmt.Sprintf("[%s] [%s] 开始转换 stream 数据", channel.Name, o.Name()), "count", count)

			count += 1
			line = strings.TrimPrefix(line, "data: ")

			// 处理结束数据
			if slices.Contains([]string{"[DONE]", ""}, strings.TrimSpace(line)) {
				if _, err := writer.Write([]byte(endMarker)); err != nil {
					return
				}
			}

			var events []string
			var data = gjson.Parse(line)
			var choices = data.Get("choices")
			if !choices.Exists() || !choices.IsArray() || len(choices.Array()) == 0 {
				continue
			}
			var choice = choices.Array()[0]
			var delta = choice.Get("delta")

			// 1、第一次收到chunk，发送message_start
			if count == 1 {
				var msg = map[string]any{
					"type": "message_start",
					"message": map[string]any{
						"id":            id,
						"type":          "message",
						"role":          "assistant",
						"content":       []any{},
						"model":         model,
						"stop_reason":   nil,
						"stop_sequence": nil,
						"usage":         map[string]any{"input_tokens": 0, "output_tokens": 0},
					},
				}
				events = append(events, "event: message_start\ndata: "+jsonutil.MustString(msg)+"\n\n")
			}

			// 2、处理思考内容
			if res := delta.Get("reasoning_content"); res.Exists() {
				if !thinkingStart {
					thinkingStart = true
					blockIndex += 1
					block := map[string]any{
						"type":          "content_block_start",
						"index":         0,
						"content_block": map[string]any{"type": "thinking", "thinking": ""},
					}
					events = append(events, fmt.Sprintf("event: content_block_start\ndata: %s\n\n", jsonutil.MustString(block)))
				}
				// 处理思考增量
				block := map[string]any{
					"type":  "content_block_delta",
					"index": 0,
					"delta": map[string]any{"type": "thinking_delta", "thinking": res.String()},
				}
				events = append(events, "event: content_block_delta\ndata: "+jsonutil.MustString(block)+"\n\n")
			}

			// 3、处理文本内容
			if res := delta.Get("content"); res.Exists() {
				if !contentStart {
					if thinkingStart { // 如果有思考内容，在正文开始之前还需要发送一个content_block_stop
						block := map[string]any{"type": "content_block_stop", "index": 0}
						events = append(events, "event: content_block_stop\ndata: "+jsonutil.MustString(block)+"\n\n")
					}
					// 发送正文content_block_start
					contentStart = true
					contentIndex = blockIndex
					blockIndex += 1
					block := map[string]any{
						"type":          "content_block_start",
						"index":         contentIndex,
						"content_block": map[string]any{"type": "text", "text": ""},
					}
					events = append(events, "event: content_block_start\ndata: "+jsonutil.MustString(block)+"\n\n")
				}

				// 处理正文增量
				block := map[string]any{
					"type":  "content_block_delta",
					"index": contentIndex,
					"delta": map[string]any{"type": "text_delta", "text": res.String()},
				}
				events = append(events, "event: content_block_delta\ndata: "+jsonutil.MustString(block)+"\n\n")
			}

			// 4、处理工具调用
			if res := delta.Get("tool_calls"); res.Exists() {
				var processed []int64
				for _, tool := range res.Array() {
					index := tool.Get("index").Int()
					if slices.Contains(processed, index) {
						continue
					}
					processed = append(processed, index)

					if _, ex := toolCallIndexes[index]; !ex {
						// 为新的tool_calls分配content_block索引
						toolCallIndex := blockIndex
						toolCallIndexes[index] = toolCallIndex
						// 生成id和名称
						id := strutil.BlankOr(tool.Get("id").String(), fmt.Sprintf("call_%d_%d", time.Now().Unix(), index))
						name := strutil.BlankOr(tool.Get("function.name").String(), fmt.Sprintf("tool_%d", index))
						//开始新的tool_use content block
						block := map[string]any{
							"type":  "content_block_start",
							"index": toolCallIndex,
							"content_block": map[string]any{
								"type":  "tool_use",
								"id":    id,
								"name":  name,
								"input": map[string]any{},
							},
						}
						events = append(events, "event: content_block_start\ndata: "+jsonutil.MustString(block)+"\n\n")

						// 存储toll_calls信息
						toolCallsInfo[index] = map[string]any{
							"id":                  id,
							"name":                name,
							"arguments":           "",
							"content_block_index": toolCallIndex,
						}
						blockIndex += 1
					}

					// 处理函数累计参数
					if arg := tool.Get("function.arguments"); arg.Exists() {
						if current, ex := toolCallsInfo[index]; ex {
							current["arguments"] = current["arguments"].(string) + arg.String()
							// 清理json片段
							if s, err := cleanJsonFragment(arg.String()); err != nil {
								block := map[string]any{
									"type":  "content_block_delta",
									"index": current["content_block_index"],
									"delta": map[string]any{"type": "input_json_delta", "partial_json": s},
								}
								events = append(events, "event: content_block_delta\ndata: "+jsonutil.MustString(block)+"\n\n")
							}
						}
					}
				}
			}

			// 5、处理流结束
			if res := choice.Get("finish_reason"); res.Exists() {
				// 先停止所有的toll_calls block
				for _, info := range toolCallsInfo {
					block := map[string]any{"type": "content_block_stop", "index": info["content_block_index"]}
					events = append(events, "event: content_block_stop\ndata: "+jsonutil.MustString(block)+"\n\n")
				}
				// 停止文本块（如果有）
				if contentStart {
					block := map[string]any{"type": "content_block_stop", "index": contentIndex}
					events = append(events, "event: content_block_stop\ndata: "+jsonutil.MustString(block)+"\n\n")
				}
				// 映射finish_reason
				block := map[string]any{
					"type": "message_delta",
					"delta": map[string]any{
						"stop_reason":   o.reason[res.String()],
						"stop_sequence": nil,
					},
				}
				// 处理使用数据
				block["usage"] = map[string]any{
					"input_tokens":  data.Get("usage.prompt_tokens").Uint(),
					"output_tokens": data.Get("usage.completion_tokens").Uint(),
				}
				events = append(events, "event: message_delta\ndata: "+jsonutil.MustString(block)+"\n\n")
				events = append(events, `event: message_stop\ndata: {"type": "message_stop"}\n\n`)

			}

			// 发送数据
			for _, event := range events {
				if _, err := writer.Write([]byte(event)); err != nil {
					slog.Warn(fmt.Sprintf("[%s] 数据写入失败", o.Name()))
					continue
				}
			}
		}
		slog.Debug(fmt.Sprintf("[%s] [%s] 流式响应处理完成", channel.Name, o.Name()))
	}()

	return response, nil
}

// 清理json片段，避免不完整的Unicode字符或转义字符
func cleanJsonFragment(fragment string) (string, error) {
	// 处理可能被截断的转义序列
	if strutil.HasOneSuffix(fragment, []string{"\\", "\\\\"}) {
		fragment = fragment[:len(fragment)-1]
	} else if strutil.HasOneSuffix(fragment, []string{"\\u", "\\u0", "\\u00"}) {
		index := strings.Index(fragment, "\\u")
		fragment = fragment[:index]
	}
	return fragment, nil
}
