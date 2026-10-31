package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type detailsSessionRunner struct {
	SessionRunner
	calls int
}

func (runner *detailsSessionRunner) WithCapability(ctx context.Context, owner syntax.DID, session string, scopes []string, operation auth.OAuthSessionOperation) error {
	runner.calls++
	return runner.SessionRunner.WithCapability(ctx, owner, session, scopes, func(ctx context.Context, client *oauth.ClientSession) error { return operation(ctx, client) })
}

// IT-002: initial detail loading has one checked upstream sequence, not two
// separately authenticated summary/history reads. Raw cursors stay private.
func TestDetailsLoadsConversationAndHistoryOnce(t *testing.T) {
	calls := []string{}
	service, _, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		if r.Method != http.MethodGet {
			t.Fatal("details caused an upstream mutation")
		}
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"request","unreadCount":2,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer","handle":"peer.test"}]}}`
		case "/xrpc/chat.bsky.convo.getMessages":
			if r.URL.Query().Get("convoId") != "direct-1" || r.URL.Query().Get("limit") != "50" {
				t.Fatal("wrong initial history query")
			}
			body = `{"messages":[{"$type":"chat.bsky.convo.defs#messageView","id":"message-2","rev":"2","sender":{"did":"did:plc:peer"},"text":"synthetic text","sentAt":"2026-10-21T12:02:00Z"},{"$type":"chat.bsky.convo.defs#deletedMessageView","id":"message-1","rev":"1","sentAt":"2026-10-21T12:01:00Z"}],"cursor":"private-details-cursor"}`
			if r.URL.Query().Get("cursor") != "" {
				if r.URL.Query().Get("cursor") != "private-details-cursor" {
					t.Fatal("opaque cursor forwarded upstream")
				}
				body = `{"messages":[]}`
			}
		default:
			t.Fatal("unexpected upstream method")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	runner := &detailsSessionRunner{SessionRunner: service.sessions}
	service.sessions = runner
	result, err := service.Details(t.Context(), info, "direct-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Conversation.ID != "direct-1" || result.Conversation.Status != "request" || result.Conversation.CanSend || result.Conversation.UnreadCount != 2 {
		t.Fatal("details changed request metadata")
	}
	if len(result.Messages.Items) != 2 || result.Messages.Items[0].Kind != "deleted" || result.Messages.Items[1].Text != "synthetic text" || result.Messages.Cursor == "" {
		t.Fatal("details lost mapped history/pagination")
	}
	want := []string{"/xrpc/chat.bsky.actor.getStatus", "/xrpc/chat.bsky.convo.getConvo", "/xrpc/chat.bsky.convo.getMessages"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v", calls)
	}
	if runner.calls != 1 {
		t.Fatalf("session boundaries=%d", runner.calls)
	}

	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "private-details-cursor") {
		t.Fatal("upstream cursor leaked")
	}
	next, err := service.History(t.Context(), info, "direct-1", result.Messages.Cursor, 50)
	if err != nil || len(next.Items) != 0 {
		t.Fatal("combined continuation cannot resume standard history")
	}

}

func TestDetailsFailsClosedBeforePrivateHistory(t *testing.T) {
	for _, mode := range []string{"disabled", "missing scopes", "revoked child", "not a member", "missing profile", "status failure", "history failure", "malformed history"} {
		t.Run(mode, func(t *testing.T) {
			historyCalls := 0
			scopes := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
			if mode == "missing scopes" {
				filtered := []string{}
				for _, scope := range scopes {
					if !strings.Contains(scope, "chat.bsky.convo.getMessages") {
						filtered = append(filtered, scope)
					}
				}
				scopes = filtered
			}
			service, store, info := newServiceFixture(t, mode != "disabled", scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				code := 200
				body := `{"chatDisabled":false}`
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
					if mode == "status failure" {
						code = 503
						body = `{"error":"Unavailable"}`
					}
				case "/xrpc/chat.bsky.convo.getConvo":
					peer := "did:plc:peer"
					owner := "did:plc:chat-service-owner"
					if mode == "missing profile" {
						peer = "did:plc:no-profile"
					}
					if mode == "not a member" {
						owner = "did:plc:not-owner"
					}
					body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"` + owner + `"},{"did":"` + peer + `"}]}}`
				case "/xrpc/chat.bsky.convo.getMessages":
					historyCalls++
					body = `{"messages":[]}`
					if mode == "history failure" {
						code = 503
						body = `{"error":"Unavailable"}`
					}
					if mode == "malformed history" {
						body = `{"messages":null}`
					}
				default:
					t.Fatal("unexpected upstream operation")
				}
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			if mode == "revoked child" {
				if _, err := store.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE token_hash=$1`, info.ChildTokenHash[:]); err != nil {
					t.Fatal(err)
				}
			}
			result, err := service.Details(t.Context(), info, "direct-1")
			if err == nil || result.Conversation.ID != "" || len(result.Messages.Items) != 0 {
				t.Fatal("details published partial/unauthorized content")
			}
			wantHistory := 0
			if mode == "history failure" || mode == "malformed history" {
				wantHistory = 1
			}
			if historyCalls != wantHistory {
				t.Fatalf("history calls=%d want=%d", historyCalls, wantHistory)
			}
		})
	}
}
