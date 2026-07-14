package push

import (
	"testing"
	"time"
)

func TestRetryScheduleIsBoundedByDeadlineAndProviderTTL(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(6 * time.Hour)
	first, ok := NextRetry(now, deadline, 1, 0)
	if !ok || first.Sub(now) != time.Second {
		t.Fatalf("first=%v ok=%v", first, ok)
	}
	capped, ok := NextRetry(now, deadline, 20, 0)
	if !ok || capped.Sub(now) != 15*time.Minute {
		t.Fatalf("capped=%v ok=%v", capped, ok)
	}
	if _, ok := NextRetry(deadline.Add(-time.Second), deadline, 20, 0); ok {
		t.Fatal("retry scheduled beyond deadline")
	}
	if ttl, ok := ProviderTTL(now, deadline); !ok || ttl != 6*time.Hour {
		t.Fatalf("ttl=%v ok=%v", ttl, ok)
	}
	if _, ok := ProviderTTL(deadline, deadline); ok {
		t.Fatal("non-positive TTL accepted")
	}
}
