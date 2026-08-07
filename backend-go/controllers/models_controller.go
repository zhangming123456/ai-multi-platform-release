package controllers

import (
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

type fetchModelsRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// Fetch POST /api/models/fetch
func (c *ModelsController) Fetch() {
	if !c.CheckPermission("token_plan:read") {
		return
	}
	var req fetchModelsRequest
	if err := c.ParseBody(&req); err != nil || req.BaseURL == "" {
		c.WriteError(http.StatusBadRequest, "base_url 不能为空")
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
			models = append(models, m.ID)
		}
	}
	sort.Strings(models)
	c.OK(map[string]interface{}{"models": models})
}
