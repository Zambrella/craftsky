package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type SuspensionReader interface {
	IsSuspended(context.Context, syntax.DID) (bool, error)
}

func ModerationEnforcement(reader SuspensionReader, allowed bool, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowed {
				next.ServeHTTP(w, r)
				return
			}
			if reader == nil {
				RejectBodyWithoutDrain(w, r)
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "moderation enforcement unavailable", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			did, ok := ctxkeys.GetDID(r.Context())
			if !ok {
				RejectBodyWithoutDrain(w, r)
				envelope.WriteError(w, 500, "internal_error", "moderation enforcement unavailable", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			suspended, err := reader.IsSuspended(r.Context(), did)
			if err != nil {
				logger.Error("moderation enforcement failed")
				RejectBodyWithoutDrain(w, r)
				envelope.WriteError(w, 500, "internal_error", "moderation enforcement unavailable", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			if suspended {
				RejectBodyWithoutDrain(w, r)
				envelope.WriteError(w, http.StatusForbidden, "account_suspended", "account is suspended from participation", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
