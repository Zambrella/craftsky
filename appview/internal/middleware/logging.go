// Package middleware holds HTTP middleware used by cmd/appview's NewServer.
//
// Every middleware is a constructor function that takes its dependencies
// at startup and returns a func(http.Handler) http.Handler — a shape that
// composes cleanly with standard library routing.
package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
)

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
// logs an Info "Request received" line with method + path + run_id, and
// puts the run_id in the request context for handlers to log against.
//
// It uses the supplied logger (typically deps.Logger), NOT slog.Default,
// so tests can capture output with a buffered handler.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			runID := uuid.New().String()
			ctx := ctxkeys.WithRunID(r.Context(), runID)
			requestPayload, payloadErr := readJSONRequestBody(r)
			logger.Info("Request received",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("run_id", runID),
			)
			requestAttrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("raw_query", r.URL.RawQuery),
				slog.Any("headers", r.Header),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int64("content_length", r.ContentLength),
				slog.String("run_id", runID),
			}
			if requestPayload != "" {
				requestAttrs = append(requestAttrs, slog.String("json_payload", requestPayload))
			}
			if payloadErr != nil {
				requestAttrs = append(requestAttrs, slog.String("json_payload_err", payloadErr.Error()))
			}
			logger.Debug("Request details", requestAttrs...)

			rw := &responseLogger{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r.WithContext(ctx))
			responseAttrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int("bytes", rw.bytes),
				slog.Duration("duration", time.Since(started)),
				slog.String("run_id", runID),
			}
			if payload := rw.JSONPayload(); payload != "" {
				responseAttrs = append(responseAttrs, slog.String("json_payload", payload))
			}
			logger.Debug("Request completed", responseAttrs...)
		})
	}
}

func readJSONRequestBody(r *http.Request) (string, error) {
	if r.Body == nil || r.Body == http.NoBody || !isJSONContentType(r.Header.Get("Content-Type")) {
		return "", nil
	}
	raw, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", err
	}
	return string(raw), err
}

type responseLogger struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
	body        []byte
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
	w.body = append(w.body, b[:n]...)
	return n, err
}

func (w *responseLogger) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseLogger) JSONPayload() string {
	if len(bytes.TrimSpace(w.body)) == 0 {
		return ""
	}
	if isJSONContentType(w.Header().Get("Content-Type")) || looksLikeJSONPayload(w.body) {
		return string(w.body)
	}
	return ""
}

func isJSONContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "json")
}

func looksLikeJSONPayload(payload []byte) bool {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
