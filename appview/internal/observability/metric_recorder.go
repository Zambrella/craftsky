package observability

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
)

type MetricKind string

const (
	MetricKindCounter      MetricKind = "counter"
	MetricKindGauge        MetricKind = "gauge"
	MetricKindDistribution MetricKind = "distribution"
)

type MetricCall struct {
	Name       string
	Kind       MetricKind
	Unit       string
	Value      float64
	Attributes map[string]string
}

type MetricRecorder interface {
	HTTPRequestStarted(ctx context.Context, method, routePattern string)
	HTTPRequestEnded(ctx context.Context, method, routePattern string)
	HTTPRequestFinished(ctx context.Context, method, routePattern string, status int, duration time.Duration, responseBytes int)
	DBOperation(ctx context.Context, operation, routePattern, resultClass string, duration time.Duration)
	PDSOperation(ctx context.Context, operation, stage, result, category string, duration time.Duration)
	TapConnected(ctx context.Context, connected bool)
	TapLastEventAge(ctx context.Context, age time.Duration)
	TapReconnect(ctx context.Context)
	TapEventReceived(ctx context.Context, eventType string)
	TapEventAcknowledged(ctx context.Context, result string)
	TapIndexerRecord(ctx context.Context, nsid, result, reason string, duration time.Duration)
	RelationshipOperation(ctx context.Context, operation, stage, result, errorClass string, duration time.Duration)
	ProfilePinOperation(ctx context.Context, operation, slot, result, errorClass string, duration time.Duration)
	NotificationDecision(ctx context.Context, category, result string)
	PushDelivery(ctx context.Context, platform, result string)
	PushOperation(ctx context.Context, stage, platform, semantics, outcome string, duration time.Duration, count int64)
	PushQueue(ctx context.Context, pending int, oldestAge time.Duration)
	ScheduledQueue(ctx context.Context, status string, count, due, overdue int, oldestDueAge time.Duration)
	ScheduledOperation(ctx context.Context, operation, result, errorClass string, duration time.Duration)
	ScheduledPublication(ctx context.Context, attempt int, startLatency, duration time.Duration)
	ScheduledCleanupQueue(ctx context.Context, pending int, oldestAge time.Duration)
	ScheduledImageValidation(ctx context.Context, result, format string, duration time.Duration, inFlight int)
	AuthRequestSweep(ctx context.Context, pending int64, oldestAge time.Duration, deleted int64, failed bool)
	TerminalPurge(ctx context.Context, operation, result, errorCategory, component, didRole string, claims int, rowsAffected, remaining int64, complete bool)
	IdentityResolution(ctx context.Context, mode, direction, result string, duration time.Duration)
	IdentityCache(ctx context.Context, result string, age time.Duration)
	FollowerGrowthCapture(ctx context.Context, result, errorCategory string, duration time.Duration, capturedProfileCount int64, latestSuccessfulRunAge *time.Duration)
}

type noopMetricRecorder struct{}

func (noopMetricRecorder) HTTPRequestStarted(context.Context, string, string) {}
func (noopMetricRecorder) HTTPRequestEnded(context.Context, string, string)   {}
func (noopMetricRecorder) HTTPRequestFinished(context.Context, string, string, int, time.Duration, int) {
}
func (noopMetricRecorder) DBOperation(context.Context, string, string, string, time.Duration) {}
func (noopMetricRecorder) PDSOperation(context.Context, string, string, string, string, time.Duration) {
}
func (noopMetricRecorder) TapConnected(context.Context, bool) {}
func (noopMetricRecorder) TapLastEventAge(context.Context, time.Duration) {
}
func (noopMetricRecorder) TapReconnect(context.Context)                 {}
func (noopMetricRecorder) TapEventReceived(context.Context, string)     {}
func (noopMetricRecorder) TapEventAcknowledged(context.Context, string) {}
func (noopMetricRecorder) TapIndexerRecord(context.Context, string, string, string, time.Duration) {
}
func (noopMetricRecorder) RelationshipOperation(context.Context, string, string, string, string, time.Duration) {
}
func (noopMetricRecorder) ProfilePinOperation(context.Context, string, string, string, string, time.Duration) {
}
func (noopMetricRecorder) NotificationDecision(context.Context, string, string) {}
func (noopMetricRecorder) PushDelivery(context.Context, string, string)         {}
func (noopMetricRecorder) PushOperation(context.Context, string, string, string, string, time.Duration, int64) {
}
func (noopMetricRecorder) PushQueue(context.Context, int, time.Duration) {}
func (noopMetricRecorder) ScheduledQueue(context.Context, string, int, int, int, time.Duration) {
}
func (noopMetricRecorder) ScheduledOperation(context.Context, string, string, string, time.Duration) {
}
func (noopMetricRecorder) ScheduledPublication(context.Context, int, time.Duration, time.Duration) {
}
func (noopMetricRecorder) ScheduledCleanupQueue(context.Context, int, time.Duration) {}
func (noopMetricRecorder) ScheduledImageValidation(context.Context, string, string, time.Duration, int) {
}
func (noopMetricRecorder) AuthRequestSweep(context.Context, int64, time.Duration, int64, bool) {}
func (noopMetricRecorder) TerminalPurge(context.Context, string, string, string, string, string, int, int64, int64, bool) {
}
func (noopMetricRecorder) IdentityResolution(context.Context, string, string, string, time.Duration) {
}
func (noopMetricRecorder) IdentityCache(context.Context, string, time.Duration) {}
func (noopMetricRecorder) FollowerGrowthCapture(context.Context, string, string, time.Duration, int64, *time.Duration) {
}

type InMemoryMetricRecorder struct {
	mu       sync.Mutex
	calls    []MetricCall
	inFlight map[string]int
}

func NewInMemoryMetricRecorder() *InMemoryMetricRecorder {
	return &InMemoryMetricRecorder{}
}

func (r *InMemoryMetricRecorder) Calls() []MetricCall {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]MetricCall, len(r.calls))
	copy(out, r.calls)
	for i := range out {
		out[i].Attributes = cloneMetricAttributes(out[i].Attributes)
	}
	return out
}

func (r *InMemoryMetricRecorder) HTTPRequestStarted(_ context.Context, method, routePattern string) {
	attrs := httpMetricAttributes(method, routePattern, 0)
	value := r.changeInFlight(attrs, 1)
	r.record(MetricCall{
		Name:       "craftsky_appview_http_requests_in_flight",
		Kind:       MetricKindGauge,
		Unit:       "request",
		Value:      float64(value),
		Attributes: attrs,
	})
}

func (r *InMemoryMetricRecorder) HTTPRequestEnded(_ context.Context, method, routePattern string) {
	attrs := httpMetricAttributes(method, routePattern, 0)
	value := r.changeInFlight(attrs, -1)
	r.record(MetricCall{
		Name:       "craftsky_appview_http_requests_in_flight",
		Kind:       MetricKindGauge,
		Unit:       "request",
		Value:      float64(value),
		Attributes: attrs,
	})
}

func (r *InMemoryMetricRecorder) HTTPRequestFinished(_ context.Context, method, routePattern string, status int, duration time.Duration, responseBytes int) {
	attrs := httpMetricAttributes(method, routePattern, status)
	r.record(MetricCall{Name: "craftsky_appview_http_requests_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_http_request_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: duration.Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_http_response_size_bytes", Kind: MetricKindDistribution, Unit: "byte", Value: float64(responseBytes), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) DBOperation(_ context.Context, operation, routePattern, resultClass string, duration time.Duration) {
	r.record(MetricCall{
		Name:  "craftsky_appview_db_operation_duration_seconds",
		Kind:  MetricKindDistribution,
		Unit:  "second",
		Value: duration.Seconds(),
		Attributes: map[string]string{
			"operation":     safeMetricOperation(operation),
			"route_pattern": safeMetricRoute(routePattern),
			"result":        safeMetricResult(resultClass),
		},
	})
}

func (r *InMemoryMetricRecorder) PDSOperation(_ context.Context, operation, stage, result, category string, duration time.Duration) {
	r.record(MetricCall{
		Name:  "craftsky_appview_pds_write_duration_seconds",
		Kind:  MetricKindDistribution,
		Unit:  "second",
		Value: duration.Seconds(),
		Attributes: map[string]string{
			"operation": safeMetricOperation(operation),
			"stage":     safeMetricStage(stage),
			"result":    safeMetricResult(result),
			"category":  safeMetricCategory(category),
		},
	})
}

func (r *InMemoryMetricRecorder) TapConnected(_ context.Context, connected bool) {
	value := float64(0)
	if connected {
		value = 1
	}
	r.record(MetricCall{Name: "craftsky_appview_tap_connected", Kind: MetricKindGauge, Value: value})
}

func (r *InMemoryMetricRecorder) TapLastEventAge(_ context.Context, age time.Duration) {
	if age < 0 {
		age = 0
	}
	r.record(MetricCall{Name: "craftsky_appview_tap_last_event_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: age.Seconds()})
}

func (r *InMemoryMetricRecorder) TapReconnect(context.Context) {
	r.record(MetricCall{Name: "craftsky_appview_tap_reconnects_total", Kind: MetricKindCounter, Value: 1})
}

func (r *InMemoryMetricRecorder) TapEventReceived(_ context.Context, eventType string) {
	r.record(MetricCall{
		Name:       "craftsky_appview_tap_events_received_total",
		Kind:       MetricKindCounter,
		Value:      1,
		Attributes: map[string]string{"type": SafeTapEventType(eventType)},
	})
}

func (r *InMemoryMetricRecorder) TapEventAcknowledged(_ context.Context, result string) {
	if safeMetricResult(result) == "error" {
		r.record(MetricCall{Name: "craftsky_appview_tap_ack_failures_total", Kind: MetricKindCounter, Value: 1})
		return
	}
	r.record(MetricCall{Name: "craftsky_appview_tap_events_acknowledged_total", Kind: MetricKindCounter, Value: 1})
}
func (r *InMemoryMetricRecorder) NotificationDecision(_ context.Context, category, result string) {
	r.record(MetricCall{Name: "craftsky_appview_notifications_total", Kind: MetricKindCounter, Value: 1, Attributes: map[string]string{"category": safeMetricCategory(category), "result": safeMetricResult(result)}})
}
func (r *InMemoryMetricRecorder) PushDelivery(_ context.Context, platform, result string) {
	r.record(MetricCall{Name: "craftsky_appview_push_deliveries_total", Kind: MetricKindCounter, Value: 1, Attributes: map[string]string{"platform": safeMetricCategory(platform), "result": safeMetricResult(result)}})
}
func (r *InMemoryMetricRecorder) PushOperation(
	_ context.Context,
	stage string,
	platform string,
	semantics string,
	outcome string,
	duration time.Duration,
	count int64,
) {
	attrs := pushOperationAttributes(stage, platform, semantics, outcome)
	if count < 1 {
		count = 1
	}
	r.record(MetricCall{
		Name: "craftsky_appview_push_operations_total", Kind: MetricKindCounter,
		Value: float64(count), Attributes: attrs,
	})
	r.record(MetricCall{
		Name: "craftsky_appview_push_operation_duration_seconds",
		Kind: MetricKindDistribution, Unit: "second",
		Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs,
	})
}
func (r *InMemoryMetricRecorder) PushQueue(_ context.Context, pending int, age time.Duration) {
	r.record(MetricCall{Name: "craftsky_appview_push_pending", Kind: MetricKindGauge, Value: float64(pending)})
	r.record(MetricCall{Name: "craftsky_appview_push_oldest_pending_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: age.Seconds()})
}

func (r *InMemoryMetricRecorder) ScheduledQueue(
	_ context.Context,
	status string,
	count int,
	due int,
	overdue int,
	oldestDueAge time.Duration,
) {
	status = safeScheduledStatus(status)
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_status", Kind: MetricKindGauge, Value: float64(count), Attributes: map[string]string{"status": status}})
	if status != "scheduled" {
		return
	}
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_due", Kind: MetricKindGauge, Value: float64(due)})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_overdue", Kind: MetricKindGauge, Value: float64(overdue)})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_oldest_due_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: nonNegativeDuration(oldestDueAge).Seconds()})
}

func (r *InMemoryMetricRecorder) ScheduledOperation(
	_ context.Context,
	operation string,
	result string,
	errorClass string,
	duration time.Duration,
) {
	attrs := scheduledOperationAttributes(operation, result, errorClass)
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_operations_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_operation_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) ScheduledPublication(
	_ context.Context,
	attempt int,
	startLatency time.Duration,
	duration time.Duration,
) {
	attrs := map[string]string{"attempt": safeScheduledAttempt(attempt)}
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_publication_start_latency_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(startLatency).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_publication_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) ScheduledCleanupQueue(
	_ context.Context,
	pending int,
	oldestAge time.Duration,
) {
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_cleanup_pending", Kind: MetricKindGauge, Value: float64(pending)})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_posts_cleanup_oldest_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: nonNegativeDuration(oldestAge).Seconds()})
}

func (r *InMemoryMetricRecorder) ScheduledImageValidation(
	_ context.Context,
	result string,
	format string,
	duration time.Duration,
	inFlight int,
) {
	if strings.TrimSpace(result) == "started" {
		r.record(MetricCall{Name: "craftsky_appview_scheduled_image_validation_in_flight", Kind: MetricKindGauge, Value: float64(safeScheduledImageInFlight(inFlight))})
		return
	}
	attrs := scheduledImageValidationAttributes(result, format)
	r.record(MetricCall{Name: "craftsky_appview_scheduled_image_validations_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_image_validation_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_scheduled_image_validation_in_flight", Kind: MetricKindGauge, Value: float64(safeScheduledImageInFlight(inFlight))})
}

func (r *InMemoryMetricRecorder) AuthRequestSweep(
	_ context.Context,
	pending int64,
	oldestAge time.Duration,
	deleted int64,
	failed bool,
) {
	if failed {
		r.record(MetricCall{Name: "craftsky_appview_auth_request_sweep_failures_total", Kind: MetricKindCounter, Value: 1})
		return
	}
	r.record(MetricCall{Name: "craftsky_appview_auth_requests_pending", Kind: MetricKindGauge, Unit: "request", Value: float64(max(pending, 0))})
	r.record(MetricCall{Name: "craftsky_appview_auth_requests_oldest_pending_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: nonNegativeDuration(oldestAge).Seconds()})
	if deleted > 0 {
		r.record(MetricCall{Name: "craftsky_appview_auth_request_sweep_deleted_total", Kind: MetricKindCounter, Unit: "request", Value: float64(deleted)})
	}
}

func (r *InMemoryMetricRecorder) TerminalPurge(
	_ context.Context,
	operation string,
	result string,
	errorCategory string,
	component string,
	didRole string,
	claims int,
	rowsAffected int64,
	remaining int64,
	complete bool,
) {
	attrs := terminalPurgeAttributes(operation, result, errorCategory, component, didRole, complete)
	r.record(MetricCall{
		Name: "craftsky_appview_terminal_purge_operations_total", Kind: MetricKindCounter,
		Value: 1, Attributes: attrs,
	})
	if claims > 0 {
		r.record(MetricCall{
			Name: "craftsky_appview_terminal_purge_claims", Kind: MetricKindDistribution,
			Unit: "item", Value: float64(claims), Attributes: attrs,
		})
	}
	if rowsAffected > 0 {
		r.record(MetricCall{
			Name: "craftsky_appview_terminal_purge_rows_affected_total", Kind: MetricKindCounter,
			Unit: "row", Value: float64(rowsAffected), Attributes: attrs,
		})
	}
	if strings.TrimSpace(operation) == "backlog" && result == "success" {
		r.record(MetricCall{
			Name: "craftsky_appview_terminal_purge_remaining", Kind: MetricKindGauge,
			Unit: "item", Value: float64(max(remaining, 0)),
		})
	}
}

func (r *InMemoryMetricRecorder) IdentityResolution(_ context.Context, mode, direction, result string, duration time.Duration) {
	attrs := identityResolutionAttributes(mode, direction, result)
	r.record(MetricCall{Name: "craftsky_appview_identity_resolutions_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_identity_resolution_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) IdentityCache(_ context.Context, result string, age time.Duration) {
	attrs := map[string]string{"result": safeIdentityCacheResult(result)}
	r.record(MetricCall{Name: "craftsky_appview_identity_cache_events_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_identity_cache_age_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(age).Seconds(), Attributes: attrs})
}

func (r *InMemoryMetricRecorder) FollowerGrowthCapture(
	_ context.Context,
	result string,
	errorCategory string,
	duration time.Duration,
	capturedProfileCount int64,
	latestSuccessfulRunAge *time.Duration,
) {
	attrs := map[string]string{
		"result":         safeFollowerGrowthResult(result),
		"error_category": safeFollowerGrowthErrorCategory(errorCategory),
	}
	r.record(MetricCall{Name: "craftsky_appview_follower_growth_captures_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_follower_growth_capture_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: nonNegativeDuration(duration).Seconds(), Attributes: attrs})
	r.record(MetricCall{Name: "craftsky_appview_follower_growth_captured_profiles", Kind: MetricKindGauge, Unit: "profile", Value: float64(max(capturedProfileCount, 0))})
	if latestSuccessfulRunAge != nil {
		r.record(MetricCall{Name: "craftsky_appview_follower_growth_latest_success_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: nonNegativeDuration(*latestSuccessfulRunAge).Seconds()})
	}
}

func (r *InMemoryMetricRecorder) TapIndexerRecord(_ context.Context, nsid, result, reason string, duration time.Duration) {
	attrs := map[string]string{
		"nsid":   SafeNSIDLabel(nsid),
		"result": safeMetricResult(result),
		"reason": SafeTapReason(reason),
	}
	r.record(MetricCall{Name: "craftsky_appview_tap_indexer_records_total", Kind: MetricKindCounter, Value: 1, Attributes: attrs})
	if duration > 0 {
		r.record(MetricCall{Name: "craftsky_appview_tap_indexer_handling_duration_seconds", Kind: MetricKindDistribution, Unit: "second", Value: duration.Seconds(), Attributes: map[string]string{
			"nsid":   attrs["nsid"],
			"result": attrs["result"],
		}})
	}
}

func (r *InMemoryMetricRecorder) RelationshipOperation(_ context.Context, operation, stage, result, errorClass string, duration time.Duration) {
	r.record(MetricCall{
		Name:  "craftsky_appview_relationship_operation_duration_seconds",
		Kind:  MetricKindDistribution,
		Unit:  "second",
		Value: duration.Seconds(),
		Attributes: map[string]string{
			"operation":   safeRelationshipOperation(operation),
			"stage":       safeRelationshipStage(stage),
			"result":      safeRelationshipResult(result),
			"error_class": safeRelationshipErrorClass(errorClass),
		},
	})
}

func (r *InMemoryMetricRecorder) ProfilePinOperation(
	_ context.Context,
	operation string,
	slot string,
	result string,
	errorClass string,
	duration time.Duration,
) {
	r.record(MetricCall{
		Name:       "craftsky_appview_profile_pin_operation_duration_seconds",
		Kind:       MetricKindDistribution,
		Unit:       "second",
		Value:      nonNegativeDuration(duration).Seconds(),
		Attributes: profilePinOperationAttributes(operation, slot, result, errorClass),
	})
}

func (r *InMemoryMetricRecorder) record(call MetricCall) {
	if r == nil {
		return
	}
	call.Attributes = cloneMetricAttributes(call.Attributes)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *InMemoryMetricRecorder) changeInFlight(attrs map[string]string, delta int) int {
	if r == nil {
		return 0
	}
	key := metricAttributeKey(attrs)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inFlight == nil {
		r.inFlight = map[string]int{}
	}
	next := r.inFlight[key] + delta
	if next < 0 {
		next = 0
	}
	r.inFlight[key] = next
	return next
}

type sentryMetricRecorder struct {
	hub      *sentry.Hub
	mu       sync.Mutex
	inFlight map[string]int
}

func newSentryMetricRecorder(hub *sentry.Hub) MetricRecorder {
	if hub == nil {
		return noopMetricRecorder{}
	}
	return &sentryMetricRecorder{hub: hub, inFlight: map[string]int{}}
}

func (r *sentryMetricRecorder) HTTPRequestStarted(ctx context.Context, method, routePattern string) {
	attrs := httpMetricAttributes(method, routePattern, 0)
	r.gauge(ctx, "craftsky_appview_http_requests_in_flight", float64(r.changeInFlight(attrs, 1)), "request", attrs)
}

func (r *sentryMetricRecorder) HTTPRequestEnded(ctx context.Context, method, routePattern string) {
	attrs := httpMetricAttributes(method, routePattern, 0)
	r.gauge(ctx, "craftsky_appview_http_requests_in_flight", float64(r.changeInFlight(attrs, -1)), "request", attrs)
}

func (r *sentryMetricRecorder) HTTPRequestFinished(ctx context.Context, method, routePattern string, status int, duration time.Duration, responseBytes int) {
	attrs := httpMetricAttributes(method, routePattern, status)
	r.count(ctx, "craftsky_appview_http_requests_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_http_request_duration_seconds", duration.Seconds(), "second", attrs)
	r.distribution(ctx, "craftsky_appview_http_response_size_bytes", float64(responseBytes), "byte", attrs)
}

func (r *sentryMetricRecorder) DBOperation(ctx context.Context, operation, routePattern, resultClass string, duration time.Duration) {
	r.distribution(ctx, "craftsky_appview_db_operation_duration_seconds", duration.Seconds(), "second", map[string]string{
		"operation":     safeMetricOperation(operation),
		"route_pattern": safeMetricRoute(routePattern),
		"result":        safeMetricResult(resultClass),
	})
}

func (r *sentryMetricRecorder) PDSOperation(ctx context.Context, operation, stage, result, category string, duration time.Duration) {
	r.distribution(ctx, "craftsky_appview_pds_write_duration_seconds", duration.Seconds(), "second", map[string]string{
		"operation": safeMetricOperation(operation),
		"stage":     safeMetricStage(stage),
		"result":    safeMetricResult(result),
		"category":  safeMetricCategory(category),
	})
}

func (r *sentryMetricRecorder) TapConnected(ctx context.Context, connected bool) {
	value := float64(0)
	if connected {
		value = 1
	}
	r.gauge(ctx, "craftsky_appview_tap_connected", value, "", nil)
}

func (r *sentryMetricRecorder) TapLastEventAge(ctx context.Context, age time.Duration) {
	if age < 0 {
		age = 0
	}
	r.gauge(ctx, "craftsky_appview_tap_last_event_age_seconds", age.Seconds(), "second", nil)
}

func (r *sentryMetricRecorder) TapReconnect(ctx context.Context) {
	r.count(ctx, "craftsky_appview_tap_reconnects_total", 1, "", nil)
}

func (r *sentryMetricRecorder) TapEventReceived(ctx context.Context, eventType string) {
	r.count(ctx, "craftsky_appview_tap_events_received_total", 1, "", map[string]string{"type": SafeTapEventType(eventType)})
}

func (r *sentryMetricRecorder) TapEventAcknowledged(ctx context.Context, result string) {
	if safeMetricResult(result) == "error" {
		r.count(ctx, "craftsky_appview_tap_ack_failures_total", 1, "", nil)
		return
	}
	r.count(ctx, "craftsky_appview_tap_events_acknowledged_total", 1, "", nil)
}

func (r *sentryMetricRecorder) TapIndexerRecord(ctx context.Context, nsid, result, reason string, duration time.Duration) {
	attrs := map[string]string{"nsid": SafeNSIDLabel(nsid), "result": safeMetricResult(result), "reason": SafeTapReason(reason)}
	r.count(ctx, "craftsky_appview_tap_indexer_records_total", 1, "", attrs)
	if duration > 0 {
		r.distribution(ctx, "craftsky_appview_tap_indexer_handling_duration_seconds", duration.Seconds(), "second", map[string]string{
			"nsid":   attrs["nsid"],
			"result": attrs["result"],
		})
	}
}
func (r *sentryMetricRecorder) RelationshipOperation(ctx context.Context, operation, stage, result, errorClass string, duration time.Duration) {
	r.distribution(ctx, "craftsky_appview_relationship_operation_duration_seconds", duration.Seconds(), "second", map[string]string{
		"operation":   safeRelationshipOperation(operation),
		"stage":       safeRelationshipStage(stage),
		"result":      safeRelationshipResult(result),
		"error_class": safeRelationshipErrorClass(errorClass),
	})
}
func (r *sentryMetricRecorder) ProfilePinOperation(ctx context.Context, operation, slot, result, errorClass string, duration time.Duration) {
	r.distribution(
		ctx,
		"craftsky_appview_profile_pin_operation_duration_seconds",
		nonNegativeDuration(duration).Seconds(),
		"second",
		profilePinOperationAttributes(operation, slot, result, errorClass),
	)
}
func (r *sentryMetricRecorder) NotificationDecision(ctx context.Context, category, result string) {
	r.count(ctx, "craftsky_appview_notifications_total", 1, "", map[string]string{"category": safeMetricCategory(category), "result": safeMetricResult(result)})
}
func (r *sentryMetricRecorder) PushDelivery(ctx context.Context, platform, result string) {
	r.count(ctx, "craftsky_appview_push_deliveries_total", 1, "", map[string]string{"platform": safeMetricCategory(platform), "result": safeMetricResult(result)})
}
func (r *sentryMetricRecorder) PushOperation(
	ctx context.Context,
	stage string,
	platform string,
	semantics string,
	outcome string,
	duration time.Duration,
	count int64,
) {
	attrs := pushOperationAttributes(stage, platform, semantics, outcome)
	if count < 1 {
		count = 1
	}
	r.count(ctx, "craftsky_appview_push_operations_total", count, "", attrs)
	r.distribution(
		ctx,
		"craftsky_appview_push_operation_duration_seconds",
		nonNegativeDuration(duration).Seconds(),
		"second",
		attrs,
	)
}
func (r *sentryMetricRecorder) PushQueue(ctx context.Context, pending int, age time.Duration) {
	r.gauge(ctx, "craftsky_appview_push_pending", float64(pending), "", nil)
	r.gauge(ctx, "craftsky_appview_push_oldest_pending_age_seconds", age.Seconds(), "second", nil)
}
func (r *sentryMetricRecorder) ScheduledQueue(ctx context.Context, status string, count, due, overdue int, oldestDueAge time.Duration) {
	status = safeScheduledStatus(status)
	r.gauge(ctx, "craftsky_appview_scheduled_posts_status", float64(count), "", map[string]string{"status": status})
	if status != "scheduled" {
		return
	}
	r.gauge(ctx, "craftsky_appview_scheduled_posts_due", float64(due), "", nil)
	r.gauge(ctx, "craftsky_appview_scheduled_posts_overdue", float64(overdue), "", nil)
	r.gauge(ctx, "craftsky_appview_scheduled_posts_oldest_due_age_seconds", nonNegativeDuration(oldestDueAge).Seconds(), "second", nil)
}
func (r *sentryMetricRecorder) ScheduledOperation(ctx context.Context, operation, result, errorClass string, duration time.Duration) {
	attrs := scheduledOperationAttributes(operation, result, errorClass)
	r.count(ctx, "craftsky_appview_scheduled_posts_operations_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_scheduled_posts_operation_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}
func (r *sentryMetricRecorder) ScheduledPublication(ctx context.Context, attempt int, startLatency, duration time.Duration) {
	attrs := map[string]string{"attempt": safeScheduledAttempt(attempt)}
	r.distribution(ctx, "craftsky_appview_scheduled_posts_publication_start_latency_seconds", nonNegativeDuration(startLatency).Seconds(), "second", attrs)
	r.distribution(ctx, "craftsky_appview_scheduled_posts_publication_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}
func (r *sentryMetricRecorder) ScheduledCleanupQueue(ctx context.Context, pending int, oldestAge time.Duration) {
	r.gauge(ctx, "craftsky_appview_scheduled_posts_cleanup_pending", float64(pending), "", nil)
	r.gauge(ctx, "craftsky_appview_scheduled_posts_cleanup_oldest_age_seconds", nonNegativeDuration(oldestAge).Seconds(), "second", nil)
}
func (r *sentryMetricRecorder) ScheduledImageValidation(ctx context.Context, result, format string, duration time.Duration, inFlight int) {
	if strings.TrimSpace(result) == "started" {
		r.gauge(ctx, "craftsky_appview_scheduled_image_validation_in_flight", float64(safeScheduledImageInFlight(inFlight)), "", nil)
		return
	}
	attrs := scheduledImageValidationAttributes(result, format)
	r.count(ctx, "craftsky_appview_scheduled_image_validations_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_scheduled_image_validation_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
	r.gauge(ctx, "craftsky_appview_scheduled_image_validation_in_flight", float64(safeScheduledImageInFlight(inFlight)), "", nil)
}
func (r *sentryMetricRecorder) AuthRequestSweep(ctx context.Context, pending int64, oldestAge time.Duration, deleted int64, failed bool) {
	if failed {
		r.count(ctx, "craftsky_appview_auth_request_sweep_failures_total", 1, "", nil)
		return
	}
	r.gauge(ctx, "craftsky_appview_auth_requests_pending", float64(max(pending, 0)), "request", nil)
	r.gauge(ctx, "craftsky_appview_auth_requests_oldest_pending_age_seconds", nonNegativeDuration(oldestAge).Seconds(), "second", nil)
	if deleted > 0 {
		r.count(ctx, "craftsky_appview_auth_request_sweep_deleted_total", deleted, "request", nil)
	}
}

func (r *sentryMetricRecorder) TerminalPurge(
	ctx context.Context,
	operation string,
	result string,
	errorCategory string,
	component string,
	didRole string,
	claims int,
	rowsAffected int64,
	remaining int64,
	complete bool,
) {
	attrs := terminalPurgeAttributes(operation, result, errorCategory, component, didRole, complete)
	r.count(ctx, "craftsky_appview_terminal_purge_operations_total", 1, "", attrs)
	if claims > 0 {
		r.distribution(ctx, "craftsky_appview_terminal_purge_claims", float64(claims), "item", attrs)
	}
	if rowsAffected > 0 {
		r.count(ctx, "craftsky_appview_terminal_purge_rows_affected_total", rowsAffected, "row", attrs)
	}
	if strings.TrimSpace(operation) == "backlog" && result == "success" {
		r.gauge(ctx, "craftsky_appview_terminal_purge_remaining", float64(max(remaining, 0)), "item", nil)
	}
}

func (r *sentryMetricRecorder) IdentityResolution(ctx context.Context, mode, direction, result string, duration time.Duration) {
	attrs := identityResolutionAttributes(mode, direction, result)
	r.count(ctx, "craftsky_appview_identity_resolutions_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_identity_resolution_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
}

func (r *sentryMetricRecorder) IdentityCache(ctx context.Context, result string, age time.Duration) {
	attrs := map[string]string{"result": safeIdentityCacheResult(result)}
	r.count(ctx, "craftsky_appview_identity_cache_events_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_identity_cache_age_seconds", nonNegativeDuration(age).Seconds(), "second", attrs)
}

func (r *sentryMetricRecorder) FollowerGrowthCapture(
	ctx context.Context,
	result string,
	errorCategory string,
	duration time.Duration,
	capturedProfileCount int64,
	latestSuccessfulRunAge *time.Duration,
) {
	attrs := map[string]string{
		"result":         safeFollowerGrowthResult(result),
		"error_category": safeFollowerGrowthErrorCategory(errorCategory),
	}
	r.count(ctx, "craftsky_appview_follower_growth_captures_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_follower_growth_capture_duration_seconds", nonNegativeDuration(duration).Seconds(), "second", attrs)
	r.gauge(ctx, "craftsky_appview_follower_growth_captured_profiles", float64(max(capturedProfileCount, 0)), "profile", nil)
	if latestSuccessfulRunAge != nil {
		r.gauge(ctx, "craftsky_appview_follower_growth_latest_success_age_seconds", nonNegativeDuration(*latestSuccessfulRunAge).Seconds(), "second", nil)
	}
}

func identityResolutionAttributes(mode, direction, result string) map[string]string {
	switch mode {
	case "cached", "authoritative":
	default:
		mode = "other"
	}
	switch direction {
	case "handle_to_did", "did_to_handle":
	default:
		direction = "other"
	}
	return map[string]string{
		"mode": mode, "direction": direction, "result": safeMetricResult(result),
	}
}

func safeIdentityCacheResult(result string) string {
	switch result {
	case "hit", "miss", "reassigned", "refresh_success", "refresh_retry":
		return result
	default:
		return "other"
	}
}

func (r *sentryMetricRecorder) count(ctx context.Context, name string, value int64, unit string, attrs map[string]string) {
	options := metricOptions(unit, attrs)
	sentry.NewMeter(r.context(ctx)).Count(name, value, options...)
}

func (r *sentryMetricRecorder) gauge(ctx context.Context, name string, value float64, unit string, attrs map[string]string) {
	options := metricOptions(unit, attrs)
	sentry.NewMeter(r.context(ctx)).Gauge(name, value, options...)
}

func (r *sentryMetricRecorder) distribution(ctx context.Context, name string, value float64, unit string, attrs map[string]string) {
	options := metricOptions(unit, attrs)
	sentry.NewMeter(r.context(ctx)).Distribution(name, value, options...)
}

func (r *sentryMetricRecorder) context(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return sentry.SetHubOnContext(ctx, r.hub)
}

func (r *sentryMetricRecorder) changeInFlight(attrs map[string]string, delta int) int {
	if r == nil {
		return 0
	}
	key := metricAttributeKey(attrs)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inFlight == nil {
		r.inFlight = map[string]int{}
	}
	next := r.inFlight[key] + delta
	if next < 0 {
		next = 0
	}
	r.inFlight[key] = next
	return next
}

func metricOptions(unit string, attrs map[string]string) []sentry.MeterOption {
	var options []sentry.MeterOption
	if unit != "" {
		options = append(options, sentry.WithUnit(unit))
	}
	if len(attrs) > 0 {
		builders := make([]attribute.Builder, 0, len(attrs))
		for key, value := range attrs {
			builders = append(builders, attribute.String(key, value))
		}
		options = append(options, sentry.WithAttributes(builders...))
	}
	return options
}

func httpMetricAttributes(method, routePattern string, status int) map[string]string {
	attrs := map[string]string{
		"method":        safeHTTPMethod(method),
		"route_pattern": safeMetricRoute(routePattern),
	}
	if status > 0 {
		attrs["status"] = safeHTTPStatus(status)
		attrs["status_class"] = safeHTTPStatusClass(status)
	}
	return attrs
}

func safeHTTPMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return strings.ToUpper(strings.TrimSpace(method))
	default:
		return "OTHER"
	}
}

func safeMetricRoute(routePattern string) string {
	routePattern = strings.TrimSpace(routePattern)
	if routePattern == "" {
		return unmatchedRoutePattern
	}
	if strings.Contains(routePattern, "?") || strings.Contains(routePattern, "did:") || strings.Contains(routePattern, "@") {
		return unmatchedRoutePattern
	}
	return routePattern
}

func safeHTTPStatus(status int) string {
	if status < 100 || status > 599 {
		return "000"
	}
	return strconv.Itoa(status)
}

func safeHTTPStatusClass(status int) string {
	if status < 100 || status > 599 {
		return "unknown"
	}
	return strconv.Itoa(status/100) + "xx"
}

func safeHTTPStatusClassString(statusClass string) string {
	switch strings.TrimSpace(statusClass) {
	case "1xx", "2xx", "3xx", "4xx", "5xx":
		return strings.TrimSpace(statusClass)
	default:
		return "unknown"
	}
}

func safeMetricOperation(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" || strings.ContainsAny(operation, "/:#?@'\" \t\n") {
		return "unknown"
	}
	return operation
}

func safeRelationshipOperation(operation string) string {
	switch strings.TrimSpace(operation) {
	case "mute", "unmute", "block", "unblock", "index_create", "index_update", "index_delete", "backfill", "authorization_follow", "authorization_like", "authorization_repost", "authorization_reply", "authorization_quote", "authorization_mention", "notification_suppression", "push_cancellation":
		return strings.TrimSpace(operation)
	default:
		return "unknown"
	}
}

func safeRelationshipStage(stage string) string {
	switch strings.TrimSpace(stage) {
	case "request", "membership", "policy", "decode", "validate", "store", "pds", "delivery", "lag", "backfill", "complete":
		return strings.TrimSpace(stage)
	default:
		return "unknown"
	}
}

func safeRelationshipResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "error", "denied", "suppressed", "canceled", "none", "some", "many":
		return strings.TrimSpace(result)
	default:
		return "unknown"
	}
}

func safeRelationshipErrorClass(errorClass string) string {
	switch strings.TrimSpace(errorClass) {
	case "none", "validation", "membership", "policy", "store", "pds", "timeout", "canceled", "internal":
		return strings.TrimSpace(errorClass)
	default:
		return "unknown"
	}
}

func safeMetricStage(stage string) string {
	stage = strings.TrimSpace(stage)
	if stage == "" || strings.ContainsAny(stage, "/:#?@'\" \t\n") {
		return "unknown"
	}
	return stage
}

func safeMetricCategory(category string) string {
	category = strings.TrimSpace(category)
	if category == "" || strings.ContainsAny(category, "/:#?@'\" \t\n") {
		return "unknown"
	}
	return category
}

func safeMetricResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "error", "canceled", "none", "indexed", "skipped", "some", "one", "many":
		return strings.TrimSpace(result)
	default:
		return "unknown"
	}
}

func pushOperationAttributes(
	stage string,
	platform string,
	semantics string,
	outcome string,
) map[string]string {
	stage = strings.TrimSpace(stage)
	switch stage {
	case "lease_recovery", "lease", "send", "finalization":
	default:
		stage = "other"
	}
	platform = strings.TrimSpace(platform)
	switch platform {
	case "ios", "android", "none":
	default:
		platform = "other"
	}
	semantics = strings.TrimSpace(semantics)
	switch semantics {
	case "unique_event", "replacement", "none":
	default:
		semantics = "other"
	}
	outcome = strings.TrimSpace(outcome)
	switch outcome {
	case "reclaimed", "claimed", "empty", "insufficient_window",
		"success", "retryable", "invalidToken", "permanentFailure",
		"finalized", "stale", "accepted_unfinalized", "error",
		"lifecycle_inactive", "cancelled", "unknown":
	default:
		outcome = "other"
	}
	return map[string]string{
		"stage": stage, "platform": platform,
		"semantics": semantics, "outcome": outcome,
	}
}

func terminalPurgeAttributes(
	operation string,
	result string,
	errorCategory string,
	component string,
	didRole string,
	complete bool,
) map[string]string {
	attrs := map[string]string{
		"operation":      safeMetricOperation(operation),
		"result":         safeTerminalPurgeResult(result),
		"error_category": safeMetricCategory(errorCategory),
		"component":      safeMetricCategory(component),
		"did_role":       safeMetricCategory(didRole),
		"complete":       strconv.FormatBool(complete),
	}
	return attrs
}

func safeTerminalPurgeResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "failure":
		return strings.TrimSpace(result)
	default:
		return "unknown"
	}
}

func safeScheduledStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "scheduled", "publishing", "retrying", "needs_attention":
		return strings.TrimSpace(status)
	default:
		return "unknown"
	}
}

func scheduledImageValidationAttributes(result, format string) map[string]string {
	switch strings.TrimSpace(result) {
	case "success", "invalid", "saturated", "cancelled":
		result = strings.TrimSpace(result)
	default:
		result = "unknown"
	}
	switch strings.TrimSpace(format) {
	case "jpeg", "png", "webp":
		format = strings.TrimSpace(format)
	default:
		format = "unknown"
	}
	return map[string]string{"result": result, "format": format}
}

func safeScheduledImageInFlight(inFlight int) int {
	switch {
	case inFlight < 0:
		return 0
	case inFlight > 1:
		return 1
	default:
		return inFlight
	}
}

func scheduledOperationAttributes(operation, result, errorClass string) map[string]string {
	switch strings.TrimSpace(operation) {
	case "claim", "publish", "retry", "needs_attention", "recover", "stale_worker", "cleanup", "cleanup_sweep":
		operation = strings.TrimSpace(operation)
	default:
		operation = "unknown"
	}
	switch strings.TrimSpace(result) {
	case "success", "failure", "stale":
		result = strings.TrimSpace(result)
	default:
		result = "unknown"
	}
	switch strings.TrimSpace(errorClass) {
	case "none", "auth_unavailable", "object_unavailable", "pds_unavailable", "dependency_unavailable", "policy_invalid", "media_invalid", "record_conflict", "lease_expired", "stale_worker", "object_delete_failed":
		errorClass = strings.TrimSpace(errorClass)
	default:
		errorClass = "unknown"
	}
	return map[string]string{
		"operation":   operation,
		"result":      result,
		"error_class": errorClass,
	}
}

func profilePinOperationAttributes(operation, slot, result, errorClass string) map[string]string {
	operation = strings.TrimSpace(operation)
	switch operation {
	case "pin", "replace", "unpin":
	default:
		operation = "pin"
	}
	slot = strings.TrimSpace(slot)
	switch slot {
	case "standard", "project":
	default:
		slot = "unknown"
	}
	result = strings.TrimSpace(result)
	switch result {
	case "success", "noop", "rejected", "error":
	default:
		result = "error"
	}
	errorClass = strings.TrimSpace(errorClass)
	switch errorClass {
	case "none", "forbidden", "not_found", "policy", "store":
	default:
		errorClass = "store"
	}
	return map[string]string{
		"operation":   operation,
		"slot":        slot,
		"result":      result,
		"error_class": errorClass,
	}
}

func safeScheduledAttempt(attempt int) string {
	if attempt < 1 || attempt > 6 {
		return "unknown"
	}
	return strconv.Itoa(attempt)
}

func nonNegativeDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}

func cloneMetricAttributes(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for key, value := range attrs {
		if key == "run_id" || key == "" || strings.ContainsAny(key, " \t\n") {
			continue
		}
		out[key] = value
	}
	return out
}

func metricAttributeKey(attrs map[string]string) string {
	normalized := cloneMetricAttributes(attrs)
	return normalized["method"] + "\x00" + normalized["route_pattern"]
}
