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
	// Check registration via mux.Handler (which returns the matched
	// pattern), not ServeHTTP — the handler would panic on the nil DB
	// in these fixtures, and we only care whether the route exists.
	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)

	devMux := http.NewServeMux()
	AddRoutes(context.Background(), devMux, testDepsEnv(app.EnvDev))
	if _, pattern := devMux.Handler(req); pattern != "GET /test/feed" {
		t.Errorf("dev: expected GET /test/feed route, got pattern %q", pattern)
	}

	prodMux := http.NewServeMux()
	AddRoutes(context.Background(), prodMux, testDepsEnv(app.EnvProd))
	if _, pattern := prodMux.Handler(req); pattern == "GET /test/feed" {
		t.Error("prod: /test/feed should NOT be registered")
	}
}
