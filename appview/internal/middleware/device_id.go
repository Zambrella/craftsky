package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

// maxDeviceIDLen bounds header size to prevent accidental / abusive
// client bugs from bloating the log stream. 256 bytes is comfortable
// headroom for UUIDs and ULIDs.
const maxDeviceIDLen = 256

// DeviceIDToucher is satisfied by *auth.CraftskySessionStore. It
// lets the DeviceID middleware record the device ID against the
// current session without importing the auth package (which would
// cycle).
type DeviceIDToucher interface {
	TouchDeviceID(ctx context.Context, did, oauthSessionID, deviceID string) error
}

// GetDeviceID extracts the X-Craftsky-Device-Id injected by DeviceID.
func GetDeviceID(ctx context.Context) (string, bool) {
	return ctxkeys.GetDeviceID(ctx)
}

// WithDeviceID stores id in ctx under the same key as the middleware.
// Exported for tests that want to skip middleware setup.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return ctxkeys.WithDeviceID(ctx, id)
}

// DeviceID returns middleware that requires a non-empty
// X-Craftsky-Device-Id header and injects its value into the request
// context.
//
// Missing, empty, or over-length headers return 400 with the canonical
// error envelope. See:
// docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §3.1
//
// Compose on top of Authenticated on any v1 route that requires auth:
//
//	h := Authenticated(svc, log)(DeviceID(store, log)(handler))
//
// When toucher is non-nil AND the DID + session ID are present in the
// request context (i.e. Authenticated ran and resolved a real session),
// the middleware records the device ID on the session row via
// TouchDeviceID. The call is fire-and-forget: it runs synchronously
// but errors are logged and swallowed — the column is best-effort
// instrumentation and must not block the request. Pass nil for
// toucher on unauthenticated routes (e.g. /v1/auth/login).
func DeviceID(toucher DeviceIDToucher, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Craftsky-Device-Id")
			if id == "" || len(id) > maxDeviceIDLen {
				logger.Warn("device-id: missing or invalid header",
					slog.Int("len", len(id)),
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusBadRequest,
					"missing_device_id",
					"X-Craftsky-Device-Id header is required",
					GetRunID(r.Context()),
					nil)
				return
			}
			ctx := ctxkeys.WithDeviceID(r.Context(), id)
			if toucher != nil {
				if did, ok := ctxkeys.GetDID(ctx); ok {
					if sid, ok := ctxkeys.GetOAuthSessionID(ctx); ok && sid != "" {
						if err := toucher.TouchDeviceID(ctx, did, sid, id); err != nil {
							logger.Warn("device-id: TouchDeviceID failed",
								slog.String("err", err.Error()),
								slog.String("run_id", GetRunID(ctx)))
						}
					}
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
