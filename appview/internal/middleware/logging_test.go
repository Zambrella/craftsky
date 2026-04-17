package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_InjectsRunIDAndLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var seenRunID string
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRunID = GetRunID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seenRunID == "" {
		t.Fatal("handler did not see a run_id in context")
	}

	logged := buf.String()
	if !strings.Contains(logged, `"msg":"Request received"`) {
		t.Errorf("log missing 'Request received': %s", logged)
	}
	if !strings.Contains(logged, `"method":"GET"`) {
		t.Errorf("log missing method: %s", logged)
	}
	if !strings.Contains(logged, `"path":"/health"`) {
		t.Errorf("log missing path: %s", logged)
	}
	if !strings.Contains(logged, seenRunID) {
		t.Errorf("log missing run_id %q: %s", seenRunID, logged)
	}
}

func TestGetRunID_EmptyWhenAbsent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := GetRunID(req.Context()); got != "" {
		t.Errorf("GetRunID = %q, want empty", got)
	}
}
