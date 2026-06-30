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
	if !strings.Contains(logged, `"route_pattern":"unmatched"`) {
		t.Errorf("log missing route_pattern fallback: %s", logged)
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

func TestLogging_RedactsAuthorizationAndDoesNotLogRequestBody(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"secret":"payload"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer raw-secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	logged := buf.String()
	if strings.Contains(logged, "raw-secret-token") || strings.Contains(logged, "Bearer raw-secret-token") {
		t.Fatalf("log contains raw authorization token: %s", logged)
	}
	if strings.Contains(logged, "payload") || strings.Contains(logged, "json_payload") {
		t.Fatalf("log contains request payload: %s", logged)
	}
}

func TestLogging_DoesNotLogResponseBodyByDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secret":"response-payload"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	logged := buf.String()
	if strings.Contains(logged, "response-payload") || strings.Contains(logged, "json_payload") {
		t.Fatalf("log contains response payload by default: %s", logged)
	}
}

func TestLogging_CompletionUsesStableFieldsAndRoutePattern(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})).
		With(slog.String("service", "craftsky_appview"), slog.String("environment", "test"))

	var seenRunID string
	mux := http.NewServeMux()
	mux.Handle("GET /v1/posts/{did}/{rkey}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRunID = GetRunID(r.Context())
		logger.Error("handler failed", slog.String("run_id", seenRunID), slog.String("component", "test"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	handler := Logging(logger)(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:raw/rkey123?cursor=secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if seenRunID == "" {
		t.Fatal("handler did not see a run_id in context")
	}
	logged := buf.String()
	for _, want := range []string{
		`"service":"craftsky_appview"`,
		`"environment":"test"`,
		`"run_id":"` + seenRunID + `"`,
		`"method":"GET"`,
		`"route_pattern":"/v1/posts/{did}/{rkey}"`,
		`"status":500`,
		`"duration"`,
		`"msg":"handler failed"`,
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log missing %s:\n%s", want, logged)
		}
	}
	for _, forbidden := range []string{"did:plc:raw", "rkey123", "cursor=secret"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("log contains raw route/query value %q:\n%s", forbidden, logged)
		}
	}
}
