package observability

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type revenueCatMetricRecorder interface {
	RevenueCatWebhook(context.Context, string, time.Duration)
	RevenueCatReconciliation(context.Context, string, string, time.Duration)
}

type subscriptionMetricRecorder interface {
	SubscriptionAssignment(context.Context, string, string)
	SubscriptionAnomalies(context.Context, int)
	BillingClosure(context.Context, string)
}

func (o *Observer) ObserveRevenueCatWebhook(ctx context.Context, outcome string, duration time.Duration) {
	if o == nil {
		return
	}
	outcome = safeRevenueCatWebhookOutcome(outcome)
	if recorder, ok := o.metricRecorder.(revenueCatMetricRecorder); ok {
		recorder.RevenueCatWebhook(ctx, outcome, duration)
	}
	result := "success"
	level := slog.LevelInfo
	if outcome != "accepted" && outcome != "duplicate" {
		result = "error"
		level = slog.LevelWarn
	}
	o.Log(ctx, level, "RevenueCat webhook handled", EventContext{
		"component": "revenuecat_webhook", "operation": "ingress",
		"result": result, "failure_stage": outcome,
	})
}

func (o *Observer) ObserveRevenueCatReconciliation(ctx context.Context, operation, outcome string, duration time.Duration) {
	if o == nil {
		return
	}
	operation = safeRevenueCatReconciliationOperation(operation)
	outcome = safeRevenueCatReconciliationOutcome(outcome)
	if recorder, ok := o.metricRecorder.(revenueCatMetricRecorder); ok {
		recorder.RevenueCatReconciliation(ctx, operation, outcome, duration)
	}
	result := "success"
	level := slog.LevelInfo
	if outcome != "success" && outcome != "empty" {
		result = "error"
		level = slog.LevelWarn
	}
	o.Log(ctx, level, "RevenueCat reconciliation completed", EventContext{
		"component": "revenuecat_reconciliation", "operation": operation,
		"result": result, "failure_stage": outcome,
	})
}

func (o *Observer) ObserveSubscriptionAssignment(ctx context.Context, operation, outcome string) {
	if o == nil {
		return
	}
	operation = safeSubscriptionAssignmentOperation(operation)
	outcome = safeSubscriptionAssignmentOutcome(outcome)
	if recorder, ok := o.metricRecorder.(subscriptionMetricRecorder); ok {
		recorder.SubscriptionAssignment(ctx, operation, outcome)
	}
	o.Log(ctx, slog.LevelInfo, "Subscription assignment completed", EventContext{
		"component": "subscriptions", "operation": operation, "result": outcome,
	})
}

func (o *Observer) ObserveSubscriptionAnomalies(ctx context.Context, count int) {
	if o == nil {
		return
	}
	if count < 0 {
		count = 0
	}
	if recorder, ok := o.metricRecorder.(subscriptionMetricRecorder); ok {
		recorder.SubscriptionAnomalies(ctx, count)
	}
	o.Log(ctx, slog.LevelInfo, "Subscription anomalies counted", EventContext{
		"component": "subscriptions", "operation": "snapshot", "result": "recorded", "count": count,
	})
}

func (o *Observer) ObserveBillingClosure(ctx context.Context, outcome string) {
	if o == nil {
		return
	}
	outcome = safeBillingClosureOutcome(outcome)
	if recorder, ok := o.metricRecorder.(subscriptionMetricRecorder); ok {
		recorder.BillingClosure(ctx, outcome)
	}
	o.Log(ctx, slog.LevelInfo, "Billing closure completed", EventContext{
		"component": "subscriptions", "operation": "closure", "result": outcome,
	})
}

func safeRevenueCatWebhookOutcome(value string) string {
	switch strings.TrimSpace(value) {
	case "accepted", "duplicate", "authentication_failed", "malformed", "oversized", "deadline_exceeded", "store_error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeRevenueCatReconciliationOperation(value string) string {
	switch strings.TrimSpace(value) {
	case "schedule", "process":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeRevenueCatReconciliationOutcome(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "empty", "provider_error", "apply_error", "store_error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeSubscriptionAssignmentOperation(value string) string {
	switch strings.TrimSpace(value) {
	case "assign", "unassign":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeSubscriptionAssignmentOutcome(value string) string {
	switch strings.TrimSpace(value) {
	case "success", "not_found", "target_ineligible", "already_assigned", "cooldown", "store_error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func safeBillingClosureOutcome(value string) string {
	switch strings.TrimSpace(value) {
	case "closed", "not_owner", "blocked", "store_error":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func (r *InMemoryMetricRecorder) RevenueCatWebhook(_ context.Context, outcome string, duration time.Duration) {
	attrs := map[string]string{"outcome": safeRevenueCatWebhookOutcome(outcome)}
	r.record(MetricCall{Name: "craftsky_appview_revenuecat_webhooks_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_revenuecat_webhook_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) RevenueCatReconciliation(_ context.Context, operation, outcome string, duration time.Duration) {
	attrs := map[string]string{
		"operation": safeRevenueCatReconciliationOperation(operation),
		"outcome":   safeRevenueCatReconciliationOutcome(outcome),
	}
	r.record(MetricCall{Name: "craftsky_appview_revenuecat_reconciliations_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_revenuecat_reconciliation_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) SubscriptionAssignment(_ context.Context, operation, outcome string) {
	r.record(MetricCall{Name: "craftsky_appview_subscription_assignments_total", Kind: MetricKindCounter, Value: 1, Attributes: map[string]string{
		"operation": safeSubscriptionAssignmentOperation(operation), "outcome": safeSubscriptionAssignmentOutcome(outcome),
	}})
}

func (r *InMemoryMetricRecorder) SubscriptionAnomalies(_ context.Context, count int) {
	r.record(MetricCall{Name: "craftsky_appview_subscription_anomalies", Kind: MetricKindGauge, Value: float64(count), Attributes: map[string]string{}})
}

func (r *InMemoryMetricRecorder) BillingClosure(_ context.Context, outcome string) {
	r.record(MetricCall{Name: "craftsky_appview_billing_closures_total", Kind: MetricKindCounter, Value: 1, Attributes: map[string]string{"outcome": safeBillingClosureOutcome(outcome)}})
}

func (r *sentryMetricRecorder) RevenueCatWebhook(ctx context.Context, outcome string, duration time.Duration) {
	attrs := map[string]string{"outcome": safeRevenueCatWebhookOutcome(outcome)}
	r.count(ctx, "craftsky_appview_revenuecat_webhooks_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_revenuecat_webhook_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}

func (r *sentryMetricRecorder) RevenueCatReconciliation(ctx context.Context, operation, outcome string, duration time.Duration) {
	attrs := map[string]string{
		"operation": safeRevenueCatReconciliationOperation(operation),
		"outcome":   safeRevenueCatReconciliationOutcome(outcome),
	}
	r.count(ctx, "craftsky_appview_revenuecat_reconciliations_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_revenuecat_reconciliation_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}

func (r *sentryMetricRecorder) SubscriptionAssignment(ctx context.Context, operation, outcome string) {
	r.count(ctx, "craftsky_appview_subscription_assignments_total", 1, "", map[string]string{
		"operation": safeSubscriptionAssignmentOperation(operation), "outcome": safeSubscriptionAssignmentOutcome(outcome),
	})
}

func (r *sentryMetricRecorder) SubscriptionAnomalies(ctx context.Context, count int) {
	r.gauge(ctx, "craftsky_appview_subscription_anomalies", float64(count), "", map[string]string{})
}

func (r *sentryMetricRecorder) BillingClosure(ctx context.Context, outcome string) {
	r.count(ctx, "craftsky_appview_billing_closures_total", 1, "", map[string]string{"outcome": safeBillingClosureOutcome(outcome)})
}
