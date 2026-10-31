package chat

import (
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-007 / FR-008, FR-009: shared actions return authoritative state;
// acceptance without rev still requires a confirming accepted conversation.
func TestConversationActionsConfirmAuthoritativeStateWithoutReading(t *testing.T) {
	accepted, muted, left := false, false, false
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		status := "request"
		if accepted {
			status = "accepted"
		}
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.acceptConvo":
			accepted = true
			body = `{}`
		case "/xrpc/chat.bsky.convo.muteConvo":
			muted = true
		case "/xrpc/chat.bsky.convo.unmuteConvo":
			muted = false
		case "/xrpc/chat.bsky.convo.leaveConvo":
			left = true
			body = `{"convoId":"direct-1","rev":"left-1"}`
		case "/xrpc/chat.bsky.convo.getConvo":
		default:
			t.Fatalf("unexpected action %s", r.URL.Path)
		}
		if strings.HasSuffix(r.URL.Path, "getConvo") || strings.HasSuffix(r.URL.Path, "muteConvo") {
			mute := "false"
			if muted {
				mute = "true"
			}
			body = `{"convo":{"id":"direct-1","status":"` + status + `","muted":` + mute + `,"unreadCount":3,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	result, err := service.Accept(t.Context(), info, "direct-1")
	if err != nil || result.Status != "accepted" {
		t.Fatalf("accept=%+v err=%v", result, err)
	}
	for _, value := range []bool{true, false} {
		result, err = service.SetMuted(t.Context(), info, "direct-1", value)
		if err != nil || result.Muted != value || result.UnreadCount != 3 {
			t.Fatalf("mute=%+v err=%v", result, err)
		}
	}
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,messaging_generation,open_reference) VALUES($1,gen_random_uuid(),'pending-before-leave','request','direct-1',2,gen_random_uuid())`, info.DID); err != nil {
		t.Fatal(err)
	}
	seedDeliveryFixture(t, store, info)
	if err := service.Leave(t.Context(), info, "direct-1"); err != nil || !left {
		t.Fatalf("leave=%v err=%v", left, err)
	}
	var state string
	if err := store.pool.QueryRow(t.Context(), `SELECT state FROM chat_notifications WHERE owner_did=$1 AND logical_key='pending-before-leave'`, info.DID).Scan(&state); err != nil || state != "cancelled" {
		t.Fatalf("pending leave delivery state=%s err=%v", state, err)
	}
	var pending int
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_push_deliveries WHERE status IN('pending','retry','leased')`).Scan(&pending); err != nil || pending != 0 {
		t.Fatalf("leave retained pending device deliveries=%d err=%v", pending, err)
	}
}
