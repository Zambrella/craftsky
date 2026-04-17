package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsListedOrigin(t *testing.T) {
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://a.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example" {
		t.Errorf("ACAO = %q, want https://a.example", got)
	}
}

func TestCORS_BlocksUnlistedOrigin(t *testing.T) {
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty for unlisted origin", got)
	}
}

func TestCORS_WildcardAllowsAny(t *testing.T) {
	handler := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://random.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://random.example" {
		t.Errorf("ACAO = %q, want echoed origin under wildcard", got)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	var nextCalled bool
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://a.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Error("next handler should not be called for OPTIONS preflight")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight should set Access-Control-Allow-Methods")
	}
}
