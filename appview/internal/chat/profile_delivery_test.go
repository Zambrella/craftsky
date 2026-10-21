package chat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/push"
)

// IT-022 / AC-030: queued work cannot outlive CraftSky profile eligibility,
// including removal between the dispatcher's initial and final checks.
func TestDeliveryRechecksCraftskyProfileBeforeSending(t *testing.T) {
	for _, removeAt := range []int{1, 2} {
		t.Run(fmt.Sprintf("remove-at-check-%d", removeAt), func(t *testing.T) {
			var store *Store
			checks, sent := 0, 0
			service, created, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				if strings.HasSuffix(r.URL.Path, "getConvo") {
					checks++
					if checks == removeAt {
						if _, err := store.pool.Exec(t.Context(), `DELETE FROM craftsky_profiles WHERE did='did:plc:peer'`); err != nil {
							t.Fatal(err)
						}
					}
					body = `{"convo":{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"incoming-1","rev":"0002","sender":{"did":"did:plc:peer"},"text":"synthetic message","sentAt":"2026-10-19T12:00:00Z"}}}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			store = created
			seedDeliveryFixture(t, store, info)
			dispatcher, err := NewDeliveryDispatcher(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), chatPushFunc(func(context.Context, push.SendRequest) (push.ProviderResult, error) {
				sent++
				return push.ProviderResult{Class: push.ResultSuccess}, nil
			}), DeliveryOptions{})
			if err != nil {
				t.Fatal(err)
			}
			processed, err := dispatcher.ProcessOne(t.Context())
			var state string
			if scanErr := store.pool.QueryRow(t.Context(), `SELECT status FROM chat_push_deliveries`).Scan(&state); scanErr != nil {
				t.Fatal(scanErr)
			}
			if err != nil || !processed || sent != 0 || state != "cancelled" {
				t.Fatalf("hidden peer delivery: processed=%t sent=%d state=%s err=%v", processed, sent, state, err)
			}
		})
	}
}

func TestSyncSkipsNonCraftskyPeersAndAdvancesCursor(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		view := countView("direct-1", "accepted", "did:plc:non-crafter", 1, ` ,"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"incoming-1","rev":"0002","sender":{"did":"did:plc:non-crafter"},"text":"synthetic message","sentAt":"2026-10-19T12:00:00Z"}`, "")
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getLog":
			if r.URL.Query().Get("cursor") == "" {
				body = `{"cursor":"0001","logs":[]}`
			} else {
				body = `{"cursor":"0002","logs":[{"$type":"chat.bsky.convo.defs#logCreateMessage","rev":"0002","convoId":"direct-1","message":{"id":"incoming-1","sender":{"did":"did:plc:non-crafter"},"text":"synthetic message"}}]}`
			}
		case "/xrpc/chat.bsky.convo.listConvos":
			body = `{"convos":[` + view + `]}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":` + view + `}`
		default:
			t.Fatalf("unexpected sync call %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	now := time.Now().UTC()
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for range 4 {
		now = now.Add(31 * time.Second)
		if err := worker.Poll(t.Context(), info.DID); err != nil {
			t.Fatal(err)
		}
	}
	var cursor string
	var conversations, notifications int
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM chat_conversation_state),(SELECT count(*) FROM chat_notifications)`).Scan(&conversations, &notifications); err != nil {
		t.Fatal(err)
	}
	// The private baseline tombstone preserves suppression; it contains no
	// peer identity/body and is never used to populate the consumer inbox.
	if cursor != "0002" || conversations != 1 || notifications != 0 {
		t.Fatalf("non-CraftSky sync: cursor=%s conversations=%d notifications=%d", cursor, conversations, notifications)
	}
}

// IT-022 / FR-013: becoming eligible must not replay pre-enable history or
// reset the existing lifetime suppression of a baseline request.
func TestBaselineSuppressionSurvivesPeerJoiningCraftsky(t *testing.T) {
	for _, bucket := range []string{"accepted", "request"} {
		t.Run(bucket, func(t *testing.T) {
			joined := false
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				messageID := "pre-enable-message"
				if joined && bucket == "request" {
					messageID = "later-request-message"
				}
				view := countView("direct-1", bucket, "did:plc:joining-crafter", 1, fmt.Sprintf(`,"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":%q,"rev":"0001","sender":{"did":"did:plc:joining-crafter"},"text":"synthetic old message","sentAt":"2026-10-19T12:00:00Z"}`, messageID), "")
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getLog":
					body = `{"cursor":"0001","logs":[]}`
				case "/xrpc/chat.bsky.convo.listConvos":
					body = `{"convos":[]}`
					if bucket == "accepted" {
						body = `{"convos":[` + view + `]}`
					}
				case "/xrpc/chat.bsky.convo.listConvoRequests":
					body = `{"requests":[]}`
					if bucket == "request" {
						body = `{"requests":[` + view + `]}`
					}
				default:
					t.Fatalf("unexpected sync call %s", r.URL.Path)
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
			joined = true
			seedCraftskyProfiles(t, store, "did:plc:joining-crafter")
			phase := "recoveryAccepted"
			if bucket == "request" {
				phase = "recoveryRequests"
			}
			if _, err := store.pool.Exec(t.Context(), `UPDATE chat_sync_state SET phase=$2,page_cursor='',recovery_id=gen_random_uuid(),next_due_at=now()-interval '1 second' WHERE owner_did=$1`, info.DID, phase); err != nil {
				t.Fatal(err)
			}
			poll()
			var notifications int
			if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_notifications`).Scan(&notifications); err != nil {
				t.Fatal(err)
			}
			if notifications != 0 {
				t.Fatal("profile creation replayed pre-enable history")
			}
		})
	}
}
