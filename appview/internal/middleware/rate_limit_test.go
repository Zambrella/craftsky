package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// IT-006: the HTTP middleware enforces independent link-preview token/device
// budgets and emits the standard rejection contract before handler work.
func TestLinkPreviewRateLimitMiddlewareBudgets(t *testing.T) {
	now := time.Unix(1_000, 0)
	config := RateLimitConfig{Classes: map[RateClass]ClassLimit{
		RateClassLinkPreview: {Window: time.Hour, PerToken: 60, PerDevice: 120},
	}}
	serve := func(limiter *LocalRateLimiter, token, device string, called *int) *httptest.ResponseRecorder {
		handler := RateLimit(limiter, RateClassLinkPreview, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			*called++
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", nil)
		request.Header.Set("X-Craftsky-Device-Id", device)
		request = request.WithContext(WithOAuthSessionID(request.Context(), token))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}

	tokenLimiter := NewLocalRateLimiter(config, func() time.Time { return now })
	tokenCalls := 0
	for request := 1; request <= 61; request++ {
		recorder := serve(tokenLimiter, "shared-token", fmt.Sprintf("device-%d", request), &tokenCalls)
		if request <= 60 && recorder.Code != http.StatusNoContent {
			t.Fatalf("token request %d status = %d, want 204", request, recorder.Code)
		}
		if request == 61 {
			assertRateLimitedResponse(t, recorder)
		}
	}
	if tokenCalls != 60 {
		t.Fatalf("token handler calls = %d, want 60", tokenCalls)
	}

	deviceLimiter := NewLocalRateLimiter(config, func() time.Time { return now })
	deviceCalls := 0
	for request := 1; request <= 121; request++ {
		recorder := serve(deviceLimiter, fmt.Sprintf("token-%d", request), "shared-device", &deviceCalls)
		if request == 121 {
			assertRateLimitedResponse(t, recorder)
		}
	}
	if deviceCalls != 120 {
		t.Fatalf("device handler calls = %d, want 120", deviceCalls)
	}

	now = now.Add(time.Hour)
	if recorder := serve(tokenLimiter, "shared-token", "device-after-reset", &tokenCalls); recorder.Code != http.StatusNoContent {
		t.Fatalf("post-reset status = %d, want 204", recorder.Code)
	}
}

func assertRateLimitedResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "" || !strings.Contains(recorder.Body.String(), `"error":"rate_limited"`) {
		t.Fatalf("rate response status/headers/body = %d %v %s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
}

func TestRateLimiterRejectsExceededDeviceBucket(t *testing.T) {
	limiter := NewLocalRateLimiter(RateLimitConfig{Classes: map[RateClass]ClassLimit{
		RateClassRead: {Window: time.Minute, PerToken: 10, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })

	if decision := limiter.Allow(RateClassRead, RateKeys{TokenKey: "session-a", DeviceID: "device-a"}); !decision.Allowed {
		t.Fatalf("first decision allowed = false: %+v", decision)
	}
	decision := limiter.Allow(RateClassRead, RateKeys{TokenKey: "session-b", DeviceID: "device-a"})
	if decision.Allowed {
		t.Fatalf("second decision allowed = true, want device limit rejection")
	}
	if decision.KeyType != "device" {
		t.Fatalf("KeyType = %q, want device", decision.KeyType)
	}
	if decision.RetryAfter <= 0 {
		t.Fatalf("RetryAfter = %v, want positive", decision.RetryAfter)
	}
}

func TestRateLimiterRejectsExceededTokenBucketWithoutIPKeys(t *testing.T) {
	limiter := NewLocalRateLimiter(RateLimitConfig{Classes: map[RateClass]ClassLimit{
		RateClassWrite: {Window: time.Minute, PerToken: 1, PerDevice: 10},
	}}, func() time.Time { return time.Unix(100, 0) })

	if decision := limiter.Allow(RateClassWrite, RateKeys{TokenKey: "session-a", DeviceID: "device-a"}); !decision.Allowed {
		t.Fatalf("first decision allowed = false: %+v", decision)
	}
	decision := limiter.Allow(RateClassWrite, RateKeys{TokenKey: "session-a", DeviceID: "device-b"})
	if decision.Allowed {
		t.Fatalf("second decision allowed = true, want token limit rejection")
	}
	if decision.KeyType != "token" {
		t.Fatalf("KeyType = %q, want token", decision.KeyType)
	}
	for _, key := range limiter.DebugKeys() {
		if strings.Contains(key, "ip") || strings.Contains(key, "127.0.0.1") {
			t.Fatalf("limiter key %q appears to use IP data", key)
		}
	}
}

func TestRateLimitMiddlewareWrites429EnvelopeAndRetryAfter(t *testing.T) {
	limiter := NewLocalRateLimiter(RateLimitConfig{Classes: map[RateClass]ClassLimit{
		RateClassAuth: {Window: time.Minute, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RateLimit(limiter, RateClassAuth, nil)(next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
		req.Header.Set("X-Craftsky-Device-Id", "device-a")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusNoContent {
			t.Fatalf("first status = %d, want 204", rec.Code)
		}
		if i == 1 {
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("second status = %d, want 429; body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After header missing")
			}
			if rec.Header().Get("X-RateLimit-Limit") != "" || rec.Header().Get("X-RateLimit-Remaining") != "" {
				t.Fatal("public X-RateLimit headers must not be exposed")
			}
			if !strings.Contains(rec.Body.String(), "rate_limited") {
				t.Fatalf("body = %q, want rate_limited", rec.Body.String())
			}
		}
	}
	if called != 1 {
		t.Fatalf("handler calls = %d, want 1", called)
	}
}

func TestRateLimitMiddlewareDetachesUnreadBodyOnRejection(t *testing.T) {
	limiter := NewLocalRateLimiter(RateLimitConfig{Classes: map[RateClass]ClassLimit{
		RateClassWrite: {Window: time.Minute, PerDevice: 1},
	}}, func() time.Time { return time.Unix(100, 0) })
	handler := RateLimit(limiter, RateClassWrite, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodPost, "/v1/posts", nil)
	first.Header.Set("X-Craftsky-Device-Id", "device-a")
	handler.ServeHTTP(httptest.NewRecorder(), first)

	probe := &countingBodyReader{reader: strings.NewReader(`{"text":"unread"}`)}
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", probe)
	request.Header.Set("X-Craftsky-Device-Id", "device-a")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", recorder.Code, recorder.Body.String())
	}
	if probe.bytesRead != 0 {
		t.Fatalf("body bytes read = %d, want 0", probe.bytesRead)
	}
	if request.Body != http.NoBody || !request.Close || recorder.Header().Get("Connection") != "close" {
		t.Fatalf("unread body was not detached: body=%T request.Close=%t headers=%v", request.Body, request.Close, recorder.Header())
	}
}
