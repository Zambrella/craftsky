package chat

import (
	"github.com/google/uuid"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-013 / FR-014: a random reference is routing only. Resolution still
// requires its exact authenticated owner, current generation and membership.
func TestNotificationOpenResolvesOnlyCurrentAccessibleOwnerConversation(t *testing.T) {
	member := true
	reads := 0
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		if strings.HasSuffix(r.URL.Path, "getConvo") {
			reads++
			owner := "did:plc:chat-service-owner"
			if !member {
				owner = "did:plc:another-owner"
			}
			body = `{"convo":{"id":"direct-1","status":"request","members":[{"did":"` + owner + `"},{"did":"did:plc:peer"}]}}`
		}
		if r.Method != "GET" {
			t.Fatal("opening a notification accepted or marked read")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	reference := uuid.New()
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,messaging_generation,open_reference) VALUES($1,gen_random_uuid(),'open-only','request','direct-1',2,$2)`, info.DID, reference); err != nil {
		t.Fatal(err)
	}
	target, err := service.NotificationOpen(t.Context(), info, reference)
	if err != nil || target.Type != "conversation" || target.ConversationID != "direct-1" || reads != 1 {
		t.Fatalf("open=%+v reads=%d err=%v", target, reads, err)
	}
	if _, err := service.NotificationOpen(t.Context(), info, uuid.New()); err == nil || reads != 1 {
		t.Fatal("unknown reference reached upstream")
	}
	member = false
	if _, err := service.NotificationOpen(t.Context(), info, reference); err == nil {
		t.Fatal("reference bypassed current membership")
	}
	member = true
	if _, err := store.Disconnect(t.Context(), info.DID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.NotificationOpen(t.Context(), info, reference); err == nil {
		t.Fatal("disconnected account opened private target")
	}
}
