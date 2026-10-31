package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
)

type capabilitySessionRunner interface {
	WithCapability(context.Context, syntax.DID, string, []string, auth.OAuthSessionOperation) error
}

type capabilityBackgroundSelector interface {
	SelectForCapability(context.Context, syntax.DID, []string) (string, error)
}

// IT-015 / RULE-004, NFR-001 / AC-019.
func TestChatBackgroundSelectsCapableSameOwnerSession(t *testing.T) {
	pool := withAuthSchema(t)
	owner, other := syntax.DID("did:plc:chat-select-owner"), syntax.DID("did:plc:chat-select-other")
	seedActiveAuthOwner(t, pool, owner)
	seedActiveAuthOwner(t, pool, other)
	capable := append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	now := time.Now().UTC()
	for _, row := range []struct {
		owner                syntax.DID
		id                   string
		scopes               []string
		lastSeen, idleExpiry time.Time
	}{
		{owner, "capable", capable, now.Add(-time.Hour), now.Add(time.Hour)},
		{owner, "newer-base", auth.DefaultScopes(), now.Add(-time.Minute), now.Add(time.Hour)},
		{owner, "expired-capable", capable, now.Add(-2 * time.Hour), now.Add(-time.Minute)},
		{other, "other-capable", capable, now, now.Add(time.Hour)},
	} {
		data := validOAuthSession(row.owner, row.id)
		data.Scopes = row.scopes
		if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at)
			VALUES($1,$2,$3,'active',1,1,now()+interval '1 day')`, row.owner, row.id, data); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,last_seen_at,idle_expires_at,created_at)
			VALUES(convert_to($2,'UTF8'),$1,$2,'active',1,$3,$4,now()-interval '2 days')`, row.owner, row.id, row.lastSeen, row.idleExpiry); err != nil {
			t.Fatal(err)
		}
	}
	selector := auth.NewBackgroundSessionSelector(pool)
	runner, ok := any(selector).(capabilityBackgroundSelector)
	if !ok {
		t.Fatal("background selector has no capability-aware boundary")
	}
	id, err := runner.SelectForCapability(t.Context(), owner, auth.OptionalChatScopes())
	if err != nil || id != "capable" {
		t.Fatalf("selected session = %q, error %v", id, err)
	}
	var lastSeen time.Time
	if err := pool.QueryRow(t.Context(), `SELECT last_seen_at FROM craftsky_sessions WHERE account_did=$1 AND oauth_session_id=$2`, owner, id).Scan(&lastSeen); err != nil {
		t.Fatal(err)
	}
	if lastSeen.Sub(now.Add(-time.Hour)).Abs() > time.Microsecond {
		t.Fatal("background selection touched child activity")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE oauth_sessions SET lifecycle_state='revocation_pending',revocation_requested_at=now() WHERE account_did=$1 AND session_id='capable'`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.SelectForCapability(t.Context(), owner, auth.OptionalChatScopes()); !errors.Is(err, auth.ErrNoUsableBackgroundSession) {
		t.Fatalf("missing eligible capability = %v", err)
	}
}

// IT-015 / BR-002, RULE-004, NFR-001 / AC-002, AC-019.
func TestChatCapabilityDenialPreservesBaseSession(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:chat-capability-owner")
	seedActiveAuthOwner(t, pool, owner)
	data := validOAuthSession(owner, "base-chat-parent")
	data.Scopes = auth.DefaultScopes()
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	data.DPoPPrivateKeyMultibase = key.Multibase()
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(
		account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
		row_version,absolute_expires_at
	) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day')`, owner, data.SessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
			DID: owner, PDSOrigin: data.HostURL, IssuerOrigin: data.AuthServerURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner, ok := any(coordinator).(capabilitySessionRunner)
	if !ok {
		t.Fatal("coordinator has no operation-capability boundary")
	}
	called := false
	err = runner.WithCapability(t.Context(), owner, data.SessionID, auth.OptionalChatScopes(), func(context.Context, *oauth.ClientSession) error {
		called = true
		return nil
	})
	if called || !errors.Is(err, auth.ErrPDSPermissionsRequired) || errors.Is(err, auth.ErrPDSSessionExpired) {
		t.Fatalf("chat denial = called %t, error %v", called, err)
	}
	if err := coordinator.WithActiveSession(t.Context(), owner, data.SessionID, func(context.Context, *oauth.ClientSession) error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatalf("ordinary access after chat denial = called %t, error %v", called, err)
	}
	var state string
	if err := pool.QueryRow(t.Context(), `SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, data.SessionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "active" {
		t.Fatalf("optional denial changed base session state to %s", state)
	}
}

// IT-015 / BR-002, RULE-004, NFR-001 / AC-002, AC-019.
func TestChatUpstreamPermissionLossPreservesBaseSession(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:chat-capability-owner")
	seedActiveAuthOwner(t, pool, owner)
	data := validOAuthSession(owner, "base-chat-parent")
	data.Scopes = append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	data.DPoPPrivateKeyMultibase = key.Multibase()
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(
		account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
		row_version,absolute_expires_at
	) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day')`, owner, data.SessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
			DID: owner, PDSOrigin: data.HostURL, IssuerOrigin: data.AuthServerURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	runner, ok := any(coordinator).(capabilitySessionRunner)
	if !ok {
		t.Fatal("coordinator has no operation-capability boundary")
	}
	called := false
	upstream := &atclient.APIError{StatusCode: http.StatusForbidden, Name: "InsufficientScope", Message: "private-provider-canary"}
	err = runner.WithCapability(t.Context(), owner, data.SessionID, auth.OptionalChatScopes(), func(context.Context, *oauth.ClientSession) error {
		called = true
		return upstream
	})
	if !called || !errors.Is(err, auth.ErrPDSPermissionsRequired) || errors.Is(err, auth.ErrPDSSessionExpired) {
		t.Fatalf("chat denial = called %t, error %v", called, err)
	}
	var cause *atclient.APIError
	if !errors.As(err, &cause) || cause != upstream {
		t.Fatal("optional permission rejection lost original cause")
	}
	called = false
	if err := coordinator.WithActiveSession(t.Context(), owner, data.SessionID, func(context.Context, *oauth.ClientSession) error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatalf("ordinary access after chat denial = called %t, error %v", called, err)
	}
	var state string
	if err := pool.QueryRow(t.Context(), `SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, data.SessionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "active" {
		t.Fatalf("optional denial changed base session state to %s", state)
	}
}

// IT-015 / RULE-004, NFR-001 / AC-019.
func TestChatRefreshRechecksEffectiveOptionalGrant(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:chat-capability-owner")
	seedActiveAuthOwner(t, pool, owner)
	data := validOAuthSession(owner, "base-chat-parent")
	data.Scopes = append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	data.DPoPPrivateKeyMultibase = key.Multibase()
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(
		account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
		row_version,absolute_expires_at
	) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day')`, owner, data.SessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	resourceCalls := 0
	client := &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == data.AuthServerTokenEndpoint {
			body, err := json.Marshal(oauth.TokenResponse{Subject: owner.String(), Scope: strings.Join(auth.DefaultScopes(), " "), AccessToken: "rotated-access", RefreshToken: "rotated-refresh"})
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
		}
		resourceCalls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: request}, nil
	})}
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: client, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
			DID: owner, PDSOrigin: data.HostURL, IssuerOrigin: data.AuthServerURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.WithCapability(t.Context(), owner, data.SessionID, auth.OptionalChatScopes(), func(ctx context.Context, session *oauth.ClientSession) error {
		if _, err := session.RefreshTokens(ctx); err != nil {
			return err
		}
		return session.APIClient().WithService(auth.ChatServiceAudience).Post(ctx, "chat.bsky.convo.sendMessage", map[string]any{"convoId": "synthetic-conversation", "message": map[string]any{"text": "synthetic-private-text"}}, nil)
	})
	if !errors.Is(err, auth.ErrPDSPermissionsRequired) || errors.Is(err, auth.ErrPDSSessionExpired) || resourceCalls != 0 {
		t.Fatalf("refreshed optional denial: resource calls=%d, err=%v", resourceCalls, err)
	}
	var stored oauth.ClientSessionData
	var raw []byte
	if err := pool.QueryRow(t.Context(), `SELECT data FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, data.SessionID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.RefreshToken != "rotated-refresh" || strings.Join(stored.Scopes, " ") != strings.Join(auth.DefaultScopes(), " ") {
		t.Fatal("refreshed credentials/effective grants were not persisted together")
	}
	if err := coordinator.WithActiveSession(t.Context(), owner, data.SessionID, func(context.Context, *oauth.ClientSession) error { return nil }); err != nil {
		t.Fatalf("base access after optional scope loss = %v", err)
	}
}

// IT-015 / RULE-004, NFR-001 / AC-019.
func TestChatPurposeClientRechecksRefreshedGrant(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:chat-capability-owner")
	seedActiveAuthOwner(t, pool, owner)
	data := validOAuthSession(owner, "base-chat-parent")
	data.Scopes = append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	data.DPoPPrivateKeyMultibase = key.Multibase()
	if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(
		account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
		row_version,absolute_expires_at
	) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day')`, owner, data.SessionID, data); err != nil {
		t.Fatal(err)
	}
	config := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	resourceCalls := 0
	client := &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == data.AuthServerTokenEndpoint {
			body, err := json.Marshal(oauth.TokenResponse{Subject: owner.String(), Scope: strings.Join(auth.DefaultScopes(), " "), AccessToken: "rotated-access", RefreshToken: "rotated-refresh"})
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
		}
		resourceCalls++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: request}, nil
	})}
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App:   &oauth.ClientApp{Client: client, Config: &config},
		Store: store, Owners: owners,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
			DID: owner, PDSOrigin: data.HostURL, IssuerOrigin: data.AuthServerURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.WithCapability(t.Context(), owner, data.SessionID, auth.OptionalChatScopes(), func(ctx context.Context, session *oauth.ClientSession) error {
		if _, err := session.RefreshTokens(ctx); err != nil {
			return err
		}
		binder, ok := session.Client.Transport.(interface {
			ResourceClient(*http.Client) *http.Client
		})
		if !ok {
			t.Fatal("purpose-specific resource transport cannot share refreshed optional authority")
		}
		resource := session.APIClient().WithService(auth.ChatServiceAudience)
		resource.Client = binder.ResourceClient(client)
		return resource.Post(ctx, "chat.bsky.convo.sendMessage", map[string]any{"convoId": "synthetic-conversation", "message": map[string]any{"text": "synthetic-private-text"}}, nil)
	})
	if !errors.Is(err, auth.ErrPDSPermissionsRequired) || errors.Is(err, auth.ErrPDSSessionExpired) || resourceCalls != 0 {
		t.Fatalf("refreshed optional denial: resource calls=%d, err=%v", resourceCalls, err)
	}
	var stored oauth.ClientSessionData
	var raw []byte
	if err := pool.QueryRow(t.Context(), `SELECT data FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, data.SessionID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.RefreshToken != "rotated-refresh" || strings.Join(stored.Scopes, " ") != strings.Join(auth.DefaultScopes(), " ") {
		t.Fatal("refreshed credentials/effective grants were not persisted together")
	}
	if err := coordinator.WithActiveSession(t.Context(), owner, data.SessionID, func(context.Context, *oauth.ClientSession) error { return nil }); err != nil {
		t.Fatalf("base access after optional scope loss = %v", err)
	}
}

// IT-006 / IT-004 / IT-015: exercise the pinned OAuth SDK, purpose-bound
// resource client and chat adapter together, including a legitimate nonce
// challenge followed by an ambiguous failure. No real service is contacted.
func TestChatPurposeWriteFaultsDoNotReplay(t *testing.T) {
	for _, method := range []syntax.NSID{"chat.bsky.convo.sendMessage", "com.atproto.moderation.createReport"} {
		for _, fault := range []string{"server", "rate", "drop", "nonceThenDrop"} {
			t.Run(method.String()+"/"+fault, func(t *testing.T) {
				pool := withAuthSchema(t)
				owners := newAuthOwnerStore(t, pool)
				storeConfig := testStoreConfig()
				storeConfig.OwnerLifecycles = owners
				store := auth.NewPostgresAuthStore(pool, storeConfig)
				owner := syntax.DID("did:plc:chat-write-fault-owner")
				seedActiveAuthOwner(t, pool, owner)
				data := validOAuthSession(owner, "chat-write-parent")
				data.Scopes = append(auth.DefaultScopes(), auth.OptionalChatScopes()...)
				data.AccessToken = "synthetic-access"
				data.RefreshToken = "synthetic-refresh"
				key, err := atcrypto.GeneratePrivateKeyP256()
				if err != nil {
					t.Fatal(err)
				}
				data.DPoPPrivateKeyMultibase = key.Multibase()
				if _, err := pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,row_version,absolute_expires_at) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day')`, owner, data.SessionID, data); err != nil {
					t.Fatal(err)
				}
				calls := 0
				audience := auth.ChatServiceAudience
				if method == "com.atproto.moderation.createReport" {
					audience = auth.ChatReportAudience
				}
				client := &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.URL.Path != "/xrpc/"+method.String() || request.Header.Get("Atproto-Proxy") != audience || request.Header.Get("DPoP") == "" {
						t.Fatal("write escaped fixed authenticated purpose")
					}
					calls++
					body, err := io.ReadAll(request.Body)
					if err != nil || !strings.Contains(string(body), "private-write-canary") {
						t.Fatal("write body lost across SDK boundary")
					}
					if fault == "drop" || (fault == "nonceThenDrop" && calls > 1) {
						return nil, errors.New("synthetic-connection-drop")
					}
					status := 500
					header := make(http.Header)
					if fault == "rate" {
						status = 429
						header.Set("Retry-After", "120")
					}
					if fault == "nonceThenDrop" {
						status = 401
						header.Set("WWW-Authenticate", `DPoP error="use_dpop_nonce"`)
						header.Set("DPoP-Nonce", "synthetic-resource-nonce")
					}
					return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(`{"error":"SyntheticFailure","message":"private-provider-canary"}`)), Request: request}, nil
				})}
				config := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
				coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{App: &oauth.ClientApp{Client: client, Config: &config}, Store: store, Owners: owners,
					AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{DID: owner, PDSOrigin: data.HostURL, IssuerOrigin: data.AuthServerURL}}})
				if err != nil {
					t.Fatal(err)
				}
				err = coordinator.WithCapability(t.Context(), owner, data.SessionID, auth.OptionalChatScopes(), func(ctx context.Context, session *oauth.ClientSession) error {
					resource, err := auth.CapabilityAPIClient(session, client)
					if err != nil {
						return err
					}
					var result map[string]any
					return chat.NewTransport(resource).Procedure(ctx, method, map[string]any{"text": "private-write-canary"}, &result)
				})
				want := 1
				if fault == "nonceThenDrop" {
					want = 2
				}
				if err == nil || calls != want {
					t.Fatalf("physical write calls=%d want=%d error type=%T", calls, want, err)
				}
			})
		}
	}
}
