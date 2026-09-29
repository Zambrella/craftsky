package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/subscriptions"
)

type legacyAccountTypeReader struct{ calls int }

func (r *legacyAccountTypeReader) ReadAccountTypes(context.Context, []syntax.DID) (map[syntax.DID]business.AccountType, error) {
	r.calls++
	return map[syntax.DID]business.AccountType{"did:plc:plus": business.AccountTypeBusiness}, nil
}

func TestDerivedBusinessAccountTypeIgnoresLegacyFlag(t *testing.T) {
	legacy := &legacyAccountTypeReader{}
	h := api.NewIdentityAccountTypeHydrator(legacy, fakeCustomisationTiers{
		"did:plc:plus": subscriptions.TierPlus, "did:plc:business": subscriptions.TierBusiness,
	})
	result, err := h.HydrateJSON(context.Background(), []byte(`{"items":[{"did":"did:plc:plus","handle":"plus.example","accountType":"business"},{"did":"did:plc:business","handle":"business.example","accountType":"regular"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(result, &body); err != nil {
		t.Fatal(err)
	}
	if body.Items[0]["accountType"] != "regular" || body.Items[1]["accountType"] != "business" || legacy.calls != 0 {
		t.Fatalf("derived types = %+v; legacy calls = %d", body.Items, legacy.calls)
	}
}
