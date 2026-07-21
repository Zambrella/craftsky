package instagram

import (
	"testing"
	"time"
)

func TestNextWebhookRetryUsesFixedBackoffAttemptAndAgeBounds(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		attempts int
		want     time.Duration
		ok       bool
	}{
		{attempts: 1, want: time.Second, ok: true},
		{attempts: 2, want: 2 * time.Second, ok: true},
		{attempts: 3, want: 4 * time.Second, ok: true},
		{attempts: 4, want: 8 * time.Second, ok: true},
		{attempts: 5, ok: false},
		{attempts: 6, ok: false},
	} {
		now := started.Add(time.Minute)
		got, ok := NextWebhookRetry(now, started, test.attempts, 0)
		if ok != test.ok {
			t.Errorf("attempt %d ok = %t, want %t", test.attempts, ok, test.ok)
			continue
		}
		if ok && !got.Equal(now.Add(test.want)) {
			t.Errorf("attempt %d next = %s, want %s", test.attempts, got, now.Add(test.want))
		}
	}

	now := started.Add(time.Minute)
	if got, ok := NextWebhookRetry(now, started, 1, 10*time.Minute); !ok || !got.Equal(now.Add(WebhookMaxBackoff)) {
		t.Fatalf("provider delay cap = (%s, %t), want %s", got, ok, now.Add(WebhookMaxBackoff))
	}
	if got, ok := NextWebhookRetry(started.Add(WebhookMaxProcessingAge), started, 1, 0); ok || !got.IsZero() {
		t.Fatalf("retry at maximum age = (%s, %t), want zero/false", got, ok)
	}
	nearDeadline := started.Add(WebhookMaxProcessingAge - time.Second)
	if got, ok := NextWebhookRetry(nearDeadline, started, 1, 0); ok || !got.IsZero() {
		t.Fatalf("retry reaching maximum age = (%s, %t), want zero/false", got, ok)
	}
}
