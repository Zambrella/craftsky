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
	for _, key := range []string{"REVENUECAT_ENABLED", "REVENUECAT_API_BASE_URL", "REVENUECAT_API_KEY", "REVENUECAT_PROJECT_ID", "REVENUECAT_ENVIRONMENT", "REVENUECAT_WEBHOOK_AUTHORIZATION", "REVENUECAT_WEBHOOK_SIGNING_SECRET", "REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE", "REVENUECAT_APP_IDS", "REVENUECAT_PRODUCT_MAPPINGS"} {
		t.Setenv(key, "")
	}
	disabled, err := loadRevenueCatConfig(EnvDev)
	if err != nil {
		t.Fatalf("load disabled RevenueCat config: %v", err)
	}
	if disabled.Enabled() || disabled.Configured() {
		t.Fatalf("default RevenueCat config = %v, want disabled", disabled)
	}

	t.Setenv("REVENUECAT_ENABLED", "true")
	if _, err := loadRevenueCatConfig(EnvDev); err == nil {
		t.Fatal("incomplete enabled RevenueCat configuration succeeded")
	}

	t.Setenv("REVENUECAT_API_BASE_URL", "https://api.revenuecat.example/v2")
	t.Setenv("REVENUECAT_API_KEY", revenueCatTestAPIKey)
	t.Setenv("REVENUECAT_PROJECT_ID", "project_test")
	t.Setenv("REVENUECAT_WEBHOOK_AUTHORIZATION", revenueCatTestAuthorization)
	t.Setenv("REVENUECAT_WEBHOOK_SIGNING_SECRET", revenueCatTestSigningSecret)
	t.Setenv("REVENUECAT_APP_IDS", "app_ios,app_android")
	t.Setenv("REVENUECAT_PRODUCT_MAPPINGS", `{"plus_monthly":{"appId":"app_ios","tier":"plus"},"business_monthly":{"appId":"app_android","tier":"business"}}`)

	configured, err := loadRevenueCatConfig(EnvDev)
	if err != nil {
		t.Fatalf("load complete RevenueCat config: %v", err)
	}
	if !configured.Enabled() || !configured.Configured() {
		t.Fatalf("complete RevenueCat config = %v, want enabled and configured", configured)
	}
	if configured.environment != "production" {
		t.Fatalf("default RevenueCat environment = %q", configured.environment)
	}
	if !configured.webhookRequireSignature {
		t.Fatal("RevenueCat webhook signature verification must default to required")
	}
	t.Setenv("REVENUECAT_ENVIRONMENT", "sandbox")
	sandbox, err := loadRevenueCatConfig(EnvDev)
	if err != nil || sandbox.environment != "sandbox" {
		t.Fatalf("sandbox RevenueCat config = %q, error %v", sandbox.environment, err)
	}
	t.Setenv("REVENUECAT_ENVIRONMENT", "preview")
	if _, err := loadRevenueCatConfig(EnvDev); err == nil {
		t.Fatal("invalid RevenueCat environment succeeded")
	}
	t.Setenv("REVENUECAT_ENVIRONMENT", "sandbox")
	t.Setenv("REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE", "false")
	t.Setenv("REVENUECAT_WEBHOOK_SIGNING_SECRET", "")
	localProxy, err := loadRevenueCatConfig(EnvDev)
	if err != nil || localProxy.webhookRequireSignature {
		t.Fatalf("local webhook signature bypass = %v, error %v", localProxy, err)
	}
	if _, err := loadRevenueCatConfig(EnvProd); err == nil || !strings.Contains(err.Error(), "REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE") {
		t.Fatalf("production webhook signature bypass error = %v", err)
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

func TestRevenueCatConfigRejectsSignatureBypassOutsideDevelopmentWhenDisabled(t *testing.T) {
	t.Setenv("REVENUECAT_ENABLED", "false")
	t.Setenv("REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE", "false")

	if _, err := loadRevenueCatConfig(EnvProd); err == nil || !strings.Contains(err.Error(), "REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE") {
		t.Fatalf("production webhook signature bypass error = %v", err)
	}
}
