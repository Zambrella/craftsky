package api

import (
	"github.com/google/uuid"
	"net/http"
	"sort"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/safetyincident"
)

// RequireOperatorPermission is an explicit, fail-closed route grant. Domain
// services additionally enforce incident assignment and restricted-data rules.
func RequireOperatorPermission(permission safetyincident.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m, ok := ctxkeys.GetModerator(r.Context())
			if !ok {
				adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
				return
			}
			if m.Permissions == nil || !safetyincident.Role(m.Role).AllowsPermission(permission) || !m.Permissions[string(permission)] {
				middleware.RejectBodyWithoutDrain(w, r)
				adminError(w, r, 403, "operator_action_forbidden", "operator action forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func OperatorIdentityHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, ok := ctxkeys.GetModerator(r.Context())
		if !ok {
			adminError(w, r, 401, "moderator_authentication_failed", "moderator authentication failed")
			return
		}
		permissions := map[string]bool{}
		for _, p := range safetyincident.OperatorPermissions() {
			permissions[string(p)] = m.Permissions[string(p)] && safetyincident.Role(m.Role).AllowsPermission(p)
		}
		assignments := []string{}
		for id, assigned := range m.AssignedIncidentReferences {
			if assigned {
				assignments = append(assignments, id)
			}
		}
		sort.Strings(assignments)
		w.Header().Set("Cache-Control", "no-store")
		body := map[string]any{"actorId": m.ActorID, "role": m.Role, "permissions": permissions, "assignedIncidentReferences": assignments, "assignmentRequired": m.Role == string(safetyincident.RoleHelper)}
		if m.DID != "" {
			body["did"] = m.DID
			body["permissionSet"] = m.PermissionSet
		}
		if m.OperatorID != uuid.Nil {
			body["operatorId"] = m.OperatorID
		}
		if m.TokenID != uuid.Nil {
			body["tokenId"] = m.TokenID
		}
		if !m.TokenExpiresAt.IsZero() {
			body["tokenExpiresAt"] = m.TokenExpiresAt
		}
		if m.RetoolUserID != nil {
			body["retoolUserId"] = *m.RetoolUserID
		}
		writeModerationJSON(w, body)
	})
}
