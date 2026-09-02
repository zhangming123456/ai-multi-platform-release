package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RecheckRectifyRequest AI 二次复核请求。
type RecheckRectifyRequest struct {
	ItemName        string         `json:"item_name"`
	Standard        string         `json:"standard"`
	StandardImages  []string       `json:"standard_images"`
	ScoreType       string         `json:"score_type"`
	MaxScore        int            `json:"max_score"`
	OriginalComment string         `json:"original_comment"`
	AISuggestion    string         `json:"ai_suggestion"`
	OriginalPhotos  []UploadedFile `json:"original_photos"`
	RectifyPhotos   []UploadedFile `json:"rectify_photos"`
	PlanID          string         `json:"plan_id"`
	ModelID         string         `json:"model_id"`
}

// RecheckRectifyResult AI 二次复核结果。
type RecheckRectifyResult struct {
	Fixed  bool    `json:"fixed"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// RecheckRectifyPhoto 复核整改照片，判断问题是否已修复。
// 以整改照片为核心依据，结合检查标准与巡店原问题判定，返回 fixed 结论与理由。
func RecheckRectifyPhoto(req RecheckRectifyRequest) (*RecheckRectifyResult, error) {
	apiKey, baseURL, model, planName, supportsVision, apiFormat, err := ResolveModelConfig(req.PlanID, req.ModelID)
	if err != nil {
		return nil, err
	}

	standardLine := ""
	if req.Standard != "" {
		standardLine = fmt.Sprintf("检查标准：%s\n", req.Standard)
	}

	originalLine := ""
	if req.OriginalComment != "" {
		originalLine = fmt.Sprintf("原问题描述：%s\n", req.OriginalComment)
	}

	suggestionLine := ""
	if req.AISuggestion != "" {
		suggestionLine = fmt.Sprintf("整改建议：%s\n", req.AISuggestion)
	}

	photoDesc := ""
	if len(req.RectifyPhotos) > 0 {
		photoDesc = fmt.Sprintf("整改照片 %d 张，请重点核对整改照片中是否已解决原问题。", len(req.RectifyPhotos))
	}

	prompt := fmt.Sprintf(`你是一名专业的连锁门店巡店督导复核员。请复核以下整改结果，判断原问题是否已修复。

检查项：%s
%s%s%s
%s

请对比整改照片与检查标准，客观判断该检查项问题是否已修复，并给出修复后的评分（满分 %d 分）。若无法从照片确认已修复，应判定为未修复。严格仅返回 JSON（不要输出其他内容）：
{
  "fixed": true或false（是否已修复）,
  "score": 修复后的评分（0 ~ %d）,
  "reason": "修复判断理由，说明依据整改照片观察到的情况"
}`, req.ItemName, standardLine, originalLine, suggestionLine, photoDesc, req.MaxScore, req.MaxScore)

	messages := []chatMessage{{
		Role:    "system",
		Content: "你是专业的连锁门店巡店督导复核员，擅长通过照片对比检查标准判断整改是否到位。使用中文回复。",
	}}

	standardImages := make([]UploadedFile, 0, 1)
	for _, img := range req.StandardImages {
		if img == "" {
			continue
		}
		if f, err := loadImageAsUploadedFile(img); err == nil {
			standardImages = append(standardImages, *f)
		}
	}
	if supportsVision && (len(standardImages) > 0 || len(req.RectifyPhotos) > 0) {
		messages = append(messages, chatMessage{Role: "user", Content: buildInspectionVisionContent(prompt, standardImages, req.RectifyPhotos)})
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
		if content == "" {
			return nil, aiCallError(planName, fmt.Errorf("模型无返回内容"))
		}
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
		content = strings.TrimSpace(raw.Choices[0].Message.Content)
	}
	if strings.HasPrefix(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) > 1 {
			content = strings.TrimSpace(parts[1])
			content = strings.TrimPrefix(content, "json")
			content = strings.TrimSpace(content)
		}
	}

	var result RecheckRectifyResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		available, aerr := getAvailablePlans()
		if aerr != nil {
			return nil, aerr
		}
		return nil, &AIGenerationError{
			Message:        fmt.Sprintf("模型「%s」返回的复核结果格式异常，无法解析。请尝试切换到其他模型配置后重试。", planName),
			AvailablePlans: available,
		}
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > float64(req.MaxScore) {
		result.Score = float64(req.MaxScore)
	}
	return &result, nil
}
