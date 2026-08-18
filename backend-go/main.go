package main

import (
	"log"
	"net/http"
	"os"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"

	"ai-multi-platform-release/backend-go/controllers"
	"ai-multi-platform-release/backend-go/middleware"
	"ai-multi-platform-release/backend-go/services"
)

func corsFilter(ctx *context.Context) {
	ctx.Output.Header("Access-Control-Allow-Origin", "*")
	ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Output.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
	ctx.Output.Header("Access-Control-Max-Age", "86400")
	if ctx.Input.Method() == http.MethodOptions {
		ctx.Abort(http.StatusNoContent, "")
	}
}

func registerRoutes() {
	web.Router("/", &controllers.AuthController{}, "get:Health")

	auth := &controllers.AuthController{}
	web.Router("/api/auth/login", auth, "post:Login")
	web.Router("/api/auth/register", auth, "post:Register")
	web.Router("/api/auth/me", auth, "get:Me")
	web.Router("/api/auth/profile", auth, "put:UpdateProfile")
	web.Router("/api/auth/password", auth, "put:ChangePassword")

	accounts := &controllers.AccountsController{}
	web.Router("/api/accounts/", accounts, "get:List;post:Create")
	web.Router("/api/accounts/:id", accounts, "get:Get;put:Update;delete:Delete")
	web.Router("/api/accounts/:id/check", accounts, "post:Check")

	contents := &controllers.ContentsController{}
	web.Router("/api/contents/", contents, "get:List;post:Create")
	web.Router("/api/contents/ai-generations", contents, "get:ListGenerations")
	web.Router("/api/contents/ai-generate", contents, "post:AIGenerate")
	web.Router("/api/contents/ai-generate-stream", contents, "post:AIGenerateStream")
	web.Router("/api/contents/:id", contents, "get:Get;put:Update;delete:Delete")

	dashboard := &controllers.DashboardController{}
	web.Router("/api/dashboard/stats", dashboard, "get:Stats")

	modelConfigs := &controllers.ModelConfigsController{}
	web.Router("/api/model-configs/", modelConfigs, "get:List;post:Create")
	web.Router("/api/model-configs/reorder", modelConfigs, "put:Reorder")
	web.Router("/api/model-configs/:config_id", modelConfigs, "put:Update;delete:Delete")

	models := &controllers.ModelsController{}
	web.Router("/api/models/fetch", models, "post:Fetch")
	web.Router("/api/models/test", models, "post:Test")

	notifications := &controllers.NotificationsController{}
	web.Router("/api/notifications/", notifications, "get:List")
	web.Router("/api/notifications/unread-count", notifications, "get:UnreadCount")
	web.Router("/api/notifications/type-counts", notifications, "get:TypeCounts")
	web.Router("/api/notifications/read-all", notifications, "post:MarkAllRead")
	web.Router("/api/notifications/stream", notifications, "get:Stream")
	web.Router("/api/notifications/:id/read", notifications, "post:MarkRead")

	notificationDict := &controllers.NotificationDictController{}
	web.Router("/api/notification-dict/groups", notificationDict, "get:ListGroups")
	web.Router("/api/notification-dict/", notificationDict, "get:List;post:Create")
	web.Router("/api/notification-dict/:id", notificationDict, "put:Update;delete:Delete")

	publish := &controllers.PublishController{}
	web.Router("/api/publish/tasks", publish, "get:ListTasks;post:CreateTask")
	web.Router("/api/publish/tasks/:id", publish, "get:GetTask")
	web.Router("/api/publish/tasks/:id/retry", publish, "post:RetryTask")
	web.Router("/api/publish/stats", publish, "get:Stats")

	reviews := &controllers.ReviewsController{}
	web.Router("/api/reviews/", reviews, "get:List")
	web.Router("/api/reviews/submit", reviews, "post:Submit")
	web.Router("/api/reviews/:id/approve", reviews, "post:Approve")
	web.Router("/api/reviews/:id/reject", reviews, "post:Reject")

	templates := &controllers.TemplatesController{}
	web.Router("/api/templates/", templates, "get:List;post:Create")
	web.Router("/api/templates/:id", templates, "get:Get;put:Update;delete:Delete")

	stores := &controllers.StoresController{}
	web.Router("/api/stores/", stores, "get:List;post:Create")
	web.Router("/api/stores/all", stores, "get:ListAll")
	web.Router("/api/stores/:store_id", stores, "get:Get;put:Update;delete:Delete")

	inspections := &controllers.InspectionsController{}
	web.Router("/api/inspections/", inspections, "get:List;post:Create")
	web.Router("/api/inspections/items", inspections, "get:ListItems")
	web.Router("/api/inspections/ai-analyze", inspections, "post:AIAnalyze")
	web.Router("/api/inspections/ai-analyze-stream", inspections, "post:AIAnalyzeStream")
	web.Router("/api/inspections/ai-analyze-item", inspections, "post:AIAnalyzeItem")
	web.Router("/api/inspections/ai-analyze-item-stream", inspections, "post:AIAnalyzeItemStream")
	web.Router("/api/inspections/:inspection_id", inspections, "get:Get;put:Update;delete:Delete")

	inspectionTasks := &controllers.InspectionTasksController{}
	web.Router("/api/inspection-tasks/", inspectionTasks, "get:List")
	web.Router("/api/inspection-tasks/:task_id", inspectionTasks, "get:Get")
	web.Router("/api/inspection-tasks/:task_id/submit-rectify", inspectionTasks, "post:SubmitRectify")
	web.Router("/api/inspection-tasks/:task_id/recheck", inspectionTasks, "post:Recheck")
	web.Router("/api/inspection-tasks/:task_id/manual-confirm", inspectionTasks, "post:ManualConfirm")

	inspectionTemplates := &controllers.InspectionTemplatesController{}
	web.Router("/api/inspection-templates/", inspectionTemplates, "get:List;post:Create")
	web.Router("/api/inspection-templates/all", inspectionTemplates, "get:ListAll")
	web.Router("/api/inspection-templates/:template_id", inspectionTemplates, "get:Get;put:Update;delete:Delete")

	inspectionMaterials := &controllers.InspectionMaterialsController{}
	web.Router("/api/inspection-materials/", inspectionMaterials, "get:List;post:Create")
	web.Router("/api/inspection-materials/:material_id", inspectionMaterials, "get:Get;put:Update;delete:Delete")

	uploads := &controllers.UploadsController{}
	web.Router("/api/uploads", uploads, "post:Upload")

	materials := &controllers.MaterialsController{}
	web.Router("/api/materials/", materials, "get:List;post:Create")
	web.Router("/api/materials/categories", materials, "get:Categories")
	web.Router("/api/materials/:material_id", materials, "get:Get;put:Update;delete:Delete")

	db := &controllers.DBController{}
	web.Router("/api/db/history", db, "get:ListHistory")
	web.Router("/api/db/execute", db, "post:Execute")

	dbChanges := &controllers.DBChangesController{}
	web.Router("/api/db-changes/submit", dbChanges, "post:Submit")
	web.Router("/api/db-changes/", dbChanges, "get:List")
	web.Router("/api/db-changes/:id/approve", dbChanges, "post:Approve")
	web.Router("/api/db-changes/:id/reject", dbChanges, "post:Reject")

	userCreationReviews := &controllers.UserCreationReviewsController{}
	web.Router("/api/user-creation-reviews/", userCreationReviews, "get:List")
	web.Router("/api/user-creation-reviews/:id/approve", userCreationReviews, "post:Approve")
	web.Router("/api/user-creation-reviews/:id/reject", userCreationReviews, "post:Reject")

	rbacUsers := &controllers.RBACUsersController{}
	web.Router("/api/v2/users", rbacUsers, "get:ListUsers;post:CreateUser")
	web.Router("/api/v2/users/:id", rbacUsers, "get:GetUser;put:UpdateUser;delete:DeleteUser")
	web.Router("/api/v2/users/:id/roles", rbacUsers, "put:UpdateUserRoles")
	web.Router("/api/v2/users/:id/password", rbacUsers, "put:ChangeUserPassword")
	web.Router("/api/v2/users/:id/permission-overrides", rbacUsers, "get:GetUserPermissionOverrides;put:UpdateUserPermissionOverrides;delete:ResetUserPermissionOverrides")
	web.Router("/api/v2/users/:id/password-status", rbacUsers, "get:GetPasswordStatus")
	web.Router("/api/v2/users/:id/permissions", rbacUsers, "get:GetUserPermissions")

	rbacRoles := &controllers.RBACRolesController{}
	web.Router("/api/v2/roles", rbacRoles, "get:ListRoles;post:CreateRole")
	web.Router("/api/v2/roles/:id", rbacRoles, "get:GetRole;put:UpdateRole;delete:DeleteRole")
	web.Router("/api/v2/roles/:id/parents", rbacRoles, "post:AddParentRole")
	web.Router("/api/v2/roles/:id/parents/:parent_id", rbacRoles, "delete:RemoveParentRole")
	web.Router("/api/v2/roles/:id/permissions", rbacRoles, "get:GetRolePermissions;put:UpdateRolePermissions")
	web.Router("/api/v2/roles/:id/permissions/direct", rbacRoles, "get:GetRoleDirectPermissions")
	web.Router("/api/v2/roles/:id/permissions/detail", rbacRoles, "get:GetRolePermissionsDetail")
	web.Router("/api/v2/roles/:id/permissions/preview/:parent_role_id", rbacRoles, "get:PreviewInheritedPermissions")
	web.Router("/api/v2/roles/:id/inheritance", rbacRoles, "get:GetRoleInheritance")

	rbacPermissions := &controllers.RBACPermissionsController{}
	web.Router("/api/v2/me/permissions", rbacPermissions, "get:GetMyPermissions")
	web.Router("/api/v2/permission-enums", rbacPermissions, "get:ListPermissionEnums")
	web.Router("/api/v2/permission-pages", rbacPermissions, "get:ListPagePermissions")
	web.Router("/api/v2/resources", rbacPermissions, "get:ListResources;post:CreateResource")
	web.Router("/api/v2/resources/:id", rbacPermissions, "put:UpdateResource;delete:DeleteResource")
	web.Router("/api/v2/permissions", rbacPermissions, "get:ListPermissions;post:CreatePermission")
	web.Router("/api/v2/permissions/:id", rbacPermissions, "put:UpdatePermission;delete:DeletePermission")

	rbacConstraints := &controllers.RBACConstraintsController{}
	web.Router("/api/v2/constraints", rbacConstraints, "get:ListConstraints;post:CreateConstraint")
	web.Router("/api/v2/constraints/:id", rbacConstraints, "get:GetConstraint;put:UpdateConstraint;delete:DeleteConstraint")
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}
	if err := services.InitDatabase(dbPath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	if err := services.SeedUsers(); err != nil {
		log.Fatalf("初始化用户失败: %v", err)
	}
	if err := services.InitRBACSystem(); err != nil {
		log.Fatalf("初始化 RBAC 系统失败: %v", err)
	}
	if err := services.SeedNotificationDict(); err != nil {
		log.Fatalf("初始化通知字典失败: %v", err)
	}
	if err := services.SeedInspectionItems(); err != nil {
		log.Fatalf("初始化巡店检查项失败: %v", err)
	}
	if err := services.SeedInspectionTemplates(); err != nil {
		log.Fatalf("初始化巡店检查表模板失败: %v", err)
	}
	if err := services.SeedInspectionTemplatesBulk(); err != nil {
		log.Fatalf("批量初始化检查表模板失败: %v", err)
	}
	if err := services.SeedInspectionMaterialsBulk(); err != nil {
		log.Fatalf("批量初始化检查项素材失败: %v", err)
	}

	web.SetStaticPath("/uploads", services.GetUploadDir())
	web.InsertFilter("*", web.BeforeRouter, corsFilter)
	web.InsertFilter("*", web.BeforeRouter, middleware.AuthRequired)

	registerRoutes()
	web.Run()
}
