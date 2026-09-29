package safetyincident

import (
	"testing"

	"github.com/google/uuid"
)

func TestAuthorizeRestrictedActionsByRoleAndAssignment(t *testing.T) {
	t.Parallel()

	incidentID := uuid.New()
	otherIncidentID := uuid.New()
	tests := []struct {
		name       string
		actor      Actor
		permission Permission
		incidentID uuid.UUID
		want       bool
	}{
		{name: "moderator reads safe metadata", actor: Actor{ID: "moderator", Role: RoleModerator}, permission: PermissionIncidentReadSafe, incidentID: incidentID, want: true},
		{name: "moderator cannot access bytes", actor: Actor{ID: "moderator", Role: RoleModerator}, permission: PermissionEvidenceAccess, incidentID: incidentID},
		{name: "assigned helper reads assigned incident", actor: Actor{ID: "helper", Role: RoleHelper, AssignedIncidents: map[uuid.UUID]bool{incidentID: true}}, permission: PermissionIncidentReadSafe, incidentID: incidentID, want: true},
		{name: "helper cannot read unassigned incident", actor: Actor{ID: "helper", Role: RoleHelper, AssignedIncidents: map[uuid.UUID]bool{incidentID: true}}, permission: PermissionIncidentReadSafe, incidentID: otherIncidentID},
		{name: "helper cannot access bytes even when assigned", actor: Actor{ID: "helper", Role: RoleHelper, AssignedIncidents: map[uuid.UUID]bool{incidentID: true}}, permission: PermissionEvidenceAccess, incidentID: incidentID},
		{name: "safety administrator accesses bytes", actor: Actor{ID: "admin", Role: RoleSafetyAdministrator}, permission: PermissionEvidenceAccess, incidentID: incidentID, want: true},
		{name: "safety administrator manages holds", actor: Actor{ID: "admin", Role: RoleSafetyAdministrator}, permission: PermissionHoldManage, incidentID: incidentID, want: true},
		{name: "safety administrator reports to authority", actor: Actor{ID: "admin", Role: RoleSafetyAdministrator}, permission: PermissionAuthorityReport, incidentID: incidentID, want: true},
		{name: "safety administrator approves disclosure", actor: Actor{ID: "admin", Role: RoleSafetyAdministrator}, permission: PermissionDisclosureApprove, incidentID: incidentID, want: true},
		{name: "empty actor denied", actor: Actor{Role: RoleSafetyAdministrator}, permission: PermissionEvidenceAccess, incidentID: incidentID},
		{name: "unknown permission denied", actor: Actor{ID: "admin", Role: RoleSafetyAdministrator}, permission: Permission("unknown"), incidentID: incidentID},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.actor.Allowed(test.permission, test.incidentID); got != test.want {
				t.Fatalf("Allowed(%q) = %t, want %t", test.permission, got, test.want)
			}
		})
	}
}
