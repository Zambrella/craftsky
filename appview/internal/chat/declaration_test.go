package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-008 / FR-011, RULE-002: only self/allowIncoming changes, with bounded
// conflict refetch/merge; group and future declaration fields survive.
func TestIncomingPolicyMergesLatestDeclarationAfterConflict(t *testing.T) {
	puts, gets := 0, 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		status := 200
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/com.atproto.repo.getRecord":
			gets++
			if r.Header.Get("Atproto-Proxy") != "" || r.URL.Query().Get("repo") != "did:plc:chat-service-owner" || r.URL.Query().Get("collection") != "chat.bsky.actor.declaration" || r.URL.Query().Get("rkey") != "self" {
				t.Fatal("declaration read authority changed")
			}
			cid := "cid-old"
			extra := "old"
			incoming := "following"
			if gets > 1 {
				cid = "cid-new"
				extra = "concurrent"
			}
			if puts > 1 {
				incoming = "none"
			}
			body = `{"uri":"at://did:plc:chat-service-owner/chat.bsky.actor.declaration/self","cid":"` + cid + `","value":{"$type":"chat.bsky.actor.declaration","allowIncoming":"` + incoming + `","allowGroupInvites":"following","future":{"value":"` + extra + `"}}}`
		case "/xrpc/com.atproto.repo.putRecord":
			puts++
			var input struct {
				Repo       string         `json:"repo"`
				Collection string         `json:"collection"`
				Rkey       string         `json:"rkey"`
				SwapRecord *string        `json:"swapRecord"`
				Record     map[string]any `json:"record"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			expected := "cid-old"
			extra := "old"
			if puts > 1 {
				expected = "cid-new"
				extra = "concurrent"
			}
			if r.Header.Get("Atproto-Proxy") != "" || input.Repo != "did:plc:chat-service-owner" || input.Collection != "chat.bsky.actor.declaration" || input.Rkey != "self" || input.SwapRecord == nil || *input.SwapRecord != expected || input.Record["allowIncoming"] != "none" || input.Record["allowGroupInvites"] != "following" || input.Record["future"].(map[string]any)["value"] != extra {
				t.Fatal("conditional merge lost latest declaration fields")
			}
			if puts == 1 {
				status = 400
				body = `{"error":"InvalidSwap","message":"private-conflict-prose"}`
			} else {
				body = `{"uri":"at://did:plc:chat-service-owner/chat.bsky.actor.declaration/self","cid":"cid-written"}`
			}
		default:
			t.Fatalf("unexpected operation %s", r.URL.Path)
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	policy, err := service.SetIncomingPolicy(t.Context(), info, "nobody")
	if err != nil || policy.AllowIncoming != "nobody" || !policy.Persisted || puts != 2 || gets != 3 {
		t.Fatalf("policy=%+v puts=%d gets=%d err=%v", policy, puts, gets, err)
	}
	if _, err := service.SetIncomingPolicy(t.Context(), info, "arbitrary"); err == nil || puts != 2 {
		t.Fatal("invalid setting dispatched")
	}
	readback, err := service.IncomingPolicy(t.Context(), info)
	if err != nil || readback.AllowIncoming != "nobody" || !readback.Persisted || puts != 2 {
		t.Fatalf("policy read=%+v err=%v", readback, err)
	}
}
