package main

import (
	"context"
	"testing"
)

func TestIdentityCacheBackfillCommandUsesDefaultAndExplicitLimits(t *testing.T) {
	t.Parallel()
	var gotLimits []int
	cmd := newIdentityCacheCmd(func(_ context.Context, limit int) (identityCacheBackfillStats, error) {
		gotLimits = append(gotLimits, limit)
		return identityCacheBackfillStats{}, nil
	})
	cmd.SetArgs([]string{"backfill"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("default backfill execute: %v", err)
	}

	cmd = newIdentityCacheCmd(func(_ context.Context, limit int) (identityCacheBackfillStats, error) {
		gotLimits = append(gotLimits, limit)
		return identityCacheBackfillStats{}, nil
	})
	cmd.SetArgs([]string{"backfill", "--limit", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("explicit backfill execute: %v", err)
	}

	want := []int{100, 10}
	if len(gotLimits) != len(want) {
		t.Fatalf("limits = %v, want %v", gotLimits, want)
	}
	for i := range want {
		if gotLimits[i] != want[i] {
			t.Fatalf("limits = %v, want %v", gotLimits, want)
		}
	}
}
