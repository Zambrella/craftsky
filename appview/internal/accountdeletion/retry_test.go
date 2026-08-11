package accountdeletion

import (
	"testing"
	"time"
)

func TestRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultRetryPolicy()
	wantDelays := []time.Duration{0, time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour}
	if len(policy.Delays) != len(wantDelays) {
		t.Fatalf("retry delay count = %d, want %d", len(policy.Delays), len(wantDelays))
	}
	for index, want := range wantDelays {
		if policy.Delays[index] != want {
			t.Fatalf("retry delay %d = %s, want %s", index, policy.Delays[index], want)
		}
	}

	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	first := policy.Decide(now, "job-1", 0, FailureTransient)
	if first.Action != RetrySchedule || !first.At.Equal(now) {
		t.Fatalf("first attempt = %+v, want immediate schedule", first)
	}

	for attempt := 1; attempt < len(wantDelays); attempt++ {
		got := policy.Decide(now, "job-1", attempt, FailureTransient)
		if got.Action != RetrySchedule {
			t.Fatalf("attempt %d action = %q, want schedule", attempt, got.Action)
		}
		jitter := got.At.Sub(now) - wantDelays[attempt]
		if jitter < -policy.MaxJitter || jitter > policy.MaxJitter {
			t.Fatalf("attempt %d jitter = %s, outside ±%s", attempt, jitter, policy.MaxJitter)
		}
		again := policy.Decide(now, "job-1", attempt, FailureTransient)
		if got != again {
			t.Fatalf("attempt %d jitter is not deterministic: %+v then %+v", attempt, got, again)
		}
	}

	exhausted := policy.Decide(now, "job-1", len(wantDelays), FailureTransient)
	if exhausted.Action != RetryNeedsAttention || exhausted.Reason != AttentionRetriesExhausted {
		t.Fatalf("exhausted decision = %+v", exhausted)
	}
	permanent := policy.Decide(now, "job-1", 0, FailurePermanent)
	if permanent.Action != RetryNeedsAttention || permanent.Reason != AttentionPermanentFailure {
		t.Fatalf("permanent decision = %+v", permanent)
	}
	oauth := policy.Decide(now, "job-1", 0, FailureOAuthUnusable)
	if oauth.Action != RetryNeedsReauthentication || oauth.Reason != AttentionOAuthUnusable {
		t.Fatalf("OAuth decision = %+v", oauth)
	}
}
