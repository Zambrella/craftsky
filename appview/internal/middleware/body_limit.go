package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
)

type BodyKind string

const (
	BodyNoBody      BodyKind = "no_body"
	BodyDefaultJSON BodyKind = "default_json"
	BodyUpload      BodyKind = "upload"
	BodyExempt      BodyKind = "exempt"
)

type BodyLimitConfig struct {
	DefaultJSONBytes int64
	UploadBytes      int64
}

func BodyLimit(cfg BodyLimitConfig, kind BodyKind, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch kind {
			case BodyDefaultJSON:
				if !enforceMaxBody(w, r, cfg.DefaultJSONBytes, logger) {
					return
				}
			case BodyUpload:
				if !enforceMaxBody(w, r, cfg.UploadBytes, logger) {
					return
				}
			case BodyNoBody:
				if !enforceNoBody(w, r, logger) {
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func enforceNoBody(w http.ResponseWriter, r *http.Request, logger *slog.Logger) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	_ = r.Body.Close()
	if err != nil {
		envelope.WriteError(w, http.StatusBadRequest, "invalid_request_body", "request body could not be read", GetRunID(r.Context()), nil)
		return false
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if logger != nil {
			logger.Warn("request body rejected: not allowed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("run_id", GetRunID(r.Context())))
		}
		envelope.WriteError(w, http.StatusBadRequest, "request_body_not_allowed", "request body is not allowed for this route", GetRunID(r.Context()), nil)
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	r.ContentLength = int64(len(raw))
	return true
}

func enforceMaxBody(w http.ResponseWriter, r *http.Request, limit int64, logger *slog.Logger) bool {
	if limit <= 0 || r.Body == nil || r.Body == http.NoBody {
		return true
	}
	limited := io.LimitReader(r.Body, limit+1)
	raw, err := io.ReadAll(limited)
	_ = r.Body.Close()
	if err != nil {
		envelope.WriteError(w, http.StatusBadRequest, "invalid_request_body", "request body could not be read", GetRunID(r.Context()), nil)
		return false
	}
	if int64(len(raw)) > limit {
		if logger != nil {
			logger.Warn("request body rejected: too large",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("limit_bytes", limit),
				slog.String("run_id", GetRunID(r.Context())))
		}
		envelope.WriteError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body exceeds the configured limit", GetRunID(r.Context()), nil)
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	r.ContentLength = int64(len(raw))
	return true
}
