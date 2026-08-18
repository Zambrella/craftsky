package middleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

func TestRequestConcurrencyLimitRejectsWithoutStartingMoreWork(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	concurrency, err := RequestConcurrencyLimit(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := concurrency(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()
	<-started

	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req = req.WithContext(ctxkeys.WithRunID(req.Context(), "request-2"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After is empty")
	}
	var body envelope.Error
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "service_unavailable" || body.RequestID != "request-2" {
		t.Fatalf("envelope = %#v", body)
	}

	close(release)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("admitted request did not release its slot")
	}
}

func TestOuterRateLimitStopsAddressRotationThroughGlobalBucket(t *testing.T) {
	t.Parallel()

	resolver, err := NewClientAddressResolver(nil, 64)
	if err != nil {
		t.Fatal(err)
	}
	limiter, err := NewBoundedLocalRateLimiter(
		RateLimitConfig{Classes: map[RateClass]ClassLimit{
			RateClassOuter: {Window: time.Minute, Global: 2, PerClient: 100},
		}},
		LocalLimiterOptions{Capacity: 16, IdleTTL: time.Hour, Now: func() time.Time { return time.Unix(100, 0) }},
	)
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	outer, err := OuterRateLimit(resolver, limiter, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	handler := outer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}))

	for index, remote := range []string{"192.0.2.1:1", "192.0.2.2:2", "192.0.2.3:3"} {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.RemoteAddr = remote
		req.Header.Set("X-Craftsky-Device-Id", "rotated-device")
		req = req.WithContext(ctxkeys.WithRunID(context.Background(), "outer-test"))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if index < 2 && recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want 204", index, recorder.Code)
		}
		if index == 2 && recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d status = %d, want 429; body=%s", index, recorder.Code, recorder.Body.String())
		}
	}
	if called != 2 {
		t.Fatalf("handler calls = %d, want 2", called)
	}
}
