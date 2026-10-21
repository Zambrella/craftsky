package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strconv"
	"strings"
	"testing"
)

// IT-001 / FR-002, FR-008: filtered-empty pages retain a private opaque
// continuation; peers with indexed CraftSky profiles need no local registration.
func TestInboxContinuesThroughFilteredGroupsAndKeepsCursorPrivate(t *testing.T) {
	for _, bucket := range []string{"accepted", "request"} {
		t.Run(bucket, func(t *testing.T) {
			listCalls := 0
			service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				if r.URL.Path != "/xrpc/chat.bsky.actor.getStatus" {
					listCalls++
					expected := "/xrpc/chat.bsky.convo.listConvos"
					field := "convos"
					if bucket == "request" {
						expected = "/xrpc/chat.bsky.convo.listConvoRequests"
						field = "requests"
						if r.URL.Query().Get("kind") != "" || r.URL.Query().Get("status") != "" {
							t.Fatal("requests used accepted-list controls")
						}
					} else if r.URL.Query().Get("kind") != "direct" || r.URL.Query().Get("status") != "accepted" {
						t.Fatal("accepted listing was not direct/accepted")
					}
					if r.URL.Path != expected {
						t.Fatal("wrong listing operation")
					}
					page := 0
					if cursor := r.URL.Query().Get("cursor"); cursor != "" {
						var err error
						page, err = strconv.Atoi(strings.TrimPrefix(cursor, "private-upstream-cursor-"))
						if err != nil {
							t.Fatal("consumer handle forwarded upstream")
						}
					}
					if page < 5 {
						body = fmt.Sprintf(`{"%s":[{"$type":"chat.bsky.convo.defs#convoView","id":"group-1","kind":{"$type":"chat.bsky.convo.defs#groupConvo"},"status":%q,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}],"cursor":"private-upstream-cursor-%d"}`, field, bucket, page+1)
					} else {
						body = fmt.Sprintf(`{"%s":[{"$type":"chat.bsky.convo.defs#convoView","id":"direct-1","status":%q,"muted":false,"unreadCount":2,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer","handle":"external.example","displayName":"External crafter"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"message-1","rev":"rev-1","sender":{"did":"did:plc:external-peer"},"text":"latest text","sentAt":"2026-10-19T12:00:00Z"}}]}`, field, bucket)
					}
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			first, err := service.ListConversations(t.Context(), info, bucket, "", 50)
			if err != nil || len(first.Items) != 0 || first.Cursor == "" || listCalls != 5 {
				t.Fatalf("filtered page=%+v calls=%d err=%v", first, listCalls, err)
			}
			raw, _ := json.Marshal(first)
			if strings.Contains(string(raw), "private-upstream-cursor") {
				t.Fatal("upstream cursor exposed")
			}
			second, err := service.ListConversations(t.Context(), info, bucket, first.Cursor, 50)
			if err != nil || len(second.Items) != 1 || second.Cursor != "" || listCalls != 6 {
				t.Fatalf("continued page=%+v calls=%d err=%v", second, listCalls, err)
			}
			if second.Items[0].Peer.DID != "did:plc:external-peer" || second.Items[0].UnreadCount != 2 || second.Items[0].LastMessage == nil || second.Items[0].LastMessage.Text != "latest text" {
				t.Fatal("external direct state lost")
			}
		})
	}
}
