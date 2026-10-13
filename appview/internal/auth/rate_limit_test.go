package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestPDSRateLimitPreservesRetryAfterAndCauseWithoutRetrying(t *testing.T) {
	calls := 0
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader(`{"error":"RateLimitExceeded","message":"private-canary"}`))}, nil
	}))
	err := client.PutRecord(context.Background(), "did:plc:owner", "app.bsky.actor.profile", "self", map[string]any{"$type": "app.bsky.actor.profile"})
	var limited *RateLimitError
	var cause *atclient.APIError
	if calls != 1 || !errors.As(err, &limited) || limited.RetrySeconds() != 120 || !errors.As(err, &cause) || cause.StatusCode != 429 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
	if strings.Contains(limited.Error(), "private-canary") {
		t.Fatal("rate-limit guidance exposes upstream values")
	}
}

func TestRetryAfterAcceptsHTTPDateAndDefaultsInvalidHeader(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	for header, want := range map[string]time.Duration{"120": 120 * time.Second, now.Add(time.Minute).Format(http.TimeFormat): time.Minute, "-10": 5 * time.Second, "invalid": 5 * time.Second} {
		if got := RetryAfter(header, now); got != want {
			t.Errorf("%q: %s, want %s", header, got, want)
		}
	}
}
