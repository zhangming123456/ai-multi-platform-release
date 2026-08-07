package controllers

import (
	"errors"
	"net/http"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
	"ai-multi-platform-release/backend-go/utils"
)

type RBACUsersController struct {
	BaseController
}

type roleAssignmentInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	RoleType    string `json:"role_type"`
}

type userDetailResponse struct {
	ID        string               `json:"id"`
	Username  string               `json:"username"`
	Email     string               `json:"email"`
	Nickname  string               `json:"nickname"`
	Role      string               `json:"role"`
	AvatarURL string               `json:"avatar_url"`
	CreatedAt string               `json:"created_at"`
	Roles     []roleAssignmentInfo `json:"roles"`
}

type createUserRequest struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Nickname  string   `json:"nickname"`
	Email     string   `json:"email"`
	AvatarURL string   `json:"avatar_url"`
	Role      string   `json:"role"`
	RoleIDs   []string `json:"role_ids"`
}

type updateUserRequest struct {
	Nickname  *string   `json:"nickname"`
	Email     *string   `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	RoleIDs   *[]string `json:"role_ids"`
}

type updateUserRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type updateUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
	OldPassword string `json:"old_password"`
}

type permissionOverrideEntry struct {
	PermissionKey string `json:"permission_key"`
	Granted       bool   `json:"granted"`
}

type updateUserPermissionOverridesRequest struct {
	Overrides []permissionOverrideEntry `json:"overrides"`
}

func userRolesInfo(userID string) []roleAssignmentInfo {
	o := services.GetOrm()
	var assignments []models.RBACUserRoleAssignment
	_, err := o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("user_id", userID).
		All(&assignments)
	if err != nil || len(assignments) == 0 {
		return []roleAssignmentInfo{}
	}
	roleIDs := make([]string, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}
	var roles []models.RBACRole
	_, err = o.QueryTable(new(models.RBACRole)).
		Filter("id__in", roleIDs).
		All(&roles)
	if err != nil {
		return []roleAssignmentInfo{}
	}
	result := make([]roleAssignmentInfo, 0, len(roles))
	for _, r := range roles {
		result = append(result, roleAssignmentInfo{
			ID: r.ID, Name: r.Name, DisplayName: r.DisplayName, RoleType: r.RoleType,
		})
	}
	return result
}

func buildUserDetail(u *models.User) userDetailResponse {
	return userDetailResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Role:      u.Role,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05"),
		Roles:     userRolesInfo(u.ID),
	}
}

func isManagerOrAdmin(u *models.User) bool {
	return u.Role == models.UserRoleAdmin || u.Role == models.UserRoleManager
}

func userRoleNames(userID string) map[string]bool {
	o := services.GetOrm()
	var assignments []models.RBACUserRoleAssignment
	_, err := o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("user_id", userID).
		All(&assignments)
	if err != nil || len(assignments) == 0 {
		return map[string]bool{}
	}
	roleIDs := make([]string, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}
	var roles []models.RBACRole
	_, err = o.QueryTable(new(models.RBACRole)).
		Filter("id__in", roleIDs).
		All(&roles)
	if err != nil {
		return map[string]bool{}
	}
	result := map[string]bool{}
	for _, r := range roles {
		if r.Name != "" {
			result[r.Name] = true
		}
	}
	return result
}

func canAccessUserTarget(currentUser *models.User, targetUser *models.User) bool {
	if isManagerOrAdmin(currentUser) {
		return true
	}
	if currentUser.ID == targetUser.ID {
		return true
	}
	currentRoles := userRoleNames(currentUser.ID)
	targetRoles := userRoleNames(targetUser.ID)
	for name := range currentRoles {
		if targetRoles[name] {
			return true
		}
	}
	return false
}

func assertNotAssigningSuperAdmin(roleIDs []string, userID string) error {
	o := services.GetOrm()
	var role models.RBACRole
	err := o.QueryTable(new(models.RBACRole)).Filter("is_super_admin", true).One(&role)
	if err != nil {
		return nil
	}
	for _, rid := range roleIDs {
		if rid == role.ID {
			if userID == "1" {
				return nil
			}
			return errors.New("超级管理员角色仅限系统唯一管理员，不可分配给其他用户")
		}
	}
	return nil
}

func notifyUserRoleChange(targetUserID, actorNickname string, roleIDs []string, prevRoleIDs map[string]bool) {
	if len(roleIDs) == 0 {
		return
	}
	newRoleSet := map[string]bool{}
	changed := false
	for _, id := range roleIDs {
		if !prevRoleIDs[id] {
			changed = true
		}
		newRoleSet[id] = true
	}
	for id := range prevRoleIDs {
		if !newRoleSet[id] {
			changed = true
		}
	}
	if !changed {
		return
	}

	roleNames := displayNamesByRoleIDs(roleIDs)
	prevIDs := make([]string, 0, len(prevRoleIDs))
	for id := range prevRoleIDs {
		prevIDs = append(prevIDs, id)
	}
	prevRoleNames := displayNamesByRoleIDs(prevIDs)

	roleList := strings.Join(roleNames, "、")
	if roleList == "" {
		roleList = "无"
	}
	prevRoleList := strings.Join(prevRoleNames, "、")
	if prevRoleList == "" {
		prevRoleList = "无"
	}

	var content string
	switch {
	case len(prevRoleNames) == 0 && len(roleNames) > 0:
		content = "你的角色已被 " + actorNickname + " 添加「" + roleList + "」"
	case len(prevRoleNames) > 0 && len(roleNames) == 0:
		content = "你的角色已被 " + actorNickname + " 移除「" + prevRoleList + "」"
	default:
		content = "你的角色已被 " + actorNickname + " 从「" + prevRoleList + "」更新为「" + roleList + "」"
	}
	_ = createNotification(targetUserID, models.NotificationTypeRolePermissionsUpdated, "角色分配已变更", content, "")
}

func displayNamesByRoleIDs(roleIDs []string) []string {
	if len(roleIDs) == 0 {
		return nil
	}
	o := services.GetOrm()
	var roles []models.RBACRole
	_, err := o.QueryTable(new(models.RBACRole)).Filter("id__in", roleIDs).All(&roles)
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		result = append(result, r.DisplayName)
	}
	return result
}

func notifyUserInfoChange(userID string, changedFields []map[string]string, actorNickname string) {
	if len(changedFields) == 0 {
		return
	}
	payload, err := services.CreateUserInfoChangeNotification(userID, changedFields, actorNickname)
	if err != nil || payload == nil {
		return
	}
	_ = createNotification(userID, payload.Type, payload.Title, payload.Content, "")
}

// ListUsers GET /api/v2/users
func (c *RBACUsersController) ListUsers() {
	if !c.CheckPermission("users:read") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	o := services.GetOrm()
	var users []models.User
	if _, err := o.QueryTable(new(models.User)).OrderBy("-created_at").All(&users); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询用户列表失败")
		return
	}
	filtered := make([]models.User, 0, len(users))
	if isManagerOrAdmin(current) {
		filtered = users
	} else {
		currentRoles := userRoleNames(current.ID)
		for _, u := range users {
			if u.ID == current.ID {
				filtered = append(filtered, u)
				continue
			}
			targetRoles := userRoleNames(u.ID)
			overlap := false
			for name := range currentRoles {
				if targetRoles[name] {
					overlap = true
					break
				}
			}
			if overlap {
				filtered = append(filtered, u)
			}
		}
	}
	items := make([]userDetailResponse, 0, len(filtered))
	for i := range filtered {
		items = append(items, buildUserDetail(&filtered[i]))
	}
	c.OK(items)
}

// CreateUser POST /api/v2/users
func (c *RBACUsersController) CreateUser() {
	if !c.CheckPermission("users:create:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req createUserRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	role := req.Role
	if role == "" {
		role = models.UserRoleOperator
	}
	if role == models.UserRoleAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员账号唯一，不可创建")
		return
	}
	if !isManagerOrAdmin(current) {
		if role != models.UserRoleOperator {
			c.WriteError(http.StatusForbidden, "仅可创建运营者角色账号")
			return
		}
		if len(req.RoleIDs) > 0 {
			c.WriteError(http.StatusForbidden, "非管理员不可直接分配 RBAC3 角色")
			return
		}
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
	if !utils.ValidatePasswordFormat(req.Password) {
		c.WriteError(http.StatusBadRequest, "密码格式不符合要求")
		return
	}

	if !isManagerOrAdmin(current) {
		hashed, err := utils.HashPassword(req.Password)
		if err != nil {
			c.WriteError(http.StatusInternalServerError, "密码加密失败")
			return
		}
		creationReq := &models.UserCreationRequest{
			ID:             newID(),
			RequesterID:    current.ID,
			Username:       req.Username,
			Email:          req.Email,
			HashedPassword: hashed,
			Nickname:       req.Nickname,
			Role:           role,
			AvatarURL:      req.AvatarURL,
			Status:         models.UserCreationStatusPending,
		}
		if _, err := o.Insert(creationReq); err != nil {
			c.WriteError(http.StatusInternalServerError, "创建审核申请失败")
			return
		}
		c.WriteError(http.StatusAccepted, "账号创建申请已提交审核，请等待管理员审批")
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "密码加密失败")
		return
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

	roleIDs := req.RoleIDs
	if len(roleIDs) == 0 && req.Role != "" {
		var roleObj models.RBACRole
		if err := o.QueryTable(new(models.RBACRole)).Filter("name", req.Role).One(&roleObj); err == nil {
			roleIDs = []string{roleObj.ID}
		}
	}
	if len(roleIDs) > 0 {
		proposed := map[string]bool{}
		for _, rid := range roleIDs {
			proposed[rid] = true
		}
		violations, err := services.ValidateUserRoleAssignments(user.ID, proposed)
		if err != nil {
			c.WriteError(http.StatusInternalServerError, "校验角色约束失败")
			return
		}
		if len(violations) > 0 {
			c.WriteError(http.StatusBadRequest, strings.Join(violations, "; "))
			return
		}
		if err := assertNotAssigningSuperAdmin(roleIDs, user.ID); err != nil {
			c.WriteError(http.StatusBadRequest, err.Error())
			return
		}
		for _, rid := range roleIDs {
			_, _ = o.Insert(&models.RBACUserRoleAssignment{
				ID: newID(), UserID: user.ID, RoleID: rid, GrantType: "direct",
			})
		}
	}
	c.Created(buildUserDetail(user))
}

// GetUser GET /api/v2/users/:id
func (c *RBACUsersController) GetUser() {
	if !c.CheckPermission("users:read") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if !canAccessUserTarget(current, &user) {
		c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
		return
	}
	c.OK(buildUserDetail(&user))
}

// UpdateUser PUT /api/v2/users/:id
func (c *RBACUsersController) UpdateUser() {
	if !c.CheckPermission("users:update:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if !canAccessUserTarget(current, &user) {
		c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
		return
	}
	var req updateUserRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if !isManagerOrAdmin(current) && req.RoleIDs != nil {
		c.WriteError(http.StatusForbidden, "仅管理员可修改角色分配")
		return
	}
	if user.Role == models.UserRoleAdmin && req.RoleIDs != nil {
		c.WriteError(http.StatusBadRequest, "超级管理员角色不可修改")
		return
	}

	var changedFields []map[string]string

	if req.Email != nil && *req.Email != "" && *req.Email != user.Email {
		if o.QueryTable(new(models.User)).Filter("email", *req.Email).Exclude("id", userID).Exist() {
			c.WriteError(http.StatusBadRequest, "该邮箱已被其他用户使用")
			return
		}
	}
	if req.Nickname != nil && *req.Nickname != user.Nickname {
		old := user.Nickname
		user.Nickname = *req.Nickname
		changedFields = append(changedFields, map[string]string{"field": "nickname", "old": old, "new": *req.Nickname})
	}
	if req.Email != nil && *req.Email != user.Email {
		old := user.Email
		user.Email = *req.Email
		changedFields = append(changedFields, map[string]string{"field": "email", "old": old, "new": *req.Email})
	}
	if req.AvatarURL != nil && *req.AvatarURL != user.AvatarURL {
		old := user.AvatarURL
		user.AvatarURL = *req.AvatarURL
		changedFields = append(changedFields, map[string]string{"field": "avatar_url", "old": old, "new": *req.AvatarURL})
	}

	if req.RoleIDs != nil {
		proposed := map[string]bool{}
		for _, rid := range *req.RoleIDs {
			proposed[rid] = true
		}
		violations, err := services.ValidateUserRoleAssignments(userID, proposed)
		if err != nil {
			c.WriteError(http.StatusInternalServerError, "校验角色约束失败")
			return
		}
		if len(violations) > 0 {
			c.WriteError(http.StatusBadRequest, strings.Join(violations, "; "))
			return
		}
		if err := assertNotAssigningSuperAdmin(*req.RoleIDs, userID); err != nil {
			c.WriteError(http.StatusBadRequest, err.Error())
			return
		}
		prevRoleIDs := map[string]bool{}
		var assignments []models.RBACUserRoleAssignment
		_, _ = o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID).All(&assignments)
		for _, a := range assignments {
			prevRoleIDs[a.RoleID] = true
		}
		_, _ = o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID).Delete()
		for _, rid := range *req.RoleIDs {
			_, _ = o.Insert(&models.RBACUserRoleAssignment{
				ID: newID(), UserID: userID, RoleID: rid, GrantType: "direct",
			})
		}
		if userID != current.ID {
			notifyUserRoleChange(userID, current.Nickname, *req.RoleIDs, prevRoleIDs)
		}
	}

	if len(changedFields) > 0 && userID != current.ID {
		notifyUserInfoChange(userID, changedFields, current.Nickname)
	}

	if _, err := o.Update(&user); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新用户失败")
		return
	}
	c.OK(buildUserDetail(&user))
}

// UpdateUserRoles PUT /api/v2/users/:id/roles
func (c *RBACUsersController) UpdateUserRoles() {
	if !c.CheckPermission("users:update:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if user.Role == models.UserRoleAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员角色不可修改")
		return
	}
	if !isManagerOrAdmin(current) {
		c.WriteError(http.StatusForbidden, "仅管理员可修改用户角色分配")
		return
	}
	var req updateUserRolesRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	proposed := map[string]bool{}
	for _, rid := range req.RoleIDs {
		proposed[rid] = true
	}
	violations, err := services.ValidateUserRoleAssignments(userID, proposed)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "校验角色约束失败")
		return
	}
	if len(violations) > 0 {
		c.WriteError(http.StatusBadRequest, strings.Join(violations, "; "))
		return
	}
	if err := assertNotAssigningSuperAdmin(req.RoleIDs, userID); err != nil {
		c.WriteError(http.StatusBadRequest, err.Error())
		return
	}
	prevRoleIDs := map[string]bool{}
	var assignments []models.RBACUserRoleAssignment
	_, _ = o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID).All(&assignments)
	for _, a := range assignments {
		prevRoleIDs[a.RoleID] = true
	}
	_, _ = o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID).Delete()
	for _, rid := range req.RoleIDs {
		_, _ = o.Insert(&models.RBACUserRoleAssignment{
			ID: newID(), UserID: userID, RoleID: rid, GrantType: "direct",
		})
	}
	if userID != current.ID {
		notifyUserRoleChange(userID, current.Nickname, req.RoleIDs, prevRoleIDs)
	}
	c.OK(buildUserDetail(&user))
}

// ChangeUserPassword PUT /api/v2/users/:id/password
func (c *RBACUsersController) ChangeUserPassword() {
	if !c.CheckPermission("users:change_password:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if !isManagerOrAdmin(current) && current.ID != userID {
		c.WriteError(http.StatusForbidden, "仅管理员或用户本人可修改密码")
		return
	}
	var req updateUserPasswordRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if !utils.ValidatePasswordFormat(req.NewPassword) {
		c.WriteError(http.StatusBadRequest, "密码格式不符合要求")
		return
	}
	isAdminReset := isManagerOrAdmin(current)
	if !isAdminReset {
		defaultPwd := user.Username + "123"
		currentIsDefault := utils.VerifyPassword(user.HashedPassword, defaultPwd)
		if !currentIsDefault {
			if req.OldPassword == "" {
				c.WriteError(http.StatusBadRequest, "非默认密码，请输入旧密码进行验证")
				return
			}
			if !utils.VerifyPassword(user.HashedPassword, req.OldPassword) {
				c.WriteError(http.StatusBadRequest, "旧密码验证失败")
				return
			}
		}
	}
	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "密码加密失败")
		return
	}
	user.HashedPassword = hashed
	if _, err := o.Update(&user); err != nil {
		c.WriteError(http.StatusInternalServerError, "修改密码失败")
		return
	}
	c.OK(map[string]interface{}{"message": "密码修改成功"})
}

// GetUserPermissions GET /api/v2/users/:id/permissions
func (c *RBACUsersController) GetUserPermissions() {
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	isSelf := userID == current.ID
	if !isSelf {
		if !canAccessUserTarget(current, &user) {
			c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
			return
		}
		ok, err := services.HasPermission(current.ID, "users:custom_permissions:read", "read", nil)
		if err != nil || !ok {
			c.WriteError(http.StatusForbidden, "无权查看其他用户的自定义权限")
			return
		}
	}

	effective, err := services.GetUserEffectiveFlatPermissions(userID, nil)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "获取有效权限失败")
		return
	}

	closure, err := services.GetRoleClosureIDs(userID, nil)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "获取角色闭包失败")
		return
	}
	allRoleIDs := make([]string, 0, len(closure))
	for id := range closure {
		allRoleIDs = append(allRoleIDs, id)
	}

	rolePermissionKeys := map[string]bool{}
	if hasSuperAdminRole(closure) {
		var perms []models.RBACPermission
		_, _ = o.QueryTable(new(models.RBACPermission)).Filter("is_active", true).All(&perms)
		for _, p := range perms {
			rolePermissionKeys[p.Key] = true
		}
	} else if len(allRoleIDs) > 0 {
		var rps []models.RBACRolePermission
		_, _ = o.QueryTable(new(models.RBACRolePermission)).Filter("role_id__in", allRoleIDs).All(&rps)
		permIDs := make([]string, 0, len(rps))
		for _, rp := range rps {
			permIDs = append(permIDs, rp.PermissionID)
		}
		if len(permIDs) > 0 {
			var perms []models.RBACPermission
			_, _ = o.QueryTable(new(models.RBACPermission)).
				Filter("id__in", permIDs).Filter("is_active", true).All(&perms)
			for _, p := range perms {
				rolePermissionKeys[p.Key] = true
			}
		}
	}

	keyList := make([]string, 0, len(rolePermissionKeys))
	for key := range rolePermissionKeys {
		keyList = append(keyList, key)
	}

	available := []map[string]interface{}{}
	if len(keyList) > 0 {
		var perms []models.RBACPermission
		_, _ = o.QueryTable(new(models.RBACPermission)).
			Filter("key__in", keyList).OrderBy("created_at").All(&perms)
		resourceIDs := map[string]bool{}
		for _, p := range perms {
			resourceIDs[p.ResourceID] = true
		}
		resourceMap := map[string]models.RBACResource{}
		if len(resourceIDs) > 0 {
			var ids []string
			for id := range resourceIDs {
				ids = append(ids, id)
			}
			var resources []models.RBACResource
			_, _ = o.QueryTable(new(models.RBACResource)).Filter("id__in", ids).All(&resources)
			for _, r := range resources {
				resourceMap[r.ID] = r
			}
		}
		for i := range perms {
			p := perms[i]
			res := resourceMap[p.ResourceID]
			available = append(available, map[string]interface{}{
				"id":         p.ID,
				"key":        p.Key,
				"operation":  p.Operation,
				"is_active":  p.IsActive,
				"created_at": p.CreatedAt.Format("2006-01-02T15:04:05"),
				"resource": map[string]interface{}{
					"id": res.ID, "key": res.Key, "name": res.Name, "description": res.Description,
				},
			})
		}
	}

	c.OK(map[string]interface{}{
		"effective_permissions": effective,
		"available_permissions": available,
	})
}

func hasSuperAdminRole(roleIDs map[string]bool) bool {
	if len(roleIDs) == 0 {
		return false
	}
	idList := make([]string, 0, len(roleIDs))
	for id := range roleIDs {
		idList = append(idList, id)
	}
	o := services.GetOrm()
	n, _ := o.QueryTable(new(models.RBACRole)).
		Filter("id__in", idList).Filter("is_super_admin", true).Count()
	return n > 0
}

// GetUserPermissionOverrides GET /api/v2/users/:id/permission-overrides
func (c *RBACUsersController) GetUserPermissionOverrides() {
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	isSelf := userID == current.ID
	if !isSelf {
		var user models.User
		if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
			c.WriteError(http.StatusNotFound, "用户不存在")
			return
		}
		if !canAccessUserTarget(current, &user) {
			c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
			return
		}
		ok, err := services.HasPermission(current.ID, "users:custom_permissions:read", "read", nil)
		if err != nil || !ok {
			c.WriteError(http.StatusForbidden, "无权查看其他用户的自定义权限覆盖")
			return
		}
	}
	var overrides []models.RBACUserPermissionOverride
	_, err := o.QueryTable(new(models.RBACUserPermissionOverride)).
		Filter("user_id", userID).OrderBy("permission_key").All(&overrides)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询权限覆盖失败")
		return
	}
	items := make([]permissionOverrideEntry, 0, len(overrides))
	for _, ov := range overrides {
		items = append(items, permissionOverrideEntry{PermissionKey: ov.PermissionKey, Granted: ov.Granted})
	}
	c.OK(items)
}

// UpdateUserPermissionOverrides PUT /api/v2/users/:id/permission-overrides
func (c *RBACUsersController) UpdateUserPermissionOverrides() {
	if !c.CheckPermission("users:custom_permissions:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if user.Role == models.UserRoleAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员账号不可修改权限覆盖")
		return
	}
	if !canAccessUserTarget(current, &user) {
		c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
		return
	}
	var req updateUserPermissionOverridesRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	_, _ = o.QueryTable(new(models.RBACUserPermissionOverride)).Filter("user_id", userID).Delete()
	for _, entry := range req.Overrides {
		_, _ = o.Insert(&models.RBACUserPermissionOverride{
			ID: newID(), UserID: userID, PermissionKey: entry.PermissionKey, Granted: entry.Granted,
		})
	}
	c.OK(req.Overrides)
}

// ResetUserPermissionOverrides DELETE /api/v2/users/:id/permission-overrides
func (c *RBACUsersController) ResetUserPermissionOverrides() {
	if !c.CheckPermission("users:custom_permissions:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if user.Role == models.UserRoleAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员账号不可修改权限覆盖")
		return
	}
	if !canAccessUserTarget(current, &user) {
		c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
		return
	}
	_, _ = o.QueryTable(new(models.RBACUserPermissionOverride)).Filter("user_id", userID).Delete()
	c.WriteNoContent()
}

// DeleteUser DELETE /api/v2/users/:id
func (c *RBACUsersController) DeleteUser() {
	if !c.CheckPermission("users:delete:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if user.Role == models.UserRoleAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员账号不可移除")
		return
	}
	if user.ID == current.ID {
		c.WriteError(http.StatusBadRequest, "不能删除自己的账号")
		return
	}
	if !canAccessUserTarget(current, &user) {
		c.WriteError(http.StatusForbidden, "无法访问该用户：非管理角色只能操作同角色用户")
		return
	}
	_, _ = o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID).Delete()
	if _, err := o.Delete(&user); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除用户失败")
		return
	}
	c.WriteNoContent()
}

// GetPasswordStatus GET /api/v2/users/:id/password-status
func (c *RBACUsersController) GetPasswordStatus() {
	if !c.CheckPermission("users:read") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	userID := c.GetPathParam("id")
	o := services.GetOrm()
	var user models.User
	if err := o.QueryTable(new(models.User)).Filter("id", userID).One(&user); err != nil {
		c.WriteError(http.StatusNotFound, "用户不存在")
		return
	}
	if !isManagerOrAdmin(current) && user.ID != current.ID {
		c.WriteError(http.StatusForbidden, "无权查看")
		return
	}
	defaultPwd := user.Username + "123"
	c.OK(map[string]interface{}{"is_default_password": utils.VerifyPassword(user.HashedPassword, defaultPwd)})
}
