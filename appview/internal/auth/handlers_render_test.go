package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
		!strings.Contains(csp, "frame-ancestors 'none'") {
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

func TestCallbackErrorRenderingUsesSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	renderErrorHTML(recorder, http.StatusBadRequest, "Try again")
	if recorder.Header().Get("Cache-Control") != "no-store" ||
		recorder.Header().Get("Referrer-Policy") != "no-referrer" ||
		!strings.Contains(recorder.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatalf("error callback headers=%v", recorder.Header())
	}
}
