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

func TestStripeCheckoutConfigurationIsAtomicEnvironmentBoundAndRedacted(t *testing.T) {
	t.Setenv("REVENUECAT_ENABLED", "true")
	t.Setenv("REVENUECAT_API_KEY", revenueCatTestAPIKey)
	t.Setenv("REVENUECAT_PROJECT_ID", "project_test")
	t.Setenv("REVENUECAT_ENVIRONMENT", "sandbox")
	t.Setenv("REVENUECAT_WEBHOOK_AUTHORIZATION", revenueCatTestAuthorization)
	t.Setenv("REVENUECAT_WEBHOOK_SIGNING_SECRET", revenueCatTestSigningSecret)
	t.Setenv("REVENUECAT_APP_IDS", "app_stripe")
	t.Setenv("REVENUECAT_PRODUCT_MAPPINGS", `{"rc_plus":{"appId":"app_stripe","tier":"plus"},"rc_business":{"appId":"app_stripe","tier":"business"}}`)
	for _, key := range []string{"STRIPE_SECRET_KEY", "STRIPE_ACCOUNT_ID", "STRIPE_PLUS_PRICE_ID", "STRIPE_BUSINESS_PRICE_ID", "STRIPE_CHECKOUT_SUCCESS_URL", "STRIPE_CHECKOUT_CANCEL_URL", "STRIPE_BILLING_RETURN_URL", "STRIPE_PORTAL_CONFIGURATION_ID"} {
		t.Setenv(key, "")
	}
	initial, err := loadRevenueCatConfig(EnvDev)
	if err != nil || initial.stripeSecretKey.Reveal() != "" {
		t.Fatal("Stripe was not optional")
	}
	values := map[string]string{"STRIPE_SECRET_KEY": "sk_test_stripe_secret_canary", "STRIPE_ACCOUNT_ID": "acct_merchant", "STRIPE_PLUS_PRICE_ID": "price_plus", "STRIPE_BUSINESS_PRICE_ID": "price_business", "STRIPE_CHECKOUT_SUCCESS_URL": "https://app.example/settings/subscriptions", "STRIPE_CHECKOUT_CANCEL_URL": "https://app.example/settings/subscriptions", "STRIPE_BILLING_RETURN_URL": "https://app.example/settings/subscriptions"}
	t.Setenv("STRIPE_SECRET_KEY", values["STRIPE_SECRET_KEY"])
	if _, err := loadRevenueCatConfig(EnvDev); err == nil {
		t.Fatal("partial Stripe configuration allowed")
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	config, err := loadRevenueCatConfig(EnvDev)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", config, config, config), values["STRIPE_SECRET_KEY"]) {
		t.Fatal("Stripe secret leaked")
	}
	t.Setenv("REVENUECAT_ENVIRONMENT", "production")
	if _, err := loadRevenueCatConfig(EnvDev); err == nil {
		t.Fatal("sandbox key allowed in production")
	}
	t.Setenv("REVENUECAT_ENVIRONMENT", "sandbox")
	t.Setenv("STRIPE_CHECKOUT_SUCCESS_URL", "https://user:password@app.example/return")
	if _, err := loadRevenueCatConfig(EnvDev); err == nil {
		t.Fatal("credential-bearing return accepted")
	}
}
