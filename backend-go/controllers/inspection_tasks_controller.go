package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type InspectionTasksController struct {
	BaseController
}

const taskTimeFormat = "2006-01-02 15:04:05"

type inspectionTaskView struct {
	ID              string `json:"id"`
	InspectionID    string `json:"inspection_id"`
	StoreID         string `json:"store_id"`
	StoreName       string `json:"store_name"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	InspectorID     string `json:"inspector_id"`
	InspectorName   string `json:"inspector_name"`
	ResponsibleID   string `json:"responsible_id"`
	ResponsibleName string `json:"responsible_name"`
	Contact         string `json:"contact"`
	Phone           string `json:"phone"`
	Channel         string `json:"channel"`
	Deadline        string `json:"deadline"`
	ClosedAt        string `json:"closed_at"`
	ItemCount       int    `json:"item_count"`
	FixedCount      int    `json:"fixed_count"`
	Overdue         bool   `json:"overdue"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type inspectionTaskItemView struct {
	ID             string                 `json:"id"`
	TaskID         string                 `json:"task_id"`
	InspectionID   string                 `json:"inspection_id"`
	ScoreID        string                 `json:"score_id"`
	ItemID         string                 `json:"item_id"`
	ItemName       string                 `json:"item_name"`
	Category       string                 `json:"category"`
	Standard       string                 `json:"standard"`
	StandardImages []string               `json:"standard_images"`
	ScoreType      string                 `json:"score_type"`
	MaxScore       int                    `json:"max_score"`
	Score          float64                `json:"score"`
	Comment        string                 `json:"comment"`
	AISuggestion   string                 `json:"ai_suggestion"`
	AIProblemDesc  string                 `json:"ai_problem_desc"`
	OriginalPhotos []string               `json:"original_photos"`
	Status         string                 `json:"status"`
	RectifyPhotos  []string               `json:"rectify_photos"`
	RectifyComment string                 `json:"rectify_comment"`
	RecheckCount   int                    `json:"recheck_count"`
	AIResult       map[string]interface{} `json:"ai_result"`
	SubmittedAt    string                 `json:"submitted_at"`
	RecheckedAt    string                 `json:"rechecked_at"`
}

// List GET /api/inspection-tasks/
func (c *InspectionTasksController) List() {
	if !c.CheckPermission("inspection:task:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.InspectionTask))
	if storeID := c.GetQuery("store_id"); storeID != "" {
		qs = qs.Filter("store_id", storeID)
	}
	if inspectionID := c.GetQuery("inspection_id"); inspectionID != "" {
		qs = qs.Filter("inspection_id", inspectionID)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询整改任务失败")
		return
	}
	var tasks []models.InspectionTask
	if _, err := qs.OrderBy("-created_at").Limit(pageSize, (page-1)*pageSize).All(&tasks); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询整改任务失败")
		return
	}
	c.OK(map[string]interface{}{
		"items":     buildInspectionTaskViews(tasks),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get GET /api/inspection-tasks/:task_id
func (c *InspectionTasksController) Get() {
	if !c.CheckPermission("inspection:task:read") {
		return
	}
	task, err := findInspectionTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "整改任务不存在")
		return
	}
	c.OK(map[string]interface{}{
		"task":  buildInspectionTaskView(task),
		"items": loadTaskItemViews(task.ID),
		"logs":  loadTaskLogs(task.ID),
	})
}

type taskSubmitRectifyItem struct {
	ItemID  string   `json:"item_id"`
	Photos  []string `json:"photos"`
	Comment string   `json:"comment"`
}

type taskSubmitRectifyRequest struct {
	Items []taskSubmitRectifyItem `json:"items"`
}

// SubmitRectify POST /api/inspection-tasks/:task_id/submit-rectify
func (c *InspectionTasksController) SubmitRectify() {
	if !c.CheckPermission("inspection:task:update:write") {
		return
	}
	task, err := findInspectionTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "整改任务不存在")
		return
	}
	if task.Status != models.TaskStatusPending && task.Status != models.TaskStatusRectifying {
		c.WriteError(http.StatusBadRequest, "当前任务状态不可提交整改")
		return
	}
	var req taskSubmitRectifyRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if len(req.Items) == 0 {
		c.WriteError(http.StatusBadRequest, "请选择要提交整改的问题项")
		return
	}
	user := c.CurrentUser()
	o := services.GetOrm()
	items, err := loadTaskItems(task.ID)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询问题项失败")
		return
	}
	itemByID := make(map[string]*models.InspectionTaskItem, len(items))
	for i := range items {
		itemByID[items[i].ID] = &items[i]
	}
	now := time.Now()
	for _, ri := range req.Items {
		item, ok := itemByID[ri.ItemID]
		if !ok {
			c.WriteError(http.StatusBadRequest, "问题项不存在或不属于该任务")
			return
		}
		if item.Status != models.TaskItemStatusPending && item.Status != models.TaskItemStatusNotFixed {
			c.WriteError(http.StatusBadRequest, "问题项「"+item.ItemName+"」当前不可提交整改")
			return
		}
		if len(ri.Photos) == 0 {
			c.WriteError(http.StatusBadRequest, "问题项「"+item.ItemName+"」请上传整改照片")
			return
		}
		item.RectifyPhotos = marshalPhotos(ri.Photos)
		item.RectifyComment = ri.Comment
		item.Status = models.TaskItemStatusSubmitted
		item.SubmittedAt = now
		if _, err := o.Update(item); err != nil {
			c.WriteError(http.StatusInternalServerError, "保存整改信息失败")
			return
		}
	}
	task.Status = models.TaskStatusRechecking
	if _, err := o.Update(task); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新任务状态失败")
		return
	}
	syncInspectionStatusByTask(task)
	addTaskLog(task.ID, "", models.TaskLogActionSubmitted,
		fmt.Sprintf("%s 提交了 %d 个问题项的整改照片，等待 AI 复核", user.Nickname, len(req.Items)),
		user.ID, user.Nickname)
	if task.InspectorID != "" && task.InspectorID != user.ID {
		_ = createNotification(task.InspectorID, models.NotificationTypeTaskSubmitted,
			"整改已提交待复核",
			fmt.Sprintf("「%s」门店负责人已提交 %d 个问题项的整改照片，请等待 AI 复核。", task.StoreName, len(req.Items)),
			task.ID)
	}
	c.OK(map[string]interface{}{"status": task.Status, "message": "整改已提交"})
}

type taskRecheckItem struct {
	ItemID string `json:"item_id"`
}

type taskRecheckRequest struct {
	Items   []taskRecheckItem `json:"items"`
	PlanID  string            `json:"plan_id"`
	ModelID string            `json:"model_id"`
}

// Recheck POST /api/inspection-tasks/:task_id/recheck
func (c *InspectionTasksController) Recheck() {
	if !c.CheckPermission("inspection:task:recheck:write") {
		return
	}
	task, err := findInspectionTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "整改任务不存在")
		return
	}
	if task.Status != models.TaskStatusRechecking {
		c.WriteError(http.StatusBadRequest, "仅已提交整改的任务可发起 AI 复核")
		return
	}
	var req taskRecheckRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	user := c.CurrentUser()
	o := services.GetOrm()
	items, err := loadTaskItems(task.ID)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询问题项失败")
		return
	}
	itemByID := make(map[string]*models.InspectionTaskItem, len(items))
	for i := range items {
		itemByID[items[i].ID] = &items[i]
	}
	now := time.Now()
	results := make([]map[string]interface{}, 0, len(req.Items))
	checkedNames := make([]string, 0, len(req.Items))
	for _, ri := range req.Items {
		item, ok := itemByID[ri.ItemID]
		if !ok {
			c.WriteError(http.StatusBadRequest, "问题项不存在或不属于该任务")
			return
		}
		if item.Status != models.TaskItemStatusSubmitted {
			continue
		}
		result, err := services.RecheckRectifyPhoto(services.RecheckRectifyRequest{
			ItemName:        item.ItemName,
			Standard:        item.Standard,
			StandardImages:  effectiveStandardImages(item.StandardImages, item.StandardImage),
			ScoreType:       item.ScoreType,
			MaxScore:        item.MaxScore,
			OriginalComment: item.Comment,
			AISuggestion:    item.AISuggestion,
			OriginalPhotos:  toUploadedFiles(unmarshalPhotos(item.OriginalPhotos)),
			RectifyPhotos:   toUploadedFiles(unmarshalPhotos(item.RectifyPhotos)),
			PlanID:          req.PlanID,
			ModelID:         req.ModelID,
		})
		if err != nil {
			if aiErr, ok := err.(*services.AIGenerationError); ok {
				c.WriteJSON(http.StatusBadRequest, map[string]interface{}{
					"detail":          aiErr.Message,
					"available_plans": aiErr.AvailablePlans,
				})
				return
			}
			c.WriteError(http.StatusInternalServerError, "AI 复核失败")
			return
		}
		item.AIResult = marshalTaskAIResult(result)
		item.RecheckCount++
		item.RecheckedAt = now
		item.Status = models.TaskItemStatusNotFixed
		action := models.TaskLogActionRecheckFailed
		if result.Fixed {
			item.Status = models.TaskItemStatusFixed
			action = models.TaskLogActionRecheckPassed
		} else if item.RecheckCount >= models.MaxTaskItemRecheckCount {
			item.Status = models.TaskItemStatusManual
			action = models.TaskLogActionManualReview
		}
		if _, err := o.Update(item); err != nil {
			c.WriteError(http.StatusInternalServerError, "保存复核结果失败")
			return
		}
		statusText := "未修复"
		if result.Fixed {
			statusText = "已修复"
		}
		addTaskLog(task.ID, item.ID, action,
			fmt.Sprintf("「%s」AI 复核第 %d 次：%s。%s", item.ItemName, item.RecheckCount, statusText, result.Reason),
			user.ID, user.Nickname)
		results = append(results, map[string]interface{}{
			"item_id":       item.ID,
			"item_name":     item.ItemName,
			"fixed":         result.Fixed,
			"score":         result.Score,
			"reason":        result.Reason,
			"status":        item.Status,
			"recheck_count": item.RecheckCount,
		})
		checkedNames = append(checkedNames, item.ItemName)
	}
	if len(results) == 0 {
		c.WriteError(http.StatusBadRequest, "没有可复核的问题项")
		return
	}
	newStatus := summarizeTaskStatus(task.ID)
	task.Status = newStatus
	if newStatus == models.TaskStatusRectified {
		task.ClosedAt = now
	}
	if _, err := o.Update(task); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新任务状态失败")
		return
	}
	syncInspectionStatusByTask(task)
	notifyTaskRecheckResult(task, results, checkedNames)
	c.OK(map[string]interface{}{
		"status": task.Status,
		"items":  results,
	})
}

type taskManualConfirmItem struct {
	ItemID    string `json:"item_id"`
	Confirmed bool   `json:"confirmed"`
	Comment   string `json:"comment"`
}

type taskManualConfirmRequest struct {
	Items []taskManualConfirmItem `json:"items"`
}

// ManualConfirm POST /api/inspection-tasks/:task_id/manual-confirm
func (c *InspectionTasksController) ManualConfirm() {
	if !c.CheckPermission("inspection:task:confirm:write") {
		return
	}
	task, err := findInspectionTask(c.GetPathParam("task_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "整改任务不存在")
		return
	}
	if task.Status != models.TaskStatusManualReview {
		c.WriteError(http.StatusBadRequest, "仅转人工审核的任务可人工确认")
		return
	}
	var req taskManualConfirmRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if len(req.Items) == 0 {
		c.WriteError(http.StatusBadRequest, "请选择要确认的问题项")
		return
	}
	user := c.CurrentUser()
	o := services.GetOrm()
	items, err := loadTaskItems(task.ID)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询问题项失败")
		return
	}
	itemByID := make(map[string]*models.InspectionTaskItem, len(items))
	for i := range items {
		itemByID[items[i].ID] = &items[i]
	}
	now := time.Now()
	for _, ri := range req.Items {
		item, ok := itemByID[ri.ItemID]
		if !ok {
			c.WriteError(http.StatusBadRequest, "问题项不存在或不属于该任务")
			return
		}
		if item.Status != models.TaskItemStatusManual {
			c.WriteError(http.StatusBadRequest, "问题项「"+item.ItemName+"」不在人工审核状态")
			return
		}
		if ri.Confirmed {
			item.Status = models.TaskItemStatusConfirmed
			addTaskLog(task.ID, item.ID, models.TaskLogActionConfirmed,
				fmt.Sprintf("「%s」人工确认整改到位%s", item.ItemName, commentSuffix(ri.Comment)),
				user.ID, user.Nickname)
		} else {
			item.Status = models.TaskItemStatusRejected
			addTaskLog(task.ID, item.ID, models.TaskLogActionRejected,
				fmt.Sprintf("「%s」人工确认整改未到位%s", item.ItemName, commentSuffix(ri.Comment)),
				user.ID, user.Nickname)
		}
		if _, err := o.Update(item); err != nil {
			c.WriteError(http.StatusInternalServerError, "保存确认结果失败")
			return
		}
	}
	newStatus := summarizeTaskStatus(task.ID)
	task.Status = newStatus
	if newStatus == models.TaskStatusConfirmed || newStatus == models.TaskStatusRejected {
		task.ClosedAt = now
	}
	if _, err := o.Update(task); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新任务状态失败")
		return
	}
	syncInspectionStatusByTask(task)
	if task.ResponsibleID != "" {
		if newStatus == models.TaskStatusConfirmed {
			_ = createNotification(task.ResponsibleID, models.NotificationTypeTaskConfirmed,
				"整改任务已闭环",
				fmt.Sprintf("「%s」整改任务已人工确认整改到位，任务闭环完成。", task.StoreName),
				task.ID)
		} else if newStatus == models.TaskStatusRejected {
			_ = createNotification(task.ResponsibleID, models.NotificationTypeTaskRejected,
				"整改任务未通过人工确认",
				fmt.Sprintf("「%s」整改任务存在人工确认未到位的项目，请继续整改。", task.StoreName),
				task.ID)
		}
	}
	c.OK(map[string]interface{}{"status": task.Status, "message": "人工确认完成"})
}

func commentSuffix(comment string) string {
	if strings.TrimSpace(comment) == "" {
		return ""
	}
	return "，备注：" + comment
}

// syncInspectionStatusByTask 根据整改任务状态同步更新关联巡店记录状态。
func syncInspectionStatusByTask(task *models.InspectionTask) {
	if task == nil || task.InspectionID == "" {
		return
	}
	var inspection models.Inspection
	o := services.GetOrm()
	if err := o.QueryTable(new(models.Inspection)).Filter("id", task.InspectionID).One(&inspection); err != nil {
		return
	}
	var newStatus string
	switch task.Status {
	case models.TaskStatusPending:
		newStatus = models.InspectionStatusPending
	case models.TaskStatusRechecking, models.TaskStatusRectifying, models.TaskStatusManualReview:
		newStatus = models.InspectionStatusRectifying
	case models.TaskStatusRectified, models.TaskStatusConfirmed, models.TaskStatusRejected:
		newStatus = models.InspectionStatusClosed
	default:
		return
	}
	if inspection.Status != newStatus {
		inspection.Status = newStatus
		inspection.UpdatedAt = time.Now()
		_, _ = o.Update(&inspection, "status", "updated_at")
	}
}

// summarizeTaskStatus 根据任务内所有问题项状态汇总任务状态。
func summarizeTaskStatus(taskID string) string {
	items, err := loadTaskItems(taskID)
	if err != nil || len(items) == 0 {
		return models.TaskStatusPending
	}
	hasManual := false
	hasNotFixed := false
	hasSubmitted := false
	for _, it := range items {
		switch it.Status {
		case models.TaskItemStatusManual:
			hasManual = true
		case models.TaskItemStatusNotFixed:
			hasNotFixed = true
		case models.TaskItemStatusSubmitted:
			hasSubmitted = true
		}
	}
	switch {
	case hasManual:
		return models.TaskStatusManualReview
	case hasNotFixed:
		return models.TaskStatusRectifying
	case hasSubmitted:
		return models.TaskStatusRechecking
	default:
		for _, it := range items {
			if it.Status == models.TaskItemStatusRejected {
				return models.TaskStatusRejected
			}
		}
		return models.TaskStatusConfirmed
	}
}

// notifyTaskRecheckResult 复核完成后向负责人与巡店人发送结果通知。
func notifyTaskRecheckResult(task *models.InspectionTask, results []map[string]interface{}, checkedNames []string) {
	fixedCount := 0
	notFixedCount := 0
	for _, r := range results {
		if fixed, _ := r["fixed"].(bool); fixed {
			fixedCount++
		} else {
			notFixedCount++
		}
	}
	if task.ResponsibleID == "" {
		return
	}
	if notFixedCount == 0 {
		_ = createNotification(task.ResponsibleID, models.NotificationTypeTaskRecheckPassed,
			"整改已通过 AI 复核",
			fmt.Sprintf("「%s」整改任务已通过 AI 复核（%s），整改闭环完成。", task.StoreName, strings.Join(checkedNames, "、")),
			task.ID)
	} else {
		_ = createNotification(task.ResponsibleID, models.NotificationTypeTaskRecheckFailed,
			"整改未通过 AI 复核",
			fmt.Sprintf("「%s」整改任务有 %d 个问题项未通过 AI 复核，请重新整改。", task.StoreName, notFixedCount),
			task.ID)
	}
	if task.InspectorID != "" && task.InspectorID != task.ResponsibleID {
		_ = createNotification(task.InspectorID, models.NotificationTypeTaskRecheckPassed,
			"整改 AI 复核完成",
			fmt.Sprintf("「%s」整改任务 AI 复核完成：通过 %d 项，未通过 %d 项。", task.StoreName, fixedCount, notFixedCount),
			task.ID)
	}
}

func findInspectionTask(taskID string) (*models.InspectionTask, error) {
	var task models.InspectionTask
	err := services.GetOrm().QueryTable(new(models.InspectionTask)).
		Filter("id", taskID).
		One(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func loadTaskItems(taskID string) ([]models.InspectionTaskItem, error) {
	var items []models.InspectionTaskItem
	_, err := services.GetOrm().QueryTable(new(models.InspectionTaskItem)).
		Filter("task_id", taskID).
		OrderBy("created_at").
		All(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func loadTaskItemViews(taskID string) []inspectionTaskItemView {
	items, err := loadTaskItems(taskID)
	if err != nil {
		return []inspectionTaskItemView{}
	}
	views := make([]inspectionTaskItemView, 0, len(items))
	for _, it := range items {
		views = append(views, buildInspectionTaskItemView(&it))
	}
	return views
}

func buildInspectionTaskItemView(it *models.InspectionTaskItem) inspectionTaskItemView {
	return inspectionTaskItemView{
		ID:             it.ID,
		TaskID:         it.TaskID,
		InspectionID:   it.InspectionID,
		ScoreID:        it.ScoreID,
		ItemID:         it.ItemID,
		ItemName:       it.ItemName,
		Category:       it.Category,
		Standard:       it.Standard,
		StandardImages: effectiveStandardImages(it.StandardImages, it.StandardImage),
		ScoreType:      it.ScoreType,
		MaxScore:       it.MaxScore,
		Score:          it.Score,
		Comment:        it.Comment,
		AISuggestion:   it.AISuggestion,
		AIProblemDesc:  it.AIProblemDesc,
		OriginalPhotos: unmarshalPhotos(it.OriginalPhotos),
		Status:         it.Status,
		RectifyPhotos:  unmarshalPhotos(it.RectifyPhotos),
		RectifyComment: it.RectifyComment,
		RecheckCount:   it.RecheckCount,
		AIResult:       unmarshalTaskAIResult(it.AIResult),
		SubmittedAt:    formatTaskTime(it.SubmittedAt),
		RecheckedAt:    formatTaskTime(it.RecheckedAt),
	}
}

func buildProblemMapping(summary services.InspectionAISummary, scores []models.InspectionScore) (map[string]services.InspectionProblem, map[string]services.InspectionProblem) {
	byID := make(map[string]services.InspectionProblem)
	byName := make(map[string]services.InspectionProblem)
	allProblems := append(append([]services.InspectionProblem{}, summary.HighRiskProblems...), summary.MainProblems...)
	for _, p := range allProblems {
		if p.ItemID != "" {
			byID[p.ItemID] = p
		}
		if p.ItemName != "" {
			byName[p.ItemName] = p
		}
	}
	for _, s := range scores {
		if s.ItemName != "" {
			byName[s.ItemName] = services.InspectionProblem{}
		}
	}
	return byID, byName
}

func resolveAIProblemDesc(itemID, itemName string, problemByID map[string]services.InspectionProblem, problemByName map[string]services.InspectionProblem) string {
	if itemID != "" {
		if p, ok := problemByID[itemID]; ok && p.Desc != "" {
			return p.Desc
		}
	}
	if itemName != "" {
		if p, ok := problemByName[itemName]; ok && p.Desc != "" {
			return p.Desc
		}
	}
	return ""
}

func buildInspectionTaskView(task *models.InspectionTask) inspectionTaskView {
	return inspectionTaskView{
		ID:              task.ID,
		InspectionID:    task.InspectionID,
		StoreID:         task.StoreID,
		StoreName:       task.StoreName,
		Title:           task.Title,
		Status:          task.Status,
		InspectorID:     task.InspectorID,
		InspectorName:   task.InspectorName,
		ResponsibleID:   task.ResponsibleID,
		ResponsibleName: task.ResponsibleName,
		Contact:         task.Contact,
		Phone:           task.Phone,
		Channel:         task.Channel,
		Deadline:        formatTaskTime(task.Deadline),
		ClosedAt:        formatTaskTime(task.ClosedAt),
		CreatedAt:       formatTaskTime(task.CreatedAt),
		UpdatedAt:       formatTaskTime(task.UpdatedAt),
	}
}

func buildInspectionTaskViews(tasks []models.InspectionTask) []inspectionTaskView {
	views := make([]inspectionTaskView, 0, len(tasks))
	taskIDs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.ID)
	}
	counts := map[string]map[string]int{}
	if len(taskIDs) > 0 {
		var items []models.InspectionTaskItem
		if _, err := services.GetOrm().QueryTable(new(models.InspectionTaskItem)).
			Filter("task_id__in", taskIDs).
			All(&items); err == nil {
			for _, it := range items {
				if counts[it.TaskID] == nil {
					counts[it.TaskID] = map[string]int{}
				}
				counts[it.TaskID]["total"]++
				switch it.Status {
				case models.TaskItemStatusFixed, models.TaskItemStatusConfirmed:
					counts[it.TaskID]["fixed"]++
				}
			}
		}
	}
	for i := range tasks {
		view := buildInspectionTaskView(&tasks[i])
		if c, ok := counts[tasks[i].ID]; ok {
			view.ItemCount = c["total"]
			view.FixedCount = c["fixed"]
		}
		if !tasks[i].Deadline.IsZero() && tasks[i].Status != models.TaskStatusRectified &&
			tasks[i].Status != models.TaskStatusConfirmed && tasks[i].Status != models.TaskStatusRejected {
			view.Overdue = time.Now().After(tasks[i].Deadline)
		}
		views = append(views, view)
	}
	return views
}

func loadTaskLogs(taskID string) []map[string]interface{} {
	var logs []models.InspectionTaskLog
	if _, err := services.GetOrm().QueryTable(new(models.InspectionTaskLog)).
		Filter("task_id", taskID).
		OrderBy("created_at").
		All(&logs); err != nil {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(logs))
	for _, l := range logs {
		result = append(result, map[string]interface{}{
			"id":            l.ID,
			"item_id":       l.ItemID,
			"action":        l.Action,
			"content":       l.Content,
			"operator_id":   l.OperatorID,
			"operator_name": l.OperatorName,
			"created_at":    formatTaskTime(l.CreatedAt),
		})
	}
	return result
}

func addTaskLog(taskID, itemID, action, content, operatorID, operatorName string) {
	log := &models.InspectionTaskLog{
		ID:           newID(),
		TaskID:       taskID,
		ItemID:       itemID,
		Action:       action,
		Content:      content,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		CreatedAt:    time.Now(),
	}
	if _, err := services.GetOrm().Insert(log); err != nil {
		_ = err
	}
}

func formatTaskTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(taskTimeFormat)
}

func marshalTaskAIResult(r *services.RecheckRectifyResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

func unmarshalTaskAIResult(raw string) map[string]interface{} {
	if raw == "" {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}

func toUploadedFiles(photos []string) []services.UploadedFile {
	files := make([]services.UploadedFile, 0, len(photos))
	for _, p := range photos {
		if strings.HasPrefix(p, "data:") {
			if idx := strings.Index(p, ","); idx > 0 {
				header := strings.SplitN(p, ";", 2)[0]
				mimeType := strings.TrimPrefix(header, "data:")
				files = append(files, services.UploadedFile{Data: p[idx+1:], MimeType: mimeType})
			}
			continue
		}
		files = append(files, services.UploadedFile{URL: p})
	}
	return files
}

// isProblemScore 判断检查项得分是否未达标（需要整改）。
// score 类型：得分低于满分视为未达标；pass_fail 类型：未选择「合格」视为未达标。
func isProblemScore(scoreType string, score float64, maxScore int, scoreOptions string) bool {
	if scoreType == "pass_fail" {
		options := models.UnmarshalScoreOptions(scoreOptions)
		if len(options) > 0 {
			passScore := -1.0
			for _, opt := range options {
				if strings.Contains(opt.Label, "合格") {
					passScore = float64(opt.Score)
					break
				}
			}
			if passScore < 0 {
				minScore := float64(options[0].Score)
				for _, opt := range options[1:] {
					if float64(opt.Score) < minScore {
						minScore = float64(opt.Score)
					}
				}
				passScore = minScore
			}
			return score != passScore
		}
		return score < float64(maxScore)
	}
	return !isFullScore(score, maxScore)
}

// autoCreateInspectionTask 巡店保存（已提交）后自动生成整改任务。
// 幂等：同一巡店已存在任务则跳过。
func autoCreateInspectionTask(inspection *models.Inspection) {
	if inspection == nil || inspection.Status == "draft" {
		return
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.InspectionTask)).Filter("inspection_id", inspection.ID).Exist() {
		return
	}
	store, err := findStore(inspection.StoreID)
	if err != nil {
		return
	}
	var scores []models.InspectionScore
	if _, err := o.QueryTable(new(models.InspectionScore)).
		Filter("inspection_id", inspection.ID).
		OrderBy("created_at").
		All(&scores); err != nil {
		return
	}
	problemScores := make([]models.InspectionScore, 0, len(scores))
	for _, s := range scores {
		if isProblemScore(s.ScoreType, s.Score, s.MaxScore, s.ScoreOptions) {
			problemScores = append(problemScores, s)
		}
	}
	if len(problemScores) == 0 {
		return
	}
	aiSummary := services.ParseInspectionAISummary(inspection.AISummary)
	problemByID, problemByName := buildProblemMapping(aiSummary, scores)

	var inspector models.User
	inspectorName := ""
	_ = o.QueryTable(new(models.User)).Filter("id", inspection.InspectorID).One(&inspector)
	if inspector.ID != "" {
		inspectorName = inspector.Nickname
	}
	responsibleID := store.ManagerID
	responsibleName := ""
	if responsibleID != "" {
		var manager models.User
		if err := o.QueryTable(new(models.User)).Filter("id", responsibleID).One(&manager); err == nil {
			responsibleName = manager.Nickname
		}
	}
	now := time.Now()
	task := &models.InspectionTask{
		ID:              newID(),
		InspectionID:    inspection.ID,
		StoreID:         store.ID,
		StoreName:       store.Name,
		Title:           fmt.Sprintf("%s 巡店整改任务", store.Name),
		Status:          models.TaskStatusPending,
		InspectorID:     inspection.InspectorID,
		InspectorName:   inspectorName,
		ResponsibleID:   responsibleID,
		ResponsibleName: responsibleName,
		Contact:         store.Contact,
		Phone:           store.Phone,
		Channel:         models.NotificationChannelInternal,
		Deadline:        now.AddDate(0, 0, models.DefaultTaskDeadlineDays),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	tx, err := o.Begin()
	if err != nil {
		return
	}
	if _, err := tx.Insert(task); err != nil {
		tx.Rollback()
		return
	}
	for _, s := range problemScores {
		aiProblemDesc := resolveAIProblemDesc(s.ItemID, s.ItemName, problemByID, problemByName)
		images := effectiveStandardImages(s.StandardImages, s.StandardImage)
		item := &models.InspectionTaskItem{
			ID:             newID(),
			TaskID:         task.ID,
			InspectionID:   inspection.ID,
			ScoreID:        s.ID,
			ItemID:         s.ItemID,
			ItemName:       s.ItemName,
			Category:       s.Category,
			Standard:       s.Standard,
			StandardImages: models.MarshalStringSlice(images),
			StandardImage:  firstImage(images),
			ScoreType:      s.ScoreType,
			MaxScore:       s.MaxScore,
			Score:          s.Score,
			Comment:        s.Comment,
			AISuggestion:   s.AISuggestion,
			AIProblemDesc:  aiProblemDesc,
			OriginalPhotos: s.Photos,
			Status:         models.TaskItemStatusPending,
			CreatedAt:      now,
		}
		if _, err := tx.Insert(item); err != nil {
			tx.Rollback()
			return
		}
	}
	if err := tx.Commit(); err != nil {
		return
	}
	itemNames := make([]string, 0, len(problemScores))
	for _, s := range problemScores {
		itemNames = append(itemNames, s.ItemName)
	}
	addTaskLog(task.ID, "", models.TaskLogActionCreated,
		fmt.Sprintf("巡店完成，自动生成整改任务，共 %d 个问题项：%s", len(problemScores), strings.Join(itemNames, "、")),
		inspection.InspectorID, inspectorName)
	if responsibleID != "" {
		_ = createNotification(responsibleID, models.NotificationTypeTaskCreated,
			"新的巡店整改任务",
			fmt.Sprintf("「%s」巡店发现 %d 个问题项（%s），请在 %s 前完成整改并提交整改照片。",
				store.Name, len(problemScores), strings.Join(itemNames, "、"), task.Deadline.Format(taskTimeFormat)),
			task.ID)
	}
}
