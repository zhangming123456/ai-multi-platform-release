package services

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

var builtinInspectionItems = []struct {
	Name     string
	Category string
	MaxScore int
}{
	{Name: "环境卫生", Category: "卫生", MaxScore: 10},
	{Name: "商品陈列", Category: "陈列", MaxScore: 10},
	{Name: "服务规范", Category: "服务", MaxScore: 10},
	{Name: "消防安全", Category: "安全", MaxScore: 10},
	{Name: "设备设施", Category: "设施", MaxScore: 10},
}

func SeedInspectionItems() error {
	o := GetOrm()
	count, err := o.QueryTable(new(models.InspectionItem)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for i, item := range builtinInspectionItems {
		entry := &models.InspectionItem{
			ID:        uuid.NewString(),
			Name:      item.Name,
			Category:  item.Category,
			MaxScore:  item.MaxScore,
			SortOrder: i,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if _, err := o.Insert(entry); err != nil {
			return err
		}
	}
	return nil
}

func GetActiveInspectionItems() ([]models.InspectionItem, error) {
	var items []models.InspectionItem
	_, err := GetOrm().QueryTable(new(models.InspectionItem)).
		Filter("is_active", true).
		OrderBy("sort_order").
		All(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

type InspectionAIResult struct {
	Scores           map[string]float64         `json:"scores"`
	Comments         map[string]string          `json:"comments"`
	Issues           string                     `json:"issues"`
	Suggestion       string                     `json:"suggestion"`
	Summary          string                     `json:"summary"`
	HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
	MainProblems     []InspectionProblem        `json:"main_problems"`
	PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
	BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
}

// InspectionProblem 结构化总结中的问题项（高危问题/主要问题）。
type InspectionProblem struct {
	ItemID   string `json:"item_id"`
	ItemName string `json:"item_name"`
	Level    string `json:"level"`
	Desc     string `json:"desc"`
}

// InspectionSuggestionItem 结构化总结中的建议项（优先整改建议/运营优化建议）。
type InspectionSuggestionItem struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

// InspectionAISummary 巡店 AI 结构化总结，持久化到巡店记录的 ai_summary 字段。
type InspectionAISummary struct {
	Summary          string                     `json:"summary"`
	HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
	MainProblems     []InspectionProblem        `json:"main_problems"`
	PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
	BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
}

// ParseInspectionAISummary 解析巡店记录中保存的 AI 结构化总结 JSON。
func ParseInspectionAISummary(raw string) InspectionAISummary {
	var s InspectionAISummary
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &s)
	}
	return s
}

// MarshalInspectionAISummary 序列化 AI 结构化总结为 JSON 字符串。
func MarshalInspectionAISummary(s InspectionAISummary) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type InspectionAIItem struct {
	ID             string
	Name           string
	Standard       string
	StandardImages []string
	ScoreType      string
	MaxScore       int
	ScoreOptions   []models.ScoreOption
}

// InspectionSkill 前端组装的检查项技能规范，用于向 AI 明确定义巡店标准与评分规则。
type InspectionSkill struct {
	ItemID         string               `json:"item_id"`
	Name           string               `json:"name"`
	Category       string               `json:"category"`
	Standard       string               `json:"standard"`
	StandardImages []string             `json:"standard_images"`
	ScoreType      string               `json:"score_type"`
	MaxScore       int                  `json:"max_score"`
	ScoreOptions   []models.ScoreOption `json:"score_options"`
	RequireRemark  bool                 `json:"require_remark"`
	RequirePhoto   bool                 `json:"require_photo"`
}

// InspectionSkillSpec 巡店 AI 技能规范：检查项标准 + 返回格式 schema。
type InspectionSkillSpec struct {
	Skills         []InspectionSkill      `json:"skills"`
	ResponseSchema map[string]interface{} `json:"response_schema"`
}

// defaultInspectionResponseSchema 默认返回格式 schema，规范 AI 输出结构。
func defaultInspectionResponseSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"scores": map[string]interface{}{
				"type":        "array",
				"description": "仅包含与巡店关键词/图片有关联的检查项评分，无关项不返回",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_id": map[string]interface{}{"type": "string", "description": "检查项 ID"},
						"score":   map[string]interface{}{"type": "number", "description": "得分，必须在允许的分值范围内"},
						"comment": map[string]interface{}{"type": "string", "description": "该检查项的反馈问题说明"},
					},
					"required": []string{"item_id", "score", "comment"},
				},
			},
			"issues":     map[string]interface{}{"type": "string", "description": "问题与备注：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项"},
			"suggestion": map[string]interface{}{"type": "string", "description": "AI 整改建议：针对每个问题给出具体、可执行的整改措施与提升建议"},
			"summary":    map[string]interface{}{"type": "string", "description": "整体一句话概括本次巡店情况，30-60字"},
			"high_risk_problems": map[string]interface{}{
				"type":        "array",
				"description": "高危风险问题列表（如消防、食安等需立即处理的问题）",
				"items":       problemItemSchema(),
			},
			"main_problems": map[string]interface{}{
				"type":        "array",
				"description": "主要问题汇总列表（除高危外需要整改的问题）",
				"items":       problemItemSchema(),
			},
			"priority_suggest": map[string]interface{}{
				"type":        "array",
				"description": "优先整改建议列表（3条，针对最严重问题）",
				"items":       suggestionItemSchema(),
			},
			"business_suggest": map[string]interface{}{
				"type":        "array",
				"description": "门店运营优化建议列表",
				"items":       suggestionItemSchema(),
			},
		},
		"required": []string{"scores", "issues", "suggestion", "summary", "high_risk_problems", "main_problems", "priority_suggest", "business_suggest"},
	}
}

func problemItemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"item_id":   map[string]interface{}{"type": "string", "description": "关联的检查项 ID，无法关联时可为空字符串"},
			"item_name": map[string]interface{}{"type": "string", "description": "关联的检查项名称，无法关联时可为空字符串"},
			"level":     map[string]interface{}{"type": "string", "description": "严重程度：高/中/低"},
			"desc":      map[string]interface{}{"type": "string", "description": "问题描述，说明具体现象与位置"},
		},
		"required": []string{"item_id", "item_name", "level", "desc"},
	}
}

func suggestionItemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{"type": "string", "description": "建议标题，简短明确"},
			"desc":  map[string]interface{}{"type": "string", "description": "建议说明，具体可执行"},
		},
		"required": []string{"title", "desc"},
	}
}

// buildInspectionSkillsPrompt 将检查项技能规范拼装为结构化文本，作为 AI 的巡店标准与规范。
func buildInspectionSkillsPrompt(skills []InspectionSkill) string {
	var b strings.Builder
	b.WriteString("【巡店检查项标准与评分规范】\n")
	b.WriteString("请严格按照以下每个检查项的 ID、标准、评分规则进行客观评分，score 必须落在该检查项允许的分值范围内。\n\n")
	for i, s := range skills {
		b.WriteString(fmt.Sprintf("%d. item_id=%s\n", i+1, s.ItemID))
		b.WriteString(fmt.Sprintf("   名称：%s\n", s.Name))
		if s.Category != "" {
			b.WriteString(fmt.Sprintf("   分类：%s\n", s.Category))
		}
		if s.Standard != "" {
			b.WriteString(fmt.Sprintf("   检查标准：%s\n", s.Standard))
		}
		if len(s.StandardImages) > 0 {
			b.WriteString(fmt.Sprintf("   标准图参考：%s\n", strings.Join(s.StandardImages, ", ")))
		}
		if s.ScoreType == "pass_fail" {
			labels := make([]string, 0, len(s.ScoreOptions))
			for _, opt := range s.ScoreOptions {
				labels = append(labels, fmt.Sprintf("%s(%.1f分)", opt.Label, opt.Score))
			}
			b.WriteString(fmt.Sprintf("   评分方式：选项评分，可选项：%s\n", strings.Join(labels, " / ")))
		} else {
			allowed := make([]string, 0, len(s.ScoreOptions))
			for _, opt := range s.ScoreOptions {
				allowed = append(allowed, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", opt.Score), "0"), "."))
			}
			if len(allowed) > 0 {
				b.WriteString(fmt.Sprintf("   评分方式：分值评分，允许分值：%s（满分 %d 分）\n", strings.Join(allowed, " / "), s.MaxScore))
			} else {
				b.WriteString(fmt.Sprintf("   评分方式：分值评分，0 ~ %d 分\n", s.MaxScore))
			}
		}
		if s.RequireRemark {
			b.WriteString("   反馈问题：必填\n")
		}
		if s.RequirePhoto {
			b.WriteString("   巡店图片：必填\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// buildInspectionInputHints 根据巡店图片与关键词生成对应的 prompt 提示段。
// 返回 (photoHint, keywordsHint)，二者均可为空字符串。
func buildInspectionInputHints(photos []UploadedFile, keywords string) (string, string) {
	photoHint := ""
	switch {
	case len(photos) > 0:
		photoHint = "\n请优先基于上传的巡店照片进行客观判断，照片中可见的问题要在问题描述中指出。"
	case strings.TrimSpace(keywords) != "":
		photoHint = "\n本次未提供巡店照片，请基于巡店关键词信息并结合通用门店运营规范给出检查结果。"
	default:
		photoHint = "\n本次未提供照片，请基于通用门店运营规范给出建议性检查结果。"
	}
	keywordsHint := ""
	if kw := strings.TrimSpace(keywords); kw != "" {
		keywordsHint = fmt.Sprintf("\n巡店补充信息（关键词/现场描述）：%s\n请结合以上信息进行客观判断——仅对关键词/图片明确涉及的检查项进行评分与分析，无关的检查项不需要返回评分。", kw)
	}
	return photoHint, keywordsHint
}

// loadImageAsUploadedFile 将图片 URL（支持 http/https 完整 URL 或 /uploads/ 相对路径）加载为 UploadedFile。
// 相对路径会读取本地上传目录并转为 base64 data URL，便于 vision 模型识别。
func loadImageAsUploadedFile(imageURL string) (*UploadedFile, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil, fmt.Errorf("图片 URL 为空")
	}

	// 完整 URL：由后端下载图片内容并转为 base64，保证模型一定能读取
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(imageURL)
		if err != nil {
			return nil, fmt.Errorf("下载图片失败 %s: %w", imageURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("下载图片失败 %s: 状态码 %d", imageURL, resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取图片内容失败 %s: %w", imageURL, err)
		}
		mimeType := resp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(imageURL)))
		}
		if mimeType == "" {
			mimeType = "image/png"
		}
		return &UploadedFile{
			Data:     base64.StdEncoding.EncodeToString(data),
			MimeType: mimeType,
			URL:      imageURL,
		}, nil
	}

	// 解析 /uploads/ 相对路径
	u, err := url.Parse(imageURL)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(u.Path)
	if !strings.HasPrefix(cleanPath, "/uploads/") {
		return nil, fmt.Errorf("不支持的图片路径: %s", imageURL)
	}
	filename := strings.TrimPrefix(cleanPath, "/uploads/")
	filename = strings.TrimPrefix(filename, "/")
	localPath := filepath.Join(GetUploadDir(), filename)

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("读取上传图片失败 %s: %w", localPath, err)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "image/png"
	}

	return &UploadedFile{
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
		URL:      imageURL,
	}, nil
}

// collectStandardImages 从检查项技能规范中提取所有标准图并加载为 UploadedFile。
func collectStandardImages(skills []InspectionSkill) []UploadedFile {
	var images []UploadedFile
	seen := make(map[string]bool)
	for _, s := range skills {
		for _, img := range s.StandardImages {
			if img == "" || seen[img] {
				continue
			}
			seen[img] = true
			if f, err := loadImageAsUploadedFile(img); err == nil {
				images = append(images, *f)
			}
		}
	}
	return images
}

// buildInspectionVisionContent 构建包含标准图与现场图的 vision content。
// 标准图在前并附带说明文本，现场图在后并附带说明文本，便于模型区分对比。
func buildInspectionVisionContent(prompt string, standardImages []UploadedFile, photos []UploadedFile) []interface{} {
	parts := make([]interface{}, 0, 1+len(standardImages)+len(photos)+2)
	parts = append(parts, map[string]interface{}{"type": "text", "text": prompt})

	if len(standardImages) > 0 {
		parts = append(parts, map[string]interface{}{"type": "text", "text": fmt.Sprintf("【检查标准参照图】共 %d 张，请作为评分基准：", len(standardImages))})
		for _, f := range standardImages {
			parts = append(parts, visionPartFromFile(f))
		}
	}

	if len(photos) > 0 {
		parts = append(parts, map[string]interface{}{"type": "text", "text": fmt.Sprintf("【巡店现场图】共 %d 张，请与标准图进行对比分析：", len(photos))})
		for _, f := range photos {
			parts = append(parts, visionPartFromFile(f))
		}
	}

	return parts
}

// visionPartFromFile 将 UploadedFile 转换为 vision API 所需的 image_url / video_url 部分。
// 所有图片统一由后端转为 base64 data URL 再交给模型：本地文件路径 / http(s) 链接都先下载/读取转 base64；
// 前端直接传来的 base64（Data 非空）则原样使用。
func visionPartFromFile(f UploadedFile) interface{} {
	imgURL := strings.TrimSpace(f.URL)
	if imgURL == "" && f.Data == "" {
		imgURL = f.Data
	}

	// 优先统一加载为 base64：http(s) 链接、/uploads/ 相对路径
	if strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://") || strings.HasPrefix(imgURL, "/uploads/") {
		if loaded, err := loadImageAsUploadedFile(imgURL); err == nil && loaded.Data != "" {
			dataURL := "data:" + loaded.MimeType + ";base64," + loaded.Data
			return map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]interface{}{"url": dataURL},
			}
		}
	}

	if f.Data == "" {
		return map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]interface{}{"url": imgURL},
		}
	}

	dataURL := "data:" + f.MimeType + ";base64," + f.Data
	return map[string]interface{}{
		"type":      "image_url",
		"image_url": map[string]interface{}{"url": dataURL},
	}
}

// AnalyzeInspection 分析门店巡店情况并生成检查报告，返回每个检查项得分、问题与建议。
// keywords 为巡店关键词/现场描述补充信息（可为空）；photos 为巡店图片（可选，可为空）。
// 当 skillSpec 非空时，使用前端组装的检查项技能规范，并通过 function calling 规范返回格式。
func AnalyzeInspection(storeName string, items []InspectionAIItem, photos []UploadedFile, keywords, planID, modelID string, skillSpec *InspectionSkillSpec) (*InspectionAIResult, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(planID, modelID)
	if err != nil {
		return nil, err
	}

	storeDesc := storeName
	if storeDesc == "" {
		storeDesc = "该门店"
	}

	photoHint, keywordsHint := buildInspectionInputHints(photos, keywords)

	// 分支：前端传入了 skills 技能规范，使用 function calling 严格规范返回格式
	if skillSpec != nil && len(skillSpec.Skills) > 0 {
		standardImages := collectStandardImages(skillSpec.Skills)
		hasStandardImages := len(standardImages) > 0
		hasPhotos := len(photos) > 0
		skillsPrompt := buildInspectionSkillsPrompt(skillSpec.Skills)
		imageHint := ""
		if hasStandardImages && hasPhotos {
			imageHint = "\n本次已附带检查标准参照图与巡店现场图，请结合标准图与现场图进行对比分析，重点关注现场与标准的差异。"
		} else if hasStandardImages {
			imageHint = "\n本次已附带检查标准参照图，请参照标准图进行评估。"
		} else if hasPhotos {
			imageHint = "\n本次已附带巡店现场图，请结合现场图进行评估。"
		}
		prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对「%s」进行巡店检查。
%s
请基于以上检查项标准与巡店信息，客观评估每个检查项的达标情况，重点输出：
1. scores：每个检查项的得分（无关项不返回）；
2. issues（问题与备注）：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项；
3. suggestion（AI 整改建议）：针对每个问题给出具体、可执行的整改措施与提升建议；
4. 巡店总结（summary / high_risk_problems / main_problems / priority_suggest / business_suggest）。
%s
%s
%s

请直接调用 submit_inspection_result 工具提交结果，不要输出任何解释性文字、markdown 代码块或其他文本。scores 仅包含与巡店关键词/图片有关联的检查项，每项的 item_id 与上面给出的 item_id 一致，score 必须落在该检查项允许的分值范围内；无关的检查项不要返回评分；issues 与 suggestion 必须详尽充实；summary 为 30-60 字整体概括；high_risk_problems 提取所有高危风险问题（消防、食安等需立即处理）；main_problems 汇总主要问题；priority_suggest 给出 3 条优先整改建议；business_suggest 给出门店运营优化建议。high_risk_problems 与 main_problems 中每项的 item_id 必须与对应检查项的 item_id 一致（优先填写 item_id），无法关联到具体检查项时 item_id 留空。`, storeDesc, skillsPrompt, keywordsHint, photoHint, imageHint)

		messages := []chatMessage{{
			Role:    "system",
			Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。必须且只能调用 submit_inspection_result 工具返回结构化结果，禁止输出工具调用以外的任何解释性文字、markdown 代码块或普通文本。",
		}}
		if supportsVision && (hasStandardImages || hasPhotos) {
			messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, photos)})
		} else {
			messages = append(messages, chatMessage{Role: "user", Content: prompt})
		}

		schema := skillSpec.ResponseSchema
		if schema == nil {
			schema = defaultInspectionResponseSchema()
		}
		tools := []toolDefinition{{
			Type: "function",
			Function: toolFunctionSpec{
				Name:        "submit_inspection_result",
				Description: "提交巡店检查评分结果，包括每个检查项的得分、反馈问题、整体问题与整改建议",
				Parameters:  schema,
			},
		}}
		// 强制调用 submit_inspection_result，避免模型输出普通文本或被截断。
		toolChoice := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": "submit_inspection_result",
			},
		}
		const inspectionMaxTokens = 12000

		var resp *http.Response
		if apiFormat == "openai_responses" {
			resp, err = callResponsesWithTools(apiKey, baseURL, model, messages, tools, toolChoice, inspectionMaxTokens)
		} else {
			resp, err = callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, toolChoice, inspectionMaxTokens)
		}
		if err != nil {
			return nil, aiCallError(planName, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
		}

		var toolArgs string
		var content string
		if apiFormat == "openai_responses" {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
			respContent, toolName, args := extractResponsesResult(respBody)
			if toolName != "" {
				toolArgs = args
				content = ""
			} else {
				content = respContent
			}
		} else {
			var raw struct {
				Choices []struct {
					Message struct {
						Content   string           `json:"content"`
						ToolCalls []toolCallResult `json:"tool_calls"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
				return nil, aiCallError(planName, err)
			}
			if len(raw.Choices) == 0 {
				return nil, aiCallError(planName, fmt.Errorf("模型无返回内容"))
			}
			choice := raw.Choices[0]
			if len(choice.Message.ToolCalls) > 0 {
				toolArgs = choice.Message.ToolCalls[0].Function.Arguments
			} else {
				content = choice.Message.Content
			}
		}

		// 优先解析工具调用参数
		if strings.TrimSpace(toolArgs) != "" {
			args := strings.TrimSpace(toolArgs)
			if parsed, perr := parseInspectionToolArgs(args); perr == nil {
				return parsed, nil
			}
		}
		// 回退：解析 content 为 JSON
		content = strings.TrimSpace(content)
		if strings.HasPrefix(content, "```") {
			parts := strings.Split(content, "```")
			if len(parts) > 1 {
				content = strings.TrimSpace(parts[1])
				content = strings.TrimPrefix(content, "json")
				content = strings.TrimSpace(content)
			}
		}
		parsed, perr := parseInspectionContentJSON(content)
		if perr != nil {
			available, aerr := getAvailablePlans()
			if aerr != nil {
				return nil, aerr
			}
			return nil, &AIGenerationError{
				Message:        fmt.Sprintf("模型「%s」返回的巡店报告格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName),
				AvailablePlans: available,
			}
		}
		return parsed, nil
	}

	// 兼容分支：未传 skills，维持原有 prompt + JSON 返回逻辑
	standardImages := make([]UploadedFile, 0)
	seenStandard := make(map[string]bool)
	itemLines := make([]string, 0, len(items))
	for _, it := range items {
		scoreDesc := fmt.Sprintf("满分 %d 分", it.MaxScore)
		if len(it.ScoreOptions) > 0 {
			labels := make([]string, 0, len(it.ScoreOptions))
			allowed := make([]string, 0, len(it.ScoreOptions))
			for _, opt := range it.ScoreOptions {
				labels = append(labels, fmt.Sprintf("%s(%.1f分)", opt.Label, opt.Score))
				allowed = append(allowed, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", opt.Score), "0"), "."))
			}
			if it.ScoreType == "pass_fail" {
				scoreDesc = strings.Join(labels, " / ") + "，请从以上选项中选择"
			} else {
				scoreDesc = "分值评分，可选分值：" + strings.Join(allowed, " / ") + " 分（满分 " + fmt.Sprintf("%d", it.MaxScore) + " 分）"
			}
		}
		line := fmt.Sprintf("  - %s（%s）", it.Name, scoreDesc)
		if it.Standard != "" {
			line += fmt.Sprintf("；检查标准：%s", it.Standard)
		}
		if len(it.StandardImages) > 0 {
			line += "；已附带标准参照图"
			for _, img := range it.StandardImages {
				if img == "" || seenStandard[img] {
					continue
				}
				seenStandard[img] = true
				if f, err := loadImageAsUploadedFile(img); err == nil {
					standardImages = append(standardImages, *f)
				}
			}
		}
		itemLines = append(itemLines, line)
	}
	hasStandardImages := len(standardImages) > 0
	hasPhotos := len(photos) > 0
	imageHint := ""
	if hasStandardImages && hasPhotos {
		imageHint = "\n本次已附带检查标准参照图与巡店现场图，请结合标准图与现场图进行对比分析，重点关注现场与标准的差异。"
	} else if hasStandardImages {
		imageHint = "\n本次已附带检查标准参照图，请参照标准图进行评估。"
	} else if hasPhotos {
		imageHint = "\n本次已附带巡店现场图，请结合现场图进行评估。"
	}

	prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对「%s」进行巡店检查。
检查项及评分标准如下：
%s

请基于以上检查项标准与巡店信息，客观评估每个检查项的达标情况，重点输出：
1. scores：每个检查项的得分；
2. issues（问题与备注）：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项；
3. suggestion（AI 整改建议）：针对每个问题给出具体、可执行的整改措施与提升建议；
4. 巡店总结（summary / high_risk_problems / main_problems / priority_suggest / business_suggest）。
%s
%s
%s

请严格以 JSON 格式返回（不要输出其他内容）：
{
  "scores": {"检查项名称": 得分（仅包含有针对性分析的项）, ...},
  "issues": "问题与备注，按检查项分条列出",
  "suggestion": "AI 整改建议，具体可执行",
  "summary": "整体一句话概括本次巡店情况（30-60字）",
  "high_risk_problems": [{"item_id":"关联检查项ID（无法关联留空）","item_name":"关联检查项名称","level":"高/中/低","desc":"问题描述"}],
  "main_problems": [{"item_id":"关联检查项ID（无法关联留空）","item_name":"关联检查项名称","level":"高/中/低","desc":"问题描述"}],
  "priority_suggest": [{"title":"建议标题","desc":"建议说明"}],
  "business_suggest": [{"title":"建议标题","desc":"建议说明"}]
}`, storeDesc, strings.Join(itemLines, "\n"), keywordsHint, photoHint, imageHint)

	messages := []chatMessage{{
		Role:    "system",
		Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。必须生成一份完整的巡店分析报告，包含每个检查项的评估结果（含标准图与现场图对比分析）。",
	}}
	if supportsVision && (hasStandardImages || hasPhotos) {
		messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, photos)})
	} else {
		messages = append(messages, chatMessage{Role: "user", Content: prompt})
	}

	var resp *http.Response
	if apiFormat == "openai_responses" {
		resp, err = callResponses(apiKey, baseURL, model, messages, false)
	} else {
		resp, err = callChatCompletions(apiKey, baseURL, model, messages, false)
	}
	if err != nil {
		return nil, aiCallError(planName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, aiCallError(planName, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	var content string
	if apiFormat == "openai_responses" {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		content, _, _ = extractResponsesResult(respBody)
	} else {
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
		content = result.Choices[0].Message.Content
	}
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) > 1 {
			content = strings.TrimSpace(parts[1])
			content = strings.TrimPrefix(content, "json")
			content = strings.TrimSpace(content)
		}
	}

	var parsed struct {
		Scores           map[string]float64         `json:"scores"`
		Issues           string                     `json:"issues"`
		Suggestion       string                     `json:"suggestion"`
		Summary          string                     `json:"summary"`
		HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
		MainProblems     []InspectionProblem        `json:"main_problems"`
		PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
		BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		available, aerr := getAvailablePlans()
		if aerr != nil {
			return nil, aerr
		}
		return nil, &AIGenerationError{
			Message:        fmt.Sprintf("模型「%s」返回的巡店报告格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName),
			AvailablePlans: available,
		}
	}
	if parsed.Scores == nil {
		parsed.Scores = map[string]float64{}
	}
	return &InspectionAIResult{
		Scores:           parsed.Scores,
		Issues:           parsed.Issues,
		Suggestion:       parsed.Suggestion,
		Summary:          parsed.Summary,
		HighRiskProblems: parsed.HighRiskProblems,
		MainProblems:     parsed.MainProblems,
		PrioritySuggest:  parsed.PrioritySuggest,
		BusinessSuggest:  parsed.BusinessSuggest,
	}, nil
}

// parseInspectionToolArgs 解析 submit_inspection_result 工具调用的 arguments JSON。
func parseInspectionToolArgs(args string) (*InspectionAIResult, error) {
	if args == "" {
		return nil, fmt.Errorf("empty tool arguments")
	}
	args = stripCodeFence(args)
	var obj struct {
		Scores []struct {
			ItemID  string  `json:"item_id"`
			Score   float64 `json:"score"`
			Comment string  `json:"comment"`
		} `json:"scores"`
		Issues           string                     `json:"issues"`
		Suggestion       string                     `json:"suggestion"`
		Summary          string                     `json:"summary"`
		HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
		MainProblems     []InspectionProblem        `json:"main_problems"`
		PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
		BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
	}
	if err := json.Unmarshal([]byte(args), &obj); err != nil {
		return nil, err
	}
	scores := make(map[string]float64, len(obj.Scores))
	comments := make(map[string]string, len(obj.Scores))
	for _, s := range obj.Scores {
		if s.ItemID == "" {
			continue
		}
		scores[s.ItemID] = s.Score
		comments[s.ItemID] = s.Comment
	}
	return &InspectionAIResult{
		Scores:           scores,
		Comments:         comments,
		Issues:           obj.Issues,
		Suggestion:       obj.Suggestion,
		Summary:          obj.Summary,
		HighRiskProblems: obj.HighRiskProblems,
		MainProblems:     obj.MainProblems,
		PrioritySuggest:  obj.PrioritySuggest,
		BusinessSuggest:  obj.BusinessSuggest,
	}, nil
}

// parseInspectionContentJSON 解析回退场景下的 content JSON（兼容对象型 scores 与数组型 scores）。
func parseInspectionContentJSON(content string) (*InspectionAIResult, error) {
	if content == "" {
		return nil, fmt.Errorf("empty content")
	}
	content = stripCodeFence(content)
	// 尝试数组型 scores
	var arr struct {
		Scores []struct {
			ItemID  string  `json:"item_id"`
			Score   float64 `json:"score"`
			Comment string  `json:"comment"`
		} `json:"scores"`
		Issues           string                     `json:"issues"`
		Suggestion       string                     `json:"suggestion"`
		Summary          string                     `json:"summary"`
		HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
		MainProblems     []InspectionProblem        `json:"main_problems"`
		PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
		BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
	}
	if err := json.Unmarshal([]byte(content), &arr); err == nil && len(arr.Scores) > 0 {
		scores := make(map[string]float64, len(arr.Scores))
		comments := make(map[string]string, len(arr.Scores))
		for _, s := range arr.Scores {
			if s.ItemID == "" {
				continue
			}
			scores[s.ItemID] = s.Score
			comments[s.ItemID] = s.Comment
		}
		return &InspectionAIResult{
			Scores:           scores,
			Comments:         comments,
			Issues:           arr.Issues,
			Suggestion:       arr.Suggestion,
			Summary:          arr.Summary,
			HighRiskProblems: arr.HighRiskProblems,
			MainProblems:     arr.MainProblems,
			PrioritySuggest:  arr.PrioritySuggest,
			BusinessSuggest:  arr.BusinessSuggest,
		}, nil
	}
	// 回退到对象型 scores
	var obj struct {
		Scores           map[string]float64         `json:"scores"`
		Issues           string                     `json:"issues"`
		Suggestion       string                     `json:"suggestion"`
		Summary          string                     `json:"summary"`
		HighRiskProblems []InspectionProblem        `json:"high_risk_problems"`
		MainProblems     []InspectionProblem        `json:"main_problems"`
		PrioritySuggest  []InspectionSuggestionItem `json:"priority_suggest"`
		BusinessSuggest  []InspectionSuggestionItem `json:"business_suggest"`
	}
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return nil, err
	}
	if obj.Scores == nil {
		obj.Scores = map[string]float64{}
	}
	return &InspectionAIResult{
		Scores:           obj.Scores,
		Issues:           obj.Issues,
		Suggestion:       obj.Suggestion,
		Summary:          obj.Summary,
		HighRiskProblems: obj.HighRiskProblems,
		MainProblems:     obj.MainProblems,
		PrioritySuggest:  obj.PrioritySuggest,
		BusinessSuggest:  obj.BusinessSuggest,
	}, nil
}

// AnalyzeInspectionStream 流式分析门店巡店情况并生成检查报告，通过 channel 发送事件。
// keywords 为巡店关键词/现场描述补充信息（可为空）；photos 为巡店图片（可选，可为空）。
// 事件类型：log / chunk / result / error。
func AnalyzeInspectionStream(storeName string, items []InspectionAIItem, photos []UploadedFile, keywords, planID, modelID string, skillSpec *InspectionSkillSpec) (<-chan AIStreamEvent, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(planID, modelID)
	if err != nil {
		return nil, err
	}

	events := make(chan AIStreamEvent, 32)
	go func() {
		defer close(events)

		storeDesc := storeName
		if storeDesc == "" {
			storeDesc = "该门店"
		}
		photoHint, keywordsHint := buildInspectionInputHints(photos, keywords)

		var messages []chatMessage
		var schema map[string]interface{}
		var tools []toolDefinition

		if skillSpec != nil && len(skillSpec.Skills) > 0 {
			standardImages := collectStandardImages(skillSpec.Skills)
			hasStandardImages := len(standardImages) > 0
			hasPhotos := len(photos) > 0
			skillsPrompt := buildInspectionSkillsPrompt(skillSpec.Skills)
			imageHint := ""
			if hasStandardImages && hasPhotos {
				imageHint = "\n本次已附带检查标准参照图与巡店现场图，请结合标准图与现场图进行对比分析，重点关注现场与标准的差异。"
			} else if hasStandardImages {
				imageHint = "\n本次已附带检查标准参照图，请参照标准图进行评估。"
			} else if hasPhotos {
				imageHint = "\n本次已附带巡店现场图，请结合现场图进行评估。"
			}
			prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对「%s」进行巡店检查。
%s
请基于以上检查项标准与巡店信息，客观评估每个检查项的达标情况，重点输出：
1. scores：每个检查项的得分（无关项不返回）；
2. issues（问题与备注）：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项；
3. suggestion（AI 整改建议）：针对每个问题给出具体、可执行的整改措施与提升建议；
4. 巡店总结（summary / high_risk_problems / main_problems / priority_suggest / business_suggest）。
%s
%s
%s

请直接调用 submit_inspection_result 工具提交结果，不要输出任何解释性文字、markdown 代码块或其他文本。scores 仅包含与巡店关键词/图片有关联的检查项，每项的 item_id 与上面给出的 item_id 一致，score 必须落在该检查项允许的分值范围内；无关的检查项不要返回评分；issues 与 suggestion 必须详尽充实；summary 为 30-60 字整体概括；high_risk_problems 提取所有高危风险问题（消防、食安等需立即处理）；main_problems 汇总主要问题；priority_suggest 给出 3 条优先整改建议；business_suggest 给出门店运营优化建议。high_risk_problems 与 main_problems 中每项的 item_id 必须与对应检查项的 item_id 一致（优先填写 item_id），无法关联到具体检查项时 item_id 留空。`, storeDesc, skillsPrompt, keywordsHint, photoHint, imageHint)
			messages = []chatMessage{{
				Role:    "system",
				Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。必须且只能调用 submit_inspection_result 工具返回结构化结果，禁止输出工具调用以外的任何解释性文字、markdown 代码块或普通文本。",
			}}
			if supportsVision && (hasStandardImages || hasPhotos) {
				messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, photos)})
			} else {
				messages = append(messages, chatMessage{Role: "user", Content: prompt})
			}
			schema = skillSpec.ResponseSchema
			if schema == nil {
				schema = defaultInspectionResponseSchema()
			}
			tools = []toolDefinition{{
				Type: "function",
				Function: toolFunctionSpec{
					Name:        "submit_inspection_result",
					Description: "提交巡店检查评分结果，包括每个检查项的得分、反馈问题、整体问题与整改建议",
					Parameters:  schema,
				},
			}}
		} else {
			standardImages := make([]UploadedFile, 0)
			seenStandard := make(map[string]bool)
			itemLines := make([]string, 0, len(items))
			for _, it := range items {
				scoreDesc := fmt.Sprintf("满分 %d 分", it.MaxScore)
				if len(it.ScoreOptions) > 0 {
					labels := make([]string, 0, len(it.ScoreOptions))
					allowed := make([]string, 0, len(it.ScoreOptions))
					for _, opt := range it.ScoreOptions {
						labels = append(labels, fmt.Sprintf("%s(%.1f分)", opt.Label, opt.Score))
						allowed = append(allowed, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", opt.Score), "0"), "."))
					}
					if it.ScoreType == "pass_fail" {
						scoreDesc = strings.Join(labels, " / ") + "，请从以上选项中选择"
					} else {
						scoreDesc = "分值评分，可选分值：" + strings.Join(allowed, " / ") + " 分（满分 " + fmt.Sprintf("%d", it.MaxScore) + " 分）"
					}
				}
				line := fmt.Sprintf("  - %s（%s）", it.Name, scoreDesc)
				if it.Standard != "" {
					line += fmt.Sprintf("；检查标准：%s", it.Standard)
				}
				if len(it.StandardImages) > 0 {
					line += "；已附带标准参照图"
					for _, img := range it.StandardImages {
						if img == "" || seenStandard[img] {
							continue
						}
						seenStandard[img] = true
						if f, err := loadImageAsUploadedFile(img); err == nil {
							standardImages = append(standardImages, *f)
						}
					}
				}
				itemLines = append(itemLines, line)
			}
			hasStandardImages := len(standardImages) > 0
			hasPhotos := len(photos) > 0
			imageHint := ""
			if hasStandardImages && hasPhotos {
				imageHint = "\n本次已附带检查标准参照图与巡店现场图，请结合标准图与现场图进行对比分析，重点关注现场与标准的差异。"
			} else if hasStandardImages {
				imageHint = "\n本次已附带检查标准参照图，请参照标准图进行评估。"
			} else if hasPhotos {
				imageHint = "\n本次已附带巡店现场图，请结合现场图进行评估。"
			}
			prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对「%s」进行巡店检查。
检查项及评分标准如下：
%s

请基于以上检查项标准与巡店信息，客观评估每个检查项的达标情况，重点输出：
1. scores：每个检查项的得分；
2. issues（问题与备注）：按检查项分条列出发现的问题与现场备注，覆盖所有不达标项；
3. suggestion（AI 整改建议）：针对每个问题给出具体、可执行的整改措施与提升建议；
4. 巡店总结（summary / high_risk_problems / main_problems / priority_suggest / business_suggest）。
%s
%s
%s

请严格以 JSON 格式返回（不要输出其他内容）：
{
  "scores": {"检查项名称": 得分（仅包含有针对性分析的项）, ...},
  "issues": "问题与备注，按检查项分条列出",
  "suggestion": "AI 整改建议，具体可执行",
  "summary": "整体一句话概括本次巡店情况（30-60字）",
  "high_risk_problems": [{"item_id":"关联检查项ID（无法关联留空）","item_name":"关联检查项名称","level":"高/中/低","desc":"问题描述"}],
  "main_problems": [{"item_id":"关联检查项ID（无法关联留空）","item_name":"关联检查项名称","level":"高/中/低","desc":"问题描述"}],
  "priority_suggest": [{"title":"建议标题","desc":"建议说明"}],
  "business_suggest": [{"title":"建议标题","desc":"建议说明"}]
}`, storeDesc, strings.Join(itemLines, "\n"), keywordsHint, photoHint, imageHint)
			messages = []chatMessage{{
				Role:    "system",
				Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。",
			}}
			if supportsVision && (hasStandardImages || hasPhotos) {
				messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, photos)})
			} else {
				messages = append(messages, chatMessage{Role: "user", Content: prompt})
			}
		}

		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}
		if kw := strings.TrimSpace(keywords); kw != "" {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("携带巡店关键词：%s", kw)}
		}
		if len(photos) > 0 {
			events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("附带 %d 张图片（多模态分析）", len(photos))}
		}

		reqSummary := map[string]interface{}{
			"model":       model,
			"baseURL":     baseURL,
			"tool_choice": len(tools) > 0,
		}
		msgSummaries := make([]map[string]interface{}, 0, len(messages))
		for _, m := range messages {
			content := m.Content
			if vc, ok := content.([]interface{}); ok {
				content = fmt.Sprintf("[vision content, %d parts]", len(vc))
			}
			if s, ok := content.(string); ok {
				if len(s) > 500 {
					content = s[:500] + "..."
				}
			}
			msgSummaries = append(msgSummaries, map[string]interface{}{
				"role":    m.Role,
				"content": content,
			})
		}
		reqSummary["messages"] = msgSummaries
		reqDetailBytes, _ := json.MarshalIndent(reqSummary, "", "  ")
		events <- AIStreamEvent{Event: "request_data", Message: fmt.Sprintf("向模型 %s 发送请求", model), Detail: string(reqDetailBytes)}

		var resp *http.Response
		if len(tools) > 0 {
			// 强制调用 submit_inspection_result，避免模型输出普通文本或被截断。
			toolChoice := map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name": "submit_inspection_result",
				},
			}
			if apiFormat == "openai_responses" {
				resp, err = callResponsesStreamWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 12000)
			} else {
				resp, err = callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 12000)
			}
		} else {
			if apiFormat == "openai_responses" {
				resp, err = callResponses(apiKey, baseURL, model, messages, true)
			} else {
				resp, err = callChatCompletions(apiKey, baseURL, model, messages, true)
			}
		}
		if err != nil {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s", planName, err)}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s", planName, resp.StatusCode, string(bodyBytes))}
			return
		}

		respSummary := map[string]interface{}{
			"status": resp.StatusCode,
			"model":  model,
			"stream": true,
		}
		respDetailBytes, _ := json.MarshalIndent(respSummary, "", "  ")
		events <- AIStreamEvent{Event: "response_data", Message: "收到模型响应", Detail: string(respDetailBytes)}

		if apiFormat == "openai_responses" {
			events <- AIStreamEvent{Event: "log", Level: "req", Message: "POST /responses → SSE 连接已建立"}
		} else {
			events <- AIStreamEvent{Event: "log", Level: "req", Message: "POST /chat/completions → SSE 连接已建立"}
		}

		scanner := bufio.NewScanner(resp.Body)
		fullContent := ""
		// 流式工具调用参数累加：key 为 call index
		accumulatedArgs := make(map[int]string)
		accumulatedName := make(map[int]string)
		accumulatedID := make(map[int]string)
		if apiFormat == "openai_responses" {
			respAccumulatedName := ""
			respAccumulatedArgs := ""
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
						respAccumulatedName = evt.Item.Name
					}
				case "response.function_call_arguments.delta":
					respAccumulatedArgs += evt.Delta
				case "response.output_text.delta":
					if evt.Delta != "" {
						fullContent += evt.Delta
						events <- AIStreamEvent{Event: "chunk", Text: evt.Delta}
					}
				}
			}
			if respAccumulatedName != "" {
				accumulatedName[0] = respAccumulatedName
			}
			if respAccumulatedArgs != "" {
				accumulatedArgs[0] = respAccumulatedArgs
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

				// 处理普通 content 增量
				if delta.Content != "" {
					fullContent += delta.Content
					events <- AIStreamEvent{Event: "chunk", Text: delta.Content}
				}

				// 处理流式 tool_calls 增量
				for _, tc := range delta.ToolCalls {
					idx := 0
					// toolCalls 数组通常按顺序出现；若不存在 index，则按 0 累加
					if tc.ID != "" {
						accumulatedID[idx] = tc.ID
					}
					if tc.Function.Name != "" {
						accumulatedName[idx] = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						accumulatedArgs[idx] += tc.Function.Arguments
					}
				}
			}
		}

		// 优先从流式工具调用结果解析
		var result *InspectionAIResult
		for idx, args := range accumulatedArgs {
			if accumulatedName[idx] != "submit_inspection_result" {
				continue
			}
			args = strings.TrimSpace(args)
			if args == "" {
				continue
			}
			if parsed, perr := parseInspectionToolArgs(args); perr == nil {
				result = parsed
				events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已解析 submit_inspection_result 工具调用结果"}
				break
			}
		}

		// 未解析到 tool_calls，回退解析 content JSON
		if result == nil {
			if parsed, perr := parseInspectionContentJSON(fullContent); perr == nil {
				result = parsed
				events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已从 content 回退解析 JSON 结果"}
			}
		}

		if result == nil {
			available, aerr := getAvailablePlans()
			if aerr != nil {
				events <- AIStreamEvent{Event: "error", Message: "获取可用模型配置失败"}
				return
			}
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」返回的巡店报告格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName), AvailablePlans: available}
			return
		}

		resultScores := make([]map[string]interface{}, 0, len(result.Scores))
		for itemID, score := range result.Scores {
			resultScores = append(resultScores, map[string]interface{}{
				"item_id": itemID,
				"score":   score,
				"comment": result.Comments[itemID],
			})
		}
		events <- AIStreamEvent{
			Event:   "result",
			Message: "AI 巡店分析完成",
			Data: map[string]interface{}{
				"scores":             resultScores,
				"issues":             result.Issues,
				"suggestion":         result.Suggestion,
				"summary":            result.Summary,
				"high_risk_problems": result.HighRiskProblems,
				"main_problems":      result.MainProblems,
				"priority_suggest":   result.PrioritySuggest,
				"business_suggest":   result.BusinessSuggest,
			},
		}
	}()

	return events, nil
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

type SingleItemAnalysisRequest struct {
	ItemName       string               `json:"item_name"`
	ItemID         string               `json:"item_id"`
	Standard       string               `json:"standard"`
	StandardImages []string             `json:"standard_images"`
	ScoreType      string               `json:"score_type"`
	MaxScore       int                  `json:"max_score"`
	ScoreOptions   []models.ScoreOption `json:"score_options"`
	Comment        string               `json:"comment"`
	CurrentScore   float64              `json:"current_score"`
	Photos         []UploadedFile       `json:"photos"`
	Keywords       string               `json:"keywords"`
	PlanID         string               `json:"plan_id"`
	ModelID        string               `json:"model_id"`
}

type SingleItemAnalysisResult struct {
	Score          float64 `json:"score"`
	Comment        string  `json:"comment"`
	Suggestion     string  `json:"suggestion"`
	PhotoRelevance *bool   `json:"photo_relevance,omitempty"`
}

func AnalyzeSingleInspectionItem(req SingleItemAnalysisRequest) (*SingleItemAnalysisResult, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(req.PlanID, req.ModelID)
	if err != nil {
		return nil, err
	}

	scoreDesc := ""
	if req.ScoreType == "pass_fail" {
		labels := make([]string, 0, len(req.ScoreOptions))
		for _, opt := range req.ScoreOptions {
			labels = append(labels, fmt.Sprintf("%s(%.1f分)", opt.Label, opt.Score))
		}
		scoreDesc = strings.Join(labels, " / ") + "，请从以上选项中选择"
	} else {
		allowed := make([]string, 0, len(req.ScoreOptions))
		for _, opt := range req.ScoreOptions {
			allowed = append(allowed, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", opt.Score), "0"), "."))
		}
		if len(allowed) > 0 {
			scoreDesc = "分值评分，可选分值：" + strings.Join(allowed, " / ") + " 分（满分 " + fmt.Sprintf("%d", req.MaxScore) + " 分）"
		} else {
			scoreDesc = fmt.Sprintf("分值评分，0 ~ %d 分", req.MaxScore)
		}
	}

	standardLine := ""
	if req.Standard != "" {
		standardLine = fmt.Sprintf("检查标准：%s\n", req.Standard)
	}

	feedbackLine := ""
	if req.Comment != "" {
		feedbackLine = fmt.Sprintf("巡店人员现场反馈：%s\n", req.Comment)
	}

	scoreLine := ""
	if req.CurrentScore > 0 {
		scoreLine = fmt.Sprintf("巡店人员当前评分：%.1f 分\n", req.CurrentScore)
	}

	standardImageLine := ""
	if len(req.StandardImages) > 0 {
		standardImageLine = fmt.Sprintf("附带 %d 张检查标准图（标准参照图），请对比标准图检查现场情况。", len(req.StandardImages))
	}

	photoDesc := ""
	if len(req.Photos) > 0 {
		photoDesc = fmt.Sprintf("附带 %d 张巡店现场图片，请结合图片内容进行客观判断。", len(req.Photos))
	}

	keywordsHint := ""
	if kw := strings.TrimSpace(req.Keywords); kw != "" {
		keywordsHint = fmt.Sprintf("巡店补充信息：%s\n", kw)
	}

	prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对以下单个检查项进行独立评分分析。

检查项：%s
%s评分方式：%s
%s%s%s%s%s

请综合标题、检查标准、标准参照图、巡店人员当前评分与反馈、现场图片等信息，客观评估该项的达标情况，生成更合理的评分与反馈问题。注意：如果提供了现场图片，需要判断图片内容是否与当前检查项相关（例如图片是否确实展示了该检查项对应的场景），并严格的仅返回 JSON（不要输出其他内容）：
{
  "score": 得分,
  "comment": "问题与备注",
  "suggestion": "整改建议",
  "photo_relevance": true或false（仅当有现场图片时返回，图片内容是否与当前检查项匹配）
}`, req.ItemName, standardLine, scoreDesc, scoreLine, feedbackLine, standardImageLine, photoDesc, keywordsHint)

	messages := []chatMessage{{
		Role:    "system",
		Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。",
	}}

	// 加载标准图与现场图，vision 模型可直接看图片进行对比与相关性判断
	standardImages := make([]UploadedFile, 0, 1)
	for _, img := range req.StandardImages {
		if img == "" {
			continue
		}
		if f, err := loadImageAsUploadedFile(img); err == nil {
			standardImages = append(standardImages, *f)
		}
	}
	if supportsVision && (len(standardImages) > 0 || len(req.Photos) > 0) {
		messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, req.Photos)})
	} else {
		messages = append(messages, chatMessage{Role: "user", Content: prompt})
	}

	var resp *http.Response
	if apiFormat == "openai_responses" {
		resp, err = callResponses(apiKey, baseURL, model, messages, false)
	} else {
		resp, err = callChatCompletions(apiKey, baseURL, model, messages, false)
	}
	if err != nil {
		return nil, aiCallError(planName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
	}

	var content string
	if apiFormat == "openai_responses" {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		content, _, _ = extractResponsesResult(respBody)
	} else {
		var raw struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return nil, aiCallError(planName, err)
		}
		if len(raw.Choices) == 0 {
			return nil, aiCallError(planName, fmt.Errorf("模型无返回内容"))
		}
		content = raw.Choices[0].Message.Content
	}
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) > 1 {
			content = strings.TrimSpace(parts[1])
			content = strings.TrimPrefix(content, "json")
			content = strings.TrimSpace(content)
		}
	}

	var result SingleItemAnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, aiCallError(planName, fmt.Errorf("模型返回格式异常，无法解析"))
	}

	return &result, nil
}

func AnalyzeSingleInspectionItemStream(req SingleItemAnalysisRequest) (<-chan AIStreamEvent, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(req.PlanID, req.ModelID)
	if err != nil {
		return nil, err
	}

	events := make(chan AIStreamEvent, 32)
	go func() {
		defer close(events)

		scoreDesc := ""
		if req.ScoreType == "pass_fail" {
			labels := make([]string, 0, len(req.ScoreOptions))
			for _, opt := range req.ScoreOptions {
				labels = append(labels, fmt.Sprintf("%s(%.1f分)", opt.Label, opt.Score))
			}
			scoreDesc = strings.Join(labels, " / ") + "，请从以上选项中选择"
		} else {
			allowed := make([]string, 0, len(req.ScoreOptions))
			for _, opt := range req.ScoreOptions {
				allowed = append(allowed, strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", opt.Score), "0"), "."))
			}
			if len(allowed) > 0 {
				scoreDesc = "分值评分，可选分值：" + strings.Join(allowed, " / ") + " 分（满分 " + fmt.Sprintf("%d", req.MaxScore) + " 分）"
			} else {
				scoreDesc = fmt.Sprintf("分值评分，0 ~ %d 分", req.MaxScore)
			}
		}

		standardLine := ""
		if req.Standard != "" {
			standardLine = fmt.Sprintf("检查标准：%s\n", req.Standard)
		}

		feedbackLine := ""
		if req.Comment != "" {
			feedbackLine = fmt.Sprintf("巡店人员现场反馈：%s\n", req.Comment)
		}

		scoreLine := ""
		if req.CurrentScore > 0 {
			scoreLine = fmt.Sprintf("巡店人员当前评分：%.1f 分\n", req.CurrentScore)
		}

		standardImageLine := ""
		if len(req.StandardImages) > 0 {
			standardImageLine = fmt.Sprintf("附带 %d 张检查标准图（标准参照图），请对比标准图检查现场情况。", len(req.StandardImages))
		}

		photoDesc := ""
		if len(req.Photos) > 0 {
			photoDesc = fmt.Sprintf("附带 %d 张巡店现场图片，请结合图片内容进行客观判断。", len(req.Photos))
		}

		keywordsHint := ""
		if kw := strings.TrimSpace(req.Keywords); kw != "" {
			keywordsHint = fmt.Sprintf("巡店补充信息：%s\n", kw)
		}

		prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导。请对以下单个检查项进行独立评分分析。

检查项：%s
%s评分方式：%s
%s%s%s%s%s

请综合标题、检查标准、标准参照图、巡店人员当前评分与反馈、现场图片等信息，客观评估该项的达标情况，生成更合理的评分与反馈问题。注意：如果提供了现场图片，需要判断图片内容是否与当前检查项相关（例如图片是否确实展示了该检查项对应的场景），并严格的仅返回 JSON（不要输出其他内容）：
{
  "score": 得分,
  "comment": "问题与备注",
  "suggestion": "整改建议",
  "photo_relevance": true或false（仅当有现场图片时返回，图片内容是否与当前检查项匹配）
}`, req.ItemName, standardLine, scoreDesc, scoreLine, feedbackLine, standardImageLine, photoDesc, keywordsHint)

		messages := []chatMessage{{
			Role:    "system",
			Content: "你是专业的连锁门店巡店督导专家，擅长门店标准化检查，输出问题与整改建议。使用中文回复。",
		}}

		// 加载标准图与现场图，vision 模型可直接看图片进行对比与相关性判断
		standardImages := make([]UploadedFile, 0, 1)
		for _, img := range req.StandardImages {
			if img == "" {
				continue
			}
			if f, err := loadImageAsUploadedFile(img); err == nil {
				standardImages = append(standardImages, *f)
			}
		}
		if supportsVision && (len(standardImages) > 0 || len(req.Photos) > 0) {
			messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, req.Photos)})
		} else {
			messages = append(messages, chatMessage{Role: "user", Content: prompt})
		}

		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}

		reqSummary := map[string]interface{}{
			"model":      model,
			"baseURL":    baseURL,
			"item":       req.ItemName,
			"score_type": req.ScoreType,
			"max_score":  req.MaxScore,
		}
		msgSummaries := make([]map[string]interface{}, 0, len(messages))
		for _, m := range messages {
			content := m.Content
			if vc, ok := content.([]interface{}); ok {
				content = fmt.Sprintf("[vision content, %d parts]", len(vc))
			}
			if s, ok := content.(string); ok {
				if len(s) > 500 {
					content = s[:500] + "..."
				}
			}
			msgSummaries = append(msgSummaries, map[string]interface{}{
				"role":    m.Role,
				"content": content,
			})
		}
		reqSummary["messages"] = msgSummaries
		reqDetailBytes, _ := json.MarshalIndent(reqSummary, "", "  ")
		events <- AIStreamEvent{Event: "request_data", Message: fmt.Sprintf("向模型 %s 发送请求", model), Detail: string(reqDetailBytes)}

		var resp *http.Response
		if apiFormat == "openai_responses" {
			resp, err = callResponses(apiKey, baseURL, model, messages, true)
		} else {
			resp, err = callChatCompletions(apiKey, baseURL, model, messages, true)
		}
		if err != nil {
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：%s", planName, err)}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s", planName, resp.StatusCode, string(bodyBytes))}
			return
		}

		respSummary := map[string]interface{}{
			"status": resp.StatusCode,
			"model":  model,
			"stream": true,
		}
		respDetailBytes, _ := json.MarshalIndent(respSummary, "", "  ")
		events <- AIStreamEvent{Event: "response_data", Message: "收到模型响应", Detail: string(respDetailBytes)}

		scanner := bufio.NewScanner(resp.Body)
		fullContent := ""
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
				}
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					continue
				}
				if evt.Type == "response.output_text.delta" && evt.Delta != "" {
					fullContent += evt.Delta
					events <- AIStreamEvent{Event: "chunk", Text: evt.Delta}
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
							Content string `json:"content"`
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
			}
		}

		content := strings.TrimSpace(fullContent)
		if strings.HasPrefix(content, "```") {
			parts := strings.Split(content, "```")
			if len(parts) > 1 {
				content = strings.TrimSpace(parts[1])
				content = strings.TrimPrefix(content, "json")
				content = strings.TrimSpace(content)
			}
		}

		var result SingleItemAnalysisResult
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			events <- AIStreamEvent{Event: "error", Message: "模型返回格式异常，无法解析"}
			return
		}

		events <- AIStreamEvent{
			Event:   "result",
			Message: "AI 分析完成",
			Data: map[string]interface{}{
				"score":           result.Score,
				"comment":         result.Comment,
				"suggestion":      result.Suggestion,
				"photo_relevance": result.PhotoRelevance,
			},
		}
	}()

	return events, nil
}
