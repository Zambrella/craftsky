package tap_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/tap"
)

type tapDiagnosticLines chan []byte

func (lines tapDiagnosticLines) Write(data []byte) (int, error) {
	select {
	case lines <- append([]byte(nil), data...):
	default:
	}
	return len(data), nil
}

func TestTapReconnectRetainsProtectedCauseWithoutCreatingIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Private", "PRIVATE_HEADER")
		http.Error(w, "PRIVATE_PROVIDER_BODY", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	lines := make(tapDiagnosticLines, 8)
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(lines, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1) + "/PRIVATE_PATH?code=PRIVATE_CAPABILITY", Ingestor: &durableIngestorSpy{}, Logger: logger, Observer: observer, AckTimeout: time.Second, ReconnectMax: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	var line []byte
	select {
	case line = <-lines:
	case <-time.After(3 * time.Second):
		t.Fatal("reconnect diagnostic missing")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("consumer failed to stop")
	}
	observer.Flush(time.Second)
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			t.Fatal("recoverable disconnect created an issue")
		}
	}
	var record map[string]any
	if err := json.Unmarshal(line, &record); err != nil {
		t.Fatal(err)
	}
	if record["operation"] != "tap.connect" || record["failure_stage"] != "connection" || record["result"] != "retry" || record["level"] != "WARN" || record["causes"] == nil {
		t.Fatalf("connection cause lost: %s", line)
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(line), "*fmt.wrapError") {
		t.Fatalf("typed transport cause missing: %s", line)
	}
	if strings.Contains(string(line)+string(serialized), "PRIVATE_") || strings.Contains(string(line)+string(serialized), server.URL) {
		t.Fatal("private connection data escaped")
	}
}
