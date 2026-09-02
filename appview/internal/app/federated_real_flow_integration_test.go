package app

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"sync"
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
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const (
	realFlowPDSOrigin       = "https://pds.real-flow.test"
	realFlowSecondPDSOrigin = "https://pds-second.real-flow.test"
	realFlowAuthOrigin      = "https://auth.real-flow.test"
)

const realFlowAuthSchemaDDL = `
	CREATE TABLE craftsky_profiles (
		did TEXT PRIMARY KEY,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	CREATE TABLE oauth_sessions (
		account_did TEXT NOT NULL,
		session_id  TEXT NOT NULL,
		data        JSONB NOT NULL,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (account_did, session_id)
	);
	CREATE TABLE oauth_auth_requests (
		state                  TEXT NOT NULL PRIMARY KEY,
		data                   JSONB NOT NULL,
		created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
		handoff_mode           TEXT NOT NULL DEFAULT 'deep_link',
		loopback_redirect_uri  TEXT,
		device_id              TEXT,
		purpose                TEXT NOT NULL DEFAULT 'login',
		account_deletion_owner_did TEXT,
		account_deletion_job_id UUID,
		CONSTRAINT oauth_auth_requests_purpose_check
			CHECK (purpose IN ('login', 'accountDeletion')),
		CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
			CHECK (
				(purpose = 'login' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL)
				OR
				(purpose = 'accountDeletion' AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL)
			)
	);
	CREATE TABLE craftsky_sessions (
		token_hash        BYTEA NOT NULL PRIMARY KEY,
		account_did       TEXT NOT NULL,
		oauth_session_id  TEXT NOT NULL,
		device_label      TEXT,
		last_device_id    TEXT,
		created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
		last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
		revoked_at        TIMESTAMPTZ,
		FOREIGN KEY (account_did, oauth_session_id)
			REFERENCES oauth_sessions (account_did, session_id)
			ON DELETE CASCADE
	);
	CREATE TABLE account_deletion_operations (
		id UUID PRIMARY KEY,
		owner_did TEXT NOT NULL UNIQUE,
		state TEXT NOT NULL,
		accepted_at TIMESTAMPTZ,
		reauth_oauth_session_id TEXT,
		deletion_oauth_session_id TEXT,
		attempt_count INTEGER NOT NULL DEFAULT 0,
		next_attempt_at TIMESTAMPTZ,
		error_category TEXT,
		intent_proof_hash BYTEA,
		confirmation_handle_hash BYTEA,
		intent_expires_at TIMESTAMPTZ,
		lease_owner TEXT,
		lease_token UUID,
		lease_expires_at TIMESTAMPTZ,
		owner_generation BIGINT NOT NULL DEFAULT 1,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		FOREIGN KEY (owner_did, deletion_oauth_session_id)
			REFERENCES oauth_sessions(account_did, session_id),
		FOREIGN KEY (owner_did, reauth_oauth_session_id)
			REFERENCES oauth_sessions(account_did, session_id)
	);
`

func withRealFlowAuthSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, realFlowAuthSchemaDDL)
	for _, name := range []string{
		"000038_owner_auth_lifecycle.up.sql",
		"000064_provider_first_registration.up.sql",
	} {
		migration, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	return pool
}

func realFlowStoreConfig() auth.StoreConfig {
	return auth.StoreConfig{
		SessionExpiry:                180 * 24 * time.Hour,
		SessionAbsoluteLifetime:      180 * 24 * time.Hour,
		SessionInactivity:            30 * 24 * time.Hour,
		AuthRequestExpiry:            30 * time.Minute,
		PendingAuthRequestCapacity:   4096,
		AuthRequestTerminalRetention: 24 * time.Hour,
		Logger: slog.New(slog.NewTextHandler(
			os.Stderr, &slog.HandlerOptions{Level: slog.LevelError},
		)),
	}
}

func newRealFlowOwnerStore(t *testing.T, pool *pgxpool.Pool) *ownerlifecycle.Store {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func realFlowURL(t *testing.T, raw string) url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return *parsed
}

type realFlowResolver map[string][]netip.Addr

func (resolver realFlowResolver) LookupNetIP(
	_ context.Context,
	network string,
	host string,
) ([]netip.Addr, error) {
	if network != "ip" {
		return nil, errors.New("unexpected resolver network")
	}
	addresses, ok := resolver[host]
	if !ok {
		return nil, errors.New("unknown test hostname")
	}
	return append([]netip.Addr(nil), addresses...), nil
}

type realFlowDialer struct {
	target string

	mu        sync.Mutex
	addresses []string
}

func (dialer *realFlowDialer) DialContext(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error) {
	dialer.mu.Lock()
	dialer.addresses = append(dialer.addresses, address)
	dialer.mu.Unlock()
	target := dialer.target
	if host, _, err := net.SplitHostPort(address); err == nil {
		if ip, err := netip.ParseAddr(host); err == nil &&
			(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
			target = address
		}
	}
	return (&net.Dialer{}).DialContext(ctx, network, target)
}

func (dialer *realFlowDialer) calls() []string {
	dialer.mu.Lock()
	defer dialer.mu.Unlock()
	return append([]string(nil), dialer.addresses...)
}

type realFlowTrap struct {
	net.Listener
	mu       sync.Mutex
	accepted int
	done     chan struct{}
}

func newRealFlowTrap(t *testing.T) *realFlowTrap {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	trap := &realFlowTrap{Listener: listener, done: make(chan struct{})}
	go func() {
		defer close(trap.done)
		for {
			connection, err := trap.Accept()
			if err != nil {
				return
			}
			trap.mu.Lock()
			trap.accepted++
			trap.mu.Unlock()
			_ = connection.Close()
		}
	}()
	t.Cleanup(func() {
		_ = trap.Close()
		select {
		case <-trap.done:
		case <-time.After(time.Second):
			t.Error("private endpoint trap goroutine did not stop")
		}
	})
	return trap
}

func (trap *realFlowTrap) count() int {
	trap.mu.Lock()
	defer trap.mu.Unlock()
	return trap.accepted
}

type realFlowRequest struct {
	host, method, path, operation string
	form                          url.Values
	dpop                          string
}

type realFlowOAuthEndpoints struct {
	authorization string
	token         string
	par           string
	revocation    string
}

func defaultRealFlowOAuthEndpoints() realFlowOAuthEndpoints {
	return realFlowOAuthEndpoints{
		authorization: realFlowAuthOrigin + "/oauth/authorize",
		token:         realFlowAuthOrigin + "/oauth/token",
		par:           realFlowAuthOrigin + "/oauth/par",
		revocation:    realFlowAuthOrigin + "/oauth/revoke",
	}
}

type realFlowServer struct {
	listener net.Listener
	server   *http.Server
	done     chan error
	roots    *x509.CertPool

	mu               sync.Mutex
	requests         []realFlowRequest
	connections      map[net.Conn]http.ConnState
	endpoints        realFlowOAuthEndpoints
	promptValues     []string
	metadataScopes   []string
	protectedIssuer  string
	protectedStatus  int
	parStatus        int
	parBody          string
	parNonceReplay   bool
	parCalls         int
	tokenStatus      int
	tokenBody        string
	tokenNonceReplay bool
	tokenCalls       int
	tokenResponse    map[string]any
	accepted         int
	closed           bool
}

type countingRealFlowListener struct {
	net.Listener
	server *realFlowServer
}

func (listener countingRealFlowListener) Accept() (net.Conn, error) {
	connection, err := listener.Listener.Accept()
	if err == nil {
		listener.server.mu.Lock()
		listener.server.accepted++
		listener.server.mu.Unlock()
	}
	return connection, err
}

func newRealFlowServer(t *testing.T, owner syntax.DID) *realFlowServer {
	t.Helper()
	certificate, roots := realFlowCertificate(t)
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &realFlowServer{
		listener: base, done: make(chan error, 1), roots: roots,
		connections:     make(map[net.Conn]http.ConnState),
		endpoints:       defaultRealFlowOAuthEndpoints(),
		metadataScopes:  []string{"atproto", "transition:generic"},
		protectedIssuer: realFlowAuthOrigin,
		protectedStatus: http.StatusOK,
		tokenResponse: map[string]any{
			"sub": owner.String(), "scope": "atproto transition:generic",
			"access_token": "access-real-flow", "refresh_token": "refresh-real-flow",
		},
		tokenStatus: http.StatusOK,
		parStatus:   http.StatusCreated,
		parBody:     `{"request_uri":"urn:ietf:params:oauth:request_uri:real-flow","expires_in":60}`,
	}
	fixture.server = &http.Server{
		Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			fixture.serve(owner, response, request)
		}),
		ReadHeaderTimeout: time.Second,
		ConnState: func(connection net.Conn, state http.ConnState) {
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			if state == http.StateClosed || state == http.StateHijacked {
				delete(fixture.connections, connection)
				return
			}
			fixture.connections[connection] = state
		},
	}
	tlsListener := tls.NewListener(
		countingRealFlowListener{Listener: base, server: fixture},
		&tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12},
	)
	go func() { fixture.done <- fixture.server.Serve(tlsListener) }()
	return fixture
}

func (server *realFlowServer) serve(
	owner syntax.DID,
	response http.ResponseWriter,
	request *http.Request,
) {
	operation := request.URL.Path
	if request.Body != nil {
		_ = request.ParseForm()
		if grant := request.Form.Get("grant_type"); grant != "" {
			operation += ":" + grant
		}
		_, _ = io.Copy(io.Discard, request.Body)
		_ = request.Body.Close()
	}
	server.mu.Lock()
	server.requests = append(server.requests, realFlowRequest{
		host: request.Host, method: request.Method,
		path: request.URL.Path, operation: operation,
		form: request.Form, dpop: request.Header.Get("DPoP"),
	})
	server.mu.Unlock()

	response.Header().Set("Content-Type", "application/json")
	switch request.URL.Path {
	case "/.well-known/oauth-protected-resource":
		server.mu.Lock()
		protectedIssuer := server.protectedIssuer
		protectedStatus := server.protectedStatus
		server.mu.Unlock()
		if protectedStatus != http.StatusOK {
			response.WriteHeader(protectedStatus)
			return
		}
		_, _ = io.WriteString(response, `{"authorization_servers":["`+protectedIssuer+`"]}`)
	case "/.well-known/oauth-authorization-server":
		server.mu.Lock()
		endpoints := server.endpoints
		promptValues := append([]string(nil), server.promptValues...)
		metadataScopes := append([]string(nil), server.metadataScopes...)
		server.mu.Unlock()
		document := map[string]any{
			"issuer":                                           realFlowAuthOrigin,
			"authorization_endpoint":                           endpoints.authorization,
			"token_endpoint":                                   endpoints.token,
			"response_types_supported":                         []string{"code"},
			"grant_types_supported":                            []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":                 []string{"S256"},
			"token_endpoint_auth_methods_supported":            []string{"none", "private_key_jwt"},
			"token_endpoint_auth_signing_alg_values_supported": []string{"ES256"},
			"scopes_supported":                                 metadataScopes,
			"authorization_response_iss_parameter_supported":   true,
			"require_pushed_authorization_requests":            true,
			"pushed_authorization_request_endpoint":            endpoints.par,
			"dpop_signing_alg_values_supported":                []string{"ES256"},
			"client_id_metadata_document_supported":            true,
			"revocation_endpoint":                              endpoints.revocation,
		}
		if promptValues != nil {
			document["prompt_values_supported"] = promptValues
		}
		_ = json.NewEncoder(response).Encode(document)
	case "/oauth/par":
		server.mu.Lock()
		server.parCalls++
		status, body := server.parStatus, server.parBody
		if server.parNonceReplay {
			response.Header().Set("DPoP-Nonce", "real-flow-nonce")
			if server.parCalls == 1 {
				status = http.StatusBadRequest
				body = `{"error":"use_dpop_nonce","error_description":"provider-secret"}`
			}
		}
		server.mu.Unlock()
		response.WriteHeader(status)
		_, _ = io.WriteString(response, body)
	case "/oauth/token":
		server.mu.Lock()
		server.tokenCalls++
		tokenStatus, tokenBody := server.tokenStatus, server.tokenBody
		if server.tokenNonceReplay && server.tokenCalls == 1 {
			tokenStatus = http.StatusBadRequest
			tokenBody = `{"error":"use_dpop_nonce","error_description":"provider-token-nonce-sentinel"}`
			response.Header().Set("DPoP-Nonce", "real-flow-token-nonce")
		}
		tokenResponse := make(map[string]any, len(server.tokenResponse))
		for key, value := range server.tokenResponse {
			tokenResponse[key] = value
		}
		server.mu.Unlock()
		response.WriteHeader(tokenStatus)
		if tokenBody != "" {
			_, _ = io.WriteString(response, tokenBody)
			break
		}
		_ = json.NewEncoder(response).Encode(tokenResponse)
	case "/oauth/revoke":
		_, _ = io.WriteString(response, `{}`)
	case "/xrpc/com.atproto.repo.getRecord":
		_ = json.NewEncoder(response).Encode(map[string]any{
			"uri":   "at://" + owner.String() + "/app.bsky.actor.profile/self",
			"cid":   "bafyrealfederatedcid",
			"value": map[string]any{"displayName": "Real Flow Alice"},
		})
	case "/xrpc/com.atproto.repo.putRecord":
		_, _ = io.WriteString(response, `{}`)
	case "/xrpc/com.atproto.repo.uploadBlob":
		_ = json.NewEncoder(response).Encode(map[string]any{
			"blob": map[string]any{
				"$type": "blob", "ref": map[string]any{"$link": "bafyrealfederatedblob"},
				"mimeType": "image/png", "size": 4,
			},
		})
	default:
		response.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(response, `{"error":"not_found"}`)
	}
}

func (server *realFlowServer) close(t *testing.T) {
	t.Helper()
	server.mu.Lock()
	if server.closed {
		server.mu.Unlock()
		return
	}
	server.closed = true
	server.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.server.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown real-flow listener: %v", err)
	}
	select {
	case err := <-server.done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("serve real-flow listener: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("real-flow listener goroutine did not stop")
	}
	deadline := time.Now().Add(time.Second)
	for {
		server.mu.Lock()
		connections := len(server.connections)
		server.mu.Unlock()
		if connections == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("live listener connections after shutdown = %d", connections)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (server *realFlowServer) observations() ([]realFlowRequest, int) {
	server.mu.Lock()
	defer server.mu.Unlock()
	return append([]realFlowRequest(nil), server.requests...), server.accepted
}

func (server *realFlowServer) setOAuthEndpoints(endpoints realFlowOAuthEndpoints) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.endpoints = endpoints
}

func (server *realFlowServer) setRegistrationPAR(promptValues []string, status int, body string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.promptValues = append([]string(nil), promptValues...)
	server.parStatus = status
	server.parBody = body
}

func (server *realFlowServer) setRegistrationMetadataScopes(scopes []string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.metadataScopes = append([]string(nil), scopes...)
}

func (server *realFlowServer) enableRegistrationDPoPNonceReplay() {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.parNonceReplay = true
}

func (server *realFlowServer) setTokenResponse(response map[string]any) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.tokenStatus = http.StatusOK
	server.tokenBody = ""
	server.tokenResponse = response
}

func (server *realFlowServer) setTokenFailure(status int, body string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.tokenStatus = status
	server.tokenBody = body
}

func (server *realFlowServer) enableTokenDPoPNonceReplay() {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.tokenNonceReplay = true
}

func (server *realFlowServer) setProtectedIssuer(issuer string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.protectedIssuer = issuer
}

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
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: startTimeout, CallbackOperationTimeout: callbackTimeout,
		RegistrationProviderOrigin: providerOrigin, RegistrationOAuth: registrationOAuth,
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
			flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
				App: oauthApp, Store: store, Owners: owners,
				StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
				RegistrationProviderOrigin: realFlowPDSOrigin, RegistrationOAuth: registrationOAuth,
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
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout:      5 * time.Second,
		CallbackOperationTimeout:   5 * time.Second,
		RegistrationProviderOrigin: realFlowPDSOrigin,
		RegistrationOAuth:          registrationOAuth,
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
			started := time.Now()
			err := flow.CompleteCallback(context.Background(), url.Values{
				"state": {state}, "iss": {realFlowAuthOrigin}, "code": {"deadline-code"},
			}, func(context.Context, auth.OAuthCallbackResult) error {
				return errors.New("deadline callback finalized")
			})
			if err == nil || time.Since(started) > time.Second {
				t.Fatalf("deadline callback err=%v duration=%s", err, time.Since(started))
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
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM owner_lifecycles`).Scan(&owners); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions`).Scan(&parents); err != nil {
				t.Fatal(err)
			}
			time.Sleep(50 * time.Millisecond)
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
		ExchangeTTL: 5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
		ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
		Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	pds := &registrationOnboardingPDS{}
	cache := &registrationOnboardingEffects{}
	tracker := &registrationOnboardingEffects{}
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
			callbackCtx, pds, result.Attempt, registrationOnboardingWriter{}, cache,
			slog.New(slog.NewTextHandler(io.Discard, nil)), tracker,
		); err != nil {
			return err
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
	if code == "" || pds.craftskyWrites != 1 || cache.calls != 1 || tracker.calls != 1 {
		t.Fatalf("onboarding code=%q writes=%d cache=%d tracker=%d", code, pds.craftskyWrites, cache.calls, tracker.calls)
	}
	if pds.blueskyReads != 1 || pds.craftskyReads != 1 {
		t.Fatalf("profile reads bluesky=%d craftsky=%d", pds.blueskyReads, pds.craftskyReads)
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
		ExchangeTTL: 5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
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
				ExchangeTTL: 5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
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
				ExchangeTTL: 5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
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
			tracker := &registrationOnboardingEffects{}
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
					callbackCtx, pds, result.Attempt, registrationOnboardingWriter{}, cache,
					slog.New(slog.NewTextHandler(io.Discard, nil)), tracker,
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
				pds.craftskyWrites != 1 || cache.calls != 1 || tracker.calls != 1 {
				t.Fatalf("provider-neutral outcome owner/parent/child=%s/%s/%s effects=%d/%d/%d",
					ownerState, parentState, childState, pds.craftskyWrites, cache.calls, tracker.calls)
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
}

func (pds *registrationOnboardingPDS) GetRecord(
	_ context.Context, _ syntax.DID, collection, _ string, _ any,
) (string, error) {
	switch collection {
	case "app.bsky.actor.profile":
		pds.blueskyReads++
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
) error {
	return client.PutRecord(ctx, request.Owner, "social.craftsky.actor.profile", "self", request.Record)
}

type registrationOnboardingEffects struct{ calls int }

func (effects *registrationOnboardingEffects) UpsertCurrentHandle(context.Context, syntax.DID) error {
	effects.calls++
	return nil
}

func (effects *registrationOnboardingEffects) AddRepo(context.Context, syntax.DID) error {
	effects.calls++
	return nil
}

func realFlowCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	now := time.Now().UTC()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CraftSky real-flow test CA"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "CraftSky real-flow upstream"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
		DNSNames: []string{
			"pds.real-flow.test", "pds-second.real-flow.test", "auth.real-flow.test",
		},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	serverDER, err := x509.CreateCertificate(
		rand.Reader, serverTemplate, ca, &serverKey.PublicKey, caKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	certificate := tls.Certificate{
		Certificate: [][]byte{serverDER, caDER}, PrivateKey: serverKey,
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	return certificate, roots
}

type realFlowDirectory struct {
	identity        *identity.Identity
	lookupError     bool
	beforeDIDLookup func()
	lookupDID       func(context.Context, syntax.DID) (*identity.Identity, error)
}

func (directory realFlowDirectory) Lookup(
	_ context.Context,
	identifier syntax.AtIdentifier,
) (*identity.Identity, error) {
	if identifier.String() != directory.identity.Handle.String() &&
		identifier.String() != directory.identity.DID.String() {
		return nil, errors.New("unknown real-flow identity")
	}
	copy := *directory.identity
	return &copy, nil
}

func (directory realFlowDirectory) LookupDID(
	ctx context.Context,
	did syntax.DID,
) (*identity.Identity, error) {
	if directory.lookupDID != nil {
		return directory.lookupDID(ctx, did)
	}
	if directory.beforeDIDLookup != nil {
		directory.beforeDIDLookup()
	}
	if directory.lookupError {
		return nil, errors.New("controlled DID lookup failure")
	}
	if did != directory.identity.DID {
		return nil, errors.New("unknown real-flow DID")
	}
	copy := *directory.identity
	return &copy, nil
}

func (directory realFlowDirectory) LookupHandle(
	_ context.Context,
	handle syntax.Handle,
) (*identity.Identity, error) {
	if handle != directory.identity.Handle {
		return nil, errors.New("unknown real-flow handle")
	}
	copy := *directory.identity
	return &copy, nil
}

func (realFlowDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

type realFlowPurposeObserver struct {
	mu         sync.Mutex
	operations map[federatedhttp.Purpose]map[string]int
}

func (observer *realFlowPurposeObserver) wrap(
	purpose federatedhttp.Purpose,
	base http.RoundTripper,
) http.RoundTripper {
	return roundTripFunc(func(request *http.Request) (*http.Response, error) {
		observer.mu.Lock()
		operations := observer.operations[purpose]
		if operations == nil {
			operations = make(map[string]int)
			observer.operations[purpose] = operations
		}
		operations[request.URL.Path]++
		observer.mu.Unlock()
		return base.RoundTrip(request)
	})
}

func (observer *realFlowPurposeObserver) count(
	purpose federatedhttp.Purpose,
	path string,
) int {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return observer.operations[purpose][path]
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func newRealFlowClients(
	t *testing.T,
	server *realFlowServer,
) (*federatedClients, *realFlowDialer, *realFlowPurposeObserver) {
	t.Helper()
	dialer := &realFlowDialer{target: server.listener.Addr().String()}
	observer := &realFlowPurposeObserver{
		operations: make(map[federatedhttp.Purpose]map[string]int),
	}
	boundary, err := federatedhttp.NewTestBoundary(
		federatedhttp.DefaultTransportProfile(),
		federatedhttp.TestNetworkDependencies{
			Resolver: realFlowResolver{
				"pds.real-flow.test":        {netip.MustParseAddr("93.184.216.34")},
				"pds-second.real-flow.test": {netip.MustParseAddr("93.184.216.36")},
				"auth.real-flow.test":       {netip.MustParseAddr("93.184.216.35")},
			},
			Dialer: dialer, TLSRootCAs: server.roots,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatal(err)
	}
	clients, err := newFederatedClientsWithBoundary(config, boundary)
	if err != nil {
		t.Fatal(err)
	}
	observedClients := []struct {
		purpose federatedhttp.Purpose
		profile federatedhttp.Profile
		client  *http.Client
	}{
		{federatedhttp.PurposeOAuthMetadata, config.OAuthMetadata, clients.metadata},
		{federatedhttp.PurposeOAuthRequest, config.OAuthRequest, clients.oauth},
		{federatedhttp.PurposePDSJSON, config.PDSJSON, clients.pdsJSON},
		{federatedhttp.PurposePDSUpload, config.PDSUpload, clients.pdsBlob},
	}
	for _, observed := range observedClients {
		profile := observed.profile
		if profile.ResponseLimit <= 0 {
			t.Fatalf("%s client response limit = %d", observed.purpose, profile.ResponseLimit)
		}
		if observed.client.Timeout <= 0 || observed.client.Timeout > 30*time.Second {
			t.Fatalf("%s client timeout = %s", observed.purpose, observed.client.Timeout)
		}
		observed.client.Transport = observer.wrap(observed.purpose, observed.client.Transport)
	}
	return clients, dialer, observer
}

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
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
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
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: store, Owners: owners,
		StartOperationTimeout: 5 * time.Second, CallbackOperationTimeout: 5 * time.Second,
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
		App: oauthApp, Store: store, Owners: owners, OperationTimeout: 5 * time.Second,
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
			did TEXT PRIMARY KEY,display_name TEXT,description TEXT,
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
		"/.well-known/oauth-protected-resource":   1,
		"/.well-known/oauth-authorization-server": 1,
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
			"/.well-known/oauth-protected-resource":   1,
			"/.well-known/oauth-authorization-server": 1,
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
