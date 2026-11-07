package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/observability"
)

func TestBunnyCleanupDiagnosticsKeepRetryContextAndProtectSecrets(t *testing.T) {
	b, w, id, _, _ := cleanupFixture(t)
	var logs bytes.Buffer
	w.logger = slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&logs, nil)))
	transport := &sentry.MockTransport{}
	w.observer = observability.New(observability.Config{Env: "test", Logger: w.logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	b.client.Transport = bunnyRoundTrip(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("password=private-cleanup-canary https://private.example/signed-capability-canary")
	})
	for _, attempt := range []int{1, 5} {
		logs.Reset()
		_, err := b.pool.Exec(context.Background(), `UPDATE bunny_video_jobs SET state='purging',cleanup_requested_at=now(),cleanup_next_attempt_at=now(),cleanup_attempts=$2 WHERE id=$1`, id, attempt-1)
		if err != nil {
			t.Fatal(err)
		}
		cleanupStep(t, w)
		var entry map[string]any
		if err = json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
			t.Fatal(err)
		}
		wantLevel := "WARN"
		wantResult := "retry"
		if attempt == 5 {
			wantLevel = "ERROR"
			wantResult = "exhausted"
		}
		if entry["level"] != wantLevel || entry["result"] != wantResult || entry["failure_stage"] != "purge_cdn" || entry["attempt"] != float64(attempt) {
			t.Fatalf("retry context missing: %s", logs.String())
		}
		data, _ := json.Marshal(transport.Events())
		output := logs.String() + string(data)
		for _, canary := range []string{"private-cleanup-canary", "signed-capability-canary", "stream-key-canary", "purge-key-canary"} {
			if strings.Contains(output, canary) {
				t.Fatal("protected value reached a diagnostic sink")
			}
		}
	}
	if len(transport.Events()) != 1 {
		t.Fatal("only exhausted retries should create an issue")
	}
}
