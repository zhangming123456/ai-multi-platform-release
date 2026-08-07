package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type UserCreationReviewsController struct {
	BaseController
}

type rejectUserCreationRequest struct {
	Reason string `json:"reason"`
}

func userCreationItem(req *models.UserCreationRequest, requesterName, reviewerName string) map[string]interface{} {
	return map[string]interface{}{
		"id":             req.ID,
		"requester_id":   req.RequesterID,
		"requester_name": requesterName,
		"username":       req.Username,
		"email":          req.Email,
		"nickname":       req.Nickname,
		"role":           req.Role,
		"status":         req.Status,
		"reviewer_id":    req.ReviewerID,
		"reviewer_name":  reviewerName,
		"reject_reason":  req.RejectReason,
		"created_at":     req.CreatedAt.Format("2006-01-02T15:04:05"),
	}
}

// List GET /api/user-creation-reviews/
func (c *UserCreationReviewsController) List() {
	if !c.CheckPermission("review:read") {
		return
	}
	o := services.GetOrm()
	var reqs []models.UserCreationRequest
	_, err := o.QueryTable(new(models.UserCreationRequest)).
		Filter("status", models.UserCreationStatusPending).
		OrderBy("-created_at").
		All(&reqs)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询审核列表失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(reqs))
	for i := range reqs {
		items = append(items, userCreationItem(&reqs[i], usernameByID(reqs[i].RequesterID), usernameByID(reqs[i].ReviewerID)))
	}
	c.OK(items)
}

// Approve POST /api/user-creation-reviews/:id/approve
func (c *UserCreationReviewsController) Approve() {
	if !c.CheckPermission("review:approve:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	requestID := c.GetPathParam("id")
	o := services.GetOrm()
	var req models.UserCreationRequest
	if err := o.QueryTable(new(models.UserCreationRequest)).Filter("id", requestID).One(&req); err != nil {
		c.WriteError(http.StatusNotFound, "审核请求不存在")
		return
	}
	if req.Status != models.UserCreationStatusPending {
		c.WriteError(http.StatusBadRequest, "该请求已被处理")
		return
	}
	if o.QueryTable(new(models.User)).Filter("username", req.Username).Exist() {
		c.WriteError(http.StatusBadRequest, "用户名 "+req.Username+" 已存在")
		return
	}
	newUser := &models.User{
		ID:             newID(),
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: req.HashedPassword,
		Nickname:       req.Nickname,
		Role:           req.Role,
		AvatarURL:      req.AvatarURL,
	}
	if _, err := o.Insert(newUser); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建用户失败")
		return
	}
	req.Status = models.UserCreationStatusApproved
	req.ReviewerID = user.ID
	if _, err := o.Update(&req); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新审核请求失败")
		return
	}
	c.OK(userCreationItem(&req, usernameByID(req.RequesterID), user.Nickname))
}

// Reject POST /api/user-creation-reviews/:id/reject
func (c *UserCreationReviewsController) Reject() {
	if !c.CheckPermission("review:reject:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var body rejectUserCreationRequest
	if err := c.ParseBody(&body); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	requestID := c.GetPathParam("id")
	o := services.GetOrm()
	var req models.UserCreationRequest
	if err := o.QueryTable(new(models.UserCreationRequest)).Filter("id", requestID).One(&req); err != nil {
		c.WriteError(http.StatusNotFound, "审核请求不存在")
		return
	}
	if req.Status != models.UserCreationStatusPending {
		c.WriteError(http.StatusBadRequest, "该请求已被处理")
		return
	}
	req.Status = models.UserCreationStatusRejected
	req.ReviewerID = user.ID
	req.RejectReason = body.Reason
	if _, err := o.Update(&req); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新审核请求失败")
		return
	}
	c.OK(userCreationItem(&req, usernameByID(req.RequesterID), user.Nickname))
}
