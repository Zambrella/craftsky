package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// UT-004 / IT-007 / FR-007: only an explicit presented message is marked;
// native capped direct aggregates are kept independent of list totals.
func TestReadNamesPresentedMessageAndCountsUseNativeBuckets(t *testing.T) {
	reads := 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"request","unreadCount":72,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
		case "/xrpc/chat.bsky.convo.getUnreadCounts":
			if r.URL.Query().Get("includeGroupChats") != "false" {
				t.Fatal("counts include groups")
			}
			body = `{"unreadAcceptedConvos":100,"unreadRequestConvos":1}`
		case "/xrpc/chat.bsky.convo.updateRead":
			reads++
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request["messageId"] != "presented-M" || request["convoId"] != "direct-1" {
				t.Fatal("read advanced past presented watermark")
			}
			body = `{"convo":{"id":"direct-1","status":"request","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
		default:
			t.Fatalf("read implicitly invoked another action: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	counts, err := service.UnreadCounts(t.Context(), info)
	if err != nil || counts.Accepted != 100 || counts.Requests != 1 || reads != 0 {
		t.Fatalf("counts=%+v reads=%d err=%v", counts, reads, err)
	}
	result, err := service.UpdateRead(t.Context(), info, "direct-1", "presented-M")
	if err != nil || result.Status != "request" || result.UnreadCount != 1 || reads != 1 {
		t.Fatalf("result=%+v reads=%d err=%v", result, reads, err)
	}
	if _, err := service.UpdateRead(t.Context(), info, "direct-1", ""); err == nil || reads != 1 {
		t.Fatal("missing watermark reached provider")
	}
}
