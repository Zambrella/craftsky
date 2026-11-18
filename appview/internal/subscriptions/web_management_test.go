package subscriptions

import "testing"

func TestSDKManagementRequiresCompleteUniqueSameEnvironmentStripeSubscription(t *testing.T) {
	valid := ProviderSubscriptionSnapshot{Store: "stripe", Environment: "sandbox", GivesAccess: true, StoreSubscriptionIdentifier: "sub_owner"}
	for _, test := range []struct {
		name     string
		snapshot CompleteSnapshot
		want     string
	}{
		{"owned item", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{Store: "stripe", Environment: "sandbox", GivesAccess: true, StoreSubscriptionIdentifier: "si_owner"}}}, "si_owner"},
		{"owned", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{valid}}, "sub_owner"},
		{"partial", CompleteSnapshot{Subscriptions: []ProviderSubscriptionSnapshot{valid}}, ""},
		{"ambiguous", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{valid, valid}}, ""},
		{"empty", CompleteSnapshot{Complete: true}, ""},
		{"wrong environment", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{Store: "stripe", Environment: "production", GivesAccess: true, StoreSubscriptionIdentifier: "sub_other"}}}, ""},
		{"wrong store", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{Store: "app_store", Environment: "sandbox", GivesAccess: true, StoreSubscriptionIdentifier: "sub_other"}}}, ""},
		{"missing identity", CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{{Store: "stripe", Environment: "sandbox", GivesAccess: true}}}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := stripeManagementSubscription(test.snapshot, "sandbox")
			if got != test.want || (err != nil) != (test.want == "") {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}
}

func TestManagementURLSupportsStripeSecretSessionOnly(t *testing.T) {
	for _, test := range []struct {
		url   string
		valid bool
	}{
		{"https://billing.stripe.com/p/session?secret=test_opaque.token", true},
		{"https://billing.stripe.com/p/session/test_opaque#capability", true},
		{"https://billing.stripe.com/p/session?secret=", false},
		{"https://billing.stripe.com/p/session?secret=one&secret=two", false},
		{"https://billing.stripe.com/p/session?secret=one&redirect=https://evil.example", false},
		{"https://billing.stripe.com.evil.example/p/session?secret=one", false},
		{"https://billing.stripe.com/p/session?secret=one#fragment", false},
		{"https://billing.stripe.com/p/session?secret=one%0A", false},
	} {
		if ValidStripeManagementURL(test.url) != test.valid {
			t.Errorf("unexpected URL classification for fixture %q", test.url)
		}
	}
}
