package middleware

import (
	"context"
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

func ModeratorDatabaseAuthentication(authenticator ModeratorAuthenticator, logger *slog.Logger, observers ...ModeratorAuthFailureObserver) func(http.Handler) http.Handler {
	return moderatorAuthentication(logger, observers, func(ctx context.Context, provided string) (ctxkeys.Moderator, error) {
		return authenticator.Authenticate(ctx, provided)
	})
}

func ModeratorAuthentication(token, actorID, sourceSystem string, logger *slog.Logger, observers ...ModeratorAuthFailureObserver) func(http.Handler) http.Handler {
	return moderatorAuthentication(logger, observers, func(_ context.Context, provided string) (ctxkeys.Moderator, error) {
		valid := token != "" && actorID != "" && sourceSystem != "" && len(provided) == len(token) && subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
		if !valid {
			return ctxkeys.Moderator{}, ErrModeratorAuthentication
		}
		return ctxkeys.Moderator{ActorID: actorID, SourceSystem: sourceSystem, Role: "safetyAdministrator"}, nil
	})
}

func moderatorAuthentication(
	logger *slog.Logger,
	observers []ModeratorAuthFailureObserver,
	authenticate func(context.Context, string) (ctxkeys.Moderator, error),
) func(http.Handler) http.Handler {
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
			moderator, err := authenticate(r.Context(), provided)
			if err != nil {
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
			ctx := ctxkeys.WithModerator(r.Context(), moderator)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
