package routes

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
)

func testDeps() *app.Deps {
	return &app.Deps{
		Config:      app.Config{Env: app.EnvDev, AllowedOrigins: []string{"*"}, DevDID: "did:plc:test"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:test"},
	}
}

func TestAddRoutes_V1WhoAmIAuthenticatedReturnsDID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "did:plc:from-header") {
		t.Errorf("body = %q, want containing 'did:plc:from-header'", rec.Body.String())
	}
}

func TestAddRoutes_V1WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-test")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_V1WhoAmIWithoutDeviceIDReturns400(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_device_id") {
		t.Errorf("body = %q, want containing 'missing_device_id'", rec.Body.String())
	}
}

func TestAddRoutes_LegacyUnprefixedWhoAmIReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (legacy path should be gone)", rec.Code)
	}
}

func TestAddRoutes_HealthStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	// The health handler calls deps.DB.Ping which panics with a nil DB in
	// tests. A panic means the route IS registered (the handler was reached);
	// only a 404 from the fallthrough NotFoundHandler means no route exists.
	func() {
		defer func() { recover() }() //nolint:errcheck
		mux.ServeHTTP(rec, req)
	}()

	if rec.Code == http.StatusNotFound {
		t.Errorf("status = 404; /health must stay unprefixed")
	}
}

func TestAddRoutes_OAuthClientMetadataStaysUnprefixed(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
	rec := httptest.NewRecorder()

	// The OAuth handler may dereference a nil OAuthApp in tests. A panic means
	// the route IS registered; only a 404 means no handler was found.
	func() {
		defer func() { recover() }() //nolint:errcheck
		mux.ServeHTTP(rec, req)
	}()

	if rec.Code == http.StatusNotFound {
		t.Errorf("status = 404; /oauth/client-metadata.json must stay unprefixed")
	}
}

func TestAddRoutes_UnknownPathReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
