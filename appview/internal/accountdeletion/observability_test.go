package accountdeletion

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestOperationalMinimizationAndTelemetryRedaction(t *testing.T) {
	t.Parallel()

	acceptedAt := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	source := OperationalSource{
		JobID:                "job-operational-1",
		DID:                  syntax.DID("did:plc:operational-canary"),
		AcceptedAt:           acceptedAt,
		Status:               StatusActive,
		Phase:                PhaseRemovingCraftskyRecords,
		Attempt:              2,
		OAuthSessionID:       "oauth-session-canary",
		StatusCapabilityHash: "status-capability-canary",
		ExpectedURIs:         []string{"at://did:plc:operational-canary/social.craftsky.post/uri-canary"},
		ReceiptIDs:           []string{"tap-receipt-canary"},
		Handle:               "handle-canary.craftsky.social",
		RecordContent:        "record-content-canary",
		RelationshipData:     "relationship-canary",
		SettingsData:         "settings-canary",
		ImportData:           "import-canary",
		FullURL:              "https://full-url-canary.invalid/private/path",
	}

	operational := NewOperationalState(source)
	encodedOperational, err := json.Marshal(operational)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{source.Handle, source.RecordContent, source.RelationshipData, source.SettingsData, source.ImportData, source.FullURL} {
		if strings.Contains(string(encodedOperational), prohibited) {
			t.Fatalf("operational state contains prohibited value %q: %s", prohibited, encodedOperational)
		}
	}

	terminalAt := acceptedAt.Add(time.Hour)
	terminal := FinalizeOperationalState(operational, terminalAt)
	encodedTerminal, err := json.Marshal(terminal)
	if err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{
		source.OAuthSessionID,
		source.StatusCapabilityHash,
		source.ExpectedURIs[0],
		source.ReceiptIDs[0],
	} {
		if strings.Contains(string(encodedTerminal), removed) {
			t.Fatalf("terminal projection contains operational value %q: %s", removed, encodedTerminal)
		}
	}

	attributes := TelemetryAttributes(PhaseRemovingCraftskyRecords, AuditOutcomeDeleted, ErrorCategoryPDS)
	encodedTelemetry, err := json.Marshal(attributes)
	if err != nil {
		t.Fatal(err)
	}
	if attributes["phase"] != string(PhaseRemovingCraftskyRecords) || attributes["outcome"] != string(AuditOutcomeDeleted) || attributes["errorCategory"] != string(ErrorCategoryPDS) {
		t.Fatalf("coarse telemetry = %#v", attributes)
	}
	for _, prohibited := range []string{
		string(source.DID), source.Handle, source.OAuthSessionID, source.StatusCapabilityHash,
		source.ExpectedURIs[0], source.ReceiptIDs[0], source.RecordContent,
		source.RelationshipData, source.SettingsData, source.ImportData, source.FullURL,
	} {
		if strings.Contains(string(encodedTelemetry), prohibited) {
			t.Fatalf("telemetry contains prohibited value %q: %s", prohibited, encodedTelemetry)
		}
	}
}

func TestDeletionTelemetryEmitsOnlyCoarseLifecycleSignals(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	metrics := &recordingDeletionMetrics{}
	telemetry := NewDeletionTelemetry(
		slog.New(slog.NewJSONHandler(&logs, nil)),
		metrics,
	)
	ctx := context.Background()
	telemetry.Phase(ctx, PhaseRemovingPrivateData)
	telemetry.AutomaticRetry(ctx, PhaseRemovingCraftskyRecords, ErrorCategoryPDS)
	telemetry.ManualRetry(ctx, PhaseRemovingCraftskyRecords)
	telemetry.NeedsAttention(ctx, PhaseWaitingForIndexerConvergence, ErrorCategoryIndexer)
	telemetry.ConvergenceDelay(ctx, 12*time.Second)
	telemetry.TerminalSuccess(ctx)
	telemetry.AuditExpired(ctx)

	logText := logs.String()
	for _, coarse := range []string{
		string(PhaseRemovingPrivateData), string(PhaseRemovingCraftskyRecords),
		string(PhaseWaitingForIndexerConvergence), string(ErrorCategoryPDS),
		string(ErrorCategoryIndexer), string(AuditOutcomeDeleted),
	} {
		if !strings.Contains(logText, coarse) {
			t.Fatalf("coarse telemetry %q missing from logs: %s", coarse, logText)
		}
	}
	for _, prohibited := range []string{
		"did:plc:telemetry-canary", "handle-canary.test", "oauth-token-canary",
		"status-token-canary", "at://uri-canary", "record-content-canary",
		"relationship-canary", "settings-canary", "import-canary",
		"https://full-url-canary.invalid/private",
	} {
		if strings.Contains(logText, prohibited) {
			t.Fatalf("logs contain prohibited value %q: %s", prohibited, logText)
		}
	}
	if len(metrics.events) != 7 {
		t.Fatalf("metric events = %d, want 7: %#v", len(metrics.events), metrics.events)
	}
}

type recordingDeletionMetrics struct {
	events []DeletionMetricEvent
}

func (metrics *recordingDeletionMetrics) RecordDeletionMetric(_ context.Context, event DeletionMetricEvent) {
	metrics.events = append(metrics.events, event)
}

var _ DeletionMetricRecorder = (*recordingDeletionMetrics)(nil)
var _ io.Writer = (*bytes.Buffer)(nil)
