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
//	h := Authenticated(svc, log)(DeviceID(log)(handler))
//
// The middleware does NOT verify that the device ID matches any
// persisted value; recording it on craftsky_sessions is the handler
// chain's responsibility.
func DeviceID(logger *slog.Logger) func(http.Handler) http.Handler {
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
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
