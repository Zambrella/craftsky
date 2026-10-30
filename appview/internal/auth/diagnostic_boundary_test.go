package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"strings"
	"testing"
	"time"
)

func TestOAuthCallbackConsumedRequestDoesNotCaptureIssue(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	config := testStoreConfig()
	config.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, config)
	owner := syntax.DID("did:plc:consumed-callback")
	seedAuthOwner(t, pool, owner)
	err := owners.WithOnboardingAuth(context.Background(), owner, func(ctx context.Context, authority ownerlifecycle.Lifecycle) error {
		ctx = auth.WithLoginAuthRequest(ctx, owner, authority.Generation, authority.AuthEpoch,
			"https://pds.example", "https://issuer.example", auth.HandoffVerifiedLink, "private-device", "")
		if err := store.SaveAuthRequestInfo(ctx, oauth.AuthRequestData{
			State: "expired-private-state", RequestURI: "urn:private-request",
		}); err != nil {
			return err
		}
		_, err := store.BeginExchange(ctx, "expired-private-state")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	clientConfig := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App:   &oauth.ClientApp{Config: &clientConfig, Resolver: oauth.NewResolver(), Dir: &callbackDirectory{}},
		Store: store, Owners: owners, AuthorityVerifier: &fakeOAuthAuthorityVerifier{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: flow}
	request := httptest.NewRequest("GET", "/oauth/callback?state=expired-private-state&code=private-code&iss=https://private-issuer", nil)
	response := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.CallbackHandler())).ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != http.StatusBadRequest || len(transport.Events()) != 0 {
		t.Fatalf("stale callback status/issues = %d/%d", response.Code, len(transport.Events()))
	}
	for _, want := range []string{"callback_request_unavailable", "OAuthCallbackDiagnostic", "run_id"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("local diagnostics missing %q", want)
		}
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{"expired-private-state", "private-code", "private-issuer"} {
		if strings.Contains(local.String(), canary) || strings.Contains(string(serialized), canary) || strings.Contains(response.Body.String(), canary) {
			t.Fatalf("protected callback value escaped: %s", canary)
		}
	}
}

func TestAuthDiagnosticRetainsUnderlyingCause(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	flow := &registrationValidationFlow{err: &pgconn.PgError{Code: "40001", Message: "opaque private auth canary"}}
	handlers := &auth.HTTPHandlers{OAuthFlow: flow, Logger: logger}
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(`{"handoffMode":"verified_link"}`))
	request = request.WithContext(middleware.WithDeviceID(request.Context(), "private-device-canary"))
	response := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.RegistrationHandler())).ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != 502 {
		t.Fatalf("status %d", response.Code)
	}
	for _, want := range []string{"pgconn.PgError", "40001", "registration.start"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("local missing %s: %s", want, local.String())
		}
	}
	events := transport.Events()
	if len(events) != 1 || events[0].Exception[0].Type != "*pgconn.PgError" {
		t.Fatalf("cause owner missing: %+v", events)
	}
	for _, private := range []string{"opaque private auth canary", "private-device-canary"} {
		if strings.Contains(local.String(), private) || strings.Contains(response.Body.String(), private) {
			t.Fatal("protected data leaked")
		}
	}
}

// Exercise the actual HTTP diagnostic boundary and serialized sinks with the
// error produced by the real callback service above, including protected values.
func verifyCallbackDiagnosticSink(t *testing.T, cause error) {
	t.Helper()
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: &callbackDiagnosticFlow{cause: cause}}
	request := httptest.NewRequest("GET", "/oauth/callback?state=oauth-state-canary&code=authorization-code-canary&iss=https://private-issuer-canary", nil)
	recorder := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.CallbackHandler())).ServeHTTP(recorder, request)
	observer.Flush(time.Second)
	if recorder.Code != 400 {
		t.Fatalf("diagnostics changed callback status: %d", recorder.Code)
	}
	for _, expected := range []string{`"failure_reason":"scope_required_missing"`, "missing=craftsky", "required=9 granted=1", "alternate_forms=0", `"run_id"`, "OAuthCallbackDiagnostic"} {
		if !strings.Contains(local.String(), expected) {
			t.Fatalf("missing %q in serialized diagnostics: %s", expected, local.String())
		}
	}
	if len(transport.Events()) != 0 {
		t.Fatal("expected scope refusal created a Sentry issue")
	}
	for _, protected := range []string{"oauth-state-canary", "authorization-code-canary", "private-issuer-canary", "old-access-token", "old-refresh-token", "pkce-verifier", "callback-device"} {
		if strings.Contains(local.String(), protected) || strings.Contains(recorder.Body.String(), protected) {
			t.Fatalf("protected callback value %q escaped", protected)
		}
	}
}

type callbackDiagnosticFlow struct {
	registrationValidationFlow
	cause error
}

func (f *callbackDiagnosticFlow) CompleteCallback(context.Context, url.Values, auth.OAuthCallbackFinalizer) error {
	return f.cause
}

func TestRegistrationCancellationDoesNotCaptureIssue(t *testing.T) {
	for _, code := range []auth.RegistrationFailureCode{auth.RegistrationFailureCanceled, auth.RegistrationFailureProviderUnavailable, auth.RegistrationFailureIncomplete} {
		t.Run(string(code), func(t *testing.T) {
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			failure := &auth.TrustedRegistrationFailure{Code: code, Metadata: auth.AuthRequestMetadata{
				Purpose: auth.RegistrationOAuthPurpose, HandoffMode: auth.HandoffVerifiedLink,
				DeviceID: "private-device-canary", RequestState: auth.AuthRequestReady, ExpiresAt: time.Now().Add(time.Minute),
				RegistrationProviderOrigin: "https://provider.example", RegistrationIssuer: "https://issuer.example",
			}}
			handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: &callbackDiagnosticFlow{cause: errors.Join(failure, errors.New("private-provider-canary"))}, LoginCompleteURL: "https://app.craftsky.social/auth/complete"}
			request := httptest.NewRequest("GET", "/oauth/callback?state=private-state-canary&error=access_denied", nil)
			response := httptest.NewRecorder()
			middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.CallbackHandler())).ServeHTTP(response, request)
			observer.Flush(time.Second)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), string(code)) {
				t.Fatalf("callback response changed: %d", response.Code)
			}
			wantEvents, result := 1, "error"
			if code == auth.RegistrationFailureCanceled {
				wantEvents, result = 0, "expected"
			}
			if len(transport.Events()) != wantEvents {
				t.Fatalf("issues = %d, want %d", len(transport.Events()), wantEvents)
			}
			for _, want := range []string{"registration.callback", "TrustedRegistrationFailure", `"result":"` + result + `"`, "run_id"} {
				if !strings.Contains(local.String(), want) {
					t.Fatalf("missing %q in local diagnostics", want)
				}
			}
			serialized, err := json.Marshal(transport.Events())
			if err != nil {
				t.Fatal(err)
			}
			for _, canary := range []string{"private-device-canary", "private-state-canary", "private-provider-canary"} {
				if strings.Contains(local.String(), canary) || strings.Contains(string(serialized), canary) || strings.Contains(response.Body.String(), canary) {
					t.Fatalf("protected value escaped: %s", canary)
				}
			}
		})
	}
}

func TestLoginRecurrenceRetainsStageProviderFactsAndCorrelation(t *testing.T) {
	for _, stage := range []string{"identity_result", "provider_discovery", "provider_metadata", "authorization_request"} {
		t.Run(stage, func(t *testing.T) {
			pool := withAuthSchema(t)
			owners := newAuthOwnerStore(t, pool)
			config := testStoreConfig()
			config.OwnerLifecycles = owners
			store := auth.NewPostgresAuthStore(pool, config)
			owner := syntax.DID("did:plc:login-diagnostic")
			seedAuthOwner(t, pool, owner)
			ident := &identity.Identity{DID: owner, Handle: "alice.example", Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example"}}}
			if stage == "identity_result" {
				ident = nil
			}
			client := &http.Client{Transport: callbackRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				status, body := 200, ""
				switch req.URL.Path {
				case "/.well-known/oauth-protected-resource":
					body = `{"resource":"https://pds.example","authorization_servers":["https://issuer.example"]}`
					if stage == "provider_discovery" {
						status, body = 502, "private-provider-canary"
					}
				case "/.well-known/oauth-authorization-server":
					body = `{"issuer":"https://issuer.example","authorization_endpoint":"https://issuer.example/authorize","token_endpoint":"https://issuer.example/token","response_types_supported":["code"],"grant_types_supported":["authorization_code","refresh_token"],"code_challenge_methods_supported":["S256"],"token_endpoint_auth_methods_supported":["none","private_key_jwt"],"token_endpoint_auth_signing_alg_values_supported":["ES256"],"scopes_supported":["atproto"],"authorization_response_iss_parameter_supported":true,"require_pushed_authorization_requests":true,"pushed_authorization_request_endpoint":"https://issuer.example/par","dpop_signing_alg_values_supported":["ES256"],"client_id_metadata_document_supported":true}`
					if stage == "provider_metadata" {
						status, body = 503, "private-provider-canary"
					}
				case "/par":
					status, body = 400, `{"error":"invalid_client_metadata","error_description":"private-provider-canary"}`
				default:
					t.Fatal("unexpected provider request")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}, Request: req}, nil
			})}
			appConfig := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
			flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{App: &oauth.ClientApp{Config: &appConfig, Client: client, Resolver: &oauth.Resolver{Client: client}, Dir: &callbackDirectory{identity: ident}}, Store: store, Owners: owners, AuthorityVerifier: &fakeOAuthAuthorityVerifier{}})
			if err != nil {
				t.Fatal(err)
			}
			var local bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			previousLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(previousLogger)
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: flow}
			req := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(`{"handle":"alice.example","handoffMode":"verified_link"}`))
			req = req.WithContext(middleware.WithDeviceID(req.Context(), "private-device-canary"))
			rec := httptest.NewRecorder()
			middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.LoginHandler())).ServeHTTP(rec, req)
			observer.Flush(time.Second)
			if rec.Code != 502 {
				t.Fatalf("login response changed: %d", rec.Code)
			}
			events := transport.Events()
			if len(events) != 1 || events[0].Tags["failure_stage"] != stage {
				t.Fatalf("missing or duplicated stage: %+v", events)
			}
			status := map[string]string{"provider_discovery": "502", "provider_metadata": "503", "authorization_request": "400"}[stage]
			if events[0].Tags["upstream_status"] != status {
				t.Fatalf("provider status lost: %v", events[0].Tags)
			}
			if stage == "authorization_request" && events[0].Tags["provider_code"] != "oauth.invalid_client_metadata" {
				t.Fatal("provider code lost")
			}
			requestID := rec.Header().Get("X-Request-ID")
			if requestID == "" || events[0].Contexts["correlation"]["run_id"] != requestID || !strings.Contains(rec.Body.String(), requestID) {
				t.Fatal("request correlation lost")
			}
			payload, err := json.Marshal(events)
			if err != nil {
				t.Fatal(err)
			}
			for _, sink := range []string{local.String(), string(payload)} {
				for _, want := range []string{stage, "OAuthLoginDiagnostic", requestID, status} {
					if want != "" && !strings.Contains(sink, want) {
						t.Fatalf("missing %q", want)
					}
				}
				for _, private := range []string{"private-provider-canary", "private-device-canary", "code_challenge", "request_uri"} {
					if strings.Contains(sink, private) {
						t.Fatalf("private data escaped: %s", private)
					}
				}
			}
		})
	}
}
