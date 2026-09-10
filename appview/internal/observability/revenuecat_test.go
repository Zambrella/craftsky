package observability

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRevenueCatObservationsAreBoundedAndPrivate(t *testing.T) {
	const canary = "did:plc:private-secret-token-customer"
	metrics := NewInMemoryMetricRecorder()
	var logs bytes.Buffer
	observer := New(Config{
		MetricRecorder: metrics,
		Logger:         slog.New(slog.NewJSONHandler(&logs, nil)),
	})

	observer.ObserveRevenueCatWebhook(context.Background(), canary, time.Millisecond)
	observer.ObserveRevenueCatReconciliation(context.Background(), canary, canary, time.Millisecond)
	observer.ObserveSubscriptionAssignment(context.Background(), canary, canary)
	observer.ObserveSubscriptionAnomalies(context.Background(), 3)
	observer.ObserveBillingClosure(context.Background(), canary)

	if strings.Contains(logs.String(), canary) {
		t.Fatalf("RevenueCat logs leaked canary: %s", logs.String())
	}
	calls := metrics.Calls()
	if len(calls) != 7 {
		t.Fatalf("billing metric calls = %#v, want seven", calls)
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("RevenueCat metric is unsafe: %v; call=%#v", err, call)
		}
		for _, value := range call.Attributes {
			if strings.Contains(value, canary) {
				t.Fatalf("RevenueCat metric leaked canary: %#v", call)
			}
		}
	}
}
