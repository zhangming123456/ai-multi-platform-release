package services

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-multi-platform-release/backend-go/models"
)

// ValidateRoleHierarchy returns true if adding parentRoleID -> childRoleID
// would not create a cycle.
func ValidateRoleHierarchy(parentRoleID, childRoleID string) (bool, error) {
	if parentRoleID == childRoleID {
		return false, nil
	}
	descendants, err := GetRoleDescendants(childRoleID)
	if err != nil {
		return false, err
	}
	for _, d := range descendants {
		if d.ID == parentRoleID {
			return false, nil
		}
	}
	return true, nil
}

// ValidateUserRoleAssignments returns human-readable constraint violations for
// a proposed role set. The effective role set merges the user's currently
// valid assignments with proposedRoleIDs.
func ValidateUserRoleAssignments(userID string, proposedRoleIDs map[string]bool) ([]string, error) {
	o := GetOrm()
	now := time.Now()

	var activeConstraints []models.RBACConstraint
	_, err := o.QueryTable(new(models.RBACConstraint)).
		Filter("is_active", true).
		All(&activeConstraints)
	if err != nil {
		return nil, err
	}
	if len(activeConstraints) == 0 {
		return nil, nil
	}

	constraintIDs := make([]string, 0, len(activeConstraints))
	for _, c := range activeConstraints {
		constraintIDs = append(constraintIDs, c.ID)
	}

	var associations []models.RBACConstraintRoleAssociation
	_, err = o.QueryTable(new(models.RBACConstraintRoleAssociation)).
		Filter("constraint_id__in", constraintIDs).
		All(&associations)
	if err != nil {
		return nil, err
	}

	associationsByConstraint := map[string]map[string]map[string]bool{}
	for _, a := range associations {
		byType, ok := associationsByConstraint[a.ConstraintID]
		if !ok {
			byType = map[string]map[string]bool{}
			associationsByConstraint[a.ConstraintID] = byType
		}
		roleSet, ok := byType[a.AssociationType]
		if !ok {
			roleSet = map[string]bool{}
			byType[a.AssociationType] = roleSet
		}
		roleSet[a.RoleID] = true
	}

	var existingAssignments []models.RBACUserRoleAssignment
	_, err = o.QueryTable(new(models.RBACUserRoleAssignment)).
		Filter("user_id", userID).
		All(&existingAssignments)
	if err != nil {
		return nil, err
	}
	existingRoleIDs := map[string]bool{}
	for _, a := range existingAssignments {
		if a.ValidFrom != nil && a.ValidFrom.After(now) {
			continue
		}
		if a.ValidUntil != nil && a.ValidUntil.Before(now) {
			continue
		}
		existingRoleIDs[a.RoleID] = true
	}

	effectiveRoleIDs := map[string]bool{}
	for id := range existingRoleIDs {
		effectiveRoleIDs[id] = true
	}
	for id := range proposedRoleIDs {
		effectiveRoleIDs[id] = true
	}

	allRelevantRoleIDs := map[string]bool{}
	for id := range effectiveRoleIDs {
		allRelevantRoleIDs[id] = true
	}
	for _, constraint := range activeConstraints {
		for _, roleSet := range associationsByConstraint[constraint.ID] {
			for id := range roleSet {
				allRelevantRoleIDs[id] = true
			}
		}
	}

	roleNameMap := map[string]string{}
	if len(allRelevantRoleIDs) > 0 {
		var roles []models.RBACRole
		ids := make([]string, 0, len(allRelevantRoleIDs))
		for id := range allRelevantRoleIDs {
			ids = append(ids, id)
		}
		_, err = o.QueryTable(new(models.RBACRole)).
			Filter("id__in", ids).
			All(&roles)
		if err != nil {
			return nil, err
		}
		for _, r := range roles {
			if r.Name != "" {
				roleNameMap[r.ID] = r.Name
			}
		}
	}

	var violations []string

	for _, constraint := range activeConstraints {
		groups := associationsByConstraint[constraint.ID]
		subjectRoleIDs := groups["subject"]
		prerequisiteRoleIDs := groups["prerequisite"]

		switch constraint.ConstraintType {
		case "mutual_exclusive":
			conflictIDs := []string{}
			for id := range subjectRoleIDs {
				if effectiveRoleIDs[id] {
					conflictIDs = append(conflictIDs, id)
				}
			}
			if len(conflictIDs) >= 2 {
				sort.Strings(conflictIDs)
				violations = append(violations, "角色 "+joinRoleNames(conflictIDs, roleNameMap)+" 互斥，不能同时分配")
			}

		case "prerequisite":
			missingPrerequisiteIDs := []string{}
			for id := range prerequisiteRoleIDs {
				if !effectiveRoleIDs[id] {
					missingPrerequisiteIDs = append(missingPrerequisiteIDs, id)
				}
			}
			if len(missingPrerequisiteIDs) > 0 {
				sortedSubject := make([]string, 0, len(subjectRoleIDs))
				for id := range subjectRoleIDs {
					if effectiveRoleIDs[id] {
						sortedSubject = append(sortedSubject, id)
					}
				}
				sort.Strings(sortedSubject)
				sort.Strings(missingPrerequisiteIDs)
				for _, subjectID := range sortedSubject {
					subjectName := roleNameMap[subjectID]
					if subjectName == "" {
						subjectName = subjectID
					}
					missingNames := joinRoleNames(missingPrerequisiteIDs, roleNameMap)
					violations = append(violations, "拥有角色 "+subjectName+" 需要先拥有角色 "+missingNames)
				}
			}

		case "cardinality":
			config := map[string]interface{}{}
			if constraint.Config != "" {
				_ = json.Unmarshal([]byte(constraint.Config), &config)
			}
			maxUsers, ok := config["max_users"].(float64)
			if !ok || len(subjectRoleIDs) == 0 {
				continue
			}

			var assigned []models.RBACUserRoleAssignment
			subjectIDs := make([]string, 0, len(subjectRoleIDs))
			for id := range subjectRoleIDs {
				subjectIDs = append(subjectIDs, id)
			}
			_, err = o.QueryTable(new(models.RBACUserRoleAssignment)).
				Filter("role_id__in", subjectIDs).
				All(&assigned)
			if err != nil {
				return nil, err
			}
			currentUserIDs := map[string]bool{}
			for _, a := range assigned {
				if a.ValidFrom != nil && a.ValidFrom.After(now) {
					continue
				}
				if a.ValidUntil != nil && a.ValidUntil.Before(now) {
					continue
				}
				currentUserIDs[a.UserID] = true
			}

			userAlreadyCounted := currentUserIDs[userID]
			wouldReceiveSubject := false
			for id := range proposedRoleIDs {
				if subjectRoleIDs[id] {
					wouldReceiveSubject = true
					break
				}
			}
			projectedCount := len(currentUserIDs)
			if wouldReceiveSubject && !userAlreadyCounted {
				projectedCount++
			}

			if projectedCount > int(maxUsers) {
				names := joinRoleNames(subjectIDs, roleNameMap)
				violations = append(violations, "角色 "+names+" 最多只能分配给 "+strconv.Itoa(int(maxUsers))+" 个用户")
			}
		}
	}

	return violations, nil
}

func joinRoleNames(roleIDs []string, roleNameMap map[string]string) string {
	names := make([]string, 0, len(roleIDs))
	for _, id := range roleIDs {
		name := roleNameMap[id]
		if name == "" {
			name = id
		}
		names = append(names, name)
	}
	return strings.Join(names, "、")
}
