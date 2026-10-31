package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

// DEL-001 / FR-020: scope, membership and sender are server-established.
func TestDeleteMessageForSelfRequiresOwnedCurrentMessage(t *testing.T) {
	for _, scenario := range []string{"own", "request", "received", "missing", "deleted", "unsupported", "foreign-conversation", "foreign-member", "group", "missing-profile", "missing-scope", "revoked-child", "disabled", "invalid-id", "bad-response", "transport-failure"} {
		t.Run(scenario, func(t *testing.T) {
			scopes := []string{"atproto"}
			for _, scope := range auth.OptionalChatScopes() {
				if scenario != "missing-scope" || !strings.HasPrefix(scope, "rpc:chat.bsky.convo.deleteMessageForSelf?") {
					scopes = append(scopes, scope)
				}
			}
			deletes := 0
			service, store, info := newServiceFixture(t, scenario != "disabled", scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getConvo":
					owner, peer, status, extra := "did:plc:chat-service-owner", "did:plc:sender", "accepted", ""
					if scenario == "foreign-member" {
						owner = "did:plc:other"
					}
					if scenario == "missing-profile" {
						peer = "did:plc:unregistered"
					}
					if scenario == "request" {
						status = "request"
					}
					if scenario == "group" {
						extra = `,"kind":{"$type":"chat.bsky.convo.defs#groupConvo"}`
					}
					body = `{"convo":{"id":"direct-1","status":"` + status + `","members":[{"did":"` + owner + `"},{"did":"` + peer + `"}]` + extra + `}}`
				case "/xrpc/chat.bsky.convo.getMessages":
					sender, kind := "did:plc:chat-service-owner", "messageView"
					if scenario == "received" {
						sender = "did:plc:sender"
					}
					if scenario == "deleted" {
						kind = "deletedMessageView"
					}
					if scenario == "unsupported" {
						kind = "futureMessage"
					}
					body = `{"messages":[{"$type":"chat.bsky.convo.defs#` + kind + `","id":"own-1","rev":"0001","sender":{"did":"` + sender + `"},"text":"private-delete-canary","sentAt":"2026-10-31T12:00:00Z"}]}`
					if scenario == "missing" {
						body = `{"messages":[]}`
					}
				case "/xrpc/chat.bsky.convo.deleteMessageForSelf":
					deletes++
					if r.Header.Get("Atproto-Proxy") != auth.ChatServiceAudience {
						t.Fatal("wrong deletion audience")
					}
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Fatal(err)
					}
					if len(input) != 2 || input["convoId"] != "direct-1" || input["messageId"] != "own-1" {
						t.Fatal("wrong deletion payload")
					}
					if scenario == "transport-failure" {
						return nil, io.ErrUnexpectedEOF
					}
					body = `{"$type":"chat.bsky.convo.defs#deletedMessageView","id":"own-1","rev":"0002","sender":{"did":"did:plc:chat-service-owner"},"sentAt":"2026-10-31T12:00:00Z"}`
					if scenario == "bad-response" {
						body = `{"id":"forged"}`
					}
				default:
					t.Fatalf("unexpected upstream operation %s", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			if scenario == "revoked-child" {
				if _, err := store.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE token_hash=$1`, info.ChildTokenHash[:]); err != nil {
					t.Fatal(err)
				}
			}
			id, message := ConversationID("direct-1"), MessageID("own-1")
			if scenario == "foreign-conversation" {
				id = "another-conversation"
			}
			if scenario == "invalid-id" {
				message = "bad/id"
			}
			err := service.DeleteMessage(t.Context(), info, id, message)
			wantSuccess := scenario == "own" || scenario == "request"
			wantCalls := 0
			if wantSuccess || scenario == "bad-response" || scenario == "transport-failure" {
				wantCalls = 1
			}
			if (err == nil) != wantSuccess || deletes != wantCalls {
				t.Fatalf("success=%v calls=%d want=%v/%d err=%v", err == nil, deletes, wantSuccess, wantCalls, err)
			}
		})
	}
}

func TestDeleteMessageRevalidatesPreviouslyLoadedOlderPage(t *testing.T) {
	deletes := 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:sender"}]}}`
		case "/xrpc/chat.bsky.convo.getMessages":
			page, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
			if page == 6 {
				body = `{"messages":[{"$type":"chat.bsky.convo.defs#messageView","id":"old-message","rev":"old-rev","sender":{"did":"did:plc:chat-service-owner"},"text":"private-old-text","sentAt":"2026-10-19T11:00:00Z"}]}`
			} else {
				body = `{"messages":[],"cursor":"` + strconv.Itoa(page+1) + `"}`
			}
		case "/xrpc/chat.bsky.convo.deleteMessageForSelf":
			deletes++
			body = `{"$type":"chat.bsky.convo.defs#deletedMessageView","id":"old-message","rev":"new-rev","sender":{"did":"did:plc:chat-service-owner"},"sentAt":"2026-10-19T11:00:00Z"}`
		default:
			t.Fatal("unexpected method")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	var cursor Continuation
	for range 7 {
		page, err := service.History(t.Context(), info, "direct-1", cursor, 50)
		if err != nil {
			t.Fatal(err)
		}
		cursor = page.Cursor
	}
	if err := service.DeleteMessage(t.Context(), info, "direct-1", "old-message"); err != nil || deletes != 1 {
		t.Fatalf("deletes=%d err=%v", deletes, err)
	}
}
