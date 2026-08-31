package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTrustedRegistrationFailureDestinationUsesOnlyStoredMetadata(t *testing.T) {
	now := time.Now()
	verifiedURL := "https://app.craftsky.social/auth/complete"
	attackerValues := url.Values{
		"redirect_uri":          {"https://attacker.example/steal"},
		"loopback_redirect_uri": {"http://127.0.0.1:6553/steal"},
	}
	tests := []struct {
		name      string
		metadata  AuthRequestMetadata
		allowDev  bool
		wantDeep  string
		wantLoop  string
		wantError bool
	}{
		{
			name:     "verified link",
			metadata: trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(time.Minute)),
			wantDeep: verifiedURL + "?error=canceled",
		},
		{
			name:     "loopback",
			metadata: trustedRegistrationMetadata(HandoffLoopback, "http://127.0.0.1:43125/oauth/handoff", now.Add(time.Minute)),
			wantLoop: "http://127.0.0.1:43125/oauth/handoff",
		},
		{
			name:     "development scheme",
			metadata: trustedRegistrationMetadata(HandoffDevScheme, "", now.Add(time.Minute)),
			allowDev: true,
			wantDeep: "craftsky-dev:///auth/complete?error=canceled",
		},
		{
			name:      "expired",
			metadata:  trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(-time.Second)),
			wantError: true,
		},
		{
			name: "consumed",
			metadata: func() AuthRequestMetadata {
				metadata := trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(time.Minute))
				metadata.RequestState = AuthRequestConsumed
				return metadata
			}(),
			wantError: true,
		},
		{
			name: "wrong purpose",
			metadata: AuthRequestMetadata{
				Purpose: LoginOAuthPurpose, HandoffMode: HandoffVerifiedLink,
				DeviceID: "device", RequestState: AuthRequestReady, ExpiresAt: now.Add(time.Minute),
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := trustedRegistrationFailurePageData(
				test.metadata, RegistrationFailureCanceled, verifiedURL, test.allowDev, attackerValues, now,
			)
			if test.wantError {
				if err == nil {
					t.Fatalf("trustedRegistrationFailurePageData accepted metadata: %+v", test.metadata)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if data.DeepLinkURL != test.wantDeep || data.LoopbackURI != test.wantLoop {
				t.Fatalf("destination=(%q,%q), want (%q,%q)", data.DeepLinkURL, data.LoopbackURI, test.wantDeep, test.wantLoop)
			}
			if strings.Contains(data.DeepLinkURL, "attacker") || strings.Contains(data.LoopbackURI, "6553") {
				t.Fatalf("callback-supplied destination was used: %+v", data)
			}
		})
	}
}

func trustedRegistrationMetadata(mode HandoffMode, loopbackURI string, expiresAt time.Time) AuthRequestMetadata {
	return AuthRequestMetadata{
		Purpose: RegistrationOAuthPurpose, HandoffMode: mode, LoopbackURI: loopbackURI,
		DeviceID: "device", RequestState: AuthRequestReady, ExpiresAt: expiresAt,
		RegistrationProviderOrigin: "https://provider.example",
		RegistrationIssuer:         "https://issuer.example",
	}
}

func TestRegistrationFailureCodeSerializesOnlyBoundedValues(t *testing.T) {
	approved := []RegistrationFailureCode{
		RegistrationFailureCanceled,
		RegistrationFailureProviderUnavailable,
		RegistrationFailureIncomplete,
	}
	for _, code := range approved {
		t.Run(string(code), func(t *testing.T) {
			encoded, err := json.Marshal(code)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != `"`+string(code)+`"` {
				t.Fatalf("encoded code=%s", encoded)
			}
			text, err := code.MarshalText()
			if err != nil || string(text) != string(code) {
				t.Fatalf("MarshalText=(%q,%v)", text, err)
			}
		})
	}

	rejected := []RegistrationFailureCode{
		"the provider says this account is blocked",
		"https://attacker.example/?token=secret",
		"canceled\r\nLocation: https://attacker.example",
		RegistrationFailureCode(strings.Repeat("x", 4096)),
		"",
	}
	for index, code := range rejected {
		t.Run(fmt.Sprintf("rejected-%d", index), func(t *testing.T) {
			if encoded, err := json.Marshal(code); err == nil {
				t.Fatalf("json.Marshal(%q)=%s, want error", code, encoded)
			}
			if text, err := code.MarshalText(); err == nil {
				t.Fatalf("MarshalText(%q)=%q, want error", code, text)
			}
		})
	}

	cause := errors.New("provider secret: authorization_code=secret")
	failure, err := newTrustedRegistrationFailure(
		trustedRegistrationMetadata(HandoffVerifiedLink, "", time.Now().Add(time.Minute)),
		RegistrationFailureCanceled,
		cause,
	)
	if err != nil {
		t.Fatal(err)
	}
	if failure.Error() != "registration could not be completed" || strings.Contains(failure.Error(), "secret") {
		t.Fatalf("trusted failure Error()=%q", failure.Error())
	}
	if !errors.Is(failure, cause) {
		t.Fatal("trusted failure did not retain its internal cause")
	}
}

func TestTrustedRegistrationFailuresReturnBoundedErrorsForEveryHandoffMode(t *testing.T) {
	technicalCause := errors.New("provider description: token=secret https://internal.example")
	tests := []struct {
		name     string
		mode     HandoffMode
		loopback string
		allowDev bool
		code     RegistrationFailureCode
		cause    error
		want     string
	}{
		{"verified denial", HandoffVerifiedLink, "", false, RegistrationFailureCanceled, errors.New("access_denied: user said no"), "https://app.craftsky.social/auth/complete?error=canceled"},
		{"verified technical", HandoffVerifiedLink, "", false, RegistrationFailureProviderUnavailable, technicalCause, "error=providerUnavailable"},
		{"loopback denial", HandoffLoopback, "http://127.0.0.1:43125/oauth/handoff", false, RegistrationFailureCanceled, errors.New("access_denied"), `JSON.stringify({error: "canceled"})`},
		{"loopback technical", HandoffLoopback, "http://127.0.0.1:43125/oauth/handoff", false, RegistrationFailureIncomplete, technicalCause, `JSON.stringify({error: "registrationIncomplete"})`},
		{"development denial", HandoffDevScheme, "", true, RegistrationFailureCanceled, errors.New("access_denied"), "craftsky-dev:///auth/complete?error=canceled"},
		{"development technical", HandoffDevScheme, "", true, RegistrationFailureIncomplete, technicalCause, "error=registrationIncomplete"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := trustedRegistrationMetadata(test.mode, test.loopback, time.Now().Add(time.Minute))
			failure, err := newTrustedRegistrationFailure(metadata, test.code, test.cause)
			if err != nil {
				t.Fatal(err)
			}
			handlers := &HTTPHandlers{
				OAuthFlow:        &recordingDeletionOAuthFlow{err: failure},
				LoginCompleteURL: "https://app.craftsky.social/auth/complete",
				AllowDevScheme:   test.allowDev,
			}
			request := httptest.NewRequest(
				http.MethodGet,
				"/oauth/callback?state=trusted&redirect_uri=https%3A%2F%2Fattacker.example%2Fsteal&code=secret-code&error_description=provider-secret",
				nil,
			)
			response := httptest.NewRecorder()
			handlers.CallbackHandler().ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			body := response.Body.String()
			if !strings.Contains(body, test.want) {
				t.Fatalf("body does not contain %q: %s", test.want, body)
			}
			for _, forbidden := range []string{"attacker.example", "secret-code", "provider-secret", "token=secret", "internal.example", "access_denied"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("body exposed %q: %s", forbidden, body)
				}
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("Cache-Control=%q", response.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestUntrustedCallbackStateReturnsOnlyGenericLockedDownHTML(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		path string
		err  error
	}{
		{"missing", "/oauth/callback?redirect_uri=https%3A%2F%2Fattacker.example%2Fsteal", ErrOAuthFlowInvalid},
		{"malformed", "/oauth/callback?state=%00%0D%0ALocation%3Ahttps%3A%2F%2Fattacker.example", ErrOAuthFlowInvalid},
		{"unknown", "/oauth/callback?state=unknown&loopback_redirect_uri=http%3A%2F%2F127.0.0.1%3A43125%2Fsteal", ErrOAuthSessionNotFound},
		{"expired", "/oauth/callback?state=expired&redirect_uri=https%3A%2F%2Fattacker.example", func() error {
			_, err := newTrustedRegistrationFailure(
				trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(-time.Second)),
				RegistrationFailureIncomplete,
				errors.New("expired"),
			)
			return err
		}()},
		{"consumed", "/oauth/callback?state=consumed&redirect_uri=https%3A%2F%2Fattacker.example", func() error {
			metadata := trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(time.Minute))
			metadata.RequestState = AuthRequestConsumed
			_, err := newTrustedRegistrationFailure(metadata, RegistrationFailureIncomplete, errors.New("consumed"))
			return err
		}()},
		{"wrong purpose", "/oauth/callback?state=deletion&redirect_uri=https%3A%2F%2Fattacker.example", func() error {
			metadata := trustedRegistrationMetadata(HandoffVerifiedLink, "", now.Add(time.Minute))
			metadata.Purpose = AccountDeletionOAuthPurpose
			_, err := newTrustedRegistrationFailure(metadata, RegistrationFailureIncomplete, errors.New("wrong purpose"))
			return err
		}()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.err == nil {
				t.Fatal("untrusted fixture unexpectedly constructed a trusted failure")
			}
			handlers := &HTTPHandlers{
				OAuthFlow:        &recordingDeletionOAuthFlow{err: test.err},
				LoginCompleteURL: "https://app.craftsky.social/auth/complete",
				AllowDevScheme:   true,
			}
			response := httptest.NewRecorder()
			handlers.CallbackHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

			if response.Code != http.StatusBadRequest || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}
			csp := response.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "default-src 'none'") ||
				!strings.Contains(csp, "script-src 'none'") ||
				!strings.Contains(csp, "connect-src 'none'") {
				t.Fatalf("CSP=%q", csp)
			}
			body := response.Body.String()
			if !strings.Contains(body, "Sign-in could not be completed") {
				t.Fatalf("generic body missing: %s", body)
			}
			for _, forbidden := range []string{"attacker.example", "127.0.0.1", "craftsky-dev:", "app.craftsky.social", "window.location", "fetch("} {
				if strings.Contains(body, forbidden) || strings.Contains(response.Header().Get("Location"), forbidden) {
					t.Fatalf("untrusted callback exposed destination %q: headers=%v body=%s", forbidden, response.Header(), body)
				}
			}
		})
	}
}

func TestCallbackRenderingCarriesOnlyCodeAndSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := renderCallbackHTML(recorder, callbackPageData{
		Code:        "short-lived-code",
		LoopbackURI: "http://127.0.0.1:43125/oauth/handoff",
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"Cache-Control":          "no-store",
		"Pragma":                 "no-cache",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := recorder.Header().Get(key); got != want {
			t.Fatalf("%s=%q, want %q", key, got, want)
		}
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'nonce-") ||
		!strings.Contains(csp, "frame-ancestors 'none'") ||
		!strings.Contains(csp, "connect-src http://127.0.0.1:43125") {
		t.Fatalf("CSP=%q", csp)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `JSON.stringify({code:`) || !strings.Contains(body, "short-lived-code") {
		t.Fatalf("callback body does not post code: %s", body)
	}
	if strings.Contains(body, "Dev-mode token") || strings.Contains(body, `JSON.stringify({token:`) {
		t.Fatalf("callback retained bearer handoff markup: %s", body)
	}
}

func TestCallbackRenderingAllowsOnlyExactLoopbackOrigin(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := renderCallbackHTML(recorder, callbackPageData{
		Code:        "short-lived-code",
		LoopbackURI: "http://127.0.0.1:43125/oauth/handoff/nested",
	})
	if err != nil {
		t.Fatal(err)
	}
	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src http://127.0.0.1:43125") {
		t.Fatalf("CSP=%q does not allow the exact listener origin", csp)
	}
	if strings.Contains(csp, "/oauth/handoff") || strings.Contains(csp, "connect-src *") {
		t.Fatalf("CSP=%q grants more than the exact listener origin", csp)
	}
}

func TestLoopbackCallbackPagePostsOneCodeOnlyPayloadToRealListener(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on loopback: %v", err)
	}
	var calls atomic.Int32
	received := make(chan map[string]any, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		defer request.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid", http.StatusBadRequest)
			return
		}
		received <- payload
		w.WriteHeader(http.StatusNoContent)
	})}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		<-serveDone
	})

	loopbackURI := "http://" + listener.Addr().String() + "/oauth/handoff"
	recorder := httptest.NewRecorder()
	if err := renderCallbackHTML(recorder, callbackPageData{
		Code: "single-use-handoff-code", LoopbackURI: loopbackURI,
	}); err != nil {
		t.Fatal(err)
	}
	origin := "http://" + listener.Addr().String()
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "connect-src "+origin) {
		t.Fatalf("CSP=%q does not authorize the exact live listener origin %q", csp, origin)
	}

	// Execute the callback page's generated fetch contract against the real
	// listener. The browser-specific release smoke remains separate; this
	// proves the rendered page and listener agree on method, URL, and code-only
	// payload without granting or transmitting a CraftSky bearer.
	requestBody := strings.NewReader(`{"code":"single-use-handoff-code"}`)
	request, err := http.NewRequest(http.MethodPost, loopbackURI, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post rendered loopback handoff: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("loopback listener status=%d", response.StatusCode)
	}
	select {
	case payload := <-received:
		if len(payload) != 1 || payload["code"] != "single-use-handoff-code" {
			t.Fatalf("loopback payload=%v, want one code field", payload)
		}
		if _, exists := payload["token"]; exists {
			t.Fatalf("loopback payload exposed a bearer: %v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("loopback listener did not receive the callback payload")
	}
	if calls.Load() != 1 {
		t.Fatalf("loopback listener calls=%d, want one", calls.Load())
	}
}

func TestVerifiedLinkAndErrorPagesDenyAllConnections(t *testing.T) {
	verified := httptest.NewRecorder()
	if err := renderCallbackHTML(verified, callbackPageData{
		Code:        "short-lived-code",
		DeepLinkURL: "https://app.craftsky.social/auth/complete?code=short-lived-code",
	}); err != nil {
		t.Fatal(err)
	}
	if csp := verified.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "connect-src 'none'") {
		t.Fatalf("verified-link CSP=%q, want connections denied", csp)
	}

	failure := httptest.NewRecorder()
	renderErrorHTML(failure, http.StatusBadRequest, "Try again")
	if csp := failure.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "connect-src 'none'") {
		t.Fatalf("error CSP=%q, want connections denied", csp)
	}
}

func TestCallbackErrorRenderingUsesSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	renderErrorHTML(recorder, http.StatusBadRequest, "Try again")
	if recorder.Header().Get("Cache-Control") != "no-store" ||
		recorder.Header().Get("Referrer-Policy") != "no-referrer" ||
		!strings.Contains(recorder.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatalf("error callback headers=%v", recorder.Header())
	}
}
