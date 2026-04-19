// Package auth defines the authentication contract the HTTP middleware
// uses and provides the dev (mock) and prod (not-yet-implemented)
// implementations.
//
// The interface is transport-agnostic: it takes a context and a token,
// returns a DID or error. HTTP-specific concerns (bearer header parsing,
// the X-Dev-DID override header) live in internal/middleware.
//
// Context helpers (WithDevDID / DevDIDFromContext) live here, not in
// middleware, so implementations can read from context without importing
// middleware — that would create a cycle.
package auth

import "context"

// AuthInfo carries the authenticated identity and its OAuth session ID.
// Dev/mock implementations return SessionID = "" and consumers must tolerate that.
type AuthInfo struct {
	DID       string
	SessionID string
}

// AuthService validates a bearer token and returns the authenticated identity.
type AuthService interface {
	Authenticate(ctx context.Context, token string) (AuthInfo, error)
}

// contextKey is unexported to prevent collisions across packages.
type contextKey string

const devDIDKey contextKey = "dev_did"

// WithDevDID returns a derived context carrying the given DID under the
// dev-DID key. Middleware calls this when the X-Dev-DID header is present.
func WithDevDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, devDIDKey, did)
}

// DevDIDFromContext extracts a dev-DID previously stored via WithDevDID.
// Returns ("", false) if none is present.
func DevDIDFromContext(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(devDIDKey).(string)
	return did, ok
}
