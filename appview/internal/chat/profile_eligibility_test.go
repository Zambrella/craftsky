package chat

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func seedCraftskyProfiles(t *testing.T, store *Store, dids ...string) {
	t.Helper()
	for _, did := range dids {
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'synthetic-profile') ON CONFLICT DO NOTHING`, did); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProfileRemovalDeniesConversationAndOperations(t *testing.T) {
	for _, removed := range []string{"did:plc:chat-service-owner", "did:plc:eligible-crafter"} {
		t.Run(removed, func(t *testing.T) {
			privateCalls := 0
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getConvo":
					body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:eligible-crafter"}]}}`
				default:
					privateCalls++
					body = `{"canChat":true,"messages":[]}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			seedCraftskyProfiles(t, store, info.DID.String(), "did:plc:eligible-crafter")
			if _, err := service.Conversation(t.Context(), info, "direct-1"); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `DELETE FROM craftsky_profiles WHERE did=$1`, removed); err != nil {
				t.Fatal(err)
			}
			checks := map[string]func() error{
				"conversation": func() error { _, err := service.Conversation(t.Context(), info, "direct-1"); return err },
				"history":      func() error { _, err := service.History(t.Context(), info, "direct-1", "", 50); return err },
				"start": func() error {
					_, err := service.StartConversation(t.Context(), info, "did:plc:eligible-crafter")
					return err
				},
				"read":   func() error { _, err := service.UpdateRead(t.Context(), info, "direct-1", "message-1"); return err },
				"accept": func() error { _, err := service.Accept(t.Context(), info, "direct-1"); return err },
				"mute":   func() error { _, err := service.SetMuted(t.Context(), info, "direct-1", true); return err },
				"leave":  func() error { return service.Leave(t.Context(), info, "direct-1") },
				"send": func() error {
					_, err := service.Send(t.Context(), info, uuid.New(), "direct-1", "synthetic text")
					return err
				},
				"report": func() error {
					_, err := service.Report(t.Context(), info, uuid.New(), "direct-1", "message-1", "com.atproto.moderation.defs#reasonOther", nil)
					return err
				},
				"heartbeat": func() error {
					_, err := service.Heartbeat(t.Context(), info, ViewActivity{View: "conversation", ConversationID: "direct-1"})
					return err
				},
			}
			for name, check := range checks {
				t.Run(name, func(t *testing.T) {
					if err := check(); !errors.Is(err, ErrInaccessibleConversation) {
						t.Fatalf("expected unavailable conversation, got %v", err)
					}
				})
			}
			availability, err := service.Availability(t.Context(), info, "did:plc:eligible-crafter")
			if err != nil || availability.CanChat || privateCalls != 0 {
				t.Fatalf("hidden profile reached private RPC: calls=%d err=%v", privateCalls, err)
			}
		})
	}
}

// IT-021 / AC-030: upstream history is shared, but the CraftSky view only
// exposes indexed CraftSky participants and preserves filtered pagination.
func TestInboxFiltersNonCraftskyPeersAcrossPages(t *testing.T) {
	for _, bucket := range []string{"accepted", "request"} {
		t.Run(bucket, func(t *testing.T) {
			calls := 0
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				if !strings.HasSuffix(r.URL.Path, "getStatus") {
					calls++
					field, peer, next := "convos", "did:plc:not-a-craftsky-profile", `,"cursor":"private-page-2"`
					if bucket == "request" {
						field = "requests"
					}
					if r.URL.Query().Get("cursor") == "private-page-2" {
						peer, next = "did:plc:eligible-crafter", ""
					}
					body = fmt.Sprintf(`{"%s":[{"$type":"chat.bsky.convo.defs#convoView","id":"direct-1","status":%q,"unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":%q,"handle":"crafter.test"}]}]%s}`, field, bucket, peer, next)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			seedCraftskyProfiles(t, store, info.DID.String(), "did:plc:eligible-crafter")
			page, err := service.ListConversations(t.Context(), info, bucket, "", 50)
			if err != nil || len(page.Items) != 1 || page.Items[0].Peer.DID != "did:plc:eligible-crafter" || calls != 2 || page.Cursor != "" {
				t.Fatalf("CraftSky filtering failed: items=%d calls=%d err=%v", len(page.Items), calls, err)
			}
		})
	}
}
