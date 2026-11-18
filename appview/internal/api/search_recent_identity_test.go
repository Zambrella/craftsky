package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/subscriptions"
)

func TestRecentSearchProfilesHydrateCurrentIdentityDecorations(t *testing.T) {
	const did = syntax.DID("did:plc:business")
	// Stored history predates the customisation and Business subscription.
	raw := []byte(`{"items":[{"id":"recent","type":"profile","payload":{"did":"did:plc:business","handle":"business.test"}}]}`)
	tiers := fakeCustomisationTiers{did: subscriptions.TierBusiness}
	reader := &fakeProfileCustomisationBatchReader{values: map[syntax.DID]api.ProfileCustomisation{
		did: {Colour: "orchid", Background: "animated-flow"},
	}}
	colours := api.NewIdentityCustomisationHydrator(reader, tiers)
	types := api.NewIdentityAccountTypeHydrator(nil, tiers)
	for _, tier := range []subscriptions.Tier{subscriptions.TierBusiness, subscriptions.TierFree} {
		tiers[did] = tier
		decorated, err := colours.HydrateJSON(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		decorated, err = types.HydrateJSON(context.Background(), decorated)
		if err != nil {
			t.Fatal(err)
		}
		var response struct {
			Items []struct {
				Payload struct {
					AccountType   string                   `json:"accountType"`
					Customisation api.ProfileCustomisation `json:"customisation"`
				} `json:"payload"`
			} `json:"items"`
		}
		if err := json.Unmarshal(decorated, &response); err != nil {
			t.Fatal(err)
		}
		profile := response.Items[0].Payload
		wantType, wantColour := "business", "orchid"
		if tier == subscriptions.TierFree {
			wantType, wantColour = "regular", "cobalt"
		}
		if profile.AccountType != wantType || profile.Customisation.Colour != wantColour {
			t.Fatalf("current tier %s: profile = %+v", tier, profile)
		}
	}
}
