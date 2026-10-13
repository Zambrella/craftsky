package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestOAuthCallbackRejectsMissingRequiredScopesBeforePersistence(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:callback-migration")
	seedAuthOwner(t, pool, owner)

	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	const (
		state        = "callback-migration-state"
		resourceA    = "https://pds-a.example"
		issuerA      = "https://issuer-a.example"
		accessToken  = "old-access-token"
		refreshToken = "old-refresh-token"
	)
	requestInfo := oauth.AuthRequestData{
		State: state, AccountDID: &owner, Scopes: []string{"atproto"},
		RequestURI: "urn:request:callback-migration", AuthServerURL: issuerA,
		AuthServerTokenEndpoint:      issuerA + "/oauth/token",
		AuthServerRevocationEndpoint: issuerA + "/oauth/revoke",
		PKCEVerifier:                 "pkce-verifier", DPoPPrivateKeyMultibase: privateKey.Multibase(),
	}
	err = owners.WithOnboardingAuth(context.Background(), owner, func(authContext context.Context, authority ownerlifecycle.Lifecycle) error {
		requestContext := auth.WithLoginAuthRequest(
			authContext, owner, authority.Generation, authority.AuthEpoch, resourceA, issuerA,
			auth.HandoffVerifiedLink, "callback-device", "",
		)
		return store.SaveAuthRequestInfo(requestContext, requestInfo)
	})
	if err != nil {
		t.Fatal(err)
	}

	var requests []string
	client := &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		switch request.URL.String() {
		case issuerA + "/oauth/token":
		case issuerA + "/oauth/revoke":
			return &http.Response{
				StatusCode: http.StatusOK, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader("")), Request: request,
			}, nil
		default:
			return nil, errors.New("unexpected outbound request")
		}
		body, err := json.Marshal(oauth.TokenResponse{
			Subject: owner.String(), Scope: "atproto",
			AccessToken: accessToken, RefreshToken: refreshToken,
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(string(body))), Request: request,
		}, nil
	})}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth-client-metadata.json",
		"https://appview.example/oauth/callback",
		auth.DefaultScopes(),
	)
	current := &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
		DID: owner, PDSOrigin: resourceA, IssuerOrigin: issuerA,
	}}
	app := &oauth.ClientApp{
		Client: client, Config: &config, Resolver: oauth.NewResolver(),
		Dir: &callbackDirectory{identity: &identity.Identity{DID: owner, Handle: "migrated.example"}},
	}
	observer := &recordingAuthorityObserver{}
	service, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App:   app,
		Store: store, Owners: owners, AuthorityVerifier: current,
		Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	finalized := false
	err = service.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {issuerA}, "code": {"authorization-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		finalized = true
		return nil
	})
	if !errors.Is(err, auth.ErrOAuthFlowInvalid) {
		t.Fatalf("callback error = %v, want invalid scope grant", err)
	}
	var diagnostic *auth.OAuthCallbackDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Reason() != "scope_required_missing" {
		t.Fatalf("missing scope rejection diagnostic: %T", err)
	}
	verifyCallbackDiagnosticSink(t, err)
	if observer.calls != 0 {
		t.Fatal("scope refusal should precede authority verification")
	}

	if finalized {
		t.Fatal("mixed-authority callback reached finalization")
	}
	if len(requests) != 1 || requests[0] != issuerA+"/oauth/token" {
		t.Fatalf("outbound requests = %v, want token exchange only with original issuer", requests)
	}

	var parentCount, childCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM oauth_sessions WHERE account_did=$1 AND session_id=$2),
			(SELECT count(*) FROM craftsky_sessions WHERE account_did=$1 AND oauth_session_id=$2)
	`, owner, state).Scan(&parentCount, &childCount); err != nil {
		t.Fatal(err)
	}
	if parentCount != 0 || childCount != 0 {
		t.Fatalf("persisted parent/child = %d/%d, want 0/0", parentCount, childCount)
	}

	var cleanupData []byte
	var cleanupStatus, requestState string
	if err := pool.QueryRow(context.Background(), `
		SELECT credential.data,credential.status,request.request_state
		FROM oauth_unverified_credentials credential
		JOIN oauth_auth_requests request ON request.state=credential.request_state
		WHERE credential.request_state=$1
	`, state).Scan(&cleanupData, &cleanupStatus, &requestState); err != nil {
		t.Fatal(err)
	}
	var cleanup oauth.ClientSessionData
	if err := json.Unmarshal(cleanupData, &cleanup); err != nil {
		t.Fatal(err)
	}
	if cleanupStatus != "pending" || requestState != string(auth.AuthRequestCleanupPending) {
		t.Fatalf("cleanup/request state = %q/%q, want pending/cleanup_pending", cleanupStatus, requestState)
	}
	if cleanup.HostURL != resourceA || cleanup.AuthServerURL != issuerA ||
		cleanup.AuthServerTokenEndpoint != issuerA+"/oauth/token" ||
		cleanup.AuthServerRevocationEndpoint != issuerA+"/oauth/revoke" {
		t.Fatalf("cleanup authority = host %q issuer %q token %q revoke %q, want original authority A",
			cleanup.HostURL, cleanup.AuthServerURL, cleanup.AuthServerTokenEndpoint,
			cleanup.AuthServerRevocationEndpoint)
	}

	revoker, err := auth.NewIndigoOAuthCredentialRevoker(app, store)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
		Pool: pool, Revoker: revoker, BatchSize: 10,
		LeaseDuration: time.Minute, OperationTimeout: 5 * time.Second,
		MaxAttempts: 3, BaseBackoff: time.Second, MaxBackoff: time.Minute,
		MaxCredentialRetention: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := processor.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("cleanup processed = %d, want 1", processed)
	}
	if len(requests) != 3 || requests[1] != issuerA+"/oauth/revoke" || requests[2] != issuerA+"/oauth/revoke" {
		t.Fatalf("outbound requests after cleanup = %v, want token and two revocations only at original issuer", requests)
	}

}

const resolvedCraftskyScopeFixture = "atproto repo?collection=social.craftsky.actor.profile&collection=social.craftsky.feed.post&collection=social.craftsky.feed.like&collection=social.craftsky.feed.repost&collection=social.craftsky.business.profile&collection=social.craftsky.business.event repo:app.bsky.actor.profile?action=create&action=update repo:app.bsky.graph.follow repo:app.bsky.graph.block blob?accept=image/jpeg&accept=image/png&accept=image/webp&accept=video/mp4&accept=text/vtt rpc:com.atproto.repo.uploadBlob?aud=* rpc:app.bsky.video.getUploadLimits?aud=*"

func TestOAuthCallbackAcceptsProviderResolvedPermissionSet(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, storeConfig)
	owner := syntax.DID("did:plc:callback-migration")
	seedAuthOwner(t, pool, owner)

	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	const (
		state        = "callback-migration-state"
		resourceA    = "https://pds-a.example"
		issuerA      = "https://issuer-a.example"
		accessToken  = "old-access-token"
		refreshToken = "old-refresh-token"
	)
	requestInfo := oauth.AuthRequestData{
		State: state, AccountDID: &owner, Scopes: []string{"atproto"},
		RequestURI: "urn:request:callback-migration", AuthServerURL: issuerA,
		AuthServerTokenEndpoint:      issuerA + "/oauth/token",
		AuthServerRevocationEndpoint: issuerA + "/oauth/revoke",
		PKCEVerifier:                 "pkce-verifier", DPoPPrivateKeyMultibase: privateKey.Multibase(),
	}
	err = owners.WithOnboardingAuth(context.Background(), owner, func(authContext context.Context, authority ownerlifecycle.Lifecycle) error {
		requestContext := auth.WithLoginAuthRequest(
			authContext, owner, authority.Generation, authority.AuthEpoch, resourceA, issuerA,
			auth.HandoffVerifiedLink, "callback-device", "",
		)
		return store.SaveAuthRequestInfo(requestContext, requestInfo)
	})
	if err != nil {
		t.Fatal(err)
	}

	var requests []string
	client := &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.String())
		switch request.URL.String() {
		case issuerA + "/oauth/token":
		case issuerA + "/oauth/revoke":
			return &http.Response{
				StatusCode: http.StatusOK, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader("")), Request: request,
			}, nil
		default:
			return nil, errors.New("unexpected outbound request")
		}
		body, err := json.Marshal(oauth.TokenResponse{
			Subject: owner.String(), Scope: resolvedCraftskyScopeFixture,
			AccessToken: accessToken, RefreshToken: refreshToken,
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(string(body))), Request: request,
		}, nil
	})}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth-client-metadata.json",
		"https://appview.example/oauth/callback",
		auth.DefaultScopes(),
	)
	current := &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{
		DID: owner, PDSOrigin: resourceA, IssuerOrigin: issuerA,
	}}
	app := &oauth.ClientApp{
		Client: client, Config: &config, Resolver: oauth.NewResolver(),
		Dir: &callbackDirectory{identity: &identity.Identity{DID: owner, Handle: "migrated.example"}},
	}
	observer := &recordingAuthorityObserver{}
	service, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App:   app,
		Store: store, Owners: owners, AuthorityVerifier: current,
		Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	finalized := false
	err = service.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {issuerA}, "code": {"authorization-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		finalized = true
		return nil
	})
	if err != nil || !finalized {
		t.Fatalf("resolved callback failed: finalized=%t error=%v", finalized, err)
	}
	if current.calls.Load() != 1 || observer.result != "success" {
		t.Fatal("resolved callback did not verify authority")
	}
	var scopes []byte
	if err := pool.QueryRow(context.Background(), `SELECT data->'scopes' FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, state).Scan(&scopes); err != nil {
		t.Fatal(err)
	}
	var stored []string
	if err := json.Unmarshal(scopes, &stored); err != nil {
		t.Fatal(err)
	}
	if strings.Join(stored, " ") != resolvedCraftskyScopeFixture {
		t.Fatal("resolved grants were not persisted for subsequent PDS operations")
	}
}
