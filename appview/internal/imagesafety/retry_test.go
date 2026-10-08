package imagesafety

import (
	"testing"
	"time"
)

func TestRetryPolicyUsesBoundedExponentialBackoff(t *testing.T) {
	t.Parallel()

	policy := RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: time.Second,
		MaxBackoff:     5 * time.Second,
	}
	tests := []struct {
		attempt int
		want    time.Duration
		retry   bool
	}{
		{attempt: 1, want: time.Second, retry: true},
		{attempt: 2, want: 2 * time.Second, retry: true},
		{attempt: 3, want: 4 * time.Second, retry: true},
		{attempt: 4, want: 5 * time.Second, retry: true},
		{attempt: 5},
		{attempt: 6},
		{attempt: 0},
	}

	for _, test := range tests {
		delay, retry := policy.Next(test.attempt)
		if retry != test.retry || delay != test.want {
			t.Errorf("RetryPolicy.Next(%d) = (%s, %t), want (%s, %t)", test.attempt, delay, retry, test.want, test.retry)
		}
	}

	for _, invalid := range []RetryPolicy{
		{},
		{MaxAttempts: 1, InitialBackoff: 0, MaxBackoff: time.Second},
		{MaxAttempts: 1, InitialBackoff: 2 * time.Second, MaxBackoff: time.Second},
	} {
		if delay, retry := invalid.Next(1); retry || delay != 0 {
			t.Errorf("invalid RetryPolicy.Next(1) = (%s, %t), want (0, false)", delay, retry)
		}
	}
}
