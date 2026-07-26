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
	NotificationDecision(ctx context.Context, category, result string)
	PushDelivery(ctx context.Context, platform, result string)
	PushQueue(ctx context.Context, pending int, oldestAge time.Duration)
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
func (noopMetricRecorder) NotificationDecision(context.Context, string, string) {}
func (noopMetricRecorder) PushDelivery(context.Context, string, string)         {}
func (noopMetricRecorder) PushQueue(context.Context, int, time.Duration)        {}

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
func (r *InMemoryMetricRecorder) PushQueue(_ context.Context, pending int, age time.Duration) {
	r.record(MetricCall{Name: "craftsky_appview_push_pending", Kind: MetricKindGauge, Value: float64(pending)})
	r.record(MetricCall{Name: "craftsky_appview_push_oldest_pending_age_seconds", Kind: MetricKindGauge, Unit: "second", Value: age.Seconds()})
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
func (r *sentryMetricRecorder) NotificationDecision(ctx context.Context, category, result string) {
	r.count(ctx, "craftsky_appview_notifications_total", 1, "", map[string]string{"category": safeMetricCategory(category), "result": safeMetricResult(result)})
}
func (r *sentryMetricRecorder) PushDelivery(ctx context.Context, platform, result string) {
	r.count(ctx, "craftsky_appview_push_deliveries_total", 1, "", map[string]string{"platform": safeMetricCategory(platform), "result": safeMetricResult(result)})
}
func (r *sentryMetricRecorder) PushQueue(ctx context.Context, pending int, age time.Duration) {
	r.gauge(ctx, "craftsky_appview_push_pending", float64(pending), "", nil)
	r.gauge(ctx, "craftsky_appview_push_oldest_pending_age_seconds", age.Seconds(), "second", nil)
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
