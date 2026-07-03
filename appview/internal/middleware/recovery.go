package middleware

import (
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/observability"
)

// Recovery catches HTTP handler panics, logs safe request context, and writes
// the standard v1 error envelope when the response has not already started.
func Recovery(logger *slog.Logger, observer *observability.Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseLogger{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				if recovered := recover(); recovered != nil {
					runID := GetRunID(r.Context())
					routePattern := observability.RoutePattern(r)
					logger.Error("HTTP panic recovered",
						slog.String("component", "http"),
						slog.String("route_pattern", routePattern),
						slog.Int("status", http.StatusInternalServerError),
						slog.String("run_id", runID),
					)
					if observer != nil {
						observer.CapturePanic(r.Context(), observability.EventContext{
							"component":         "http",
							"route_pattern":     routePattern,
							"http_method":       r.Method,
							"http_status":       http.StatusInternalServerError,
							"http_status_class": "5xx",
							"run_id":            runID,
						}, recovered)
					}
					if !rw.wroteHeader {
						envelope.WriteError(rw, http.StatusInternalServerError, "internal_error", "internal server error", runID, nil)
					}
				}
			}()
			next.ServeHTTP(rw, r)
		})
	}
}
