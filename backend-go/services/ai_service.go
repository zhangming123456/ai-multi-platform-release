package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"

	"ai-multi-platform-release/backend-go/models"
)

// AIGenerationError 表示 AI 生成失败，附带可用模型配置列表供前端提示切换。
type AIGenerationError struct {
	Message        string
	AvailablePlans []map[string]interface{}
}

func (e *AIGenerationError) Error() string {
	return e.Message
}

type AIVariant struct {
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Hashtags []string `json:"hashtags"`
}

type UploadedFile struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

var platformStyleMap = map[string]string{
	"xiaohongshu":  "小红书风格：活泼、种草、使用emoji、段落短小、口语化",
	"douyin":       "抖音风格：吸引眼球、节奏快、有悬念感、适合短视频文案",
	"wechat_mp":    "微信公众号风格：专业、深度、结构清晰、适合长文阅读",
	"wechat_video": "视频号风格：简洁、正能量、适合中年受众、有温度",
}

func getAIBaseURL() string {
	return web.AppConfig.DefaultString("ai_api_base_url", "https://api.deepseek.com/v1")
}

func getAIModel() string {
	return web.AppConfig.DefaultString("ai_model", "deepseek-chat")
}

func getAIAPIKey() string {
	v, _ := web.AppConfig.String("ai_api_key")
	return v
}

// getAvailablePlans 返回所有已启用的模型配置，供错误提示使用。
func getAvailablePlans() ([]map[string]interface{}, error) {
	o := GetOrm()
	var plans []models.ModelConfig
	_, err := o.QueryTable(new(models.ModelConfig)).
		Filter("enabled", true).
		OrderBy("created_at").
		All(&plans)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(plans))
	for _, p := range plans {
		result = append(result, map[string]interface{}{
			"id":           p.ID,
			"name":         p.Name,
			"display_name": p.DisplayName,
			"provider":     p.Provider,
			"model":        p.Model,
		})
	}
	return result, nil
}

// parseModelField 解析 model 字段，兼容 JSON 数组格式和旧版逗号分隔格式。
func parseModelField(raw string) []map[string]interface{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") {
		var data []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &data); err == nil {
			var result []map[string]interface{}
			for _, item := range data {
				id, _ := item["id"].(string)
				if id == "" {
					continue
				}
				types, ok := item["types"]
				if !ok {
					types = item["type"]
				}
				typesList := make([]interface{}, 0)
				switch t := types.(type) {
				case string:
					typesList = append(typesList, t)
				case []interface{}:
					typesList = t
				default:
					typesList = append(typesList, "text")
				}
				result = append(result, map[string]interface{}{
					"id":            id,
					"types":         typesList,
					"contextInput":  item["contextInput"],
					"contextOutput": item["contextOutput"],
				})
			}
			return result
		}
	}
	var result []map[string]interface{}
	for _, m := range strings.Split(raw, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			result = append(result, map[string]interface{}{
				"id": m, "types": []interface{}{"text"}, "contextInput": nil, "contextOutput": nil,
			})
		}
	}
	return result
}

// ResolveModelConfig 解析模型配置，返回 (api_key, base_url, model, plan_name, supports_vision)。
// 失败时返回 *AIGenerationError。
func ResolveModelConfig(planID, modelID string) (string, string, string, string, bool, error) {
	apiKey := getAIAPIKey()
	baseURL := getAIBaseURL()
	model := getAIModel()
	planName := "默认配置"
	supportsVision := false

	if planID != "" {
		o := GetOrm()
		plan := &models.ModelConfig{ID: planID}
		if err := o.Read(plan); err == nil {
			if plan.APIKey != "" {
				apiKey = plan.APIKey
			}
			if plan.BaseURL != "" {
				baseURL = plan.BaseURL
			}
			entries := parseModelField(plan.Model)
			modelIDs := make([]string, 0, len(entries))
			for _, e := range entries {
				if id, ok := e["id"].(string); ok {
					modelIDs = append(modelIDs, id)
				}
			}
			if modelID != "" && containsString(modelIDs, modelID) {
				model = modelID
			} else if len(modelIDs) > 0 {
				model = modelIDs[0]
			}
			for _, e := range entries {
				id, _ := e["id"].(string)
				if id == model {
					types, _ := e["types"].([]interface{})
					for _, t := range types {
						if ts, ok := t.(string); ok && ts == "vision" {
							supportsVision = true
							break
						}
					}
					break
				}
			}
			if !supportsVision {
				supportsVision = plan.Multimodal
			}
			if plan.Name != "" {
				planName = plan.Name
			} else {
				planName = model
			}
		} else {
			available, aerr := getAvailablePlans()
			if aerr != nil {
				return "", "", "", "", false, aerr
			}
			return "", "", "", "", false, &AIGenerationError{
				Message:        fmt.Sprintf("未找到指定的模型配置（plan_id=%s），请切换到可用的模型配置。", planID),
				AvailablePlans: available,
			}
		}
	}

	if apiKey == "" {
		available, aerr := getAvailablePlans()
		if aerr != nil {
			return "", "", "", "", false, aerr
		}
		return "", "", "", "", false, &AIGenerationError{
			Message:        "未配置 API Key，请在模型配置中添加有效的 API 密钥后重试。",
			AvailablePlans: available,
		}
	}

	return apiKey, baseURL, model, planName, supportsVision, nil
}

type chatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

// writeAILog 将模型调用请求或响应记录到 backend-go/logs/ai/ 目录，按天分片。
// timestamp 由调用方统一生成，便于同一次调用的 request/response 文件名一一对应。
func writeAILog(timestamp, suffix, callType, baseURL, model string, stream bool, statusCode int, body []byte) {
	logDir := filepath.Join("logs", "ai")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}
	filename := fmt.Sprintf("%s_%s.%s.log", timestamp, callType, suffix)
	path := filepath.Join(logDir, filename)
	entry := map[string]interface{}{
		"timestamp": time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		"type":      callType,
		"base_url":  baseURL,
		"model":     model,
		"stream":    stream,
		"body_size": len(body),
		"body":      string(abbreviateBase64InJSON(body)),
	}
	if suffix == "response" {
		entry["status_code"] = statusCode
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0644)
}

// abbreviateBase64InJSON 将 JSON body 中的长 base64 字符串（含 data URI）替换为 [base64:<长度>] 占位符，便于日志阅读。
func abbreviateBase64InJSON(body []byte) []byte {
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	v = abbreviateBase64Value(v)
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

func abbreviateBase64Value(v interface{}) interface{} {
	switch x := v.(type) {
	case string:
		if isBase64Like(x) {
			return fmt.Sprintf("[base64:%d]", len(x))
		}
		return x
	case map[string]interface{}:
		for k, val := range x {
			x[k] = abbreviateBase64Value(val)
		}
		return x
	case []interface{}:
		for i, val := range x {
			x[i] = abbreviateBase64Value(val)
		}
		return x
	default:
		return v
	}
}

var base64LikeRegex = regexp.MustCompile(`^[A-Za-z0-9+/=]{200,}$`)

func isBase64Like(s string) bool {
	if strings.HasPrefix(s, "data:") && strings.Contains(s, ";base64,") {
		return true
	}
	return base64LikeRegex.MatchString(s)
}

// loggingBody 包装 http.Response.Body，在读取结束时把响应内容写入 AI 调用日志。
type loggingBody struct {
	rc        io.ReadCloser
	buf       *bytes.Buffer
	closed    bool
	timestamp string
	callType  string
	baseURL   string
	model     string
	stream    bool
}

func (b *loggingBody) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	if n > 0 {
		b.buf.Write(p[:n])
	}
	return n, err
}

func (b *loggingBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	writeAILog(b.timestamp, "response", b.callType, b.baseURL, b.model, b.stream, http.StatusOK, b.buf.Bytes())
	return b.rc.Close()
}

func callChatCompletions(apiKey, baseURL, model string, messages []chatMessage, stream bool) (*http.Response, error) {
	payload := chatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.8,
		MaxTokens:   4000,
		Stream:      stream,
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 120 * time.Second}
	ts := time.Now().Format("2006-01-02_20060102_150405.000")
	writeAILog(ts, "request", "chat.completions", baseURL, model, stream, 0, reqBody)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	const maxRespSize = 10 * 1024 * 1024
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespSize))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	writeAILog(ts, "response", "chat.completions", baseURL, model, stream, resp.StatusCode, respBody)
	return resp, nil
}

// toolDefinition 描述一个 function calling 工具（OpenAI 兼容格式）。
type toolDefinition struct {
	Type     string           `json:"type"` // 固定 "function"
	Function toolFunctionSpec `json:"function"`
}

// toolFunctionSpec 工具的函数签名。
type toolFunctionSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// toolCallResult 模型返回的工具调用。
type toolCallResult struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// callChatCompletionsWithTools 调用 chat completions，支持 tools/function calling。
// tools 为空时等价于普通调用；toolChoice 为 "auto"/"none"/{"type":"function","function":{"name":"xxx"}}。
func callChatCompletionsWithTools(apiKey, baseURL, model string, messages []chatMessage, tools []toolDefinition, toolChoice interface{}, maxTokens int) (*http.Response, error) {
	if maxTokens <= 0 {
		maxTokens = 4000
	}
	payload := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.4,
		"max_tokens":  maxTokens,
		"stream":      false,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
		if toolChoice != nil {
			payload["tool_choice"] = toolChoice
		} else {
			payload["tool_choice"] = "auto"
		}
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 120 * time.Second}
	ts := time.Now().Format("2006-01-02_20060102_150405.000")
	writeAILog(ts, "request", "chat.completions.tools", baseURL, model, false, 0, reqBody)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	const maxRespSize = 10 * 1024 * 1024
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespSize))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	writeAILog(ts, "response", "chat.completions.tools", baseURL, model, false, resp.StatusCode, respBody)
	return resp, nil
}

// callChatCompletionsStreamWithTools 流式调用 chat completions，支持 tools/function calling。
// 返回的 Response.Body 需要调用方自行关闭。
func callChatCompletionsStreamWithTools(apiKey, baseURL, model string, messages []chatMessage, tools []toolDefinition, toolChoice interface{}, maxTokens int) (*http.Response, error) {
	if maxTokens <= 0 {
		maxTokens = 4000
	}
	payload := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.4,
		"max_tokens":  maxTokens,
		"stream":      true,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
		if toolChoice != nil {
			payload["tool_choice"] = toolChoice
		} else {
			payload["tool_choice"] = "auto"
		}
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{Timeout: 120 * time.Second}
	ts := time.Now().Format("2006-01-02_20060102_150405.000")
	writeAILog(ts, "request", "chat.completions.stream.tools", baseURL, model, true, 0, reqBody)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	resp.Body = &loggingBody{
		rc:        resp.Body,
		buf:       &buf,
		timestamp: ts,
		callType:  "chat.completions.stream.tools",
		baseURL:   baseURL,
		model:     model,
		stream:    true,
	}
	return resp, nil
}

// GenerateContentVariants 生成多个内容变体。
func GenerateContentVariants(topic, platform, style string, keywords []string, count int, planID, modelID string) ([]AIVariant, error) {
	apiKey, baseURL, model, planName, _, err := ResolveModelConfig(planID, modelID)
	if err != nil {
		return nil, err
	}

	platformStyle := platformStyleMap[platform]
	if platformStyle == "" {
		platformStyle = "通用社交媒体风格"
	}
	if style != "" {
		platformStyle += "，额外要求：" + style
	}
	keywordsStr := ""
	if len(keywords) > 0 {
		keywordsStr = "，关键词：" + strings.Join(keywords, ", ")
	}
	subject := strings.TrimSpace(topic)
	if subject == "" {
		if len(keywords) > 0 {
			subject = "围绕关键词进行创作：" + strings.Join(keywords, "、")
		} else {
			subject = "通用内容创作"
		}
	}

	prompt := fmt.Sprintf(`请为以下主题生成 %d 个不同风格的内容变体，目标平台：%s。
风格要求：%s%s

主题：%s

请以 JSON 数组格式返回，每个变体包含 title（标题）、body（正文）、hashtags（标签列表）。
直接返回 JSON，不要包含其他说明文字。`, count, platform, platformStyle, keywordsStr, subject)

	resp, err := callChatCompletions(apiKey, baseURL, model, []chatMessage{
		{Role: "system", Content: "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。使用中文回复。"},
		{Role: "user", Content: prompt},
	}, false)
	if err != nil {
		return nil, aiCallError(planName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, aiCallError(planName, err)
	}
	if len(result.Choices) == 0 {
		return nil, aiCallError(planName, fmt.Errorf("模型无返回内容"))
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if strings.HasPrefix(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) > 1 {
			content = strings.TrimSpace(parts[1])
			content = strings.TrimPrefix(content, "json")
			content = strings.TrimSpace(content)
		}
	}

	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		available, aerr := getAvailablePlans()
		if aerr != nil {
			return nil, aerr
		}
		return nil, &AIGenerationError{
			Message:        fmt.Sprintf("模型「%s」返回的内容格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName),
			AvailablePlans: available,
		}
	}

	variants := make([]AIVariant, 0, len(data))
	for _, item := range data {
		hashtags := make([]string, 0)
		if h, ok := item["hashtags"].([]interface{}); ok {
			for _, v := range h {
				if s, ok := v.(string); ok {
					hashtags = append(hashtags, s)
				}
			}
		}
		variants = append(variants, AIVariant{
			Title:    toString(item["title"]),
			Body:     toString(item["body"]),
			Hashtags: hashtags,
		})
	}
	return variants, nil
}

func aiCallError(planName string, err error) error {
	available, aerr := getAvailablePlans()
	if aerr != nil {
		return aerr
	}
	return &AIGenerationError{
		Message:        fmt.Sprintf("模型「%s」调用失败：%s。请尝试切换到其他模型配置后重试。", planName, err),
		AvailablePlans: available,
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

var titleRe = regexp.MustCompile(`【标题】\s*\n?(.*?)(?:【正文】|\z)`)
var bodyRe = regexp.MustCompile(`【正文】\s*\n?(.*?)(?:【标签】|\z)`)
var tagsRe = regexp.MustCompile(`【标签】\s*\n?(.*?)\z`)

// ParseStreamedContent 解析流式生成的文本内容，提取标题、正文和标签。
func ParseStreamedContent(content string) AIVariant {
	title := ""
	body := ""
	hashtagsStr := ""

	if m := titleRe.FindStringSubmatch(content); m != nil {
		title = strings.TrimSpace(m[1])
	}
	if m := bodyRe.FindStringSubmatch(content); m != nil {
		body = strings.TrimSpace(m[1])
	}
	if m := tagsRe.FindStringSubmatch(content); m != nil {
		hashtagsStr = strings.TrimSpace(m[1])
	}

	if title == "" && body == "" {
		contentClean := strings.TrimSpace(content)
		if strings.HasPrefix(contentClean, "```") {
			parts := strings.Split(contentClean, "```")
			if len(parts) > 1 {
				contentClean = strings.TrimSpace(parts[1])
				contentClean = strings.TrimPrefix(contentClean, "json")
				contentClean = strings.TrimSpace(contentClean)
			}
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(contentClean), &data); err == nil {
			hashtags := make([]string, 0)
			if h, ok := data["hashtags"].([]interface{}); ok {
				for _, v := range h {
					if s, ok := v.(string); ok {
						hashtags = append(hashtags, s)
					}
				}
			}
			return AIVariant{
				Title:    toString(data["title"]),
				Body:     toString(data["body"]),
				Hashtags: hashtags,
			}
		}
		// 兼容 JSON 数组格式
		var list []map[string]interface{}
		if err := json.Unmarshal([]byte(contentClean), &list); err == nil && len(list) > 0 {
			item := list[0]
			hashtags := make([]string, 0)
			if h, ok := item["hashtags"].([]interface{}); ok {
				for _, v := range h {
					if s, ok := v.(string); ok {
						hashtags = append(hashtags, s)
					}
				}
			}
			return AIVariant{
				Title:    toString(item["title"]),
				Body:     toString(item["body"]),
				Hashtags: hashtags,
			}
		}
		return AIVariant{Title: "(未解析到标题)", Body: strings.TrimSpace(content)}
	}

	hashtags := make([]string, 0)
	for _, t := range regexp.MustCompile(`[\s,，]+`).Split(hashtagsStr, -1) {
		t = strings.TrimSpace(strings.TrimPrefix(t, "#"))
		if t != "" {
			hashtags = append(hashtags, t)
		}
	}
	return AIVariant{Title: title, Body: body, Hashtags: hashtags}
}

// AIStreamEvent 流式生成事件。事件类型: log / chunk / done / error / result / request_data / response_data。
type AIStreamEvent struct {
	Event          string                   `json:"event"`
	Level          string                   `json:"level,omitempty"`
	Message        string                   `json:"message,omitempty"`
	Text           string                   `json:"text,omitempty"`
	Variant        *AIVariant               `json:"variant,omitempty"`
	Data           interface{}              `json:"data,omitempty"`
	Detail         string                   `json:"detail,omitempty"`
	Model          string                   `json:"model,omitempty"`
	PlanName       string                   `json:"plan_name,omitempty"`
	AvailablePlans []map[string]interface{} `json:"available_plans,omitempty"`
}

func buildVisionContent(prompt string, files []UploadedFile) []interface{} {
	parts := make([]interface{}, 0, len(files)+1)
	parts = append(parts, map[string]interface{}{"type": "text", "text": prompt})
	for _, f := range files {
		imgURL := f.URL
		if imgURL == "" {
			imgURL = f.Data
		}
		if strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://") {
			if strings.HasPrefix(f.MimeType, "video/") {
				parts = append(parts, map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": imgURL},
				})
			} else {
				parts = append(parts, map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]interface{}{"url": imgURL},
				})
			}
		} else {
			dataURL := "data:" + f.MimeType + ";base64," + f.Data
			if strings.HasPrefix(f.MimeType, "video/") {
				parts = append(parts, map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": dataURL},
				})
			} else {
				parts = append(parts, map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]interface{}{"url": dataURL},
				})
			}
		}
	}
	return parts
}

// GenerateContentStream 流式生成内容，通过 channel 发送事件。
func GenerateContentStream(topic, platform, style string, keywords []string, planID, modelID string, files []UploadedFile) (<-chan AIStreamEvent, error) {
	events := make(chan AIStreamEvent, 16)

	apiKey, baseURL, model, planName, _, err := ResolveModelConfig(planID, modelID)
	if err != nil {
		if aiErr, ok := err.(*AIGenerationError); ok {
			go func() {
				events <- AIStreamEvent{Event: "error", Message: aiErr.Message, AvailablePlans: aiErr.AvailablePlans}
				close(events)
			}()
			return events, nil
		}
		return nil, err
	}

	go func() {
		defer close(events)
		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}

		platformStyle := platformStyleMap[platform]
		if platformStyle == "" {
			platformStyle = "通用社交媒体风格"
		}
		if style != "" {
			platformStyle += "，额外要求：" + style
		}
		keywordsStr := ""
		if len(keywords) > 0 {
			keywordsStr = "，关键词：" + strings.Join(keywords, ", ")
		}
		subject := strings.TrimSpace(topic)
		if subject == "" {
			if len(keywords) > 0 {
				subject = "围绕关键词进行创作：" + strings.Join(keywords, "、")
			} else if len(files) > 0 {
				subject = "围绕上传的图片/视频素材进行创作"
			} else {
				subject = "通用内容创作"
			}
		}
		fileHint := ""
		if len(files) > 0 {
			fileHint = "\n\n请结合上传的图片/视频内容进行分析创作，将素材中的关键信息融入文案。"
		}
		prompt := fmt.Sprintf(`请为以下主题生成一篇适合%s发布的内容。
风格要求：%s%s

主题：%s%s

请严格按以下格式输出（不要添加其他内容）：

【标题】
标题内容

【正文】
正文内容

【标签】
#标签1 #标签2 #标签3`, platform, platformStyle, keywordsStr, subject, fileHint)

		if len(files) > 0 {
			imgCount := 0
			vidCount := 0
			for _, f := range files {
				if strings.HasPrefix(f.MimeType, "image/") {
					imgCount++
				} else if strings.HasPrefix(f.MimeType, "video/") {
					vidCount++
				}
			}
			parts := make([]string, 0, 2)
			if imgCount > 0 {
				parts = append(parts, fmt.Sprintf("%d 张图片", imgCount))
			}
			if vidCount > 0 {
				parts = append(parts, fmt.Sprintf("%d 个视频", vidCount))
			}
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("附带 %s（多模态分析）", strings.Join(parts, "、"))}
		}

		messages := []chatMessage{{Role: "system", Content: "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。使用中文回复。"}}
		if len(files) > 0 {
			messages = append(messages, chatMessage{Role: "user", Content: buildVisionContent(prompt, files)})
		} else {
			messages = append(messages, chatMessage{Role: "user", Content: prompt})
		}

		resp, err := callChatCompletions(apiKey, baseURL, model, messages, true)
		if err != nil {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s。请尝试切换到其他模型配置后重试。", planName, err)}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d %s。请尝试切换到其他模型配置后重试。", planName, resp.StatusCode, string(b))}
			return
		}

		fullContent := ""
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				delta := chunk.Choices[0].Delta.Content
				fullContent += delta
				events <- AIStreamEvent{Event: "chunk", Text: delta}
			}
		}

		variant := ParseStreamedContent(fullContent)
		events <- AIStreamEvent{
			Event:    "done",
			Variant:  &variant,
			Model:    model,
			PlanName: planName,
		}
	}()

	return events, nil
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// ModelSupportsFiles 判断指定模型配置是否支持文件上传（多模态）。
func ModelSupportsFiles(planID, modelID string) bool {
	if planID == "" {
		return false
	}
	o := GetOrm()
	plan := &models.ModelConfig{ID: planID}
	if err := o.Read(plan); err != nil {
		return false
	}
	for _, e := range parseModelField(plan.Model) {
		id, _ := e["id"].(string)
		if id == "" || (modelID != "" && id != modelID) {
			continue
		}
		types, ok := e["types"].([]interface{})
		if !ok {
			continue
		}
		for _, t := range types {
			if ts, ok := t.(string); ok && (ts == "vision" || ts == "image" || ts == "video") {
				return true
			}
		}
	}
	return false
}
