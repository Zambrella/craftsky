package chat

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
	"time"
)

// IT-020 / NFR-005, FR-016: view activity changes account cadence only;
// it never performs an inline getLog or extends child inactivity expiry.
func TestViewHeartbeatExpiresAndDoesNotTouchSessionActivity(t *testing.T) {
	logs := 0
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getLog":
			logs++
			body = `{"cursor":"0001","logs":[]}`
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}]}}`
		case "/xrpc/chat.bsky.convo.listConvos":
			body = `{"convos":[]}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		default:
			t.Fatalf("unexpected method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	now := time.Now().UTC().Truncate(time.Microsecond)
	service.now = func() time.Time { return now }
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		now = now.Add(31 * time.Second)
		if err := worker.Poll(t.Context(), info.DID); err != nil {
			t.Fatal(err)
		}
	}
	var beforeIdle, beforeSeen time.Time
	if err := store.pool.QueryRow(t.Context(), `SELECT idle_expires_at,last_seen_at FROM craftsky_sessions WHERE token_hash=$1`, info.ChildTokenHash[:]).Scan(&beforeIdle, &beforeSeen); err != nil {
		t.Fatal(err)
	}
	before := logs
	snapshot, err := service.Heartbeat(t.Context(), info, ViewActivity{View: "conversation", ConversationID: "direct-1"})
	if err != nil || snapshot.Revision == 0 || logs != before {
		t.Fatalf("heartbeat=%+v logs=%d err=%v", snapshot, logs, err)
	}
	now = now.Add(4 * time.Second)
	if err := worker.Poll(t.Context(), info.DID); err != nil {
		t.Fatal(err)
	}
	var due time.Time
	if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&due); err != nil || due.Sub(now) != 4*time.Second {
		t.Fatalf("foreground due delta=%s err=%v", due.Sub(now), err)
	}
	now = now.Add(13 * time.Second)
	if err := worker.Poll(t.Context(), info.DID); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&due); err != nil || due.Sub(now) != 30*time.Second {
		t.Fatalf("expired heartbeat due delta=%s err=%v", due.Sub(now), err)
	}
	var afterIdle, afterSeen time.Time
	if err := store.pool.QueryRow(t.Context(), `SELECT idle_expires_at,last_seen_at FROM craftsky_sessions WHERE token_hash=$1`, info.ChildTokenHash[:]).Scan(&afterIdle, &afterSeen); err != nil {
		t.Fatal(err)
	}
	if !afterIdle.Equal(beforeIdle) || !afterSeen.Equal(beforeSeen) {
		t.Fatal("automatic activity extended session")
	}
	if _, err := store.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE token_hash=$1`, info.ChildTokenHash[:]); err != nil {
		t.Fatal(err)
	}
	var retained int
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_view_activity WHERE owner_did=$1`, info.DID).Scan(&retained); err != nil || retained != 0 {
		t.Fatalf("logout retained device view activity=%d err=%v", retained, err)
	}
}
