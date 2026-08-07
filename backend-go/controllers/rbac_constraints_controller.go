package controllers

import (
	"net/http"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type RBACConstraintsController struct {
	BaseController
}

type constraintAssociationRequest struct {
	RoleIDs          []string `json:"role_ids"`
	AssociationTypes []string `json:"association_types"`
}

type createConstraintRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	ConstraintType   string   `json:"constraint_type"`
	Config           string   `json:"config"`
	RoleIDs          []string `json:"role_ids"`
	AssociationTypes []string `json:"association_types"`
}

type updateConstraintRequest struct {
	Name             *string   `json:"name"`
	Description      *string   `json:"description"`
	ConstraintType   *string   `json:"constraint_type"`
	Config           *string   `json:"config"`
	IsActive         *bool     `json:"is_active"`
	RoleIDs          *[]string `json:"role_ids"`
	AssociationTypes *[]string `json:"association_types"`
}

func constraintRoleRefs(constraintID string) (subjectRoles, prerequisiteRoles []map[string]interface{}) {
	o := services.GetOrm()
	var assocs []models.RBACConstraintRoleAssociation
	_, err := o.QueryTable(new(models.RBACConstraintRoleAssociation)).
		Filter("constraint_id", constraintID).All(&assocs)
	if err != nil {
		return
	}
	roleIDs := map[string]bool{}
	for _, a := range assocs {
		roleIDs[a.RoleID] = true
	}
	roleMap := map[string]models.RBACRole{}
	if len(roleIDs) > 0 {
		var ids []string
		for id := range roleIDs {
			ids = append(ids, id)
		}
		var roles []models.RBACRole
		_, _ = o.QueryTable(new(models.RBACRole)).Filter("id__in", ids).All(&roles)
		for _, r := range roles {
			roleMap[r.ID] = r
		}
	}
	subjectRoles = []map[string]interface{}{}
	prerequisiteRoles = []map[string]interface{}{}
	for _, a := range assocs {
		role, ok := roleMap[a.RoleID]
		if !ok {
			continue
		}
		ref := map[string]interface{}{"id": role.ID, "name": role.Name, "display_name": role.DisplayName}
		if a.AssociationType == "subject" {
			subjectRoles = append(subjectRoles, ref)
		} else if a.AssociationType == "prerequisite" {
			prerequisiteRoles = append(prerequisiteRoles, ref)
		}
	}
	return
}

func constraintItem(c *models.RBACConstraint) map[string]interface{} {
	subjectRoles, prerequisiteRoles := constraintRoleRefs(c.ID)
	return map[string]interface{}{
		"id":                 c.ID,
		"name":               c.Name,
		"description":        c.Description,
		"constraint_type":    c.ConstraintType,
		"config":             c.Config,
		"is_active":          c.IsActive,
		"created_at":         c.CreatedAt.Format("2006-01-02T15:04:05"),
		"updated_at":         c.CreatedAt.Format("2006-01-02T15:04:05"),
		"subject_roles":      subjectRoles,
		"prerequisite_roles": prerequisiteRoles,
	}
}

func validateConstraintAssociations(roleIDs, associationTypes []string) string {
	if len(roleIDs) != len(associationTypes) {
		return "role_ids 与 association_types 数量不一致"
	}
	for _, at := range associationTypes {
		if at != "subject" && at != "prerequisite" {
			return "无效的关联类型: " + at
		}
	}
	o := services.GetOrm()
	for _, rid := range roleIDs {
		if !o.QueryTable(new(models.RBACRole)).Filter("id", rid).Exist() {
			return "角色不存在: " + rid
		}
	}
	return ""
}

// ListConstraints GET /api/v2/constraints
func (c *RBACConstraintsController) ListConstraints() {
	if !c.CheckPermission("constraints:read") {
		return
	}
	o := services.GetOrm()
	var constraints []models.RBACConstraint
	_, err := o.QueryTable(new(models.RBACConstraint)).OrderBy("created_at").All(&constraints)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询约束列表失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(constraints))
	for i := range constraints {
		items = append(items, constraintItem(&constraints[i]))
	}
	c.OK(items)
}

// CreateConstraint POST /api/v2/constraints
func (c *RBACConstraintsController) CreateConstraint() {
	if !c.CheckPermission("constraints:manage:write") {
		return
	}
	var req createConstraintRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	o := services.GetOrm()
	if o.QueryTable(new(models.RBACConstraint)).Filter("name", req.Name).Exist() {
		c.WriteError(http.StatusConflict, "约束名称已存在: "+req.Name)
		return
	}
	if req.ConstraintType != "mutual_exclusive" &&
		req.ConstraintType != "prerequisite" &&
		req.ConstraintType != "cardinality" {
		c.WriteError(http.StatusBadRequest, "不支持的约束类型")
		return
	}
	if msg := validateConstraintAssociations(req.RoleIDs, req.AssociationTypes); msg != "" {
		c.WriteError(http.StatusBadRequest, msg)
		return
	}
	config := req.Config
	if config == "" {
		config = "{}"
	}
	ct := &models.RBACConstraint{
		ID:             newID(),
		Name:           req.Name,
		Description:    req.Description,
		ConstraintType: req.ConstraintType,
		Config:         config,
	}
	if _, err := o.Insert(ct); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建约束失败")
		return
	}
	for i, rid := range req.RoleIDs {
		_, _ = o.Insert(&models.RBACConstraintRoleAssociation{
			ID: newID(), ConstraintID: ct.ID, RoleID: rid, AssociationType: req.AssociationTypes[i],
		})
	}
	c.Created(constraintItem(ct))
}

// GetConstraint GET /api/v2/constraints/:id
func (c *RBACConstraintsController) GetConstraint() {
	if !c.CheckPermission("constraints:read") {
		return
	}
	ct, err := getConstraintByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "约束不存在")
		return
	}
	c.OK(constraintItem(ct))
}

func getConstraintByID(constraintID string) (*models.RBACConstraint, error) {
	o := services.GetOrm()
	var ct models.RBACConstraint
	err := o.QueryTable(new(models.RBACConstraint)).Filter("id", constraintID).One(&ct)
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

// UpdateConstraint PUT /api/v2/constraints/:id
func (c *RBACConstraintsController) UpdateConstraint() {
	if !c.CheckPermission("constraints:manage:write") {
		return
	}
	ct, err := getConstraintByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "约束不存在")
		return
	}
	var req updateConstraintRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	o := services.GetOrm()
	if req.Name != nil && *req.Name != ct.Name {
		if o.QueryTable(new(models.RBACConstraint)).
			Filter("name", *req.Name).Exclude("id", ct.ID).Exist() {
			c.WriteError(http.StatusConflict, "约束名称已存在: "+*req.Name)
			return
		}
		ct.Name = *req.Name
	}
	if req.ConstraintType != nil && *req.ConstraintType != ct.ConstraintType {
		if *req.ConstraintType != "mutual_exclusive" &&
			*req.ConstraintType != "prerequisite" &&
			*req.ConstraintType != "cardinality" {
			c.WriteError(http.StatusBadRequest, "不支持的约束类型")
			return
		}
		ct.ConstraintType = *req.ConstraintType
	}
	if req.Description != nil {
		ct.Description = *req.Description
	}
	if req.Config != nil {
		ct.Config = *req.Config
	}
	if req.IsActive != nil {
		ct.IsActive = *req.IsActive
	}
	if req.RoleIDs != nil {
		roleIDs := *req.RoleIDs
		associationTypes := []string{}
		if req.AssociationTypes != nil {
			associationTypes = *req.AssociationTypes
		}
		if msg := validateConstraintAssociations(roleIDs, associationTypes); msg != "" {
			c.WriteError(http.StatusBadRequest, msg)
			return
		}
		_, _ = o.QueryTable(new(models.RBACConstraintRoleAssociation)).
			Filter("constraint_id", ct.ID).Delete()
		for i, rid := range roleIDs {
			_, _ = o.Insert(&models.RBACConstraintRoleAssociation{
				ID: newID(), ConstraintID: ct.ID, RoleID: rid, AssociationType: associationTypes[i],
			})
		}
	}
	if _, err := o.Update(ct); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新约束失败")
		return
	}
	c.OK(constraintItem(ct))
}

// DeleteConstraint DELETE /api/v2/constraints/:id
func (c *RBACConstraintsController) DeleteConstraint() {
	if !c.CheckPermission("constraints:manage:write") {
		return
	}
	ct, err := getConstraintByID(c.GetPathParam("id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "约束不存在")
		return
	}
	o := services.GetOrm()
	_, _ = o.QueryTable(new(models.RBACConstraintRoleAssociation)).
		Filter("constraint_id", ct.ID).Delete()
	if _, err := o.Delete(ct); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除约束失败")
		return
	}
	c.OK(map[string]interface{}{"message": "约束已删除"})
}
