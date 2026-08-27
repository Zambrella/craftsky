package followergrowth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/observability"
)

func TestFollowerGrowthAttemptLogsRemainBoundedAndPrivate(t *testing.T) {
	var output bytes.Buffer
	observer := observability.New(observability.Config{
		Logger: slog.New(slog.NewJSONHandler(&output, nil)),
	})
	knownAge := 24 * time.Hour
	observer.FollowerGrowthCapture(
		context.Background(),
		"did:plc:private-owner",
		"database_unavailable_for_alice.example",
		2*time.Second,
		9_876_543,
		&knownAge,
	)

	logged := output.String()
	for _, required := range []string{
		`"component":"follower_growth"`,
		`"operation":"capture"`,
		`"result":"unknown"`,
		`"error_category":"unknown"`,
	} {
		if !strings.Contains(logged, required) {
			t.Fatalf("log missing %s: %s", required, logged)
		}
	}
	for _, forbidden := range []string{
		"did:plc:private-owner",
		"alice.example",
		"9876543",
		"24h",
	} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("log leaked %q: %s", forbidden, logged)
		}
	}
}

func TestFollowerGrowthMetricsRemainBoundedAndPrivate(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	knownAge := 3 * time.Hour
	recorder.FollowerGrowthCapture(
		context.Background(),
		"success",
		"none",
		2*time.Second,
		42,
		&knownAge,
	)
	recorder.FollowerGrowthCapture(
		context.Background(),
		"did:plc:private-owner",
		"database_unavailable_for_alice.example",
		2*time.Second,
		9_876_543,
		&knownAge,
	)

	calls := recorder.Calls()
	if len(calls) != 8 {
		t.Fatalf("metric calls = %+v, want four measurements per observation", calls)
	}
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Errorf("invalid metric call %+v: %v", call, err)
		}
		for key, value := range call.Attributes {
			if key != "result" && key != "error_category" {
				t.Errorf("metric %s has unbounded attribute %s=%q", call.Name, key, value)
			}
		}
	}
	if calls[2].Value != 42 {
		t.Fatalf("captured profile measurement = %v, want 42", calls[2].Value)
	}
	if len(calls[2].Attributes) != 0 || len(calls[3].Attributes) != 0 {
		t.Fatalf("count/age measurements became dimensions: %+v", calls)
	}
	if calls[4].Attributes["result"] != "unknown" || calls[4].Attributes["error_category"] != "unknown" {
		t.Fatalf("unbounded labels were not collapsed: %+v", calls[4])
	}
	raw, err := json.Marshal(calls)
	if err != nil {
		t.Fatalf("marshal metric calls: %v", err)
	}
	for _, forbidden := range []string{
		"did:plc:private-owner",
		"alice.example",
		"database_unavailable_for_alice.example",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("telemetry leaked %q: %s", forbidden, raw)
		}
	}
}

func TestFollowerGrowthMetricsOmitUnknownLatestSuccessAge(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	recorder.FollowerGrowthCapture(
		context.Background(),
		"error",
		"capture",
		time.Second,
		0,
		nil,
	)

	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_follower_growth_latest_success_age_seconds" {
			t.Fatalf("unknown latest-success age emitted as gauge: %+v", call)
		}
	}
}
