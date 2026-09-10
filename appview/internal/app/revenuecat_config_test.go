package app

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	revenueCatTestAPIKey        = "rc-api-key-private-canary"
	revenueCatTestAuthorization = "rc-webhook-authorization-private-canary"
	revenueCatTestSigningSecret = "rc-webhook-signing-private-canary"
)

func TestRevenueCatConfigIsDisabledByDefaultAndRequiresCompleteConfiguration(t *testing.T) {
	for _, key := range []string{"REVENUECAT_ENABLED", "REVENUECAT_API_BASE_URL", "REVENUECAT_API_KEY", "REVENUECAT_PROJECT_ID", "REVENUECAT_WEBHOOK_AUTHORIZATION", "REVENUECAT_WEBHOOK_SIGNING_SECRET", "REVENUECAT_APP_IDS", "REVENUECAT_PRODUCT_MAPPINGS"} {
		t.Setenv(key, "")
	}
	disabled, err := loadRevenueCatConfig()
	if err != nil {
		t.Fatalf("load disabled RevenueCat config: %v", err)
	}
	if disabled.Enabled() || disabled.Configured() {
		t.Fatalf("default RevenueCat config = %v, want disabled", disabled)
	}

	t.Setenv("REVENUECAT_ENABLED", "true")
	if _, err := loadRevenueCatConfig(); err == nil {
		t.Fatal("incomplete enabled RevenueCat configuration succeeded")
	}

	t.Setenv("REVENUECAT_API_BASE_URL", "https://api.revenuecat.example/v2")
	t.Setenv("REVENUECAT_API_KEY", revenueCatTestAPIKey)
	t.Setenv("REVENUECAT_PROJECT_ID", "project_test")
	t.Setenv("REVENUECAT_WEBHOOK_AUTHORIZATION", revenueCatTestAuthorization)
	t.Setenv("REVENUECAT_WEBHOOK_SIGNING_SECRET", revenueCatTestSigningSecret)
	t.Setenv("REVENUECAT_APP_IDS", "app_ios,app_android")
	t.Setenv("REVENUECAT_PRODUCT_MAPPINGS", `{"plus_monthly":{"appId":"app_ios","tier":"plus"},"business_monthly":{"appId":"app_android","tier":"business"}}`)

	configured, err := loadRevenueCatConfig()
	if err != nil {
		t.Fatalf("load complete RevenueCat config: %v", err)
	}
	if !configured.Enabled() || !configured.Configured() {
		t.Fatalf("complete RevenueCat config = %v, want enabled and configured", configured)
	}
	if configured.ReconciliationPollInterval() <= 0 || configured.ReconciliationScheduleInterval() <= 0 || configured.LeaseDuration() <= 0 || configured.IngressDeadline() > 10*time.Second {
		t.Fatalf("unsafe RevenueCat worker/ingress budgets: %v", configured)
	}
	diagnostic := fmt.Sprintf("%v %+v %#v", configured, configured, configured)
	for _, canary := range []string{revenueCatTestAPIKey, revenueCatTestAuthorization, revenueCatTestSigningSecret} {
		if strings.Contains(diagnostic, canary) {
			t.Fatalf("formatted RevenueCat config leaked %q: %s", canary, diagnostic)
		}
	}
}
