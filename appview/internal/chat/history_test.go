package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-002 / FR-003: requests remain readable without accepting or marking
// them read; tombstones/unknown variants preserve stable chronology and cursor.
func TestHistoryKeepsSafePlaceholdersAndPrivatePagination(t *testing.T) {
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatal("history read caused an upstream action")
		}
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"request","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}]}}`
		case "/xrpc/chat.bsky.convo.getMessages":
			if r.URL.Query().Get("convoId") != "direct-1" {
				t.Fatal("wrong conversation selected")
			}
			if r.URL.Query().Get("cursor") == "" {
				body = `{"messages":[{"$type":"chat.bsky.convo.defs#messageView","id":"message-3","rev":"3","sender":{"did":"did:plc:peer"},"text":"same text","sentAt":"2026-10-19T12:03:00Z"},{"$type":"chat.bsky.convo.defs#deletedMessageView","id":"message-2","rev":"4","sender":{"did":"did:plc:peer"},"sentAt":"2026-10-19T12:02:00Z"},{"$type":"future.message","id":"message-1","rev":"1","sentAt":"2026-10-19T12:01:00Z","privateData":"unknown-private-canary"}],"cursor":"private-history-cursor"}`
			} else {
				if r.URL.Query().Get("cursor") != "private-history-cursor" {
					t.Fatal("consumer cursor forwarded")
				}
				body = `{"messages":[],"cursor":"private-history-cursor-2"}`
			}
		default:
			t.Fatal("unexpected upstream method")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	page, err := service.History(t.Context(), info, "direct-1", "", 50)
	if err != nil || len(page.Items) != 3 || page.Cursor == "" {
		t.Fatalf("history=%+v err=%v", page, err)
	}
	if page.Items[0].ID != "message-1" || page.Items[0].Kind != "unsupported" || page.Items[1].Kind != "deleted" || page.Items[1].Text != "" || page.Items[2].Text != "same text" {
		t.Fatal("history placeholder/order mapping failed")
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "unknown-private-canary") || strings.Contains(string(raw), "private-history-cursor") {
		t.Fatal("raw unsupported content/cursor exposed")
	}
	empty, err := service.History(t.Context(), info, "direct-1", page.Cursor, 50)
	if err != nil || len(empty.Items) != 0 || empty.Cursor == "" {
		t.Fatal("empty page ended valid history pagination")
	}
}
