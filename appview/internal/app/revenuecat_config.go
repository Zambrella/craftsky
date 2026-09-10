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
	enabled                bool
	configured             bool
	apiBaseURL             string
	apiKey                 Secret
	projectID              string
	webhookAuthorization   Secret
	webhookSigningSecret   Secret
	appIDs                 []string
	products               map[string]subscriptions.ProductMapping
	webhookBodyLimit       int64
	ingressDeadline        time.Duration
	signatureTolerance     time.Duration
	apiTimeout             time.Duration
	pageLimit              int
	maxPages               int
	maxResponseBytes       int64
	reconciliationPoll     time.Duration
	reconciliationSchedule time.Duration
	leaseDuration          time.Duration
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

func loadRevenueCatConfig() (RevenueCatConfig, error) {
	enabled, err := boolEnv("REVENUECAT_ENABLED", false)
	if err != nil {
		return RevenueCatConfig{}, err
	}
	if !enabled {
		return RevenueCatConfig{}, nil
	}

	config := RevenueCatConfig{
		enabled:              true,
		apiBaseURL:           strings.TrimSpace(getEnvWithDefault("REVENUECAT_API_BASE_URL", defaultRevenueCatAPIBaseURL)),
		apiKey:               Secret(strings.TrimSpace(os.Getenv("REVENUECAT_API_KEY"))),
		projectID:            strings.TrimSpace(os.Getenv("REVENUECAT_PROJECT_ID")),
		webhookAuthorization: Secret(strings.TrimSpace(os.Getenv("REVENUECAT_WEBHOOK_AUTHORIZATION"))),
		webhookSigningSecret: Secret(strings.TrimSpace(os.Getenv("REVENUECAT_WEBHOOK_SIGNING_SECRET"))),
		appIDs:               splitCommaEnv("REVENUECAT_APP_IDS"),
	}
	for key, value := range map[string]string{
		"REVENUECAT_API_KEY":                config.apiKey.Reveal(),
		"REVENUECAT_PROJECT_ID":             config.projectID,
		"REVENUECAT_WEBHOOK_AUTHORIZATION":  config.webhookAuthorization.Reveal(),
		"REVENUECAT_WEBHOOK_SIGNING_SECRET": config.webhookSigningSecret.Reveal(),
		"REVENUECAT_APP_IDS":                strings.Join(config.appIDs, ","),
		"REVENUECAT_PRODUCT_MAPPINGS":       strings.TrimSpace(os.Getenv("REVENUECAT_PRODUCT_MAPPINGS")),
	} {
		if value == "" {
			return RevenueCatConfig{}, fmt.Errorf("%s is required when REVENUECAT_ENABLED=true", key)
		}
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
