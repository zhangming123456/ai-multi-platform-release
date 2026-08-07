package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type PublishController struct {
	BaseController
}

type publishTaskCreateRequest struct {
	ContentID   string `json:"content_id"`
	AccountID   string `json:"account_id"`
	ScheduledAt string `json:"scheduled_at"`
}

// ListTasks GET /api/publish/tasks
func (c *PublishController) ListTasks() {
	if !c.CheckPermission("publish:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	page, pageSize := c.ParsePagination()
	var accounts []models.Account
	_, _ = services.GetOrm().QueryTable(new(models.Account)).
		Filter("user_id", user.ID).
		All(&accounts, "id")
	accountIDs := make([]string, 0, len(accounts))
	for _, a := range accounts {
		accountIDs = append(accountIDs, a.ID)
	}
	if len(accountIDs) == 0 {
		c.OK([]models.PublishTask{})
		return
	}
	qs := services.GetOrm().QueryTable(new(models.PublishTask)).
		Filter("account_id__in", accountIDs)
	if statusFilter := c.GetQuery("status"); statusFilter != "" {
		qs = qs.Filter("status", statusFilter)
	}
	var tasks []models.PublishTask
	_, err := qs.OrderBy("-created_at").Limit(pageSize).Offset((page - 1) * pageSize).All(&tasks)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询发布任务失败")
		return
	}
	c.OK(tasks)
}

// CreateTask POST /api/publish/tasks
func (c *PublishController) CreateTask() {
	if !c.CheckPermission("publish:create:write") {
		return
	}
	var req publishTaskCreateRequest
	if err := c.ParseBody(&req); err != nil || req.ContentID == "" || req.AccountID == "" {
		c.WriteError(http.StatusBadRequest, "content_id 和 account_id 不能为空")
		return
	}
	scheduled, err := parseOptionalTime(req.ScheduledAt)
	if err != nil {
		c.WriteError(http.StatusBadRequest, "scheduled_at 时间格式错误")
		return
	}
	task := &models.PublishTask{
		ID:          newID(),
		ContentID:   req.ContentID,
		AccountID:   req.AccountID,
		Status:      models.PublishTaskStatusPending,
		ScheduledAt: scheduled,
	}
	if _, err := services.GetOrm().Insert(task); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建发布任务失败")
		return
	}
	c.Created(task)
}

func findPublishTask(taskID string) (*models.PublishTask, error) {
	var task models.PublishTask
	err := services.GetOrm().QueryTable(new(models.PublishTask)).
		Filter("id", taskID).
		One(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTask GET /api/publish/tasks/:task_id
func (c *PublishController) GetTask() {
	if !c.CheckPermission("publish:read") {
		return
	}
	task, err := findPublishTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "任务不存在")
		return
	}
	c.OK(task)
}

// RetryTask POST /api/publish/tasks/:task_id/retry
func (c *PublishController) RetryTask() {
	if !c.CheckPermission("publish:retry:write") {
		return
	}
	task, err := findPublishTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "任务不存在或状态不允许重试")
		return
	}
	if task.Status != models.PublishTaskStatusFailed {
		c.WriteError(http.StatusNotFound, "任务不存在或状态不允许重试")
		return
	}
	task.Status = models.PublishTaskStatusPending
	task.RetryCount++
	task.ErrorMessage = ""
	if _, err := services.GetOrm().Update(task); err != nil {
		c.WriteError(http.StatusInternalServerError, "重试任务失败")
		return
	}
	c.OK(task)
}

// Stats GET /api/publish/stats
func (c *PublishController) Stats() {
	if !c.CheckPermission("publish:read") {
		return
	}
	o := services.GetOrm()
	total, _ := o.QueryTable(new(models.PublishTask)).Count()
	pending, _ := o.QueryTable(new(models.PublishTask)).Filter("status", models.PublishTaskStatusPending).Count()
	publishing, _ := o.QueryTable(new(models.PublishTask)).Filter("status", models.PublishTaskStatusPublishing).Count()
	published, _ := o.QueryTable(new(models.PublishTask)).Filter("status", models.PublishTaskStatusPublished).Count()
	failed, _ := o.QueryTable(new(models.PublishTask)).Filter("status", models.PublishTaskStatusFailed).Count()
	c.OK(map[string]interface{}{
		"total":      total,
		"pending":    pending,
		"publishing": publishing,
		"published":  published,
		"failed":     failed,
	})
}
