package controllers

import (
	"net/http"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type DashboardController struct {
	BaseController
}

var platformNames = map[string]string{
	"wechat_mp":    "微信公众号",
	"xiaohongshu":  "小红书",
	"douyin":       "抖音",
	"wechat_video": "视频号",
}

// Stats GET /api/dashboard/stats
func (c *DashboardController) Stats() {
	if !c.CheckPermission("dashboard:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	o := services.GetOrm()

	var totalAccounts int64
	_ = o.Raw("SELECT COUNT(*) FROM accounts WHERE user_id = ?", user.ID).QueryRow(&totalAccounts)

	today := time.Now().Format("2006-01-02")
	var todayPublished int64
	_ = o.Raw("SELECT COUNT(*) FROM publish_tasks t JOIN accounts a ON t.account_id = a.id WHERE a.user_id = ? AND t.status = ? AND date(t.published_at) = ?",
		user.ID, models.PublishTaskStatusPublished, today).QueryRow(&todayPublished)

	var pendingTasks int64
	_ = o.Raw("SELECT COUNT(*) FROM publish_tasks t JOIN accounts a ON t.account_id = a.id WHERE a.user_id = ? AND t.status IN (?, ?)",
		user.ID, models.PublishTaskStatusPending, models.PublishTaskStatusPublishing).QueryRow(&pendingTasks)

	var aiGeneratedCount int64
	_ = o.Raw("SELECT COUNT(*) FROM contents WHERE user_id = ? AND ai_generated = 1", user.ID).QueryRow(&aiGeneratedCount)

	platformStats := make([]map[string]interface{}, 0, len(platformNames))
	for key, name := range platformNames {
		var acctCount, activeCount, articles int64
		_ = o.Raw("SELECT COUNT(*) FROM accounts WHERE user_id = ? AND platform = ?", user.ID, key).QueryRow(&acctCount)
		_ = o.Raw("SELECT COUNT(*) FROM accounts WHERE user_id = ? AND platform = ? AND status = ?",
			user.ID, key, models.AccountStatusActive).QueryRow(&activeCount)
		_ = o.Raw("SELECT COUNT(*) FROM contents WHERE user_id = ? AND platform = ?", user.ID, key).QueryRow(&articles)
		platformStats = append(platformStats, map[string]interface{}{
			"platform": key,
			"name":     name,
			"accounts": acctCount,
			"active":   activeCount,
			"articles": articles,
		})
	}

	type recentPublish struct {
		TaskID   string `orm:"column(task_id)"`
		Title    string `orm:"column(title)"`
		Platform string `orm:"column(platform)"`
		Account  string `orm:"column(account)"`
		Status   string `orm:"column(status)"`
		Time     string `orm:"column(time)"`
	}
	var recent []recentPublish
	_, err := o.Raw(`SELECT t.id AS task_id, c.title AS title, a.platform AS platform, a.nickname AS account,
		t.status AS status, strftime('%H:%M', t.created_at) AS time
		FROM publish_tasks t
		JOIN contents c ON t.content_id = c.id
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ?
		ORDER BY t.created_at DESC
		LIMIT 5`, user.ID).QueryRows(&recent)
	if err != nil {
		recent = []recentPublish{}
	}

	recentPublishes := make([]map[string]interface{}, 0, len(recent))
	for _, r := range recent {
		recentPublishes = append(recentPublishes, map[string]interface{}{
			"id":       r.TaskID,
			"title":    r.Title,
			"platform": r.Platform,
			"account":  r.Account,
			"status":   r.Status,
			"time":     r.Time,
		})
	}

	c.OK(map[string]interface{}{
		"total_accounts":     totalAccounts,
		"today_published":    todayPublished,
		"pending_tasks":      pendingTasks,
		"ai_generated_count": aiGeneratedCount,
		"platform_stats":     platformStats,
		"recent_publishes":   recentPublishes,
	})
}
