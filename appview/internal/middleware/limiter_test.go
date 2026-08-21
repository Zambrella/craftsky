package middleware

import (
	"testing"
	"time"
)

func TestBoundedRateLimiterUsesSharedOverflowAtCapacity(t *testing.T) {
	t.Parallel()

	now := time.Unix(100, 0)
	limiter, err := NewBoundedLocalRateLimiter(
		RateLimitConfig{Classes: map[RateClass]ClassLimit{
			RateClassOuter: {Window: time.Minute, PerClient: 1},
		}},
		LocalLimiterOptions{Capacity: 2, IdleTTL: 2 * time.Minute, Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"client-a", "client-b", "client-c"} {
		if decision := limiter.Allow(RateClassOuter, RateKeys{ClientKey: key}); !decision.Allowed {
			t.Fatalf("first request for %s rejected: %+v", key, decision)
		}
	}
	if got := limiter.EntryCount(); got != 2 {
		t.Fatalf("EntryCount() = %d, want hard capacity 2", got)
	}
	decision := limiter.Allow(RateClassOuter, RateKeys{ClientKey: "client-d"})
	if decision.Allowed || decision.KeyType != "client_overflow" {
		t.Fatalf("rotated key decision = %+v, want shared overflow rejection", decision)
	}
	if got := limiter.EntryCount(); got != 2 {
		t.Fatalf("EntryCount() after overflow = %d, want hard capacity 2", got)
	}

	now = now.Add(3 * time.Minute)
	if decision := limiter.Allow(RateClassOuter, RateKeys{ClientKey: "client-e"}); !decision.Allowed {
		t.Fatalf("request after idle expiry rejected: %+v", decision)
	}
	if got := limiter.EntryCount(); got > 2 {
		t.Fatalf("EntryCount() after expiry = %d, exceeds capacity 2", got)
	}
}
