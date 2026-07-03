package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{Env: "test", MetricRecorder: recorder})
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

	calls := recorder.Calls()
	for _, call := range calls {
		if call.Name != "craftsky_appview_http_requests_total" {
			continue
		}
		if call.Attributes["route_pattern"] != "/v1/posts/{did}/{rkey}" {
			t.Fatalf("route_pattern = %q, want /v1/posts/{did}/{rkey}; call=%#v", call.Attributes["route_pattern"], call)
		}
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("HTTP metric call failed validation: %v; call=%#v", err, call)
		}
		return
	}
	t.Fatalf("missing HTTP request counter call: %#v", calls)
}
