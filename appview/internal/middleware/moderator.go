package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

const bearerPrefix = "Bearer "

// ModeratorAuthentication is independent from member and development auth.
// All failures deliberately share one response and contain no credential data.
type ModeratorAuthFailureObserver interface{ ObserveModeratorAuthFailure(time.Time) bool }
type ModeratorAuthSuccessObserver interface{ ObserveModeratorAuthSuccess(time.Time) }

func ModeratorAuthentication(token, actorID, sourceSystem string, logger *slog.Logger, observers ...ModeratorAuthFailureObserver) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			provided := ""
			if strings.HasPrefix(authorization, bearerPrefix) {
				provided = strings.TrimPrefix(authorization, bearerPrefix)
			}
			valid := token != "" && actorID != "" && sourceSystem != "" && len(provided) == len(token) && subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
			if !valid {
				if len(observers) > 0 && observers[0] != nil {
					observers[0].ObserveModeratorAuthFailure(time.Now())
				}
				logger.Warn("moderator authentication failed", slog.String("result", "denied"))
				RejectBodyWithoutDrain(w, r)
				envelope.WriteError(w, http.StatusUnauthorized, "moderator_authentication_failed", "moderator authentication failed", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			if len(observers) > 0 && observers[0] != nil {
				if observer, ok := observers[0].(ModeratorAuthSuccessObserver); ok {
					observer.ObserveModeratorAuthSuccess(time.Now())
				}
			}
			ctx := ctxkeys.WithModerator(r.Context(), ctxkeys.Moderator{ActorID: actorID, SourceSystem: sourceSystem})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
