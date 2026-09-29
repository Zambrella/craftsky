package subscriptions

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestFeatureAccessFollowsAssignedDIDAndEffectiveTier(t *testing.T) {
	beneficiary := syntax.DID("did:plc:beneficiary")
	payer := syntax.DID("did:plc:payer")
	for _, tc := range []struct {
		name     string
		access   SelfAccess
		plus     bool
		business bool
	}{
		{name: "unassigned payer", access: ProjectSelfAccess(payer, nil)},
		{name: "assigned plus", access: ProjectSelfAccess(beneficiary, &LocalAssignment{Tier: TierPlus, GivesAccess: true}), plus: true},
		{name: "cancelled but accessible business", access: ProjectSelfAccess(beneficiary, &LocalAssignment{Tier: TierBusiness, GivesAccess: true}), plus: true, business: true},
		{name: "dormant business assignment", access: ProjectSelfAccess(beneficiary, &LocalAssignment{Tier: TierBusiness, GivesAccess: false})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.access.AllowsPlus(); got != tc.plus {
				t.Errorf("AllowsPlus() = %v, want %v", got, tc.plus)
			}
			if got := tc.access.AllowsBusiness(); got != tc.business {
				t.Errorf("AllowsBusiness() = %v, want %v", got, tc.business)
			}
		})
	}
}
