package services

import (
	"strings"

	"ai-multi-platform-release/backend-go/models"
)

func ResolveField(groupKey, key string) string {
	o := GetOrm()
	var entry models.NotificationDict
	err := o.QueryTable(new(models.NotificationDict)).
		Filter("group_key", groupKey).
		Filter("dict_key", key).
		Filter("is_active", true).
		One(&entry)
	if err == nil && entry.DictValue != "" {
		return entry.DictValue
	}
	return key
}

func IsFieldPrivate(groupKey, key string) bool {
	o := GetOrm()
	var entry models.NotificationDict
	err := o.QueryTable(new(models.NotificationDict)).
		Filter("group_key", groupKey).
		Filter("dict_key", key).
		Filter("is_active", true).
		One(&entry)
	return err == nil && entry.IsPrivate
}

func MaskValue(val string) string {
	if val == "" {
		return ""
	}
	length := len(val)
	if length == 1 {
		return val
	}
	if length == 2 {
		return val[:1] + "*"
	}
	if length == 3 {
		return val[:1] + "*" + val[length-1:]
	}
	if length >= 9 {
		return val[:3] + "***" + val[length-2:]
	}
	return val[:2] + "***" + val[length-1:]
}

func CreateUserInfoChangeNotification(userID string, changedFields []map[string]string, actorNickname string) (*NotificationPayload, error) {
	if len(changedFields) == 0 {
		return nil, nil
	}
	var content string
	parts := make([]string, 0, len(changedFields))
	for _, item := range changedFields {
		field := item["field"]
		fieldLabel := ResolveField("user_fields", field)
		isPrivate := IsFieldPrivate("user_fields", field)
		oldVal := strings.TrimSpace(item["old"])
		newVal := strings.TrimSpace(item["new"])
		oldDisplay, newDisplay := oldVal, newVal
		if isPrivate {
			oldDisplay = MaskValue(oldVal)
			newDisplay = MaskValue(newVal)
		}
		switch {
		case oldVal != "" && newVal != "":
			parts = append(parts, fieldLabel+" 从「"+oldDisplay+"」更新为「"+newDisplay+"」")
		case newVal != "":
			parts = append(parts, fieldLabel+" 添加「"+newDisplay+"」")
		case oldVal != "":
			parts = append(parts, fieldLabel+" 移除「"+oldDisplay+"」")
		}
	}
	if len(parts) == 1 {
		content = "你的" + parts[0][:len(parts[0])] + "已被 " + actorNickname + " 修改"
		// rebuild single-field content with exact format
		item := changedFields[0]
		field := item["field"]
		fieldLabel := ResolveField("user_fields", field)
		isPrivate := IsFieldPrivate("user_fields", field)
		oldVal := strings.TrimSpace(item["old"])
		newVal := strings.TrimSpace(item["new"])
		oldDisplay, newDisplay := oldVal, newVal
		if isPrivate {
			oldDisplay = MaskValue(oldVal)
			newDisplay = MaskValue(newVal)
		}
		var detail string
		switch {
		case oldVal != "" && newVal != "":
			detail = "从「" + oldDisplay + "」更新为「" + newDisplay + "」"
		case newVal != "":
			detail = "添加「" + newDisplay + "」"
		case oldVal != "":
			detail = "移除「" + oldDisplay + "」"
		}
		if detail != "" {
			content = "你的" + fieldLabel + "已被 " + actorNickname + " 修改：" + detail
		} else {
			content = "你的" + fieldLabel + "已被 " + actorNickname + " 修改"
		}
	} else {
		content = "你的个人信息已被 " + actorNickname + " 修改：" + strings.Join(parts, "；")
	}
	return &NotificationPayload{
		Type:    models.NotificationTypeRoleUpdated,
		Title:   "个人信息已变更",
		Content: content,
	}, nil
}
