package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

type AgeEligibilityReader interface {
	Restricted(context.Context, syntax.DID) (bool, error)
}

func AgeEligibilityEnforcement(reader AgeEligibilityReader, retained bool, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reader == nil || retained {
				next.ServeHTTP(w, r)
				return
			}
			did, ok := GetDID(r.Context())
			if !ok {
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "authenticated account unavailable", GetRunID(r.Context()), nil)
				return
			}
			restricted, err := reader.Restricted(r.Context(), did)
			if err != nil {
				logger.Error("age eligibility read failed", slog.String("operation", "eligibility.enforce"), slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusServiceUnavailable, "eligibility_unavailable", "account eligibility unavailable", GetRunID(r.Context()), nil)
				return
			}
			if restricted {
				envelope.WriteError(w, http.StatusForbidden, "account_age_restricted", "account access is restricted; retained safety and account controls remain available", GetRunID(r.Context()), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
