package controllers

import (
	"net/http"
	"time"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type AccountsController struct {
	BaseController
}

type accountCreateRequest struct {
	Platform       string `json:"platform"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url"`
	CookieData     string `json:"cookie_data"`
	AccessToken    string `json:"access_token"`
	TokenExpiresAt string `json:"token_expires_at"`
}

type accountUpdateRequest struct {
	Nickname       *string `json:"nickname"`
	AvatarURL      *string `json:"avatar_url"`
	Status         *string `json:"status"`
	CookieData     *string `json:"cookie_data"`
	AccessToken    *string `json:"access_token"`
	TokenExpiresAt *string `json:"token_expires_at"`
}

func parseOptionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, TimeFormat, "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, errTimeFormat
}

var errTimeFormat = timeErr("时间格式错误")

type timeErr string

func (e timeErr) Error() string { return string(e) }

func findOwnAccount(accountID, userID string) (*models.Account, error) {
	var account models.Account
	err := services.GetOrm().QueryTable(new(models.Account)).
		Filter("id", accountID).
		Filter("user_id", userID).
		One(&account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// List GET /api/accounts/
func (c *AccountsController) List() {
	if !c.CheckPermission("accounts:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	qs := services.GetOrm().QueryTable(new(models.Account)).
		Filter("user_id", user.ID)
	if platform := c.GetQuery("platform"); platform != "" {
		qs = qs.Filter("platform", platform)
	}
	var accounts []models.Account
	if _, err := qs.OrderBy("-created_at").All(&accounts); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询账号失败")
		return
	}
	c.OK(accounts)
}

// Create POST /api/accounts/
func (c *AccountsController) Create() {
	if !c.CheckPermission("account:create:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req accountCreateRequest
	if err := c.ParseBody(&req); err != nil || req.Platform == "" {
		c.WriteError(http.StatusBadRequest, "平台不能为空")
		return
	}
	expires, err := parseOptionalTime(req.TokenExpiresAt)
	if err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	account := &models.Account{
		ID:             newID(),
		UserID:         user.ID,
		Platform:       req.Platform,
		Nickname:       req.Nickname,
		AvatarURL:      req.AvatarURL,
		Status:         models.AccountStatusActive,
		CookieData:     req.CookieData,
		AccessToken:    req.AccessToken,
		TokenExpiresAt: expires,
	}
	if _, err := services.GetOrm().Insert(account); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建账号失败")
		return
	}
	c.Created(account)
}

// Get GET /api/accounts/:account_id
func (c *AccountsController) Get() {
	if !c.CheckPermission("accounts:read") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	account, err := findOwnAccount(c.GetPathParam("account_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "账号不存在")
		return
	}
	c.OK(account)
}

// Update PUT /api/accounts/:account_id
func (c *AccountsController) Update() {
	if !c.CheckPermission("account:update:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	account, err := findOwnAccount(c.GetPathParam("account_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "账号不存在")
		return
	}
	var req accountUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Nickname != nil {
		account.Nickname = *req.Nickname
	}
	if req.AvatarURL != nil {
		account.AvatarURL = *req.AvatarURL
	}
	if req.Status != nil {
		account.Status = *req.Status
	}
	if req.CookieData != nil {
		account.CookieData = *req.CookieData
	}
	if req.AccessToken != nil {
		account.AccessToken = *req.AccessToken
	}
	if req.TokenExpiresAt != nil {
		expires, perr := parseOptionalTime(*req.TokenExpiresAt)
		if perr != nil {
			c.WriteError(http.StatusBadRequest, perr.Error())
			return
		}
		account.TokenExpiresAt = expires
	}
	if _, err := services.GetOrm().Update(account); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新账号失败")
		return
	}
	c.OK(account)
}

// Delete DELETE /api/accounts/:account_id
func (c *AccountsController) Delete() {
	if !c.CheckPermission("account:delete:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	account, err := findOwnAccount(c.GetPathParam("account_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "账号不存在")
		return
	}
	if _, err := services.GetOrm().Delete(account); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除账号失败")
		return
	}
	c.WriteNoContent()
}

// Check POST /api/accounts/:account_id/check
func (c *AccountsController) Check() {
	if !c.CheckPermission("account:check:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	account, err := findOwnAccount(c.GetPathParam("account_id"), user.ID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "账号不存在")
		return
	}
	now := time.Now()
	account.Status = models.AccountStatusActive
	account.ErrorMessage = ""
	account.LastCheckAt = &now
	if _, err := services.GetOrm().Update(account); err != nil {
		c.WriteError(http.StatusInternalServerError, "检查账号失败")
		return
	}
	c.OK(map[string]interface{}{
		"id":            account.ID,
		"status":        account.Status,
		"last_check_at": account.LastCheckAt,
		"error_message": account.ErrorMessage,
	})
}
