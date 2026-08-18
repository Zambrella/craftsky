package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS_AllowsListedOrigin(t *testing.T) {
	handler := CORS([]string{"https://a.example"}, testCORSMethods{"/": {http.MethodGet}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := CORS([]string{"https://a.example"}, testCORSMethods{"/": {http.MethodGet}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := CORS([]string{"*"}, testCORSMethods{"/": {http.MethodGet}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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
	handler := CORS([]string{"https://a.example"}, testCORSMethods{"/": {http.MethodGet}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://a.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Error("next handler should not be called for OPTIONS preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight should set Access-Control-Allow-Methods")
	}
}

func TestCORS_PreflightAllowsCraftskyHeadersWithoutCredentials(t *testing.T) {
	handler := CORS([]string{"https://app.craftsky.social"}, testCORSMethods{"/v1/whoami": {http.MethodGet}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/whoami", nil)
	req.Header.Set("Origin", "https://app.craftsky.social")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-Craftsky-Device-Id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.craftsky.social" {
		t.Fatalf("ACAO = %q, want app origin", got)
	}
	allowedHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, want := range []string{"Authorization", "Content-Type", "X-Craftsky-Device-Id"} {
		if !containsHeaderToken(allowedHeaders, want) {
			t.Fatalf("Access-Control-Allow-Headers = %q, missing %s", allowedHeaders, want)
		}
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func TestCORS_PreflightDerivesPatchAndRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	catalogue := testCORSMethods{
		"/v1/settings": {http.MethodGet, http.MethodPatch},
	}
	handler := CORS([]string{"https://app.craftsky.social"}, catalogue)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight reached application handler")
	}))

	tests := []struct {
		name    string
		path    string
		origin  string
		method  string
		headers string
		want    int
	}{
		{name: "PATCH allowed", path: "/v1/settings", origin: "https://app.craftsky.social", method: http.MethodPatch, headers: "Authorization, Content-Type", want: http.StatusNoContent},
		{name: "unknown path", path: "/v1/unknown", origin: "https://app.craftsky.social", method: http.MethodPatch, want: http.StatusNotFound},
		{name: "wrong method", path: "/v1/settings", origin: "https://app.craftsky.social", method: http.MethodDelete, want: http.StatusMethodNotAllowed},
		{name: "disallowed origin", path: "/v1/settings", origin: "https://evil.example", method: http.MethodPatch, want: http.StatusForbidden},
		{name: "disallowed header", path: "/v1/settings", origin: "https://app.craftsky.social", method: http.MethodPatch, headers: "X-Secret", want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", tt.method)
			if tt.headers != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.headers)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.want, rec.Body.String())
			}
			vary := strings.Join(rec.Header().Values("Vary"), ",")
			for _, want := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
				if !containsHeaderToken(vary, want) {
					t.Fatalf("Vary = %q, missing %s", vary, want)
				}
			}
			if tt.want == http.StatusNoContent {
				if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, PATCH, OPTIONS" {
					t.Fatalf("Access-Control-Allow-Methods = %q", got)
				}
			}
		})
	}
}

type testCORSMethods map[string][]string

func (c testCORSMethods) AllowedMethods(path string) ([]string, bool) {
	methods, ok := c[path]
	return methods, ok
}

func containsHeaderToken(header, want string) bool {
	for _, token := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(token), want) {
			return true
		}
	}
	return false
}
