package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/observability"
)

func TestRecovery_ReturnsV1EnvelopeAndContinuesServing(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	mux := http.NewServeMux()
	mux.Handle("GET /v1/panic", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	mux.Handle("GET /v1/ok", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
	})
	handler := Logging(logger)(HTTPMetrics(observer)(Recovery(logger, observer)(mux)))

	panicReq := httptest.NewRequest(http.MethodGet, "/v1/panic", nil)
	panicRec := httptest.NewRecorder()
	handler.ServeHTTP(panicRec, panicReq)

	if panicRec.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500; body=%s", panicRec.Code, panicRec.Body.String())
	}
	var body struct {
		Error     string `json:"error"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(panicRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("panic body not JSON envelope: %v; body=%s", err, panicRec.Body.String())
	}
	if body.Error != "internal_error" || body.Message == "" || body.RequestID == "" {
		t.Fatalf("panic envelope = %+v, want internal_error/message/requestId", body)
	}
	logged := buf.String()
	if !strings.Contains(logged, `"msg":"HTTP panic recovered"`) || !strings.Contains(logged, body.RequestID) || !strings.Contains(logged, `"route_pattern":"/v1/panic"`) {
		t.Fatalf("panic log missing safe context: %s", logged)
	}
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1", len(events))
	}
	if events[0].Tags["component"] != "http" || events[0].Tags["route_pattern"] != "/v1/panic" || events[0].Tags["run_id"] != body.RequestID {
		t.Fatalf("panic Sentry event missing safe context: %#v", events[0].Tags)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/v1/ok", nil)
	okRec := httptest.NewRecorder()
	handler.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want 204; body=%s", okRec.Code, okRec.Body.String())
	}
}
