package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
)

func SeedInspectionTemplates() error {
	o := GetOrm()
	count, err := o.QueryTable(new(models.InspectionTemplate)).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	template := &models.InspectionTemplate{
		ID:          uuid.NewString(),
		Name:        "标准门店巡检",
		Description: "覆盖门店形象、卫生、陈列、服务、安全等常规检查项，适用于日常巡店",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	now := time.Now()
	items := []struct {
		Category      string
		Title         string
		Standard      string
		ScoreType     string
		MaxScore      int
		ScoreOptions  []models.ScoreOption
		RequireRemark bool
		RequirePhoto  bool
		ShowRemark    bool
		ShowPhoto     bool
	}{
		{Category: "形象", Title: "门头形象", Standard: "门店招牌完整、干净、夜间亮灯正常，无破损缺字", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "卫生", Title: "环境卫生", Standard: "地面、柜台、货架无灰尘污渍，垃圾及时清理", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "陈列", Title: "商品陈列", Standard: "商品摆放整齐、标签清晰、无过期临期商品混放", ScoreType: "score", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("score", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
		{Category: "服务", Title: "服务规范", Standard: "员工着装统一、佩戴工牌、服务用语规范", ScoreType: "pass_fail", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("pass_fail", 10), RequireRemark: true, RequirePhoto: false, ShowRemark: true, ShowPhoto: true},
		{Category: "安全", Title: "消防安全", Standard: "灭火器在有效期内、消防通道畅通无堵塞", ScoreType: "pass_fail", MaxScore: 10, ScoreOptions: models.DefaultScoreOptions("pass_fail", 10), RequireRemark: true, RequirePhoto: true, ShowRemark: true, ShowPhoto: true},
	}
	entries := make([]*models.InspectionTemplateItem, 0, len(items))
	for i, item := range items {
		entries = append(entries, &models.InspectionTemplateItem{
			ID:            uuid.NewString(),
			TemplateID:    template.ID,
			Category:      item.Category,
			Title:         item.Title,
			Standard:      item.Standard,
			ScoreType:     item.ScoreType,
			MaxScore:      item.MaxScore,
			ScoreOptions:  models.MarshalScoreOptions(item.ScoreOptions),
			RequireRemark: item.RequireRemark,
			RequirePhoto:  item.RequirePhoto,
			ShowRemark:    item.ShowRemark,
			ShowPhoto:     item.ShowPhoto,
			SortOrder:     i,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Insert(template); err != nil {
		tx.Rollback()
		return err
	}
	for _, e := range entries {
		if _, err := tx.Insert(e); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func GetInspectionTemplateItems(templateID string) ([]models.InspectionTemplateItem, error) {
	var items []models.InspectionTemplateItem
	_, err := GetOrm().QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", templateID).
		Filter("is_active", true).
		OrderBy("sort_order").
		All(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func EffectiveScoreOptions(item *models.InspectionTemplateItem) []models.ScoreOption {
	options := models.UnmarshalScoreOptions(item.ScoreOptions)
	if len(options) == 0 {
		return models.DefaultScoreOptions(item.ScoreType, item.MaxScore)
	}
	return options
}

func ValidateScoreOptions(options []models.ScoreOption) error {
	if len(options) == 0 {
		return errors.New("评分选项不能为空")
	}
	allowed := map[float64]bool{
		0: true, 0.5: true, 1: true, 2: true, 3: true, 4: true, 5: true,
	}
	minScore := 0.0
	maxScore := 5.0
	hasMin := false
	hasMax := false
	seenScores := make(map[float64]bool)
	seenLabels := make(map[string]bool)
	prevScore := -1.0
	for _, opt := range options {
		if !allowed[opt.Score] {
			return errors.New("分值仅允许 0、0.5、1、2、3、4、5")
		}
		if seenScores[opt.Score] {
			return errors.New("选项分值不能重复")
		}
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			return errors.New("选项标签不能为空")
		}
		if seenLabels[label] {
			return errors.New("选项标签不能重复")
		}
		if opt.Score <= prevScore {
			return errors.New("选项分值必须从小到大排列")
		}
		seenScores[opt.Score] = true
		seenLabels[label] = true
		prevScore = opt.Score
		if opt.Score == minScore {
			hasMin = true
		}
		if opt.Score == maxScore {
			hasMax = true
		}
	}
	if !hasMin || !hasMax {
		return errors.New("必须包含最小值 0 与最大值 5")
	}
	return nil
}

func ValidatePassFailOptions(options []models.ScoreOption) error {
	if len(options) == 0 {
		return errors.New("评分选项不能为空")
	}
	seenLabels := make(map[string]bool)
	for _, opt := range options {
		if opt.Score <= 0 {
			return errors.New("选项序号必须为正整数")
		}
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			return errors.New("选项不能为空")
		}
		if seenLabels[label] {
			return errors.New("选项不能重复")
		}
		seenLabels[label] = true
	}
	return nil
}

func GetActiveTemplates() ([]models.InspectionTemplate, error) {
	var templates []models.InspectionTemplate
	_, err := GetOrm().QueryTable(new(models.InspectionTemplate)).
		Filter("is_active", true).
		OrderBy("-created_at").
		All(&templates)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func FindInspectionTemplate(templateID string) (*models.InspectionTemplate, error) {
	if strings.TrimSpace(templateID) == "" {
		return nil, errors.New("模板不存在")
	}
	var template models.InspectionTemplate
	err := GetOrm().QueryTable(new(models.InspectionTemplate)).
		Filter("id", templateID).
		One(&template)
	if err != nil {
		return nil, errors.New("模板不存在")
	}
	return &template, nil
}

func SaveInspectionTemplate(template *models.InspectionTemplate, items []*models.InspectionTemplateItem) error {
	o := GetOrm()
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if template.ID == "" {
		template.ID = uuid.NewString()
		template.CreatedAt = time.Now()
		template.UpdatedAt = time.Now()
		if _, err := tx.Insert(template); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		template.UpdatedAt = time.Now()
		if _, err := tx.Update(template); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.QueryTable(new(models.InspectionTemplateItem)).
			Filter("template_id", template.ID).
			Delete(); err != nil {
			tx.Rollback()
			return err
		}
	}
	now := time.Now()
	for i, item := range items {
		item.ID = uuid.NewString()
		item.TemplateID = template.ID
		item.SortOrder = i
		item.IsActive = true
		item.CreatedAt = now
		item.UpdatedAt = now
		if _, err := tx.Insert(item); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func DeleteInspectionTemplate(templateID string) error {
	o := GetOrm()
	template, err := FindInspectionTemplate(templateID)
	if err != nil {
		return err
	}
	hasInspection := o.QueryTable(new(models.Inspection)).
		Filter("template_id", template.ID).
		Exist()
	if hasInspection {
		return errors.New("该模板已被巡店记录使用，无法删除")
	}
	tx, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", template.ID).
		Delete(); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Delete(template); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func CountTemplateItems(templateID string) (int64, error) {
	return GetOrm().QueryTable(new(models.InspectionTemplateItem)).
		Filter("template_id", templateID).
		Filter("is_active", true).
		Count()
}

// AIGenerateTemplateItemRequest AI 生成检查项请求。
type AIGenerateTemplateItemRequest struct {
	PlanID             string         `json:"plan_id"`
	ModelID            string         `json:"model_id"`
	Description        string         `json:"description"`
	Photos             []UploadedFile `json:"photos"`
	ExistingCategories []string       `json:"existing_categories"`
}

// AIGenerateTemplateItemResult AI 生成的检查项结果。
type AIGenerateTemplateItemResult struct {
	Category       string               `json:"category"`
	Title          string               `json:"title"`
	Standard       string               `json:"standard"`
	StandardImages []string             `json:"standard_images"`
	ScoreType      string               `json:"score_type"`
	MaxScore       int                  `json:"max_score"`
	ScoreOptions   []models.ScoreOption `json:"score_options"`
}

// AIGenerateTemplateResult AI 生成的模板信息（名称、描述）与检查项列表。
type AIGenerateTemplateResult struct {
	TemplateName        string                         `json:"template_name"`
	TemplateDescription string                         `json:"template_description"`
	Items               []AIGenerateTemplateItemResult `json:"items"`
}

// buildTemplateItemAIMessages 构建 AI 生成检查项所需的 messages / tools / toolChoice。
func buildTemplateItemAIMessages(req AIGenerateTemplateItemRequest, supportsVision bool) ([]chatMessage, []toolDefinition, interface{}) {
	photoDesc := ""
	uploadedPhotos := make([]UploadedFile, 0, len(req.Photos))
	for _, p := range req.Photos {
		if p.Data == "" {
			continue
		}
		uploadedPhotos = append(uploadedPhotos, p)
	}
	if len(uploadedPhotos) > 0 {
		photoDesc = fmt.Sprintf("已上传 %d 张图片，请优先基于图片内容识别可检查的项目。", len(uploadedPhotos))
	}

	descLine := ""
	if d := strings.TrimSpace(req.Description); d != "" {
		descLine = fmt.Sprintf("用户文字描述：%s\n", d)
	}

	catLine := ""
	if len(req.ExistingCategories) > 0 {
		catLine = fmt.Sprintf("当前模板已有分类（请优先将识别出的检查项归入这些分类，分类名尽量保持一致；无法匹配时再创建新分类）：%s\n", strings.Join(req.ExistingCategories, "、"))
	}

	prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导专家，擅长根据门店现场图片或文字描述，设计标准化检查表。

任务：请从用户提供的图片和/或文字描述中，生成：
1. template_name（模板名称）：简短明确，概括该检查表用途，20 字以内；
2. template_description（模板描述）：一句话说明模板适用场景与覆盖范围，50 字以内；
3. items（检查项列表）：识别所有适合作为巡店检查的项目，每个项目包含：
   - category（分类）：检查项所属分类，要求简洁、统一，优先使用已有分类；
   - title（检查项标题）：简短明确，15 字以内；
   - standard（检查标准）：具体可执行的判定标准，30-80 字；
   - standard_images（标准图 URL 列表）：如果用户上传的图片适合作为该检查项的标准参考图，请使用图片原始 data URL 填入（支持多张），否则留空数组；
   - score_type（评分方式）：固定为 "score"（分值评分）；
   - max_score（满分）：固定为 5；
   - score_options（评分选项）：固定为 [{"score":0,"label":"0分"},{"score":5,"label":"5分"}]。

%s
%s
%s

请直接调用 submit_template_items 工具提交结果，禁止输出工具调用以外的任何解释性文字、markdown 代码块或普通文本。`, photoDesc, descLine, catLine)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"template_name":        map[string]interface{}{"type": "string", "description": "模板名称，概括检查表用途，20 字以内"},
			"template_description": map[string]interface{}{"type": "string", "description": "模板描述，说明适用场景与覆盖范围，50 字以内"},
			"items": map[string]interface{}{
				"type":        "array",
				"description": "识别出的检查项列表",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"category":        map[string]interface{}{"type": "string", "description": "检查项分类，优先匹配已有分类"},
						"title":           map[string]interface{}{"type": "string", "description": "检查项标题，15 字以内"},
						"standard":        map[string]interface{}{"type": "string", "description": "具体可执行的检查标准"},
						"standard_images": map[string]interface{}{"type": "array", "description": "标准图 URL 列表，无法确定则留空", "items": map[string]interface{}{"type": "string"}},
						"score_type":      map[string]interface{}{"type": "string", "enum": []string{"score"}, "description": "评分方式"},
						"max_score":       map[string]interface{}{"type": "integer", "description": "满分"},
						"score_options":   map[string]interface{}{"type": "array", "description": "评分选项"},
					},
					"required": []string{"category", "title", "standard", "score_type", "max_score", "score_options"},
				},
			},
		},
		"required": []string{"items"},
	}

	messages := []chatMessage{{
		Role:    "system",
		Content: "你是专业的连锁门店巡店督导专家，擅长设计标准化检查表。使用中文回复。必须且只能调用 submit_template_items 工具返回结构化结果，禁止输出工具调用以外的任何解释性文字、markdown 代码块或普通文本。",
	}}

	if supportsVision && len(uploadedPhotos) > 0 {
		messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, nil, uploadedPhotos)})
	} else {
		messages = append(messages, chatMessage{Role: "user", Content: prompt})
	}

	tools := []toolDefinition{{
		Type: "function",
		Function: toolFunctionSpec{
			Name:        "submit_template_items",
			Description: "提交 AI 识别生成的检查表模板检查项列表",
			Parameters:  schema,
		},
	}}
	toolChoice := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name": "submit_template_items",
		},
	}
	return messages, tools, toolChoice
}

// GenerateTemplateItemsByAI 根据图片或文字描述，让 AI 识别并生成结构化检查项列表。
func GenerateTemplateItemsByAI(req AIGenerateTemplateItemRequest) ([]AIGenerateTemplateItemResult, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(req.PlanID, req.ModelID)
	if err != nil {
		return nil, err
	}

	messages, tools, toolChoice := buildTemplateItemAIMessages(req, supportsVision)

	var resp *http.Response
	if apiFormat == "openai_responses" {
		resp, err = callResponsesWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
	} else {
		resp, err = callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
	}
	if err != nil {
		return nil, aiCallError(planName, err)
	}
	if resp.StatusCode == http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		lower := strings.ToLower(string(bodyBytes))
		if strings.Contains(lower, "tool_choice") || strings.Contains(lower, "tool choice") {
			if apiFormat == "openai_responses" {
				resp, err = callResponsesWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
			} else {
				resp, err = callChatCompletionsWithTools(apiKey, baseURL, model, messages, tools, nil, 8000)
			}
			if err != nil {
				return nil, aiCallError(planName, err)
			}
		} else {
			return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, aiCallError(planName, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)))
	}

	var args string
	if apiFormat == "openai_responses" {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		content, toolName, toolArgs := extractResponsesResult(respBody)
		if toolName != "" {
			args = strings.TrimSpace(toolArgs)
		} else {
			args = stripCodeFence(content)
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
			args = strings.TrimSpace(choice.Message.ToolCalls[0].Function.Arguments)
		} else {
			args = stripCodeFence(choice.Message.Content)
		}
	}

	parsed, parseErr := parseGeneratedItems(args, req.ExistingCategories)
	if parseErr != nil {
		return nil, aiCallError(planName, fmt.Errorf("模型返回格式异常，无法解析"))
	}
	return parsed, nil
}

// GenerateTemplateItemsByAIStream 根据图片或文字描述，流式生成检查项列表，通过 events 输出 SSE 事件。
func GenerateTemplateItemsByAIStream(req AIGenerateTemplateItemRequest, events chan<- AIStreamEvent) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(req.PlanID, req.ModelID)
	if err != nil {
		events <- AIStreamEvent{Event: "error", Message: err.Error()}
		return
	}

	messages, tools, toolChoice := buildTemplateItemAIMessages(req, supportsVision)

	events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("使用模型配置 %s（%s）", planName, model)}
	if len(req.Photos) > 0 {
		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("附带 %d 张图片（多模态识别）", len(req.Photos))}
	}
	if len(req.ExistingCategories) > 0 {
		events <- AIStreamEvent{Event: "log", Level: "info", Message: fmt.Sprintf("已提供 %d 个已有分类用于归类", len(req.ExistingCategories))}
	}

	var resp *http.Response
	if apiFormat == "openai_responses" {
		resp, err = callResponsesStreamWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
	} else {
		resp, err = callChatCompletionsStreamWithTools(apiKey, baseURL, model, messages, tools, toolChoice, 8000)
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
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」调用失败：HTTP %d: %s", planName, resp.StatusCode, string(bodyBytes))}
		return
	}
	defer resp.Body.Close()

	if apiFormat == "openai_responses" {
		events <- AIStreamEvent{Event: "log", Level: "req", Message: "POST /responses → SSE 连接已建立"}
	} else {
		events <- AIStreamEvent{Event: "log", Level: "req", Message: "POST /chat/completions → SSE 连接已建立"}
	}

	scanner := bufio.NewScanner(resp.Body)
	fullContent := ""
	accumulatedArgs := make(map[int]string)
	accumulatedName := make(map[int]string)
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
	}
	if err := scanner.Err(); err != nil {
		events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("读取模型流失败：%s", err)}
		return
	}

	var result *AIGenerateTemplateResult
	for idx, args := range accumulatedArgs {
		if accumulatedName[idx] != "submit_template_items" {
			continue
		}
		args = strings.TrimSpace(args)
		if args == "" {
			continue
		}
		if parsed, perr := parseGeneratedTemplateResult(args, req.ExistingCategories); perr == nil {
			result = parsed
			events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已解析 submit_template_items 工具调用结果"}
			break
		}
	}
	if result == nil {
		if parsed, perr := parseGeneratedTemplateResult(stripCodeFence(fullContent), req.ExistingCategories); perr == nil {
			result = parsed
			events <- AIStreamEvent{Event: "log", Level: "ok", Message: "已从文本内容回退解析 JSON 结果"}
		}
	}
	if result == nil || len(result.Items) == 0 {
		events <- AIStreamEvent{Event: "error", Message: fmt.Sprintf("模型「%s」返回的检查项格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName)}
		return
	}
	events <- AIStreamEvent{Event: "result", Message: "AI 智能添加检查项完成", Data: map[string]interface{}{
		"template_name":        result.TemplateName,
		"template_description": result.TemplateDescription,
		"items":                result.Items,
	}}
}

type generatedItemRaw struct {
	Category       string           `json:"category"`
	Title          string           `json:"title"`
	Standard       string           `json:"standard"`
	StandardImages []string         `json:"standard_images"`
	ScoreType      string           `json:"score_type"`
	MaxScore       json.RawMessage  `json:"max_score"`
	ScoreOptions   []scoreOptionRaw `json:"score_options"`
}

type generatedTemplateRaw struct {
	TemplateName        string             `json:"template_name"`
	TemplateDescription string             `json:"template_description"`
	Items               []generatedItemRaw `json:"items"`
}

type scoreOptionRaw struct {
	Score json.RawMessage `json:"score"`
	Label string          `json:"label"`
}

func parseIntRaw(raw json.RawMessage, fallback float64) float64 {
	if len(raw) == 0 {
		return fallback
	}
	s := strings.Trim(string(raw), `"`)
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return n
}

func normalizeRawItems(items []generatedItemRaw, existingCategories []string) []AIGenerateTemplateItemResult {
	result := make([]AIGenerateTemplateItemResult, 0, len(items))
	for _, it := range items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			continue
		}
		opts := make([]models.ScoreOption, 0, len(it.ScoreOptions))
		for _, o := range it.ScoreOptions {
			opts = append(opts, models.ScoreOption{Score: parseIntRaw(o.Score, 0), Label: o.Label})
		}
		result = append(result, AIGenerateTemplateItemResult{
			Category:       it.Category,
			Title:          title,
			Standard:       it.Standard,
			StandardImages: it.StandardImages,
			ScoreType:      it.ScoreType,
			MaxScore:       int(parseIntRaw(it.MaxScore, 5)),
			ScoreOptions:   opts,
		})
	}
	return normalizeGeneratedItems(result, existingCategories)
}

func parseGeneratedItems(raw string, existingCategories []string) ([]AIGenerateTemplateItemResult, error) {
	result, err := parseGeneratedTemplateResult(raw, existingCategories)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("无法解析为检查项列表")
	}
	return result.Items, nil
}

func parseGeneratedTemplateResult(raw string, existingCategories []string) (*AIGenerateTemplateResult, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("空内容")
	}
	var wrapper generatedTemplateRaw
	if err := json.Unmarshal([]byte(s), &wrapper); err == nil && (wrapper.Items != nil || strings.TrimSpace(wrapper.TemplateName) != "") {
		return normalizeGeneratedTemplate(wrapper, existingCategories), nil
	}
	var arr []generatedItemRaw
	if err := json.Unmarshal([]byte(s), &arr); err == nil && len(arr) > 0 {
		return &AIGenerateTemplateResult{
			Items: normalizeRawItems(arr, existingCategories),
		}, nil
	}
	var one generatedItemRaw
	if err := json.Unmarshal([]byte(s), &one); err == nil && strings.TrimSpace(one.Title) != "" {
		return &AIGenerateTemplateResult{
			Items: normalizeRawItems([]generatedItemRaw{one}, existingCategories),
		}, nil
	}
	return nil, fmt.Errorf("无法解析为模板信息")
}

func normalizeGeneratedTemplate(raw generatedTemplateRaw, existingCategories []string) *AIGenerateTemplateResult {
	return &AIGenerateTemplateResult{
		TemplateName:        strings.TrimSpace(raw.TemplateName),
		TemplateDescription: strings.TrimSpace(raw.TemplateDescription),
		Items:               normalizeRawItems(raw.Items, existingCategories),
	}
}

func normalizeGeneratedItems(items []AIGenerateTemplateItemResult, existingCategories []string) []AIGenerateTemplateItemResult {
	result := make([]AIGenerateTemplateItemResult, 0, len(items))
	for _, it := range items {
		it.Title = strings.TrimSpace(it.Title)
		it.Standard = strings.TrimSpace(it.Standard)
		if it.Title == "" {
			continue
		}
		it.Category = matchCategory(strings.TrimSpace(it.Category), existingCategories)
		if it.ScoreType == "" {
			it.ScoreType = "score"
		}
		if it.MaxScore <= 0 {
			it.MaxScore = 5
		}
		if len(it.ScoreOptions) == 0 {
			it.ScoreOptions = []models.ScoreOption{{Score: 0, Label: "0分"}, {Score: 5, Label: "5分"}}
		}
		if it.StandardImages == nil {
			it.StandardImages = []string{}
		}
		result = append(result, it)
	}
	return result
}

func matchCategory(category string, existingCategories []string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return "未分类"
	}
	lower := strings.ToLower(category)
	for _, c := range existingCategories {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(c)) == lower {
			return strings.TrimSpace(c)
		}
	}
	for _, c := range existingCategories {
		trimmed := strings.TrimSpace(c)
		if trimmed == "" {
			continue
		}
		lowerC := strings.ToLower(trimmed)
		if strings.Contains(lowerC, lower) || strings.Contains(lower, lowerC) {
			return trimmed
		}
	}
	return category
}

// SeedInspectionTemplatesBulk 批量创建 10 条检查表模板（每条带若干检查项），用于测试数据初始化
func SeedInspectionTemplatesBulk() error {
	o := GetOrm()
	templateNames := []string{
		"门店日常巡检模板", "新店开业验收模板", "季度品质稽核模板", "月度安全专项模板", "节假日促销巡检模板",
		"新品上市陈列模板", "门店员工服务模板", "VIP 到店接待模板", "清洁卫生专项模板", "消防设备专项模板",
	}
	categoryPool := []string{"形象", "卫生", "陈列", "服务", "安全", "仓储", "员工管理", "商品管理"}
	titlePool := map[string][]struct {
		title    string
		standard string
	}{
		"形象": {
			{title: "门头招牌", standard: "招牌干净明亮，夜间灯箱正常开启，无破损缺字"},
			{title: "橱窗展示", standard: "橱窗道具摆放整齐，无破损，画面整洁"},
			{title: "店门状态", standard: "玻璃门清洁无指纹贴，把手完好，开合顺畅"},
		},
		"卫生": {
			{title: "地面清洁", standard: "地面无污渍、无垃圾、无水渍"},
			{title: "货架清洁", standard: "货架层板无积尘、无蛛网、无废弃包装"},
			{title: "洗手间", standard: "洗手间无异味、地面干爽、厕纸与洗手液充足"},
		},
		"陈列": {
			{title: "商品堆头", standard: "堆头饱满，价格标签对齐，POP 完好"},
			{title: "价签规范", standard: "每个陈列位置均有对应价签，与商品一一对应"},
			{title: "临期商品", standard: "临期商品单独陈列并有明显提示标识"},
		},
		"服务": {
			{title: "员工着装", standard: "员工统一着装，工牌佩戴正确，仪容整洁"},
			{title: "接待用语", standard: "顾客进店 3 秒内问候，使用规范礼貌用语"},
			{title: "收银操作", standard: "收银唱收唱付，双手递接小票与商品"},
		},
		"安全": {
			{title: "灭火器", standard: "灭火器在有效期内，压力指示正常，摆放位置清晰"},
			{title: "应急照明", standard: "应急照明灯完好，断电测试可正常点亮"},
			{title: "消防通道", standard: "消防通道畅通，无商品或杂物堆放"},
		},
		"仓储": {
			{title: "仓库整洁", standard: "仓库货物分类码放，通道畅通，地面无垃圾"},
			{title: "库存标识", standard: "每个货位标识清晰，品名/数量一致"},
		},
		"员工管理": {
			{title: "考勤打卡", standard: "员工按规定到岗，考勤打卡记录正常"},
			{title: "岗位培训", standard: "在岗员工已完成对应岗位培训并通过考核"},
		},
		"商品管理": {
			{title: "库存深度", standard: "畅销品不缺货，滞销品库存合理"},
			{title: "补货及时", standard: "销售空缺位 30 分钟内完成补货"},
		},
	}

	for idx, name := range templateNames {
		// 避免重复：同名模板已存在则跳过
		existCount, _ := o.QueryTable(new(models.InspectionTemplate)).Filter("name", name).Count()
		if existCount > 0 {
			continue
		}
		now := time.Now()
		tpl := &models.InspectionTemplate{
			ID:          uuid.NewString(),
			Name:        name,
			Description: name + " - 自动生成测试数据",
			IsActive:    idx%3 != 0, // 部分停用
			ScoringMode: map[bool]string{true: "additive", false: "deductive"}[idx%2 == 0],
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		// 每条模板从分类池中取 3-5 个分类，每个分类 2-3 个检查项
		catCount := 3 + idx%3
		selectedCats := categoryPool[:catCount]
		entries := make([]*models.InspectionTemplateItem, 0)
		sortOrder := 0
		for _, cat := range selectedCats {
			options := titlePool[cat]
			itemCount := 2 + idx%2
			if itemCount > len(options) {
				itemCount = len(options)
			}
			for i := 0; i < itemCount; i++ {
				scoreType := "score"
				if (sortOrder+i)%3 == 0 {
					scoreType = "pass_fail"
				}
				maxScore := 5
				requirePhoto := (sortOrder+i)%2 == 0
				requireRemark := (sortOrder+i)%2 == 1
				entries = append(entries, &models.InspectionTemplateItem{
					ID:            uuid.NewString(),
					TemplateID:    tpl.ID,
					Category:      cat,
					Title:         options[i].title,
					Standard:      options[i].standard,
					ScoreType:     scoreType,
					MaxScore:      maxScore,
					ScoreOptions:  models.MarshalScoreOptions(models.DefaultScoreOptions(scoreType, maxScore)),
					RequireRemark: requireRemark,
					RequirePhoto:  requirePhoto,
					ShowRemark:    true,
					ShowPhoto:     true,
					SortOrder:     sortOrder,
					IsActive:      true,
					CreatedAt:     now,
					UpdatedAt:     now,
				})
				sortOrder++
			}
		}
		tx, err := o.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Insert(tpl); err != nil {
			tx.Rollback()
			return err
		}
		for _, e := range entries {
			if _, err := tx.Insert(e); err != nil {
				tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
