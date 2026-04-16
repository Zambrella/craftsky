package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"social.craftsky/appview/internal/auth"
)

const didKey contextKey = "did"

// GetDID extracts the authenticated DID injected by the Authenticated
// middleware. Returns ("", false) if no middleware ran or if the request
// reached the handler without authentication (which shouldn't happen on
// routes wired via Authenticated, but GetDID stays safe either way).
func GetDID(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(didKey).(string)
	return did, ok
}

// Authenticated returns middleware that validates a bearer token via
// authService and injects the authenticated DID into the request context.
//
// Follows the same constructor-returning-wrapper shape as Logging and
// CORS so routing code can compose them uniformly:
//
//	mux.Handle("/whoami", middleware.Authenticated(deps.AuthService, deps.Logger)(handler))
//
// Flow:
//  1. Extract the bearer token from the Authorization header. Missing or
//     malformed → 401.
//  2. If the request carries X-Dev-DID, inject it into the context via
//     auth.WithDevDID. MockAuthService reads this; other impls ignore it.
//  3. Call authService.Authenticate(ctx, token). Error → 401.
//  4. Inject the returned DID into the context under didKey and call next.
//
// The X-Dev-DID sniff is unconditional: in prod, NotImplementedAuthService
// errors regardless, so 401 is the outcome. This keeps the middleware
// free of Env-awareness.
func Authenticated(authService auth.AuthService, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const bearerPrefix = "Bearer "
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				logger.Warn("auth: missing or malformed Authorization header",
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if token == "" {
				logger.Warn("auth: empty bearer token",
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			if devDID := r.Header.Get("X-Dev-DID"); devDID != "" {
				ctx = auth.WithDevDID(ctx, devDID)
			}

			did, err := authService.Authenticate(ctx, token)
			if err != nil {
				logger.Warn("auth: Authenticate returned error",
					slog.String("err", err.Error()),
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(ctx, didKey, did)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
