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
	"sync"
	"time"
	"unicode/utf8"

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

// VariantScore 文案质量评分结果（启发式规则，0-100）。
type VariantScore struct {
	Total      int      `json:"total"`
	TitleScore int      `json:"title_score"`
	BodyScore  int      `json:"body_score"`
	TagScore   int      `json:"tag_score"`
	ExtraScore int      `json:"extra_score"`
	Tags       []string `json:"tags"`
	Comment    string   `json:"comment"`
}

// ScoreVariant 基于启发式规则对文案质量进行评分，用于多版本优选。
// 评分维度：标题（30 分）、正文（40 分）、话题标签（20 分）、附加加分（10 分）。
func ScoreVariant(v AIVariant, platform string) VariantScore {
	sc := VariantScore{}

	titleLen := utf8.RuneCountInString(v.Title)
	switch {
	case titleLen == 0:
		sc.Tags = append(sc.Tags, "标题为空")
	case titleLen <= 40:
		sc.TitleScore = 30
		sc.Tags = append(sc.Tags, "标题简洁有力")
	default:
		sc.TitleScore = 18
		sc.Tags = append(sc.Tags, "标题偏长")
	}

	bodyLen := utf8.RuneCountInString(v.Body)
	switch {
	case bodyLen == 0:
		sc.Tags = append(sc.Tags, "正文为空")
	case bodyLen >= 80 && bodyLen <= 2000:
		sc.BodyScore = 40
		sc.Tags = append(sc.Tags, "正文内容充实")
	case bodyLen > 2000:
		sc.BodyScore = 28
		sc.Tags = append(sc.Tags, "正文偏长")
	default:
		sc.BodyScore = 20
		sc.Tags = append(sc.Tags, "正文偏短")
	}

	switch {
	case len(v.Hashtags) >= 3 && len(v.Hashtags) <= 5:
		sc.TagScore = 20
		sc.Tags = append(sc.Tags, "话题标签数量适中")
	case len(v.Hashtags) > 0:
		sc.TagScore = 10
		sc.Tags = append(sc.Tags, "话题标签偏少")
	default:
		sc.Tags = append(sc.Tags, "缺少话题标签")
	}

	emojis := 0
	for _, r := range v.Body {
		if r >= 0x1F000 && r <= 0x1FAFF {
			emojis++
		}
	}
	switch platform {
	case "xiaohongshu", "weibo", "douyin":
		if emojis >= 2 {
			sc.ExtraScore = 10
			sc.Tags = append(sc.Tags, "emoji 使用到位")
		} else if emojis > 0 {
			sc.ExtraScore = 5
		} else {
			sc.Tags = append(sc.Tags, "建议增加 emoji 提升氛围感")
		}
	default:
		if emojis <= 1 {
			sc.ExtraScore = 10
			sc.Tags = append(sc.Tags, "风格克制，符合平台调性")
		} else {
			sc.ExtraScore = 5
		}
	}

	sc.Total = sc.TitleScore + sc.BodyScore + sc.TagScore + sc.ExtraScore
	switch {
	case sc.Total >= 90:
		sc.Comment = "优秀"
	case sc.Total >= 75:
		sc.Comment = "良好"
	case sc.Total >= 60:
		sc.Comment = "达标"
	default:
		sc.Comment = "待优化"
	}
	return sc
}

type UploadedFile struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

// contentModelRequestQueue 串行执行内容创作模型请求，避免多平台或多用户同时触发模型限流。
var (
	contentModelRequestQueue = make(chan func(), 64)
	contentModelQueueOnce    sync.Once
)

func enqueueContentModelRequest(job func()) {
	contentModelQueueOnce.Do(func() {
		go func() {
			for queuedJob := range contentModelRequestQueue {
				queuedJob()
			}
		}()
	})
	contentModelRequestQueue <- job
}

var platformStyleMap = map[string]string{
	"xiaohongshu":    "小红书风格：活泼、种草、使用emoji、段落短小、口语化",
	"douyin":         "抖音风格：吸引眼球、节奏快、有悬念感、适合短视频文案",
	"wechat_mp":      "微信公众号风格：专业、深度、结构清晰、适合长文阅读",
	"wechat_video":   "视频号风格：简洁、正能量、适合中年受众、有温度",
	"wechat_moments": "朋友圈风格：极简精炼、生活化、适合转发",
	"weibo":          "微博风格：年轻活泼、高互动、强话题性、短平快",
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
	filename := fmt.Sprintf("%s_%s.%s.json", timestamp, callType, suffix)
	path := filepath.Join(logDir, filename)
	entry := map[string]interface{}{
		"timestamp": time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		"type":      callType,
		"base_url":  baseURL,
		"model":     model,
		"stream":    stream,
		"body_size": len(body),
		"body":      formatAILogBody(body),
	}
	if suffix == "response" {
		entry["status_code"] = statusCode
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0644)
}

// formatAILogBody 将普通 JSON 和 SSE 响应解析为对象/数组，避免日志 body 变成转义字符串。
func formatAILogBody(body []byte) interface{} {
	abbreviated := abbreviateBase64InJSON(body)
	var value interface{}
	if json.Unmarshal(abbreviated, &value) == nil {
		return value
	}

	lines := strings.Split(string(abbreviated), "\n")
	chunks := make([]interface{}, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk interface{}
		if json.Unmarshal([]byte(data), &chunk) == nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) > 0 {
		return chunks
	}
	return string(abbreviated)
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

// loadCampaignContext 加载活动并构造注入内容。
// 活动不存在或宣发图加载失败时返回空值，不阻断创作。
func loadCampaignContext(campaignID string) (contextText string, files []UploadedFile, campaignName string) {
	if strings.TrimSpace(campaignID) == "" {
		return "", nil, ""
	}
	var campaign models.Campaign
	if err := GetOrm().QueryTable(new(models.Campaign)).
		Filter("id", campaignID).
		Filter("status", models.CampaignStatusActive).
		One(&campaign); err != nil {
		return "", nil, ""
	}
	var b strings.Builder
	b.WriteString("【活动背景】\n")
	b.WriteString("活动名称：" + campaign.Name + "\n")
	if campaign.Description != "" {
		b.WriteString("活动介绍：" + campaign.Description + "\n")
	}
	if campaign.Location != "" {
		b.WriteString("活动地点：" + campaign.Location + "\n")
	}
	var platforms []string
	if campaign.Platforms != "" {
		_ = json.Unmarshal([]byte(campaign.Platforms), &platforms)
	}
	if len(platforms) > 0 {
		b.WriteString("面向平台：" + strings.Join(platforms, "、") + "\n")
	}
	b.WriteString("\n请围绕以上活动背景进行创作，内容需贴合活动调性与卖点。")
	contextText = b.String()

	var mediaURLs []string
	if campaign.MediaURLs != "" {
		_ = json.Unmarshal([]byte(campaign.MediaURLs), &mediaURLs)
	}
	for _, u := range mediaURLs {
		uf, err := loadImageAsUploadedFile(u)
		if err != nil || uf == nil {
			continue
		}
		files = append(files, *uf)
	}
	return contextText, files, campaign.Name
}

// buildContentVariantMessages 构建内容创作 AI 调用的 messages / tools / toolChoice。
// count > 1 时要求模型一次返回多个变体（submit_content_variants），否则返回单个（submit_content_variant）。
func buildContentVariantMessages(topic, platform, style string, keywords []string, files []UploadedFile, count int, campaignContext string) ([]chatMessage, []toolDefinition, interface{}) {
	platformStyle := platformStyleMap[platform]
	if platformStyle == "" {
		platformStyle = "通用社交媒体风格"
	}
	if style != "" {
		platformStyle += "，额外要求：" + style
	}
	if st := loadStyleTemplate(platform); st != "" {
		platformStyle = st
	}
	keywordsStr := ""
	if len(keywords) > 0 {
		keywordsStr = "，关键词：" + strings.Join(keywords, ", ")
	}
	if st := loadStyleTemplateKeywords(platform); st != "" {
		if keywordsStr == "" {
			keywordsStr = "，关键词：" + st
		} else {
			keywordsStr += "，" + st
		}
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
	campaignHint := ""
	if campaignContext != "" {
		campaignHint = "\n\n" + campaignContext
	}

	toolName := "submit_content_variant"
	toolDesc := "提交 AI 生成的内容变体（含标题、正文、推荐话题标签）"
	itemSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":    map[string]interface{}{"type": "string", "description": "内容标题，简洁有吸引力"},
			"body":     map[string]interface{}{"type": "string", "description": "内容正文"},
			"hashtags": map[string]interface{}{"type": "array", "description": "推荐话题标签列表，3-5 个，不含 # 前缀", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"title", "body", "hashtags"},
	}

	var schema map[string]interface{}
	var callDesc string
	if count > 1 {
		toolName = "submit_content_variants"
		toolDesc = "提交 AI 生成的多个内容变体（每个含标题、正文、推荐话题标签）"
		schema = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"variants": map[string]interface{}{
					"type":        "array",
					"description": "内容变体列表",
					"items":       itemSchema,
				},
			},
			"required": []string{"variants"},
		}
		callDesc = fmt.Sprintf(`请为以下主题生成 %d 个不同风格的内容变体，目标平台：%s。
风格要求：%s%s

主题：%s%s%s

			请优先调用 %s 工具提交结果；如果当前模型不支持工具调用，则只输出符合上述结构的合法 JSON，不要输出解释文字。每个变体必须包含 title（标题）、body（正文）、hashtags（推荐话题标签列表，不含 # 前缀）。`, count, platform, platformStyle, keywordsStr, subject, fileHint, campaignHint, toolName)
	} else {
		schema = itemSchema
		callDesc = fmt.Sprintf(`请为以下主题生成一篇适合%s发布的内容。
风格要求：%s%s

主题：%s%s%s

			请优先调用 %s 工具提交结果；如果当前模型不支持工具调用，则只输出符合上述结构的合法 JSON，不要输出解释文字。数据结构必须包含：
- title（标题）
- body（正文）
- hashtags（推荐话题标签列表，3-5 个，不含 # 前缀）
			`, platform, platformStyle, keywordsStr, subject, fileHint, campaignHint, toolName)
	}

	systemContent := "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。使用中文回复。优先调用" + toolName + "工具返回结构化结果；如果当前模型不支持工具调用，则只返回符合工具参数结构的合法 JSON，禁止输出解释性文字或 markdown。"
	if pt := loadPromptTemplate(platform); pt != "" {
		systemContent = pt
	}

	messages := []chatMessage{{Role: "system", Content: systemContent}}
	if len(files) > 0 {
		messages = append(messages, chatMessage{Role: "user", Content: buildVisionContent(callDesc, files)})
	} else {
		messages = append(messages, chatMessage{Role: "user", Content: callDesc})
	}

	tools := []toolDefinition{{
		Type: "function",
		Function: toolFunctionSpec{
			Name:        toolName,
			Description: toolDesc,
			Parameters:  schema,
		},
	}}
	toolChoice := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name": toolName,
		},
	}
	return messages, tools, toolChoice
}

// loadPromptTemplate 从数据库读取目标平台启用的默认 Prompt 模板（system 内容），不存在则返回空。
func loadPromptTemplate(platform string) string {
	o := GetOrm()
	var p models.PromptTemplate
	err := o.QueryTable(new(models.PromptTemplate)).
		Filter("platform", platform).
		Filter("status", models.PromptTemplateStatusActive).
		Filter("is_default", true).
		One(&p)
	if err != nil {
		return ""
	}
	role := strings.TrimSpace(p.Role)
	rules := strings.TrimSpace(p.Rules)
	if role == "" {
		role = "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容。使用中文回复。"
	}
	if rules != "" {
		role += "\n强制规则：\n" + rules
	}
	return role
}

// loadStyleTemplate 从数据库读取目标平台启用的风格模板描述。
func loadStyleTemplate(platform string) string {
	o := GetOrm()
	var s models.StyleTemplate
	err := o.QueryTable(new(models.StyleTemplate)).
		Filter("platform", platform).
		Filter("status", models.StyleTemplateStatusActive).
		OrderBy("-created_at").
		One(&s)
	if err != nil || strings.TrimSpace(s.Description) == "" {
		return ""
	}
	return s.Description
}

// loadStyleTemplateKeywords 从数据库读取目标平台启用的风格模板关键词。
func loadStyleTemplateKeywords(platform string) string {
	o := GetOrm()
	var s models.StyleTemplate
	err := o.QueryTable(new(models.StyleTemplate)).
		Filter("platform", platform).
		Filter("status", models.StyleTemplateStatusActive).
		OrderBy("-created_at").
		One(&s)
	if err != nil || strings.TrimSpace(s.Keywords) == "" {
		return ""
	}
	keywords := []string{}
	if err := json.Unmarshal([]byte(s.Keywords), &keywords); err != nil {
		return ""
	}
	return strings.Join(keywords, ", ")
}

func extractHashtags(v interface{}) []string {
	hashtags := make([]string, 0)
	if list, ok := v.([]interface{}); ok {
		for _, item := range list {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					hashtags = append(hashtags, s)
				}
			}
		}
	} else if s, ok := v.(string); ok && s != "" {
		for _, t := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' || r == ' ' || r == '#' }) {
			t = strings.TrimSpace(t)
			if t != "" {
				hashtags = append(hashtags, t)
			}
		}
	}
	return hashtags
}

func buildVariantList(items []map[string]interface{}) []AIVariant {
	variants := make([]AIVariant, 0, len(items))
	for _, item := range items {
		v := AIVariant{Title: toString(item["title"]), Body: toString(item["body"]), Hashtags: extractHashtags(item["hashtags"])}
		if v.Title != "" || v.Body != "" || len(v.Hashtags) > 0 {
			variants = append(variants, v)
		}
	}
	return variants
}

func parseContentVariantArgs(raw string) (AIVariant, bool) {
	clean := stripCodeFence(raw)
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(clean), &obj); err == nil {
		v := AIVariant{Title: toString(obj["title"]), Body: toString(obj["body"]), Hashtags: extractHashtags(obj["hashtags"])}
		if v.Title != "" || v.Body != "" || len(v.Hashtags) > 0 {
			return v, true
		}
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(clean), &list); err == nil && len(list) > 0 {
		v := AIVariant{Title: toString(list[0]["title"]), Body: toString(list[0]["body"]), Hashtags: extractHashtags(list[0]["hashtags"])}
		if v.Title != "" || v.Body != "" || len(v.Hashtags) > 0 {
			return v, true
		}
	}
	return AIVariant{}, false
}

func parseContentVariantsArgs(raw string) ([]AIVariant, bool) {
	clean := stripCodeFence(raw)
	var wrapper struct {
		Variants []map[string]interface{} `json:"variants"`
	}
	if err := json.Unmarshal([]byte(clean), &wrapper); err == nil && len(wrapper.Variants) > 0 {
		return buildVariantList(wrapper.Variants), true
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(clean), &list); err == nil && len(list) > 0 {
		return buildVariantList(list), true
	}
	return nil, false
}

func hashtagsText(hashtags []string) string {
	clean := make([]string, 0, len(hashtags))
	for _, t := range hashtags {
		t = strings.TrimSpace(strings.TrimPrefix(t, "#"))
		if t != "" {
			clean = append(clean, "#"+t)
		}
	}
	return strings.Join(clean, " ")
}

// GenerateContentVariants 生成多个内容变体。
func GenerateContentVariants(topic, platform, style string, keywords []string, count int, planID, modelID, campaignID string) ([]AIVariant, error) {
	apiKey, baseURL, model, planName, supportsVision, err := ResolveModelConfig(planID, modelID)
	if err != nil {
		return nil, err
	}

	contextText, campaignFiles, _ := loadCampaignContext(campaignID)
	files := make([]UploadedFile, 0, len(campaignFiles))
	if supportsVision {
		files = append(files, campaignFiles...)
	}
	if len(campaignFiles) > 0 && !supportsVision {
		fmt.Printf("[ai_service] 活动宣发图已降级为仅文本注入（模型 %s 不支持视觉）\n", model)
	}

	messages, tools, toolChoice := buildContentVariantMessages(topic, platform, style, keywords, files, count, contextText)

	resp, err := callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
	if err != nil {
		return nil, aiCallError(planName, err)
	}
	if resp.StatusCode == http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		lower := strings.ToLower(string(bodyBytes))
		if strings.Contains(lower, "tool_choice") || strings.Contains(lower, "tool choice") {
			resp, err = callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
			if err != nil {
				return nil, aiCallError(planName, err)
			}
		} else {
			return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string           `json:"content"`
				ToolCalls []toolCallResult `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, aiCallError(planName, err)
	}
	if len(result.Choices) == 0 {
		return nil, aiCallError(planName, fmt.Errorf("模型无返回内容"))
	}

	args := ""
	if len(result.Choices[0].Message.ToolCalls) > 0 {
		args = strings.TrimSpace(result.Choices[0].Message.ToolCalls[0].Function.Arguments)
	} else {
		args = strings.TrimSpace(result.Choices[0].Message.Content)
	}

	variants, ok := parseContentVariantsArgs(args)
	if !ok {
		if v, ok2 := parseContentVariantArgs(args); ok2 {
			variants = []AIVariant{v}
			ok = true
		}
	}
	if !ok || len(variants) == 0 {
		available, aerr := getAvailablePlans()
		if aerr != nil {
			return nil, aerr
		}
		return nil, &AIGenerationError{
			Message:        fmt.Sprintf("模型「%s」返回的内容格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName),
			AvailablePlans: available,
		}
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

// generateContentVariantsFallback 在模型不支持 function calling 时使用纯 JSON 请求兜底。
func generateContentVariantsFallback(apiKey, baseURL, model string, messages []chatMessage, count int) ([]AIVariant, error) {
	fallbackMessages := append([]chatMessage(nil), messages...)
	fallbackMessages = append(fallbackMessages, chatMessage{
		Role: "user",
		Content: fmt.Sprintf("当前接口不支持工具调用。请只返回合法 JSON，不要 markdown 或解释文字。%s",
			contentVariantJSONFormat(count)),
	})
	resp, err := callChatCompletionsWithTools(apiKey, baseURL, model, fallbackMessages, nil, nil, 8000)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("模型无返回内容")
	}
	content := stripCodeFence(strings.TrimSpace(result.Choices[0].Message.Content))
	if count > 1 {
		if variants, ok := parseContentVariantsArgs(content); ok {
			return variants, nil
		}
	} else if variant, ok := parseContentVariantArgs(content); ok {
		return []AIVariant{variant}, nil
	}
	if variants, ok := parseContentVariantsArgs(content); ok {
		return variants, nil
	}
	if variant, ok := parseContentVariantArgs(content); ok {
		return []AIVariant{variant}, nil
	}
	return nil, fmt.Errorf("纯 JSON 兜底响应格式异常")
}

func contentVariantJSONFormat(count int) string {
	if count > 1 {
		return fmt.Sprintf("返回 {\"variants\":[...]}，数组中包含 %d 个对象，每个对象包含 title、body、hashtags。", count)
	}
	return "返回 {\"title\":\"标题\",\"body\":\"正文\",\"hashtags\":[\"话题\"]}。"
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

// AIStreamEvent 流式生成事件。事件类型: log / chunk / done / error / result / request_data / response_data。
type AIStreamEvent struct {
	Event          string                   `json:"event"`
	Level          string                   `json:"level,omitempty"`
	Message        string                   `json:"message,omitempty"`
	Text           string                   `json:"text,omitempty"`
	Variant        *AIVariant               `json:"variant,omitempty"`
	VariantIndex   int                      `json:"variant_index,omitempty"`
	Score          *VariantScore            `json:"score,omitempty"`
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

// normalizeContentFiles 将内容创作中的文件链接统一下载为 Base64，避免模型无法访问内网或本地链接。
func normalizeContentFiles(files []UploadedFile) ([]UploadedFile, []error) {
	normalized := make([]UploadedFile, 0, len(files))
	errs := make([]error, 0)
	for _, file := range files {
		if file.Data != "" {
			file.URL = ""
			normalized = append(normalized, file)
			continue
		}
		if file.URL == "" {
			errs = append(errs, fmt.Errorf("文件缺少 data 和 url"))
			continue
		}
		loaded, err := loadImageAsUploadedFile(file.URL)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		loaded.URL = ""
		normalized = append(normalized, *loaded)
	}
	return normalized, errs
}

// GenerateContentStream 流式生成内容，通过 channel 发送事件。
func GenerateContentStream(topic, platform, style string, keywords []string, planID, modelID string, files []UploadedFile, campaignID string, versionNum int) (<-chan AIStreamEvent, error) {
	events := make(chan AIStreamEvent, 16)
	if versionNum <= 0 {
		versionNum = 1
	}
	if versionNum > 3 {
		versionNum = 3
	}

	apiKey, baseURL, model, planName, supportsVision, err := ResolveModelConfig(planID, modelID)
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

	go enqueueContentModelRequest(func() {
		defer close(events)
		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}

		contextText, campaignFiles, campaignName := loadCampaignContext(campaignID)
		allFiles := make([]UploadedFile, 0, len(files)+len(campaignFiles))
		allFiles = append(allFiles, files...)
		if supportsVision {
			allFiles = append(allFiles, campaignFiles...)
		}
		if len(allFiles) > 0 {
			normalizedFiles, fileErrors := normalizeContentFiles(allFiles)
			allFiles = normalizedFiles
			for _, fileErr := range fileErrors {
				events <- AIStreamEvent{Event: "log", Level: "warn", Message: fmt.Sprintf("素材转 Base64 失败，已跳过：%s", fileErr)}
			}
		}
		if len(campaignFiles) > 0 {
			if supportsVision {
				events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("已注入活动「%s」创作背景（含 %d 张宣发图）", campaignName, len(campaignFiles))}
			} else {
				events <- AIStreamEvent{Event: "log", Level: "warn", Message: fmt.Sprintf("已注入活动「%s」创作背景（当前模型不支持视觉，宣发图已降级为仅文本注入）", campaignName)}
			}
		} else if contextText != "" {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("已注入活动「%s」创作背景", campaignName)}
		}

		messages, tools, toolChoice := buildContentVariantMessages(topic, platform, style, keywords, allFiles, versionNum, contextText)

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

		resp, err := callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
		if err != nil {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s。请尝试切换到其他模型配置后重试。", planName, err)}
			return
		}
		if resp.StatusCode == http.StatusBadRequest {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			lower := strings.ToLower(string(bodyBytes))
			if strings.Contains(lower, "tool_choice") || strings.Contains(lower, "tool choice") {
				events <- AIStreamEvent{Event: "log", Level: "warn", Message: "当前模型不支持强制工具调用，已自动降级为自动模式"}
				resp, err = callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
				if err != nil {
					events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s。请尝试切换到其他模型配置后重试。", planName, err)}
					return
				}
			} else {
				events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s。请尝试切换到其他模型配置后重试。", planName, resp.StatusCode, string(bodyBytes))}
				return
			}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s。请尝试切换到其他模型配置后重试。", planName, resp.StatusCode, string(b))}
			return
		}

		events <- AIStreamEvent{Event: "log", Level: "req", Message: "POST /chat/completions → SSE 连接已建立"}

		fullContent := ""
		accumulatedArgs := make(map[int]string)
		accumulatedName := make(map[int]string)
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
						Content   string           `json:"content"`
						ToolCalls []toolCallResult `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fullContent += delta.Content
				events <- AIStreamEvent{Event: "chunk", Text: delta.Content}
			}
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Function.Name != "" {
					accumulatedName[idx] = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					accumulatedArgs[idx] += tc.Function.Arguments
				}
			}
		}
		if err := scanner.Err(); err != nil {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("读取模型流失败：%s", err)}
			return
		}

		var variants []AIVariant
		for idx, args := range accumulatedArgs {
			name := accumulatedName[idx]
			args = strings.TrimSpace(args)
			if args == "" {
				continue
			}
			if name == "submit_content_variants" {
				if list, ok := parseContentVariantsArgs(args); ok && len(list) > 0 {
					variants = list
					events <- AIStreamEvent{Event: "log", Level: "ok", Message: fmt.Sprintf("已解析 submit_content_variants 工具调用结果（%d 个版本）", len(list))}
					break
				}
			} else if name == "submit_content_variant" {
				if v, ok := parseContentVariantArgs(args); ok {
					variants = []AIVariant{v}
					events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已解析 submit_content_variant 工具调用结果"}
					break
				}
			}
		}
		if len(variants) == 0 {
			if list, ok := parseContentVariantsArgs(stripCodeFence(fullContent)); ok && len(list) > 0 {
				variants = list
				events <- AIStreamEvent{Event: "log", Level: "ok", Message: fmt.Sprintf("已从文本内容回退解析 %d 个 JSON 变体", len(list))}
			} else if v, ok := parseContentVariantArgs(stripCodeFence(fullContent)); ok {
				variants = []AIVariant{v}
				events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已从文本内容回退解析 JSON 结果"}
			}
		}
		if len(variants) == 0 {
			events <- AIStreamEvent{Event: "log", Level: "warn", Message: "模型未返回工具调用，正在切换为纯 JSON 兼容模式"}
			fallbackVariants, fallbackErr := generateContentVariantsFallback(apiKey, baseURL, model, messages, versionNum)
			if fallbackErr == nil {
				variants = fallbackVariants
				events <- AIStreamEvent{Event: "log", Level: "ok", Message: fmt.Sprintf("纯 JSON 兼容模式解析成功（%d 个版本）", len(variants))}
			} else {
				events <- AIStreamEvent{Event: "log", Level: "warn", Message: fmt.Sprintf("纯 JSON 兼容模式失败：%s", fallbackErr)}
			}
		}
		if len(variants) == 0 {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」返回的内容格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName)}
			return
		}
		for i := range variants {
			v := variants[i]
			score := ScoreVariant(v, platform)
			events <- AIStreamEvent{
				Event:        "done",
				Variant:      &v,
				VariantIndex: i,
				Score:        &score,
				Model:        model,
				PlanName:     planName,
			}
		}
	})

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
