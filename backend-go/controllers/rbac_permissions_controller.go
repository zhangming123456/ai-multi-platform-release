package controllers

import (
	"net/http"
	"sort"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type RBACPermissionsController struct {
	BaseController
}

type createResourceRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
	IsActive    *bool  `json:"is_active"`
}

type updateResourceRequest struct {
	Key         *string `json:"key"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type createPermissionRequest struct {
	ResourceID string `json:"resource_id"`
	Operation  string `json:"operation"`
	Key        string `json:"key"`
	IsActive   *bool  `json:"is_active"`
}

type updatePermissionRequest struct {
	Key       *string `json:"key"`
	Operation *string `json:"operation"`
	IsActive  *bool   `json:"is_active"`
}

func isValidPermissionKey(key string) bool {
	parts := strings.Split(key, ":")
	if len(parts) == 2 {
		return parts[0] != "" && parts[1] != "" &&
			!strings.Contains(parts[0], "read") && !strings.Contains(parts[0], "write")
	}
	if len(parts) == 3 {
		return parts[0] != "" && parts[1] != "" && parts[2] != "" &&
			(parts[2] == "read" || parts[2] == "write") &&
			!strings.Contains(parts[0], "read") && !strings.Contains(parts[0], "write") &&
			!strings.Contains(parts[1], "read") && !strings.Contains(parts[1], "write")
	}
	return false
}

func isPagePermissionKey(key string) bool {
	parts := strings.Split(key, ":")
	return len(parts) == 2 && (parts[1] == "read" || parts[1] == "write")
}

func resourceRef(r *models.RBACResource) map[string]interface{} {
	return map[string]interface{}{
		"id":          r.ID,
		"key":         r.Key,
		"name":        r.Name,
		"description": r.Description,
	}
}

func permissionItem(p *models.RBACPermission, r *models.RBACResource) map[string]interface{} {
	return map[string]interface{}{
		"id":         p.ID,
		"key":        p.Key,
		"operation":  p.Operation,
		"is_active":  p.IsActive,
		"created_at": p.CreatedAt.Format("2006-01-02T15:04:05"),
		"resource":   resourceRef(r),
	}
}

// GetMyPermissions GET /api/v2/me/permissions
func (c *RBACPermissionsController) GetMyPermissions() {
	current := c.CurrentUser()
	if current == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	effective, err := services.GetUserEffectiveFlatPermissions(current.ID, nil)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "获取权限失败")
		return
	}
	c.OK(effective)
}

// ListPermissionEnums GET /api/v2/permission-enums
func (c *RBACPermissionsController) ListPermissionEnums() {
	if !c.CheckPermissionAny([]string{"permissions:read", "permissions:manage:read"}) {
		return
	}
	o := services.GetOrm()
	var perms []models.RBACPermission
	_, err := o.QueryTable(new(models.RBACPermission)).
		Filter("is_active", true).All(&perms)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询权限列表失败")
		return
	}
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].Key < perms[j].Key
	})
	resourceMap := loadResourceMap(perms)
	items := make([]map[string]interface{}, 0, len(perms))
	for i := range perms {
		res, ok := resourceMap[perms[i].ResourceID]
		name := ""
		if ok {
			name = res.Name
		}
		items = append(items, map[string]interface{}{
			"key": perms[i].Key, "name": name,
		})
	}
	c.OK(items)
}

func loadResourceMap(perms []models.RBACPermission) map[string]models.RBACResource {
	resourceIDs := map[string]bool{}
	for _, p := range perms {
		resourceIDs[p.ResourceID] = true
	}
	result := map[string]models.RBACResource{}
	if len(resourceIDs) == 0 {
		return result
	}
	var ids []string
	for id := range resourceIDs {
		ids = append(ids, id)
	}
	var resources []models.RBACResource
	_, _ = services.GetOrm().QueryTable(new(models.RBACResource)).
		Filter("id__in", ids).All(&resources)
	for _, r := range resources {
		result[r.ID] = r
	}
	return result
}

// ListPagePermissions GET /api/v2/permission-pages
func (c *RBACPermissionsController) ListPagePermissions() {
	if !c.CheckPermissionAny([]string{"permissions:read", "permissions:manage:read"}) {
		return
	}
	o := services.GetOrm()
	var perms []models.RBACPermission
	_, err := o.QueryTable(new(models.RBACPermission)).
		Filter("is_active", true).All(&perms)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询权限列表失败")
		return
	}
	pagePerms := make([]models.RBACPermission, 0, len(perms))
	for _, p := range perms {
		if isPagePermissionKey(p.Key) {
			pagePerms = append(pagePerms, p)
		}
	}
	sort.Slice(pagePerms, func(i, j int) bool {
		return pagePerms[i].Key < pagePerms[j].Key
	})
	resourceMap := loadResourceMap(pagePerms)
	items := make([]map[string]interface{}, 0, len(pagePerms))
	for i := range pagePerms {
		res, ok := resourceMap[pagePerms[i].ResourceID]
		item := map[string]interface{}{"key": pagePerms[i].Key}
		if ok {
			item["name"] = res.Name
			item["description"] = res.Description
		}
		items = append(items, item)
	}
	c.OK(items)
}

func resourceNode(r *models.RBACResource, children []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":          r.ID,
		"key":         r.Key,
		"name":        r.Name,
		"description": r.Description,
		"parent_id":   r.ParentID,
		"is_active":   r.IsActive,
		"created_at":  r.CreatedAt.Format("2006-01-02T15:04:05"),
		"children":    children,
	}
}

// ListResources GET /api/v2/resources
func (c *RBACPermissionsController) ListResources() {
	if !c.CheckPermissionAny([]string{"permissions:read", "permissions:manage:read"}) {
		return
	}
	o := services.GetOrm()
	var resources []models.RBACResource
	_, err := o.QueryTable(new(models.RBACResource)).OrderBy("created_at").All(&resources)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询资源列表失败")
		return
	}
	childrenMap := map[string][]map[string]interface{}{}
	nodeMap := map[string]models.RBACResource{}
	for _, r := range resources {
		nodeMap[r.ID] = r
	}
	for _, r := range resources {
		if r.ParentID != "" {
			if _, ok := nodeMap[r.ParentID]; ok {
				childrenMap[r.ParentID] = append(childrenMap[r.ParentID], resourceNode(&r, nil))
			}
		}
	}
	roots := []map[string]interface{}{}
	for _, r := range resources {
		if r.ParentID == "" {
			roots = append(roots, resourceNode(&r, childrenMap[r.ID]))
		} else if _, ok := nodeMap[r.ParentID]; !ok {
			roots = append(roots, resourceNode(&r, childrenMap[r.ID]))
		}
	}
	c.OK(roots)
}

// CreateResource POST /api/v2/resources
func (c *RBACPermissionsController) CreateResource() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	var req createResourceRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if !isValidPermissionKey(req.Key) {
		c.WriteError(http.StatusBadRequest, "权限 key 格式无效：page 为 {name}:read，action 为 {name}:{operation}:{read|write}")
		return
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.RBACResource)).Filter("key", req.Key).Exist() {
		c.WriteError(http.StatusConflict, "资源 key 已存在")
		return
	}
	if req.ParentID != "" {
		if !o.QueryTable(new(models.RBACResource)).Filter("id", req.ParentID).Exist() {
			c.WriteError(http.StatusBadRequest, "父资源不存在")
			return
		}
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	resource := &models.RBACResource{
		ID: newID(), Key: req.Key, Name: req.Name,
		Description: req.Description, ParentID: req.ParentID, IsActive: isActive,
	}
	if _, err := o.Insert(resource); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建资源失败")
		return
	}
	c.Created(resourceNode(resource, nil))
}

// UpdateResource PUT /api/v2/resources/:id
func (c *RBACPermissionsController) UpdateResource() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	resourceID := c.GetPathParam("id")
	o := services.GetOrm()
	var res models.RBACResource
	if err := o.QueryTable(new(models.RBACResource)).Filter("id", resourceID).One(&res); err != nil {
		c.WriteError(http.StatusNotFound, "资源不存在")
		return
	}
	var req updateResourceRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Key != nil && *req.Key != res.Key {
		if !isValidPermissionKey(*req.Key) {
			c.WriteError(http.StatusBadRequest, "权限 key 格式无效")
			return
		}
		if o.QueryTable(new(models.RBACResource)).
			Filter("key", *req.Key).Exclude("id", res.ID).Exist() {
			c.WriteError(http.StatusConflict, "资源 key 已存在")
			return
		}
		res.Key = *req.Key
	}
	if req.Name != nil {
		res.Name = *req.Name
	}
	if req.Description != nil {
		res.Description = *req.Description
	}
	if req.IsActive != nil {
		res.IsActive = *req.IsActive
	}
	if _, err := o.Update(&res); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新资源失败")
		return
	}
	c.OK(resourceRef(&res))
}

// DeleteResource DELETE /api/v2/resources/:id
func (c *RBACPermissionsController) DeleteResource() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	resourceID := c.GetPathParam("id")
	o := services.GetOrm()
	var res models.RBACResource
	if err := o.QueryTable(new(models.RBACResource)).Filter("id", resourceID).One(&res); err != nil {
		c.WriteError(http.StatusNotFound, "资源不存在")
		return
	}
	_, _ = o.QueryTable(new(models.RBACPermission)).Filter("resource_id", res.ID).Delete()
	if _, err := o.Delete(&res); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除资源失败")
		return
	}
	c.WriteNoContent()
}

// ListPermissions GET /api/v2/permissions
func (c *RBACPermissionsController) ListPermissions() {
	if !c.CheckPermissionAny([]string{"permissions:read", "permissions:manage:read"}) {
		return
	}
	o := services.GetOrm()
	var perms []models.RBACPermission
	_, err := o.QueryTable(new(models.RBACPermission)).OrderBy("created_at").All(&perms)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询权限列表失败")
		return
	}
	resourceMap := loadResourceMap(perms)
	items := make([]map[string]interface{}, 0, len(perms))
	for i := range perms {
		res, ok := resourceMap[perms[i].ResourceID]
		if !ok {
			continue
		}
		items = append(items, permissionItem(&perms[i], &res))
	}
	c.OK(items)
}

// CreatePermission POST /api/v2/permissions
func (c *RBACPermissionsController) CreatePermission() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	var req createPermissionRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	o := services.GetOrm()
	var res models.RBACResource
	if err := o.QueryTable(new(models.RBACResource)).Filter("id", req.ResourceID).One(&res); err != nil {
		c.WriteError(http.StatusBadRequest, "资源不存在")
		return
	}
	key := req.Key
	if key == "" {
		key = res.Key + ":" + req.Operation
	}
	if !isValidPermissionKey(key) {
		c.WriteError(http.StatusBadRequest, "权限 key 格式无效：page 为 {name}:read，action 为 {name}:{operation}:{read|write}")
		return
	}
	if o.QueryTable(new(models.RBACPermission)).Filter("key", key).Exist() {
		c.WriteError(http.StatusConflict, "权限 key 已存在")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	perm := &models.RBACPermission{
		ID: newID(), ResourceID: req.ResourceID,
		Operation: req.Operation, Key: key, IsActive: isActive,
	}
	if _, err := o.Insert(perm); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建权限失败")
		return
	}
	c.Created(permissionItem(perm, &res))
}

// UpdatePermission PUT /api/v2/permissions/:id
func (c *RBACPermissionsController) UpdatePermission() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	permissionID := c.GetPathParam("id")
	o := services.GetOrm()
	var perm models.RBACPermission
	if err := o.QueryTable(new(models.RBACPermission)).Filter("id", permissionID).One(&perm); err != nil {
		c.WriteError(http.StatusNotFound, "权限不存在")
		return
	}
	var res models.RBACResource
	_ = o.QueryTable(new(models.RBACResource)).Filter("id", perm.ResourceID).One(&res)
	var req updatePermissionRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Key != nil && *req.Key != perm.Key {
		if !isValidPermissionKey(*req.Key) {
			c.WriteError(http.StatusBadRequest, "权限 key 格式无效")
			return
		}
		if o.QueryTable(new(models.RBACPermission)).
			Filter("key", *req.Key).Exclude("id", perm.ID).Exist() {
			c.WriteError(http.StatusConflict, "权限 key 已存在")
			return
		}
		perm.Key = *req.Key
	}
	if req.Operation != nil {
		perm.Operation = *req.Operation
	}
	if req.IsActive != nil {
		perm.IsActive = *req.IsActive
	}
	if _, err := o.Update(&perm); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新权限失败")
		return
	}
	c.OK(permissionItem(&perm, &res))
}

// DeletePermission DELETE /api/v2/permissions/:id
func (c *RBACPermissionsController) DeletePermission() {
	if !c.CheckPermission("permissions:manage:write") {
		return
	}
	permissionID := c.GetPathParam("id")
	o := services.GetOrm()
	var perm models.RBACPermission
	if err := o.QueryTable(new(models.RBACPermission)).Filter("id", permissionID).One(&perm); err != nil {
		c.WriteError(http.StatusNotFound, "权限不存在")
		return
	}
	if _, err := o.Delete(&perm); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除权限失败")
		return
	}
	c.WriteNoContent()
}
