package observability

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObserveDBOperationRecordsBoundedComparableTelemetry(t *testing.T) {
	observer := New(Config{Env: "test"})
	errBoom := errors.New("boom")

	if err := observer.ObserveDB(context.Background(), DBOperation{
		Operation:    "search.posts",
		RoutePattern: "/v1/search/posts",
		RunID:        "run-123",
	}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("ObserveDB success: %v", err)
	}
	if err := observer.ObserveDB(context.Background(), DBOperation{
		Operation:    "search.posts",
		RoutePattern: "/v1/search/posts",
		RunID:        "run-456",
	}, func(context.Context) error {
		return errBoom
	}); !errors.Is(err, errBoom) {
		t.Fatalf("ObserveDB error = %v, want boom", err)
	}

	rec := httptest.NewRecorder()
	observer.MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"# HELP craftsky_appview_db_operation_duration_seconds Duration of bounded AppView DB operations in seconds.",
		"# TYPE craftsky_appview_db_operation_duration_seconds histogram",
		`operation="search.posts"`,
		`route_pattern="/v1/search/posts"`,
		`result="success"`,
		`result="error"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"alpaca", "did:plc:", "SELECT "} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metrics output contains raw query/identity/sql value %q:\n%s", forbidden, body)
		}
	}
}
