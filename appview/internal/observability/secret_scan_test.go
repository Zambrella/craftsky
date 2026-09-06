package observability

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMigrationTelemetryDoesNotSerializeSensitiveErrorMaterial(t *testing.T) {
	canaries := []string{
		"oauth-access-canary",
		"oauth-refresh-canary",
		"dpop-private-key-canary",
		"dpop-proof-canary",
		"Bearer craftsky-session-canary",
		"confirmation-hash-canary",
		`{"accessToken":"raw-session-json-canary"}`,
	}
	untrustedError := strings.Join(canaries, " ")
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})

	observer.ObserveAuthorityVerification(untrustedError, untrustedError, untrustedError, time.Second)
	observer.ObserveOAuthMetadataCache(context.Background(), untrustedError, untrustedError)
	observer.ObserveRepositoryRepair(untrustedError, untrustedError, untrustedError, time.Second, 8)
	observer.ObserveRepositorySnapshotVerification(untrustedError, untrustedError, time.Second)

	classified := ClassifyError(errors.New(untrustedError), EventContext{
		"operation": untrustedError,
		"result":    untrustedError,
		"token":     untrustedError,
	})
	safeContext := SanitizeEventContext(EventContext{
		"component":      "repository_repair",
		"error_category": untrustedError,
		"session":        untrustedError,
	})
	encoded, err := json.Marshal(struct {
		Metrics    []MetricCall
		Error      ClassifiedError
		LogContext EventContext
	}{Metrics: recorder.Calls(), Error: classified, LogContext: safeContext})
	if err != nil {
		t.Fatalf("marshal telemetry: %v", err)
	}
	output := string(encoded)
	for _, canary := range canaries {
		if strings.Contains(output, canary) {
			t.Fatalf("sensitive canary %q escaped into telemetry: %s", canary, output)
		}
	}
	if !strings.Contains(output, `"unknown"`) {
		t.Fatalf("telemetry lacks bounded fallback reason: %s", output)
	}
}
