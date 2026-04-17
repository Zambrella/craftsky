package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newUnreachablePool returns a pool whose Ping fails quickly. Used to test
// the 503 path without needing a live DB.
func newUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// 127.0.0.1:1 is a reserved port; connect will refuse immediately.
	cfg, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return pool
}

func TestHealth_ReturnsServiceUnavailableWhenDBDown(t *testing.T) {
	pool := newUnreachablePool(t)
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := HealthHandler(pool, logger)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "db unreachable") {
		t.Errorf("body = %q, want 'db unreachable'", rec.Body.String())
	}
}
