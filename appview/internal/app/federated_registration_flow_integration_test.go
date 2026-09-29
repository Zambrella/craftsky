package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testlog"
)

func newRealRegistrationFlow(
	t *testing.T,
	pool *pgxpool.Pool,
	clients *federatedClients,
	startTimeout time.Duration,
) (*auth.OAuthFlowService, *auth.PostgresAuthStore) {
	return newRealRegistrationFlowWithTimeouts(t, pool, clients, startTimeout, 5*time.Second)
}

func newRealRegistrationFlowWithTimeouts(
	t *testing.T,
	pool *pgxpool.Pool,
	clients *federatedClients,
	startTimeout time.Duration,
	callbackTimeout time.Duration,
) (*auth.OAuthFlowService, *auth.PostgresAuthStore) {
	return newRealRegistrationFlowForProvider(
		t, pool, clients, realFlowPDSOrigin, startTimeout, callbackTimeout,
	)
}

func newRealRegistrationFlowForProvider(
	t *testing.T,
	pool *pgxpool.Pool,
	clients *federatedClients,
	providerOrigin string,
	startTimeout time.Duration,
	callbackTimeout time.Duration,
) (*auth.OAuthFlowService, *auth.PostgresAuthStore) {
	t.Helper()
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
	registrationOAuth, err := auth.NewRegistrationOAuthAdapter(oauthApp)
	if err != nil {
		t.Fatal(err)
	}
	authorityVerifier, err := newAuthoritativeOAuthVerifier(oauthApp.Dir, oauthApp.Resolver)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: startTimeout, CallbackOperationTimeout: callbackTimeout,
		RegistrationProviderOrigin: providerOrigin, RegistrationOAuth: registrationOAuth,
		AuthorityVerifier: authorityVerifier,
	})
	if err != nil {
		t.Fatal(err)
	}
	return flow, store
}

func TestProviderRegistrationStartFailureClassificationDoesNotFallback(t *testing.T) {
	const providerText = "provider-secret: retry without prompt=create"
	tests := []struct {
		name             string
		prompted         bool
		parStatus        int
		parError         string
		invalidMetadata  bool
		invalidEndpoint  bool
		timeout          bool
		nonceReplay      bool
		wantCode         auth.RegistrationOAuthFailureCode
		wantHTTPAttempts int
	}{
		{name: "advertised prompt provider text ignored", prompted: true, parStatus: 400, parError: "invalid_request", wantCode: auth.RegistrationOAuthProviderUnavailable, wantHTTPAttempts: 1},
		{name: "absent prompt provider text ignored", parStatus: 400, parError: "invalid_request", wantCode: auth.RegistrationOAuthIncomplete, wantHTTPAttempts: 1},
		{name: "invalid client", parStatus: 400, parError: "invalid_client", wantCode: auth.RegistrationOAuthIncomplete, wantHTTPAttempts: 1},
		{name: "invalid DPoP", parStatus: 400, parError: "invalid_dpop_proof", wantCode: auth.RegistrationOAuthIncomplete, wantHTTPAttempts: 1},
		{name: "invalid scope", parStatus: 400, parError: "invalid_scope", wantCode: auth.RegistrationOAuthIncomplete, wantHTTPAttempts: 1},
		{name: "invalid metadata", invalidMetadata: true, wantCode: auth.RegistrationOAuthIncomplete},
		{name: "invalid endpoint", invalidEndpoint: true, wantCode: auth.RegistrationOAuthIncomplete},
		{name: "timeout", timeout: true, wantCode: auth.RegistrationOAuthProviderUnavailable, wantHTTPAttempts: 1},
		{name: "rate limited", parStatus: 429, parError: "slow_down", wantCode: auth.RegistrationOAuthProviderUnavailable, wantHTTPAttempts: 1},
		{name: "server failure", parStatus: 503, parError: "server_error", wantCode: auth.RegistrationOAuthProviderUnavailable, wantHTTPAttempts: 1},
		{name: "same-body DPoP nonce replay", nonceReplay: true, wantHTTPAttempts: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			upstream := newRealFlowServer(t, syntax.DID("did:plc:registrationfailure"))
			promptValues := []string(nil)
			if test.prompted {
				promptValues = []string{"create"}
			}
			body := `{"request_uri":"urn:ietf:params:oauth:request_uri:classified","expires_in":60}`
			status := test.parStatus
			if status == 0 {
				status = http.StatusCreated
			}
			if test.parError != "" {
				body = `{"error":"` + test.parError + `","error_description":"` + providerText + `"}`
			}
			upstream.setRegistrationPAR(promptValues, status, body)
			if test.invalidMetadata {
				upstream.setRegistrationMetadataScopes([]string{"transition:generic"})
			}
			if test.invalidEndpoint {
				endpoints := defaultRealFlowOAuthEndpoints()
				endpoints.authorization = "http://auth.real-flow.test/oauth/authorize"
				upstream.setOAuthEndpoints(endpoints)
			}
			if test.nonceReplay {
				upstream.enableRegistrationDPoPNonceReplay()
			}
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			logicalPARAttempts := 0
			if test.timeout {
				next := clients.oauth.Transport
				clients.oauth.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.URL.Path == "/oauth/par" {
						logicalPARAttempts++
						return nil, context.DeadlineExceeded
					}
					return next.RoundTrip(request)
				})
			}
			flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)

			_, startErr := flow.StartRegistration(
				context.Background(), auth.HandoffVerifiedLink, "", "failure-device",
			)
			requests, _ := upstream.observations()
			var forms []url.Values
			for _, request := range requests {
				if request.path == "/oauth/par" {
					forms = append(forms, request.form)
				}
			}
			if test.timeout {
				if logicalPARAttempts != 1 {
					t.Fatalf("logical PAR attempts = %d, want one", logicalPARAttempts)
				}
			} else if len(forms) != test.wantHTTPAttempts {
				t.Fatalf("PAR HTTP attempts = %d, want %d", len(forms), test.wantHTTPAttempts)
			}
			for i, form := range forms {
				wantPrompt := ""
				if test.prompted {
					wantPrompt = "create"
				}
				if form.Get("prompt") != wantPrompt || form.Get("login_hint") != "" {
					t.Fatalf("PAR %d prompt/login_hint = %q/%q", i+1, form.Get("prompt"), form.Get("login_hint"))
				}
				if i > 0 && form.Encode() != forms[0].Encode() {
					t.Fatal("DPoP nonce replay changed the PAR body")
				}
			}

			if test.nonceReplay {
				if startErr != nil {
					t.Fatalf("nonce replay StartRegistration: %v", startErr)
				}
				return
			}
			var failure *auth.RegistrationOAuthError
			exposedProviderError := test.parError != "" && strings.Contains(startErr.Error(), test.parError)
			if !errors.As(startErr, &failure) || failure.Code != test.wantCode || strings.Contains(startErr.Error(), providerText) || exposedProviderError {
				t.Fatalf("StartRegistration error = %v (%+v), want redacted %q", startErr, failure, test.wantCode)
			}
			var requestsCount, reservationsCount int
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_requests`).Scan(&requestsCount); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_request_reservations`).Scan(&reservationsCount); err != nil {
				t.Fatal(err)
			}
			if requestsCount != 0 || reservationsCount != 0 {
				t.Fatalf("failed start left requests/reservations = %d/%d", requestsCount, reservationsCount)
			}
		})
	}
}

func TestProviderRegistrationAdvertisedCreatePromptDoesNotDowngrade(t *testing.T) {
	const providerText = "provider-secret: retry without prompt=create"
	tests := []struct {
		name        string
		status      int
		body        string
		wantFailure bool
	}{
		{
			name:   "prompted PAR succeeds",
			status: http.StatusCreated,
			body:   `{"request_uri":"urn:ietf:params:oauth:request_uri:prompted","expires_in":60}`,
		},
		{
			name:        "prompted PAR rejection is bounded",
			status:      http.StatusBadRequest,
			body:        `{"error":"invalid_request","error_description":"` + providerText + `"}`,
			wantFailure: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			upstream := newRealFlowServer(t, syntax.DID("did:plc:registrationprompt"))
			upstream.setRegistrationPAR([]string{"create"}, test.status, test.body)
			clients, _, observer := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
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
			registrationOAuth, err := auth.NewRegistrationOAuthAdapter(oauthApp)
			if err != nil {
				t.Fatal(err)
			}
			authorityVerifier, err := newAuthoritativeOAuthVerifier(oauthApp.Dir, oauthApp.Resolver)
			if err != nil {
				t.Fatal(err)
			}
			flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
				App: oauthApp, Store: store, Owners: owners,
				StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
				RegistrationProviderOrigin: realFlowPDSOrigin, RegistrationOAuth: registrationOAuth,
				AuthorityVerifier: authorityVerifier,
			})
			if err != nil {
				t.Fatal(err)
			}

			redirect, startErr := flow.StartRegistration(
				context.Background(), auth.HandoffVerifiedLink, "", "prompt-device",
			)
			requests, _ := upstream.observations()
			var parRequests []realFlowRequest
			for _, request := range requests {
				if request.path == "/oauth/par" {
					parRequests = append(parRequests, request)
				}
			}
			if len(parRequests) != 1 || observer.count(federatedhttp.PurposeOAuthRequest, "/oauth/par") != 1 {
				t.Fatalf("PAR requests = %d, want one logical request without downgrade", len(parRequests))
			}
			if parRequests[0].form.Get("prompt") != "create" || parRequests[0].form.Get("login_hint") != "" {
				t.Fatalf("prompt/login_hint = %q/%q", parRequests[0].form.Get("prompt"), parRequests[0].form.Get("login_hint"))
			}

			if test.wantFailure {
				var failure *auth.RegistrationOAuthError
				if !errors.As(startErr, &failure) || failure.Code != auth.RegistrationOAuthProviderUnavailable ||
					!failure.Prompted || strings.Contains(startErr.Error(), providerText) {
					t.Fatalf("StartRegistration error = %v, want redacted prompted provider unavailable", startErr)
				}
				var requestsCount, reservationsCount int
				if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_requests`).Scan(&requestsCount); err != nil {
					t.Fatal(err)
				}
				if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_request_reservations`).Scan(&reservationsCount); err != nil {
					t.Fatal(err)
				}
				if requestsCount != 0 || reservationsCount != 0 {
					t.Fatalf("failed prompted PAR left requests/reservations = %d/%d", requestsCount, reservationsCount)
				}
				return
			}
			if startErr != nil {
				t.Fatalf("StartRegistration: %v", startErr)
			}
			redirectURL, err := url.Parse(redirect)
			if err != nil || redirectURL.Query().Get("request_uri") != "urn:ietf:params:oauth:request_uri:prompted" {
				t.Fatalf("authorization redirect = %q, err=%v", redirect, err)
			}
			metadata, err := store.LoadAuthRequestMetadata(context.Background(), parRequests[0].form.Get("state"))
			if err != nil || metadata.RequestState != auth.AuthRequestReady || metadata.RegistrationIssuer != realFlowAuthOrigin {
				t.Fatalf("usable prompted request metadata = %+v, err=%v", metadata, err)
			}
		})
	}
}

func TestProviderRegistrationServerFirstDiscoveryAndPAR(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationstart")
	upstream := newRealFlowServer(t, owner)
	clients, _, observer := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})

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
	registrationOAuth, err := auth.NewRegistrationOAuthAdapter(oauthApp)
	if err != nil {
		t.Fatal(err)
	}
	authorityVerifier, err := newAuthoritativeOAuthVerifier(oauthApp.Dir, oauthApp.Resolver)
	if err != nil {
		t.Fatal(err)
	}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout:      5 * time.Second,
		CallbackOperationTimeout:   5 * time.Second,
		RegistrationProviderOrigin: realFlowPDSOrigin,
		RegistrationOAuth:          registrationOAuth,
		AuthorityVerifier:          authorityVerifier,
	})
	if err != nil {
		t.Fatal(err)
	}

	redirect, err := flow.StartRegistration(
		context.Background(), auth.HandoffVerifiedLink, "", "registration-device",
	)
	if err != nil {
		t.Fatalf("StartRegistration: %v", err)
	}
	redirectURL, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if redirectURL.Scheme != "https" || redirectURL.Host != "auth.real-flow.test" ||
		redirectURL.Path != "/oauth/authorize" || redirectURL.Query().Get("client_id") != artifacts.Config.ClientID ||
		redirectURL.Query().Get("request_uri") != "urn:ietf:params:oauth:request_uri:real-flow" {
		t.Fatalf("authorization redirect = %q", redirect)
	}

	requests, _ := upstream.observations()
	var par *realFlowRequest
	for i := range requests {
		if requests[i].path == "/oauth/par" {
			par = &requests[i]
			break
		}
	}
	if par == nil {
		t.Fatal("PAR request was not sent")
	}
	if par.form.Get("client_id") != artifacts.Config.ClientID ||
		par.form.Get("scope") != "atproto transition:generic" || par.form.Get("state") == "" ||
		par.form.Get("code_challenge") == "" || par.form.Get("code_challenge_method") != "S256" ||
		par.dpop == "" {
		t.Fatal("PAR omitted required client, scope, state, PKCE, or DPoP protection")
	}
	if _, present := par.form["login_hint"]; present {
		t.Fatalf("PAR login_hint = %q, want absent", par.form.Get("login_hint"))
	}
	if par.form.Get("prompt") != "" {
		t.Fatalf("PAR prompt = %q, want absent when not advertised", par.form.Get("prompt"))
	}
	if observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-protected-resource") != 1 ||
		observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-authorization-server") != 1 ||
		observer.count(federatedhttp.PurposeOAuthRequest, "/oauth/par") != 1 {
		t.Fatal("registration did not use one server-first discovery and PAR sequence")
	}

	metadata, err := store.LoadAuthRequestMetadata(context.Background(), par.form.Get("state"))
	if err != nil {
		t.Fatalf("load persisted registration request: %v", err)
	}
	var persistedData []byte
	if err := pool.QueryRow(
		context.Background(), `SELECT data FROM oauth_auth_requests WHERE state=$1`, par.form.Get("state"),
	).Scan(&persistedData); err != nil {
		t.Fatalf("load persisted OAuth request: %v", err)
	}
	var requestInfo oauth.AuthRequestData
	if err := json.Unmarshal(persistedData, &requestInfo); err != nil {
		t.Fatalf("decode persisted OAuth request: %v", err)
	}
	if metadata.Purpose != auth.RegistrationOAuthPurpose ||
		metadata.RegistrationProviderOrigin != realFlowPDSOrigin ||
		metadata.RegistrationIssuer != realFlowAuthOrigin || metadata.Owner != "" ||
		metadata.DeviceID != "registration-device" || requestInfo.PKCEVerifier == "" ||
		requestInfo.DPoPPrivateKeyMultibase == "" {
		t.Fatalf("persisted registration request metadata = %+v", metadata)
	}
}

func TestProviderRegistrationStartsFromAuthorizationServerIssuer(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	upstream := newRealFlowServer(t, syntax.DID("did:plc:registrationissuer"))
	upstream.mu.Lock()
	upstream.protectedStatus = http.StatusNotFound
	upstream.promptValues = []string{"create"}
	upstream.mu.Unlock()
	clients, _, observer := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})

	flow, _ := newRealRegistrationFlowForProvider(
		t, pool, clients, realFlowAuthOrigin, 5*time.Second, 5*time.Second,
	)
	redirect, err := flow.StartRegistration(
		context.Background(), auth.HandoffVerifiedLink, "", "registration-issuer-device",
	)
	if err != nil {
		t.Fatalf("StartRegistration: %v", err)
	}
	redirectURL, err := url.Parse(redirect)
	if err != nil || redirectURL.Host != "auth.real-flow.test" || redirectURL.Query().Get("request_uri") == "" {
		t.Fatalf("authorization redirect = %q, err=%v", redirect, err)
	}
	if observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-protected-resource") != 1 ||
		observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-authorization-server") != 1 ||
		observer.count(federatedhttp.PurposeOAuthRequest, "/oauth/par") != 1 {
		t.Fatal("registration did not fall back from resource discovery to the configured issuer")
	}
	requests, _ := upstream.observations()
	for _, request := range requests {
		if request.path == "/oauth/par" && request.form.Get("prompt") != "create" {
			t.Fatalf("PAR prompt = %q, want create", request.form.Get("prompt"))
		}
	}
}

// IT-007: registration accepts the token subject only after its authoritative
// PDS discovers the exact issuer stored before browser authorization.
func TestProviderRegistrationCallbackProvesSeparatePDSAndIssuerAuthority(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationcallback")
	handle := syntax.Handle("registration.real-flow.test")
	upstream := newRealFlowServer(t, owner)
	clients, _, observer := newRealFlowClients(t, upstream)
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
	flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)

	if _, err := flow.StartRegistration(
		context.Background(), auth.HandoffVerifiedLink, "", "registration-callback-device",
	); err != nil {
		t.Fatalf("StartRegistration: %v", err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `
		SELECT state FROM oauth_auth_requests WHERE purpose='registration'
	`).Scan(&state); err != nil {
		t.Fatal(err)
	}

	var result auth.OAuthCallbackResult
	err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"registration-callback-code"},
	}, func(_ context.Context, callback auth.OAuthCallbackResult) error {
		result = callback
		return nil
	})
	if err != nil {
		var requestState string
		var quarantine int
		_ = pool.QueryRow(context.Background(), `SELECT request_state FROM oauth_auth_requests WHERE state=$1`, state).Scan(&requestState)
		_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1`, state).Scan(&quarantine)
		t.Fatalf("CompleteCallback: %v (request_state=%s quarantine=%d)", err, requestState, quarantine)
	}
	if result.Session.AccountDID != owner || result.Session.HostURL != realFlowPDSOrigin ||
		result.Session.AuthServerURL != realFlowAuthOrigin || result.Handle != handle ||
		result.Attempt.Purpose != auth.RegistrationOAuthPurpose {
		t.Fatalf("registration callback result = %+v", result)
	}
	metadata, err := store.LoadAuthRequestMetadata(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Owner != owner || metadata.OwnerGeneration <= 0 || metadata.AuthEpoch <= 0 ||
		metadata.RegistrationIssuer != realFlowAuthOrigin {
		t.Fatalf("bound registration metadata = %+v", metadata)
	}
	var parentState string
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, state).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if parentState != "pending_handoff" {
		t.Fatalf("registration parent state = %q, want pending_handoff", parentState)
	}
	if observer.count(federatedhttp.PurposeOAuthRequest, "/oauth/token") != 1 ||
		observer.count(federatedhttp.PurposeOAuthMetadata, "/.well-known/oauth-protected-resource") != 2 {
		t.Fatal("registration callback did not perform one token exchange and authoritative PDS discovery")
	}
}

// IR-001 / UT-011 / IT-017: a real failing registration token endpoint must
// not place provider-controlled response text in global logs, and an upstream
// response that cannot disprove issuance remains ambiguous and retryable.
func TestProviderRegistrationTokenFailureIsRedactedAmbiguousAndUnavailable(t *testing.T) {
	const providerSentinel = "provider-token-error-sentinel"

	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationtokenfailure")
	upstream := newRealFlowServer(t, owner)
	upstream.setTokenFailure(
		http.StatusServiceUnavailable,
		`{"error":"server_error","error_description":"`+providerSentinel+`"}`,
	)
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "token-failure-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"token-failure-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		return errors.New("token failure finalized callback")
	})
	var failure *auth.TrustedRegistrationFailure
	if !errors.As(err, &failure) || failure.Code != auth.RegistrationFailureProviderUnavailable {
		t.Errorf("callback failure = %T %v, want trusted providerUnavailable", err, err)
	}
	var requestState string
	if err := pool.QueryRow(context.Background(), `
		SELECT request_state FROM oauth_auth_requests WHERE state=$1
	`, state).Scan(&requestState); err != nil {
		t.Fatal(err)
	}
	if requestState != string(auth.AuthRequestExchangeAmbiguous) {
		t.Errorf("request state = %q, want exchange_ambiguous", requestState)
	}
	if strings.Contains(logs.String(), providerSentinel) {
		t.Fatalf("global logs retained provider token body: %s", logs.String())
	}
}

func TestProviderRegistrationTokenOutcomeClassificationAndDurability(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*realFlowServer, *federatedClients)
		wantCode  auth.RegistrationFailureCode
		wantState auth.AuthRequestState
		sentinel  string
	}{
		{
			name: "response lost after transmission",
			configure: func(_ *realFlowServer, clients *federatedClients) {
				next := clients.oauth.Transport
				clients.oauth.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
					response, err := next.RoundTrip(request)
					if err != nil || request.URL.Path != "/oauth/token" {
						return response, err
					}
					_ = response.Body.Close()
					return nil, io.ErrUnexpectedEOF
				})
			},
			wantCode:  auth.RegistrationFailureProviderUnavailable,
			wantState: auth.AuthRequestExchangeAmbiguous,
		},
		{
			name: "rate limited",
			configure: func(upstream *realFlowServer, _ *federatedClients) {
				upstream.setTokenFailure(http.StatusTooManyRequests, `{"error":"slow_down","error_description":"token-429-sentinel"}`)
			},
			wantCode:  auth.RegistrationFailureProviderUnavailable,
			wantState: auth.AuthRequestExchangeAmbiguous,
			sentinel:  "token-429-sentinel",
		},
		{
			name: "definite protocol rejection",
			configure: func(upstream *realFlowServer, _ *federatedClients) {
				upstream.setTokenFailure(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"token-400-sentinel"}`)
			},
			wantCode:  auth.RegistrationFailureIncomplete,
			wantState: auth.AuthRequestExchangeFailed,
			sentinel:  "token-400-sentinel",
		},
		{
			name: "malformed success",
			configure: func(upstream *realFlowServer, _ *federatedClients) {
				upstream.setTokenFailure(http.StatusOK, `{"access_token":"malformed-token-sentinel"`)
			},
			wantCode:  auth.RegistrationFailureIncomplete,
			wantState: auth.AuthRequestExchangeAmbiguous,
			sentinel:  "malformed-token-sentinel",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:uncertaintoken" + strings.ReplaceAll(test.name, " ", ""))
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "uncertain-token-device"); err != nil {
				t.Fatal(err)
			}
			test.configure(upstream, clients)
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			previousLogger := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
			t.Cleanup(func() { slog.SetDefault(previousLogger) })
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"uncertain-token-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				return errors.New("uncertain token finalized callback")
			})
			var failure *auth.TrustedRegistrationFailure
			if !errors.As(err, &failure) || failure.Code != test.wantCode {
				t.Errorf("callback failure = %T %v, want trusted %s", err, err, test.wantCode)
			}
			var requestState string
			if err := pool.QueryRow(context.Background(), `SELECT request_state FROM oauth_auth_requests WHERE state=$1`, state).Scan(&requestState); err != nil {
				t.Fatal(err)
			}
			if requestState != string(test.wantState) {
				t.Errorf("request state = %q, want %s", requestState, test.wantState)
			}
			if test.sentinel != "" && strings.Contains(logs.String(), test.sentinel) {
				t.Fatalf("global logs retained provider token body: %s", logs.String())
			}
		})
	}
}

// IR-001 / IT-008 / IT-017: recoverable credentials from a bounded, valid
// JSON success body remain cleanup-owned when Indigo rejects another field.
func TestProviderRegistrationMalformedSuccessQuarantinesRecoverableTokens(t *testing.T) {
	const (
		accessToken  = "malformed-success-access-token-sentinel"
		refreshToken = "malformed-success-refresh-token-sentinel"
		rawMarker    = "malformed-success-raw-body-sentinel"
	)

	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationmalformedsuccess")
	upstream := newRealFlowServer(t, owner)
	upstream.setTokenResponse(map[string]any{
		"sub": 17, "scope": "atproto transition:generic",
		"access_token": accessToken, "refresh_token": refreshToken,
		"raw_marker": rawMarker,
	})
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "malformed-success-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	finalized := false
	err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"malformed-success-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		finalized = true
		return nil
	})
	var failure *auth.TrustedRegistrationFailure
	if !errors.As(err, &failure) || failure.Code != auth.RegistrationFailureIncomplete || finalized {
		t.Fatalf("callback failure = %T %v finalized=%t, want trusted registrationIncomplete without finalization", err, err, finalized)
	}

	var owners, parents, children int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM owner_lifecycles`).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&parents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_sessions`).Scan(&children); err != nil {
		t.Fatal(err)
	}
	if owners != 0 || parents != 0 || children != 0 {
		t.Fatalf("malformed success activated owners/parents/children=%d/%d/%d", owners, parents, children)
	}
	var nonQuarantinePersistence string
	if err := pool.QueryRow(context.Background(), `
		SELECT to_jsonb(request)::text FROM oauth_auth_requests request WHERE state=$1
	`, state).Scan(&nonQuarantinePersistence); err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []string{accessToken, refreshToken, rawMarker} {
		if strings.Contains(logs.String(), sentinel) {
			t.Fatalf("global logs retained malformed success sentinel %q: %s", sentinel, logs.String())
		}
		if strings.Contains(nonQuarantinePersistence, sentinel) {
			t.Fatalf("non-quarantine auth-request persistence retained malformed success sentinel %q", sentinel)
		}
	}

	var requestState string
	var credentialStatus *string
	var credentialData *string
	if err := pool.QueryRow(context.Background(), `
		SELECT request.request_state,credential.status,credential.data::text
		FROM oauth_auth_requests request
		LEFT JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE request.state=$1
	`, state).Scan(&requestState, &credentialStatus, &credentialData); err != nil {
		t.Fatal(err)
	}
	requests, _ := upstream.observations()
	revocations := 0
	for _, request := range requests {
		if request.path == "/oauth/revoke" {
			revocations++
		}
	}
	if requestState != string(auth.AuthRequestCleanupPending) || credentialStatus == nil ||
		*credentialStatus != "pending" || credentialData == nil {
		t.Fatalf("malformed success state=%s credential=%v data-present=%t revocations=%d, want cleanup_pending/pending structured quarantine",
			requestState, credentialStatus, credentialData != nil, revocations)
	}
	if !strings.Contains(*credentialData, accessToken) || !strings.Contains(*credentialData, refreshToken) {
		t.Fatal("structured quarantine omitted recoverable token strings")
	}
	if strings.Contains(*credentialData, rawMarker) || strings.Contains(*credentialData, `"sub": 17`) {
		t.Fatal("structured quarantine persisted the raw malformed token response")
	}
	if revocations != 0 {
		t.Fatal("durably quarantined malformed-success credentials were also immediately revoked")
	}
}

func TestProviderRegistrationTokenDPoPNonceReplayIsRedactedAndSingleUse(t *testing.T) {
	const nonceSentinel = "provider-token-nonce-sentinel"

	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationtokennonce")
	upstream := newRealFlowServer(t, owner)
	upstream.enableTokenDPoPNonceReplay()
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: syntax.Handle("token-nonce.real-flow.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "token-nonce-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	if err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"token-nonce-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error { return nil }); err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	requests, _ := upstream.observations()
	var tokenRequests []realFlowRequest
	for _, request := range requests {
		if request.path == "/oauth/token" {
			tokenRequests = append(tokenRequests, request)
		}
	}
	if len(tokenRequests) != 2 {
		t.Fatalf("token HTTP attempts = %d, want one nonce replay", len(tokenRequests))
	}
	if tokenRequests[0].form.Encode() != tokenRequests[1].form.Encode() ||
		tokenRequests[0].dpop == "" || tokenRequests[1].dpop == "" || tokenRequests[0].dpop == tokenRequests[1].dpop {
		t.Fatal("token nonce replay changed the form or reused the DPoP proof")
	}
	if strings.Contains(logs.String(), nonceSentinel) {
		t.Fatalf("global logs retained nonce response body: %s", logs.String())
	}
}

// IT-008: every invalid post-exchange token or authority result fails closed,
// and any received credential is durable before identity-network work.
func TestProviderRegistrationCallbackFaultBoundariesConverge(t *testing.T) {
	owner := syntax.DID("did:plc:registrationfault")
	validToken := func() map[string]any {
		return map[string]any{
			"sub": owner.String(), "scope": "atproto transition:generic",
			"access_token": "access-registration-fault", "refresh_token": "refresh-registration-fault",
		}
	}
	tests := []struct {
		name             string
		mutateToken      func(map[string]any)
		lookupError      bool
		missingPDS       bool
		mismatchedIssuer bool
	}{
		{name: "missing access token", mutateToken: func(token map[string]any) { delete(token, "access_token") }},
		{name: "missing refresh token", mutateToken: func(token map[string]any) { delete(token, "refresh_token") }},
		{name: "missing mandatory scope", mutateToken: func(token map[string]any) { token["scope"] = "atproto" }},
		{name: "duplicate scope", mutateToken: func(token map[string]any) { token["scope"] = "atproto atproto transition:generic" }},
		{name: "blank scope element", mutateToken: func(token map[string]any) { token["scope"] = "atproto  transition:generic" }},
		{name: "malformed scope element", mutateToken: func(token map[string]any) { token["scope"] = "atproto transition:\ngeneric" }},
		{name: "malformed DID", mutateToken: func(token map[string]any) { token["sub"] = "not-a-did" }},
		{name: "DID lookup failure", lookupError: true},
		{name: "missing PDS", missingPDS: true},
		{name: "mismatched authorization server", mismatchedIssuer: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			upstream := newRealFlowServer(t, owner)
			token := validToken()
			if test.mutateToken != nil {
				test.mutateToken(token)
			}
			upstream.setTokenResponse(token)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			resolved := &identity.Identity{
				DID: owner, Handle: syntax.Handle("fault.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}
			if test.missingPDS {
				resolved.Services = nil
			}
			clients.directory = realFlowDirectory{
				identity: resolved, lookupError: test.lookupError,
				beforeDIDLookup: func() {
					var status string
					if err := pool.QueryRow(context.Background(), `
						SELECT status FROM oauth_unverified_credentials
					`).Scan(&status); err != nil || status != "held" {
						t.Errorf("credential was not durably held before DID lookup: status=%q err=%v", status, err)
					}
				},
			}
			flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "fault-device"); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			if test.mismatchedIssuer {
				upstream.setProtectedIssuer("https://other-auth.real-flow.test")
			}
			finalized := false
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"fault-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				finalized = true
				return nil
			})
			if err == nil || finalized {
				t.Fatalf("invalid callback err=%v finalized=%t", err, finalized)
			}
			var requestState string
			var credentialStatus *string
			if err := pool.QueryRow(context.Background(), `
				SELECT request.request_state,credential.status
				FROM oauth_auth_requests request
				LEFT JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
				WHERE request.state=$1
			`, state).Scan(&requestState, &credentialStatus); err != nil {
				t.Fatal(err)
			}
			if requestState != string(auth.AuthRequestCleanupPending) || credentialStatus == nil || *credentialStatus != "pending" {
				t.Fatalf("fault state=%s credential=%v, want cleanup_pending/pending", requestState, credentialStatus)
			}
			var owners, parents int
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM owner_lifecycles`).Scan(&owners); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&parents); err != nil {
				t.Fatal(err)
			}
			if owners != 0 || parents != 0 {
				t.Fatalf("invalid callback created owners/parents=%d/%d", owners, parents)
			}
		})
	}

	t.Run("quarantine persistence failure immediately revokes", func(t *testing.T) {
		pool := withRealFlowAuthSchema(t)
		upstream := newRealFlowServer(t, owner)
		clients, _, _ := newRealFlowClients(t, upstream)
		t.Cleanup(func() {
			clients.boundary.CloseIdleConnections()
			upstream.close(t)
		})
		clients.directory = realFlowDirectory{identity: &identity.Identity{
			DID: owner, Handle: syntax.Handle("fault.real-flow.test"),
			Services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
			},
		}}
		flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
		if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "quarantine-fault-device"); err != nil {
			t.Fatal(err)
		}
		var state string
		if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `
			CREATE FUNCTION reject_registration_quarantine() RETURNS trigger
			LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled quarantine failure'; END $$;
			CREATE TRIGGER reject_registration_quarantine
			BEFORE INSERT ON oauth_unverified_credentials
			FOR EACH ROW EXECUTE FUNCTION reject_registration_quarantine();
		`); err != nil {
			t.Fatal(err)
		}
		err := flow.CompleteCallback(context.Background(), url.Values{
			"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"quarantine-fault-code"},
		}, func(context.Context, auth.OAuthCallbackResult) error {
			return errors.New("finalizer must not run")
		})
		if err == nil {
			t.Fatal("quarantine persistence failure completed callback")
		}
		var requestState string
		if err := pool.QueryRow(context.Background(), `
			SELECT request_state FROM oauth_auth_requests WHERE state=$1
		`, state).Scan(&requestState); err != nil {
			t.Fatal(err)
		}
		requests, _ := upstream.observations()
		revocations := 0
		for _, request := range requests {
			if request.path == "/oauth/revoke" {
				revocations++
			}
		}
		if requestState != string(auth.AuthRequestExchangeFailed) || revocations != 2 {
			t.Fatalf("quarantine failure state=%s revocations=%d, want exchange_failed and both tokens revoked", requestState, revocations)
		}
	})
}

// IT-009: verified registration enters the same exclusive lifecycle fence as
// ordinary onboarding and admits only absent, departed, or active owners.
func TestProviderRegistrationCallbackEnforcesOwnerLifecycleEligibility(t *testing.T) {
	tests := []struct {
		name       string
		ownerState string
		wantBound  bool
	}{
		{name: "absent owner", wantBound: true},
		{name: "departed owner", ownerState: "departed", wantBound: true},
		{name: "active owner", ownerState: "active", wantBound: true},
		{name: "deletion pending owner", ownerState: "deletion_pending"},
		{name: "deleting owner", ownerState: "deleting"},
		{name: "terminal owner", ownerState: "terminal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:lifecycle" + strings.ReplaceAll(test.name, " ", ""))
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			clients.directory = realFlowDirectory{identity: &identity.Identity{
				DID: owner, Handle: syntax.Handle("lifecycle.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}}
			if test.ownerState != "" {
				_, err := pool.Exec(context.Background(), `
					INSERT INTO owner_lifecycles(
						owner_did,state,generation,auth_epoch,transition_reason,
						transitioned_at,terminal_at,created_at,updated_at
					) VALUES($1,$2,4,7,'lifecycleFixture',now(),
					         CASE WHEN $2='terminal' THEN now() ELSE NULL END,now(),now())
				`, owner, test.ownerState)
				if err != nil {
					t.Fatal(err)
				}
			}
			flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "lifecycle-device"); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			finalized := false
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"lifecycle-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				finalized = true
				return nil
			})
			metadata, loadErr := store.LoadAuthRequestMetadata(context.Background(), state)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if test.wantBound {
				if err != nil || !finalized || metadata.Owner != owner || metadata.OwnerGeneration <= 0 || metadata.AuthEpoch <= 0 {
					t.Fatalf("eligible callback err=%v finalized=%t metadata=%+v", err, finalized, metadata)
				}
				return
			}
			if err == nil || finalized || metadata.Owner != "" || metadata.RequestState != auth.AuthRequestCleanupPending {
				t.Fatalf("ineligible callback err=%v finalized=%t metadata=%+v", err, finalized, metadata)
			}
			var credentialStatus string
			if err := pool.QueryRow(context.Background(), `
				SELECT status FROM oauth_unverified_credentials WHERE request_state=$1
			`, state).Scan(&credentialStatus); err != nil {
				t.Fatal(err)
			}
			if credentialStatus != "pending" {
				t.Fatalf("ineligible credential status=%s", credentialStatus)
			}
		})
	}
}

// IT-019: explicit provider denial consumes one trusted registration attempt,
// returns only the bounded cancellation result, and cannot be replayed.
func TestProviderRegistrationAccessDeniedConsumesOnceWithoutSession(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationdenied")
	upstream := newRealFlowServer(t, owner)
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)

	if _, err := flow.StartRegistration(
		context.Background(), auth.HandoffVerifiedLink, "", "registration-denied-device",
	); err != nil {
		t.Fatalf("StartRegistration: %v", err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `
		SELECT state FROM oauth_auth_requests WHERE purpose='registration'
	`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	handlers := &auth.HTTPHandlers{
		OAuthFlow:        flow,
		LoginCompleteURL: "https://app.craftsky.social/auth/complete",
	}
	providerDescription := "account denied: token=provider-secret https://internal.example"
	callbackPath := "/oauth/callback?state=" + url.QueryEscape(state) +
		"&error=access_denied&error_description=" + url.QueryEscape(providerDescription)

	first := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(first, httptest.NewRequest(http.MethodGet, callbackPath, nil))
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), "error=canceled") {
		t.Fatalf("first denial status=%d body=%s", first.Code, first.Body.String())
	}
	if strings.Contains(first.Body.String(), providerDescription) || strings.Contains(first.Body.String(), "provider-secret") ||
		strings.Contains(first.Body.String(), "internal.example") {
		t.Fatalf("first denial echoed provider description: %s", first.Body.String())
	}

	metadata, err := store.LoadAuthRequestMetadata(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.RequestState != auth.AuthRequestExchangeFailed {
		t.Fatalf("denied request state=%s, want exchange_failed", metadata.RequestState)
	}
	var oauthSessions, craftskySessions int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&oauthSessions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_sessions`).Scan(&craftskySessions); err != nil {
		t.Fatal(err)
	}
	requests, _ := upstream.observations()
	for _, request := range requests {
		if request.path == "/oauth/token" {
			t.Fatal("access_denied performed a token exchange")
		}
	}
	if oauthSessions != 0 || craftskySessions != 0 {
		t.Fatalf("access_denied created OAuth/Craftsky sessions=%d/%d", oauthSessions, craftskySessions)
	}

	replay := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(replay, httptest.NewRequest(http.MethodGet, callbackPath, nil))
	if replay.Code != http.StatusBadRequest || strings.Contains(replay.Body.String(), "error=canceled") ||
		strings.Contains(replay.Body.String(), "app.craftsky.social") {
		t.Fatalf("replay remained usable: status=%d body=%s", replay.Code, replay.Body.String())
	}
}

// IT-010: callback parameters and durable state are single-use under malformed
// retries and concurrent duplicate callbacks.
func TestProviderRegistrationCallbackStateAndCodeAreOneTime(t *testing.T) {
	t.Run("duplicate exact parameters are rejected before exchange", func(t *testing.T) {
		for _, parameter := range []string{"state", "iss", "code"} {
			t.Run(parameter, func(t *testing.T) {
				pool := withRealFlowAuthSchema(t)
				owner := syntax.DID("did:plc:duplicate" + parameter)
				upstream := newRealFlowServer(t, owner)
				clients, _, _ := newRealFlowClients(t, upstream)
				t.Cleanup(func() {
					clients.boundary.CloseIdleConnections()
					upstream.close(t)
				})
				clients.directory = realFlowDirectory{identity: &identity.Identity{
					DID: owner, Handle: syntax.Handle("duplicate.real-flow.test"),
					Services: map[string]identity.ServiceEndpoint{
						"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
					},
				}}
				flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
				if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "duplicate-device"); err != nil {
					t.Fatal(err)
				}
				var state string
				if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
					t.Fatal(err)
				}
				params := url.Values{"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"duplicate-code"}}
				params[parameter] = append(params[parameter], params[parameter][0])
				if err := flow.CompleteCallback(context.Background(), params, func(context.Context, auth.OAuthCallbackResult) error {
					return nil
				}); !errors.Is(err, auth.ErrOAuthFlowInvalid) {
					t.Fatalf("duplicate %s error=%v, want invalid flow", parameter, err)
				}
				requests, _ := upstream.observations()
				for _, request := range requests {
					if request.path == "/oauth/token" {
						t.Fatalf("duplicate %s reached token exchange", parameter)
					}
				}
			})
		}
	})

	t.Run("concurrent duplicate callback exchanges once", func(t *testing.T) {
		pool := withRealFlowAuthSchema(t)
		owner := syntax.DID("did:plc:registrationrace")
		upstream := newRealFlowServer(t, owner)
		clients, _, _ := newRealFlowClients(t, upstream)
		t.Cleanup(func() {
			clients.boundary.CloseIdleConnections()
			upstream.close(t)
		})
		clients.directory = realFlowDirectory{identity: &identity.Identity{
			DID: owner, Handle: syntax.Handle("race.real-flow.test"),
			Services: map[string]identity.ServiceEndpoint{
				"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
			},
		}}
		flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
		if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "race-device"); err != nil {
			t.Fatal(err)
		}
		var state string
		if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
			t.Fatal(err)
		}
		params := url.Values{"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"race-code"}}
		start := make(chan struct{})
		results := make(chan error, 2)
		for range 2 {
			go func() {
				<-start
				results <- flow.CompleteCallback(context.Background(), params, func(context.Context, auth.OAuthCallbackResult) error {
					return nil
				})
			}()
		}
		close(start)
		successes := 0
		for range 2 {
			if err := <-results; err == nil {
				successes++
			}
		}
		var parents int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&parents); err != nil {
			t.Fatal(err)
		}
		requests, _ := upstream.observations()
		tokenRequests := 0
		for _, request := range requests {
			if request.path == "/oauth/token" {
				tokenRequests++
			}
		}
		if successes != 1 || tokenRequests != 1 || parents != 1 {
			t.Fatalf("race successes=%d token requests=%d parents=%d", successes, tokenRequests, parents)
		}
		if err := flow.CompleteCallback(context.Background(), params, func(context.Context, auth.OAuthCallbackResult) error {
			return nil
		}); err == nil {
			t.Fatal("replayed callback succeeded")
		}
	})
}

// IT-020 callback portion: operation deadlines cancel remote dependencies and
// expired work cannot bind authority or create a parent later.
func TestProviderRegistrationCallbackDeadlineCancelsDependenciesWithoutLateActivation(t *testing.T) {
	tests := []struct {
		name           string
		blockToken     bool
		wantState      auth.AuthRequestState
		wantCredential string
	}{
		{name: "token exchange", blockToken: true, wantState: auth.AuthRequestExchangeAmbiguous},
		{name: "DID resolution", wantState: auth.AuthRequestCleanupPending, wantCredential: "pending"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:callbackdeadline" + strings.ReplaceAll(test.name, " ", ""))
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			dependencyCanceled := make(chan struct{})
			if test.blockToken {
				next := clients.oauth.Transport
				clients.oauth.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if request.URL.Path == "/oauth/token" {
						<-request.Context().Done()
						close(dependencyCanceled)
						return nil, request.Context().Err()
					}
					return next.RoundTrip(request)
				})
			}
			directory := realFlowDirectory{identity: &identity.Identity{
				DID: owner, Handle: syntax.Handle("deadline.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}}
			if !test.blockToken {
				directory.lookupDID = func(ctx context.Context, _ syntax.DID) (*identity.Identity, error) {
					<-ctx.Done()
					close(dependencyCanceled)
					return nil, ctx.Err()
				}
			}
			clients.directory = directory
			flow, _ := newRealRegistrationFlowWithTimeouts(t, pool, clients, 5*time.Second, 25*time.Millisecond)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "deadline-device"); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"deadline-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				return errors.New("deadline callback finalized")
			})
			if err == nil {
				t.Fatal("deadline callback returned nil")
			}
			if test.blockToken {
				var failure *auth.TrustedRegistrationFailure
				if !errors.As(err, &failure) || failure.Code != auth.RegistrationFailureProviderUnavailable {
					t.Fatalf("token deadline failure = %T %v, want trusted providerUnavailable", err, err)
				}
			}
			select {
			case <-dependencyCanceled:
			case <-time.After(time.Second):
				t.Fatal("callback deadline did not cancel dependency")
			}
			var requestState string
			var credentialStatus *string
			if err := pool.QueryRow(context.Background(), `
				SELECT request.request_state,credential.status
				FROM oauth_auth_requests request
				LEFT JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
				WHERE request.state=$1
			`, state).Scan(&requestState, &credentialStatus); err != nil {
				t.Fatal(err)
			}
			if requestState != string(test.wantState) ||
				(test.wantCredential != "" && (credentialStatus == nil || *credentialStatus != test.wantCredential)) {
				t.Fatalf("deadline state=%s credential=%v", requestState, credentialStatus)
			}
			var owners, parents int
			if err := pool.QueryRow(context.Background(), `
					SELECT
						(SELECT count(*)::int FROM owner_lifecycles),
						(SELECT count(*)::int FROM oauth_sessions)
			`).Scan(&owners, &parents); err != nil {
				t.Fatal(err)
			}
			if owners != 0 || parents != 0 {
				t.Fatalf("deadline created owners/parents=%d/%d", owners, parents)
			}
		})
	}
}

// IT-021 integration completion: the production callback publishes only the
// exact authority and parent committed while consuming its held quarantine.
func TestProviderRegistrationCallbackPublishesAtomicBoundAuthority(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationatomic")
	upstream := newRealFlowServer(t, owner)
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: syntax.Handle("atomic.real-flow.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "atomic-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"atomic-code"},
	}, func(_ context.Context, result auth.OAuthCallbackResult) error {
		if result.Metadata.Owner != owner || result.Metadata.OwnerGeneration != result.Attempt.OwnerGeneration ||
			result.Metadata.AuthEpoch != result.Attempt.AuthEpoch || result.Metadata.RequestState != auth.AuthRequestExchangeStarted {
			return fmt.Errorf("callback published unbound metadata: %+v attempt=%+v", result.Metadata, result.Attempt)
		}
		var requestOwner string
		var requestGeneration, requestEpoch, parentGeneration, parentEpoch int64
		var parentState string
		var quarantines int
		if err := pool.QueryRow(context.Background(), `
			SELECT request.owner_did,request.owner_generation,request.auth_epoch,
			       parent.owner_generation,parent.auth_epoch,parent.lifecycle_state
			FROM oauth_auth_requests request
			JOIN oauth_sessions parent
			  ON parent.account_did=request.owner_did AND parent.session_id=request.state
			WHERE request.state=$1
		`, state).Scan(
			&requestOwner, &requestGeneration, &requestEpoch,
			&parentGeneration, &parentEpoch, &parentState,
		); err != nil {
			return err
		}
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1
		`, state).Scan(&quarantines); err != nil {
			return err
		}
		if requestOwner != owner.String() || requestGeneration != result.Attempt.OwnerGeneration ||
			requestEpoch != result.Attempt.AuthEpoch || parentGeneration != requestGeneration ||
			parentEpoch != requestEpoch || parentState != "pending_handoff" || quarantines != 0 {
			return errors.New("atomic registration authority/parent/quarantine invariant failed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
}

// IT-011: a verified new DID uses the ordinary profile and handoff path, and
// neither the OAuth parent nor Craftsky child activates before confirmation.
func TestProviderRegistrationCompletesSharedOnboardingAndConfirmedHandoff(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	if _, err := pool.Exec(context.Background(), `
		ALTER TABLE craftsky_profiles
			ADD COLUMN crafts TEXT[] NOT NULL DEFAULT '{}',
			ADD COLUMN record_cid TEXT,
			ADD COLUMN indexed_at TIMESTAMPTZ NOT NULL DEFAULT now();
		CREATE TABLE bluesky_profiles (
			did TEXT PRIMARY KEY,
			display_name TEXT,
			description TEXT,
			pronouns TEXT,
			avatar_cid TEXT,
			avatar_mime TEXT,
			banner_cid TEXT,
			banner_mime TEXT,
			record_cid TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:registrationonboarding")
	upstream := newRealFlowServer(t, owner)
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: syntax.Handle("onboarding.real-flow.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	owners := newRealFlowOwnerStore(t, pool)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
		Pool: pool, Owners: owners, Sessions: children,
		RepositoryJobs: discardRepositoryJobs,
		ExchangeTTL:    5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
		ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
		Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	profileCID := syntax.CID("bafyreiregistrationonboarding")
	profileRecord := map[string]any{
		"displayName": "Registration Alice",
		"description": "Textile maker",
		"avatar": map[string]any{
			"ref": map[string]any{"$link": "bafkregistrationavatar"}, "mimeType": "image/jpeg",
		},
		"banner": map[string]any{
			"ref": map[string]any{"$link": "bafkregistrationbanner"}, "mimeType": "image/png",
		},
	}
	pds := &registrationOnboardingPDS{blueskyCID: profileCID, blueskyRecord: profileRecord}
	cache := &registrationOnboardingEffects{}
	profileHandler := index.NewBlueskyProfile(pool)
	profileProjector := oauthBlueskyProfileProjection{handler: profileHandler}
	craftskyProjector := oauthCraftskyProfileProjection{
		handler: index.NewTransactionalCraftskyProfile(
			pool, slog.Default(), notifications.NoopActorDeletion{},
		),
	}
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "onboarding-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	var code string
	err = flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"onboarding-code"},
	}, func(callbackCtx context.Context, result auth.OAuthCallbackResult) error {
		if _, err := store.ResumePendingOnboardingSession(callbackCtx, result.Attempt); err != nil {
			return err
		}
		if err := auth.InitializeProfileAndIdentityCache(
			callbackCtx, pds, result.Attempt, registrationOnboardingWriter{},
			profileProjector, craftskyProjector, cache,
			testlog.Discard(),
		); err != nil {
			return err
		}
		var displayName, description, avatarCID, bannerCID, recordCID string
		if err := pool.QueryRow(callbackCtx, `
			SELECT display_name,description,avatar_cid,banner_cid,record_cid
			FROM bluesky_profiles WHERE did=$1
		`, owner).Scan(&displayName, &description, &avatarCID, &bannerCID, &recordCID); err != nil {
			return fmt.Errorf("profile was not projected before handoff: %w", err)
		}
		if displayName != "Registration Alice" || description != "Textile maker" ||
			avatarCID != "bafkregistrationavatar" || bannerCID != "bafkregistrationbanner" ||
			recordCID != profileCID.String() {
			return fmt.Errorf("unexpected pre-handoff projection %q/%q/%q/%q/%q",
				displayName, description, avatarCID, bannerCID, recordCID)
		}
		var memberRows int
		if err := pool.QueryRow(callbackCtx, `
			SELECT count(*) FROM craftsky_profiles
			WHERE did=$1 AND record_cid='bafyregistrationcraftskyprofile'
		`, owner).Scan(&memberRows); err != nil || memberRows != 1 {
			return fmt.Errorf("Craftsky membership was not projected before handoff: rows=%d err=%v", memberRows, err)
		}
		var err error
		code, err = handoffs.CreateExchange(
			callbackCtx, result.Attempt, result.Handle, result.Metadata.DeviceID,
		)
		return err
	})
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if code == "" || pds.craftskyWrites != 1 || cache.calls != 1 {
		t.Fatalf("onboarding code=%q writes=%d cache=%d", code, pds.craftskyWrites, cache.calls)
	}
	if pds.blueskyReads != 1 || pds.craftskyReads != 1 {
		t.Fatalf("profile reads bluesky=%d craftsky=%d", pds.blueskyReads, pds.craftskyReads)
	}
	rawProfile, err := json.Marshal(profileRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := profileHandler.Handle(context.Background(), tap.Event{
		URI: syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.profile/self"),
		CID: profileCID, DID: owner, Collection: "app.bsky.actor.profile", Rkey: "self",
		Action: "create", Record: rawProfile,
	}); err != nil {
		t.Fatalf("Tap profile replay: %v", err)
	}
	var projectedRows int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM bluesky_profiles WHERE did=$1 AND record_cid=$2
	`, owner, profileCID).Scan(&projectedRows); err != nil || projectedRows != 1 {
		t.Fatalf("replayed profile rows = %d, %v", projectedRows, err)
	}
	exchange, err := handoffs.Exchange(context.Background(), code, "onboarding-device")
	if err != nil {
		t.Fatal(err)
	}
	var parentState, childState string
	if err := pool.QueryRow(context.Background(), `
		SELECT parent.lifecycle_state,child.lifecycle_state
		FROM oauth_sessions parent
		JOIN craftsky_sessions child
		  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
		WHERE parent.account_did=$1 AND parent.session_id=$2
	`, owner, state).Scan(&parentState, &childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "pending_handoff" || childState != "pending_confirmation" {
		t.Fatalf("pre-confirm parent/child=%s/%s", parentState, childState)
	}
	lifecycle, err := owners.Get(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: lifecycle.Generation,
		To: ownerlifecycle.StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoffs.Confirm(context.Background(), exchange.Token, exchange.ReceiptID, "onboarding-device"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT parent.lifecycle_state,child.lifecycle_state
		FROM oauth_sessions parent
		JOIN craftsky_sessions child
		  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
		WHERE parent.account_did=$1 AND parent.session_id=$2
	`, owner, state).Scan(&parentState, &childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "active" || childState != "active" {
		t.Fatalf("confirmed parent/child=%s/%s", parentState, childState)
	}
}

// IT-017: provider credentials stay outside Craftsky, while PDS OAuth
// credentials and DPoP material remain on the AppView side of the handoff.
func TestProviderRegistrationCredentialBoundaryInventory(t *testing.T) {
	const (
		providerEmail     = "provider-email-sentinel@example.test"
		providerPassword  = "provider-password-sentinel"
		authorizationCode = "authorization-code-sentinel"
		accessToken       = "access-token-sentinel"
		refreshToken      = "refresh-token-sentinel"
	)
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationboundary")
	upstream := newRealFlowServer(t, owner)
	upstream.setTokenResponse(map[string]any{
		"sub": owner.String(), "scope": "atproto transition:generic",
		"access_token": accessToken, "refresh_token": refreshToken,
	})
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: syntax.Handle("boundary.real-flow.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	owners := newRealFlowOwnerStore(t, pool)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
		Pool: pool, Owners: owners, Sessions: children,
		RepositoryJobs: discardRepositoryJobs,
		ExchangeTTL:    5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
		ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
		Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "boundary-device")
	if err != nil {
		t.Fatal(err)
	}
	var state string
	var requestData []byte
	if err := pool.QueryRow(context.Background(), `SELECT state,data FROM oauth_auth_requests`).Scan(&state, &requestData); err != nil {
		t.Fatal(err)
	}
	var requestInfo oauth.AuthRequestData
	if err := json.Unmarshal(requestData, &requestInfo); err != nil {
		t.Fatal(err)
	}
	if requestInfo.DPoPPrivateKeyMultibase == "" {
		t.Fatal("server request persistence omitted DPoP private key")
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	pds := &registrationOnboardingPDS{}
	handlers := &auth.HTTPHandlers{
		OAuthFlow: flow, LoginCompleteURL: "https://app.craftsky.social/auth/complete",
		NewPendingPDSClient: func(context.Context, auth.CallbackAttempt) (auth.PDSClient, error) { return pds, nil },
		OnboardingProfile:   registrationOnboardingWriter{}, Handoffs: handoffs,
		Logger: logger,
	}
	callback := "/oauth/callback?state=" + url.QueryEscape(state) +
		"&iss=" + url.QueryEscape(realFlowAuthOrigin) +
		"&code=" + url.QueryEscape(authorizationCode) +
		"&email=" + url.QueryEscape(providerEmail) +
		"&password=" + url.QueryEscape(providerPassword)
	request := httptest.NewRequest(http.MethodGet, callback, nil)
	request = request.WithContext(ctxkeys.WithRunID(request.Context(), "boundary-request-id"))
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", response.Code, response.Body.String())
	}

	var parentData []byte
	if err := pool.QueryRow(context.Background(), `SELECT data FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, state).Scan(&parentData); err != nil {
		t.Fatal(err)
	}
	serverPersistence := string(requestData) + string(parentData)
	for _, secret := range []string{accessToken, refreshToken, requestInfo.DPoPPrivateKeyMultibase} {
		if !strings.Contains(serverPersistence, secret) {
			t.Fatalf("server persistence omitted expected OAuth secret %q", secret)
		}
	}
	for _, providerCredential := range []string{providerEmail, providerPassword} {
		if strings.Contains(serverPersistence, providerCredential) {
			t.Fatalf("provider credential entered AppView persistence: %q", providerCredential)
		}
	}
	publicSurfaces := authURL + response.Body.String() + logs.String()
	for _, secret := range []string{
		providerEmail, providerPassword, authorizationCode, accessToken, refreshToken,
		requestInfo.DPoPPrivateKeyMultibase,
	} {
		if strings.Contains(publicSurfaces, secret) {
			t.Fatalf("public boundary retained %q: %s", secret, publicSurfaces)
		}
	}
	if !strings.Contains(response.Body.String(), "/auth/complete?code=") {
		t.Fatalf("callback omitted approved code-only handoff: %s", response.Body.String())
	}
	metadata, err := store.LoadAuthRequestMetadata(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Purpose != auth.RegistrationOAuthPurpose || metadata.Owner != owner {
		t.Fatalf("registration persistence authority = %+v", metadata)
	}
}

// IT-012: provider-first selection accepts an existing eligible owner without
// a DID-newness branch or duplicate owner lifecycle.
func TestProviderRegistrationAcceptsExistingOwnerAsNormalSignIn(t *testing.T) {
	for _, initialState := range []ownerlifecycle.State{
		ownerlifecycle.StateDeparted,
		ownerlifecycle.StateActive,
	} {
		t.Run(string(initialState), func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:registrationexisting" + string(initialState))
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			clients.directory = realFlowDirectory{identity: &identity.Identity{
				DID: owner, Handle: syntax.Handle("existing.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}}
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES($1,$2,4,7,'existingFixture',now(),now(),now())
			`, owner, initialState); err != nil {
				t.Fatal(err)
			}
			flow, store := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
			owners := newRealFlowOwnerStore(t, pool)
			children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
				Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
			})
			if err != nil {
				t.Fatal(err)
			}
			handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
				Pool: pool, Owners: owners, Sessions: children,
				RepositoryJobs: discardRepositoryJobs,
				ExchangeTTL:    5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
				ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
				Now: time.Now,
			})
			if err != nil {
				t.Fatal(err)
			}
			deviceID := "existing-" + string(initialState)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", deviceID); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			var code string
			err = flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"existing-code"},
			}, func(callbackCtx context.Context, result auth.OAuthCallbackResult) error {
				if _, err := store.ResumePendingOnboardingSession(callbackCtx, result.Attempt); err != nil {
					return err
				}
				var err error
				code, err = handoffs.CreateExchange(callbackCtx, result.Attempt, result.Handle, deviceID)
				return err
			})
			if err != nil {
				t.Fatalf("CompleteCallback: %v", err)
			}
			var ownerCount int
			var persistedState ownerlifecycle.State
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*),min(state) FROM owner_lifecycles WHERE owner_did=$1
			`, owner).Scan(&ownerCount, &persistedState); err != nil {
				t.Fatal(err)
			}
			if ownerCount != 1 || persistedState != initialState {
				t.Fatalf("owner count/state=%d/%s, want 1/%s", ownerCount, persistedState, initialState)
			}
			exchange, err := handoffs.Exchange(context.Background(), code, deviceID)
			if err != nil {
				t.Fatal(err)
			}
			if initialState == ownerlifecycle.StateDeparted {
				if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
					Owner: owner, ExpectedGeneration: 4,
					To: ownerlifecycle.StateActive, Reason: "profileCreated",
				}); err != nil {
					t.Fatal(err)
				}
			}
			if err := handoffs.Confirm(context.Background(), exchange.Token, exchange.ReceiptID, deviceID); err != nil {
				t.Fatal(err)
			}
			var parentState string
			if err := pool.QueryRow(context.Background(), `
				SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
			`, owner, state).Scan(&parentState); err != nil {
				t.Fatal(err)
			}
			if parentState != "active" {
				t.Fatalf("confirmed parent state=%s", parentState)
			}
		})
	}
}

func TestProviderRegistrationLifecycleAndHandoffAreNeutralAcrossConfiguredOrigins(t *testing.T) {
	for _, providerOrigin := range []string{realFlowPDSOrigin, realFlowSecondPDSOrigin} {
		t.Run(providerOrigin, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:providerneutral")
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			clients.directory = realFlowDirectory{identity: &identity.Identity{
				DID: owner, Handle: syntax.Handle("provider-neutral.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}}
			flow, store := newRealRegistrationFlowForProvider(
				t, pool, clients, providerOrigin, 5*time.Second, 5*time.Second,
			)
			owners := newRealFlowOwnerStore(t, pool)
			children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
				Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
			})
			if err != nil {
				t.Fatal(err)
			}
			handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
				Pool: pool, Owners: owners, Sessions: children,
				RepositoryJobs: discardRepositoryJobs,
				ExchangeTTL:    5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
				ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
				Now: time.Now,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "provider-neutral-device"); err != nil {
				t.Fatal(err)
			}
			var state, storedProvider string
			if err := pool.QueryRow(context.Background(), `
				SELECT state,registration_provider_origin FROM oauth_auth_requests
			`).Scan(&state, &storedProvider); err != nil {
				t.Fatal(err)
			}
			if storedProvider != providerOrigin {
				t.Fatalf("stored provider=%q, want %q", storedProvider, providerOrigin)
			}
			pds := &registrationOnboardingPDS{}
			cache := &registrationOnboardingEffects{}
			var code string
			err = flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"provider-neutral-code"},
			}, func(callbackCtx context.Context, result auth.OAuthCallbackResult) error {
				if result.Session.HostURL != realFlowPDSOrigin || result.Session.AuthServerURL != realFlowAuthOrigin {
					return fmt.Errorf("provider entered verified session contract: %+v", result.Session)
				}
				if _, err := store.ResumePendingOnboardingSession(callbackCtx, result.Attempt); err != nil {
					return err
				}
				if err := auth.InitializeProfileAndIdentityCache(
					callbackCtx, pds, result.Attempt, registrationOnboardingWriter{}, nil, nil, cache,
					testlog.Discard(),
				); err != nil {
					return err
				}
				var err error
				code, err = handoffs.CreateExchange(
					callbackCtx, result.Attempt, result.Handle, "provider-neutral-device",
				)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			exchange, err := handoffs.Exchange(context.Background(), code, "provider-neutral-device")
			if err != nil {
				t.Fatal(err)
			}
			lifecycle, err := owners.Get(context.Background(), owner)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
				Owner: owner, ExpectedGeneration: lifecycle.Generation,
				To: ownerlifecycle.StateActive, Reason: "profileCreated",
			}); err != nil {
				t.Fatal(err)
			}
			if err := handoffs.Confirm(
				context.Background(), exchange.Token, exchange.ReceiptID, "provider-neutral-device",
			); err != nil {
				t.Fatal(err)
			}
			var ownerState ownerlifecycle.State
			var parentState, childState string
			var parentData []byte
			if err := pool.QueryRow(context.Background(), `
				SELECT owner.state,parent.lifecycle_state,child.lifecycle_state,parent.data
				FROM owner_lifecycles owner
				JOIN oauth_sessions parent ON parent.account_did=owner.owner_did
				JOIN craftsky_sessions child
				  ON child.account_did=parent.account_did AND child.oauth_session_id=parent.session_id
				WHERE owner.owner_did=$1 AND parent.session_id=$2
			`, owner, state).Scan(&ownerState, &parentState, &childState, &parentData); err != nil {
				t.Fatal(err)
			}
			if ownerState != ownerlifecycle.StateActive || parentState != "active" || childState != "active" ||
				pds.craftskyWrites != 1 || cache.calls != 1 {
				t.Fatalf("provider-neutral outcome owner/parent/child=%s/%s/%s effects=%d/%d",
					ownerState, parentState, childState, pds.craftskyWrites, cache.calls)
			}
			if providerOrigin == realFlowSecondPDSOrigin && strings.Contains(string(parentData), providerOrigin) {
				t.Fatalf("configured start provider entered provider-neutral OAuth session: %s", parentData)
			}
		})
	}
}

// IT-022: a fatal finalizer failure abandons only the attempt parent after
// verified binding; it cannot activate a session or disturb prior authority.
func TestProviderRegistrationPostBindingFailureAbandonsOnlyAttempt(t *testing.T) {
	tests := []struct {
		name          string
		existingOwner bool
		failure       error
	}{
		{name: "new owner profile initialization", failure: fmt.Errorf("%w: injected", auth.ErrProfileInitFailed)},
		{name: "existing owner handoff preparation", existingOwner: true, failure: auth.ErrHandoffInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := withRealFlowAuthSchema(t)
			owner := syntax.DID("did:plc:registrationfailure" + strings.ReplaceAll(test.name, " ", ""))
			upstream := newRealFlowServer(t, owner)
			clients, _, _ := newRealFlowClients(t, upstream)
			t.Cleanup(func() {
				clients.boundary.CloseIdleConnections()
				upstream.close(t)
			})
			clients.directory = realFlowDirectory{identity: &identity.Identity{
				DID: owner, Handle: syntax.Handle("failure.real-flow.test"),
				Services: map[string]identity.ServiceEndpoint{
					"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
				},
			}}
			if test.existingOwner {
				if _, err := pool.Exec(context.Background(), `
					INSERT INTO owner_lifecycles(
						owner_did,state,generation,auth_epoch,transition_reason,
						transitioned_at,created_at,updated_at
					) VALUES($1,'active',4,7,'existingFixture',now(),now(),now())
				`, owner); err != nil {
					t.Fatal(err)
				}
				if _, err := pool.Exec(context.Background(), `
					INSERT INTO oauth_sessions(
						account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
						row_version,absolute_expires_at,created_at,updated_at
					) VALUES($1,'prior-active','{}','active',4,7,1,now()+interval '1 hour',now(),now())
				`, owner); err != nil {
					t.Fatal(err)
				}
			}
			flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
			if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "failure-device"); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"failure-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				return test.failure
			})
			if !errors.Is(err, test.failure) {
				t.Fatalf("CompleteCallback error=%v, want %v", err, test.failure)
			}
			var lifecycle ownerlifecycle.State
			if err := pool.QueryRow(context.Background(), `
				SELECT state FROM owner_lifecycles WHERE owner_did=$1
			`, owner).Scan(&lifecycle); err != nil {
				t.Fatal(err)
			}
			wantLifecycle := ownerlifecycle.StateDeparted
			if test.existingOwner {
				wantLifecycle = ownerlifecycle.StateActive
			}
			if lifecycle != wantLifecycle {
				t.Fatalf("owner lifecycle=%s, want %s", lifecycle, wantLifecycle)
			}
			var attemptParentState, requestState string
			if err := pool.QueryRow(context.Background(), `
				SELECT parent.lifecycle_state,request.request_state
				FROM oauth_sessions parent
				JOIN oauth_auth_requests request ON request.state=parent.session_id
				WHERE parent.account_did=$1 AND parent.session_id=$2
			`, owner, state).Scan(&attemptParentState, &requestState); err != nil {
				t.Fatal(err)
			}
			if attemptParentState != "revocation_pending" || requestState != "consumed" {
				t.Fatalf("failed attempt parent/request=%s/%s", attemptParentState, requestState)
			}
			var activeAttemptChildren, quarantines int
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*) FROM craftsky_sessions
				WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state='active'
			`, owner, state).Scan(&activeAttemptChildren); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1
			`, state).Scan(&quarantines); err != nil {
				t.Fatal(err)
			}
			if activeAttemptChildren != 0 || quarantines != 0 {
				t.Fatalf("failed attempt active children/quarantines=%d/%d", activeAttemptChildren, quarantines)
			}
			if test.existingOwner {
				var priorState string
				if err := pool.QueryRow(context.Background(), `
					SELECT lifecycle_state FROM oauth_sessions
					WHERE account_did=$1 AND session_id='prior-active'
				`, owner).Scan(&priorState); err != nil {
					t.Fatal(err)
				}
				if priorState != "active" {
					t.Fatalf("prior session state=%s", priorState)
				}
			}
		})
	}
}

func TestProviderRegistrationPendingParentPersistenceFailureLeavesCleanupOnly(t *testing.T) {
	pool := withRealFlowAuthSchema(t)
	owner := syntax.DID("did:plc:registrationbindingfailure")
	upstream := newRealFlowServer(t, owner)
	clients, _, _ := newRealFlowClients(t, upstream)
	t.Cleanup(func() {
		clients.boundary.CloseIdleConnections()
		upstream.close(t)
	})
	clients.directory = realFlowDirectory{identity: &identity.Identity{
		DID: owner, Handle: syntax.Handle("binding-failure.real-flow.test"),
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: realFlowPDSOrigin},
		},
	}}
	flow, _ := newRealRegistrationFlow(t, pool, clients, 5*time.Second)
	if _, err := flow.StartRegistration(context.Background(), auth.HandoffVerifiedLink, "", "binding-failure-device"); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM oauth_auth_requests`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE FUNCTION reject_registration_parent() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'injected pending parent failure';
		END $$;
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE TRIGGER reject_registration_parent
		BEFORE INSERT ON oauth_sessions
		FOR EACH ROW EXECUTE FUNCTION reject_registration_parent()
	`); err != nil {
		t.Fatal(err)
	}
	finalized := false
	err := flow.CompleteCallback(context.Background(), url.Values{
		"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"binding-failure-code"},
	}, func(context.Context, auth.OAuthCallbackResult) error {
		finalized = true
		return nil
	})
	if err == nil || finalized {
		t.Fatalf("binding failure err=%v finalized=%t", err, finalized)
	}
	var lifecycle ownerlifecycle.State
	var requestState, credentialState string
	if err := pool.QueryRow(context.Background(), `
		SELECT owner.state,request.request_state,credential.status
		FROM owner_lifecycles owner
		JOIN oauth_auth_requests request ON request.owner_did IS NULL
		JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE owner.owner_did=$1 AND request.state=$2
	`, owner, state).Scan(&lifecycle, &requestState, &credentialState); err != nil {
		t.Fatal(err)
	}
	var parents int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
	`, owner, state).Scan(&parents); err != nil {
		t.Fatal(err)
	}
	if lifecycle != ownerlifecycle.StateDeparted || requestState != "cleanup_pending" ||
		credentialState != "pending" || parents != 0 {
		t.Fatalf("binding failure lifecycle/request/credential/parents=%s/%s/%s/%d",
			lifecycle, requestState, credentialState, parents)
	}
}

type registrationOnboardingPDS struct {
	blueskyReads   int
	craftskyReads  int
	craftskyWrites int
	blueskyCID     syntax.CID
	blueskyRecord  map[string]any
}

func (pds *registrationOnboardingPDS) GetRecord(
	_ context.Context, _ syntax.DID, collection, _ string, result any,
) (string, error) {
	switch collection {
	case "app.bsky.actor.profile":
		pds.blueskyReads++
		if pds.blueskyRecord != nil {
			profile, ok := result.(*map[string]any)
			if !ok {
				return "", errors.New("unexpected Bluesky profile result")
			}
			*profile = pds.blueskyRecord
			return pds.blueskyCID.String(), nil
		}
		return "", auth.ErrRecordNotFound
	case "social.craftsky.actor.profile":
		pds.craftskyReads++
		return "", auth.ErrRecordNotFound
	default:
		return "", errors.New("unexpected profile collection")
	}
}

func (pds *registrationOnboardingPDS) PutRecord(
	_ context.Context, _ syntax.DID, collection, _ string, _ any,
) error {
	if collection != "social.craftsky.actor.profile" {
		return errors.New("unexpected profile write")
	}
	pds.craftskyWrites++
	return nil
}

func (*registrationOnboardingPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("unexpected create record")
}

func (*registrationOnboardingPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("unexpected delete record")
}

func (*registrationOnboardingPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected blob upload")
}

type registrationOnboardingWriter struct{}

func (registrationOnboardingWriter) PutOnboardingProfile(
	ctx context.Context, client auth.PDSClient, request auth.OnboardingProfileWrite,
) (syntax.CID, error) {
	if err := client.PutRecord(ctx, request.Owner, "social.craftsky.actor.profile", "self", request.Record); err != nil {
		return "", err
	}
	return "bafyregistrationcraftskyprofile", nil
}

type registrationOnboardingEffects struct{ calls int }

func (effects *registrationOnboardingEffects) RefreshCurrentHandle(context.Context, syntax.DID) error {
	effects.calls++
	return nil
}
