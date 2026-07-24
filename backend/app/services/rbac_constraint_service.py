from __future__ import annotations

from datetime import datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.rbac_constraint import RBACConstraint, RBACConstraintRoleAssociation
from app.models.rbac_role import RBACRole
from app.models.rbac_user_role_assignment import RBACUserRoleAssignment
from app.services.rbac_service import get_role_descendants


async def validate_role_hierarchy(
    parent_role_id: str, child_role_id: str, db: AsyncSession
) -> bool:
    """Return True if adding parent -> child would not create a cycle.

    A role cannot be its own parent, and a child cannot already be a
    descendant of the proposed parent.
    """
    if parent_role_id == child_role_id:
        return False

    descendants = await get_role_descendants(parent_role_id, db)
    descendant_ids = {role.id for role in descendants}
    return child_role_id not in descendant_ids


async def validate_user_role_assignments(
    user_id: str, proposed_role_ids: set[str], db: AsyncSession
) -> list[str]:
    """Return human-readable constraint violations for a proposed role set.

    The effective role set merges the user's currently valid assignments with
    ``proposed_role_ids``. Callers that are replacing roles should pass the
    full replacement set; callers that are adding roles should pass the union
    of existing and new role IDs.
    """
    now = datetime.utcnow()

    active_constraints_result = await db.execute(
        select(RBACConstraint).where(RBACConstraint.is_active.is_(True))
    )
    active_constraints = active_constraints_result.scalars().all()
    if not active_constraints:
        return []

    constraint_ids = {constraint.id for constraint in active_constraints}
    associations_result = await db.execute(
        select(RBACConstraintRoleAssociation).where(
            RBACConstraintRoleAssociation.constraint_id.in_(constraint_ids)
        )
    )
    associations = associations_result.scalars().all()

    associations_by_constraint: dict[str, dict[str, set[str]]] = {}
    for association in associations:
        by_type = associations_by_constraint.setdefault(association.constraint_id, {})
        by_type.setdefault(association.association_type, set()).add(association.role_id)

    existing_assignments_result = await db.execute(
        select(RBACUserRoleAssignment).where(
            RBACUserRoleAssignment.user_id == user_id,
            (
                RBACUserRoleAssignment.valid_from.is_(None)
                | (RBACUserRoleAssignment.valid_from <= now)
            ),
            (
                RBACUserRoleAssignment.valid_until.is_(None)
                | (RBACUserRoleAssignment.valid_until >= now)
            ),
        )
    )
    existing_role_ids = {
        assignment.role_id for assignment in existing_assignments_result.scalars().all()
    }
    effective_role_ids = existing_role_ids | proposed_role_ids

    all_relevant_role_ids: set[str] = set(effective_role_ids)
    for constraint in active_constraints:
        for role_id_set in associations_by_constraint.get(constraint.id, {}).values():
            all_relevant_role_ids.update(role_id_set)

    role_name_map: dict[str, str] = {}
    if all_relevant_role_ids:
        role_rows_result = await db.execute(
            select(RBACRole.id, RBACRole.name).where(
                RBACRole.id.in_(all_relevant_role_ids)
            )
        )
        role_name_map = {
            row[0]: row[1] for row in role_rows_result.all() if row[1] is not None
        }

    violations: list[str] = []

    for constraint in active_constraints:
        groups = associations_by_constraint.get(constraint.id, {})
        subject_role_ids = groups.get("subject", set())
        prerequisite_role_ids = groups.get("prerequisite", set())

        if constraint.constraint_type == "mutual_exclusive":
            conflict_ids = subject_role_ids & effective_role_ids
            if len(conflict_ids) >= 2:
                names = _join_role_names(sorted(conflict_ids), role_name_map)
                violations.append(f"角色 {names} 互斥，不能同时分配")

        elif constraint.constraint_type == "prerequisite":
            missing_prerequisite_ids = prerequisite_role_ids - effective_role_ids
            if missing_prerequisite_ids:
                for subject_id in subject_role_ids & effective_role_ids:
                    subject_name = role_name_map.get(subject_id, subject_id)
                    missing_names = _join_role_names(
                        sorted(missing_prerequisite_ids), role_name_map
                    )
                    violations.append(
                        f"拥有角色 {subject_name} 需要先拥有角色 {missing_names}"
                    )

        elif constraint.constraint_type == "cardinality":
            config = constraint.config or {}
            max_users = config.get("max_users")
            if max_users is None or not subject_role_ids:
                continue

            current_users_result = await db.execute(
                select(RBACUserRoleAssignment.user_id)
                .where(
                    RBACUserRoleAssignment.role_id.in_(subject_role_ids),
                    (
                        RBACUserRoleAssignment.valid_from.is_(None)
                        | (RBACUserRoleAssignment.valid_from <= now)
                    ),
                    (
                        RBACUserRoleAssignment.valid_until.is_(None)
                        | (RBACUserRoleAssignment.valid_until >= now)
                    ),
                )
                .distinct()
            )
            current_user_ids = set(current_users_result.scalars().all())

            user_already_counted = user_id in current_user_ids
            would_receive_subject = bool(subject_role_ids & proposed_role_ids)
            projected_count = len(current_user_ids) + (
                1 if would_receive_subject and not user_already_counted else 0
            )

            if projected_count > max_users:
                names = _join_role_names(sorted(subject_role_ids), role_name_map)
                violations.append(f"角色 {names} 最多只能分配给 {max_users} 个用户")

    return violations


def _join_role_names(role_ids: list[str], role_name_map: dict[str, str]) -> str:
    names = [role_name_map.get(role_id, role_id) for role_id in role_ids]
    return "、".join(names)
