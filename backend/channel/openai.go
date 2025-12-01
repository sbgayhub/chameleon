package channel

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gookit/goutil/errorx"
	"github.com/sbgayhub/chameleon/backend/statistics"
	"github.com/tidwall/gjson"
)

func fetchOpenaiModels(node *Channel) error {
	var url string
	if strings.HasSuffix(node.URL, "/") {
		url = node.URL + "models"
	} else {
		url = node.URL + "/v1/models"
	}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+node.ApiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errorx.Ef("请求出现异常，HTTP状态码：%d", response.StatusCode)
	}

	bytes, err := io.ReadAll(response.Body)
	defer func(Body io.ReadCloser) { _ = Body.Close() }(response.Body)
	if err != nil {
		return err
	}

	data := gjson.ParseBytes(bytes)
	for _, model := range data.Get("data.#.id").Array() {
		node.Models = append(node.Models, model.String())
	}
	slog.Info("获取模型列表完成", "channel", node.Name, "count", len(node.Models))
	return nil
}

func testOpenaiChannel(node *Channel) (string, error) {
	if node.TestModel == "" && len(node.Models) == 0 {
		return "", errorx.E("当前渠道下无可用模型，请检查渠道地址或key")
	}
	if node.TestModel == "" {
		node.TestModel = node.Models[0]
	}

	var url string
	if strings.HasSuffix(node.URL, "/") {
		url = node.URL + "chat/completions"
	} else {
		url = node.URL + "/v1/chat/completions"
	}
	var body = fmt.Sprintf(`{"model":"%s","messages":[{"role": "user", "content": "你是谁"}]}`, node.TestModel)
	request, err := http.NewRequest(http.MethodPost, url, io.NopCloser(strings.NewReader(body)))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+node.ApiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", errorx.Ef("请求出现异常，HTTP状态码：%d", response.StatusCode)
	}

	bytes, err := io.ReadAll(response.Body)
	defer func(Body io.ReadCloser) { _ = Body.Close() }(response.Body)
	if err != nil {
		return "", err
	}
	data := gjson.ParseBytes(bytes)
	if err := data.Get("error"); err.Exists() {
		statistics.UpdateStatistics(node.Name, false, 0, 0)
		return "", errorx.E(err.Get("message").String())
	}
	statistics.UpdateStatistics(node.Name, true, data.Get("usage.prompt_tokens").Uint(), data.Get("usage.completion_tokens").Uint())
	return data.Get("choices.0.message.content").String(), nil
}
