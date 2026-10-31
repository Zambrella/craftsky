package chat

import (
	"context"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
	"time"
)

// IT-012 / FR-013: leave/reconnect/reappearance never rearm a request.
// Positive accepted state followed by a new request is a new generation.
func TestRequestNotificationLifetimeSurvivesLeaveAndReconnect(t *testing.T) {
	step := 0
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		status := "request"
		kind := "logCreateMessage"
		if step == 3 {
			kind = "logLeaveConvo"
		}
		if step == 5 {
			kind = "logAcceptConvo"
			status = "accepted"
		}
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.listConvos":
			body = `{"convos":[]}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		case "/xrpc/chat.bsky.convo.getLog":
			if step == 0 {
				body = `{"cursor":"0001","logs":[]}`
			} else {
				message := ""
				if kind == "logCreateMessage" {
					message = fmt.Sprintf(`,"message":{"id":"request-message-%d","sender":{"did":"did:plc:peer"}}`, step)
				}
				body = fmt.Sprintf(`{"logs":[{"$type":"chat.bsky.convo.defs#%s","rev":"%04d","convoId":"request-1"%s}]}`, kind, step+1, message)
			}
		case "/xrpc/chat.bsky.convo.getConvo":
			body = fmt.Sprintf(`{"convo":{"id":"request-1","status":%q,"unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}]}}`, status)
		default:
			t.Fatalf("unexpected method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	now := time.Now().UTC()
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	poll := func() {
		t.Helper()
		now = now.Add(31 * time.Second)
		if err := worker.Poll(t.Context(), info.DID); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		poll()
	}
	for step = 1; step <= 6; step++ {
		poll()
		if step == 2 {
			connection, err := store.Disconnect(t.Context(), info.DID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Connect(t.Context(), info.DID, connection.Generation, nil); err != nil {
				t.Fatal(err)
			}
		}
		if step == 3 {
			var left bool
			if err := store.pool.QueryRow(t.Context(), `SELECT left_by_owner FROM chat_conversation_state WHERE owner_did=$1 AND conversation_id='request-1'`, info.DID).Scan(&left); err != nil || !left {
				t.Fatal("leave state forgotten", err)
			}
		}
		var total int
		if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_notifications`).Scan(&total); err != nil {
			t.Fatal(err)
		}
		expected := 1
		if step == 6 {
			expected = 2
		}
		if total != expected {
			t.Fatalf("step=%d request alerts=%d expected=%d", step, total, expected)
		}
	}
}
