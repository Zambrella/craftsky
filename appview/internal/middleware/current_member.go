package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

type CurrentMemberChecker interface {
	IsCurrentMember(context.Context, syntax.DID) (bool, error)
}

// CurrentMember enforces the current craftsky_profiles membership boundary
// after authentication. It deliberately maps a departed member to the same
// public profile-not-found contract used by other membership-aware surfaces.
func CurrentMember(checker CurrentMemberChecker, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			did, ok := GetDID(r.Context())
			if !ok || did == "" {
				logger.Error("current membership check missing authenticated DID",
					slog.String("run_id", GetRunID(r.Context())),
					slog.String("error_category", "internal"))
				envelope.WriteError(w, http.StatusInternalServerError,
					"missing_authenticated_did", "authenticated DID missing", GetRunID(r.Context()), nil)
				return
			}
			current, err := checker.IsCurrentMember(r.Context(), did)
			if err != nil {
				logger.Error("current membership check failed",
					slog.String("run_id", GetRunID(r.Context())),
					slog.String("error_category", "database"))
				envelope.WriteError(w, http.StatusServiceUnavailable,
					"membership_unavailable", "membership unavailable", GetRunID(r.Context()), nil)
				return
			}
			if !current {
				envelope.WriteError(w, http.StatusNotFound,
					"profile_not_found", "profile not found", GetRunID(r.Context()), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
