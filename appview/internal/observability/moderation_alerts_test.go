package observability

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestModerationAlertThresholds(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	if AdminAuthFailureAlert(4, 5*time.Minute) || !AdminAuthFailureAlert(5, 5*time.Minute) || AdminAuthFailureAlert(5, 5*time.Minute+time.Nanosecond) {
		t.Fatal("admin auth threshold must be five failures within five minutes")
	}
	if ExpiryOverdueAlert(now.Add(-time.Hour), now) || !ExpiryOverdueAlert(now.Add(-time.Hour-time.Nanosecond), now) {
		t.Fatal("expiry threshold must activate only after one hour")
	}
	if ModerationNotificationStalledAlert(now.Add(-15*time.Minute+time.Nanosecond), now) || !ModerationNotificationStalledAlert(now.Add(-15*time.Minute), now) {
		t.Fatal("notification threshold must activate at fifteen minutes")
	}
}

func TestModeratorAuthFailureWindowExcludesOlderFailures(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	observer := New(Config{})
	observer.ObserveModeratorAuthFailure(now.Add(-ModerationAdminAuthFailureWindow - time.Nanosecond))
	for range ModerationAdminAuthFailureThreshold - 1 {
		if observer.ObserveModeratorAuthFailure(now) {
			t.Fatal("alert activated with only four failures inside the window")
		}
	}
	if !observer.ObserveModeratorAuthFailure(now) {
		t.Fatal("alert did not activate at five failures inside the window")
	}
}

func TestModerationObservationsAreBoundedAndRedacted(t *testing.T) {
	const sensitive = "SENTINEL_PRIVATE_EVIDENCE_did:plc:raw"
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	recorder := NewInMemoryMetricRecorder()
	sink := &moderationLogSink{}
	observer := New(Config{
		SentryDSN:      "https://public@example.com/1",
		LogsEnabled:    true,
		MetricsEnabled: true,
		MetricRecorder: recorder,
		LogSink:        sink,
	})

	observer.ObserveModeratorAuthSuccess(now)
	for range 5 {
		observer.ObserveModeratorAuthFailure(now)
	}
	observer.ObserveModerationOperation(context.Background(), ModerationOperationObservation{
		Operation:     sensitive,
		Result:        sensitive,
		ActorType:     sensitive,
		RequestID:     sensitive,
		CaseReference: sensitive,
		OccurredAt:    now,
	})
	observer.ObserveModerationOperation(context.Background(), ModerationOperationObservation{
		Operation:     "decision",
		Result:        "success",
		ActorType:     "moderator",
		RequestID:     strings.Repeat("a", 64),
		CaseReference: "MOD-550e8400-e29b-41d4-a716-446655440000",
		OccurredAt:    now,
	})
	observer.ObserveModerationExpiry(now.Add(-time.Hour-time.Nanosecond), now)
	observer.ObserveModerationNotificationQueue(1, 15*time.Minute)

	var observable strings.Builder
	for _, call := range recorder.Calls() {
		observable.WriteString(call.Name)
		for key, value := range call.Attributes {
			observable.WriteString(key)
			observable.WriteString(value)
		}
	}
	for _, event := range sink.events {
		observable.WriteString(event.message)
		for key, value := range event.attrs {
			observable.WriteString(key)
			observable.WriteString(value.(string))
		}
	}
	if strings.Contains(observable.String(), sensitive) || strings.Contains(observable.String(), "did:plc:") {
		t.Fatalf("moderation telemetry leaked sensitive input: %s", observable.String())
	}
	assertModerationMetric(t, recorder.Calls(), "craftsky_appview_moderation_admin_auth_total", 6)
	assertModerationMetric(t, recorder.Calls(), "craftsky_appview_moderation_work_oldest_age_seconds", 2)
	assertModerationMetric(t, recorder.Calls(), "craftsky_appview_moderation_operations_total", 2)
	var safeOperation EventContext
	for _, event := range sink.events {
		if event.message == "moderation operation completed" && event.attrs["case_reference"] != "unknown" {
			safeOperation = event.attrs
		}
	}
	if safeOperation["actor_type"] != "moderator" || safeOperation["request_id"] != strings.Repeat("a", 64) ||
		safeOperation["case_reference"] != "MOD-550e8400-e29b-41d4-a716-446655440000" || safeOperation["occurred_at"] != now.Format(time.RFC3339Nano) {
		t.Fatalf("safe moderation identifiers or timestamp missing: %#v", safeOperation)
	}
}

type moderationLogEvent struct {
	message string
	attrs   EventContext
}

type moderationLogSink struct{ events []moderationLogEvent }

func (s *moderationLogSink) Emit(_ context.Context, _ slog.Level, message string, attrs EventContext) {
	s.events = append(s.events, moderationLogEvent{message: message, attrs: attrs})
}

func assertModerationMetric(t *testing.T, calls []MetricCall, name string, want int) {
	t.Helper()
	got := 0
	for _, call := range calls {
		if call.Name == name {
			got++
		}
	}
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}
