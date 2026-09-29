package subscriptions

import "testing"

func TestTierValidationIsClosed(t *testing.T) {
	tests := []struct {
		name  string
		tier  Tier
		valid bool
		paid  bool
	}{
		{name: "free", tier: TierFree, valid: true},
		{name: "plus", tier: TierPlus, valid: true, paid: true},
		{name: "business", tier: TierBusiness, valid: true, paid: true},
		{name: "empty", tier: ""},
		{name: "unknown", tier: "premium"},
		{name: "wrong case", tier: "Plus"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.tier.Valid(); got != test.valid {
				t.Errorf("Tier(%q).Valid() = %t, want %t", test.tier, got, test.valid)
			}
			if got := test.tier.Paid(); got != test.paid {
				t.Errorf("Tier(%q).Paid() = %t, want %t", test.tier, got, test.paid)
			}
		})
	}
}
