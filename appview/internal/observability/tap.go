package observability

import (
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

func (o *Observer) SetTapConnected(connected bool) {
	if o == nil {
		return
	}
	if connected {
		o.tapConnected.Set(1)
		return
	}
	o.tapConnected.Set(0)
}

func (o *Observer) ObserveTapReconnect() {
	if o == nil {
		return
	}
	o.tapReconnects.Inc()
}

func (o *Observer) ObserveTapEventReceived(eventType string) {
	if o == nil {
		return
	}
	o.tapEventsReceived.WithLabelValues(SafeTapEventType(eventType)).Inc()
}

func (o *Observer) ObserveTapEventAcknowledged(err error) {
	if o == nil {
		return
	}
	if err != nil {
		o.tapAckFailures.Inc()
		return
	}
	o.tapEventsAcked.Inc()
}

func (o *Observer) ObserveTapLastEventAt(t time.Time) {
	if o == nil || t.IsZero() {
		return
	}
	o.tapLastEventUnixNano.Store(t.UnixNano())
}

func (o *Observer) ObserveIndexerSkipped(nsid string, reason string) {
	if o == nil {
		return
	}
	o.tapIndexerRecords.WithLabelValues(SafeNSIDLabel(nsid), "skipped", SafeTapReason(reason)).Inc()
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
	o.tapIndexerRecords.WithLabelValues(label, result, reason).Inc()
	o.tapIndexerDuration.WithLabelValues(label, result).Observe(duration.Seconds())
}
