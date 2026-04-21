// Package ctxkeys declares shared context keys and helpers used by both
// the middleware and auth handler packages. Splitting them out here breaks
// the import cycle that would arise if auth imported middleware.
package ctxkeys

import "context"

// contextKey is a private named type so these keys never collide with
// keys from other packages.
type contextKey string

const (
	DIDKey            contextKey = "did"
	OAuthSessionIDKey contextKey = "oauth_session_id"
	DeviceIDKey       contextKey = "device_id"
)

// GetDID extracts the authenticated DID from ctx.
// Returns ("", false) if not present.
func GetDID(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(DIDKey).(string)
	return did, ok
}

// GetOAuthSessionID extracts the OAuth session ID from ctx.
// Returns ("", false) if not present.
func GetOAuthSessionID(ctx context.Context) (string, bool) {
	sid, ok := ctx.Value(OAuthSessionIDKey).(string)
	return sid, ok
}

// WithDID stores did in ctx under DIDKey.
func WithDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, DIDKey, did)
}

// WithOAuthSessionID stores sid in ctx under OAuthSessionIDKey.
func WithOAuthSessionID(ctx context.Context, sid string) context.Context {
	return context.WithValue(ctx, OAuthSessionIDKey, sid)
}

// GetDeviceID extracts the X-Craftsky-Device-Id injected by the
// DeviceID middleware. Returns ("", false) if not present.
func GetDeviceID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(DeviceIDKey).(string)
	return id, ok
}

// WithDeviceID stores id in ctx under DeviceIDKey.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, DeviceIDKey, id)
}
