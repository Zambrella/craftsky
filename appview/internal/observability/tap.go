package observability

import (
	"context"
	"math/rand"
	"strings"
	"time"
)

const (
	NSIDLabelUnsupported = "unsupported"
	NSIDLabelMalformed   = "malformed"
)

var knownNSIDLabels = map[string]struct{}{
	"social.craftsky.actor.profile": {},
	"social.craftsky.feed.post":     {},
	"social.craftsky.feed.like":     {},
	"social.craftsky.feed.repost":   {},
	"app.bsky.graph.follow":         {},
	"app.bsky.actor.profile":        {},
}

func SafeNSIDLabel(nsid string) string {
	nsid = strings.TrimSpace(nsid)
	if nsid == "" || strings.ContainsAny(nsid, "/:#?") || !strings.Contains(nsid, ".") {
		return NSIDLabelMalformed
	}
	if _, ok := knownNSIDLabels[nsid]; ok {
		return nsid
	}
	return NSIDLabelUnsupported
}

func SafeTapEventType(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "record":
		return "record"
	case "identity":
		return "identity"
	default:
		return "other"
	}
}

func SafeTapReason(reason string) string {
	switch reason {
	case "none", "malformed", "identity", "unsupported", "indexer_error", "ack_error", "panic":
		return reason
	default:
		return "other"
	}
}

func (o *Observer) StartTapSpan(ctx context.Context, operation string, force bool) (context.Context, *Span) {
	if o == nil || !o.tracingEnabled {
		return ctx, &Span{}
	}
	if !force {
		if !o.tapTracingEnabled || o.tapTracesSampleRate <= 0 {
			return ctx, &Span{}
		}
		if o.tapTracesSampleRate < 1 && rand.Float64() >= o.tapTracesSampleRate {
			return ctx, &Span{}
		}
	}
	return o.StartSpan(ctx, SpanContext{
		Operation: operation,
		Component: "tap",
		Attributes: EventContext{
			"component": "tap",
			"operation": operation,
		},
	})
}

func (o *Observer) SetTapConnected(connected bool) {
	if o == nil {
		return
	}
	o.metricRecorder.TapConnected(context.Background(), connected)
}

func (o *Observer) ObserveTapReconnect() {
	if o == nil {
		return
	}
	o.metricRecorder.TapReconnect(context.Background())
}

func (o *Observer) ObserveTapEventReceived(eventType string) {
	if o == nil {
		return
	}
	eventType = SafeTapEventType(eventType)
	o.metricRecorder.TapEventReceived(context.Background(), eventType)
}

func (o *Observer) ObserveTapEventAcknowledged(err error) {
	if o == nil {
		return
	}
	if err != nil {
		o.metricRecorder.TapEventAcknowledged(context.Background(), "error")
		return
	}
	o.metricRecorder.TapEventAcknowledged(context.Background(), "success")
}

func (o *Observer) ObserveTapLastEventAt(t time.Time) {
	if o == nil || t.IsZero() {
		return
	}
	o.tapLastEventAt = t
}

func (o *Observer) ObserveIndexerSkipped(nsid string, reason string) {
	if o == nil {
		return
	}
	label := SafeNSIDLabel(nsid)
	reason = SafeTapReason(reason)
	o.metricRecorder.TapIndexerRecord(context.Background(), label, "skipped", reason, 0)
}

func (o *Observer) ObserveIndexerHandled(nsid string, err error, duration time.Duration) {
	if o == nil {
		return
	}
	result := "indexed"
	reason := "none"
	if err != nil {
		result = "error"
		reason = "indexer_error"
	}
	label := SafeNSIDLabel(nsid)
	o.metricRecorder.TapIndexerRecord(context.Background(), label, result, reason, duration)
}
