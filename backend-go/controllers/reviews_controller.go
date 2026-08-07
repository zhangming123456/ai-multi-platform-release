package controllers

import (
	"fmt"
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type ReviewsController struct {
	BaseController
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func reviewResponse(content *models.Content, username string) map[string]interface{} {
	return map[string]interface{}{
		"id":         content.ID,
		"title":      content.Title,
		"body":       content.Body,
		"platform":   content.Platform,
		"status":     content.Status,
		"created_at": content.CreatedAt,
		"user_id":    content.UserID,
		"username":   username,
	}
}

func usernameByID(userID string) string {
	var user models.User
	err := services.GetOrm().QueryTable(new(models.User)).Filter("id", userID).One(&user)
	if err != nil {
		return ""
	}
	return user.Username
}

func findAnyContent(contentID string) (*models.Content, error) {
	var content models.Content
	err := services.GetOrm().QueryTable(new(models.Content)).
		Filter("id", contentID).
		One(&content)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// List GET /api/reviews/
func (c *ReviewsController) List() {
	if !c.CheckPermission("review:read") {
		return
	}
	var contents []models.Content
	_, err := services.GetOrm().QueryTable(new(models.Content)).
		Filter("status", models.ContentStatusPendingReview).
		OrderBy("-created_at").
		All(&contents)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询审核列表失败")
		return
	}
	result := make([]map[string]interface{}, 0, len(contents))
	for i := range contents {
		result = append(result, reviewResponse(&contents[i], usernameByID(contents[i].UserID)))
	}
	c.OK(result)
}

// Submit POST /api/reviews/:content_id/submit
func (c *ReviewsController) Submit() {
	if !c.CheckPermission("review:submit:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	content, err := findAnyContent(c.GetPathParam("content_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	if content.UserID != user.ID {
		c.WriteError(http.StatusForbidden, "只能提交自己的内容")
		return
	}
	if content.Status != models.ContentStatusDraft && content.Status != models.ContentStatusRejected {
		c.WriteError(http.StatusBadRequest, "当前状态不允许提交审核")
		return
	}
	content.Status = models.ContentStatusPendingReview
	if _, err := services.GetOrm().Update(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "提交审核失败")
		return
	}

	reviewerIDs := userIDsWithPermission("review:approve")
	for _, rid := range reviewerIDs {
		_ = createNotification(rid, "review_submit", "新内容待审核",
			fmt.Sprintf("%s 提交了内容「%s」待审核", user.Nickname, content.Title), content.ID)
	}

	c.OK(reviewResponse(content, user.Username))
}

// Approve POST /api/reviews/:content_id/approve
func (c *ReviewsController) Approve() {
	if !c.CheckPermission("review:approve:write") {
		return
	}
	content, err := findAnyContent(c.GetPathParam("content_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	if content.Status != models.ContentStatusPendingReview {
		c.WriteError(http.StatusBadRequest, "内容不在待审核状态")
		return
	}
	content.Status = models.ContentStatusReady
	if _, err := services.GetOrm().Update(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "审核操作失败")
		return
	}

	_ = createNotification(content.UserID, "review_approved", "内容审核通过",
		fmt.Sprintf("您的内容「%s」已通过审核", content.Title), content.ID)

	c.OK(reviewResponse(content, usernameByID(content.UserID)))
}

// Reject POST /api/reviews/:content_id/reject
func (c *ReviewsController) Reject() {
	if !c.CheckPermission("review:reject:write") {
		return
	}
	content, err := findAnyContent(c.GetPathParam("content_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "内容不存在")
		return
	}
	if content.Status != models.ContentStatusPendingReview {
		c.WriteError(http.StatusBadRequest, "内容不在待审核状态")
		return
	}
	var req rejectRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "驳回原因不能为空")
		return
	}
	content.Status = models.ContentStatusRejected
	if _, err := services.GetOrm().Update(content); err != nil {
		c.WriteError(http.StatusInternalServerError, "审核操作失败")
		return
	}

	_ = createNotification(content.UserID, "review_rejected", "内容审核驳回",
		fmt.Sprintf("您的内容「%s」已被驳回，原因：%s", content.Title, req.Reason), content.ID)

	c.OK(reviewResponse(content, usernameByID(content.UserID)))
}
