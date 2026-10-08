package api

import (
	"net/http"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/safetyincident"
)

func RequireSafetyPermission(permission safetyincident.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			moderator, ok := ctxkeys.GetModerator(r.Context())
			if !ok {
				adminError(w, r, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed")
				return
			}
			incidentID, err := uuid.Parse(r.PathValue("incidentReference"))
			if err != nil {
				adminError(w, r, http.StatusBadRequest, "invalid_request", "invalid incident reference")
				return
			}
			assignments := make(map[uuid.UUID]bool, len(moderator.AssignedIncidentReferences))
			for reference, assigned := range moderator.AssignedIncidentReferences {
				if parsed, parseErr := uuid.Parse(reference); parseErr == nil {
					assignments[parsed] = assigned
				}
			}
			var permissions map[safetyincident.Permission]bool
			if moderator.Permissions != nil {
				permissions = make(map[safetyincident.Permission]bool, len(moderator.Permissions))
				for permission, allowed := range moderator.Permissions {
					permissions[safetyincident.Permission(permission)] = allowed
				}
			}
			actor := safetyincident.Actor{
				ID: moderator.ActorID, Role: safetyincident.Role(moderator.Role),
				Permissions: permissions, AssignedIncidents: assignments,
			}
			if !actor.Allowed(permission, incidentID) {
				adminError(w, r, http.StatusForbidden, "safety_action_forbidden", "safety action forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
