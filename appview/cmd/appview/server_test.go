package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

type serverStubResolver struct{ handle syntax.Handle }

func (s serverStubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return s.handle, nil
}
func (s serverStubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return "", nil
}

var _ api.HandleResolver = serverStubResolver{}

func TestNewServer_HTTPMetricsUseRoutePattern(t *testing.T) {
	observer := observability.New(observability.Config{Env: "test"})
	deps := &app.Deps{
		Config: app.Config{
			Env:            app.EnvDev,
			AllowedOrigins: []string{"*"},
			DevDID:         "did:plc:test",
		},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:    &auth.MockAuthService{DefaultDID: "did:plc:test"},
		HandleResolver: serverStubResolver{handle: syntax.Handle("stub.example")},
		Observability:  observer,
	}
	handler := NewServer(context.Background(), deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:raw/rkey123?cursor=secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)
	body := metricsRec.Body.String()

	if !strings.Contains(body, `craftsky_appview_http_requests_total`) {
		t.Fatalf("metrics missing HTTP request counter:\n%s", body)
	}
	if !strings.Contains(body, `route_pattern="/v1/posts/{did}/{rkey}"`) {
		t.Fatalf("metrics missing route pattern label:\n%s", body)
	}
	for _, forbidden := range []string{"did:plc:raw", "rkey123", "cursor=secret"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metrics contain raw route/query value %q:\n%s", forbidden, body)
		}
	}
}
