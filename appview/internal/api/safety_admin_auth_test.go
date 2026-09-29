package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/safetyincident"
)

func TestSafetyAdminAuthorizationEnforcesRoleAndAssignment(t *testing.T) {
	incidentID := uuid.New()
	tests := []struct {
		name       string
		moderator  ctxkeys.Moderator
		permission safetyincident.Permission
		wantStatus int
	}{
		{name: "ordinary moderator safe read", moderator: ctxkeys.Moderator{ActorID: "mod", SourceSystem: "admin-api", Role: string(safetyincident.RoleModerator)}, permission: safetyincident.PermissionIncidentReadSafe, wantStatus: http.StatusNoContent},
		{name: "ordinary moderator denied evidence", moderator: ctxkeys.Moderator{ActorID: "mod", SourceSystem: "admin-api", Role: string(safetyincident.RoleModerator)}, permission: safetyincident.PermissionEvidenceAccess, wantStatus: http.StatusForbidden},
		{name: "assigned helper safe read", moderator: ctxkeys.Moderator{ActorID: "helper", SourceSystem: "admin-api", Role: string(safetyincident.RoleHelper), AssignedIncidentReferences: map[string]bool{incidentID.String(): true}}, permission: safetyincident.PermissionIncidentReadSafe, wantStatus: http.StatusNoContent},
		{name: "unassigned helper denied", moderator: ctxkeys.Moderator{ActorID: "helper", SourceSystem: "admin-api", Role: string(safetyincident.RoleHelper)}, permission: safetyincident.PermissionIncidentReadSafe, wantStatus: http.StatusForbidden},
		{name: "administrator evidence", moderator: ctxkeys.Moderator{ActorID: "admin", SourceSystem: "admin-api", Role: string(safetyincident.RoleSafetyAdministrator)}, permission: safetyincident.PermissionEvidenceAccess, wantStatus: http.StatusNoContent},
		{name: "administrator hold", moderator: ctxkeys.Moderator{ActorID: "admin", SourceSystem: "admin-api", Role: string(safetyincident.RoleSafetyAdministrator)}, permission: safetyincident.PermissionHoldManage, wantStatus: http.StatusNoContent},
		{name: "administrator authority report", moderator: ctxkeys.Moderator{ActorID: "admin", SourceSystem: "admin-api", Role: string(safetyincident.RoleSafetyAdministrator)}, permission: safetyincident.PermissionAuthorityReport, wantStatus: http.StatusNoContent},
		{name: "administrator disclosure", moderator: ctxkeys.Moderator{ActorID: "admin", SourceSystem: "admin-api", Role: string(safetyincident.RoleSafetyAdministrator)}, permission: safetyincident.PermissionDisclosureApprove, wantStatus: http.StatusNoContent},
		{name: "administrator without explicit permission denied", moderator: ctxkeys.Moderator{ActorID: "admin", SourceSystem: "admin-api", Role: string(safetyincident.RoleSafetyAdministrator), Permissions: map[string]bool{}}, permission: safetyincident.PermissionEvidenceAccess, wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/incidents/"+incidentID.String(), nil)
			request.SetPathValue("incidentReference", incidentID.String())
			request = request.WithContext(ctxkeys.WithModerator(request.Context(), test.moderator))
			recorder := httptest.NewRecorder()
			handler := RequireSafetyPermission(test.permission)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", recorder.Code, recorder.Body.String(), test.wantStatus)
			}
		})
	}
}
