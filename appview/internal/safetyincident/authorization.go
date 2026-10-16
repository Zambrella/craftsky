package safetyincident

import "github.com/google/uuid"

type Role string

const (
	PermissionModerationRead    Permission = "moderation.read"
	PermissionModerationManage  Permission = "moderation.manage"
	PermissionSafetyRead        Permission = "safety.read"
	PermissionIntakeManage      Permission = "intake.manage"
	PermissionEligibilityReview Permission = "eligibility.review"
	PermissionOperatorManage    Permission = "operator.manage"
	RoleModerator               Role       = "moderator"
	RoleHelper                  Role       = "helper"
	RoleSafetyAdministrator     Role       = "safetyAdministrator"
)

// OperatorPermissions is the explicit grant vocabulary. Role alone never grants
// these permissions to an authenticated database operator.
func OperatorPermissions() []Permission {
	return []Permission{PermissionModerationRead, PermissionModerationManage, PermissionSafetyRead,
		PermissionIntakeManage, PermissionEligibilityReview, PermissionOperatorManage,
		PermissionIncidentReadSafe, PermissionIncidentConfirm, PermissionEvidenceAccess,
		PermissionEvidencePreserve, PermissionHoldManage, PermissionAuthorityReport,
		PermissionDisclosureApprove, PermissionScanRetry, PermissionWorkflowManage}
}

func (role Role) AllowsPermission(permission Permission) bool {
	switch role {
	case RoleSafetyAdministrator:
		for _, known := range OperatorPermissions() {
			if permission == known {
				return true
			}
		}
	case RoleModerator:
		switch permission {
		case PermissionModerationRead, PermissionModerationManage, PermissionSafetyRead,
			PermissionIntakeManage, PermissionIncidentReadSafe, PermissionIncidentConfirm, PermissionScanRetry:
			return true
		}
	case RoleHelper:
		return permission == PermissionIncidentReadSafe
	}
	return false
}

type Permission string

const (
	PermissionIncidentReadSafe  Permission = "incident.readSafe"
	PermissionIncidentConfirm   Permission = "incident.confirm"
	PermissionEvidenceAccess    Permission = "evidence.access"
	PermissionEvidencePreserve  Permission = "evidence.preserve"
	PermissionHoldManage        Permission = "hold.manage"
	PermissionAuthorityReport   Permission = "authority.report"
	PermissionDisclosureApprove Permission = "disclosure.approve"
	PermissionScanRetry         Permission = "scan.retry"
	PermissionWorkflowManage    Permission = "workflow.manage"
)

type Actor struct {
	ID                string
	Role              Role
	Permissions       map[Permission]bool
	AssignedIncidents map[uuid.UUID]bool
}

func (actor Actor) Allowed(permission Permission, incidentID uuid.UUID) bool {
	if incidentID == uuid.Nil || !actor.HasPermission(permission) {
		return false
	}
	return actor.Role != RoleHelper || actor.AssignedIncidents[incidentID]
}

func (actor Actor) HasPermission(permission Permission) bool {
	if actor.ID == "" || !actor.Role.AllowsPermission(permission) {
		return false
	}
	if actor.Permissions != nil {
		return actor.Permissions[permission]
	}
	switch actor.Role {
	case RoleSafetyAdministrator:
		switch permission {
		case PermissionIncidentReadSafe, PermissionIncidentConfirm, PermissionEvidenceAccess,
			PermissionEvidencePreserve, PermissionHoldManage, PermissionAuthorityReport,
			PermissionDisclosureApprove, PermissionScanRetry, PermissionWorkflowManage:
			return true
		}
	case RoleModerator:
		return permission == PermissionIncidentReadSafe || permission == PermissionIncidentConfirm
	case RoleHelper:
		return permission == PermissionIncidentReadSafe
	}
	return false
}
