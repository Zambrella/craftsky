package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// erroringGetPDSClient always errors on GetRecord (non-404). Used to
// exercise InitializeProfile's error propagation.
type erroringGetPDSClient struct{}

func (erroringGetPDSClient) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", errors.New("boom")
}
func (erroringGetPDSClient) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
	return nil
}
func (erroringGetPDSClient) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (erroringGetPDSClient) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}
func (erroringGetPDSClient) UploadBlob(_ context.Context, _ string, _ []byte) (*auth.UploadedBlob, error) {
	return nil, nil
}

// handlersFixture builds a test HTTPHandlers backed by a real oauth.ClientApp
// and the same prevalidated discovery artifacts used by production wiring.
func handlersFixture(t *testing.T, hostname string) *auth.HTTPHandlers {
	t.Helper()
	pool := withAuthSchema(t)
	input := auth.ClientConfigInput{
		Mode:        auth.ClientModeLocalhost,
		CallbackURL: mustURL(t, "http://127.0.0.1:18080/oauth/callback"),
		Scopes:      []string{"atproto", "transition:generic"},
	}
	if hostname != "" {
		privateKey, err := atcrypto.GeneratePrivateKeyP256()
		if err != nil {
			t.Fatal(err)
		}
		origin := "https://" + hostname
		input = auth.ClientConfigInput{
			Mode:            auth.ClientModeConfidential,
			ClientID:        mustURL(t, origin+"/oauth/client-metadata.json"),
			CallbackURL:     mustURL(t, origin+"/oauth/callback"),
			JWKSURL:         mustURL(t, origin+"/oauth/jwks.json"),
			ClientSecretKey: privateKey.Multibase(),
			ClientKeyID:     "primary",
			Scopes:          []string{"atproto", "transition:generic"},
		}
	}
	artifacts, err := auth.BuildClientArtifacts(input)
	if err != nil {
		t.Fatal(err)
	}
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	oauthApp := oauth.NewClientApp(&artifacts.Config, store)
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handlers := auth.NewHTTPHandlers(oauthApp, artifacts, craftsky, pool, logger)
	handlers.OnboardingProfile = testOnboardingProfileWriter{}
	lifecycle, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: newAuthOwnerStore(t, pool), Sessions: craftsky, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	handlers.SessionLifecycle = lifecycle
	return handlers
}

func TestClientMetadataDoesNotReflectRequestHost(t *testing.T) {
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:            auth.ClientModeConfidential,
		ClientID:        mustURL(t, "https://appview.craftsky.social/oauth/client-metadata.json"),
		CallbackURL:     mustURL(t, "https://appview.craftsky.social/oauth/callback"),
		JWKSURL:         mustURL(t, "https://appview.craftsky.social/oauth/jwks.json"),
		ClientSecretKey: privateKey.Multibase(),
		ClientKeyID:     "primary",
		Scopes:          []string{"atproto", "transition:generic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &auth.HTTPHandlers{
		OAuth:          oauth.NewClientApp(&artifacts.Config, nil),
		ClientMetadata: artifacts.Metadata,
		PublicJWKS:     artifacts.JWKS,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	request := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	request.Host = "attacker.invalid"
	request.Header.Set("Forwarded", "host=forwarded-attacker.invalid")
	request.Header.Set("X-Forwarded-Host", "proxy-attacker.invalid")
	recorder := httptest.NewRecorder()

	h.ClientMetadataHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("discovery security headers = %#v", recorder.Header())
	}
	var metadata oauth.ClientMetadata
	if err := json.NewDecoder(recorder.Body).Decode(&metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.JWKSURI == nil || *metadata.JWKSURI != "https://appview.craftsky.social/oauth/jwks.json" {
		t.Fatalf("jwks_uri = %v, want canonical origin", metadata.JWKSURI)
	}
	if strings.Contains(recorder.Body.String(), "attacker") {
		t.Fatalf("metadata reflected hostile host: %s", recorder.Body.String())
	}
}

func TestClientMetadata_Localhost(t *testing.T) {
	h := handlersFixture(t, "")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	h.ClientMetadataHandler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var meta oauth.ClientMetadata
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(meta.ClientID, "http://localhost?") {
		t.Fatalf("client_id: %q", meta.ClientID)
	}
	if !meta.DPoPBoundAccessTokens {
		t.Fatal("DPoPBoundAccessTokens must be true per atproto spec")
	}
}

func TestJWKS_LocalhostEmpty(t *testing.T) {
	h := handlersFixture(t, "")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/oauth/jwks.json", nil)
	h.JWKSHandler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	if rr.Header().Get("Cache-Control") != "no-store" || rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("JWKS security headers = %#v", rr.Header())
	}
	var jwks oauth.JWKS
	if err := json.NewDecoder(rr.Body).Decode(&jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 0 {
		t.Fatalf("expected 0 keys in localhost mode, got %d", len(jwks.Keys))
	}
}

// postLogin posts a JSON body to LoginHandler and returns the response.
func postLogin(t *testing.T, h *auth.HTTPHandlers, body string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithDeviceID(req.Context(), "device-login-test"))
	h.LoginHandler().ServeHTTP(rr, req)
	return rr
}

// expectEnvelopeError asserts the response body is a canonical
// envelope.Error with the given status and code, and that the message
// is non-empty.
func expectEnvelopeError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, status, rr.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v; body: %s", err, rr.Body.String())
	}
	if env.Error != code {
		t.Errorf("error = %q, want %q", env.Error, code)
	}
	if env.Message == "" {
		t.Errorf("message is empty")
	}
	// requestId may be "" if Logging middleware didn't run in the test
	// harness; we don't assert presence here.
}

func TestLogin_MissingHandle(t *testing.T) {
	h := handlersFixture(t, "")
	rr := postLogin(t, h, `{}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "handle_required")
}

func TestLogin_MalformedHandle(t *testing.T) {
	// "not a handle" has no dot and contains a space — fails syntax.ParseHandle.
	// We reject this at the boundary rather than letting indigo's resolver
	// chase a clearly-invalid identifier.
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"not a handle","handoffMode":"verified_link"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "invalid_handle")
}

func TestLogin_InvalidHandoffMode(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoffMode":"wat"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "invalid_handoff_mode")
}

func TestLoginDevSchemeRequiresExplicitServerCapability(t *testing.T) {
	disabled := handlersFixture(t, "")
	rr := postLogin(t, disabled, `{"handle":"alice.example","handoffMode":"dev_scheme"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "invalid_handoff_mode")

	enabled := handlersFixture(t, "")
	enabled.AllowDevScheme = true
	rr = postLogin(t, enabled, `{"handle":"alice.example","handoffMode":"dev_scheme"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("enabled dev scheme status = %d, want 503 after reaching unavailable OAuth flow; body=%s", rr.Code, rr.Body.String())
	}
}

func TestLogin_LoopbackMissingRedirect(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoffMode":"loopback"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "loopback_redirect_uri_required")
}

func TestLogin_LoopbackRedirectRejectsNonLoopback(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoffMode":"loopback","loopbackRedirectUri":"https://evil.example/"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "loopback_redirect_uri_invalid")
}

func TestLogin_LoopbackRedirectRejectsJavaScript(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoffMode":"loopback","loopbackRedirectUri":"javascript:alert(1)"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "loopback_redirect_uri_invalid")
}

func TestLogin_AcceptsCamelCaseBody(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoffMode":"verified_link"}`)
	if rr.Code != http.StatusServiceUnavailable {
		// The fixture intentionally has no OAuthFlow. Reaching the canonical
		// availability response proves camelCase decoding and validation passed.
		t.Fatalf("got %d, want 503 (body decoded, reached OAuth flow)", rr.Code)
	}
}

func TestLogin_RejectsSnakeCaseBody(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoff_mode":"verified_link"}`)
	// handoffMode absent -> invalid_handoff_mode 400.
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rr.Code)
	}
}

type registrationValidationFlow struct {
	calls int
	err   error
}

func (*registrationValidationFlow) StartLogin(context.Context, syntax.Handle, auth.HandoffMode, string, string) (string, error) {
	return "", errors.New("unexpected login start")
}

func (flow *registrationValidationFlow) StartRegistration(context.Context, auth.HandoffMode, string, string) (string, error) {
	flow.calls++
	return "https://auth.example/authorize", flow.err
}

func TestRegistrationHandlerMapsBoundedStartFailures(t *testing.T) {
	tests := []struct {
		name      string
		failure   *auth.RegistrationOAuthError
		wantError string
	}{
		{
			name: "provider unavailable",
			failure: &auth.RegistrationOAuthError{
				Stage: auth.RegistrationOAuthStagePAR, Code: auth.RegistrationOAuthProviderUnavailable,
			},
			wantError: "registration_provider_unavailable",
		},
		{
			name: "registration incomplete",
			failure: &auth.RegistrationOAuthError{
				Stage: auth.RegistrationOAuthStageMetadata, Code: auth.RegistrationOAuthIncomplete,
			},
			wantError: "registration_incomplete",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flow := &registrationValidationFlow{err: test.failure}
			handlers := &auth.HTTPHandlers{OAuthFlow: flow, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(`{"handoffMode":"verified_link"}`))
			request = request.WithContext(middleware.WithDeviceID(request.Context(), "classification-device"))
			response := httptest.NewRecorder()

			handlers.RegistrationHandler().ServeHTTP(response, request)

			if response.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want 502; body=%s", response.Code, response.Body.String())
			}
			var problem envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Error != test.wantError || strings.Contains(response.Body.String(), "metadata") || strings.Contains(response.Body.String(), "par") {
				t.Fatalf("bounded error = %+v, want %q", problem, test.wantError)
			}
		})
	}
}

func (*registrationValidationFlow) CompleteCallback(context.Context, url.Values, auth.OAuthCallbackFinalizer) error {
	return nil
}

func TestRegistrationRequestValidationDoesNotCallProvider(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		deviceID   string
		wantStatus int
		wantError  string
		wantCalls  int
	}{
		{name: "missing device", body: `{"handoffMode":"verified_link"}`, wantStatus: http.StatusBadRequest, wantError: "missing_device_id"},
		{name: "invalid device", body: `{"handoffMode":"verified_link"}`, deviceID: "not valid", wantStatus: http.StatusBadRequest, wantError: "invalid_device_id"},
		{name: "missing handoff", body: `{}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "invalid_handoff_mode"},
		{name: "invalid handoff", body: `{"handoffMode":"browser"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "invalid_handoff_mode"},
		{name: "loopback URI missing", body: `{"handoffMode":"loopback"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "loopback_redirect_uri_required"},
		{name: "loopback URI invalid", body: `{"handoffMode":"loopback","loopbackRedirectUri":"https://attacker.example"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "loopback_redirect_uri_invalid"},
		{name: "loopback URI forbidden for verified link", body: `{"handoffMode":"verified_link","loopbackRedirectUri":"http://127.0.0.1:1234/callback"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "loopback_redirect_uri_invalid"},
		{name: "unknown handle field", body: `{"handoffMode":"verified_link","handle":"alice.example"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "invalid_body"},
		{name: "unknown provider field", body: `{"handoffMode":"verified_link","provider":"https://attacker.example"}`, deviceID: "registration-device", wantStatus: http.StatusBadRequest, wantError: "invalid_body"},
		{name: "valid without handle", body: `{"handoffMode":"verified_link"}`, deviceID: "registration-device", wantStatus: http.StatusOK, wantCalls: 1},
		{name: "valid loopback", body: `{"handoffMode":"loopback","loopbackRedirectUri":"http://127.0.0.1:1234/callback"}`, deviceID: "registration-device", wantStatus: http.StatusOK, wantCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flow := &registrationValidationFlow{}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handlers := &auth.HTTPHandlers{OAuthFlow: flow, Logger: logger}
			handler := middleware.DeviceID(nil, logger)(handlers.RegistrationHandler())
			request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(test.body))
			if test.deviceID != "" {
				request.Header.Set("X-Craftsky-Device-Id", test.deviceID)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantError != "" {
				var problem envelope.Error
				if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
					t.Fatalf("decode error envelope: %v", err)
				}
				if problem.Error != test.wantError {
					t.Fatalf("error = %q, want %q", problem.Error, test.wantError)
				}
			}
			if flow.calls != test.wantCalls {
				t.Fatalf("provider calls = %d, want %d", flow.calls, test.wantCalls)
			}
		})
	}
}

// Helper: seed an oauth session and a craftsky session, return the bearer token.
func seedSession(t *testing.T, h *auth.HTTPHandlers, did, sid string) string {
	t.Helper()
	ctx := context.Background()
	seedActiveOAuthSession(t, h.Pool, did, sid)
	token, err := h.CraftskySessions.Create(ctx, did, sid, "")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestLogout_SingleDevice_SetsRevokedAt(t *testing.T) {
	h := handlersFixture(t, "")
	token := seedSession(t, h, "did:plc:a", "s1")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// Logout assumes Authenticated middleware ran. Inject DID/sid into ctx directly.
	ctx := middleware.WithDID(req.Context(), "did:plc:a")
	ctx = middleware.WithOAuthSessionID(ctx, "s1")
	ctx = middleware.WithDeviceID(ctx, "device-a")
	h.LogoutHandler().ServeHTTP(rr, req.WithContext(ctx))

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var revokedAt *time.Time
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT revoked_at FROM craftsky_sessions WHERE account_did='did:plc:a'`).Scan(&revokedAt); err != nil {
		t.Fatal(err)
	}
	if revokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}
	// OAuth session should still exist
	var count int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM oauth_sessions WHERE account_did='did:plc:a'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("oauth_sessions count: got %d want 1", count)
	}
}

func TestLogout_SharedInstallation_QueuesOwnerScopedCleanup(t *testing.T) {
	h := handlersFixture(t, "")
	h.NotificationSubscriptions = api.NewPostStore(h.Pool)
	tokenA := seedSession(t, h, "did:plc:a", "s1")
	seedSession(t, h, "did:plc:b", "s2")

	ctx := context.Background()
	if _, err := h.Pool.Exec(ctx, `
		CREATE TABLE push_installations (
		  id UUID PRIMARY KEY,
		  device_id TEXT NOT NULL UNIQUE,
		  platform TEXT NOT NULL,
		  fcm_token TEXT NOT NULL,
		  active BOOLEAN NOT NULL DEFAULT true,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  deactivated_at TIMESTAMPTZ
		);
		CREATE TABLE push_account_subscriptions (
		  id UUID PRIMARY KEY,
		  installation_id UUID NOT NULL REFERENCES push_installations(id) ON DELETE CASCADE,
		  account_did TEXT NOT NULL,
		  routing_id UUID NOT NULL UNIQUE,
		  active BOOLEAN NOT NULL DEFAULT true,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  deactivated_at TIMESTAMPTZ,
		  UNIQUE (installation_id, account_did)
		);
		CREATE TABLE push_deliveries (
		  id UUID PRIMARY KEY,
		  account_subscription_id UUID NOT NULL REFERENCES push_account_subscriptions(id) ON DELETE CASCADE,
		  status TEXT NOT NULL,
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		  lease_owner TEXT,
		  lease_expires_at TIMESTAMPTZ
		);
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ('10000000-0000-0000-0000-000000000001', 'shared-device', 'ios', 'shared-token');
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		VALUES
		  ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'did:plc:a', '30000000-0000-0000-0000-000000000001'),
		  ('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'did:plc:b', '30000000-0000-0000-0000-000000000002')
	`); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	reqCtx := middleware.WithDID(req.Context(), "did:plc:a")
	reqCtx = middleware.WithOAuthSessionID(reqCtx, "s1")
	reqCtx = middleware.WithDeviceID(reqCtx, "shared-device")
	recorder := httptest.NewRecorder()
	h.LogoutHandler().ServeHTTP(recorder, req.WithContext(reqCtx))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var activeA, activeB int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_sessions WHERE account_did='did:plc:a' AND revoked_at IS NULL`).Scan(&activeA); err != nil {
		t.Fatal(err)
	}
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_sessions WHERE account_did='did:plc:b' AND revoked_at IS NULL`).Scan(&activeB); err != nil {
		t.Fatal(err)
	}
	if activeA != 0 || activeB != 1 {
		t.Fatalf("active sessions: A=%d B=%d, want A=0 B=1", activeA, activeB)
	}

	var subscriptionA, subscriptionB bool
	if err := h.Pool.QueryRow(ctx, `SELECT active FROM push_account_subscriptions WHERE account_did='did:plc:a'`).Scan(&subscriptionA); err != nil {
		t.Fatal(err)
	}
	if err := h.Pool.QueryRow(ctx, `SELECT active FROM push_account_subscriptions WHERE account_did='did:plc:b'`).Scan(&subscriptionB); err != nil {
		t.Fatal(err)
	}
	if !subscriptionA || !subscriptionB {
		t.Fatalf("inline logout changed subscriptions: A=%t B=%t, want both true until cleanup", subscriptionA, subscriptionB)
	}
	var cleanupJobs int
	if err := h.Pool.QueryRow(ctx, `
		SELECT count(*) FROM auth_auxiliary_cleanup_jobs
		WHERE owner_did='did:plc:a' AND kind='installation_push'
		  AND installation_id='shared-device' AND state='pending'
	`).Scan(&cleanupJobs); err != nil {
		t.Fatal(err)
	}
	if cleanupJobs != 1 {
		t.Fatalf("Alice cleanup jobs = %d, want 1", cleanupJobs)
	}
}

type failingNotificationCleaner struct{}

func (failingNotificationCleaner) DeactivateForInstallation(context.Context, string, string) error {
	return errors.New("cleanup failed")
}
func (failingNotificationCleaner) DeactivateForAccount(context.Context, string) error {
	return errors.New("cleanup failed")
}

func TestLogoutCommitsLocalRevocationWithoutInlineNotificationCleanup(t *testing.T) {
	h := handlersFixture(t, "")
	h.NotificationSubscriptions = failingNotificationCleaner{}
	token := seedSession(t, h, "did:plc:a", "s1")
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := middleware.WithDID(req.Context(), "did:plc:a")
	ctx = middleware.WithOAuthSessionID(ctx, "s1")
	ctx = middleware.WithDeviceID(ctx, "device-a")
	recorder := httptest.NewRecorder()
	h.LogoutHandler().ServeHTTP(recorder, req.WithContext(ctx))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d", recorder.Code)
	}
	var revokedAt *time.Time
	if err := h.Pool.QueryRow(context.Background(), `SELECT revoked_at FROM craftsky_sessions WHERE account_did='did:plc:a'`).Scan(&revokedAt); err != nil {
		t.Fatal(err)
	}
	if revokedAt == nil {
		t.Fatal("session remained active while durable push cleanup was queued")
	}
	var cleanupJobs int
	if err := h.Pool.QueryRow(context.Background(), `
		SELECT count(*) FROM auth_auxiliary_cleanup_jobs
		WHERE owner_did='did:plc:a' AND kind='installation_push'
		  AND installation_id='device-a' AND state='pending'
	`).Scan(&cleanupJobs); err != nil {
		t.Fatal(err)
	}
	if cleanupJobs != 1 {
		t.Fatalf("cleanup jobs = %d, want 1", cleanupJobs)
	}
}

func TestLogout_AllDevices_RevokeAllCleansUpEvenIfOAuthLogoutFails(t *testing.T) {
	h := handlersFixture(t, "")
	seedSession(t, h, "did:plc:b", "s1")
	seedSession(t, h, "did:plc:b", "s2")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/logout?all=true", nil)
	req.Header.Set("Authorization", "Bearer dummy")
	ctx := middleware.WithDID(req.Context(), "did:plc:b")
	ctx = middleware.WithOAuthSessionID(ctx, "s1")
	h.LogoutHandler().ServeHTTP(rr, req.WithContext(ctx))

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	// Both Craftsky sessions for this DID should be revoked.
	var unrevokedCount int
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_sessions WHERE account_did='did:plc:b' AND revoked_at IS NULL`).Scan(&unrevokedCount); err != nil {
		t.Fatal(err)
	}
	if unrevokedCount != 0 {
		t.Fatalf("expected 0 unrevoked sessions, got %d", unrevokedCount)
	}
}

func TestSavedPostStateSurvivesSessionDeviceAndAccountLifecycle(t *testing.T) {
	h := handlersFixture(t, "")
	ctx := context.Background()
	if _, err := h.Pool.Exec(ctx, `
		ALTER TABLE craftsky_profiles ADD COLUMN record_cid TEXT;
		CREATE TABLE craftsky_posts (
			uri TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			rkey TEXT NOT NULL,
			cid TEXT NOT NULL
		);
		CREATE TABLE saved_post_folders (
			id UUID PRIMARY KEY,
			owner_did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (owner_did, id)
		);
		CREATE TABLE saved_posts (
			owner_did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
			post_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
			folder_id UUID,
			saved_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (owner_did, post_uri),
			FOREIGN KEY (owner_did, folder_id)
				REFERENCES saved_post_folders(owner_did, id)
				ON DELETE SET NULL (folder_id)
		);
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:a', 'a-cid'), ('did:plc:b', 'b-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES ('at://did:plc:b/social.craftsky.feed.post/one', 'did:plc:b', 'one', 'post-cid');
		INSERT INTO saved_post_folders (id, owner_did, name, created_at, updated_at)
		VALUES
			('00000000-0000-4000-8000-000000000001', 'did:plc:a', 'Alice', now(), now()),
			('00000000-0000-4000-8000-000000000002', 'did:plc:b', 'Bob', now(), now());
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES
			('did:plc:a', 'at://did:plc:b/social.craftsky.feed.post/one', '00000000-0000-4000-8000-000000000001', now()),
			('did:plc:b', 'at://did:plc:b/social.craftsky.feed.post/one', '00000000-0000-4000-8000-000000000002', now());
	`); err != nil {
		t.Fatalf("seed saved lifecycle state: %v", err)
	}

	assertSavedState := func(owner string, want int) {
		t.Helper()
		var saves, folders int
		if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM saved_posts WHERE owner_did = $1`, owner).Scan(&saves); err != nil {
			t.Fatalf("count %s saves: %v", owner, err)
		}
		if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM saved_post_folders WHERE owner_did = $1`, owner).Scan(&folders); err != nil {
			t.Fatalf("count %s folders: %v", owner, err)
		}
		if saves != want || folders != want {
			t.Fatalf("%s saved state = %d saves/%d folders, want %d/%d", owner, saves, folders, want, want)
		}
	}

	tokenA := seedSession(t, h, "did:plc:a", "a-session-1")
	seedSession(t, h, "did:plc:b", "b-session-1")
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+tokenA)
	requestCtx := middleware.WithDID(request.Context(), "did:plc:a")
	requestCtx = middleware.WithOAuthSessionID(requestCtx, "a-session-1")
	requestCtx = middleware.WithDeviceID(requestCtx, "device-a")
	recorder := httptest.NewRecorder()
	h.LogoutHandler().ServeHTTP(recorder, request.WithContext(requestCtx))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("single-session logout = %d/%s", recorder.Code, recorder.Body.String())
	}
	assertSavedState("did:plc:a", 1)
	assertSavedState("did:plc:b", 1)

	// A fresh local installation/account switch creates a new session; it does
	// not recreate or transfer membership-owned saved state.
	seedSession(t, h, "did:plc:a", "a-session-2")
	request = httptest.NewRequest(http.MethodPost, "/auth/logout?all=true", nil)
	request.Header.Set("Authorization", "Bearer replacement-token")
	requestCtx = middleware.WithDID(request.Context(), "did:plc:a")
	requestCtx = middleware.WithOAuthSessionID(requestCtx, "a-session-2")
	recorder = httptest.NewRecorder()
	h.LogoutHandler().ServeHTTP(recorder, request.WithContext(requestCtx))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("all-session logout = %d/%s", recorder.Code, recorder.Body.String())
	}
	assertSavedState("did:plc:a", 1)
	assertSavedState("did:plc:b", 1)

	// Token/session expiry deletes only auth rows. Trigger the real lazy OAuth
	// cleanup after aging Bob's session, then verify private membership state.
	if _, err := h.Pool.Exec(ctx, `
		UPDATE oauth_sessions
		SET created_at = now() - interval '2 hours', updated_at = now() - interval '2 hours'
		WHERE account_did = 'did:plc:b'
	`); err != nil {
		t.Fatalf("age Bob OAuth session: %v", err)
	}
	expiringStore := auth.NewPostgresAuthStore(h.Pool, auth.StoreConfig{
		SessionExpiry:     time.Hour,
		SessionInactivity: time.Hour,
		AuthRequestExpiry: time.Hour,
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	})
	_, _ = expiringStore.GetSession(ctx, syntax.DID("did:plc:b"), "b-session-1")
	assertSavedState("did:plc:a", 1)
	assertSavedState("did:plc:b", 1)

	if _, err := h.Pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:a'`); err != nil {
		t.Fatalf("delete Alice membership: %v", err)
	}
	assertSavedState("did:plc:a", 0)
	assertSavedState("did:plc:b", 1)
}

// TestCallbackTemplate_XSSRegression is a regression test against
// accidentally swapping html/template for text/template in
// handlers_render.go. With html/template's contextual escaping in
// place, a hostile loopback_redirect_uri cannot break out of the JS
// string literal in the rendered <script> tag. Without it, the regex
// at /auth/login ingress would be the only line of defence; this test
// fails loudly the moment the contextual escaping disappears.
func TestCallbackTemplate_XSSRegression(t *testing.T) {
	var buf bytes.Buffer
	hostile := `http://127.0.0.1:1234/x"></script><script>alert(1)//`
	if err := auth.RenderCallbackForTest(&buf, "tok", hostile); err != nil {
		t.Fatalf("RenderCallbackForTest: %v", err)
	}
	out := buf.String()
	// The hostile substring must not appear literally — html/template's
	// JS-string-context escaping should rewrite the special chars.
	if strings.Contains(out, `</script><script>`) {
		t.Fatalf("XSS payload survived template rendering — contextual escaping broken!\nrendered:\n%s", out)
	}
	if strings.Contains(out, `alert(1)`) {
		// alert(1) is fine literally inside a JS string — but only as long
		// as the surrounding quotes are intact. The check above already
		// catches a broken-out script tag.
		t.Logf("note: 'alert(1)' literal appears in output — fine if inside a JS string literal. Rendered:\n%s", out)
	}
}

func TestInitializeProfile_BlueskyErrorPropagates(t *testing.T) {
	// Lightweight alternative to driving ProcessCallback end-to-end:
	// verify the error-path wiring by invoking the function directly.
	// The callback happy path is exercised by the existing tests that
	// use handlersFixture's injected onboarding profile writer.
	err := auth.InitializeProfile(
		context.Background(),
		erroringGetPDSClient{},
		loginAttempt(syntax.DID("did:plc:me")),
		testOnboardingProfileWriter{},
	)
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Fatalf("want ErrProfileInitFailed; got %v", err)
	}
}
