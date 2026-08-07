package services

import (
	"time"

	"ai-multi-platform-release/backend-go/models"
)

type PermissionAccess struct {
	Read  bool
	Write bool
}

type UserEffectivePermissions struct {
	AccessMap    map[string]*PermissionAccess
	GrantedKeys  map[string]bool
	HasOverrides bool
	IsSuperAdmin bool
}

var READ_OPERATIONS = map[string]bool{"read": true, "execute": true, "custom_permissions": true}
var WRITE_OPERATIONS = map[string]bool{"create": true, "update": true, "delete": true, "approve": true, "reject": true, "execute": true, "custom_permissions": true}

var ACTION_PARENT_PAGE = map[string]string{
	"content":    "content:read",
	"publish":    "publish:read",
	"templates":  "templates:read",
	"review":     "review:read",
	"db_change":  "sql_review:read",
	"db":         "db:read",
	"db_history": "db:read",
}

func PermissionScope(permKey string) string {
	parts := splitKey(permKey)
	if len(parts) >= 3 {
		return parts[2]
	}
	if len(parts) == 2 {
		return parts[1]
	}
	return "read"
}

func splitKey(key string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(key); i++ {
		if i == len(key) || key[i] == ':' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	return parts
}

func ResolveReadKeys(permKey string) []string {
	parts := splitKey(permKey)
	if len(parts) == 2 {
		return []string{permKey}
	}
	if len(parts) == 3 {
		scope := parts[2]
		if scope == "read" {
			return []string{permKey}
		}
		return []string{permKey, parts[0] + ":" + parts[1] + ":read"}
	}
	return []string{permKey}
}

func ResolveWriteKeys(permKey string) []string {
	parts := splitKey(permKey)
	if len(parts) == 2 {
		return []string{permKey}
	}
	if len(parts) == 3 {
		if parts[2] == "write" {
			return []string{permKey}
		}
	}
	return []string{permKey}
}

func ImpliedReadKeys(permKey string) []string {
	parts := splitKey(permKey)
	if len(parts) != 3 || parts[2] != "write" {
		return nil
	}
	var implied []string
	if parts[1] != "read" {
		implied = append(implied, parts[0]+":"+parts[1]+":read")
	}
	if parent, ok := ACTION_PARENT_PAGE[parts[0]]; ok {
		implied = append(implied, parent)
	}
	return implied
}

func GetPermissionDisplayNames(keys []string) (map[string]string, error) {
	names := map[string]string{}
	if len(keys) == 0 {
		return names, nil
	}
	o := GetOrm()
	var perms []models.RBACPermission
	unique := map[string]bool{}
	for _, k := range keys {
		unique[k] = true
	}
	var keyList []string
	for k := range unique {
		keyList = append(keyList, k)
	}
	if len(keyList) == 0 {
		return names, nil
	}
	_, err := o.QueryTable(new(models.RBACPermission)).
		Filter("key__in", keyList).
		Filter("is_active", true).
		All(&perms)
	if err != nil {
		return names, err
	}
	resourceIDs := map[string]bool{}
	for _, p := range perms {
		resourceIDs[p.ResourceID] = true
	}
	var resList []string
	for rid := range resourceIDs {
		resList = append(resList, rid)
	}
	resourceNames := map[string]string{}
	if len(resList) > 0 {
		var resources []models.RBACResource
		_, err = o.QueryTable(new(models.RBACResource)).
			Filter("id__in", resList).
			All(&resources)
		if err == nil {
			for _, r := range resources {
				resourceNames[r.ID] = r.Name
			}
		}
	}

	for _, p := range perms {
		name := resourceNames[p.ResourceID]
		if name == "" {
			name = p.Key
		}
		parts := splitKey(p.Key)
		isAction := len(parts) == 3
		if isAction && parts[2] == "read" {
			if len(name) < 2 || name[:2] != "查看" {
				names[p.Key] = "查看" + name
			} else {
				names[p.Key] = name
			}
		} else {
			names[p.Key] = name
		}
	}

	for _, key := range keys {
		if _, ok := names[key]; ok {
			continue
		}
		parts := splitKey(key)
		if len(parts) != 3 {
			continue
		}
		resource, op, scope := parts[0], parts[1], parts[2]
		counterpartKey := resource + ":" + op + ":write"
		if scope == "read" {
			counterpartKey = resource + ":" + op + ":write"
		} else {
			counterpartKey = resource + ":" + op + ":read"
		}
		if counterpart, ok := names[counterpartKey]; ok {
			if scope == "read" {
				if len(counterpart) < 2 || counterpart[:2] != "查看" {
					names[key] = "查看" + counterpart
				} else {
					names[key] = counterpart
				}
			} else {
				names[key] = counterpart
			}
		}
	}
	return names, nil
}

func GetRoleAncestors(roleID string) ([]models.RBACRole, error) {
	o := GetOrm()
	var ancestors []models.RBACRole
	seen := map[string]bool{roleID: true}
	frontier := map[string]bool{roleID: true}

	for len(frontier) > 0 {
		var frontierList []string
		for id := range frontier {
			frontierList = append(frontierList, id)
		}
		var roles []models.RBACRole
		var hierarchies []models.RBACRoleHierarchy
		_, err := o.QueryTable(new(models.RBACRoleHierarchy)).
			Filter("child_role_id__in", frontierList).
			All(&hierarchies)
		if err != nil {
			return ancestors, err
		}
		parentIDs := map[string]bool{}
		for _, h := range hierarchies {
			parentIDs[h.ParentRoleID] = true
		}
		if len(parentIDs) == 0 {
			break
		}
		var pidList []string
		for id := range parentIDs {
			pidList = append(pidList, id)
		}
		_, err = o.QueryTable(new(models.RBACRole)).
			Filter("id__in", pidList).
			All(&roles)
		if err != nil {
			return ancestors, err
		}
		frontier = map[string]bool{}
		for _, role := range roles {
			if seen[role.ID] {
				continue
			}
			seen[role.ID] = true
			ancestors = append(ancestors, role)
			frontier[role.ID] = true
		}
	}
	return ancestors, nil
}

func GetRoleDescendants(roleID string) ([]models.RBACRole, error) {
	o := GetOrm()
	var descendants []models.RBACRole
	seen := map[string]bool{roleID: true}
	frontier := map[string]bool{roleID: true}

	for len(frontier) > 0 {
		var frontierList []string
		for id := range frontier {
			frontierList = append(frontierList, id)
		}
		var hierarchies []models.RBACRoleHierarchy
		_, err := o.QueryTable(new(models.RBACRoleHierarchy)).
			Filter("parent_role_id__in", frontierList).
			All(&hierarchies)
		if err != nil {
			return descendants, err
		}
		childIDs := map[string]bool{}
		for _, h := range hierarchies {
			childIDs[h.ChildRoleID] = true
		}
		if len(childIDs) == 0 {
			break
		}
		var cidList []string
		for id := range childIDs {
			cidList = append(cidList, id)
		}
		var roles []models.RBACRole
		_, err = o.QueryTable(new(models.RBACRole)).
			Filter("id__in", cidList).
			All(&roles)
		if err != nil {
			return descendants, err
		}
		frontier = map[string]bool{}
		for _, role := range roles {
			if seen[role.ID] {
				continue
			}
			seen[role.ID] = true
			descendants = append(descendants, role)
			frontier[role.ID] = true
		}
	}
	return descendants, nil
}

func ComputeUserEffectivePermissions(userID string, activeRoleIDs []string) (*UserEffectivePermissions, error) {
	o := GetOrm()
	now := time.Now()

	var assignments []models.RBACUserRoleAssignment
	qs := o.QueryTable(new(models.RBACUserRoleAssignment)).Filter("user_id", userID)
	_, err := qs.All(&assignments)
	if err != nil {
		return nil, err
	}

	assignedRoleIDs := map[string]bool{}
	for _, a := range assignments {
		if a.ValidFrom != nil && a.ValidFrom.After(now) {
			continue
		}
		if a.ValidUntil != nil && a.ValidUntil.Before(now) {
			continue
		}
		assignedRoleIDs[a.RoleID] = true
	}
	if activeRoleIDs != nil {
		active := map[string]bool{}
		for _, id := range activeRoleIDs {
			active[id] = true
		}
		for id := range assignedRoleIDs {
			if !active[id] {
				delete(assignedRoleIDs, id)
			}
		}
	}

	empty := &UserEffectivePermissions{
		AccessMap:    map[string]*PermissionAccess{},
		GrantedKeys:  map[string]bool{},
		HasOverrides: false,
		IsSuperAdmin: false,
	}
	if len(assignedRoleIDs) == 0 {
		return empty, nil
	}

	superAdmin, err := closureHasSuperAdmin(assignedRoleIDs)
	if err != nil {
		return nil, err
	}
	if superAdmin {
		return superAdminPermissions(), nil
	}

	allRoleIDs := map[string]bool{}
	for roleID := range assignedRoleIDs {
		allRoleIDs[roleID] = true
		ancestors, err := GetRoleAncestors(roleID)
		if err != nil {
			return nil, err
		}
		for _, a := range ancestors {
			allRoleIDs[a.ID] = true
		}
	}

	superAdmin2, err := closureHasSuperAdmin(allRoleIDs)
	if err != nil {
		return nil, err
	}
	if superAdmin2 {
		return superAdminPermissions(), nil
	}

	var roleIDList []string
	for id := range allRoleIDs {
		roleIDList = append(roleIDList, id)
	}

	var rolePerms []models.RBACRolePermission
	_, err = o.QueryTable(new(models.RBACRolePermission)).
		Filter("role_id__in", roleIDList).
		All(&rolePerms)
	if err != nil {
		return nil, err
	}
	permIDs := map[string]bool{}
	for _, rp := range rolePerms {
		permIDs[rp.PermissionID] = true
	}
	roleAccessMap := map[string]*PermissionAccess{}
	if len(permIDs) > 0 {
		var pidList []string
		for id := range permIDs {
			pidList = append(pidList, id)
		}
		var perms []models.RBACPermission
		_, err = o.QueryTable(new(models.RBACPermission)).
			Filter("id__in", pidList).
			Filter("is_active", true).
			All(&perms)
		if err != nil {
			return nil, err
		}
		permByID := map[string]models.RBACPermission{}
		for _, p := range perms {
			permByID[p.ID] = p
		}
		for _, rp := range rolePerms {
			perm, ok := permByID[rp.PermissionID]
			if !ok {
				continue
			}
			access := roleAccessMap[perm.Key]
			if access == nil {
				access = &PermissionAccess{}
				roleAccessMap[perm.Key] = access
			}
			scope := PermissionScope(perm.Key)
			opLower := perm.Operation
			switch scope {
			case "read":
				access.Read = true
			case "write":
				access.Write = true
			default:
				if READ_OPERATIONS[opLower] {
					access.Read = true
				}
				if WRITE_OPERATIONS[opLower] {
					access.Write = true
				}
			}
		}
	}

	var overrides []models.RBACUserPermissionOverride
	_, err = o.QueryTable(new(models.RBACUserPermissionOverride)).
		Filter("user_id", userID).
		All(&overrides)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return &UserEffectivePermissions{
			AccessMap:    roleAccessMap,
			GrantedKeys:  map[string]bool{},
			HasOverrides: false,
			IsSuperAdmin: false,
		}, nil
	}

	grantedKeys := map[string]bool{}
	for _, ov := range overrides {
		if ov.Granted {
			grantedKeys[ov.PermissionKey] = true
		}
	}
	return &UserEffectivePermissions{
		AccessMap:    roleAccessMap,
		GrantedKeys:  grantedKeys,
		HasOverrides: true,
		IsSuperAdmin: false,
	}, nil
}

func superAdminPermissions() *UserEffectivePermissions {
	var perms []models.RBACPermission
	_, err := GetOrm().QueryTable(new(models.RBACPermission)).
		Filter("is_active", true).
		All(&perms)
	if err != nil {
		return &UserEffectivePermissions{
			AccessMap:    map[string]*PermissionAccess{},
			GrantedKeys:  map[string]bool{},
			HasOverrides: false,
			IsSuperAdmin: true,
		}
	}
	accessMap := map[string]*PermissionAccess{}
	for _, p := range perms {
		accessMap[p.Key] = &PermissionAccess{Read: true, Write: true}
	}
	return &UserEffectivePermissions{
		AccessMap:    accessMap,
		GrantedKeys:  map[string]bool{},
		HasOverrides: false,
		IsSuperAdmin: true,
	}
}

func closureHasSuperAdmin(roleIDs map[string]bool) (bool, error) {
	if len(roleIDs) == 0 {
		return false, nil
	}
	var idList []string
	for id := range roleIDs {
		idList = append(idList, id)
	}
	var roles []models.RBACRole
	_, err := GetOrm().QueryTable(new(models.RBACRole)).
		Filter("id__in", idList).
		Filter("is_super_admin", true).
		All(&roles)
	if err != nil {
		return false, err
	}
	return len(roles) > 0, nil
}

func FlattenEffectivePermissions(accessMap map[string]*PermissionAccess) (map[string]string, error) {
	flat := map[string]bool{}
	for permKey, access := range accessMap {
		if access.Read || access.Write {
			for _, readKey := range ResolveReadKeys(permKey) {
				flat[readKey] = true
			}
		}
		if access.Write {
			for _, writeKey := range ResolveWriteKeys(permKey) {
				flat[writeKey] = true
			}
			for _, implied := range ImpliedReadKeys(permKey) {
				flat[implied] = true
			}
		}
	}
	if len(flat) == 0 {
		return map[string]string{}, nil
	}
	var keys []string
	for key := range flat {
		keys = append(keys, key)
	}
	names, err := GetPermissionDisplayNames(keys)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for key := range flat {
		if v, ok := names[key]; ok {
			result[key] = v
		} else {
			result[key] = key
		}
	}
	return result, nil
}

func GetUserEffectiveFlatPermissions(userID string, activeRoleIDs []string) (map[string]string, error) {
	bundle, err := ComputeUserEffectivePermissions(userID, activeRoleIDs)
	if err != nil {
		return nil, err
	}
	flat, err := FlattenEffectivePermissions(bundle.AccessMap)
	if err != nil {
		return nil, err
	}
	if bundle.IsSuperAdmin || !bundle.HasOverrides {
		return flat, nil
	}
	result := map[string]string{}
	for k, v := range flat {
		if bundle.GrantedKeys[k] {
			result[k] = v
		}
	}
	return result, nil
}

func HasPermission(userID, permissionKey, mode string, activeRoleIDs []string) (bool, error) {
	bundle, err := ComputeUserEffectivePermissions(userID, activeRoleIDs)
	if err != nil {
		return false, err
	}
	if bundle.IsSuperAdmin {
		return true, nil
	}
	flat, err := FlattenEffectivePermissions(bundle.AccessMap)
	if err != nil {
		return false, err
	}
	if !bundle.HasOverrides {
		inRole := flat[permissionKey]
		if inRole == "" && mode == "read" {
			parts := splitKey(permissionKey)
			if len(parts) == 3 && parts[2] == "read" {
				inRole = flat[parts[0]+":"+parts[1]+":write"]
			}
		}
		return inRole != "", nil
	}
	candidateKeys := map[string]bool{}
	if mode == "read" {
		for _, k := range ResolveReadKeys(permissionKey) {
			candidateKeys[k] = true
		}
		parts := splitKey(permissionKey)
		if len(parts) == 3 && parts[2] == "read" {
			candidateKeys[parts[0]+":"+parts[1]+":write"] = true
		}
	} else {
		for _, k := range ResolveWriteKeys(permissionKey) {
			candidateKeys[k] = true
		}
	}
	for key := range candidateKeys {
		if flat[key] != "" && bundle.GrantedKeys[key] {
			return true, nil
		}
	}
	return false, nil
}

func GetRoleAncestorIDs(roleID string) ([]string, error) {
	ancestors, err := GetRoleAncestors(roleID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(ancestors))
	for _, a := range ancestors {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

func GetRoleClosureIDs(userID string, activeRoleIDs []string) (map[string]bool, error) {
	o := GetOrm()
	now := time.Now()
	var assignments []models.RBACUserRoleAssignment
	_, err := o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("user_id", userID).
		All(&assignments)
	if err != nil {
		return nil, err
	}
	assigned := map[string]bool{}
	for _, a := range assignments {
		if a.ValidFrom != nil && a.ValidFrom.After(now) {
			continue
		}
		if a.ValidUntil != nil && a.ValidUntil.Before(now) {
			continue
		}
		assigned[a.RoleID] = true
	}
	if activeRoleIDs != nil {
		active := map[string]bool{}
		for _, id := range activeRoleIDs {
			active[id] = true
		}
		for id := range assigned {
			if !active[id] {
				delete(assigned, id)
			}
		}
	}
	closure := map[string]bool{}
	for roleID := range assigned {
		closure[roleID] = true
		ancestors, err := GetRoleAncestors(roleID)
		if err != nil {
			return nil, err
		}
		for _, a := range ancestors {
			closure[a.ID] = true
		}
	}
	return closure, nil
}
