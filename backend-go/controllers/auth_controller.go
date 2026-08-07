package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
	"ai-multi-platform-release/backend-go/utils"
)

type AuthController struct {
	BaseController
}

// Health GET / 与 /healthz
func (c *AuthController) Health() {
	c.OK(map[string]interface{}{"status": "ok"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

type profileUpdateRequest struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}

type passwordChangeRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Login POST /api/auth/login
func (c *AuthController) Login() {
	var req loginRequest
	if err := c.ParseBody(&req); err != nil || req.Username == "" || req.Password == "" {
		c.WriteError(http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	user, err := authenticateUser(req.Username, req.Password)
	if err != nil || user == nil {
		c.WriteError(http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := utils.CreateAccessToken(user.ID, user.Username, user.Role, 1440)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "生成令牌失败")
		return
	}

	c.OK(map[string]interface{}{
		"access_token": token,
		"token_type":   "bearer",
		"user":         toUserInfo(user),
	})
}

// Register POST /api/auth/register
func (c *AuthController) Register() {
	var req registerRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Role == "admin" {
		c.WriteError(http.StatusBadRequest, "超级管理员账号唯一，不可通过注册创建")
		return
	}
	if req.Username == "" || req.Password == "" {
		c.WriteError(http.StatusBadRequest, "用户名和密码不能为空")
		return
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.User)).Filter("username", req.Username).Exist() {
		c.WriteError(http.StatusBadRequest, "该用户名已存在")
		return
	}
	if req.Email != "" && o.QueryTable(new(models.User)).Filter("email", req.Email).Exist() {
		c.WriteError(http.StatusBadRequest, "该邮箱已被使用")
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "密码加密失败")
		return
	}
	role := req.Role
	if role == "" {
		role = models.UserRoleOperator
	}
	user := &models.User{
		ID:             newID(),
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: hashed,
		Nickname:       req.Nickname,
		Role:           role,
		AvatarURL:      req.AvatarURL,
	}
	if _, err := o.Insert(user); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建用户失败")
		return
	}
	c.Created(toUserInfo(user))
}

// Me GET /api/auth/me
func (c *AuthController) Me() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	perms, err := services.GetUserEffectiveFlatPermissions(user.ID, nil)
	if err != nil {
		perms = map[string]string{}
	}
	info := toUserInfo(user)
	c.OK(map[string]interface{}{
		"id":          info.ID,
		"username":    info.Username,
		"email":       info.Email,
		"nickname":    info.Nickname,
		"role":        info.Role,
		"avatar_url":  info.AvatarURL,
		"created_at":  info.CreatedAt,
		"roles":       userRoles(user.ID),
		"permissions": perms,
	})
}

// UpdateProfile PUT /api/auth/profile
func (c *AuthController) UpdateProfile() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req profileUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	o := services.GetOrm()
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if _, err := o.Update(user); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新资料失败")
		return
	}
	c.OK(toUserInfo(user))
}

// ChangePassword PUT /api/auth/password
func (c *AuthController) ChangePassword() {
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req passwordChangeRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if !utils.VerifyPassword(user.HashedPassword, req.OldPassword) {
		c.WriteError(http.StatusBadRequest, "旧密码错误")
		return
	}
	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "密码加密失败")
		return
	}
	user.HashedPassword = hashed
	if _, err := services.GetOrm().Update(user); err != nil {
		c.WriteError(http.StatusInternalServerError, "密码修改失败")
		return
	}
	c.OK(map[string]string{"message": "密码修改成功"})
}

func authenticateUser(loginID, password string) (*models.User, error) {
	o := services.GetOrm()
	var user models.User
	qs := o.QueryTable(new(models.User))
	err := qs.Filter("username", loginID).One(&user)
	if err != nil {
		err = o.QueryTable(new(models.User)).Filter("email", loginID).One(&user)
	}
	if err != nil {
		return nil, nil
	}
	if !utils.VerifyPassword(user.HashedPassword, password) {
		return nil, nil
	}
	return &user, nil
}
