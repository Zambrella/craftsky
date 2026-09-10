package subscriptions

import (
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestSelfAccessProjection(t *testing.T) {
	did := syntax.DID("did:plc:beneficiary")
	periodEnded := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	staleSince := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		assignment *LocalAssignment
		want       SelfAccess
	}{
		{
			name: "free without an assignment",
			want: SelfAccess{DID: did, EffectiveTier: TierFree},
		},
		{
			name: "accessible plus assignment",
			assignment: &LocalAssignment{
				Tier: TierPlus, GivesAccess: true, AccessEndsAt: &periodEnded,
			},
			want: SelfAccess{
				DID: did, EffectiveTier: TierPlus, GivesAccess: true,
				AccessEndsAt: &periodEnded, AssignedTier: tierPointer(TierPlus),
			},
		},
		{
			name: "accessible business assignment",
			assignment: &LocalAssignment{
				Tier: TierBusiness, GivesAccess: true,
			},
			want: SelfAccess{
				DID: did, EffectiveTier: TierBusiness, GivesAccess: true,
				AssignedTier: tierPointer(TierBusiness),
			},
		},
		{
			name: "dormant assignment remains visible",
			assignment: &LocalAssignment{
				Tier: TierPlus, GivesAccess: false, AccessEndsAt: &periodEnded,
			},
			want: SelfAccess{
				DID: did, EffectiveTier: TierFree, AccessEndsAt: &periodEnded,
				AssignedTier: tierPointer(TierPlus),
			},
		},
		{
			name: "stale reconciliation and passed period do not revoke provider access",
			assignment: &LocalAssignment{
				Tier: TierBusiness, GivesAccess: true, AccessEndsAt: &periodEnded,
				ReconciliationStaleSince: &staleSince,
			},
			want: SelfAccess{
				DID: did, EffectiveTier: TierBusiness, GivesAccess: true,
				AccessEndsAt: &periodEnded, AssignedTier: tierPointer(TierBusiness),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ProjectSelfAccess(did, test.assignment)
			if got.DID != test.want.DID ||
				got.EffectiveTier != test.want.EffectiveTier ||
				got.GivesAccess != test.want.GivesAccess ||
				!sameTime(got.AccessEndsAt, test.want.AccessEndsAt) ||
				!sameTier(got.AssignedTier, test.want.AssignedTier) {
				t.Fatalf("ProjectSelfAccess() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func tierPointer(tier Tier) *Tier { return &tier }

func sameTier(left, right *Tier) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func sameTime(left, right *time.Time) bool {
	return left == nil && right == nil || left != nil && right != nil && left.Equal(*right)
}
