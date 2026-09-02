package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ModelsController struct {
	BaseController
}

// 1x1 像素黑色 PNG 的 base64 编码，用于检测模型是否支持 image_url
const testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func testVisionSupport(baseURL, apiKey, model, apiFormat string) bool {
	var payload map[string]interface{}
	if apiFormat == "openai_responses" {
		payload = map[string]interface{}{
			"model": model,
			"input": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "input_text", "text": "describe this image briefly"},
						{"type": "input_image", "image_url": "data:image/png;base64," + testImageBase64},
					},
				},
			},
			"max_output_tokens": 4096,
			"stream":            false,
		}
	} else {
		payload = map[string]interface{}{
			"model": model,
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "text", "text": "describe this image briefly"},
						{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64," + testImageBase64}},
					},
				},
			},
			"max_tokens": 4096,
			"stream":     false,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	endpoint := "/chat/completions"
	if apiFormat == "openai_responses" {
		endpoint = "/responses"
	}
	url := strings.TrimRight(baseURL, "/") + endpoint
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type fetchModelsRequest struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	ConfigID string `json:"config_id"`
}

type testModelRequest struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	APIShape string `json:"api_format"`
	ConfigID string `json:"config_id"`
}

func (c *ModelsController) resolveCredentials(reqBaseURL, reqAPIKey, configID string) (string, string) {
	if configID != "" {
		if config, err := findModelConfig(configID); err == nil {
			if config.APIKey != "" {
				reqAPIKey = config.APIKey
			}
			if config.BaseURL != "" {
				reqBaseURL = config.BaseURL
			}
		}
	}
	return reqBaseURL, reqAPIKey
}

// Fetch POST /api/models/fetch
func (c *ModelsController) Fetch() {
	if !c.CheckPermission("token_plan:read") {
		return
	}
	var req fetchModelsRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	req.BaseURL, req.APIKey = c.resolveCredentials(req.BaseURL, req.APIKey, req.ConfigID)
	if req.BaseURL == "" {
		c.WriteError(http.StatusBadRequest, "base_url 不能为空")
		return
	}
	if isMaskedAPIKey(req.APIKey) {
		c.WriteError(http.StatusBadRequest, "API 密钥已隐式化，请先填写新的真实密钥")
		return
	}
	url := strings.TrimRight(req.BaseURL, "/") + "/models"
	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.WriteError(http.StatusBadGateway, "请求失败："+err.Error())
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			c.WriteError(http.StatusGatewayTimeout, "请求超时，请检查地址是否可达")
			return
		}
		c.WriteError(http.StatusBadGateway, "请求失败："+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.WriteError(http.StatusBadGateway, "服务商返回错误：HTTP "+strconv.Itoa(resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.WriteError(http.StatusBadGateway, "读取响应失败")
		return
	}
	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		c.WriteError(http.StatusBadGateway, "响应解析失败")
		return
	}

	models := make([]string, 0, len(data.Data))
	for _, m := range data.Data {
		if m.ID != "" {
			id := strings.TrimPrefix(m.ID, "models/")
			models = append(models, id)
		}
	}
	sort.Strings(models)
	c.OK(map[string]interface{}{"models": models})
}

// Test POST /api/models/test
func (c *ModelsController) Test() {
	if !c.CheckPermission("token_plan:read") {
		return
	}
	var req testModelRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	req.BaseURL, req.APIKey = c.resolveCredentials(req.BaseURL, req.APIKey, req.ConfigID)
	if req.BaseURL == "" || req.Model == "" {
		c.WriteError(http.StatusBadRequest, "base_url 与 model 不能为空")
		return
	}
	if isMaskedAPIKey(req.APIKey) {
		c.WriteError(http.StatusBadRequest, "API 密钥已隐式化，请先填写新的真实密钥")
		return
	}
	start := time.Now()
	var payload map[string]interface{}
	if req.APIShape == "openai_responses" {
		payload = map[string]interface{}{
			"model":             req.Model,
			"input":             []map[string]string{{"role": "user", "content": "hi"}},
			"max_output_tokens": 4096,
			"stream":            false,
		}
	} else {
		payload = map[string]interface{}{
			"model":      req.Model,
			"messages":   []map[string]string{{"role": "user", "content": "hi"}},
			"max_tokens": 4096,
			"stream":     false,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		c.WriteError(http.StatusBadGateway, "请求构造失败")
		return
	}
	endpoint := "/chat/completions"
	if req.APIShape == "openai_responses" {
		endpoint = "/responses"
	}
	url := strings.TrimRight(req.BaseURL, "/") + endpoint
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		c.WriteError(http.StatusBadGateway, "请求构造失败："+err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			c.WriteError(http.StatusGatewayTimeout, "请求超时，请检查服务地址或网络")
			return
		}
		c.WriteError(http.StatusBadGateway, "连接失败："+err.Error())
		return
	}
	defer resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	if resp.StatusCode >= 400 {
		detail := ""
		if b, rerr := io.ReadAll(io.LimitReader(resp.Body, 512)); rerr == nil {
			detail = strings.TrimSpace(string(b))
		}
		msg := "连通性测试失败（HTTP " + strconv.Itoa(resp.StatusCode) + "）"
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			msg = "API Key 无效或没有访问权限"
		case http.StatusNotFound:
			msg = "模型不存在或请求地址错误"
		case http.StatusTooManyRequests:
			msg = "请求过于频繁，已被服务商限流"
		case http.StatusBadRequest:
			msg = "请求参数不被服务商接受，请检查模型 ID"
		default:
			if resp.StatusCode >= 500 {
				msg = "服务商服务异常"
			}
		}
		if detail != "" {
			if len(detail) > 200 {
				detail = detail[:200]
			}
			msg += "：" + detail
		}
		c.WriteError(http.StatusBadGateway, msg)
		return
	}

	supportsVision := testVisionSupport(req.BaseURL, req.APIKey, req.Model, req.APIShape)

	c.OK(map[string]interface{}{
		"ok":              true,
		"model":           req.Model,
		"latency_ms":      latency,
		"supports_vision": supportsVision,
	})
}
