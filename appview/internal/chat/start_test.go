package chat

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

// UT-003 / IT-003 / FR-004: reading availability never creates a chat;
// explicit starts select the same canonical direct conversation.
func TestAvailabilityDoesNotCreateAndExplicitStartIsCanonical(t *testing.T) {
	starts := 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvoAvailability":
			body = `{"canChat":true}`
		case "/xrpc/chat.bsky.convo.getConvoForMembers":
			starts++
			body = `{"convo":{"id":"canonical-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
		default:
			t.Fatalf("unexpected operation %s", r.URL.Path)
		}
		if r.URL.Path != "/xrpc/chat.bsky.actor.getStatus" && r.URL.Query().Get("members") != "did:plc:external-peer" {
			t.Fatal("wrong participant")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	available, err := service.Availability(t.Context(), info, "did:plc:external-peer")
	if err != nil || !available.CanChat || starts != 0 {
		t.Fatalf("availability=%+v starts=%d err=%v", available, starts, err)
	}
	for range 2 {
		conversation, err := service.StartConversation(t.Context(), info, "did:plc:external-peer")
		if err != nil || conversation.ID != "canonical-1" {
			t.Fatalf("canonical=%+v err=%v", conversation, err)
		}
	}
	if starts != 2 {
		t.Fatalf("explicit starts=%d", starts)
	}
	if _, err := service.StartConversation(t.Context(), info, info.DID); err == nil {
		t.Fatal("self conversation allowed")
	}
}
