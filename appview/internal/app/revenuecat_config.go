package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"social.craftsky/appview/internal/subscriptions"
)

const (
	defaultRevenueCatAPIBaseURL             = "https://api.revenuecat.com/v2"
	defaultRevenueCatWebhookBodyLimit int64 = 256 * 1024
	maximumRevenueCatWebhookBodyLimit       = 1024 * 1024
	maximumRevenueCatAPITimeout             = 30 * time.Second
	maximumRevenueCatPollInterval           = time.Minute
	maximumRevenueCatScheduleInterval       = 24 * time.Hour
	maximumRevenueCatLeaseDuration          = 10 * time.Minute
)

// RevenueCatConfig is disabled by default and keeps provider credentials and
// private catalog identifiers out of ordinary process diagnostics.
type RevenueCatConfig struct {
	enabled                     bool
	configured                  bool
	apiBaseURL                  string
	apiKey                      Secret
	projectID                   string
	stripeSecretKey             Secret
	stripeCheckout              subscriptions.WebCheckoutConfig
	stripePortalConfigurationID string
	environment                 string
	webhookAuthorization        Secret
	webhookSigningSecret        Secret
	webhookRequireSignature     bool
	appIDs                      []string
	products                    map[string]subscriptions.ProductMapping
	webhookBodyLimit            int64
	ingressDeadline             time.Duration
	signatureTolerance          time.Duration
	apiTimeout                  time.Duration
	pageLimit                   int
	maxPages                    int
	maxResponseBytes            int64
	reconciliationPoll          time.Duration
	reconciliationSchedule      time.Duration
	leaseDuration               time.Duration
}

func (c RevenueCatConfig) Enabled() bool    { return c.enabled }
func (c RevenueCatConfig) Configured() bool { return c.configured }
func (c RevenueCatConfig) ReconciliationPollInterval() time.Duration {
	return c.reconciliationPoll
}
func (c RevenueCatConfig) ReconciliationScheduleInterval() time.Duration {
	return c.reconciliationSchedule
}
func (c RevenueCatConfig) LeaseDuration() time.Duration   { return c.leaseDuration }
func (c RevenueCatConfig) IngressDeadline() time.Duration { return c.ingressDeadline }

func (c RevenueCatConfig) String() string {
	return fmt.Sprintf(
		"RevenueCatConfig{enabled:%t,configured:%t,credentials:[REDACTED],catalog:[REDACTED],apiBaseURL:%q,webhookBodyLimit:%d,ingressDeadline:%s,apiTimeout:%s,reconciliationPoll:%s,reconciliationSchedule:%s,leaseDuration:%s}",
		c.enabled, c.configured, redactedRevenueCatOrigin(c.apiBaseURL), c.webhookBodyLimit,
		c.ingressDeadline, c.apiTimeout, c.reconciliationPoll, c.reconciliationSchedule, c.leaseDuration,
	)
}

func (c RevenueCatConfig) GoString() string { return c.String() }

func (c RevenueCatConfig) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, c.String())
}

func redactedRevenueCatOrigin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func loadRevenueCatConfig(env Env) (RevenueCatConfig, error) {
	webhookRequireSignature, err := boolEnv("REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE", true)
	if err != nil {
		return RevenueCatConfig{}, err
	}
	if !webhookRequireSignature && env != EnvDev {
		return RevenueCatConfig{}, errors.New("REVENUECAT_WEBHOOK_REQUIRE_SIGNATURE=false is only allowed in dev")
	}
	enabled, err := boolEnv("REVENUECAT_ENABLED", false)
	if err != nil {
		return RevenueCatConfig{}, err
	}
	if !enabled {
		return RevenueCatConfig{webhookRequireSignature: webhookRequireSignature}, nil
	}

	config := RevenueCatConfig{
		enabled:                     true,
		stripeSecretKey:             Secret(strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))),
		stripeCheckout:              subscriptions.WebCheckoutConfig{AccountID: strings.TrimSpace(os.Getenv("STRIPE_ACCOUNT_ID")), PlusPriceID: strings.TrimSpace(os.Getenv("STRIPE_PLUS_PRICE_ID")), BusinessPriceID: strings.TrimSpace(os.Getenv("STRIPE_BUSINESS_PRICE_ID")), SuccessURL: strings.TrimSpace(os.Getenv("STRIPE_CHECKOUT_SUCCESS_URL")), CancelURL: strings.TrimSpace(os.Getenv("STRIPE_CHECKOUT_CANCEL_URL")), BillingReturnURL: strings.TrimSpace(os.Getenv("STRIPE_BILLING_RETURN_URL"))},
		stripePortalConfigurationID: strings.TrimSpace(os.Getenv("STRIPE_PORTAL_CONFIGURATION_ID")),
		apiBaseURL:                  strings.TrimSpace(getEnvWithDefault("REVENUECAT_API_BASE_URL", defaultRevenueCatAPIBaseURL)),
		apiKey:                      Secret(strings.TrimSpace(os.Getenv("REVENUECAT_API_KEY"))),
		projectID:                   strings.TrimSpace(os.Getenv("REVENUECAT_PROJECT_ID")),
		environment:                 strings.TrimSpace(getEnvWithDefault("REVENUECAT_ENVIRONMENT", "production")),
		webhookAuthorization:        Secret(strings.TrimSpace(os.Getenv("REVENUECAT_WEBHOOK_AUTHORIZATION"))),
		webhookSigningSecret:        Secret(strings.TrimSpace(os.Getenv("REVENUECAT_WEBHOOK_SIGNING_SECRET"))),
		webhookRequireSignature:     webhookRequireSignature,
		appIDs:                      splitCommaEnv("REVENUECAT_APP_IDS"),
	}
	stripeValues := []string{config.stripeSecretKey.Reveal(), config.stripeCheckout.AccountID, config.stripeCheckout.PlusPriceID, config.stripeCheckout.BusinessPriceID, config.stripeCheckout.SuccessURL, config.stripeCheckout.CancelURL, config.stripeCheckout.BillingReturnURL, config.stripePortalConfigurationID}
	stripeConfigured := false
	for _, value := range stripeValues {
		if value != "" {
			stripeConfigured = true
		}
	}
	if stripeConfigured {
		if config.stripeSecretKey.Reveal() == "" {
			return RevenueCatConfig{}, errors.New("STRIPE_SECRET_KEY is required for Stripe checkout")
		}
		if err := subscriptions.ValidateWebCheckoutConfig(config.stripeCheckout); err != nil {
			return RevenueCatConfig{}, err
		}
		prefix := "sk_live_"
		if config.environment == "sandbox" {
			prefix = "sk_test_"
		}
		if !strings.HasPrefix(config.stripeSecretKey.Reveal(), prefix) {
			return RevenueCatConfig{}, errors.New("STRIPE_SECRET_KEY must match REVENUECAT_ENVIRONMENT")
		}
		if config.stripePortalConfigurationID != "" && !strings.HasPrefix(config.stripePortalConfigurationID, "bpc_") {
			return RevenueCatConfig{}, errors.New("invalid STRIPE_PORTAL_CONFIGURATION_ID")
		}
	}
	if config.environment != "production" && config.environment != "sandbox" {
		return RevenueCatConfig{}, errors.New("REVENUECAT_ENVIRONMENT must be production or sandbox")
	}
	for key, value := range map[string]string{
		"REVENUECAT_API_KEY":               config.apiKey.Reveal(),
		"REVENUECAT_PROJECT_ID":            config.projectID,
		"REVENUECAT_WEBHOOK_AUTHORIZATION": config.webhookAuthorization.Reveal(),
		"REVENUECAT_APP_IDS":               strings.Join(config.appIDs, ","),
		"REVENUECAT_PRODUCT_MAPPINGS":      strings.TrimSpace(os.Getenv("REVENUECAT_PRODUCT_MAPPINGS")),
	} {
		if value == "" {
			return RevenueCatConfig{}, fmt.Errorf("%s is required when REVENUECAT_ENABLED=true", key)
		}
	}
	if config.webhookRequireSignature && config.webhookSigningSecret.Reveal() == "" {
		return RevenueCatConfig{}, errors.New("REVENUECAT_WEBHOOK_SIGNING_SECRET is required when webhook signature verification is enabled")
	}
	baseURL, err := url.Parse(config.apiBaseURL)
	if err != nil || baseURL.Scheme != "https" || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return RevenueCatConfig{}, errors.New("REVENUECAT_API_BASE_URL must be an HTTPS URL without credentials, query, or fragment")
	}

	var rawMappings map[string]struct {
		AppID string             `json:"appId"`
		Tier  subscriptions.Tier `json:"tier"`
	}
	if err := json.Unmarshal([]byte(os.Getenv("REVENUECAT_PRODUCT_MAPPINGS")), &rawMappings); err != nil || len(rawMappings) == 0 {
		return RevenueCatConfig{}, errors.New("REVENUECAT_PRODUCT_MAPPINGS must be a non-empty JSON object")
	}
	apps := make(map[string]struct{}, len(config.appIDs))
	for _, appID := range config.appIDs {
		apps[appID] = struct{}{}
	}
	config.products = make(map[string]subscriptions.ProductMapping, len(rawMappings))
	for productID, mapping := range rawMappings {
		if strings.TrimSpace(productID) == "" || strings.TrimSpace(mapping.AppID) == "" || !mapping.Tier.Paid() {
			return RevenueCatConfig{}, errors.New("REVENUECAT_PRODUCT_MAPPINGS contains an invalid product mapping")
		}
		if _, ok := apps[mapping.AppID]; !ok {
			return RevenueCatConfig{}, errors.New("REVENUECAT_PRODUCT_MAPPINGS names an unconfigured app")
		}
		config.products[productID] = subscriptions.ProductMapping{AppID: mapping.AppID, Tier: mapping.Tier}
	}
	if config.webhookBodyLimit, err = boundedInt64Env("REVENUECAT_WEBHOOK_BODY_LIMIT_BYTES", defaultRevenueCatWebhookBodyLimit, 1, maximumRevenueCatWebhookBodyLimit); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.ingressDeadline, err = boundedPositiveDurationEnv("REVENUECAT_WEBHOOK_INGRESS_DEADLINE", 5*time.Second, 10*time.Second); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.signatureTolerance, err = boundedPositiveDurationEnv("REVENUECAT_WEBHOOK_SIGNATURE_TOLERANCE", 5*time.Minute, 10*time.Minute); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.apiTimeout, err = boundedPositiveDurationEnv("REVENUECAT_API_TIMEOUT", 10*time.Second, maximumRevenueCatAPITimeout); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.pageLimit, err = boundedIntEnv("REVENUECAT_API_PAGE_LIMIT", 100, 1, 100); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.maxPages, err = boundedIntEnv("REVENUECAT_API_MAX_PAGES", 20, 1, 100); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.maxResponseBytes, err = boundedInt64Env("REVENUECAT_API_MAX_RESPONSE_BYTES", 1024*1024, 1, 4*1024*1024); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.reconciliationPoll, err = boundedPositiveDurationEnv("REVENUECAT_RECONCILIATION_POLL_INTERVAL", time.Second, maximumRevenueCatPollInterval); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.reconciliationSchedule, err = boundedPositiveDurationEnv("REVENUECAT_RECONCILIATION_SCHEDULE_INTERVAL", 6*time.Hour, maximumRevenueCatScheduleInterval); err != nil {
		return RevenueCatConfig{}, err
	}
	if config.leaseDuration, err = boundedPositiveDurationEnv("REVENUECAT_RECONCILIATION_LEASE_DURATION", time.Minute, maximumRevenueCatLeaseDuration); err != nil {
		return RevenueCatConfig{}, err
	}
	config.configured = true
	return config, nil
}
