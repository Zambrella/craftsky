package observability

import (
	"context"
	"strconv"
	"time"
)

type linkPreviewMetricRecorder interface {
	LinkPreview(context.Context, map[string]string, time.Duration)
}

func (o *Observer) ObserveLinkPreview(
	stage, result, errorClass string,
	status, redirects, bytes int,
	duration time.Duration,
) {
	if o == nil {
		return
	}
	attrs := map[string]string{
		"stage":       boundedLinkPreviewValue(stage, "admission", "fetch", "parse", "image", "complete", "upload", "schedule", "publish", "cleanup"),
		"result":      boundedLinkPreviewValue(result, "success", "rejected", "failed", "timeout", "disabled", "rate_limited"),
		"error_class": boundedLinkPreviewValue(errorClass, "none", "validation", "policy", "dns", "redirect", "response", "image", "timeout", "upstream", "auth", "quota", "internal"),
		"status":      linkPreviewStatusClass(status),
		"redirects":   linkPreviewCountBucket(redirects),
		"bytes":       linkPreviewBytesBucket(bytes),
	}
	if recorder, ok := o.metricRecorder.(linkPreviewMetricRecorder); ok {
		recorder.LinkPreview(context.Background(), attrs, nonNegativeDuration(duration))
	}
}

func boundedLinkPreviewValue(value string, allowed ...string) string {
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return "unknown"
}

func linkPreviewStatusClass(status int) string {
	if status < 100 || status > 599 {
		return "unknown"
	}
	return strconv.Itoa(status/100) + "xx"
}

func linkPreviewCountBucket(count int) string {
	switch {
	case count <= 0:
		return "0"
	case count == 1:
		return "1"
	case count <= 3:
		return "2-3"
	case count <= 5:
		return "4-5"
	default:
		return "6+"
	}
}

func linkPreviewBytesBucket(bytes int) string {
	switch {
	case bytes <= 0:
		return "0"
	case bytes <= 16*1024:
		return "1-16k"
	case bytes <= 256*1024:
		return "16-256k"
	case bytes <= 1_000_000:
		return "256k-1m"
	default:
		return "1m+"
	}
}

func (r *InMemoryMetricRecorder) LinkPreview(_ context.Context, attrs map[string]string, duration time.Duration) {
	r.record(MetricCall{Name: "craftsky_appview_link_previews_total", Kind: MetricKindCounter, Value: 1, Attributes: cloneMetricAttributes(attrs)})
	r.record(MetricCall{Name: "craftsky_appview_link_preview_duration", Kind: MetricKindDistribution, Unit: "second", Value: duration.Seconds(), Attributes: cloneMetricAttributes(attrs)})
}

func (r *sentryMetricRecorder) LinkPreview(ctx context.Context, attrs map[string]string, duration time.Duration) {
	r.count(ctx, "craftsky_appview_link_previews_total", 1, "", attrs)
	r.distribution(ctx, "craftsky_appview_link_preview_duration", duration.Seconds(), "second", attrs)
}
