package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

// deviceIDPattern enforces the wire-spec format for
// X-Craftsky-Device-Id: 1–128 bytes of [A-Za-z0-9_-]. This comfortably
// accommodates UUIDs / ULIDs while rejecting whitespace, punctuation,
// and oversize payloads in a single check.
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

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
// X-Craftsky-Device-Id header matching ^[A-Za-z0-9_-]{1,128}$ and
// injects its value into the request context.
//
// Missing / empty headers return 400 with error code
// "missing_device_id"; present-but-malformed headers (disallowed
// characters or over 128 bytes) return 400 with "invalid_device_id".
// See:
// docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §3.1
//
// Compose on top of Authenticated on any v1 route that requires auth:
//
//	h := Authenticated(svc, log)(DeviceID(store, log)(handler))
//
// When toucher is non-nil AND the DID + session ID are present in the
// request context (i.e. Authenticated ran and resolved a real session),
// the middleware records the device ID on the session row via
// TouchDeviceID. The call runs synchronously but errors are logged and
// swallowed — the column is best-effort instrumentation and must not
// block the request.
//
// It is safe to pass the same toucher to every route (including
// unauthenticated ones like /v1/auth/login): the context guards
// short-circuit before TouchDeviceID is called when no session is
// resolved. Pass nil only in tests that want to assert the touch path
// is unreachable.
func DeviceID(toucher DeviceIDToucher, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Craftsky-Device-Id")
			if id == "" {
				logger.Warn("device-id: missing header",
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusBadRequest,
					"missing_device_id",
					"X-Craftsky-Device-Id header is required",
					GetRunID(r.Context()),
					nil)
				return
			}
			if !deviceIDPattern.MatchString(id) {
				logger.Warn("device-id: malformed header",
					slog.Int("len", len(id)),
					slog.String("run_id", GetRunID(r.Context())))
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_device_id",
					"X-Craftsky-Device-Id is malformed",
					GetRunID(r.Context()),
					nil)
				return
			}
			ctx := ctxkeys.WithDeviceID(r.Context(), id)
			if toucher != nil {
				if did, ok := ctxkeys.GetDID(ctx); ok {
					if sid, ok := ctxkeys.GetOAuthSessionID(ctx); ok && sid != "" {
						if err := toucher.TouchDeviceID(ctx, did.String(), sid, id); err != nil {
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
