package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/integrations/instagrammeta"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/push"
)

func TestInstagramCanariesStayOutOfDiagnosticsTelemetryPushPDSAndURLs(t *testing.T) {
	t.Parallel()

	canaries := []string{
		"CSKY-PRIV-ATE1-CODE-X",
		`{"private":"instagram-webhook-body-canary"}`,
		"private.instagram.username",
		"17841400000000000",
		"private-imported-handle",
		"EAAG-private-meta-token",
		"private-export-payload",
		"private-upstream-response",
	}
	now := time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)
	models := []any{
		instagram.CreatedVerification{Challenge: canaries[0], DMURL: "https://www.instagram.com/direct/t/synthetic"},
		instagram.VerificationAttempt{CandidateIGSID: canaries[3], CandidateUsername: canaries[2]},
		instagram.AccountView{Username: canaries[2]},
		instagram.ImportEntry{Username: canaries[4]},
		instagram.PrivateSuggestion{
			ID:               uuid.MustParse("00000000-0000-0000-0000-000000000702"),
			ImportedUsername: canaries[4],
		},
		instagram.SuggestionEligibilityRequest{ImportedUsername: canaries[4]},
		instagrammeta.WorkItem{SenderIGSID: canaries[3], OfficialAccountID: canaries[3]},
	}
	var diagnostic strings.Builder
	for _, model := range models {
		fmt.Fprintf(&diagnostic, "%v %+v %#v ", model, model, model)
	}

	transport := &sentry.MockTransport{}
	metrics := NewInMemoryMetricRecorder()
	observer := New(Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: transport, LogsEnabled: true, MetricsEnabled: true,
		MetricRecorder: metrics,
	})
	privateContext := EventContext{
		"component": "instagram", "operation": "instagram.verify", "result": "error",
		"challenge": canaries[0], "username": canaries[2], "igsid": canaries[3],
		"request_body": canaries[1], "token": canaries[5], "upstream": canaries[7],
	}
	observer.Log(context.Background(), slog.LevelWarn, "Instagram verification attempt failed", privateContext)
	observer.CaptureError(context.Background(), privateContext, fmt.Errorf("provider failed: %s", canaries[7]))
	observer.ObservePushDelivery("ios", "success")
	if !observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}
	telemetry, err := json.Marshal(struct {
		Events  []*sentry.Event
		Metrics []MetricCall
	}{transport.Events(), metrics.Calls()})
	if err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(push.BuildPayload(notifications.Follow, "safe-routing-id", "", push.RoutingFacts{
		NotificationID: "00000000-0000-0000-0000-000000000701",
	}))
	if err != nil {
		t.Fatal(err)
	}
	pdsRecord, err := json.Marshal(&bsky.GraphFollow{
		LexiconTypeID: "app.bsky.graph.follow",
		Subject:       "did:plc:synthetic-target",
		CreatedAt:     now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	urlOutput := "https://www.instagram.com/direct/t/synthetic"

	outputs := []struct {
		name string
		text string
	}{
		{"diagnostics", diagnostic.String()},
		{"telemetry", string(telemetry)},
		{"push", string(payload)},
		{"pds", string(pdsRecord)},
		{"url", urlOutput},
	}
	for _, output := range outputs {
		for _, canary := range canaries {
			if strings.Contains(output.text, canary) {
				t.Fatalf("%s leaked Instagram canary %q: %s", output.name, canary, output.text)
			}
		}
	}
}
