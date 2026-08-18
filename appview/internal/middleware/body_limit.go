package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/observability"
)

type BodyKind string

const (
	BodyNoBody      BodyKind = "no_body"
	BodyDefaultJSON BodyKind = "default_json"
	BodyUpload      BodyKind = "upload"
	BodyExempt      BodyKind = "exempt"
)

type BodyLimitConfig struct {
	DefaultJSONBytes       int64
	UploadBytes            int64
	DefaultJSONReadTimeout time.Duration
	UploadReadTimeout      time.Duration
}

// BodyPrecheck performs only header-derived admission. It deliberately does
// not read the request body, so it is safe to place before authentication.
func BodyPrecheck(cfg BodyLimitConfig, kind BodyKind, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if admitBodyHeaders(w, r, cfg, kind, logger) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// BodyLimit installs a streaming size bound and route-specific read deadline.
// It never buffers, drains, or rehydrates the body. Handlers remain the sole
// consumer; boundary failures override a handler's generic decode response.
func BodyLimit(cfg BodyLimitConfig, kind BodyKind, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !admitBodyHeaders(w, r, cfg, kind, logger) {
				return
			}
			limit, timeout := bodyBudget(cfg, kind)
			state := &bodyReadState{}
			bodyWriter := &bodyReadResponseWriter{ResponseWriter: w, request: r, state: state}
			if limit > 0 && r.Body != nil && r.Body != http.NoBody {
				r.Body = &observedRequestBody{
					ReadCloser: http.MaxBytesReader(w, r.Body, limit),
					writer:     w,
					state:      state,
				}
			}
			if timeout > 0 {
				controller := http.NewResponseController(w)
				if err := controller.SetReadDeadline(time.Now().Add(timeout)); err != nil && !errors.Is(err, http.ErrNotSupported) && logger != nil {
					logger.Warn("request body deadline unavailable",
						slog.String("route_pattern", observability.RoutePattern(r)),
						slog.String("run_id", GetRunID(r.Context())))
				}
				defer func() { _ = controller.SetReadDeadline(time.Time{}) }()
			}
			next.ServeHTTP(bodyWriter, r)
			bodyWriter.finish()
		})
	}
}

type bodyReadState struct {
	mu  sync.Mutex
	err error
}

func (state *bodyReadState) record(err error) {
	if !isBodyBoundaryError(err) {
		return
	}
	state.mu.Lock()
	if state.err == nil {
		state.err = err
	}
	state.mu.Unlock()
}

func (state *bodyReadState) load() error {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.err
}

type observedRequestBody struct {
	io.ReadCloser
	writer http.ResponseWriter
	state  *bodyReadState
}

func (body *observedRequestBody) Read(buffer []byte) (int, error) {
	read, err := body.ReadCloser.Read(buffer)
	if isBodyBoundaryError(err) {
		body.writer.Header().Set("Connection", "close")
		body.state.record(err)
	}
	return read, err
}

type bodyReadResponseWriter struct {
	http.ResponseWriter
	request *http.Request
	state   *bodyReadState
	mu      sync.Mutex
	handled bool
}

func (writer *bodyReadResponseWriter) Unwrap() http.ResponseWriter { return writer.ResponseWriter }

func (writer *bodyReadResponseWriter) WriteHeader(status int) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.handled {
		return
	}
	if err := writer.state.load(); err != nil {
		writer.writeBoundaryError(err)
		return
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *bodyReadResponseWriter) Write(body []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.handled {
		return len(body), nil
	}
	if err := writer.state.load(); err != nil {
		writer.writeBoundaryError(err)
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

func (writer *bodyReadResponseWriter) finish() {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.handled {
		return
	}
	if err := writer.state.load(); err != nil {
		writer.writeBoundaryError(err)
	}
}

func (writer *bodyReadResponseWriter) writeBoundaryError(err error) {
	writer.handled = true
	_ = WriteBodyReadError(writer.ResponseWriter, writer.request, err)
}

func admitBodyHeaders(w http.ResponseWriter, r *http.Request, cfg BodyLimitConfig, kind BodyKind, logger *slog.Logger) bool {
	limit, _ := bodyBudget(cfg, kind)
	if kind == BodyNoBody && (r.ContentLength > 0 || len(r.TransferEncoding) > 0) {
		RejectBodyWithoutDrain(w, r)
		logBodyRejection(logger, r, "not_allowed", 0)
		envelope.WriteError(w, http.StatusBadRequest, "request_body_not_allowed", "request body is not allowed for this route", GetRunID(r.Context()), nil)
		return false
	}
	if limit > 0 && r.ContentLength > limit {
		RejectBodyWithoutDrain(w, r)
		logBodyRejection(logger, r, "too_large", limit)
		envelope.WriteError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body exceeds the configured limit", GetRunID(r.Context()), nil)
		return false
	}
	if kind == BodyDefaultJSON && r.ContentLength != 0 {
		contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
		if contentType != "" {
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err != nil || !strings.EqualFold(mediaType, "application/json") {
				envelope.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", GetRunID(r.Context()), nil)
				return false
			}
		}
	}
	return true
}

// RejectBodyWithoutDrain detaches an unread request body and disables
// connection reuse before an early admission/authentication response. The
// server closes the connection instead of reading attacker-controlled bytes
// merely to preserve keep-alive.
func RejectBodyWithoutDrain(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.Body == nil || r.Body == http.NoBody || (r.ContentLength == 0 && len(r.TransferEncoding) == 0) {
		return
	}
	r.Close = true
	w.Header().Set("Connection", "close")
	r.Body = http.NoBody
}

func bodyBudget(cfg BodyLimitConfig, kind BodyKind) (int64, time.Duration) {
	switch kind {
	case BodyDefaultJSON:
		return cfg.DefaultJSONBytes, cfg.DefaultJSONReadTimeout
	case BodyUpload:
		return cfg.UploadBytes, cfg.UploadReadTimeout
	default:
		return 0, 0
	}
}

// WriteBodyReadError writes the canonical envelope for streaming body-limit or
// body-deadline failures. It returns true when it owned the response.
func WriteBodyReadError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		RejectBodyWithoutDrain(w, r)
		envelope.WriteError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body exceeds the configured limit", GetRunID(r.Context()), nil)
		return true
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		RejectBodyWithoutDrain(w, r)
		envelope.WriteError(w, http.StatusRequestTimeout, "request_body_timeout", "request body was not received in time", GetRunID(r.Context()), nil)
		return true
	}
	return false
}

func isBodyBoundaryError(err error) bool {
	if err == nil {
		return false
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}

func logBodyRejection(logger *slog.Logger, r *http.Request, reason string, limit int64) {
	if logger == nil {
		return
	}
	logger.Warn("request body rejected",
		slog.String("reason", reason),
		slog.String("method", r.Method),
		slog.String("route_pattern", observability.RoutePattern(r)),
		slog.Int64("limit_bytes", limit),
		slog.String("run_id", GetRunID(r.Context())))
}
