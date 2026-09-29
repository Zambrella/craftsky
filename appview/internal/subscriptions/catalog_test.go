package subscriptions

import "testing"

func TestCatalogAuthorizesOnlyConfiguredEnvironmentProducts(t *testing.T) {
	catalog, err := NewCatalog(CatalogConfig{
		ProjectID: "proj-production",
		AppIDs:    []string{"app-ios", "app-android"},
		Products: map[string]ProductMapping{
			"prod-plus":         {AppID: "app-ios", Tier: TierPlus},
			"prod-business":     {AppID: "app-android", Tier: TierBusiness},
			"prod-unconfigured": {AppID: "app-other", Tier: TierPlus},
			"prod-invalid-tier": {AppID: "app-ios", Tier: Tier("enterprise")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		projectID   string
		environment string
		productID   string
		want        ProductMapping
		allowed     bool
	}{
		{name: "Plus", projectID: "proj-production", environment: "production", productID: "prod-plus", want: ProductMapping{AppID: "app-ios", Tier: TierPlus}, allowed: true},
		{name: "Business", projectID: "proj-production", environment: "production", productID: "prod-business", want: ProductMapping{AppID: "app-android", Tier: TierBusiness}, allowed: true},
		{name: "sandbox", projectID: "proj-production", environment: "sandbox", productID: "prod-plus"},
		{name: "unknown project", projectID: "proj-other", environment: "production", productID: "prod-plus"},
		{name: "unknown product", projectID: "proj-production", environment: "production", productID: "prod-other"},
		{name: "unconfigured app", projectID: "proj-production", environment: "production", productID: "prod-unconfigured"},
		{name: "invalid tier", projectID: "proj-production", environment: "production", productID: "prod-invalid-tier"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := catalog.Authorize(test.projectID, test.environment, test.productID)
			if ok != test.allowed || got != test.want {
				t.Fatalf("Authorize() = %+v, %t; want %+v, %t", got, ok, test.want, test.allowed)
			}
		})
	}
	sandboxCatalog, err := NewCatalog(CatalogConfig{
		ProjectID:   "proj-production",
		Environment: "sandbox",
		AppIDs:      []string{"app-ios"},
		Products:    map[string]ProductMapping{"prod-plus": {AppID: "app-ios", Tier: TierPlus}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := sandboxCatalog.Authorize("proj-production", "sandbox", "prod-plus"); !ok || got.Tier != TierPlus {
		t.Fatalf("sandbox authorization = %+v, %t", got, ok)
	}
	if _, ok := sandboxCatalog.Authorize("proj-production", "production", "prod-plus"); ok {
		t.Fatal("sandbox catalog authorized production state")
	}
	if _, err := NewCatalog(CatalogConfig{ProjectID: "project", Environment: "preview"}); err == nil {
		t.Fatal("invalid environment succeeded")
	}
}
