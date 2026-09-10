package app

import (
	"net/http"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
)

func TestRouteDependenciesIncludeSubscriptionStore(t *testing.T) {
	store := &subscriptions.Store{}
	routeDeps := RouteDependencies(&Deps{Subscriptions: store})
	if routeDeps.Subscriptions != store {
		t.Fatal("subscription store was dropped at the route composition boundary")
	}
}

func testRevenueCatConfig() RevenueCatConfig {
	return RevenueCatConfig{
		enabled: true, configured: true,
		apiBaseURL: "https://api.revenuecat.example/v2", apiKey: revenueCatTestAPIKey,
		projectID: "project_test", webhookAuthorization: revenueCatTestAuthorization,
		webhookSigningSecret: revenueCatTestSigningSecret, appIDs: []string{"app_ios"},
		products:         map[string]subscriptions.ProductMapping{"plus_monthly": {AppID: "app_ios", Tier: subscriptions.TierPlus}},
		webhookBodyLimit: 1024, ingressDeadline: time.Second, signatureTolerance: time.Minute,
		apiTimeout: time.Second, pageLimit: 100, maxPages: 10, maxResponseBytes: 1024,
		reconciliationPoll: time.Second, reconciliationSchedule: time.Hour, leaseDuration: time.Minute,
	}
}

func TestRevenueCatCompositionRequiresCompleteConfiguration(t *testing.T) {
	incomplete, err := newSubscriptionDependencies(nil, RevenueCatConfig{}, http.DefaultClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	if incomplete.store == nil {
		t.Fatal("local subscription store must remain available while RevenueCat is disabled")
	}
	if incomplete.webhook != nil || incomplete.reconciler != nil {
		t.Fatal("incomplete RevenueCat configuration wired provider ingress or work")
	}

	configured := testRevenueCatConfig()
	complete, err := newSubscriptionDependencies(nil, configured, http.DefaultClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	if complete.store == nil || complete.webhook == nil || complete.reconciler == nil {
		t.Fatalf("complete RevenueCat composition = %#v, want store, one webhook, and one reconciler", complete)
	}
}
