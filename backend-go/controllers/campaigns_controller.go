package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type campaignQueryFilter struct {
	Keyword   string
	Platform  string
	Status    string
	Location  string
	StartDate string
	EndDate   string
}

func queryCampaignFilter(c *CampaignsController, forceStatus string) campaignQueryFilter {
	f := campaignQueryFilter{
		Keyword:   c.GetQuery("keyword"),
		Platform:  c.GetQuery("platform"),
		Location:  c.GetQuery("location"),
		StartDate: c.GetQuery("start_date"),
		EndDate:   c.GetQuery("end_date"),
	}
	if forceStatus != "" {
		f.Status = forceStatus
	} else {
		f.Status = c.GetQuery("status")
	}
	return f
}

func buildCampaignCond(userID string, f campaignQueryFilter) *orm.Condition {
	cond := orm.NewCondition().And("user_id", userID)
	if keyword := strings.TrimSpace(f.Keyword); keyword != "" {
		kw := orm.NewCondition().Or("name__icontains", keyword).Or("description__icontains", keyword)
		cond = cond.AndCond(kw)
	}
	if platform := strings.TrimSpace(f.Platform); platform != "" {
		cond = cond.And("platforms__icontains", "\""+platform+"\"")
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		cond = cond.And("status", status)
	}
	if location := strings.TrimSpace(f.Location); location != "" {
		cond = cond.And("location__icontains", location)
	}
	if v := strings.TrimSpace(f.StartDate); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			cond = cond.And("created_at__gte", t)
		}
	}
	if v := strings.TrimSpace(f.EndDate); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			cond = cond.And("created_at__lt", t.AddDate(0, 0, 1))
		}
	}
	return cond
}

type CampaignsController struct {
	BaseController
}

type campaignCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MediaURLs   []string `json:"media_urls"`
	Location    string   `json:"location"`
	Platforms   []string `json:"platforms"`
}

type campaignUpdateRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	MediaURLs   *[]string `json:"media_urls"`
	Location    *string   `json:"location"`
	Platforms   *[]string `json:"platforms"`
	Status      *string   `json:"status"`
}

func marshalStringList(list []string) string {
	if len(list) == 0 {
		return ""
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func campaignMap(c *models.Campaign) map[string]interface{} {
	mediaURLs := []string{}
	platforms := []string{}
	if c.MediaURLs != "" {
		_ = json.Unmarshal([]byte(c.MediaURLs), &mediaURLs)
	}
	if c.Platforms != "" {
		_ = json.Unmarshal([]byte(c.Platforms), &platforms)
	}
	return map[string]interface{}{
		"id":          c.ID,
		"user_id":     c.UserID,
		"name":        c.Name,
		"description": c.Description,
		"media_urls":  mediaURLs,
		"location":    c.Location,
		"platforms":   platforms,
		"status":      c.Status,
		"created_at":  c.CreatedAt,
		"updated_at":  c.UpdatedAt,
	}
}

func findOwnCampaign(campaignID, userID string) (*models.Campaign, error) {
	var campaign models.Campaign
	err := services.GetOrm().QueryTable(new(models.Campaign)).
		Filter("id", campaignID).
		Filter("user_id", userID).
		One(&campaign)
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// List GET /api/campaigns/
func (c *CampaignsController) List() {
	if !c.CheckPermission("campaign:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	cond := buildCampaignCond(user.ID, queryCampaignFilter(c, ""))
	qs := services.GetOrm().QueryTable(new(models.Campaign)).SetCond(cond)
	count, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询活动失败")
		return
	}
	var campaigns []models.Campaign
	_, err = qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&campaigns)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询活动失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(campaigns))
	for i := range campaigns {
		items = append(items, campaignMap(&campaigns[i]))
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     count,
		"page":      page,
		"page_size": pageSize,
	})
}

// Create POST /api/campaigns/
func (c *CampaignsController) Create() {
	if !c.CheckPermission("campaign:create:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req campaignCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.WriteError(http.StatusBadRequest, "活动名称不能为空")
		return
	}
	campaign := &models.Campaign{
		ID:          newID(),
		UserID:      user.ID,
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		MediaURLs:   marshalStringList(req.MediaURLs),
		Location:    req.Location,
		Platforms:   marshalStringList(req.Platforms),
		Status:      models.CampaignStatusActive,
	}
	if _, err := services.GetOrm().Insert(campaign); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建活动失败")
		return
	}
	c.Created(campaignMap(campaign))
}

// Get GET /api/campaigns/:campaign_id
func (c *CampaignsController) Get() {
	if !c.CheckPermission("campaign:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	campaign, err := findOwnCampaign(c.GetPathParam("campaign_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "活动不存在")
		return
	}
	c.OK(campaignMap(campaign))
}

// Update PUT /api/campaigns/:campaign_id
func (c *CampaignsController) Update() {
	if !c.CheckPermission("campaign:update:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	campaign, err := findOwnCampaign(c.GetPathParam("campaign_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "活动不存在")
		return
	}
	var req campaignUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体无效")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.WriteError(http.StatusBadRequest, "活动名称不能为空")
			return
		}
		campaign.Name = name
	}
	if req.Description != nil {
		campaign.Description = *req.Description
	}
	if req.MediaURLs != nil {
		campaign.MediaURLs = marshalStringList(*req.MediaURLs)
	}
	if req.Location != nil {
		campaign.Location = *req.Location
	}
	if req.Platforms != nil {
		campaign.Platforms = marshalStringList(*req.Platforms)
	}
	if req.Status != nil {
		if *req.Status != models.CampaignStatusActive && *req.Status != models.CampaignStatusArchived {
			c.WriteError(http.StatusBadRequest, "无效的活动状态")
			return
		}
		campaign.Status = *req.Status
	}
	if _, err := services.GetOrm().Update(campaign); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新活动失败")
		return
	}
	c.OK(campaignMap(campaign))
}

// Delete DELETE /api/campaigns/:campaign_id
func (c *CampaignsController) Delete() {
	if !c.CheckPermission("campaign:delete:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	campaign, err := findOwnCampaign(c.GetPathParam("campaign_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "活动不存在")
		return
	}
	campaign.Status = models.CampaignStatusArchived
	if _, err := services.GetOrm().Update(campaign); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除活动失败")
		return
	}
	c.WriteNoContent()
}

// Options GET /api/campaigns/options
func (c *CampaignsController) Options() {
	if !c.CheckPermission("campaign:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	cond := buildCampaignCond(user.ID, queryCampaignFilter(c, ""))
	qs := services.GetOrm().QueryTable(new(models.Campaign)).SetCond(cond)
	count, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询活动失败")
		return
	}
	var campaigns []models.Campaign
	_, err = qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&campaigns)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询活动失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(campaigns))
	for i := range campaigns {
		items = append(items, campaignMap(&campaigns[i]))
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     count,
		"page":      page,
		"page_size": pageSize,
	})
}
