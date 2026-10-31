package chat

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

func countView(id, status, peer string, unread int, extras, peerExtras string) string {
	return fmt.Sprintf(`{"$type":"chat.bsky.convo.defs#convoView","id":%q,"status":%q,"unreadCount":%d,"members":[{"did":"did:plc:chat-service-owner"},{"did":%q%s}]%s}`, id, status, unread, peer, peerExtras, extras)
}

func TestFilteredCountsDoNotRequireGlobalUnreadPermission(t *testing.T) {
	scopes := []string{"atproto"}
	for _, scope := range auth.OptionalChatScopes() {
		if !strings.Contains(scope, "getUnreadCounts") {
			scopes = append(scopes, scope)
		}
	}
	service, _, info := newServiceFixture(t, true, scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		if strings.HasSuffix(r.URL.Path, "listConvos") {
			body = `{"convos":[]}`
		}
		if strings.HasSuffix(r.URL.Path, "listConvoRequests") {
			body = `{"requests":[]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	status, err := service.Status(t.Context(), info)
	if err != nil || status.State != StateReady || !status.Capabilities.CanRead {
		t.Fatalf("unused global count grant blocks messaging: state=%s read=%t err=%v", status.State, status.Capabilities.CanRead, err)
	}
	if _, err := service.UnreadCounts(t.Context(), info); err != nil {
		t.Fatal(err)
	}
	for _, scope := range auth.OptionalChatScopes() {
		if strings.Contains(scope, "getUnreadCounts") {
			t.Fatal("unused global count authority still requested")
		}
	}
}

func TestCountsCapAndRejectIncompleteEnumeration(t *testing.T) {
	for _, mode := range []string{"cap", "cycle", "budget", "provider failure"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body, status := `{"chatDisabled":false}`, http.StatusOK
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.listConvoRequests":
					body = `{"requests":[]}`
				case "/xrpc/chat.bsky.convo.listConvos":
					calls++
					switch mode {
					case "cap":
						items := []string{}
						for i := range 100 {
							items = append(items, countView(fmt.Sprintf("unread-%d", i), "accepted", "did:plc:peer", 1, "", ""))
						}
						body = `{"convos":[` + strings.Join(items, ",") + `],"cursor":"not-needed-after-cap"}`
					case "cycle":
						body = `{"convos":[],"cursor":"same"}`
					case "budget":
						body = fmt.Sprintf(`{"convos":[],"cursor":"page-%d"}`, calls)
					case "provider failure":
						if calls == 1 {
							body = `{"convos":[` + countView("unread", "accepted", "did:plc:peer", 1, "", "") + `],"cursor":"next"}`
						} else {
							status, body = 503, `{"error":"Unavailable"}`
						}
					}
				default:
					t.Fatalf("unexpected counts call %s", r.URL.Path)
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			counts, err := service.UnreadCounts(t.Context(), info)
			if mode == "cap" {
				if err != nil || counts.Accepted != 100 || counts.Requests != 0 || calls != 1 {
					t.Fatalf("cap=%+v calls=%d err=%v", counts, calls, err)
				}
				return
			}
			if err == nil || counts != (UnreadCounts{}) {
				t.Fatalf("partial count published: %+v err=%v", counts, err)
			}
			if mode == "budget" && (calls != 20 || !errors.Is(err, ErrOperationUnavailable)) {
				t.Fatalf("unbounded work: calls=%d err=%v", calls, err)
			}
			if mode == "cycle" && calls != 2 {
				t.Fatalf("cycle calls=%d", calls)
			}
		})
	}
}

// UT-012 / AC-008, AC-031: global counts are not evidence of visible chats.
// A pending request still needs attention after reading or muting it.
func TestCountsOnlyIncludeCraftskyChatsAndAllPendingRequests(t *testing.T) {
	acceptedCalls, requestCalls := 0, 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getUnreadCounts":
			body = `{"unreadAcceptedConvos":100,"unreadRequestConvos":100}`
		case "/xrpc/chat.bsky.convo.listConvos":
			acceptedCalls++
			items := []string{
				countView("external", "accepted", "did:plc:non-crafter", 9, "", ""),
				countView("read", "accepted", "did:plc:peer", 0, "", ""),
				countView("muted", "accepted", "did:plc:peer", 1, `,"muted":true`, ""),
				countView("disabled", "accepted", "did:plc:peer", 1, "", `,"chatDisabled":true`),
				countView("blocked", "accepted", "did:plc:peer", 1, "", `,"viewer":{"blockedBy":true}`),
			}
			if r.URL.Query().Get("cursor") == "accepted-next" {
				item := countView("unread", "accepted", "did:plc:peer", 10, "", "")
				items = []string{item, item, countView("profile-muted", "accepted", "did:plc:peer", 1, "", `,"viewer":{"muted":true}`)}
				body = `{"convos":[` + strings.Join(items, ",") + `]}`
			} else {
				body = `{"convos":[` + strings.Join(items, ",") + `],"cursor":"accepted-next"}`
			}
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			requestCalls++
			if r.URL.Query().Get("cursor") == "requests-next" {
				items := []string{countView("read-request", "request", "did:plc:peer", 0, "", ""), countView("unread-request", "request", "did:plc:peer", 1, "", ""), countView("muted-request", "request", "did:plc:peer", 0, `,"muted":true`, "")}
				body = `{"requests":[` + strings.Join(items, ",") + `]}`
			} else {
				body = `{"requests":[` + countView("hidden-request", "request", "did:plc:non-crafter", 1, "", "") + `],"cursor":"requests-next"}`
			}
		default:
			t.Fatalf("unexpected counts operation %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	counts, err := service.UnreadCounts(t.Context(), info)
	if err != nil || counts.Accepted != 2 || counts.Requests != 3 || acceptedCalls != 2 || requestCalls != 2 {
		t.Fatalf("filtered counts=%+v accepted/request pages=%d/%d err=%v", counts, acceptedCalls, requestCalls, err)
	}
}
