package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
	"sync"
	"testing"
	"time"
)

var discardRepositoryJobs = auth.RepositoryJobTxEnqueuerFunc(
	func(context.Context, pgx.Tx, syntax.DID, auth.RepositoryJobKind) error { return nil },
)

const (
	realFlowPDSOrigin        = "https://pds.real-flow.test"
	realFlowSecondPDSOrigin  = "https://pds-second.real-flow.test"
	realFlowAuthOrigin       = "https://auth.real-flow.test"
	realFlowSecondAuthOrigin = "https://auth-second.real-flow.test"
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
		"000066_pds_migration_identity.up.sql",
	} {
		migration, err := testdb.ReadMigration(name)
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
	authorization, dpop           string
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
	connectionsDone  chan struct{}
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
				if fixture.closed && len(fixture.connections) == 0 && fixture.connectionsDone != nil {
					close(fixture.connectionsDone)
					fixture.connectionsDone = nil
				}
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
		form: request.Form, authorization: request.Header.Get("Authorization"),
		dpop: request.Header.Get("DPoP"),
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
		issuer := realFlowAuthOrigin
		if request.Host == "auth-second.real-flow.test" {
			issuer = realFlowSecondAuthOrigin
			endpoints = realFlowOAuthEndpoints{
				authorization: issuer + "/oauth/authorize",
				token:         issuer + "/oauth/token",
				par:           issuer + "/oauth/par",
				revocation:    issuer + "/oauth/revoke",
			}
		}
		document := map[string]any{
			"issuer":                                           issuer,
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
	if len(server.connections) != 0 {
		server.connectionsDone = make(chan struct{})
	}
	connectionsDone := server.connectionsDone
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
	if connectionsDone != nil {
		select {
		case <-connectionsDone:
		case <-time.After(time.Second):
			server.mu.Lock()
			connections := len(server.connections)
			server.mu.Unlock()
			t.Fatalf("live listener connections after shutdown = %d", connections)
		}
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
			"auth-second.real-flow.test",
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
				"pds.real-flow.test":         {netip.MustParseAddr("93.184.216.34")},
				"pds-second.real-flow.test":  {netip.MustParseAddr("93.184.216.36")},
				"auth.real-flow.test":        {netip.MustParseAddr("93.184.216.35")},
				"auth-second.real-flow.test": {netip.MustParseAddr("93.184.216.37")},
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
		{federatedhttp.PurposePDSRepository, config.PDSRepository, clients.pdsRepository},
	}
	for _, observed := range observedClients {
		profile := observed.profile
		if profile.ResponseLimit <= 0 {
			t.Fatalf("%s client response limit = %d", observed.purpose, profile.ResponseLimit)
		}
		if observed.client.Timeout != profile.TotalTimeout {
			t.Fatalf("%s client timeout = %s", observed.purpose, observed.client.Timeout)
		}
		observed.client.Transport = observer.wrap(observed.purpose, observed.client.Transport)
	}
	return clients, dialer, observer
}
