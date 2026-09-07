package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type identityBackfillCandidatesFake struct {
	dids []syntax.DID
}

func (fake identityBackfillCandidatesFake) BackfillCandidateDIDs(context.Context, int, time.Time) ([]syntax.DID, error) {
	return fake.dids, nil
}

type authoritativeIdentityRefreshFake struct {
	calls []syntax.DID
	fail  map[syntax.DID]error
}

func (fake *authoritativeIdentityRefreshFake) RefreshCurrentHandle(_ context.Context, did syntax.DID) error {
	fake.calls = append(fake.calls, did)
	return fake.fail[did]
}

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

func TestIdentityCacheBackfillUsesAuthoritativeIdentityRefreshService(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	valid := syntax.DID("did:plc:valid-handle")
	invalid := syntax.DID("did:plc:invalid-handle")
	transient := syntax.DID("did:plc:transient")
	candidates := identityBackfillCandidatesFake{dids: []syntax.DID{valid, invalid, transient}}
	refresher := &authoritativeIdentityRefreshFake{fail: map[syntax.DID]error{
		transient: errors.New("temporary identity resolution failure"),
	}}

	stats, err := runIdentityCacheBackfill(context.Background(), candidates, refresher, 10, now)
	if err != nil {
		t.Fatalf("run identity cache backfill: %v", err)
	}
	if stats.Candidates != 3 || stats.Upserted != 2 || stats.Failed != 1 {
		t.Fatalf("stats=%+v, want candidates=3 refreshed=2 failed=1", stats)
	}
	want := []syntax.DID{valid, invalid, transient}
	if len(refresher.calls) != len(want) {
		t.Fatalf("authoritative refresh calls=%v, want %v", refresher.calls, want)
	}
	for i := range want {
		if refresher.calls[i] != want[i] {
			t.Fatalf("authoritative refresh calls=%v, want %v", refresher.calls, want)
		}
	}
}
