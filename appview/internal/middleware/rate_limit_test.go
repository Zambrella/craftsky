package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
