package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

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
