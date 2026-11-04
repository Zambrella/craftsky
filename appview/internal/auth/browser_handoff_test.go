package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/testlog"
)

func browserTestState() string { return base64.RawURLEncoding.EncodeToString(make([]byte, 32)) }
func browserTestURI(origin, path string) string {
	return origin + path + "?state=" + browserTestState()
}

func TestBrowserCompletionTargetsAreExactOriginPathAndIntentBound(t *testing.T) {
	valid := browserTestURI(DefaultWebAppOrigin, BrowserLoginCompletionPath)
	if err := ValidateBrowserCompletionURI(valid, DefaultWebAppOrigin, BrowserLoginCompletionPath); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		strings.Replace(valid, "app.craftsky.social", "evil.example", 1),
		strings.Replace(valid, "https://", "http://", 1),
		strings.Replace(valid, "app.craftsky.social", "app.craftsky.social.evil.example", 1),
		strings.Replace(valid, "app.craftsky.social", "name@app.craftsky.social", 1),
		strings.Replace(valid, BrowserLoginCompletionPath, "/auth/complete", 1),
		strings.Replace(valid, BrowserLoginCompletionPath, BrowserDeletionCompletionPath, 1),
		strings.Replace(valid, BrowserLoginCompletionPath, "/%61uth/browser/complete", 1),
		valid + "&next=https://evil.example", valid + "&state=" + browserTestState(), valid + "#fragment",
		DefaultWebAppOrigin + BrowserLoginCompletionPath,
		DefaultWebAppOrigin + BrowserLoginCompletionPath + "?state=short",
	} {
		if err := ValidateBrowserCompletionURI(raw, DefaultWebAppOrigin, BrowserLoginCompletionPath); err == nil {
			t.Errorf("unsafe callback accepted: %q", raw)
		}
	}
	local := browserTestURI("http://127.0.0.1:25319", BrowserLoginCompletionPath)
	if err := ValidateBrowserCompletionURI(local, "http://127.0.0.1:25319", BrowserLoginCompletionPath); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBrowserCompletionURI(local, "http://127.0.0.1:25320", BrowserLoginCompletionPath); err == nil {
		t.Fatal("different worktree origin accepted")
	}
}

func TestWebAppOriginAllowsOnlyControlledProductionOrExplicitDevelopmentLoopback(t *testing.T) {
	for _, raw := range []string{"", DefaultWebAppOrigin, DefaultWebAppOrigin + "/", PreviewWebAppOrigin, PreviewWebAppOrigin + "/"} {
		if _, err := ValidateWebAppOrigin(raw, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"https://preview.example", "http://127.0.0.1:25319", "https://app.craftsky.social/path", "https://app.craftsky.social?query=1", "https://user@app.craftsky.social", "https://app.craftsky.social:444"} {
		if _, err := ValidateWebAppOrigin(raw, false); err == nil {
			t.Fatalf("production origin accepted: %q", raw)
		}
	}
	for _, raw := range []string{"http://localhost:25319", "http://127.0.0.2:25319", "http://127.0.0.1", "http://127.0.0.1:0", "http://127.0.0.1:65536", "http://127.0.0.1:25319/path"} {
		if _, err := ValidateWebAppOrigin(raw, true); err == nil {
			t.Fatalf("development origin accepted: %q", raw)
		}
	}
	if got, err := ValidateWebAppOrigin("http://127.0.0.1:25319", true); err != nil || got != "http://127.0.0.1:25319" {
		t.Fatal(got, err)
	}
}

func TestPreviewBrowserCompletionAllowsOnlyTheStableOwnedAlias(t *testing.T) {
	for _, path := range []string{BrowserLoginCompletionPath, BrowserDeletionCompletionPath} {
		target := browserTestURI(PreviewWebAppOrigin, path)
		if err := ValidateBrowserCompletionURI(target, DefaultWebAppOrigin, path); err != nil {
			t.Fatal("production rejected the controlled preview callback", err)
		}
		if !validStoredBrowserCompletion(target, path) {
			t.Fatal("preview callback cannot survive stored metadata validation")
		}
		location, err := browserCompletionURL(target, DefaultWebAppOrigin, path, url.Values{"code": {"SINGLE_USE_CANARY"}})
		if err != nil || !strings.HasPrefix(location, PreviewWebAppOrigin+path+"?") {
			t.Fatal("callback did not retain its preview origin", err)
		}
		for _, origin := range []string{
			"http://preview-craftsky-public-app.hugetoothstudios.workers.dev",
			"https://12345678-craftsky-public-app.hugetoothstudios.workers.dev",
			"https://other-craftsky-public-app.hugetoothstudios.workers.dev",
			"https://preview-craftsky-public-app.attacker.workers.dev",
			PreviewWebAppOrigin + ".evil.example",
			PreviewWebAppOrigin + ":443",
		} {
			if ValidateBrowserCompletionURI(browserTestURI(origin, path), DefaultWebAppOrigin, path) == nil {
				t.Fatalf("untrusted preview callback accepted: %s", origin)
			}
		}
		if ValidateBrowserCompletionURI(target, "http://127.0.0.1:25319", path) == nil {
			t.Fatal("preview callback crossed into a development worktree")
		}
	}
}

func TestPreviewLoginAndRegistrationReachTheExistingBrowserFlow(t *testing.T) {
	for _, registration := range []bool{false, true} {
		flow := &browserFlowRecorder{}
		handler := &HTTPHandlers{Logger: testlog.Discard(), OAuthFlow: flow, WebAppOrigin: DefaultWebAppOrigin}
		target := browserTestURI(PreviewWebAppOrigin, BrowserLoginCompletionPath)
		payload := map[string]string{"handoffMode": "browser", "browserRedirectUri": target}
		if !registration {
			payload["handle"] = "alice.example"
		}
		body, _ := json.Marshal(payload)
		request := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(string(body)))
		request = request.WithContext(ctxkeys.WithDeviceID(request.Context(), "browser-device"))
		response := httptest.NewRecorder()
		if registration {
			handler.RegistrationHandler().ServeHTTP(response, request)
		} else {
			handler.LoginHandler().ServeHTTP(response, request)
		}
		if response.Code != 200 || flow.calls != 1 || flow.target != target || flow.mode != HandoffBrowser {
			t.Fatalf("preview browser start failed: status=%d calls=%d", response.Code, flow.calls)
		}
	}
}

type browserFlowRecorder struct {
	mode    HandoffMode
	target  string
	calls   int
	result  OAuthCallbackResult
	failure error
}

func (s *browserFlowRecorder) StartLogin(_ context.Context, _ syntax.Handle, mode HandoffMode, target, device string) (string, error) {
	s.calls++
	s.mode = mode
	s.target = target
	return "https://provider.example/authorize", nil
}
func (s *browserFlowRecorder) StartRegistration(_ context.Context, mode HandoffMode, target, device string) (string, error) {
	s.calls++
	s.mode = mode
	s.target = target
	return "https://provider.example/register", nil
}
func (s *browserFlowRecorder) CompleteCallback(ctx context.Context, _ url.Values, finish OAuthCallbackFinalizer) error {
	if s.failure != nil {
		return s.failure
	}
	return finish(ctx, s.result)
}

type browserHandoffRecorder struct{}

func (browserHandoffRecorder) CreateExchange(context.Context, CallbackAttempt, syntax.Handle, string) (string, error) {
	return "SINGLE_USE_CANARY", nil
}
func (browserHandoffRecorder) Exchange(context.Context, string, string) (HandoffExchangeResult, error) {
	return HandoffExchangeResult{}, errors.New("not used")
}
func (browserHandoffRecorder) Confirm(context.Context, string, uuid.UUID, string) error {
	return errors.New("not used")
}

func TestBrowserLoginAndRegistrationPersistOnlyReviewedDestination(t *testing.T) {
	for _, registration := range []bool{false, true} {
		for _, valid := range []bool{false, true} {
			flow := &browserFlowRecorder{}
			origin := "http://127.0.0.1:25319"
			handler := &HTTPHandlers{Logger: testlog.Discard(), OAuthFlow: flow, WebAppOrigin: origin}
			target := browserTestURI(origin, BrowserLoginCompletionPath)
			if !valid {
				target = strings.Replace(target, "25319", "25320", 1)
			}
			body := map[string]string{"handoffMode": "browser", "browserRedirectUri": target}
			if !registration {
				body["handle"] = "alice.example"
			}
			raw, _ := json.Marshal(body)
			request := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(string(raw)))
			request = request.WithContext(ctxkeys.WithDeviceID(request.Context(), "browser-device"))
			response := httptest.NewRecorder()
			if registration {
				handler.RegistrationHandler().ServeHTTP(response, request)
			} else {
				handler.LoginHandler().ServeHTTP(response, request)
			}
			if valid {
				if response.Code != 200 || flow.calls != 1 || flow.mode != HandoffBrowser || flow.target != target {
					t.Fatalf("browser start failed: %d calls=%d", response.Code, flow.calls)
				}
			} else if response.Code != 400 || flow.calls != 0 {
				t.Fatalf("invalid browser start reached provider: %d calls=%d", response.Code, flow.calls)
			}
		}
	}
}

func TestBrowserOAuthCallbackRedirectPreservesIntentWithoutMobileHandoffOrBodyCapability(t *testing.T) {
	origin := "http://127.0.0.1:25319"
	target := browserTestURI(origin, BrowserLoginCompletionPath)
	flow := &browserFlowRecorder{result: OAuthCallbackResult{Handle: "alice.example", Metadata: AuthRequestMetadata{Purpose: ChatOAuthPurpose, HandoffMode: HandoffBrowser, LoopbackURI: target, DeviceID: "browser-device"}}}
	handlers := &HTTPHandlers{Logger: testlog.Discard(), OAuthFlow: flow, Handoffs: browserHandoffRecorder{}, WebAppOrigin: origin}
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest("GET", "/oauth/callback?code=PROVIDER_CODE&state=PROVIDER_STATE", nil))
	if response.Code != 303 || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("callback status/headers: %d %v", response.Code, response.Header())
	}
	destination, err := url.Parse(response.Header().Get("Location"))
	if err != nil || destination.Path != BrowserLoginCompletionPath || destination.Query().Get("state") != browserTestState() || destination.Query().Get("code") != "SINGLE_USE_CANARY" {
		t.Fatal("wrong browser completion")
	}
	if strings.Contains(response.Body.String(), "CANARY") || strings.Contains(response.Header().Get("Location"), "PROVIDER_") {
		t.Fatal("callback capability reflected or upstream state forwarded")
	}
	handlers.WebAppOrigin = "http://127.0.0.1:25320"
	response = httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest("GET", "/oauth/callback", nil))
	if response.Code != 400 || response.Header().Get("Location") != "" {
		t.Fatal("changed callback origin did not fail closed")
	}
}

func TestBrowserRegistrationFailureKeepsOnlyReviewedErrorAndIntent(t *testing.T) {
	metadata := AuthRequestMetadata{Purpose: RegistrationOAuthPurpose, HandoffMode: HandoffBrowser, LoopbackURI: browserTestURI(DefaultWebAppOrigin, BrowserLoginCompletionPath), DeviceID: "browser-device", RegistrationProviderOrigin: "https://provider.example", RegistrationIssuer: "https://provider.example", RequestState: AuthRequestReady, ExpiresAt: time.Now().Add(time.Minute)}
	flow := &browserFlowRecorder{failure: &TrustedRegistrationFailure{Metadata: metadata, Code: RegistrationFailureCanceled}}
	handlers := &HTTPHandlers{Logger: testlog.Discard(), OAuthFlow: flow, WebAppOrigin: DefaultWebAppOrigin}
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest("GET", "/oauth/callback?error_description=PRIVATE_CANARY", nil))
	if response.Code != 303 || strings.Contains(response.Header().Get("Location"), "PRIVATE_CANARY") || !strings.Contains(response.Header().Get("Location"), "error=canceled") {
		t.Fatalf("failure callback=%d %s", response.Code, response.Header().Get("Location"))
	}
}

func TestBrowserDeletionCallbackHasDedicatedTargetAndDoesNotIssueOrdinaryHandoff(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	job := uuid.New()
	origin := "http://127.0.0.1:25319"
	callbacks := &recordingDeletionOAuthCallbacks{metadata: AccountDeletionAuthRequest{Purpose: AccountDeletionOAuthPurpose, Owner: owner, JobID: job.String()}}
	flow := &browserFlowRecorder{result: OAuthCallbackResult{Metadata: AuthRequestMetadata{Purpose: AccountDeletionOAuthPurpose, Owner: owner, JobID: job, HandoffMode: HandoffBrowser, LoopbackURI: browserTestURI(origin, BrowserDeletionCompletionPath)}, Attempt: CallbackAttempt{Owner: owner, State: "fresh-deletion-session", Purpose: AccountDeletionOAuthPurpose}}}
	handlers := &HTTPHandlers{Logger: testlog.Discard(), OAuthFlow: flow, DeletionOAuthCallbacks: callbacks, WebAppOrigin: origin}
	response := httptest.NewRecorder()
	handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest("GET", "/oauth/callback", nil))
	destination, err := url.Parse(response.Header().Get("Location"))
	if response.Code != 303 || err != nil || destination.Path != BrowserDeletionCompletionPath || destination.Query().Get("state") != browserTestState() || destination.Query().Get("job-id") != job.String() || destination.Query().Get("proof") != "proof-secret" || destination.Query().Has("code") {
		t.Fatalf("browser deletion completion invalid: status=%d", response.Code)
	}
	if response.Body.Len() != 0 || response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("deletion proof must not be reflected into content")
	}
}

type browserChatAuthorizationRecorder struct {
	calls          int
	info           AuthInfo
	mode           HandoffMode
	target, device string
}

func (recorder *browserChatAuthorizationRecorder) StartChatAuthorization(_ context.Context, info AuthInfo, _ string, mode HandoffMode, target, device string, _ *bool) (string, error) {
	recorder.calls++
	recorder.info, recorder.mode, recorder.target, recorder.device = info, mode, target, device
	return "https://provider.example/authorize", nil
}

func TestBrowserChatAuthorizationUsesExactDestinationAndAuthenticatedOwner(t *testing.T) {
	origin := "http://127.0.0.1:25319"
	for _, valid := range []bool{true, false} {
		recorder := &browserChatAuthorizationRecorder{}
		target := browserTestURI(origin, BrowserLoginCompletionPath)
		if !valid {
			target = strings.Replace(target, "25319", "25320", 1)
		}
		body, _ := json.Marshal(map[string]any{"handoffMode": "browser", "browserRedirectUri": target, "pushEnabled": false})
		request := httptest.NewRequest("POST", "/v1/chat/authorization", strings.NewReader(string(body)))
		request.Header.Set("Authorization", "Bearer synthetic-bearer")
		ctx := ctxkeys.WithDID(request.Context(), "did:plc:authenticated-owner")
		ctx = ctxkeys.WithOAuthSessionID(ctx, "synthetic-parent")
		ctx = ctxkeys.WithDeviceID(ctx, "synthetic-device")
		ctx = ctxkeys.WithChildTokenHash(ctx, sha256.Sum256([]byte("synthetic-bearer")))
		response := httptest.NewRecorder()
		ChatAuthorizationHandlerWithBrowser(recorder, false, origin, testlog.Discard()).ServeHTTP(response, request.WithContext(ctx))
		if valid {
			if response.Code != 200 || recorder.calls != 1 || recorder.info.DID != "did:plc:authenticated-owner" || recorder.info.SessionID != "synthetic-parent" || recorder.mode != HandoffBrowser || recorder.target != target || recorder.device != "synthetic-device" {
				t.Fatal("browser messaging authorization lost destination or authenticated authority")
			}
		} else if response.Code != 400 || recorder.calls != 0 {
			t.Fatal("unbound browser messaging destination reached OAuth")
		}
	}
}
