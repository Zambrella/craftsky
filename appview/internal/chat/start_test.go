package chat

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestExplicitStartAcceptsForInitiator(t *testing.T) {
	for _, test := range []struct {
		name            string
		initialStatus   string
		confirmedStatus string
		acceptFails     bool
		wantError       bool
	}{
		{name: "new conversation", initialStatus: "request", confirmedStatus: "accepted"},
		{name: "existing accepted conversation", initialStatus: "accepted"},
		{name: "acceptance fails", initialStatus: "request", acceptFails: true, wantError: true},
		{name: "acceptance unconfirmed", initialStatus: "request", confirmedStatus: "request", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			accepts, confirmations := 0, 0
			service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				status := test.initialStatus
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getConvoAvailability":
					body = `{"canChat":true}`
				case "/xrpc/chat.bsky.convo.getConvoForMembers":
				case "/xrpc/chat.bsky.convo.acceptConvo":
					accepts++
					var input struct {
						ConvoID string `json:"convoId"`
					}
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ConvoID != "canonical-1" || r.Method != http.MethodPost {
						t.Fatal("accept must target the initiating user's canonical conversation")
					}
					if test.acceptFails {
						return nil, errors.New("accept unavailable")
					}
					body = `{}`
				case "/xrpc/chat.bsky.convo.getConvo":
					confirmations++
					if r.URL.Query().Get("convoId") != "canonical-1" {
						t.Fatal("wrong confirmation conversation")
					}
					status = test.confirmedStatus
				default:
					t.Fatalf("unexpected operation %s", r.URL.Path)
				}
				if strings.HasSuffix(r.URL.Path, "getConvoForMembers") || strings.HasSuffix(r.URL.Path, "getConvo") {
					body = `{"convo":{"id":"canonical-1","status":"` + status + `","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			conversation, err := service.StartConversation(t.Context(), info, "did:plc:external-peer")
			if test.wantError {
				if err == nil {
					t.Fatal("start succeeded without confirmed acceptance")
				}
			} else if err != nil || conversation.Status != "accepted" || !conversation.CanSend {
				t.Fatalf("start status=%s canSend=%v err=%v", conversation.Status, conversation.CanSend, err)
			}
			wantAccepts, wantConfirmations := 1, 1
			if test.initialStatus == "accepted" {
				wantAccepts, wantConfirmations = 0, 0
			}
			if test.acceptFails {
				wantConfirmations = 0
			}
			if accepts != wantAccepts || confirmations != wantConfirmations {
				t.Fatalf("accepts=%d confirmations=%d, want %d/%d", accepts, confirmations, wantAccepts, wantConfirmations)
			}
		})
	}
}

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
