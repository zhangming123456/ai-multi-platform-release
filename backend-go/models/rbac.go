package models

import (
	"time"
)

const (
	RoleTypeAdmin = "admin"
	RoleTypeOther = "other"
)

type RBACRole struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	Name         string    `orm:"column(name);unique;size(50)" json:"name"`
	DisplayName  string    `orm:"column(display_name);size(100)" json:"display_name"`
	Description  string    `orm:"column(description);type(text);null" json:"description"`
	RoleType     string    `orm:"column(role_type);size(20);default(other)" json:"role_type"`
	IsSuperAdmin bool      `orm:"column(is_super_admin);default(false)" json:"is_super_admin"`
	IsBuiltin    bool      `orm:"column(is_builtin);default(false)" json:"is_builtin"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt    time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (r *RBACRole) TableName() string {
	return "rbac_roles"
}

type RBACResource struct {
	ID          string    `orm:"column(id);pk;size(36)" json:"id"`
	Key         string    `orm:"column(key);unique;size(100)" json:"key"`
	Name        string    `orm:"column(name);size(100)" json:"name"`
	Description string    `orm:"column(description);type(text);null" json:"description"`
	ParentID    string    `orm:"column(parent_id);size(36);null" json:"parent_id"`
	IsActive    bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt   time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACResource) TableName() string {
	return "rbac_resources"
}

type RBACPermission struct {
	ID         string    `orm:"column(id);pk;size(36)" json:"id"`
	ResourceID string    `orm:"column(resource_id);size(36);index" json:"resource_id"`
	Operation  string    `orm:"column(operation);size(50)" json:"operation"`
	Key        string    `orm:"column(key);unique;size(150)" json:"key"`
	IsActive   bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt  time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACPermission) TableName() string {
	return "rbac_permissions"
}

type RBACRoleHierarchy struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	ParentRoleID string    `orm:"column(parent_role_id);size(36);index" json:"parent_role_id"`
	ChildRoleID  string    `orm:"column(child_role_id);size(36);index" json:"child_role_id"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACRoleHierarchy) TableName() string {
	return "rbac_role_hierarchy"
}

type RBACRolePermission struct {
	ID           string    `orm:"column(id);pk;size(36)" json:"id"`
	RoleID       string    `orm:"column(role_id);size(36);index" json:"role_id"`
	PermissionID string    `orm:"column(permission_id);size(36);index" json:"permission_id"`
	GrantType    string    `orm:"column(grant_type);size(20);default(direct)" json:"grant_type"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACRolePermission) TableName() string {
	return "rbac_role_permissions"
}

type RBACUserRoleAssignment struct {
	ID         string     `orm:"column(id);pk;size(36)" json:"id"`
	UserID     string     `orm:"column(user_id);size(36);index" json:"user_id"`
	RoleID     string     `orm:"column(role_id);size(36);index" json:"role_id"`
	GrantType  string     `orm:"column(grant_type);size(20);default(direct)" json:"grant_type"`
	ValidFrom  *time.Time `orm:"column(valid_from);type(datetime);null" json:"valid_from"`
	ValidUntil *time.Time `orm:"column(valid_until);type(datetime);null" json:"valid_until"`
	CreatedAt  time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACUserRoleAssignment) TableName() string {
	return "rbac_user_role_assignments"
}

type RBACUserPermissionOverride struct {
	ID            string    `orm:"column(id);pk;size(36)" json:"id"`
	UserID        string    `orm:"column(user_id);size(36);index" json:"user_id"`
	PermissionKey string    `orm:"column(permission_key);size(150)" json:"permission_key"`
	Granted       bool      `orm:"column(granted);default(true)" json:"granted"`
	CreatedAt     time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt     time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func (r *RBACUserPermissionOverride) TableName() string {
	return "rbac_user_permission_overrides"
}

type RBACConstraint struct {
	ID             string    `orm:"column(id);pk;size(36)" json:"id"`
	Name           string    `orm:"column(name);size(100)" json:"name"`
	Description    string    `orm:"column(description);type(text);null" json:"description"`
	ConstraintType string    `orm:"column(constraint_type);size(50)" json:"constraint_type"`
	Config         string    `orm:"column(config);type(text);null" json:"config"`
	IsActive       bool      `orm:"column(is_active);default(true)" json:"is_active"`
	CreatedAt      time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

func (r *RBACConstraint) TableName() string {
	return "rbac_constraints"
}

type RBACConstraintRoleAssociation struct {
	ID              string `orm:"column(id);pk;size(36)" json:"id"`
	ConstraintID    string `orm:"column(constraint_id);size(36);index" json:"constraint_id"`
	RoleID          string `orm:"column(role_id);size(36);index" json:"role_id"`
	AssociationType string `orm:"column(association_type);size(50)" json:"association_type"`
}

func (r *RBACConstraintRoleAssociation) TableName() string {
	return "rbac_constraint_role_associations"
}
