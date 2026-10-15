package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type recentFailureReader struct{}

func (recentFailureReader) ListRecentMembers(context.Context, syntax.DID) ([]api.RecentMemberRow, error) {
	return nil, &pgconn.PgError{Code: "XX000", Message: "private-member-canary private-token-canary private-preferences-canary"}
}

// IT-006: serialized sinks preserve operation/native cause and exclude payloads.
func TestRecentMembersFailureDiagnostics(t *testing.T) {
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ctx := middleware.WithDID(t.Context(), recentDID("v"))
	ctx = observability.WithRequestDiagnosticContext(ctx, observability.RequestDiagnosticContext{Path: "/v1/profiles/recent", RoutePattern: "/v1/profiles/recent", PublicTargets: false})
	ctx = observability.WithRequestObserver(ctx, observer)
	req := httptest.NewRequest("GET", "/v1/profiles/recent", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	api.GetRecentMembersHandler(recentFailureReader{}).ServeHTTP(rec, req)
	if rec.Code != 500 {
		t.Fatalf("status = %d", rec.Code)
	}
	if !observer.Flush(time.Second) {
		t.Fatal("diagnostics flush failed")
	}
	events, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	if local.Len() == 0 || len(transport.Events()) != 1 {
		t.Fatalf("missing or duplicated failure owner: %s / %s", local.String(), events)
	}
	for _, sink := range []string{local.String(), string(events)} {
		if !strings.Contains(sink, "api.GetRecentMembersHandler") || !strings.Contains(sink, "PgError") {
			t.Fatalf("missing permitted operation/native cause: %s", sink)
		}
		for _, canary := range []string{"private-member-canary", "private-token-canary", "private-preferences-canary"} {
			if strings.Contains(sink, canary) {
				t.Fatalf("protected value leaked: %s", canary)
			}
		}
	}
}
