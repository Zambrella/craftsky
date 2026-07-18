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
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/observability"
)

const statusClientClosedRequest = 499

// GetRunID extracts the per-request ID injected by the Logging middleware.
// Returns "" if no middleware ran (e.g. from a test that skipped it).
//
// Thin wrapper around ctxkeys.GetRunID so the auth handler package (which
// cannot import middleware without inducing an import cycle) can still
// read the run_id via ctxkeys directly while middleware callers keep the
// familiar middleware.GetRunID spelling.
func GetRunID(ctx context.Context) string {
	return ctxkeys.GetRunID(ctx)
}

// Logging returns middleware that assigns every request a UUID run_id,
// logs request start/completion with method, run_id, status, duration, and
// route pattern where available. It puts the run_id in the request context
// for handlers to log against.
//
// It uses the supplied logger (typically deps.Logger), NOT slog.Default,
// so tests can capture output with a buffered handler.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			runID := uuid.New().String()
			ctx := ctxkeys.WithRunID(r.Context(), runID)
			ctx = observability.WithRoutePatternRecorder(ctx)
			logger.Info("Request received",
				slog.String("method", r.Method),
				slog.String("run_id", runID),
			)
			requestAttrs := []any{
				slog.String("method", r.Method),
				slog.Int64("content_length", r.ContentLength),
				slog.String("run_id", runID),
			}
			logger.Debug("Request details", requestAttrs...)

			req := r.WithContext(ctx)
			rw := &responseLogger{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, req)
			routePattern := observability.RecordedRoutePattern(req.Context(), observability.RoutePattern(req))
			status := rw.status
			if req.Context().Err() == context.Canceled {
				status = statusClientClosedRequest
			}
			responseAttrs := []any{
				slog.String("method", r.Method),
				slog.String("route_pattern", routePattern),
				slog.Int("status", status),
				slog.Int("bytes", rw.bytes),
				slog.Duration("duration", time.Since(started)),
				slog.String("run_id", runID),
			}
			logger.Debug("Request completed", responseAttrs...)
		})
	}
}

type responseLogger struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *responseLogger) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseLogger) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *responseLogger) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
