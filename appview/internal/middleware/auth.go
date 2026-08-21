package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
)

// DevAuthMode controls whether and how X-Dev-DID may select an identity.
type DevAuthMode uint8

const (
	DevAuthDisabled DevAuthMode = iota
	DevAuthLocal
	DevAuthRemote
)

// DevAuthPolicy is explicit at every authenticated route stack. Remote mode
// requires Secret to match X-Craftsky-Dev-Authorization.
type DevAuthPolicy struct {
	Mode   DevAuthMode
	Secret string
}

// GetDID extracts the authenticated DID injected by the Authenticated
// middleware. Returns ("", false) if no middleware ran or if the request
// reached the handler without authentication (which shouldn't happen on
// routes wired via Authenticated, but GetDID stays safe either way).
func GetDID(ctx context.Context) (syntax.DID, bool) {
	return ctxkeys.GetDID(ctx)
}

// GetOAuthSessionID extracts the OAuth session ID injected by the
// Authenticated middleware. Returns ("", false) if not present.
func GetOAuthSessionID(ctx context.Context) (string, bool) {
	return ctxkeys.GetOAuthSessionID(ctx)
}

// WithDID stores did in ctx under the same key the Authenticated middleware uses.
// Exported for tests that want to skip middleware setup.
func WithDID(ctx context.Context, did syntax.DID) context.Context {
	return ctxkeys.WithDID(ctx, did)
}

// WithOAuthSessionID stores sid in ctx under the same key the Authenticated middleware uses.
// Exported for tests.
func WithOAuthSessionID(ctx context.Context, sid string) context.Context {
	return ctxkeys.WithOAuthSessionID(ctx, sid)
}

// Authenticated returns middleware that validates a bearer token via
// authService and injects the authenticated DID into the request context.
//
// Follows the same constructor-returning-wrapper shape as Logging and
// CORS so routing code can compose them uniformly:
//
//	mux.Handle("/whoami", middleware.Authenticated(deps.AuthService, deps.Logger, policy)(handler))
//
// Flow:
//  1. Extract the bearer token from the Authorization header. Missing or
//     malformed → 401.
//  2. Apply the explicit development-authorization policy before honoring
//     X-Dev-DID. Production callers must select DevAuthDisabled.
//  3. Call authService.Authenticate(ctx, token). Error → 401.
//  4. Inject the returned DID into the context under didKey and call next.
func Authenticated(authService auth.AuthService, logger *slog.Logger, policy DevAuthPolicy) func(http.Handler) http.Handler {
	return authenticatedWith(authService.Authenticate, logger, policy)
}

// AuthenticatedRecovery is reserved for the explicit recovery route class.
// It shares the same bearer parsing, development policy, and 401/503 taxonomy
// as Authenticated, but delegates lifecycle eligibility to the narrower
// recovery authentication contract.
func AuthenticatedRecovery(authService auth.RecoveryAuthService, logger *slog.Logger, policy DevAuthPolicy) func(http.Handler) http.Handler {
	return authenticatedWith(authService.AuthenticateRecovery, logger, policy)
}

func authenticatedWith(
	authenticate func(context.Context, string) (auth.AuthInfo, error),
	logger *slog.Logger,
	policy DevAuthPolicy,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const bearerPrefix = "Bearer "
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				RejectBodyWithoutDrain(w, r)
				logger.Warn("auth: missing or malformed Authorization header",
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required", GetRunID(r.Context()), nil)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if token == "" {
				RejectBodyWithoutDrain(w, r)
				logger.Warn("auth: empty bearer token",
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required", GetRunID(r.Context()), nil)
				return
			}

			ctx := r.Context()
			if devDID := r.Header.Get("X-Dev-DID"); devDID != "" {
				if policy.Mode == DevAuthDisabled {
					RejectBodyWithoutDrain(w, r)
					envelope.WriteError(w, http.StatusBadRequest, "dev_auth_disabled", "development authorization is disabled", GetRunID(r.Context()), nil)
					return
				}
				if policy.Mode == DevAuthRemote && !devAuthorizationMatches(policy.Secret, r.Header.Get("X-Craftsky-Dev-Authorization")) {
					// Do not surface whether the credential was missing or wrong. A
					// valid real bearer can still authenticate; the dev fallback cannot.
					devDID = ""
				}
				if devDID != "" {
					parsed, err := syntax.ParseDID(devDID)
					if err != nil {
						RejectBodyWithoutDrain(w, r)
						logger.Warn("auth: invalid X-Dev-DID header",
							slog.String("error_category", "validation"),
							slog.String("run_id", GetRunID(r.Context())))
						envelope.WriteError(w, http.StatusBadRequest, "invalid_dev_did", "X-Dev-DID is malformed", GetRunID(r.Context()), nil)
						return
					}
					ctx = auth.WithDevDID(ctx, parsed)
				}
			}

			info, err := authenticate(ctx, token)
			if err != nil {
				RejectBodyWithoutDrain(w, r)
				if errors.Is(err, auth.ErrAuthTokenInvalid) {
					logger.Warn("auth: bearer rejected",
						slog.String("error_category", "invalid_credential"),
						slog.String("run_id", GetRunID(r.Context())))
					envelope.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required", GetRunID(r.Context()), nil)
					return
				}
				logger.Error("auth: authoritative session lookup unavailable",
					slog.String("error_category", "infrastructure"),
					slog.String("run_id", GetRunID(r.Context())))
				w.Header().Set("Retry-After", "5")
				envelope.WriteError(w, http.StatusServiceUnavailable, "authentication_unavailable", "authentication is temporarily unavailable", GetRunID(r.Context()), nil)
				return
			}

			ctx = ctxkeys.WithDID(ctx, info.DID)
			if info.SessionID != "" {
				ctx = ctxkeys.WithOAuthSessionID(ctx, info.SessionID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func devAuthorizationMatches(expected, presented string) bool {
	expectedHash := sha256.Sum256([]byte(expected))
	presentedHash := sha256.Sum256([]byte(presented))
	return expected != "" && subtle.ConstantTimeCompare(expectedHash[:], presentedHash[:]) == 1
}
