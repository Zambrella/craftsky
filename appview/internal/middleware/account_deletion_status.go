package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api/envelope"
)

type deletionStatusGrantKey struct{}

func DeletionStatusGrantFromContext(ctx context.Context) (accountdeletion.StatusGrant, bool) {
	grant, ok := ctx.Value(deletionStatusGrantKey{}).(accountdeletion.StatusGrant)
	return grant, ok
}

func AccountDeletionStatus(
	service accountdeletion.StatusRouteService,
	action accountdeletion.StatusAction,
	logger *slog.Logger,
) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			const prefix = "DeletionStatus "
			if service == nil || !strings.HasPrefix(authorization, prefix) || len(authorization) == len(prefix) {
				envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", GetRunID(r.Context()), nil)
				return
			}
			jobID, err := uuid.Parse(r.PathValue("jobId"))
			deviceID, hasDeviceID := GetDeviceID(r.Context())
			if err != nil || !hasDeviceID {
				envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", GetRunID(r.Context()), nil)
				return
			}
			grant, err := service.AuthorizeStatusRoute(r.Context(), strings.TrimPrefix(authorization, prefix), jobID, deviceID, action)
			if err != nil {
				logger.Warn("account deletion status authorization failed",
					slog.String("run_id", GetRunID(r.Context())),
					slog.String("error_category", "authorization"))
				envelope.WriteError(w, http.StatusUnauthorized, "invalid_deletion_status", "invalid account deletion status credential", GetRunID(r.Context()), nil)
				return
			}
			ctx := context.WithValue(r.Context(), deletionStatusGrantKey{}, grant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
