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

func testDeps() *app.Deps { return testDepsEnv(app.EnvDev) }

func testDepsEnv(env app.Env) *app.Deps {
	return &app.Deps{
		Config:      app.Config{Env: env, AllowedOrigins: []string{"*"}, DevDID: "did:plc:test"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:test"},
	}
}

func TestAddRoutes_WhoAmIAuthenticatedReturnsDID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "did:plc:from-header") {
		t.Errorf("body = %q, want containing 'did:plc:from-header'", rec.Body.String())
	}
}

func TestAddRoutes_WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
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

func TestAddRoutes_TestFeedDevOnly(t *testing.T) {
	// Dev: route registered.
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDepsEnv(app.EnvDev))
	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	func() {
		// Handler panics on nil DB; we only care that it was reached
		// (i.e. the route is registered, not falling through to 404).
		defer func() { _ = recover() }()
		mux.ServeHTTP(rec, req)
	}()
	if rec.Code == http.StatusNotFound {
		t.Error("dev: /test/feed should be registered")
	}

	// Prod: route NOT registered.
	mux = http.NewServeMux()
	AddRoutes(context.Background(), mux, testDepsEnv(app.EnvProd))
	req = httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("prod: /test/feed should be 404, got %d", rec.Code)
	}
}
