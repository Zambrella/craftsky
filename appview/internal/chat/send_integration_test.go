package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func (runner testSessionRunner) WithCapabilityEffect(ctx context.Context, expected []ownerlifecycle.ExpectedOwner, owner syntax.DID, parent string, required []string, operation auth.OAuthSessionOperation) error {
	if parent != "synthetic-parent" || !auth.HasEffectiveScopes(runner.scopes, required) {
		return auth.ErrPDSPermissionsRequired
	}
	return runner.owners.WithActiveEffectSessionAuth(ctx, expected, owner, parent, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
		return operation(ctx, &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: owner, Scopes: runner.scopes}})
	})
}

// IT-006 / FR-005, FR-006: attempting commits before transmission and a
// provider failure after possible acceptance is never replayed by any layer.
func TestSendCommitsAttemptBeforeNetworkAndDoesNotReplay(t *testing.T) {
	var store *Store
	id := uuid.New()
	sends, calls := 0, 0
	service, created, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body := `{"chatDisabled":false}`
		status := 200
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:external-peer"}]}}`
		case "/xrpc/chat.bsky.convo.sendMessage":
			sends++
			var state string
			if err := store.pool.QueryRow(t.Context(), `SELECT state FROM chat_operations WHERE operation_id=$1`, id).Scan(&state); err != nil || state != "attempting" {
				t.Fatalf("pre-call receipt=%s err=%v", state, err)
			}
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request) != 2 || request["convoId"] != "direct-1" {
				t.Fatal("unexpected send body")
			}
			body = `{"error":"InternalError","message":"private-provider-canary"}`
			status = 500
		default:
			t.Fatalf("unexpected upstream method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	store = created
	ledger, err := NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{2}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	service.operations = ledger
	first, _ := service.Send(t.Context(), info, id, "direct-1", "private-message-canary")
	again, err := service.Send(t.Context(), info, id, "direct-1", "private-message-canary")
	if first.State != "uncertain" || again.State != "uncertain" || err != nil || sends != 1 {
		t.Fatalf("send states=%s/%s sends=%d err=%v", first.State, again.State, sends, err)
	}
	protected, err := service.Receipt(t.Context(), info, id)
	if err != nil || protected.State != "uncertain" || sends != 1 {
		t.Fatalf("protected receipt=%+v err=%v", protected, err)
	}
	if _, err := service.Receipt(t.Context(), info, uuid.New()); err == nil {
		t.Fatal("unknown receipt exposed")
	}
	before := calls
	if _, err := service.Send(t.Context(), info, uuid.New(), "direct-1", "   "); err == nil || calls != before {
		t.Fatal("invalid text reached upstream")
	}
}
