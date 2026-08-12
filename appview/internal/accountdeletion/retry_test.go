package accountdeletion

import (
	"testing"
	"time"
)

func TestRetryPolicyCapsDelayWithoutEnteringAttentionState(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	policy := RetryPolicy{Delays: []time.Duration{0, time.Minute, 6 * time.Hour}}

	if got := policy.Next(now, "job", 100); !got.Equal(now.Add(6 * time.Hour)) {
		t.Fatalf("capped retry = %s, want %s", got, now.Add(6*time.Hour))
	}
}
