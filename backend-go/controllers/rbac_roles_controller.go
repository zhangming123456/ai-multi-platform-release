package controllers

import (
	"net/http"
	"sort"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type RBACRolesController struct {
	BaseController
}

type createRoleRequest struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Description   string   `json:"description"`
	RoleType      string   `json:"role_type"`
	ParentRoleIDs []string `json:"parent_role_ids"`
}

type updateRoleRequest struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	RoleType    *string `json:"role_type"`
}

type parentRoleRequest struct {
	ParentID string `json:"parent_id"`
}

type updateRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

func roleRef(r *models.RBACRole) map[string]interface{} {
	return map[string]interface{}{
		"id":           r.ID,
		"name":         r.Name,
		"display_name": r.DisplayName,
	}
}

func buildRoleItem(role *models.RBACRole) map[string]interface{} {
	parentRoles := []map[string]interface{}{}
	childRoles := []map[string]interface{}{}
	if parents, err := services.GetRoleAncestors(role.ID); err == nil {
		for i := range parents {
			parentRoles = append(parentRoles, roleRef(&parents[i]))
		}
	}
	if children, err := services.GetRoleDescendants(role.ID); err == nil {
		for i := range children {
			childRoles = append(childRoles, roleRef(&children[i]))
		}
	}
	ancestors := append([]map[string]interface{}{}, parentRoles...)
	descendants := append([]map[string]interface{}{}, childRoles...)
	return map[string]interface{}{
		"id":              role.ID,
		"name":            role.Name,
		"display_name":    role.DisplayName,
		"description":     role.Description,
		"role_type":       role.RoleType,
		"parent_roles":    parentRoles,
		"child_roles":     childRoles,
		"all_ancestors":   ancestors,
		"all_descendants": descendants,
	}
}

func roleUsers(roleID string) []string {
	o := services.GetOrm()
	var assignments []models.RBACUserRoleAssignment
	_, err := o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("role_id", roleID).All(&assignments)
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(assignments))
	for _, a := range assignments {
		result = append(result, a.UserID)
	}
	return result
}

func notifyRoleRename(roleID, actorNickname string, oldDisplay, newDisplay string) {
	userIDs := roleUsers(roleID)
	content := "角色「" + oldDisplay + "」已被 " + actorNickname + " 重命名"
	for _, uid := range userIDs {
		_ = createNotification(uid, models.NotificationTypeRoleUpdated, "角色名称已变更", content, roleID)
	}
}

func notifyRolePermissionsChanged(roleID, actorNickname string, roleDisplay string) {
	userIDs := roleUsers(roleID)
	content := "你的角色「" + roleDisplay + "」的权限已被 " + actorNickname + " 变更"
	for _, uid := range userIDs {
		_ = createNotification(uid, models.NotificationTypeRolePermissionsUpdated, "角色权限已变更", content, roleID)
	}
}

// ListRoles GET /api/v2/roles
func (c *RBACRolesController) ListRoles() {
	if !c.CheckPermission("roles:read") {
		return
	}
	o := services.GetOrm()
	var roles []models.RBACRole
	if _, err := o.QueryTable(new(models.RBACRole)).All(&roles); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询角色列表失败")
		return
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})
	items := make([]map[string]interface{}, 0, len(roles))
	for i := range roles {
		items = append(items, buildRoleItem(&roles[i]))
	}
	c.OK(items)
}

// CreateRole POST /api/v2/roles
func (c *RBACRolesController) CreateRole() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	var req createRoleRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.WriteError(http.StatusBadRequest, "角色名称不能为空")
		return
	}
	if req.RoleType != "" && req.RoleType != models.RoleTypeOther {
		c.WriteError(http.StatusBadRequest, "仅可创建自定义角色")
		return
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.RBACRole)).Filter("name", name).Exist() {
		c.WriteError(http.StatusBadRequest, "该角色名称已存在")
		return
	}
	role := &models.RBACRole{
		ID:          newID(),
		Name:        name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		RoleType:    models.RoleTypeOther,
	}
	if _, err := o.Insert(role); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建角色失败")
		return
	}

	if len(req.ParentRoleIDs) > 0 {
		linkIDs := []string{}
		firstParent, err := getRoleByID(req.ParentRoleIDs[0])
		if err != nil {
			c.WriteError(http.StatusNotFound, "父角色不存在: "+req.ParentRoleIDs[0])
			return
		}
		if firstParent.IsBuiltin {
			c.WriteError(http.StatusBadRequest, "内置角色不可作为父角色")
			return
		}
		for _, pid := range req.ParentRoleIDs {
			parent, err := getRoleByID(pid)
			if err != nil {
				c.WriteError(http.StatusNotFound, "父角色不存在: "+pid)
				return
			}
			if parent.IsBuiltin {
				c.WriteError(http.StatusBadRequest, "内置角色不可作为父角色")
				return
			}
			linkIDs = append(linkIDs, pid)
			ancestors, _ := services.GetRoleAncestorIDs(pid)
			for _, aid := range ancestors {
				if !containsString(linkIDs, aid) {
					linkIDs = append(linkIDs, aid)
				}
			}
		}
		for _, linkID := range linkIDs {
			_, _ = o.Insert(&models.RBACRoleHierarchy{
				ID: newID(), ParentRoleID: linkID, ChildRoleID: role.ID,
			})
		}
	}
	c.Created(buildRoleItem(role))
}

func getRoleByID(roleID string) (*models.RBACRole, error) {
	o := services.GetOrm()
	var role models.RBACRole
	err := o.QueryTable(new(models.RBACRole)).Filter("id", roleID).One(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// GetRole GET /api/v2/roles/:id
func (c *RBACRolesController) GetRole() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	c.OK(buildRoleItem(role))
}

// UpdateRole PUT /api/v2/roles/:id
func (c *RBACRolesController) UpdateRole() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	var req updateRoleRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if role.IsBuiltin && req.RoleType != nil {
		c.WriteError(http.StatusBadRequest, "内置角色不可修改角色类型")
		return
	}
	renamed := false
	oldDisplay := role.DisplayName
	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName != "" && newName != role.Name {
			if o := services.GetOrm(); o.QueryTable(new(models.RBACRole)).
				Filter("name", newName).Exclude("id", role.ID).Exist() {
				c.WriteError(http.StatusBadRequest, "该角色名称已存在")
				return
			}
			role.Name = newName
			renamed = true
		}
	}
	if req.DisplayName != nil {
		role.DisplayName = *req.DisplayName
		if role.DisplayName != oldDisplay {
			renamed = true
		}
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.RoleType != nil {
		role.RoleType = *req.RoleType
	}
	if _, err := services.GetOrm().Update(role); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新角色失败")
		return
	}
	if renamed {
		notifyRoleRename(role.ID, current.Nickname, oldDisplay, role.DisplayName)
	}
	c.OK(buildRoleItem(role))
}

// DeleteRole DELETE /api/v2/roles/:id
func (c *RBACRolesController) DeleteRole() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	if role.IsBuiltin {
		c.WriteError(http.StatusBadRequest, "内置角色不可移除")
		return
	}
	o := services.GetOrm()
	_, _ = o.QueryTable(new(models.RBACRoleHierarchy)).Filter("parent_role_id", role.ID).Delete()
	_, _ = o.QueryTable(new(models.RBACRoleHierarchy)).Filter("child_role_id", role.ID).Delete()
	_, _ = o.QueryTable(new(models.RBACRolePermission)).Filter("role_id", role.ID).Delete()
	if _, err := o.Delete(role); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除角色失败")
		return
	}
	c.WriteNoContent()
}

// AddParentRole POST /api/v2/roles/:id/parents
func (c *RBACRolesController) AddParentRole() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	var req parentRoleRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	parentRole, err := getRoleByID(req.ParentID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "父角色不存在: "+req.ParentID)
		return
	}
	if parentRole.ID == role.ID {
		c.WriteError(http.StatusBadRequest, "角色不能作为自身的父角色")
		return
	}
	descendants, err := services.GetRoleDescendants(role.ID)
	if err == nil {
		for i := range descendants {
			if descendants[i].ID == parentRole.ID {
				c.WriteError(http.StatusBadRequest, "不能将后代角色添加为父角色，否则会造成循环继承")
				return
			}
		}
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.RBACRoleHierarchy)).
		Filter("parent_role_id", parentRole.ID).
		Filter("child_role_id", role.ID).
		Exist() {
		c.WriteError(http.StatusBadRequest, "该父角色已存在")
		return
	}
	_, _ = o.Insert(&models.RBACRoleHierarchy{
		ID: newID(), ParentRoleID: parentRole.ID, ChildRoleID: role.ID,
	})
	ancestors, _ := services.GetRoleAncestorIDs(parentRole.ID)
	for _, aid := range ancestors {
		if aid != role.ID {
			_, _ = o.Insert(&models.RBACRoleHierarchy{
				ID: newID(), ParentRoleID: aid, ChildRoleID: role.ID,
			})
		}
	}
	c.OK(buildRoleItem(role))
}

// RemoveParentRole DELETE /api/v2/roles/:id/parents/:parent_id
func (c *RBACRolesController) RemoveParentRole() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	if role.IsBuiltin {
		c.WriteError(http.StatusBadRequest, "内置角色不可修改父角色")
		return
	}
	parentID := c.GetPathParam("parent_id")
	parentRole, err := getRoleByID(parentID)
	if err != nil {
		c.WriteError(http.StatusNotFound, "父角色不存在: "+parentID)
		return
	}
	o := services.GetOrm()
	relatedIDs := []string{parentRole.ID}
	ancestors, _ := services.GetRoleAncestorIDs(parentRole.ID)
	relatedIDs = append(relatedIDs, ancestors...)
	_, _ = o.QueryTable(new(models.RBACRoleHierarchy)).
		Filter("child_role_id", role.ID).
		Filter("parent_role_id__in", relatedIDs).
		Delete()
	c.WriteNoContent()
}

// GetRolePermissions GET /api/v2/roles/:id/permissions
func (c *RBACRolesController) GetRolePermissions() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	roleIDs := []string{role.ID}
	ancestors, _ := services.GetRoleAncestorIDs(role.ID)
	roleIDs = append(roleIDs, ancestors...)
	keys := rolePermissionKeys(roleIDs)
	names, _ := services.GetPermissionDisplayNames(keys)
	result := map[string]string{}
	for _, key := range keys {
		if name, ok := names[key]; ok {
			result[key] = name
		} else {
			result[key] = key
		}
	}
	c.OK(result)
}

func rolePermissionKeys(roleIDs []string) []string {
	o := services.GetOrm()
	var rps []models.RBACRolePermission
	_, err := o.QueryTable(new(models.RBACRolePermission)).
		Filter("role_id__in", roleIDs).All(&rps)
	if err != nil {
		return nil
	}
	permIDs := make([]string, 0, len(rps))
	for _, rp := range rps {
		permIDs = append(permIDs, rp.PermissionID)
	}
	keySet := map[string]bool{}
	if len(permIDs) > 0 {
		var perms []models.RBACPermission
		_, _ = o.QueryTable(new(models.RBACPermission)).
			Filter("id__in", permIDs).Filter("is_active", true).All(&perms)
		for _, p := range perms {
			keySet[p.Key] = true
		}
	}
	for key := range keySet {
		for _, rk := range services.ResolveReadKeys(key) {
			keySet[rk] = true
		}
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	return keys
}

// GetRoleDirectPermissions GET /api/v2/roles/:id/permissions/direct
func (c *RBACRolesController) GetRoleDirectPermissions() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	o := services.GetOrm()
	var rps []models.RBACRolePermission
	_, _ = o.QueryTable(new(models.RBACRolePermission)).
		Filter("role_id", role.ID).All(&rps)
	permIDs := make([]string, 0, len(rps))
	for _, rp := range rps {
		permIDs = append(permIDs, rp.PermissionID)
	}
	result := map[string]interface{}{}
	if len(permIDs) > 0 {
		var perms []models.RBACPermission
		_, _ = o.QueryTable(new(models.RBACPermission)).
			Filter("id__in", permIDs).Filter("is_active", true).All(&perms)
		keys := make([]string, 0, len(perms))
		for _, p := range perms {
			keys = append(keys, p.Key)
		}
		names, _ := services.GetPermissionDisplayNames(keys)
		for _, p := range perms {
			name := names[p.Key]
			if name == "" {
				name = p.Key
			}
			result[p.Key] = map[string]interface{}{
				"key": p.Key, "name": name, "operation": p.Operation,
			}
		}
	}
	c.OK(result)
}

// GetRolePermissionsDetail GET /api/v2/roles/:id/permissions/detail
func (c *RBACRolesController) GetRolePermissionsDetail() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	roleIDs := []string{role.ID}
	ancestors, _ := services.GetRoleAncestorIDs(role.ID)
	roleIDs = append(roleIDs, ancestors...)
	o := services.GetOrm()
	var rps []models.RBACRolePermission
	_, _ = o.QueryTable(new(models.RBACRolePermission)).
		Filter("role_id__in", roleIDs).All(&rps)
	permIDs := make([]string, 0, len(rps))
	for _, rp := range rps {
		permIDs = append(permIDs, rp.PermissionID)
	}
	result := map[string]interface{}{}
	if len(permIDs) > 0 {
		var perms []models.RBACPermission
		_, _ = o.QueryTable(new(models.RBACPermission)).
			Filter("id__in", permIDs).Filter("is_active", true).All(&perms)
		keys := make([]string, 0, len(perms))
		for _, p := range perms {
			keys = append(keys, p.Key)
		}
		names, _ := services.GetPermissionDisplayNames(keys)
		for _, p := range perms {
			name := names[p.Key]
			if name == "" {
				name = p.Key
			}
			result[p.Key] = map[string]interface{}{
				"key": p.Key, "name": name, "operation": p.Operation,
			}
		}
	}
	c.OK(result)
}

// UpdateRolePermissions PUT /api/v2/roles/:id/permissions
func (c *RBACRolesController) UpdateRolePermissions() {
	if !c.CheckPermission("roles:manage:write") {
		return
	}
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	if role.IsSuperAdmin {
		c.WriteError(http.StatusBadRequest, "超级管理员角色权限不可修改")
		return
	}
	var req updateRolePermissionsRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	o := services.GetOrm()
	var invalidIDs []string
	for _, pid := range req.PermissionIDs {
		if !o.QueryTable(new(models.RBACPermission)).Filter("id", pid).Exist() {
			invalidIDs = append(invalidIDs, pid)
		}
	}
	if len(invalidIDs) > 0 {
		c.WriteError(http.StatusBadRequest, "无效的权限ID: "+strings.Join(invalidIDs, ", "))
		return
	}
	_, _ = o.QueryTable(new(models.RBACRolePermission)).Filter("role_id", role.ID).Delete()
	for _, pid := range req.PermissionIDs {
		_, _ = o.Insert(&models.RBACRolePermission{
			ID: newID(), RoleID: role.ID, PermissionID: pid,
		})
	}
	display := role.DisplayName
	if display == "" {
		display = role.Name
	}
	notifyRolePermissionsChanged(role.ID, current.Nickname, display)
	c.OK(buildRoleItem(role))
}

// PreviewInheritedPermissions GET /api/v2/roles/:id/permissions/preview/:parent_role_id
func (c *RBACRolesController) PreviewInheritedPermissions() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	parentRole, err := getRoleByID(c.GetPathParam("parent_role_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "父角色不存在")
		return
	}
	roleIDs := []string{role.ID, parentRole.ID}
	ancestors, _ := services.GetRoleAncestorIDs(role.ID)
	roleIDs = append(roleIDs, ancestors...)
	parentAncestors, _ := services.GetRoleAncestorIDs(parentRole.ID)
	roleIDs = append(roleIDs, parentAncestors...)
	keys := rolePermissionKeys(roleIDs)
	names, _ := services.GetPermissionDisplayNames(keys)
	result := map[string]string{}
	for _, key := range keys {
		if name, ok := names[key]; ok {
			result[key] = name
		} else {
			result[key] = key
		}
	}
	c.OK(result)
}

// GetRoleInheritance GET /api/v2/roles/:id/inheritance
func (c *RBACRolesController) GetRoleInheritance() {
	if !c.CheckPermission("roles:read") {
		return
	}
	role, err := getRoleByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "角色不存在")
		return
	}
	parentRoles := []map[string]interface{}{}
	childRoles := []map[string]interface{}{}
	if parents, err := services.GetRoleAncestors(role.ID); err == nil {
		for i := range parents {
			parentRoles = append(parentRoles, roleRef(&parents[i]))
		}
	}
	if children, err := services.GetRoleDescendants(role.ID); err == nil {
		for i := range children {
			childRoles = append(childRoles, roleRef(&children[i]))
		}
	}
	c.OK(map[string]interface{}{
		"parent_roles": parentRoles,
		"child_roles":  childRoles,
	})
}
