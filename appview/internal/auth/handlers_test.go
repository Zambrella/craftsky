package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// noopPDSClient satisfies auth.PDSClient without touching any real PDS.
// Used by handlersFixture so the OAuth callback tests that don't care
// about onboarding-on-login don't fail in InitializeProfile. GetRecord
// returns 404 (record missing), causing InitializeProfile to emit an
// empty-Craftsky-profile write — which also returns nil here, a no-op.
type noopPDSClient struct{}

func (noopPDSClient) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", auth.ErrRecordNotFound
}
func (noopPDSClient) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
	return nil
}

// erroringGetPDSClient always errors on GetRecord (non-404). Used to
// exercise InitializeProfile's error propagation.
type erroringGetPDSClient struct{}

func (erroringGetPDSClient) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", errors.New("boom")
}
func (erroringGetPDSClient) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
	return nil
}

// handlersFixture builds a test HTTPHandlers backed by a real
// oauth.ClientApp built from BuildClientConfig, and the Postgres
// test-schema stores.
func handlersFixture(t *testing.T, hostname string) *auth.HTTPHandlers {
	t.Helper()
	pool := withAuthSchema(t)
	cfg, err := auth.BuildClientConfig(hostname, "", "", []string{"atproto", "transition:generic"})
	if err != nil {
		t.Fatal(err)
	}
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	oauthApp := oauth.NewClientApp(&cfg, store)
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noopPDS := func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return noopPDSClient{}, nil
	}
	return auth.NewHTTPHandlers(oauthApp, craftsky, pool, logger, true /* devMode */, noopPDS)
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
		`{"handle":"not a handle","handoffMode":"deep_link"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "invalid_handle")
}

func TestLogin_InvalidHandoffMode(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoffMode":"wat"}`)
	expectEnvelopeError(t, rr, http.StatusBadRequest, "invalid_handoff_mode")
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
		`{"handle":"alice.example","handoffMode":"deep_link"}`)
	if rr.Code != http.StatusBadGateway {
		// We expect StartAuthFlow to fail because the test fixture has no
		// real PDS; the important assertion is that the request decoded
		// and validation PASSED (otherwise we'd get 400 invalid_handoff_mode).
		t.Fatalf("got %d, want 502 (body decoded, reached StartAuthFlow)", rr.Code)
	}
}

func TestLogin_RejectsSnakeCaseBody(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoff_mode":"deep_link"}`)
	// handoffMode absent -> invalid_handoff_mode 400.
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rr.Code)
	}
}

// Helper: seed an oauth session and a craftsky session, return the bearer token.
func seedSession(t *testing.T, h *auth.HTTPHandlers, did, sid string) string {
	t.Helper()
	ctx := context.Background()
	if _, err := h.Pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ($1, $2, '{}')`,
		did, sid); err != nil {
		t.Fatal(err)
	}
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
	// use handlersFixture's noopPDSClient.
	err := auth.InitializeProfile(context.Background(), erroringGetPDSClient{}, syntax.DID("did:plc:me"))
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Fatalf("want ErrProfileInitFailed; got %v", err)
	}
}
