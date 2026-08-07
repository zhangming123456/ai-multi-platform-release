package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type DBChangesController struct {
	BaseController
}

type submitChangeRequest struct {
	SQL         string `json:"sql"`
	ChangeType  string `json:"change_type"`
	Description string `json:"description"`
}

type rejectChangeRequest struct {
	Reason string `json:"reason"`
}

func sqlChangeItem(req *models.SqlChangeRequest, requesterName string) map[string]interface{} {
	approvedBy := []string{}
	if req.ApprovedBy != "" {
		for _, x := range strings.Split(req.ApprovedBy, ",") {
			if strings.TrimSpace(x) != "" {
				approvedBy = append(approvedBy, x)
			}
		}
	}
	return map[string]interface{}{
		"id":                 req.ID,
		"requester_id":       req.RequesterID,
		"requester_name":     requesterName,
		"change_type":        req.ChangeType,
		"sql_text":           req.SQLText,
		"description":        req.Description,
		"status":             req.Status,
		"approvals":          req.Approvals,
		"required_approvals": req.RequiredApprovals,
		"approved_by":        approvedBy,
		"reject_reason":      req.RejectReason,
		"execute_message":    req.ExecuteMessage,
		"created_at":         req.CreatedAt.Format("2006-01-02T15:04:05"),
	}
}

// Submit POST /api/db-changes/submit
func (c *DBChangesController) Submit() {
	if !c.CheckPermission("db_change:submit:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req submitChangeRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	sql := strings.TrimSpace(req.SQL)
	if sql == "" {
		c.WriteError(http.StatusBadRequest, "SQL 命令不能为空")
		return
	}
	if req.ChangeType != models.SqlChangeTypeUpdate &&
		req.ChangeType != models.SqlChangeTypeDelete &&
		req.ChangeType != models.SqlChangeTypeRowDelete {
		c.WriteError(http.StatusBadRequest, "不支持的变更类型")
		return
	}

	change := &models.SqlChangeRequest{
		ID:                newID(),
		RequesterID:       user.ID,
		ChangeType:        req.ChangeType,
		SQLText:           sql,
		Description:       req.Description,
		Status:            models.SqlChangeStatusPending,
		RequiredApprovals: 2,
	}
	if _, err := services.GetOrm().Insert(change); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建审核请求失败")
		return
	}

	reviewerIDs := userIDsWithPermission("db_change:approve:write")
	typeLabel := "删除"
	if req.ChangeType == models.SqlChangeTypeUpdate {
		typeLabel = "修改"
	}
	preview := sql
	if len(preview) > 100 {
		preview = preview[:100]
	}
	for _, rid := range reviewerIDs {
		_ = createNotification(rid, "review_submit", "SQL 变更待审核", user.Nickname+" 提交了一条 SQL "+typeLabel+"操作待审核："+preview, change.ID)
	}
	c.OK(sqlChangeItem(change, user.Nickname))
}

// List GET /api/db-changes/
func (c *DBChangesController) List() {
	if !c.CheckPermission("sql_review:read") {
		return
	}
	statusFilter := c.GetQuery("status")
	if statusFilter == "" {
		statusFilter = models.SqlChangeStatusPending
	}
	o := services.GetOrm()
	var changes []models.SqlChangeRequest
	_, err := o.QueryTable(new(models.SqlChangeRequest)).
		Filter("status", statusFilter).
		OrderBy("-created_at").
		All(&changes)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询审核列表失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(changes))
	for i := range changes {
		items = append(items, sqlChangeItem(&changes[i], usernameByID(changes[i].RequesterID)))
	}
	c.OK(map[string]interface{}{"items": items, "total": len(items)})
}

// Approve POST /api/db-changes/:id/approve
func (c *DBChangesController) Approve() {
	if !c.CheckPermission("db_change:approve:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	changeID := c.GetPathParam("id")
	o := services.GetOrm()
	var change models.SqlChangeRequest
	if err := o.QueryTable(new(models.SqlChangeRequest)).Filter("id", changeID).One(&change); err != nil {
		c.WriteError(http.StatusNotFound, "审核请求不存在")
		return
	}
	if change.Status != models.SqlChangeStatusPending && change.Status != models.SqlChangeStatusApproved {
		c.WriteError(http.StatusBadRequest, "该请求已被处理")
		return
	}
	if change.RequesterID == user.ID {
		c.WriteError(http.StatusBadRequest, "不能审核自己提交的请求")
		return
	}
	approvedIDs := []string{}
	if change.ApprovedBy != "" {
		for _, x := range strings.Split(change.ApprovedBy, ",") {
			if strings.TrimSpace(x) != "" {
				approvedIDs = append(approvedIDs, x)
			}
		}
	}
	for _, id := range approvedIDs {
		if id == user.ID {
			c.WriteError(http.StatusBadRequest, "您已审核过该请求")
			return
		}
	}
	approvedIDs = append(approvedIDs, user.ID)
	change.ApprovedBy = strings.Join(approvedIDs, ",")
	change.Approvals = len(approvedIDs)

	if change.Approvals >= change.RequiredApprovals {
		success, msg := executeChangeSQL(&change)
		if success {
			change.Status = models.SqlChangeStatusExecuted
			change.ExecuteMessage = msg
			_ = createNotification(change.RequesterID, "review_approved", "SQL 变更已执行",
				"您提交的 SQL 变更已通过审核并自动执行成功："+truncateString(change.SQLText, 80), change.ID)
		} else {
			change.Status = models.SqlChangeStatusExecuteFailed
			change.ExecuteMessage = msg
			_ = createNotification(change.RequesterID, "review_approved", "SQL 变更执行失败",
				"您提交的 SQL 变更审核通过但执行失败："+msg, change.ID)
		}
		_, _ = o.Insert(&models.SqlHistory{
			ID:        newID(),
			UserID:    change.RequesterID,
			SQLText:   change.SQLText,
			IsSuccess: success,
			Message:   msg,
		})
	} else {
		change.Status = models.SqlChangeStatusApproved
		_ = createNotification(change.RequesterID, "review_approved", "SQL 变更审核进度",
			fmt.Sprintf("您提交的 SQL 变更已获得 %d/%d 人审核通过", change.Approvals, change.RequiredApprovals), change.ID)
	}
	if _, err := o.Update(&change); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新审核请求失败")
		return
	}
	c.OK(sqlChangeItem(&change, usernameByID(change.RequesterID)))
}

// Reject POST /api/db-changes/:id/reject
func (c *DBChangesController) Reject() {
	if !c.CheckPermission("db_change:reject:write") {
		return
	}
	var req rejectChangeRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	changeID := c.GetPathParam("id")
	o := services.GetOrm()
	var change models.SqlChangeRequest
	if err := o.QueryTable(new(models.SqlChangeRequest)).Filter("id", changeID).One(&change); err != nil {
		c.WriteError(http.StatusNotFound, "审核请求不存在")
		return
	}
	if change.Status != models.SqlChangeStatusPending {
		c.WriteError(http.StatusBadRequest, "该请求已被处理")
		return
	}
	change.Status = models.SqlChangeStatusRejected
	change.RejectReason = req.Reason
	if change.RejectReason == "" {
		change.RejectReason = "未填写原因"
	}
	_ = createNotification(change.RequesterID, "review_rejected", "SQL 变更被驳回",
		"您提交的 SQL 变更被驳回，原因："+change.RejectReason, change.ID)
	if _, err := o.Update(&change); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新审核请求失败")
		return
	}
	c.OK(sqlChangeItem(&change, usernameByID(change.RequesterID)))
}

func executeChangeSQL(change *models.SqlChangeRequest) (bool, string) {
	db, err := services.GetSQLDB()
	if err != nil {
		return false, "执行失败: " + err.Error()
	}
	if _, err := db.Exec(change.SQLText); err != nil {
		return false, "执行失败: " + err.Error()
	}
	return true, "命令执行成功"
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
