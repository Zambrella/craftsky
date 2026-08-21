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
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
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
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const (
	realFlowPDSOrigin  = "https://pds.real-flow.test"
	realFlowAuthOrigin = "https://auth.real-flow.test"
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
		account_deletion_job_id UUID
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
	migration, err := os.ReadFile("../../migrations/000038_owner_auth_lifecycle.up.sql")
	if err != nil {
		t.Fatalf("read auth lifecycle migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply auth lifecycle migration: %v", err)
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

	mu          sync.Mutex
	requests    []realFlowRequest
	connections map[net.Conn]http.ConnState
	endpoints   realFlowOAuthEndpoints
	accepted    int
	closed      bool
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
		connections: make(map[net.Conn]http.ConnState),
		endpoints:   defaultRealFlowOAuthEndpoints(),
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
	})
	server.mu.Unlock()

	response.Header().Set("Content-Type", "application/json")
	switch request.URL.Path {
	case "/.well-known/oauth-protected-resource":
		_, _ = io.WriteString(response, `{"authorization_servers":["`+realFlowAuthOrigin+`"]}`)
	case "/.well-known/oauth-authorization-server":
		server.mu.Lock()
		endpoints := server.endpoints
		server.mu.Unlock()
		_ = json.NewEncoder(response).Encode(map[string]any{
			"issuer":                                           realFlowAuthOrigin,
			"authorization_endpoint":                           endpoints.authorization,
			"token_endpoint":                                   endpoints.token,
			"response_types_supported":                         []string{"code"},
			"grant_types_supported":                            []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":                 []string{"S256"},
			"token_endpoint_auth_methods_supported":            []string{"none", "private_key_jwt"},
			"token_endpoint_auth_signing_alg_values_supported": []string{"ES256"},
			"scopes_supported":                                 []string{"atproto", "transition:generic"},
			"authorization_response_iss_parameter_supported":   true,
			"require_pushed_authorization_requests":            true,
			"pushed_authorization_request_endpoint":            endpoints.par,
			"dpop_signing_alg_values_supported":                []string{"ES256"},
			"client_id_metadata_document_supported":            true,
			"revocation_endpoint":                              endpoints.revocation,
		})
	case "/oauth/par":
		response.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(response, `{"request_uri":"urn:ietf:params:oauth:request_uri:real-flow","expires_in":60}`)
	case "/oauth/token":
		_ = json.NewEncoder(response).Encode(map[string]any{
			"sub": owner.String(), "scope": "atproto transition:generic",
			"access_token": "access-real-flow", "refresh_token": "refresh-real-flow",
		})
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
		DNSNames:    []string{"pds.real-flow.test", "auth.real-flow.test"},
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
	identity *identity.Identity
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
	_ context.Context,
	did syntax.DID,
) (*identity.Identity, error) {
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
				"pds.real-flow.test":  {netip.MustParseAddr("93.184.216.34")},
				"auth.real-flow.test": {netip.MustParseAddr("93.184.216.35")},
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
