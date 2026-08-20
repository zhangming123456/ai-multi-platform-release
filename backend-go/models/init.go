package models

import (
	"github.com/beego/beego/v2/client/orm"
)

func InitModels() {
	orm.RegisterModel(
		new(User),
		new(Account),
		new(Content),
		new(PublishTask),
		new(Template),
		new(ModelConfig),
		new(AIGenerationRecord),
		new(Campaign),
		new(Notification),
		new(NotificationDict),
		new(SqlHistory),
		new(SqlChangeRequest),
		new(UserCreationRequest),
		new(RBACRole),
		new(RBACResource),
		new(RBACPermission),
		new(RBACRoleHierarchy),
		new(RBACRolePermission),
		new(RBACUserRoleAssignment),
		new(RBACUserPermissionOverride),
		new(RBACConstraint),
		new(RBACConstraintRoleAssociation),
		new(Store),
		new(InspectionItem),
		new(Inspection),
		new(InspectionScore),
		new(InspectionTemplate),
		new(InspectionTemplateItem),
		new(InspectionMaterial),
		new(Material),
		new(InspectionTask),
		new(InspectionTaskItem),
		new(InspectionTaskLog),
	)
}
