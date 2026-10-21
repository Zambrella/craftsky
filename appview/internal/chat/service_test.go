package chat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

type testSessionRunner struct {
	owners *ownerlifecycle.Store
	scopes []string
}

func (runner testSessionRunner) WithCapability(ctx context.Context, owner syntax.DID, parent string, required []string, operation auth.OAuthSessionOperation) error {
	if parent != "synthetic-parent" || !auth.HasEffectiveScopes(runner.scopes, required) {
		return auth.ErrPDSPermissionsRequired
	}
	return runner.owners.WithExistingAuth(ctx, owner, func(ctx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateActive {
			return ownerlifecycle.ErrOwnerNotActive
		}
		return operation(ctx, &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: owner, Scopes: runner.scopes}})
	})
}

func newServiceFixture(t *testing.T, enabled bool, scopes []string, upstream http.RoundTripper) (*Service, *Store, auth.AuthInfo) {
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
	owner := syntax.DID("did:plc:chat-service-owner")
	if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(pool, owners)
	if err != nil {
		t.Fatal(err)
	}
	seedCraftskyProfiles(t, store, owner.String(), "did:plc:external-peer", "did:plc:peer", "did:plc:sender")
	if _, err := store.Connection(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	if enabled {
		if _, err := store.Connect(t.Context(), owner, 1, nil); err != nil {
			t.Fatal(err)
		}
	}
	service, err := NewService(ServiceOptions{Store: store, Sessions: testSessionRunner{owners: owners, scopes: scopes}, Transport: func(*oauth.ClientSession) (*Transport, error) {
		return NewTransport(&atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: upstream}}), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	service.operations, err = NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{3}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("synthetic-bearer"))
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at) VALUES($1,'synthetic-parent','{}','active',1,1,now()+interval '1 day')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at,last_device_id) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day','synthetic-device')`, hash[:], owner); err != nil {
		t.Fatal(err)
	}
	return service, store, auth.AuthInfo{DID: owner, SessionID: "synthetic-parent", ChildTokenHash: hash}
}

// IT-005 / IT-013 / FR-016: a sibling child keeping the parent usable
// cannot lend its authority to a request whose exact child was revoked.
func TestStatusRejectsRevokedExactChildDespiteLiveSibling(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		calls := 0
		service, store, info := newServiceFixture(t, enabled, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
		}))
		sibling := sha256.Sum256([]byte("other-bearer"))
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day')`, sibling[:], info.DID); err != nil {
			t.Fatal(err)
		}
		if _, err := store.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE token_hash=$1`, info.ChildTokenHash[:]); err != nil {
			t.Fatal(err)
		}
		status, err := service.Status(t.Context(), info)
		if err != nil || status.State != StateAuthorizationNeeded || calls != 0 {
			t.Fatalf("status=%+v calls=%d err=%v", status, calls, err)
		}
	}
}

// UT-001 / FR-001: partial core consent is not ready; independently lost
// reporting/declaration authority does not stop ordinary messaging.
func TestStatusRequiresCoreGrantAndConfiguredOperations(t *testing.T) {
	full := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	for _, test := range []struct {
		name, omit, want string
		noKey            bool
	}{
		{"missing message read", "rpc:chat.bsky.convo.getMessages?", "authorizationNeeded", false},
		{"missing accept", "rpc:chat.bsky.convo.acceptConvo?", "authorizationNeeded", false},
		{"missing report", "rpc:com.atproto.moderation.createReport?", "ready", false},
		{"missing policy", "repo:chat.bsky.actor.declaration?", "ready", false},
		{"operation key unavailable", "", "unavailable", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			scopes := make([]string, 0, len(full))
			for _, scope := range full {
				if test.omit == "" || !strings.HasPrefix(scope, test.omit) {
					scopes = append(scopes, scope)
				}
			}
			calls := 0
			service, _, info := newServiceFixture(t, true, scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
			}))
			if test.noKey {
				service.operations = nil
			}
			status, err := service.Status(t.Context(), info)
			if err != nil || string(status.State) != test.want {
				t.Fatalf("state=%s err=%v", status.State, err)
			}
			if test.want != "ready" && calls != 0 {
				t.Fatal("incomplete connection dispatched eligibility read")
			}
		})
	}
}

// IT-005 / IT-017: clients can observe reconnect even if both sampled states
// are ready, so an old private presentation cannot survive a new generation.
func TestStatusCarriesCurrentMessagingGeneration(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
	}))
	before, err := service.Status(t.Context(), info)
	if err != nil || before.Generation < 1 {
		t.Fatal("current messaging generation absent")
	}
	disconnected, err := store.Disconnect(t.Context(), info.DID)
	if err != nil {
		t.Fatal(err)
	}
	connected, err := store.Connect(t.Context(), info.DID, disconnected.Generation, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, err := service.Status(t.Context(), info)
	if err != nil || after.State != StateReady || after.Generation != connected.Generation || after.Generation <= before.Generation {
		t.Fatal("reconnect generation not reflected")
	}
}

// UT-001 / AT-002 / FR-001: unavailable and restricted are honest states,
// and no provider prose or credential is serialized to the consumer.
func TestStatusDistinguishesConsentRestrictionAndFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		enabled bool
		scopes  []string
		body    string
		err     error
		want    string
	}{
		{"disabled", false, auth.DefaultScopes(), `{}`, nil, "disabled"},
		{"needs authorization", true, auth.DefaultScopes(), `{}`, nil, "authorizationNeeded"},
		{"restricted", true, append([]string{"atproto"}, auth.OptionalChatScopes()...), `{"chatDisabled":true}`, nil, "restricted"},
		{"unavailable", true, append([]string{"atproto"}, auth.OptionalChatScopes()...), ``, errors.New("private-provider-canary"), "unavailable"},
		{"missing authoritative eligibility", true, append([]string{"atproto"}, auth.OptionalChatScopes()...), `{}`, nil, "unavailable"},
		{"ready", true, append([]string{"atproto"}, auth.OptionalChatScopes()...), `{"chatDisabled":false}`, nil, "ready"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service, _, info := newServiceFixture(t, test.enabled, test.scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if test.err != nil {
					return nil, test.err
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body)), Request: r}, nil
			}))
			status, err := service.Status(t.Context(), info)
			if err != nil || string(status.State) != test.want {
				t.Fatalf("status=%+v err=%v", status, err)
			}
			if (!test.enabled || test.want == "authorizationNeeded") && calls != 0 {
				t.Fatal("unauthorized status dispatched")
			}
			raw, _ := json.Marshal(status)
			if strings.Contains(string(raw), "private-provider-canary") {
				t.Fatal("provider prose serialized")
			}
		})
	}
}

// UT-001 / IT-015: an eligibility read can refresh the optional grant.
// Independent capabilities must describe the resulting grant, not its snapshot.
func TestStatusUsesGrantAfterEligibilityRefresh(t *testing.T) {
	full := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	service, _, info := newServiceFixture(t, true, full, nil)
	service.transport = func(session *oauth.ClientSession) (*Transport, error) {
		return NewTransport(&atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var refreshed []string
			for _, scope := range session.Data.Scopes {
				if !strings.HasPrefix(scope, "rpc:com.atproto.moderation.createReport?") && !strings.HasPrefix(scope, "repo:chat.bsky.actor.declaration?") {
					refreshed = append(refreshed, scope)
				}
			}
			session.Data.Scopes = refreshed
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
		})}}), nil
	}
	status, err := service.Status(t.Context(), info)
	if err != nil || status.State != StateReady || status.Capabilities.CanReport || status.Capabilities.CanSetPolicy || !status.Capabilities.CanRead {
		t.Fatalf("post-refresh status=%+v err=%v", status, err)
	}
}

// UT-006 / FR-015: a transient eligibility failure must remain distinct from
// lost grant authority, so an existing same-generation client can retain its
// loaded presentation while every new operation still rechecks eligibility.
func TestStatusTransientEligibilityKeepsKnownGrant(t *testing.T) {
	service, _, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"InternalError"}`)), Request: r}, nil
	}))
	status, err := service.Status(t.Context(), info)
	if err != nil || status.State != StateUnavailable || !status.Enabled || !status.Capabilities.CanRead {
		t.Fatalf("transient eligibility status=%+v err=%v", status, err)
	}
}

// UT-006 / FR-015: status exposes only safe synchronization metadata.
func TestStatusIncludesSafeSyncRecoveryMetadata(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
	}))
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,safe_state,last_success_at,cursor) VALUES($1,'rateLimited',$2,'private-cursor-canary')`, info.DID.String(), at); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(t.Context(), info)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(status)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	if body["syncState"] != "rateLimited" || body["lastSuccessAt"] != "2026-10-19T12:00:00Z" || body["recoveryCode"] != "rateLimited" {
		t.Fatalf("missing safe recovery metadata: %s", raw)
	}
	if strings.Contains(string(raw), "private-cursor-canary") {
		t.Fatal("cursor leaked")
	}
}
