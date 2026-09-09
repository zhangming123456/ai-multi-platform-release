package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	batchMaxTaskPerCall = 3
)

// BatchTask 表示一个生成任务（平台 × 内容形式），任务内按版本数输出多篇。
type BatchTask struct {
	Platform    string
	ContentForm string
}

// BatchItemResult 是一次 submit_batch_variants 调用中某个组合的解析结果。
type BatchItemResult struct {
	Platform    string
	ContentForm string
	Variants    []AIVariant
}

// BatchRequest 是合并分批生成的完整请求。
type BatchRequest struct {
	Topic           string
	Style           string
	Keywords        []string
	Platforms       []string
	ContentForms    []string
	VersionNum      int
	PlanID          string
	ModelID         string
	Files           []UploadedFile
	CampaignContext string
	CampaignName    string
}

var contentFormSpec = map[string]string{
	"post":           "图文笔记/推文：包含完整标题、正文、推荐话题标签；正文需自然流畅、可直接发布",
	"script":         "短视频脚本：标题 + 分镜/口播稿 + 建议话题标签，脚本需含开场钩子、卖点展示、结尾行动号召",
	"snippet":        "短文案/标题：1 条核心短文案（30-80 字）+ 3-5 条备选标题 + 推荐话题标签",
	"live_script":    "直播脚本：直播主题 + 开场话术 + 产品讲解节奏 + 互动节奏 + 促单话术 + 下播收尾",
	"product_detail": "商品详情页文案：商品主标题 + 卖点清单 + 使用场景 + 规格/参数说明 + 购买引导与信任背书",
}

var contentFormLabel = map[string]string{
	"post":           "图文笔记/推文",
	"script":         "短视频脚本",
	"snippet":        "短文案/标题",
	"live_script":    "直播脚本",
	"product_detail": "商品详情页文案",
}

var platformDisplayNames = map[string]string{
	"wechat_mp":      "公众号",
	"xiaohongshu":    "小红书",
	"douyin":         "抖音",
	"wechat_video":   "视频号",
	"wechat_moments": "朋友圈",
	"weibo":          "微博",
}

// EventContext 是临时录入活动的上下文，由 controller 解析后传入。
type EventContext struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Location    string         `json:"location"`
	MediaFiles  []UploadedFile `json:"media_files"`
}

// BuildGenerationContext 将 campaign_id 或临时活动上下文解析为统一的
// 活动背景文本与素材文件（活动宣发图由调用方按模型能力决定是否携带）。
func BuildGenerationContext(campaignID string, ev *EventContext) (contextText string, files []UploadedFile, name string) {
	if campaignID != "" {
		return loadCampaignContext(campaignID)
	}
	if ev == nil {
		return "", nil, ""
	}
	var b strings.Builder
	b.WriteString("【活动背景】\n")
	b.WriteString("活动名称：" + strings.TrimSpace(ev.Name) + "\n")
	if strings.TrimSpace(ev.Description) != "" {
		b.WriteString("活动介绍：" + strings.TrimSpace(ev.Description) + "\n")
	}
	if strings.TrimSpace(ev.Location) != "" {
		b.WriteString("活动地点：" + strings.TrimSpace(ev.Location) + "\n")
	}
	return b.String(), ev.MediaFiles, strings.TrimSpace(ev.Name)
}

func platformStyleResolved(platform, style string) string {
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
	return platformStyle
}

func batchItemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"platform":     map[string]interface{}{"type": "string", "description": "目标平台标识，取自任务清单"},
			"content_form": map[string]interface{}{"type": "string", "description": "内容形式标识，取自任务清单"},
			"variants": map[string]interface{}{
				"type":        "array",
				"description": "该组合下的内容变体列表，数量等于任务要求版本数",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title":    map[string]interface{}{"type": "string", "description": "内容标题，简洁有吸引力"},
						"body":     map[string]interface{}{"type": "string", "description": "内容正文"},
						"hashtags": map[string]interface{}{"type": "array", "description": "推荐话题标签列表，3-5 个，不含 # 前缀", "items": map[string]interface{}{"type": "string"}},
					},
					"required": []string{"title", "body", "hashtags"},
				},
			},
		},
		"required": []string{"platform", "content_form", "variants"},
	}
}

// buildBatchMessages 组装一批任务的 messages / tools / toolChoice。
// 批内任务来自多个平台时使用通用 system 提示词与任务清单（批任务 Prompt）；
// 当批内任务同属一个配置了专属 Prompt 模板的平台时，沿用该平台专属 system。
func buildBatchMessages(r BatchRequest, tasks []BatchTask) ([]chatMessage, []toolDefinition, interface{}, bool) {
	subject := strings.TrimSpace(r.Topic)
	if subject == "" {
		if len(r.Keywords) > 0 {
			subject = "围绕关键词进行创作：" + strings.Join(r.Keywords, "、")
		} else if len(r.Files) > 0 {
			subject = "围绕上传的图片/视频素材进行创作"
		} else {
			subject = "通用内容创作"
		}
	}
	fileHint := ""
	if len(r.Files) > 0 {
		fileHint = "\n\n请结合上传的图片/视频内容进行分析创作，将素材中的关键信息融入每篇文案。"
	}
	campaignHint := ""
	if strings.TrimSpace(r.CampaignContext) != "" {
		campaignHint = "\n\n【活动背景】（仅提供一次，供本批所有任务参考）\n" + strings.TrimSpace(r.CampaignContext)
	}

	singlePlatform := ""
	platformSet := map[string]bool{}
	for _, t := range tasks {
		platformSet[t.Platform] = true
		if singlePlatform == "" {
			singlePlatform = t.Platform
		} else if singlePlatform != t.Platform {
			singlePlatform = ""
		}
	}
	ownSystem := ""
	if singlePlatform != "" && len(platformSet) == 1 {
		ownSystem = loadPromptTemplate(singlePlatform)
	}

	toolName := "submit_batch_variants"
	callLines := make([]string, 0, 8)
	if ownSystem == "" {
		callLines = append(callLines, fmt.Sprintf("请一次性为本批 %d 个任务分别创作内容。每个任务独立成篇，平台风格、内容形式互不混淆，禁止互相串味。", len(tasks)))
	} else {
		callLines = append(callLines, fmt.Sprintf("请为目标平台一次性完成本批 %d 个任务，每个任务按对应内容形式独立创作。", len(tasks)))
	}
	callLines = append(callLines, subject+fileHint+campaignHint)

	platformRefs := make([]string, 0)
	formRefs := make([]string, 0)
	seenPlatform := map[string]bool{}
	seenForm := map[string]bool{}
	for _, t := range tasks {
		if !seenPlatform[t.Platform] {
			seenPlatform[t.Platform] = true
			display := platformDisplayNames[t.Platform]
			if display == "" {
				display = t.Platform
			}
			platformRefs = append(platformRefs, fmt.Sprintf("%s（标识 %s，风格：%s）", display, t.Platform, platformStyleResolved(t.Platform, r.Style)))
		}
		if !seenForm[t.ContentForm] {
			seenForm[t.ContentForm] = true
			label := contentFormLabel[t.ContentForm]
			if label == "" {
				label = t.ContentForm
			}
			formRefs = append(formRefs, fmt.Sprintf("%s（标识 %s，要求：%s）", label, t.ContentForm, contentFormSpec[t.ContentForm]))
		}
	}
	if len(platformRefs) > 0 {
		callLines = append(callLines, "【平台说明】\n"+strings.Join(platformRefs, "\n"))
	}
	if len(formRefs) > 0 {
		callLines = append(callLines, "【内容形式要求】\n"+strings.Join(formRefs, "\n"))
	}
	taskLines := make([]string, 0, len(tasks))
	for i, t := range tasks {
		label := contentFormLabel[t.ContentForm]
		if label == "" {
			label = t.ContentForm
		}
		taskLines = append(taskLines, fmt.Sprintf("任务 %d：%s × %s，输出 %d 个互有差异的版本", i+1, platformDisplayNames[t.Platform], label, r.VersionNum))
	}
	callLines = append(callLines, "【任务清单】\n"+strings.Join(taskLines, "\n"))
	callLines = append(callLines, fmt.Sprintf(`【输出约定】
- 通过 %s 工具一次性提交本批全部结果；若当前模型不支持工具调用，则只输出符合工具参数结构的合法 JSON
- JSON 结构：{ "items": [ { "platform": "...", "content_form": "...", "variants": [ { "title": "...", "body": "...", "hashtags": [...] } ] } ] }
- items 与任务清单一一对应（platform + content_form 唯一标识一个任务），variants 长度等于该任务版本数
- items 中的 platform 必须填写平台标识（xiaohongshu/douyin/wechat_mp/wechat_video/wechat_moments/weibo），content_form 必须填写形式标识（post/script/snippet/live_script/product_detail），禁止使用中文名称或占位编号
- 仅输出结果，不要输出解释文字`, toolName))

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"items": map[string]interface{}{
				"type":        "array",
				"description": "本批全部任务的结果，按任务清单顺序逐条给出",
				"items":       batchItemSchema(),
			},
		},
		"required": []string{"items"},
	}
	systemContent := "你是一个专业的社交媒体内容创作助手，擅长为不同平台生成适配的优质内容，并能一次完成多个独立创作任务。使用中文回复。优先调用 " + toolName + " 工具返回结构化结果；如果当前模型不支持工具调用，则只返回符合工具参数结构的合法 JSON，禁止输出解释性文字或 markdown。"
	if ownSystem != "" {
		systemContent = ownSystem
	}
	messages := []chatMessage{{Role: "system", Content: systemContent}}
	if len(r.Files) > 0 {
		callDesc := strings.Join(callLines, "\n\n")
		messages = append(messages, chatMessage{Role: "user", Content: buildVisionContent(callDesc, r.Files)})
	} else {
		messages = append(messages, chatMessage{Role: "user", Content: strings.Join(callLines, "\n\n")})
	}
	tools := []toolDefinition{{
		Type: "function",
		Function: toolFunctionSpec{
			Name:        toolName,
			Description: "提交本批所有平台/内容形式组合的内容变体",
			Parameters:  schema,
		},
	}}
	toolChoice := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name": toolName,
		},
	}
	return messages, tools, toolChoice, ownSystem == ""
}

func parseBatchItemsArgs(raw string) ([]BatchItemResult, bool) {
	clean := strings.TrimSpace(stripCodeFence(raw))
	var wrapper struct {
		Items []struct {
			Platform    string                   `json:"platform"`
			ContentForm string                   `json:"content_form"`
			Variants    []map[string]interface{} `json:"variants"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(clean), &wrapper); err == nil && len(wrapper.Items) > 0 {
		results := make([]BatchItemResult, 0, len(wrapper.Items))
		ok := false
		for _, it := range wrapper.Items {
			variants := buildVariantList(it.Variants)
			if len(variants) == 0 {
				continue
			}
			results = append(results, BatchItemResult{Platform: strings.TrimSpace(it.Platform), ContentForm: strings.TrimSpace(it.ContentForm), Variants: variants})
			ok = true
		}
		if ok {
			return results, true
		}
	}
	var list []map[string]interface{}
	if err := json.Unmarshal([]byte(clean), &list); err == nil && len(list) > 0 {
		results := make([]BatchItemResult, 0, len(list))
		for _, it := range list {
			variants := buildVariantList(toStringMapSlice(it["variants"]))
			if len(variants) == 0 {
				continue
			}
			results = append(results, BatchItemResult{
				Platform:    toString(it["platform"]),
				ContentForm: toString(it["content_form"]),
				Variants:    variants,
			})
		}
		if len(results) > 0 {
			return results, true
		}
	}
	return nil, false
}

var platformKeyAliases = map[string][]string{
	"xiaohongshu":    {"xiaohongshu", "xhs", "rednote", "小红书"},
	"douyin":         {"douyin", "dy", "抖音"},
	"wechat_mp":      {"wechat_mp", "weixin_mp", "wx_mp", "公众号", "微信公众号"},
	"wechat_video":   {"wechat_video", "weixin_video", "视频号", "微信视频号"},
	"wechat_moments": {"wechat_moments", "weixin_moments", "朋友圈"},
	"weibo":          {"weibo", "微博"},
}

var contentFormKeyAliases = map[string][]string{
	"post":           {"post", "图文", "推文", "笔记"},
	"script":         {"script", "短视频脚本", "脚本", "口播"},
	"snippet":        {"snippet", "短文案", "标题"},
	"live_script":    {"live_script", "直播脚本", "直播"},
	"product_detail": {"product_detail", "商品详情", "详情页"},
}

type keyAlias struct {
	code  string
	alias string
}

func buildSortedAliases(m map[string][]string) []keyAlias {
	list := make([]keyAlias, 0)
	for code, aliases := range m {
		for _, alias := range aliases {
			list = append(list, keyAlias{code: code, alias: strings.ToLower(alias)})
		}
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i].alias) > len(list[j].alias) })
	return list
}

var sortedPlatformAliases = buildSortedAliases(platformKeyAliases)
var sortedFormAliases = buildSortedAliases(contentFormKeyAliases)

func taskOrders(tasks []BatchTask) (platformOrder, formOrder []string) {
	seenP := map[string]bool{}
	seenF := map[string]bool{}
	for _, t := range tasks {
		if !seenP[t.Platform] {
			seenP[t.Platform] = true
			platformOrder = append(platformOrder, t.Platform)
		}
		if !seenF[t.ContentForm] {
			seenF[t.ContentForm] = true
			formOrder = append(formOrder, t.ContentForm)
		}
	}
	return
}

func parseOrderNumber(raw, prefix string) (int, bool) {
	s := strings.TrimSpace(raw)
	if len(s) < 2 || !strings.EqualFold(s[:1], prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(s[1:]))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func matchAlias(aliasList []keyAlias, r string) (string, bool) {
	for _, ka := range aliasList {
		if r == ka.alias {
			return ka.code, true
		}
	}
	for _, ka := range aliasList {
		if strings.Contains(r, ka.alias) {
			return ka.code, true
		}
	}
	return "", false
}

func resolvePlatformCode(raw string, order []string) (string, bool) {
	r := strings.ToLower(strings.TrimSpace(raw))
	if r == "" {
		return "", false
	}
	for _, code := range order {
		if code == r {
			return code, true
		}
	}
	if n, ok := parseOrderNumber(r, "p"); ok && n <= len(order) {
		return order[n-1], true
	}
	return matchAlias(sortedPlatformAliases, r)
}

func resolveContentFormCode(raw string, order []string) (string, bool) {
	r := strings.ToLower(strings.TrimSpace(raw))
	if r == "" {
		return "", false
	}
	for _, code := range order {
		if code == r {
			return code, true
		}
	}
	if n, ok := parseOrderNumber(r, "f"); ok && n <= len(order) {
		return order[n-1], true
	}
	return matchAlias(sortedFormAliases, r)
}

func toStringMapSlice(v interface{}) []map[string]interface{} {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// GenerateContentBatchStream 跨平台合并分批生成：把 (平台 × 内容形式) 任务自动分批，
// 每批一次模型调用（批任务 Prompt），产出后逐篇推送 done 事件。
func GenerateContentBatchStream(r BatchRequest) (<-chan AIStreamEvent, error) {
	events := make(chan AIStreamEvent, 16)
	if r.VersionNum <= 0 {
		r.VersionNum = 1
	}
	if r.VersionNum > 3 {
		r.VersionNum = 3
	}

	var tasks []BatchTask
	for _, p := range r.Platforms {
		for _, f := range r.ContentForms {
			tasks = append(tasks, BatchTask{Platform: p, ContentForm: f})
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("无生成任务")
	}

	apiKey, baseURL, model, planName, _, apiFormat, err := ResolveModelConfig(r.PlanID, r.ModelID)
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
		if len(r.Files) > 0 {
			normalized, fileErrors := normalizeContentFiles(r.Files)
			r.Files = normalized
			for _, fe := range fileErrors {
				events <- AIStreamEvent{Event: "log", Level: "warn", Message: fmt.Sprintf("素材转 Base64 失败，已跳过：%s", fe)}
			}
		}
		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}
		if r.CampaignName != "" {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("已注入活动「%s」创作背景", r.CampaignName)}
		}
		if len(r.Files) > 0 {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("附带 %d 个素材（多模态分析）", len(r.Files))}
		}
		jsonOnly := isJSONOnlyContentModel(baseURL)
		if jsonOnly {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: "当前端点不支持工具调用，批任务以纯 JSON 模式输出"}
		}
		var shared []BatchTask
		dedicated := map[string][]BatchTask{}
		for _, t := range tasks {
			if loadPromptTemplate(t.Platform) != "" {
				dedicated[t.Platform] = append(dedicated[t.Platform], t)
			} else {
				shared = append(shared, t)
			}
		}
		var batches [][]BatchTask
		for _, p := range r.Platforms {
			if v := dedicated[p]; len(v) > 0 {
				batches = append(batches, v)
			}
		}
		for start := 0; start < len(shared); start += batchMaxTaskPerCall {
			end := start + batchMaxTaskPerCall
			if end > len(shared) {
				end = len(shared)
			}
			batches = append(batches, shared[start:end])
		}
		batchTotal := len(batches)
		if batchTotal > 1 {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("共 %d 个组合任务，已合并为 %d 批执行", len(tasks), batchTotal)}
		}

		batchIndex := 0
		for _, batch := range batches {
			runBatch := func(tasks []BatchTask, idx int) {
				events <- AIStreamEvent{Event: "log", Level: "req", Message: fmt.Sprintf("开始生成第 %d/%d 批（%d 个组合任务）…", idx+1, batchTotal, len(tasks))}
				messages, tools, toolChoice, _ := buildBatchMessages(r, tasks)
				requestTools := tools
				requestToolChoice := toolChoice
				if jsonOnly {
					requestTools = nil
					requestToolChoice = nil
				}
				var resp *http.Response
				if apiFormat == "openai_responses" {
					resp, err = callResponsesStreamWithTools(apiKey, baseURL, model, messages, requestTools, requestToolChoice, 8000)
				} else {
					resp, err = callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, requestTools, requestToolChoice, 8000)
				}
				if err != nil {
					events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s", planName, err)}
					return
				}
				if resp.StatusCode == http.StatusBadRequest {
					bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
					resp.Body.Close()
					lower := strings.ToLower(string(bodyBytes))
					if strings.Contains(lower, "tool_choice") || strings.Contains(lower, "tool choice") {
						events <- AIStreamEvent{Event: "log", Level: "warn", Message: "当前模型不支持强制工具调用，已自动降级为自动模式"}
						if apiFormat == "openai_responses" {
							resp, err = callResponsesStreamWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
						} else {
							resp, err = callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
						}
						if err != nil {
							events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s", planName, err)}
							return
						}
					} else {
						events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s", planName, resp.StatusCode, string(bodyBytes))}
						return
					}
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
					events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s", planName, resp.StatusCode, string(b))}
					return
				}

				fullContent := ""
				toolName := ""
				toolArgs := ""
				scanner := bufio.NewScanner(resp.Body)
				if apiFormat == "openai_responses" {
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if !strings.HasPrefix(line, "data:") {
							continue
						}
						data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
						if data == "[DONE]" {
							break
						}
						var evt struct {
							Type  string `json:"type"`
							Delta string `json:"delta"`
							Item  struct {
								Type string `json:"type"`
								Name string `json:"name"`
							} `json:"item"`
						}
						if err := json.Unmarshal([]byte(data), &evt); err != nil {
							continue
						}
						switch evt.Type {
						case "response.output_item.added":
							if evt.Item.Type == "function_call" {
								toolName = evt.Item.Name
							}
						case "response.function_call_arguments.delta":
							toolArgs += evt.Delta
						case "response.output_text.delta":
							if evt.Delta != "" {
								fullContent += evt.Delta
								events <- AIStreamEvent{Event: "chunk", Text: evt.Delta}
							}
						}
					}
				} else {
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
							if tc.Function.Name != "" {
								toolName = tc.Function.Name
							}
							if tc.Function.Arguments != "" {
								toolArgs += tc.Function.Arguments
							}
						}
					}
				}
				if err := scanner.Err(); err != nil {
					events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("读取模型流失败：%s", err)}
					return
				}

				var results []BatchItemResult
				if toolName == "submit_batch_variants" {
					if list, ok := parseBatchItemsArgs(toolArgs); ok {
						results = list
						events <- AIStreamEvent{Event: "log", Level: "ok", Message: fmt.Sprintf("已解析批任务工具结果（%d 个组合）", len(list))}
					}
				}
				if len(results) == 0 {
					if list, ok := parseBatchItemsArgs(fullContent); ok {
						results = list
						events <- AIStreamEvent{Event: "log", Level: "ok", Message: fmt.Sprintf("已从文本内容回退解析批任务 JSON（%d 个组合）", len(list))}
					}
				}
				if len(results) == 0 {
					events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("第 %d 批结果格式异常，无法解析。", idx+1)}
					return
				}
				platformOrder, formOrder := taskOrders(tasks)
				for _, item := range results {
					pk, pok := resolvePlatformCode(item.Platform, platformOrder)
					fk, fok := resolveContentFormCode(item.ContentForm, formOrder)
					if !pok || !fok {
						continue
					}
					matched := false
					for _, t := range tasks {
						if t.Platform == pk && t.ContentForm == fk {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
					for i := range item.Variants {
						v := item.Variants[i]
						score := ScoreVariant(v, pk)
						events <- AIStreamEvent{
							Event:        "done",
							Platform:     pk,
							ContentForm:  fk,
							Variant:      &v,
							VariantIndex: i,
							Score:        &score,
							BatchIndex:   idx,
							BatchTotal:   batchTotal,
							Model:        model,
							PlanName:     planName,
						}
					}
				}
			}
			runBatch(batch, batchIndex)
			batchIndex++
		}
		events <- AIStreamEvent{Event: "log", Level: "ok", Message: "全部批次生成完成"}
	})
	return events, nil
}
