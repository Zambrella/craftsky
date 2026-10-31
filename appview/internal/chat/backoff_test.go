package chat

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
	"time"
)

// IT-020: session selection also occurs inside durable claim/failure handling.
func TestPollSelectorFailureUsesClaimedBackoff(t *testing.T) {
	for _, limited := range []bool{false, true} {
		service, store, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("failed selector dispatched upstream")
			return nil, nil
		}))
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor,failure_count) VALUES($1,'ready','saved-cursor',2)`, info.DID); err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)
		cause := errors.New("synthetic selection failure")
		var failure error = cause
		want := 4 * time.Second
		if limited {
			failure = &auth.RateLimitError{RetryAfter: 120 * time.Second, Cause: cause}
			want = 120 * time.Second
		}
		worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return "", failure }), SyncOptions{Now: func() time.Time { return now }, Jitter: func() float64 { return 0 }})
		if err != nil {
			t.Fatal(err)
		}
		if err := worker.Poll(t.Context(), info.DID); !errors.Is(err, cause) {
			t.Fatal("selection cause lost")
		}
		var due time.Time
		var cursor string
		var failures int
		var released bool
		if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at,cursor,failure_count,claim_token IS NULL FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&due, &cursor, &failures, &released); err != nil {
			t.Fatal(err)
		}
		if due.Sub(now) != want || cursor != "saved-cursor" || failures != 3 || !released {
			t.Fatalf("selection backoff=%s failures=%d released=%t", due.Sub(now), failures, released)
		}
	}
}

// UT-006 / IT-020: provider Retry-After exceeds the ordinary backoff cap;
// rate limiting neither resets valid progress nor retries the same call.
func TestPollRateLimitKeepsCursorAndHonorsLongRetryAfter(t *testing.T) {
	calls := 0
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status := 200
		body := `{"chatDisabled":false}`
		header := make(http.Header)
		if strings.HasSuffix(r.URL.Path, "getLog") {
			calls++
			status = 429
			header.Set("Retry-After", "120")
			body = `{"error":"RateLimitExceeded","message":"private-rate-canary"}`
			if r.URL.Query().Get("cursor") != "saved-private-cursor" {
				t.Fatal("valid cursor reset")
			}
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor) VALUES($1,'ready','saved-private-cursor')`, info.DID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	err = worker.Poll(t.Context(), info.DID)
	var limited *auth.RateLimitError
	if !errors.As(err, &limited) || limited.RetryAfter != 120*time.Second || calls != 1 {
		t.Fatalf("rate limit calls=%d err=%v", calls, err)
	}
	var due time.Time
	var state, cursor string
	if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at,safe_state,cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&due, &state, &cursor); err != nil {
		t.Fatal(err)
	}
	if due.Sub(now) != 120*time.Second || state != "rateLimited" || cursor != "saved-private-cursor" {
		t.Fatalf("due delta=%s state=%s", due.Sub(now), state)
	}
}
