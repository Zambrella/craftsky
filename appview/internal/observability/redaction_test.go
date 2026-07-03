package observability

import (
	"net/http"
	"testing"
)

func TestRedactHeadersRemovesSensitiveTelemetryValues(t *testing.T) {
	headers := http.Header{
		"Authorization":            []string{"Bearer craftsky-session-token"},
		"Cookie":                   []string{"sid=oauth-refresh-token"},
		"DPoP":                     []string{"proof-material"},
		"X-Craftsky-Device-Id":     []string{"device-123"},
		"X-Request-Id":             []string{"safe-request-id"},
		"Content-Type":             []string{"application/json"},
		"X-Forwarded-For":          []string{"203.0.113.10"},
		"X-Craftsky-Session-Token": []string{"alternate-session-token"},
	}

	redacted := RedactHeaders(headers)

	for _, key := range []string{"Authorization", "Cookie", "DPoP", "X-Craftsky-Device-Id", "X-Craftsky-Session-Token"} {
		if got := redacted.Get(key); got != "[REDACTED]" {
			t.Fatalf("%s = %q, want [REDACTED]", key, got)
		}
	}
	if got := redacted.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want preserved", got)
	}
	if got := redacted.Get("X-Request-Id"); got != "safe-request-id" {
		t.Fatalf("X-Request-Id = %q, want preserved", got)
	}
	if headers.Get("Authorization") != "Bearer craftsky-session-token" {
		t.Fatal("RedactHeaders mutated the input headers")
	}
}
