package app

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ownerlifecycle"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRealOAuthMetadataRejectsPrivateEndpointsBeforeTrapOrBaseDial(t *testing.T) {
	trap := newRealFlowTrap(t)
	privateOrigin := "https://" + trap.Addr().String()
	tests := []struct {
		name string
		set  func(*realFlowOAuthEndpoints)
	}{
		{
			name: "PAR",
			set: func(endpoints *realFlowOAuthEndpoints) {
				endpoints.par = privateOrigin + "/oauth/par"
			},
		},
		{
			name: "token",
			set: func(endpoints *realFlowOAuthEndpoints) {
				endpoints.token = privateOrigin + "/oauth/token"
			},
		},
		{
			name: "revocation",
			set: func(endpoints *realFlowOAuthEndpoints) {
				endpoints.revocation = privateOrigin + "/oauth/revoke"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owner := syntax.DID("did:plc:privateendpoint")
			upstream := newRealFlowServer(t, owner)
			endpoints := defaultRealFlowOAuthEndpoints()
			test.set(&endpoints)
			upstream.setOAuthEndpoints(endpoints)
			clients, dialer, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})

			request, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet,
				realFlowAuthOrigin+"/.well-known/oauth-authorization-server", nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			response, err := clients.metadata.Do(request)
			if response != nil {
				_ = response.Body.Close()
				t.Fatal("private-endpoint metadata response escaped validation")
			}
			if !errors.Is(err, federatedhttp.ErrDestinationRejected) ||
				federatedhttp.Classify(err) != federatedhttp.KindDestinationRejected {
				t.Fatalf("metadata error = %v, want destination rejection", err)
			}
			var boundaryError *federatedhttp.Error
			if !errors.As(err, &boundaryError) {
				t.Fatalf("metadata error = %v, want typed boundary error", err)
			}
			for _, forbidden := range []string{"127.0.0.1", trap.Addr().String()} {
				if strings.Contains(boundaryError.Error(), forbidden) {
					t.Fatalf("redacted metadata error exposed %q: %v", forbidden, boundaryError)
				}
			}
			for _, address := range dialer.calls() {
				if strings.Contains(address, "127.0.0.1") {
					t.Fatalf("base dialer reached private endpoint: %v", dialer.calls())
				}
			}
			if trap.count() != 0 {
				t.Fatalf("private endpoint trap connections = %d, want zero", trap.count())
			}

			clients.boundary.CloseIdleConnections()
			upstream.close(t)
			requests, accepted := upstream.observations()
			if len(requests) != 1 || accepted != 1 || len(dialer.calls()) != 1 {
				t.Fatalf(
					"metadata fixture requests=%v accepted=%d base_dials=%v",
					requests, accepted, dialer.calls(),
				)
			}
		})
	}
}

func TestHandleFirstOAuthRejectsMismatchedTokenSubject(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:handlefirstowner")
	handle := syntax.Handle("handle-first.real-flow.test")
	upstream := newRealFlowServer(t, owner)
	upstream.setTokenResponse(map[string]any{
		"sub": "did:plc:differentowner", "scope": "atproto transition:generic",
		"access_token": "mismatched-access", "refresh_token": "mismatched-refresh",
	})
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: handle,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	owners := newRealFlowOwnerStore(t, pool)
	storeConfig := realFlowStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = clients.boundary
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode: auth.ClientModeLocalhost, CallbackURL: realFlowURL(t, "http://127.0.0.1:18080/oauth/callback"),
		Scopes: []string{"atproto", "transition:generic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	oauthApp := oauth.NewClientApp(&artifacts.Config, store)
	oauthApp.Client = clients.oauth
	oauthApp.Resolver.Client = clients.metadata
	oauthApp.Dir = clients.directory
	authorityVerifier, err := newAuthoritativeOAuthVerifier(oauthApp.Dir, oauthApp.Resolver)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
		AuthorityVerifier: authorityVerifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.StartLogin(context.Background(), handle, auth.HandoffVerifiedLink, "", "subject-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	finalized := false
	err = flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"subject-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		finalized = true
		return nil
	})
	if !errors.Is(err, auth.ErrOAuthFlowInvalid) || finalized {
		t.Fatalf("mismatched subject completion err=%v finalized=%t", err, finalized)
	}
	var requestState string
	var sessions int
	if err := pool.QueryRow(context.Background(), `SELECT request_state FROM oauth_auth_requests WHERE state=$1`, state).Scan(&requestState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if requestState != "exchange_ambiguous" || sessions != 0 {
		t.Fatalf("mismatched subject request/sessions=%s/%d", requestState, sessions)
	}
}

func TestRealFederatedOAuthSessionAndPDSFlowsUsePurposeClients(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:realflowowner")
	handle := syntax.Handle("alice.real-flow.test")
	upstream := newRealFlowServer(t, owner)
	clients, dialer, observer := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})

	identity := &identity.Identity{
		DID: owner, Handle: handle,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}
	directory := realFlowDirectory{identity: identity}
	clients.directory = directory
	owners := newRealFlowOwnerStore(t, pool)
	storeConfig := realFlowStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = clients.boundary
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:        auth.ClientModeLocalhost,
		CallbackURL: realFlowURL(t, "http://127.0.0.1:18080/oauth/callback"),
		Scopes:      []string{"atproto", "transition:generic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	oauthApp := oauth.NewClientApp(&artifacts.Config, store)
	oauthApp.Client = clients.oauth
	oauthApp.Resolver.Client = clients.metadata
	oauthApp.Dir = clients.directory
	authorityVerifier, err := newAuthoritativeOAuthVerifier(oauthApp.Dir, oauthApp.Resolver)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
		AuthorityVerifier: authorityVerifier,
	})
	if err != nil {
		t.Fatal(err)
	}

	redirect, err := flow.StartLogin(
		context.Background(), handle, auth.HandoffVerifiedLink, "", "real-flow-device",
	)
	if err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	redirectURL, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if redirectURL.Scheme != "https" || redirectURL.Host != "auth.real-flow.test" ||
		redirectURL.Path != "/oauth/authorize" || redirectURL.Query().Get("request_uri") == "" {
		t.Fatalf("authorization redirect = %q", redirect)
	}
	requests, _ := upstream.observations()
	var parRequest *realFlowRequest
	for index := range requests {
		if requests[index].path == "/oauth/par" {
			parRequest = &requests[index]
			break
		}
	}
	if parRequest == nil || parRequest.form.Get("login_hint") != owner.String() ||
		parRequest.form.Get("prompt") != "" {
		t.Fatalf("handle-first PAR = %+v, want owner login_hint and no registration prompt", parRequest)
	}
	var state string
	if err := pool.QueryRow(
		context.Background(), `SELECT state FROM oauth_auth_requests`,
	).Scan(&state); err != nil {
		t.Fatal(err)
	}

	var callbackResult auth.OAuthCallbackResult
	err = flow.CompleteCallback(
		context.Background(),
		url.Values{
			"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"real-flow-code"},
		},
		func(callbackContext context.Context, result auth.OAuthCallbackResult) error {
			callbackResult = result
			stored, err := store.ResumePendingOnboardingSession(
				callbackContext, result.Attempt,
			)
			if err != nil {
				return err
			}
			pending, err := clients.newPendingPDSClient(
				callbackContext, oauthApp.Config, stored.Data,
			)
			if err != nil {
				return err
			}
			var profile map[string]any
			_, err = pending.GetRecord(
				callbackContext, owner, "app.bsky.actor.profile", "self", &profile,
			)
			return err
		},
	)
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if callbackResult.Session.AccountDID != owner ||
		callbackResult.Session.HostURL != realFlowPDSOrigin {
		t.Fatalf("callback session = %+v", callbackResult.Session)
	}

	current, err := owners.Get(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: current.Generation,
		To: ownerlifecycle.StateActive, Reason: "real flow handoff confirmed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET lifecycle_state='active',owner_generation=$3,updated_at=now()
		WHERE account_did=$1 AND session_id=$2
	`, owner, state, active.Generation); err != nil {
		t.Fatal(err)
	}
	coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: oauthApp, Store: store, Owners: owners, AuthorityVerifier: authorityVerifier,
		OperationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = coordinator.WithActiveSession(
		context.Background(), owner, state,
		func(operationContext context.Context, session *oauth.ClientSession) error {
			pds, err := clients.newPDSClient(operationContext, session, nil)
			if err != nil {
				return err
			}
			var profile map[string]any
			cid, err := pds.GetRecord(
				operationContext, owner, "app.bsky.actor.profile", "self", &profile,
			)
			if err != nil {
				return err
			}
			if cid != "bafyrealfederatedcid" || profile["displayName"] != "Real Flow Alice" {
				return errors.New("unexpected real-flow PDS read")
			}
			if err := pds.PutRecord(
				operationContext, owner, "social.craftsky.actor.profile", "self",
				map[string]any{"crafts": []string{"knitting"}},
			); err != nil {
				return err
			}
			blob, err := pds.UploadBlob(operationContext, "image/png", []byte{1, 2, 3, 4})
			if err != nil {
				return err
			}
			if blob.CID != "bafyrealfederatedblob" || blob.MIME != "image/png" || blob.Size != 4 {
				return errors.New("unexpected real-flow upload response")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("stored-session PDS operation: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `
		ALTER TABLE craftsky_profiles
			ADD COLUMN crafts TEXT[] NOT NULL DEFAULT '{}',
			ADD COLUMN record_cid TEXT,
			ADD COLUMN indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE bluesky_profiles (
			did TEXT PRIMARY KEY,display_name TEXT,description TEXT,pronouns TEXT,
			avatar_cid TEXT,avatar_mime TEXT,banner_cid TEXT,banner_mime TEXT,
			record_cid TEXT NOT NULL,indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_profiles(did,record_cid)
		VALUES($1,'craftsky-member-cid')
	`, owner); err != nil {
		t.Fatal(err)
	}
	anonymous, err := auth.NewAnonymousPDSClient(clients.directory, clients.pdsJSON, clients.boundary)
	if err != nil {
		t.Fatal(err)
	}
	backfiller := index.NewBlueskyBackfiller(anonymous, index.NewBlueskyProfile(pool))
	if err := backfiller.Backfill(context.Background(), owner); err != nil {
		t.Fatalf("anonymous Bluesky backfill: %v", err)
	}
	var displayName string
	if err := pool.QueryRow(context.Background(), `
		SELECT display_name FROM bluesky_profiles WHERE did=$1
	`, owner).Scan(&displayName); err != nil {
		t.Fatal(err)
	}
	if displayName != "Real Flow Alice" {
		t.Fatalf("backfilled display name = %q", displayName)
	}

	revoker, err := auth.NewIndigoOAuthCredentialRevoker(oauthApp, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := revoker.RevokeSession(context.Background(), callbackResult.Session); err != nil {
		t.Fatalf("revoke callback session: %v", err)
	}

	clients.boundary.CloseIdleConnections()
	upstream.close(t)
	requests, accepted := upstream.observations()
	if accepted == 0 || accepted > 4 {
		t.Fatalf("accepted TLS connections = %d, want bounded reuse", accepted)
	}
	if len(dialer.calls()) != accepted {
		t.Fatalf("base dials = %v, accepted connections = %d", dialer.calls(), accepted)
	}
	for _, address := range dialer.calls() {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			t.Fatalf("validated dial address = %q: %v", address, err)
		}
		ip, err := netip.ParseAddr(host)
		if err != nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || port != "443" {
			t.Fatalf("validated dial address = %q, want mapped public IP on 443", address)
		}
	}
	wantOperations := map[string]int{
		"/.well-known/oauth-protected-resource":   3,
		"/.well-known/oauth-authorization-server": 3,
		"/oauth/par":                        1,
		"/oauth/token:authorization_code":   1,
		"/oauth/revoke":                     2,
		"/xrpc/com.atproto.repo.getRecord":  3,
		"/xrpc/com.atproto.repo.putRecord":  1,
		"/xrpc/com.atproto.repo.uploadBlob": 1,
	}
	gotOperations := make(map[string]int)
	for _, request := range requests {
		gotOperations[request.operation]++
		if strings.Contains(request.host, "127.0.0.1") || request.method == "" || request.path == "" {
			t.Fatalf("invalid real-flow request observation = %+v", request)
		}
	}
	for operation, count := range wantOperations {
		if gotOperations[operation] != count {
			t.Fatalf("operation %s count = %d, want %d; all=%v", operation, gotOperations[operation], count, gotOperations)
		}
	}
	wantPurposeOperations := map[federatedhttp.Purpose]map[string]int{
		federatedhttp.PurposeOAuthMetadata: {
			"/.well-known/oauth-protected-resource":   3,
			"/.well-known/oauth-authorization-server": 3,
		},
		federatedhttp.PurposeOAuthRequest: {
			"/oauth/par": 1, "/oauth/token": 1, "/oauth/revoke": 2,
		},
		federatedhttp.PurposePDSJSON: {
			"/xrpc/com.atproto.repo.getRecord": 3,
			"/xrpc/com.atproto.repo.putRecord": 1,
		},
		federatedhttp.PurposePDSUpload: {
			"/xrpc/com.atproto.repo.uploadBlob": 1,
		},
	}
	wantRequestCount := 0
	for purpose, operations := range wantPurposeOperations {
		for operation, count := range operations {
			wantRequestCount += count
			if got := observer.count(purpose, operation); got != count {
				t.Fatalf("%s client %s count = %d, want %d", purpose, operation, got, count)
			}
		}
	}
	if len(requests) != wantRequestCount {
		t.Fatalf("listener request count = %d, want %d", len(requests), wantRequestCount)
	}
}

func TestOAuthAuthorityMetadataCacheLeavesOAuthFlowsFresh(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:metadatafreshflows")
	handle := syntax.Handle("metadata-fresh.real-flow.test")
	upstream := newRealFlowServer(t, owner)
	clients, _, observer := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})

	var didLookups atomic.Int32
	directory := realFlowDirectory{
		identity: &identity.Identity{
			DID: owner, Handle: handle,
			Services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
			},
		},
		beforeDIDLookup: func() { didLookups.Add(1) },
	}
	owners := newRealFlowOwnerStore(t, pool)
	storeConfig := realFlowStoreConfig()
	storeConfig.OwnerLifecycles = owners
	storeConfig.EndpointValidator = clients.boundary
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:        auth.ClientModeLocalhost,
		CallbackURL: realFlowURL(t, "http://127.0.0.1:18080/oauth/callback"),
		Scopes:      []string{"atproto", "transition:generic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	oauthApp := oauth.NewClientApp(&artifacts.Config, store)
	oauthApp.Client = clients.oauth
	oauthApp.Resolver.Client = clients.metadata
	oauthApp.Dir = directory
	fresh, operations, err := newOAuthAuthorityVerifiers(
		directory, oauthApp.Resolver, 5*time.Minute, 100, clients.metadata.Timeout, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, err := operations.ResolveCurrent(context.Background(), owner); err != nil {
			t.Fatal(err)
		}
	}
	if got := didLookups.Load(); got != 2 {
		t.Fatalf("warm operation DID lookups = %d, want 2", got)
	}
	if protected := observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-protected-resource"); protected != 1 {
		t.Fatalf("warm operation protected-resource requests = %d, want 1", protected)
	}
	if authorization := observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-authorization-server"); authorization != 1 {
		t.Fatalf("warm operation authorization-server requests = %d, want 1", authorization)
	}

	registrationOAuth, err := auth.NewRegistrationOAuthAdapter(oauthApp)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout:      5 * time.Second,
		CallbackOperationTimeout:   5 * time.Second,
		RegistrationProviderOrigin: realFlowPDSOrigin,
		RegistrationOAuth:          registrationOAuth,
		AuthorityVerifier:          fresh,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.StartLogin(
		context.Background(), handle, auth.HandoffVerifiedLink, "", "metadata-login-device",
	); err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	var loginState string
	if err := pool.QueryRow(
		context.Background(), `SELECT state FROM oauth_auth_requests WHERE purpose='login'`,
	).Scan(&loginState); err != nil {
		t.Fatal(err)
	}
	if err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {loginState}, "iss": {realFlowAuthOrigin}, "code": {"metadata-login-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error { return nil }); err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	upstream.setRegistrationPAR(
		nil,
		http.StatusCreated,
		`{"request_uri":"urn:ietf:params:oauth:request_uri:metadata-registration","expires_in":60}`,
	)
	if _, err := flow.StartRegistration(
		context.Background(), auth.HandoffVerifiedLink, "", "metadata-registration-device",
	); err != nil {
		t.Fatalf("StartRegistration: %v", err)
	}

	if protected := observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-protected-resource"); protected != 4 {
		t.Fatalf("operation/start/callback/registration protected-resource requests = %d, want 4", protected)
	}
	if authorization := observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-authorization-server"); authorization != 4 {
		t.Fatalf("operation/start/callback/registration authorization-server requests = %d, want 4", authorization)
	}
}

// IT-001 / IT-004 / IT-005: the concrete uncached verifier protects the real
// coordinator-to-PDS effect boundary and leaves cleanup bound to its issuer.
func TestAuthoritativeOAuthVerifierProtectsFederatedEffect(t *testing.T) {
	const (
		accessCanary  = "phase-five-old-access-canary"
		refreshCanary = "phase-five-old-refresh-canary"
	)

	type fixture struct {
		pool        *pgxpool.Pool
		owner       syntax.DID
		sessionID   string
		clients     *federatedClients
		upstream    *realFlowServer
		coordinator *auth.OAuthSessionCoordinator
		lookups     *atomic.Int64
	}
	newFixture := func(
		t *testing.T,
		currentPDS string,
		currentIssuer string,
		lookup func(context.Context, syntax.DID) (*identity.Identity, error),
		operationTimeout time.Duration,
	) fixture {
		t.Helper()
		pool := withRealFlowAuthSchema(t)
		owner := syntax.DID("did:plc:phasefiveauthority")
		sessionID := "phase-five-parent"
		upstream := newRealFlowServer(t, owner)
		upstream.setProtectedIssuer(currentIssuer)
		clients, _, _ := newRealFlowClients(t, upstream)
		t.Cleanup(func() {
			clients.boundary.CloseIdleConnections()
			upstream.close(t)
		})
		lookups := &atomic.Int64{}
		clients.authoritativeDirectory = realFlowDirectory{lookupDID: func(ctx context.Context, did syntax.DID) (*identity.Identity, error) {
			lookups.Add(1)
			if lookup != nil {
				return lookup(ctx, did)
			}
			return &identity.Identity{
				DID: did, Handle: syntax.Handle("phase-five.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: currentPDS},
				},
			}, nil
		}}

		owners := newRealFlowOwnerStore(t, pool)
		storeConfig := realFlowStoreConfig()
		storeConfig.OwnerLifecycles = owners
		storeConfig.EndpointValidator = clients.boundary
		store := auth.NewPostgresAuthStore(pool, storeConfig)
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO craftsky_profiles(did) VALUES($1)
		`, owner); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,
				transitioned_at,created_at,updated_at
			) VALUES($1,'active',1,1,'phase five fixture',now(),now(),now())
		`, owner); err != nil {
			t.Fatal(err)
		}
		privateKey, err := atcrypto.GeneratePrivateKeyP256()
		if err != nil {
			t.Fatal(err)
		}
		data := oauth.ClientSessionData{
			AccountDID: owner, SessionID: sessionID,
			HostURL: realFlowPDSOrigin, AuthServerURL: realFlowAuthOrigin,
			AuthServerTokenEndpoint:      realFlowAuthOrigin + "/oauth/token",
			AuthServerRevocationEndpoint: realFlowAuthOrigin + "/oauth/revoke",
			Scopes:                       []string{"atproto"}, AccessToken: accessCanary, RefreshToken: refreshCanary,
			DPoPPrivateKeyMultibase: privateKey.Multibase(),
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO oauth_sessions(
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				row_version,absolute_expires_at,created_at,updated_at
			) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day',now(),now())
		`, owner, sessionID, data); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO craftsky_sessions(
				token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
				last_seen_at,idle_expires_at
			) VALUES(convert_to('phase-five-child','UTF8'),$1,$2,'active',1,now(),now()+interval '1 day')
		`, owner, sessionID); err != nil {
			t.Fatal(err)
		}
		config := oauth.NewPublicConfig(
			"https://appview.example/oauth/client-metadata.json",
			"https://appview.example/oauth/callback",
			[]string{"atproto"},
		)
		oauthApp := oauth.NewClientApp(&config, store)
		oauthApp.Client = clients.oauth
		oauthApp.Resolver.Client = clients.metadata
		oauthApp.Dir = clients.authoritativeDirectory
		_, verifier, err := newOAuthAuthorityVerifiers(
			clients.authoritativeDirectory,
			oauthApp.Resolver,
			5*time.Minute,
			100,
			clients.metadata.Timeout,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
			App: oauthApp, Store: store, Owners: owners, AuthorityVerifier: verifier,
			OperationTimeout: operationTimeout,
		})
		if err != nil {
			t.Fatal(err)
		}
		return fixture{
			pool: pool, owner: owner, sessionID: sessionID, clients: clients,
			upstream: upstream, coordinator: coordinator, lookups: lookups,
		}
	}
	runEffect := func(f fixture) (bool, error) {
		effectCalled := false
		err := f.coordinator.WithActiveSession(
			context.Background(), f.owner, f.sessionID,
			func(ctx context.Context, session *oauth.ClientSession) error {
				effectCalled = true
				pds, err := f.clients.newPDSClient(ctx, session, nil)
				if err != nil {
					return err
				}
				return pds.PutRecord(
					ctx, f.owner, "social.craftsky.actor.profile", "self",
					map[string]any{"crafts": []string{"knitting"}},
				)
			},
		)
		return effectCalled, err
	}
	assertActiveState := func(t *testing.T, f fixture) {
		t.Helper()
		var ownerState, parentState, childState string
		var profiles int
		if err := f.pool.QueryRow(context.Background(), `
			SELECT owner.state,parent.lifecycle_state,child.lifecycle_state,
			       (SELECT count(*) FROM craftsky_profiles WHERE did=$1)
			FROM owner_lifecycles owner
			JOIN oauth_sessions parent ON parent.account_did=owner.owner_did
			JOIN craftsky_sessions child
			  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
			WHERE owner.owner_did=$1 AND parent.session_id=$2
		`, f.owner, f.sessionID).Scan(&ownerState, &parentState, &childState, &profiles); err != nil {
			t.Fatal(err)
		}
		if ownerState != "active" || parentState != "active" || childState != "active" || profiles != 1 {
			t.Fatalf("preserved owner/parent/child/profile = %s/%s/%s/%d", ownerState, parentState, childState, profiles)
		}
	}
	containsCanary := func(request realFlowRequest) bool {
		wire := request.authorization + " " + request.dpop + " " + request.form.Encode()
		return strings.Contains(wire, accessCanary) || strings.Contains(wire, refreshCanary)
	}

	t.Run("matching authority keeps DID fresh and reuses validated metadata", func(t *testing.T) {
		f := newFixture(t, realFlowPDSOrigin, realFlowAuthOrigin, nil, 5*time.Second)
		for range 2 {
			called, err := runEffect(f)
			if err != nil || !called {
				t.Fatalf("matching effect called=%t err=%v", called, err)
			}
		}
		if f.lookups.Load() != 2 {
			t.Fatalf("uncached DID lookups=%d, want 2", f.lookups.Load())
		}
		requests, _ := f.upstream.observations()
		writes, protectedMetadata, authorizationMetadata := 0, 0, 0
		for _, request := range requests {
			if request.path == "/.well-known/oauth-protected-resource" {
				protectedMetadata++
			}
			if request.path == "/.well-known/oauth-authorization-server" {
				authorizationMetadata++
			}
			if request.path == "/xrpc/com.atproto.repo.putRecord" {
				writes++
				if request.host != "pds.real-flow.test" || !strings.Contains(request.authorization, accessCanary) {
					t.Fatalf("protected write destination/authorization = %s/%q", request.host, request.authorization)
				}
			}
		}
		if writes != 2 || protectedMetadata != 1 || authorizationMetadata != 1 {
			t.Fatalf(
				"writes/protected metadata/authorization metadata=%d/%d/%d, want 2/1/1; requests=%+v",
				writes, protectedMetadata, authorizationMetadata, requests,
			)
		}
		assertActiveState(t, f)
	})

	t.Run("mismatching authority fences before effect and cleans up only at original issuer", func(t *testing.T) {
		f := newFixture(t, realFlowSecondPDSOrigin, realFlowSecondAuthOrigin, nil, 5*time.Second)
		called, err := runEffect(f)
		if called || !errors.Is(err, auth.ErrPDSSessionExpired) {
			t.Fatalf("stale effect called=%t err=%v", called, err)
		}
		var ownerState, parentState, childState string
		if err := f.pool.QueryRow(context.Background(), `
			SELECT owner.state,parent.lifecycle_state,child.lifecycle_state
			FROM owner_lifecycles owner
			JOIN oauth_sessions parent ON parent.account_did=owner.owner_did
			JOIN craftsky_sessions child
			  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
			WHERE owner.owner_did=$1 AND parent.session_id=$2
		`, f.owner, f.sessionID).Scan(&ownerState, &parentState, &childState); err != nil {
			t.Fatal(err)
		}
		if ownerState != "active" || parentState != "revocation_pending" || childState != "revoked" {
			t.Fatalf("stale owner/parent/child = %s/%s/%s", ownerState, parentState, childState)
		}

		storeConfig := realFlowStoreConfig()
		storeConfig.EndpointValidator = f.clients.boundary
		store := auth.NewPostgresAuthStore(f.pool, storeConfig)
		config := oauth.NewPublicConfig(
			"https://appview.example/oauth/client-metadata.json",
			"https://appview.example/oauth/callback",
			[]string{"atproto"},
		)
		oauthApp := &oauth.ClientApp{Client: f.clients.oauth, Config: &config}
		revoker, err := auth.NewIndigoOAuthCredentialRevoker(oauthApp, store)
		if err != nil {
			t.Fatal(err)
		}
		processor, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
			Pool: f.pool, Revoker: revoker, BatchSize: 1, LeaseDuration: time.Minute,
			OperationTimeout: 5 * time.Second, MaxAttempts: 2,
			BaseBackoff: time.Second, MaxBackoff: time.Minute, MaxCredentialRetention: time.Hour,
		})
		if err != nil {
			t.Fatal(err)
		}
		if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
			t.Fatalf("original-issuer cleanup processed=%d err=%v", processed, err)
		}
		requests, _ := f.upstream.observations()
		revocations := 0
		for _, request := range requests {
			if request.host == "pds-second.real-flow.test" || request.host == "auth-second.real-flow.test" {
				if request.authorization != "" || request.dpop != "" || containsCanary(request) {
					t.Fatalf("old credential or DPoP proof reached current authority: %+v", request)
				}
			}
			if request.path == "/oauth/revoke" {
				revocations++
				if request.host != "auth.real-flow.test" || !containsCanary(request) {
					t.Fatalf("revocation destination/credential = %+v", request)
				}
			}
		}
		if revocations != 2 {
			t.Fatalf("revocations=%d, want access and refresh at original issuer; requests=%+v", revocations, requests)
		}
	})

	for _, test := range []struct {
		name          string
		currentPDS    string
		lookup        func(context.Context, syntax.DID) (*identity.Identity, error)
		configure     func(*realFlowServer)
		timeout       time.Duration
		wantNoRequest bool
	}{
		{
			name: "DID timeout", timeout: 25 * time.Millisecond, wantNoRequest: true,
			lookup: func(ctx context.Context, _ syntax.DID) (*identity.Identity, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
		{
			name: "DID DNS failure", timeout: time.Second, wantNoRequest: true,
			lookup: func(context.Context, syntax.DID) (*identity.Identity, error) {
				return nil, &net.DNSError{Err: "no such host", Name: "plc.directory.test"}
			},
		},
		{
			name: "OAuth metadata outage", currentPDS: realFlowPDSOrigin, timeout: time.Second,
			configure: func(server *realFlowServer) {
				server.mu.Lock()
				server.protectedStatus = http.StatusServiceUnavailable
				server.mu.Unlock()
			},
		},
		{
			name: "outbound policy rejection", currentPDS: "https://127.0.0.1", timeout: time.Second,
			wantNoRequest: true,
		},
	} {
		t.Run(test.name+" is retryable and preserves lifecycle", func(t *testing.T) {
			currentPDS := test.currentPDS
			if currentPDS == "" {
				currentPDS = realFlowPDSOrigin
			}
			f := newFixture(t, currentPDS, realFlowAuthOrigin, test.lookup, test.timeout)
			if test.configure != nil {
				test.configure(f.upstream)
			}
			called, err := runEffect(f)
			if err == nil || errors.Is(err, auth.ErrPDSSessionExpired) || called {
				t.Fatalf("retryable effect called=%t err=%v", called, err)
			}
			assertActiveState(t, f)
			requests, _ := f.upstream.observations()
			if test.wantNoRequest && len(requests) != 0 {
				t.Fatalf("forbidden transient destination received requests: %+v", requests)
			}
			for _, request := range requests {
				if containsCanary(request) {
					t.Fatalf("retryable authority check sent credentials: %+v", request)
				}
			}
		})
	}
}
