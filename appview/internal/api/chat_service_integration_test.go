package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

type chatServiceSessions struct{ owners *ownerlifecycle.Store }

func (s chatServiceSessions) WithCapabilityEffect(ctx context.Context, expected []ownerlifecycle.ExpectedOwner, owner syntax.DID, parent string, required []string, operation auth.OAuthSessionOperation) error {
	scopes := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	if parent != "synthetic-parent" || !auth.HasEffectiveScopes(scopes, required) {
		return auth.ErrPDSPermissionsRequired
	}
	return s.owners.WithActiveEffectSessionAuth(ctx, expected, owner, parent, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
		return operation(ctx, &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: owner, Scopes: scopes}})
	})
}

func (s chatServiceSessions) WithCapability(ctx context.Context, owner syntax.DID, parent string, required []string, operation auth.OAuthSessionOperation) error {
	scopes := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	if parent != "synthetic-parent" || !auth.HasEffectiveScopes(scopes, required) {
		return auth.ErrPDSPermissionsRequired
	}
	return s.owners.WithExistingAuth(ctx, owner, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
		return operation(ctx, &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: owner, Scopes: scopes}})
	})
}

type chatServiceTransport func(*http.Request) (*http.Response, error)

func (f chatServiceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newConsumerChatService(t *testing.T, upstream http.RoundTripper) (*chat.Service, auth.AuthInfo) {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	fence, err := ownerlifecycle.NewFencer(pool, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fence, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:chat-contract-owner")
	if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
		t.Fatal(err)
	}
	// Consumer message operations require both participants to have CraftSky profiles.
	if _, err := pool.Exec(t.Context(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'synthetic-profile'),('did:plc:external-peer','synthetic-profile')`, owner); err != nil {
		t.Fatal(err)
	}
	store, err := chat.NewStore(pool, owners)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Connection(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Connect(t.Context(), owner, 1, nil); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("synthetic-bearer"))
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at) VALUES($1,'synthetic-parent','{}','active',1,1,now()+interval '1 day')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day')`, hash[:], owner); err != nil {
		t.Fatal(err)
	}
	ledger, err := chat.NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{3}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := chat.NewService(chat.ServiceOptions{Store: store, Sessions: chatServiceSessions{owners}, Operations: ledger, Transport: func(*oauth.ClientSession) (*chat.Transport, error) {
		return chat.NewTransport(&atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: upstream}}), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	return service, auth.AuthInfo{DID: owner, SessionID: "synthetic-parent", ChildTokenHash: hash}
}

// IT-013 / AC-016, AC-019: test the real upstream/service/HTTP contract rather
// than injecting the already-normalized client exception.
func TestChatUpstreamConversationDenialReachesConsumer(t *testing.T) {
	status := 404
	service, info := newConsumerChatService(t, chatServiceTransport(func(r *http.Request) (*http.Response, error) {
		code, body := 200, `{"chatDisabled":false}`
		if strings.HasSuffix(r.URL.Path, "getConvo") {
			code, body = status, `{"error":"ConvoNotFound","message":"private-denial-canary"}`
		}
		return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	mux := http.NewServeMux()
	mux.Handle("GET /v1/chat/conversations/{id}", ChatConsumerHandler(service, "conversation"))
	mux.Handle("POST /v1/chat/sync", ChatConsumerHandler(service, "sync"))
	for _, upstreamStatus := range []int{403, 404, 410, 500} {
		status = upstreamStatus
		for _, request := range []struct{ method, path, body string }{
			{"GET", "/v1/chat/conversations/direct-1", ""},
			{"POST", "/v1/chat/sync", `{"view":"conversation","conversationId":"direct-1"}`},
		} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, chatContractRequest(request.method, request.path, request.body))
			want, code := 404, "chat_conversation_unavailable"
			if status == 500 {
				want, code = 503, "chat_unavailable"
			}
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != want || response.Error != code {
				t.Fatalf("upstream=%d route=%s status=%d code=%s", status, request.path, w.Code, response.Error)
			}
			if strings.Contains(w.Body.String(), "private-denial-canary") {
				t.Fatal("private cause exposed")
			}
		}
		_, err := service.Conversation(t.Context(), info, "direct-1")
		var provider *atclient.APIError
		if !errors.As(err, &provider) || provider.StatusCode != status {
			t.Fatal("typed provider cause lost")
		}
		if status != 500 && !errors.Is(err, chat.ErrInaccessibleConversation) {
			t.Fatal("access denial not classified")
		}
	}
}

// UT-006 / IT-019: provider backoff must survive the real service/HTTP boundary.
func TestChatConsumerPreservesUpstreamRetryAfter(t *testing.T) {
	service, _ := newConsumerChatService(t, chatServiceTransport(func(r *http.Request) (*http.Response, error) {
		header := make(http.Header)
		header.Set("Retry-After", "120")
		return &http.Response{StatusCode: 429, Header: header, Body: io.NopCloser(strings.NewReader(`{"error":"RateLimitExceeded","message":"private-rate-canary"}`)), Request: r}, nil
	}))
	mux := http.NewServeMux()
	mux.Handle("POST /v1/chat/sync", ChatConsumerHandler(service, "sync"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, chatContractRequest("POST", "/v1/chat/sync", `{"view":"inbox"}`))
	if w.Code != 429 || w.Header().Get("Retry-After") != "120" {
		t.Fatalf("rate contract: status=%d retry=%q", w.Code, w.Header().Get("Retry-After"))
	}
	if strings.Contains(w.Body.String(), "private-rate-canary") {
		t.Fatal("provider prose exposed")
	}
}

// IT-006 / IT-019: a real persisted rejection and duplicate keep their receipt
// through HTTP, instead of masquerading as ambiguous/503 responses.
func TestChatSendRejectedReceiptSurvivesHTTPAndDuplicate(t *testing.T) {
	sends := 0
	service, _ := newConsumerChatService(t, chatServiceTransport(func(r *http.Request) (*http.Response, error) {
		status, body := 200, `{"chatDisabled":false}`
		switch {
		case strings.HasSuffix(r.URL.Path, "getConvo"):
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-contract-owner"},{"did":"did:plc:external-peer"}]}}`
		case strings.HasSuffix(r.URL.Path, "sendMessage"):
			sends++
			status = 400
			body = `{"error":"InvalidRequest","message":"private-rejection-canary"}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	mux := http.NewServeMux()
	mux.Handle("POST /v1/chat/conversations/{id}/messages", ChatSendHandler(service))
	id := uuid.New()
	for range 2 {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, chatContractRequest("POST", "/v1/chat/conversations/direct-1/messages", `{"operationId":"`+id.String()+`","text":"private-send-input"}`))
		if w.Code != http.StatusOK {
			t.Fatalf("failed receipt replaced by HTTP %d", w.Code)
		}
		var receipt chat.SendOutcome
		if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || receipt.State != "failed" || receipt.OperationID != id || receipt.FailureCode != "rejected" {
			t.Fatalf("rejected receipt: status=%d state=%s code=%s", w.Code, receipt.State, receipt.FailureCode)
		}
		if receipt.Message != nil || strings.Contains(w.Body.String(), "private-") {
			t.Fatal("failed response exposed content")
		}
	}
	if sends != 1 {
		t.Fatalf("physical sends=%d", sends)
	}
}
