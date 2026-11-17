package subscriptions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWebPaywallKeyModes(t *testing.T) {
	for _, env := range []string{"sandbox", "production"} {
		for _, key := range []string{"", "strp_sb_fixture", "strp_fixture", "test_fixture", "sk_live_fixture", "strp_sb_", "strp_sb_bad\n"} {
			want := env == "sandbox" && key == "strp_sb_fixture" || env == "production" && key == "strp_fixture"
			if ValidWebPaywallKey(key, env) != want {
				t.Errorf("key mode validation mismatch")
			}
		}
	}
	if ValidWebPaywallKey("strp_fixture", "invalid") {
		t.Fatal("accepted unknown environment")
	}
	private := WebPaywall{AppUserID: "private-owner-canary"}
	if strings.Contains(fmt.Sprintf("%+v %#v", private, private), private.AppUserID) {
		t.Fatal("private UUID leaked in formatting")
	}
}
func TestWebPaywallUsesOwnerAndRequiresFreshNonPendingState(t *testing.T) {
	store, pool, _, did, account, now := checkoutFixture(t)
	store.checkoutConfig.RevenueCatWebKey = "strp_fixture"
	prepared, err := store.PrepareWebPaywall(t.Context(), did, TierBusiness, now)
	if err != nil || prepared.AppUserID != account.RevenueCatAppUserID.String() || prepared.Offering != "business" {
		t.Fatal("wrong identified paywall", err)
	}
	store.checkoutConfig.RevenueCatWebKey = "strp_sb_fixture"
	if _, err := store.PrepareWebPaywall(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutUnavailable) {
		t.Fatal("accepted wrong environment")
	}
	store.checkoutConfig.RevenueCatWebKey = "strp_fixture"
	if _, err := store.PrepareWebPaywall(t.Context(), did, TierPlus, now.Add(6*time.Minute)); !errors.Is(err, ErrSubscriptionReconciliationRequired) {
		t.Fatal("accepted stale state")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET web_checkout_pending=true WHERE id=$1`, account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PrepareWebPaywall(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutPending) {
		t.Fatal("accepted pending checkout")
	}
}
