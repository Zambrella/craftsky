// Package middleware holds HTTP middleware used by cmd/appview's NewServer.
//
// Every middleware is a constructor function that takes its dependencies
// at startup and returns a func(http.Handler) http.Handler — a shape that
// composes cleanly with standard library routing.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a named type so middleware values can't collide with
// other packages' context keys.
type contextKey string

const runIDKey contextKey = "run_id"

// GetRunID extracts the per-request ID injected by the Logging middleware.
// Returns "" if no middleware ran (e.g. from a test that skipped it).
func GetRunID(ctx context.Context) string {
	if id, ok := ctx.Value(runIDKey).(string); ok {
		return id
	}
	return ""
}

// Logging returns middleware that assigns every request a UUID run_id,
// logs an Info "Request received" line with method + path + run_id, and
// puts the run_id in the request context for handlers to log against.
//
// It uses the supplied logger (typically deps.Logger), NOT slog.Default,
// so tests can capture output with a buffered handler.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runID := uuid.New().String()
			ctx := context.WithValue(r.Context(), runIDKey, runID)
			logger.Info("Request received",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("run_id", runID),
			)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
