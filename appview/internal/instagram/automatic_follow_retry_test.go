package instagram

import (
	"testing"
	"time"
)

func TestAutomaticFollowRetryUsesBoundedExponentialBackoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	policy := defaultAutomaticFollowRetryPolicy()
	for _, test := range []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 1, want: time.Second},
		{attempts: 2, want: 2 * time.Second},
		{attempts: 3, want: 4 * time.Second},
		{attempts: 10, want: 5 * time.Minute},
	} {
		got := nextAutomaticFollowRetry(policy, now, test.attempts)
		if !got.Equal(now.Add(test.want)) {
			t.Errorf(
				"attempt %d retry = %s, want %s",
				test.attempts,
				got,
				now.Add(test.want),
			)
		}
	}
}

func TestAutomaticFollowStoreOptionsEnforceHardMaxima(t *testing.T) {
	t.Parallel()

	valid := AutomaticFollowStoreOptions{
		LeaseDuration:  AutomaticFollowLeaseDuration,
		InitialBackoff: AutomaticFollowInitialBackoff,
		MaxBackoff:     AutomaticFollowMaxBackoff,
	}
	if _, err := NewAutomaticFollowStoreWithOptions(nil, valid); err != nil {
		t.Fatalf("exact maxima rejected: %v", err)
	}
	for name, options := range map[string]AutomaticFollowStoreOptions{
		"lease": {
			LeaseDuration:  AutomaticFollowLeaseDuration + time.Nanosecond,
			InitialBackoff: AutomaticFollowInitialBackoff,
			MaxBackoff:     AutomaticFollowMaxBackoff,
		},
		"initial backoff": {
			LeaseDuration:  AutomaticFollowLeaseDuration,
			InitialBackoff: AutomaticFollowInitialBackoff + time.Nanosecond,
			MaxBackoff:     AutomaticFollowMaxBackoff,
		},
		"maximum backoff": {
			LeaseDuration:  AutomaticFollowLeaseDuration,
			InitialBackoff: AutomaticFollowInitialBackoff,
			MaxBackoff:     AutomaticFollowMaxBackoff + time.Nanosecond,
		},
	} {
		if _, err := NewAutomaticFollowStoreWithOptions(nil, options); err == nil {
			t.Errorf("%s above maximum accepted", name)
		}
	}
}
