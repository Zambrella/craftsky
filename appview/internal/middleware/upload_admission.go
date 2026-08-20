package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

var errUploadBodyAdmissionSaturated = errors.New("upload body admission is saturated")

// UploadBodyAdmission bounds request bodies retained by upload handlers. A
// permit is held from before the first body read until the handler returns, so
// the configured capacity covers both the encoded body and downstream decode
// or remote-write work that keeps those bytes live.
type UploadBodyAdmission struct {
	permits chan struct{}
	waiters chan struct{}
	wait    time.Duration
}

func NewUploadBodyAdmission(capacity int, wait time.Duration) (*UploadBodyAdmission, error) {
	if capacity <= 0 {
		return nil, errors.New("upload body admission capacity must be positive")
	}
	if wait <= 0 {
		return nil, errors.New("upload body admission wait must be positive")
	}
	return &UploadBodyAdmission{
		permits: make(chan struct{}, capacity),
		waiters: make(chan struct{}, capacity),
		wait:    wait,
	}, nil
}

func (admission *UploadBodyAdmission) Handler(next http.Handler) http.Handler {
	if admission == nil {
		panic("upload body admission is required")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := admission.acquire(r.Context()); err != nil {
			RejectBodyWithoutDrain(w, r)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			w.Header().Set("Retry-After", "1")
			envelope.WriteError(
				w,
				http.StatusServiceUnavailable,
				"upload_capacity_unavailable",
				"upload capacity is temporarily unavailable",
				GetRunID(r.Context()),
				nil,
			)
			return
		}
		defer admission.release()
		next.ServeHTTP(w, r)
	})
}

func (admission *UploadBodyAdmission) acquire(ctx context.Context) error {
	select {
	case admission.permits <- struct{}{}:
		return nil
	default:
	}

	select {
	case admission.waiters <- struct{}{}:
		defer func() { <-admission.waiters }()
	default:
		return errUploadBodyAdmissionSaturated
	}

	timer := time.NewTimer(admission.wait)
	defer timer.Stop()
	select {
	case admission.permits <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return errUploadBodyAdmissionSaturated
	}
}

func (admission *UploadBodyAdmission) release() {
	<-admission.permits
}
