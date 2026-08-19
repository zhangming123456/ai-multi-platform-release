package services

import (
	"encoding/json"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/utils"
)

type resourceDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type builtinRoleDef struct {
	Name         string
	DisplayName  string
	RoleType     string
	IsSuperAdmin bool
	IsBuiltin    bool
}

var RBACResources = []resourceDef{
	{Key: "dashboard:read", Name: "仪表盘", Description: "系统首页仪表盘，展示核心数据概览"},
	{Key: "platforms:read", Name: "平台管理", Description: "管理已接入的第三方内容平台"},
	{Key: "content:read", Name: "内容列表", Description: "查看和管理所有内容列表"},
	{Key: "publish:read", Name: "发布管理", Description: "管理内容发布任务和发布计划"},
	{Key: "templates:read", Name: "模板管理", Description: "管理内容创作模板"},
	{Key: "review:read", Name: "内容审核", Description: "管理内容审核流程"},
	{Key: "sql_review:read", Name: "SQL审核", Description: "管理SQL变更审核流程"},
	{Key: "accounts:read", Name: "平台账号", Description: "管理各平台的登录账号信息"},
	{Key: "token_plan:read", Name: "AI 模型服务商管理", Description: "管理AI模型服务商接入与调用配额"},
	{Key: "api_docs:read", Name: "API文档", Description: "查看系统API接口文档"},
	{Key: "db:read", Name: "数据库控制台", Description: "访问数据库控制台"},
	{Key: "users:read", Name: "用户管理", Description: "管理系统用户列表和基本信息"},
	{Key: "permissions:read", Name: "权限管理", Description: "管理角色权限分配"},
	{Key: "roles:read", Name: "角色管理", Description: "管理角色定义和角色继承关系"},
	{Key: "constraints:read", Name: "约束管理", Description: "管理职责分离约束规则"},
	{Key: "notification:enum:read", Name: "通知消息字典", Description: "管理通知消息中的字段和枚举显示名称"},
	{Key: "permissions:manage:read", Name: "查看权限字典", Description: "创建、编辑、删除权限资源定义"},
	{Key: "permissions:manage:write", Name: "维护权限字典", Description: "创建、编辑、删除权限资源定义"},
	{Key: "roles:manage:write", Name: "维护角色", Description: "创建、编辑、删除角色定义和层级关系"},
	{Key: "constraints:manage:write", Name: "维护约束", Description: "创建、编辑、删除职责分离约束规则"},
	{Key: "notification:enum:write", Name: "维护通知消息字典", Description: "创建、编辑、删除通知消息字典条目"},
	{Key: "content:create:write", Name: "创建内容", Description: "创建新的内容条目"},
	{Key: "content:update:write", Name: "编辑内容", Description: "编辑已有内容条目的标题、正文等信息"},
	{Key: "content:delete:write", Name: "删除内容", Description: "删除已有内容条目"},
	{Key: "content:ai_generate:write", Name: "AI生成内容", Description: "使用AI辅助生成内容"},
	{Key: "publish:create:write", Name: "创建发布", Description: "创建内容发布任务"},
	{Key: "publish:retry:write", Name: "重试发布", Description: "重新执行失败的发布任务"},
	{Key: "templates:create:write", Name: "创建模板", Description: "创建新的内容模板"},
	{Key: "templates:update:write", Name: "编辑模板", Description: "编辑已有的内容模板"},
	{Key: "templates:delete:write", Name: "删除模板", Description: "删除已有的内容模板"},
	{Key: "review:submit:write", Name: "提交审核", Description: "将内容提交至审核流程"},
	{Key: "review:approve:write", Name: "通过审核", Description: "批准待审核的内容"},
	{Key: "review:reject:write", Name: "驳回审核", Description: "驳回审核不通过的内容"},
	{Key: "db:execute:write", Name: "执行SQL", Description: "在数据库控制台中执行SQL语句"},
	{Key: "users:create:write", Name: "创建用户", Description: "创建新的系统用户"},
	{Key: "users:update:read", Name: "查看用户", Description: "查看用户详细信息"},
	{Key: "users:update:write", Name: "编辑用户", Description: "编辑用户的昵称、邮箱等基本信息"},
	{Key: "users:delete:write", Name: "删除用户", Description: "删除系统用户"},
	{Key: "users:change_password:write", Name: "修改用户密码", Description: "修改用户的登录密码"},
	{Key: "users:custom_permissions:write", Name: "自定义用户权限", Description: "为个别用户配置自定义权限覆盖"},
	{Key: "account:create:write", Name: "创建平台账号", Description: "创建新的平台登录账号"},
	{Key: "account:update:write", Name: "编辑平台账号", Description: "编辑平台账号信息"},
	{Key: "account:delete:write", Name: "删除平台账号", Description: "删除平台登录账号"},
	{Key: "account:check:write", Name: "校验平台账号", Description: "校验平台账号的有效性"},
	{Key: "db_change:submit:write", Name: "提交SQL变更", Description: "提交SQL变更申请"},
	{Key: "db_change:approve:write", Name: "通过SQL变更", Description: "批准SQL变更申请"},
	{Key: "db_change:reject:write", Name: "驳回SQL变更", Description: "驳回SQL变更申请"},
	{Key: "model_config:create:write", Name: "创建模型配置", Description: "创建新的AI模型配置"},
	{Key: "model_config:update:write", Name: "编辑模型配置", Description: "编辑AI模型配置参数"},
	{Key: "model_config:delete:write", Name: "删除模型配置", Description: "删除AI模型配置"},
	{Key: "db_history:view:read", Name: "查看SQL历史", Description: "查看SQL执行历史记录"},
	{Key: "stores:read", Name: "门店管理", Description: "查看门店列表和门店信息"},
	{Key: "stores:create:write", Name: "创建门店", Description: "创建新的门店"},
	{Key: "stores:update:write", Name: "编辑门店", Description: "编辑门店信息"},
	{Key: "stores:delete:write", Name: "删除门店", Description: "删除门店"},
	{Key: "inspection:read", Name: "巡店记录", Description: "查看巡店检查记录和检查项"},
	{Key: "inspection:create:write", Name: "发起巡店", Description: "发起新的巡店检查"},
	{Key: "inspection:update:write", Name: "编辑巡店", Description: "编辑巡店检查记录"},
	{Key: "inspection:delete:write", Name: "删除巡店", Description: "删除巡店检查记录"},
	{Key: "inspection:ai:write", Name: "AI巡店", Description: "使用AI分析巡店照片生成检查报告"},
	{Key: "inspection:template:read", Name: "查看检查表模板", Description: "查看巡店检查表模板及其检查项"},
	{Key: "inspection:template:create:write", Name: "创建检查表模板", Description: "创建巡店检查表模板"},
	{Key: "inspection:template:update:write", Name: "编辑检查表模板", Description: "编辑巡店检查表模板及其检查项"},
	{Key: "inspection:template:delete:write", Name: "删除检查表模板", Description: "删除巡店检查表模板"},
	{Key: "inspection:template:ai_create", Name: "AI智能添加检查项", Description: "使用AI从图片或描述中识别并智能添加检查表模板检查项"},
	{Key: "inspection:material:read", Name: "查看巡店素材库", Description: "查看巡店素材库中的检查标准图片与检查项"},
	{Key: "inspection:material:create:write", Name: "创建巡店素材", Description: "创建新的巡店检查素材"},
	{Key: "inspection:material:update:write", Name: "编辑巡店素材", Description: "编辑巡店检查素材"},
	{Key: "inspection:material:delete:write", Name: "删除巡店素材", Description: "删除巡店检查素材"},
	{Key: "inspection:task:read", Name: "查看整改任务", Description: "查看巡店整改任务列表和详情"},
	{Key: "inspection:task:update:write", Name: "提交整改", Description: "上传整改照片并提交整改结果"},
	{Key: "inspection:task:recheck:write", Name: "AI复核整改", Description: "发起AI复核整改照片，判断问题是否修复"},
	{Key: "inspection:task:confirm:write", Name: "人工确认整改", Description: "对转人工审核的整改问题项进行人工判定"},
	{Key: "material:read", Name: "查看素材管理", Description: "查看素材库中的图片等素材"},
	{Key: "material:create:write", Name: "创建素材", Description: "创建新的素材"},
	{Key: "material:update:write", Name: "编辑素材", Description: "编辑素材信息"},
	{Key: "material:delete:write", Name: "删除素材", Description: "删除素材"},
}

func AllPermissionKeys() []string {
	keys := make([]string, 0, len(RBACResources))
	for _, r := range RBACResources {
		keys = append(keys, r.Key)
	}
	return keys
}

var builtinRoles = []builtinRoleDef{
	{Name: "admin", DisplayName: "超级管理员", RoleType: "admin", IsSuperAdmin: true, IsBuiltin: true},
	{Name: "manager", DisplayName: "管理员", RoleType: "admin", IsSuperAdmin: false, IsBuiltin: true},
	{Name: "operator", DisplayName: "运营者", RoleType: "other", IsSuperAdmin: false, IsBuiltin: true},
	{Name: "reviewer", DisplayName: "审核员", RoleType: "other", IsSuperAdmin: false, IsBuiltin: true},
}

var defaultRolePermissions = map[string][]string{
	"admin": AllPermissionKeys(),
	"manager": []string{
		"dashboard:read", "platforms:read", "content:read", "publish:read",
		"templates:read", "review:read", "sql_review:read", "accounts:read",
		"token_plan:read", "api_docs:read", "permissions:read", "roles:read",
		"constraints:read", "permissions:manage:read", "permissions:manage:write",
		"roles:manage:write", "constraints:manage:write", "notification:enum:read",
		"notification:enum:write", "users:read", "users:create:write",
		"users:update:read", "users:update:write", "users:delete:write",
		"users:change_password:write", "users:custom_permissions:write",
		"content:create:write", "content:update:write", "content:delete:write",
		"content:ai_generate:write", "review:submit:write", "review:approve:write",
		"review:reject:write", "db_change:submit:write", "db_change:approve:write",
		"db_change:reject:write", "templates:create:write", "templates:update:write",
		"templates:delete:write", "account:create:write", "account:update:write",
		"account:delete:write", "account:check:write", "publish:create:write",
		"publish:retry:write", "model_config:create:write", "model_config:update:write",
		"model_config:delete:write", "db:execute:write", "db_history:view:read",
		"stores:read", "stores:create:write", "stores:update:write",
		"stores:delete:write", "inspection:read", "inspection:create:write",
		"inspection:update:write", "inspection:delete:write", "inspection:ai:write",
		"inspection:template:read", "inspection:template:create:write",
		"inspection:template:update:write", "inspection:template:delete:write",
		"inspection:template:ai_create",
		"inspection:material:read", "inspection:material:create:write",
		"inspection:material:update:write", "inspection:material:delete:write",
		"inspection:task:read", "inspection:task:update:write",
		"inspection:task:recheck:write", "inspection:task:confirm:write",
		"material:read", "material:create:write",
		"material:update:write", "material:delete:write",
	},
	"operator": []string{
		"dashboard:read", "platforms:read", "content:read", "publish:read",
		"templates:read", "accounts:read", "token_plan:read", "api_docs:read",
		"users:read", "users:create:write", "users:update:write",
		"users:change_password:write", "users:custom_permissions:write", "roles:read",
		"content:create:write", "content:update:write", "content:delete:write",
		"content:ai_generate:write", "review:submit:write", "db_change:submit:write",
		"templates:create:write", "templates:update:write", "templates:delete:write",
		"account:create:write", "account:update:write", "account:delete:write",
		"account:check:write", "publish:create:write", "publish:retry:write",
		"stores:read", "stores:create:write", "stores:update:write",
		"inspection:read", "inspection:create:write", "inspection:update:write",
		"inspection:ai:write", "inspection:template:read",
		"inspection:template:create:write", "inspection:template:update:write",
		"inspection:template:ai_create",
		"inspection:material:read", "inspection:material:create:write",
		"inspection:material:update:write",
		"inspection:task:read", "inspection:task:update:write",
		"inspection:task:recheck:write", "inspection:task:confirm:write",
		"material:read", "material:create:write", "material:update:write",
	},
	"reviewer": []string{
		"dashboard:read", "content:read", "review:read", "sql_review:read",
		"platforms:read", "accounts:read", "review:approve:write", "review:reject:write",
		"db_change:approve:write", "db_change:reject:write",
		"templates:create:write", "templates:update:write", "templates:delete:write",
		"stores:read", "inspection:read", "inspection:template:read",
		"inspection:material:read", "inspection:task:read", "inspection:task:confirm:write",
		"material:read",
	},
}

func ExtractOperation(key string) string {
	parts := splitKey(key)
	if len(parts) == 2 {
		return parts[1]
	}
	if len(parts) >= 3 {
		return parts[len(parts)-2]
	}
	return parts[len(parts)-1]
}

func InitRBACSystem() error {
	o := GetOrm()
	if err := ensureBuiltinRoles(o); err != nil {
		return err
	}
	if err := ensureResourcesAndPermissions(o); err != nil {
		return err
	}
	if err := ensureBuiltinRolePermissions(o); err != nil {
		return err
	}
	if err := syncUserRoleAssignments(o); err != nil {
		return err
	}
	if err := ensureDefaultConstraints(o); err != nil {
		return err
	}
	return nil
}

func ensureBuiltinRoles(o orm.Ormer) error {
	for _, def := range builtinRoles {
		exists := o.QueryTable(new(models.RBACRole)).Filter("name", def.Name).Exist()
		if exists {
			continue
		}
		role := &models.RBACRole{
			ID:           uuid.NewString(),
			Name:         def.Name,
			DisplayName:  def.DisplayName,
			RoleType:     def.RoleType,
			IsSuperAdmin: def.IsSuperAdmin,
			IsBuiltin:    def.IsBuiltin,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if _, err := o.Insert(role); err != nil {
			return err
		}
	}
	return nil
}

func ensureResourcesAndPermissions(o orm.Ormer) error {
	for _, def := range RBACResources {
		var resource models.RBACResource
		err := o.QueryTable(new(models.RBACResource)).Filter("key", def.Key).One(&resource)
		if err == orm.ErrNoRows {
			resource = models.RBACResource{
				ID:          uuid.NewString(),
				Key:         def.Key,
				Name:        def.Name,
				Description: def.Description,
				IsActive:    true,
				CreatedAt:   time.Now(),
			}
			if _, err := o.Insert(&resource); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		var perm models.RBACPermission
		err = o.QueryTable(new(models.RBACPermission)).Filter("key", def.Key).One(&perm)
		if err == nil {
			continue
		}
		if err != orm.ErrNoRows {
			return err
		}
		perm = models.RBACPermission{
			ID:         uuid.NewString(),
			ResourceID: resource.ID,
			Operation:  ExtractOperation(def.Key),
			Key:        def.Key,
			IsActive:   true,
			CreatedAt:  time.Now(),
		}
		if _, err := o.Insert(&perm); err != nil {
			return err
		}
	}
	return nil
}

func ensureBuiltinRolePermissions(o orm.Ormer) error {
	for roleName, keys := range defaultRolePermissions {
		var role models.RBACRole
		if err := o.QueryTable(new(models.RBACRole)).Filter("name", roleName).One(&role); err != nil {
			continue
		}
		for _, key := range keys {
			var perm models.RBACPermission
			if err := o.QueryTable(new(models.RBACPermission)).Filter("key", key).One(&perm); err != nil {
				continue
			}
			exists := o.QueryTable(new(models.RBACRolePermission)).
				Filter("role_id", role.ID).
				Filter("permission_id", perm.ID).
				Exist()
			if exists {
				continue
			}
			rp := &models.RBACRolePermission{
				ID:           uuid.NewString(),
				RoleID:       role.ID,
				PermissionID: perm.ID,
				GrantType:    "direct",
				CreatedAt:    time.Now(),
			}
			if _, err := o.Insert(rp); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncUserRoleAssignments(o orm.Ormer) error {
	var users []models.User
	if _, err := o.QueryTable(new(models.User)).All(&users); err != nil {
		return err
	}
	rolesByID := map[string]string{}
	for _, u := range users {
		if _, ok := rolesByID[u.Role]; ok {
			continue
		}
		var role models.RBACRole
		if err := o.QueryTable(new(models.RBACRole)).Filter("name", u.Role).One(&role); err == nil {
			rolesByID[u.ID] = role.ID
		}
	}
	for _, u := range users {
		roleID, ok := rolesByID[u.ID]
		if !ok {
			continue
		}
		exists := o.QueryTable(new(models.RBACUserRoleAssignment)).
			Filter("user_id", u.ID).
			Filter("role_id", roleID).
			Exist()
		if exists {
			continue
		}
		assignment := &models.RBACUserRoleAssignment{
			ID:        uuid.NewString(),
			UserID:    u.ID,
			RoleID:    roleID,
			GrantType: "direct",
			CreatedAt: time.Now(),
		}
		if _, err := o.Insert(assignment); err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultConstraints(o orm.Ormer) error {
	exists := o.QueryTable(new(models.RBACConstraint)).Filter("name", "运营者与审核员互斥").Exist()
	if exists {
		return nil
	}
	config, _ := json.Marshal(map[string]string{"scope": "static"})
	constraint := &models.RBACConstraint{
		ID:             uuid.NewString(),
		Name:           "运营者与审核员互斥",
		Description:    "同一用户不能同时担任运营者和审核员角色，防止利益冲突",
		ConstraintType: "mutual_exclusive",
		Config:         string(config),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	if _, err := o.Insert(constraint); err != nil {
		return err
	}
	var roles []models.RBACRole
	if _, err := o.QueryTable(new(models.RBACRole)).
		Filter("name__in", []string{"operator", "reviewer"}).
		All(&roles); err != nil {
		return err
	}
	for _, role := range roles {
		association := &models.RBACConstraintRoleAssociation{
			ID:              uuid.NewString(),
			ConstraintID:    constraint.ID,
			RoleID:          role.ID,
			AssociationType: "subject",
		}
		if _, err := o.Insert(association); err != nil {
			return err
		}
	}
	return nil
}

func SeedUsers() error {
	o := GetOrm()
	seeds := []struct {
		ID       string
		Username string
		Email    string
		Password string
		Nickname string
		Role     string
	}{
		{ID: "1", Username: "admin", Email: "admin@admin.com", Password: "admin123", Nickname: "超级管理员", Role: "admin"},
		{ID: "", Username: "manager", Email: "manager@example.com", Password: "manager123", Nickname: "管理员", Role: "manager"},
		{ID: "", Username: "operator", Email: "operator@example.com", Password: "operator123", Nickname: "运营小二", Role: "operator"},
		{ID: "", Username: "reviewer", Email: "reviewer@example.com", Password: "reviewer123", Nickname: "审核专员", Role: "reviewer"},
	}
	for _, s := range seeds {
		exists := o.QueryTable(new(models.User)).Filter("username", s.Username).Exist()
		if exists {
			continue
		}
		hashed, err := utils.HashPassword(s.Password)
		if err != nil {
			return err
		}
		user := &models.User{
			ID:             s.ID,
			Username:       s.Username,
			Email:          s.Email,
			HashedPassword: hashed,
			Nickname:       s.Nickname,
			Role:           s.Role,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if user.ID == "" {
			user.ID = uuid.NewString()
		}
		if _, err := o.Insert(user); err != nil {
			return err
		}
	}
	return nil
}

func SeedNotificationDict() error {
	o := GetOrm()
	n, err := o.QueryTable(new(models.NotificationDict)).Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	entries := []struct {
		category string
		groupKey string
		dictKey  string
		dictVal  string
		sort     int
	}{
		{"field", "user_fields", "nickname", "昵称", 1},
		{"field", "user_fields", "email", "邮箱", 2},
		{"field", "user_fields", "username", "用户名", 3},
		{"field", "user_fields", "phone", "手机号", 4},
		{"field", "user_fields", "status", "状态", 5},
		{"enum", "notification_type", "review_submit", "审核提交", 1},
		{"enum", "notification_type", "review_approved", "审核通过", 2},
		{"enum", "notification_type", "review_rejected", "审核驳回", 3},
		{"enum", "notification_type", "role_updated", "角色更新", 4},
		{"enum", "notification_type", "role_permissions_updated", "权限变更", 5},
		{"enum", "user_status", "active", "启用", 1},
		{"enum", "user_status", "inactive", "禁用", 2},
	}
	for _, e := range entries {
		entry := &models.NotificationDict{
			ID:        uuid.NewString(),
			Category:  e.category,
			GroupKey:  e.groupKey,
			DictKey:   e.dictKey,
			DictValue: e.dictVal,
			IsActive:  true,
			SortOrder: e.sort,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if _, err := o.Insert(entry); err != nil {
			return err
		}
	}
	return nil
}
