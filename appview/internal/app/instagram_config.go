package app

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const instagramSecretMinBytes = 32

var (
	instagramAccountIDPattern  = regexp.MustCompile(`^[0-9]+$`)
	instagramAPIVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)
)

// InstagramDataConfig controls the AppView-private Instagram data plane.
// Keeping this separate from InstagramMetaConfig lets imports, privacy
// controls, and existing suggestions remain available during a Meta outage or
// before the external integration is enabled.
type InstagramDataConfig struct {
	hmacKey []byte
}

func (c InstagramDataConfig) Available() bool {
	return len(c.hmacKey) != 0
}

// HMACKey returns a defensive copy of the stable private-data key.
func (c InstagramDataConfig) HMACKey() []byte {
	return append([]byte(nil), c.hmacKey...)
}

func (c InstagramDataConfig) String() string {
	return fmt.Sprintf("InstagramDataConfig{available:%t,hmacKey:[REDACTED]}", c.Available())
}

func (c InstagramDataConfig) GoString() string {
	return c.String()
}

func (c InstagramDataConfig) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, c.String())
}

// InstagramMetaConfig is the complete server-held Meta bundle. Sensitive and
// identity-like values stay private so ordinary struct formatting cannot leak
// them; narrow accessors exist only for dependency construction.
type InstagramMetaConfig struct {
	enabled            bool
	configured         bool
	appSecret          string
	verifyToken        string
	accessToken        string
	instagramAccountID string
	apiVersion         string
	apiBaseURL         *url.URL
	dmURL              *url.URL
	repliesEnabled     bool
}

func (c InstagramMetaConfig) Enabled() bool {
	return c.enabled
}

func (c InstagramMetaConfig) Configured() bool {
	return c.configured
}

func (c InstagramMetaConfig) AppSecret() string {
	return c.appSecret
}

func (c InstagramMetaConfig) VerifyToken() string {
	return c.verifyToken
}

func (c InstagramMetaConfig) AccessToken() string {
	return c.accessToken
}

func (c InstagramMetaConfig) InstagramAccountID() string {
	return c.instagramAccountID
}

func (c InstagramMetaConfig) APIVersion() string {
	return c.apiVersion
}

func (c InstagramMetaConfig) APIBaseURL() *url.URL {
	if c.apiBaseURL == nil {
		return nil
	}
	copy := *c.apiBaseURL
	return &copy
}

func (c InstagramMetaConfig) DMURL() *url.URL {
	if c.dmURL == nil {
		return nil
	}
	copy := *c.dmURL
	return &copy
}

func (c InstagramMetaConfig) RepliesEnabled() bool {
	return c.repliesEnabled
}

// GraphAPIURL constructs a versioned URL from validated, individual path
// segments. Callers cannot smuggle a query, fragment, or additional path into
// a resource identifier.
func (c InstagramMetaConfig) GraphAPIURL(segments ...string) (string, error) {
	if !c.configured || c.apiBaseURL == nil {
		return "", errors.New("Instagram Meta configuration is unavailable")
	}
	path := make([]string, 0, len(segments)+1)
	path = append(path, c.apiVersion)
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "/?#") {
			return "", errors.New("invalid Instagram Graph API path segment")
		}
		path = append(path, segment)
	}
	return c.apiBaseURL.JoinPath(path...).String(), nil
}

func (c InstagramMetaConfig) String() string {
	return fmt.Sprintf(
		"InstagramMetaConfig{enabled:%t,configured:%t,credentials:[REDACTED],account:[REDACTED],apiVersion:%q,apiBaseURL:%q,dmURL:[REDACTED],repliesEnabled:%t}",
		c.enabled,
		c.configured,
		c.apiVersion,
		redactedInstagramAPIBase(c.apiBaseURL),
		c.repliesEnabled,
	)
}

func (c InstagramMetaConfig) GoString() string {
	return c.String()
}

func (c InstagramMetaConfig) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, c.String())
}

// InstagramLimits contains the exact security/privacy defaults and hard
// maxima. LoadConfig permits only positive values at or below these defaults.
type InstagramLimits struct {
	ChallengeTTL                time.Duration
	WebhookBodyLimitBytes       int64
	WebhookMaxEvents            int
	WebhookGlobalPerMinute      int
	WebhookIPPerMinute          int
	ChallengeDIDPer15Minutes    int
	ChallengeDevicePer15Minutes int
	ChallengeIPPer15Minutes     int
	InvalidIGSIDPer15Minutes    int
	InvalidIPPer15Minutes       int
	ConfirmationDIDPerHour      int
	ConfirmationDevicePerHour   int
	ImportsDIDPerHour           int
	ImportsDevicePerHour        int
	ImportMaxEntries            int
	PageDefault                 int
	PageMax                     int
	MetaHTTPTimeout             time.Duration
	MetaResponseLimitBytes      int64
	MetaLookupConcurrency       int
	MetaLookupsPerIGSIDPerHour  int
	WorkerConcurrency           int
	WorkerLeaseDuration         time.Duration
	WorkerMaxAttempts           int
	WorkerBackoffInitial        time.Duration
	WorkerBackoffMax            time.Duration
	WorkerMaxProcessingAge      time.Duration
	DMReplyWindow               time.Duration
	NotificationWindow          time.Duration
	NotificationCountCap        int
	OperatorBatchMax            int
}

// InstagramDeploymentConfig pins the trust/scaling assumptions that shared
// rate limiting relies on. Forwarded headers remain ignored unless a later
// request boundary sees a peer in TrustedProxyCIDRs.
type InstagramDeploymentConfig struct {
	replicaCount      int
	sharedRateLimits  bool
	trustedProxyCIDRs []netip.Prefix
}

func (c InstagramDeploymentConfig) ReplicaCount() int {
	return c.replicaCount
}

func (c InstagramDeploymentConfig) SharedRateLimits() bool {
	return c.sharedRateLimits
}

func (c InstagramDeploymentConfig) TrustedProxyCIDRs() []netip.Prefix {
	return append([]netip.Prefix(nil), c.trustedProxyCIDRs...)
}

func defaultInstagramLimits() InstagramLimits {
	return InstagramLimits{
		ChallengeTTL:                10 * time.Minute,
		WebhookBodyLimitBytes:       256 * 1024,
		WebhookMaxEvents:            100,
		WebhookGlobalPerMinute:      1000,
		WebhookIPPerMinute:          300,
		ChallengeDIDPer15Minutes:    5,
		ChallengeDevicePer15Minutes: 10,
		ChallengeIPPer15Minutes:     30,
		InvalidIGSIDPer15Minutes:    10,
		InvalidIPPer15Minutes:       30,
		ConfirmationDIDPerHour:      20,
		ConfirmationDevicePerHour:   30,
		ImportsDIDPerHour:           10,
		ImportsDevicePerHour:        20,
		ImportMaxEntries:            10000,
		PageDefault:                 20,
		PageMax:                     50,
		MetaHTTPTimeout:             5 * time.Second,
		MetaResponseLimitBytes:      64 * 1024,
		MetaLookupConcurrency:       20,
		MetaLookupsPerIGSIDPerHour:  5,
		WorkerConcurrency:           4,
		WorkerLeaseDuration:         60 * time.Second,
		WorkerMaxAttempts:           5,
		WorkerBackoffInitial:        time.Second,
		WorkerBackoffMax:            5 * time.Minute,
		WorkerMaxProcessingAge:      15 * time.Minute,
		DMReplyWindow:               24 * time.Hour,
		NotificationWindow:          5 * time.Minute,
		NotificationCountCap:        99,
		OperatorBatchMax:            500,
	}
}

func loadInstagramConfig(env Env) (InstagramDataConfig, InstagramMetaConfig, InstagramLimits, InstagramDeploymentConfig, error) {
	data, err := loadInstagramDataConfig(env)
	if err != nil {
		return InstagramDataConfig{}, InstagramMetaConfig{}, InstagramLimits{}, InstagramDeploymentConfig{}, err
	}
	meta, err := loadInstagramMetaConfig(env, data.Available())
	if err != nil {
		return InstagramDataConfig{}, InstagramMetaConfig{}, InstagramLimits{}, InstagramDeploymentConfig{}, err
	}
	limits, err := loadInstagramLimits()
	if err != nil {
		return InstagramDataConfig{}, InstagramMetaConfig{}, InstagramLimits{}, InstagramDeploymentConfig{}, err
	}
	deployment, err := loadInstagramDeploymentConfig(env, data.Available())
	if err != nil {
		return InstagramDataConfig{}, InstagramMetaConfig{}, InstagramLimits{}, InstagramDeploymentConfig{}, err
	}
	return data, meta, limits, deployment, nil
}

func loadInstagramDataConfig(env Env) (InstagramDataConfig, error) {
	raw := strings.TrimSpace(os.Getenv("INSTAGRAM_DATA_HMAC_KEY"))
	if raw == "" {
		return InstagramDataConfig{}, nil
	}
	if len(raw) < instagramSecretMinBytes {
		return InstagramDataConfig{}, fmt.Errorf("INSTAGRAM_DATA_HMAC_KEY must contain at least %d bytes", instagramSecretMinBytes)
	}
	if env == EnvProd && unsafeInstagramPlaceholder(raw) {
		return InstagramDataConfig{}, errors.New("INSTAGRAM_DATA_HMAC_KEY contains an unsafe placeholder value")
	}
	return InstagramDataConfig{hmacKey: []byte(raw)}, nil
}

func loadInstagramMetaConfig(env Env, dataAvailable bool) (InstagramMetaConfig, error) {
	enabled, err := boolEnv("INSTAGRAM_META_ENABLED", false)
	if err != nil {
		return InstagramMetaConfig{}, err
	}
	repliesEnabled, err := boolEnv("INSTAGRAM_META_REPLIES_ENABLED", false)
	if err != nil {
		return InstagramMetaConfig{}, err
	}
	_, repliesSet := os.LookupEnv("INSTAGRAM_META_REPLIES_ENABLED")

	values := []struct {
		key   string
		value string
	}{
		{"INSTAGRAM_META_APP_SECRET", strings.TrimSpace(os.Getenv("INSTAGRAM_META_APP_SECRET"))},
		{"INSTAGRAM_META_VERIFY_TOKEN", strings.TrimSpace(os.Getenv("INSTAGRAM_META_VERIFY_TOKEN"))},
		{"INSTAGRAM_META_ACCESS_TOKEN", strings.TrimSpace(os.Getenv("INSTAGRAM_META_ACCESS_TOKEN"))},
		{"INSTAGRAM_META_ACCOUNT_ID", strings.TrimSpace(os.Getenv("INSTAGRAM_META_ACCOUNT_ID"))},
		{"INSTAGRAM_META_API_VERSION", strings.TrimSpace(os.Getenv("INSTAGRAM_META_API_VERSION"))},
		{"INSTAGRAM_META_API_BASE_URL", strings.TrimSpace(os.Getenv("INSTAGRAM_META_API_BASE_URL"))},
		{"INSTAGRAM_META_DM_URL", strings.TrimSpace(os.Getenv("INSTAGRAM_META_DM_URL"))},
	}
	anyBundleValue := repliesSet
	complete := repliesSet
	for _, value := range values {
		anyBundleValue = anyBundleValue || value.value != ""
		complete = complete && value.value != ""
	}

	if enabled && !dataAvailable {
		return InstagramMetaConfig{}, errors.New("INSTAGRAM_DATA_HMAC_KEY is required when INSTAGRAM_META_ENABLED=true")
	}
	if enabled && !complete {
		for _, value := range values {
			if value.value == "" {
				return InstagramMetaConfig{}, fmt.Errorf("%s is required when INSTAGRAM_META_ENABLED=true", value.key)
			}
		}
		if !repliesSet {
			return InstagramMetaConfig{}, errors.New("INSTAGRAM_META_REPLIES_ENABLED must be explicitly set when INSTAGRAM_META_ENABLED=true")
		}
	}
	if env == EnvProd && anyBundleValue && !complete {
		return InstagramMetaConfig{}, errors.New("partial Instagram Meta configuration is not allowed in prod")
	}
	if env == EnvProd && complete && !dataAvailable {
		return InstagramMetaConfig{}, errors.New("INSTAGRAM_DATA_HMAC_KEY is required when Instagram Meta configuration is present")
	}
	if !complete {
		return InstagramMetaConfig{}, nil
	}

	for _, secret := range values[:3] {
		if len(secret.value) < instagramSecretMinBytes {
			return InstagramMetaConfig{}, fmt.Errorf("%s must contain at least %d bytes", secret.key, instagramSecretMinBytes)
		}
		if env == EnvProd && unsafeInstagramPlaceholder(secret.value) {
			return InstagramMetaConfig{}, fmt.Errorf("%s contains an unsafe placeholder value", secret.key)
		}
	}
	accountID := values[3].value
	if !instagramAccountIDPattern.MatchString(accountID) {
		return InstagramMetaConfig{}, errors.New("INSTAGRAM_META_ACCOUNT_ID must contain decimal digits only")
	}
	apiVersion := values[4].value
	if !instagramAPIVersionPattern.MatchString(apiVersion) {
		return InstagramMetaConfig{}, errors.New("INSTAGRAM_META_API_VERSION must use the form vN.N")
	}
	apiBaseURL, err := parseInstagramHTTPSURL("INSTAGRAM_META_API_BASE_URL", values[5].value, true)
	if err != nil {
		return InstagramMetaConfig{}, err
	}
	dmURL, err := parseInstagramHTTPSURL("INSTAGRAM_META_DM_URL", values[6].value, false)
	if err != nil {
		return InstagramMetaConfig{}, err
	}

	return InstagramMetaConfig{
		enabled:            enabled,
		configured:         true,
		appSecret:          values[0].value,
		verifyToken:        values[1].value,
		accessToken:        values[2].value,
		instagramAccountID: accountID,
		apiVersion:         apiVersion,
		apiBaseURL:         apiBaseURL,
		dmURL:              dmURL,
		repliesEnabled:     repliesEnabled,
	}, nil
}

func loadInstagramLimits() (InstagramLimits, error) {
	limits := defaultInstagramLimits()
	var err error
	if limits.ChallengeTTL, err = boundedDurationEnv("INSTAGRAM_CHALLENGE_TTL", limits.ChallengeTTL, limits.ChallengeTTL); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WebhookBodyLimitBytes, err = boundedInt64Env("INSTAGRAM_WEBHOOK_BODY_LIMIT_BYTES", limits.WebhookBodyLimitBytes, 1, limits.WebhookBodyLimitBytes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WebhookMaxEvents, err = boundedIntEnv("INSTAGRAM_WEBHOOK_MAX_EVENTS", limits.WebhookMaxEvents, 1, limits.WebhookMaxEvents); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WebhookGlobalPerMinute, err = boundedIntEnv("INSTAGRAM_WEBHOOK_GLOBAL_PER_MINUTE", limits.WebhookGlobalPerMinute, 1, limits.WebhookGlobalPerMinute); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WebhookIPPerMinute, err = boundedIntEnv("INSTAGRAM_WEBHOOK_IP_PER_MINUTE", limits.WebhookIPPerMinute, 1, limits.WebhookIPPerMinute); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ChallengeDIDPer15Minutes, err = boundedIntEnv("INSTAGRAM_CHALLENGE_DID_PER_15_MINUTES", limits.ChallengeDIDPer15Minutes, 1, limits.ChallengeDIDPer15Minutes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ChallengeDevicePer15Minutes, err = boundedIntEnv("INSTAGRAM_CHALLENGE_DEVICE_PER_15_MINUTES", limits.ChallengeDevicePer15Minutes, 1, limits.ChallengeDevicePer15Minutes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ChallengeIPPer15Minutes, err = boundedIntEnv("INSTAGRAM_CHALLENGE_IP_PER_15_MINUTES", limits.ChallengeIPPer15Minutes, 1, limits.ChallengeIPPer15Minutes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.InvalidIGSIDPer15Minutes, err = boundedIntEnv("INSTAGRAM_INVALID_IGSID_PER_15_MINUTES", limits.InvalidIGSIDPer15Minutes, 1, limits.InvalidIGSIDPer15Minutes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.InvalidIPPer15Minutes, err = boundedIntEnv("INSTAGRAM_INVALID_IP_PER_15_MINUTES", limits.InvalidIPPer15Minutes, 1, limits.InvalidIPPer15Minutes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ConfirmationDIDPerHour, err = boundedIntEnv("INSTAGRAM_CONFIRMATION_DID_PER_HOUR", limits.ConfirmationDIDPerHour, 1, limits.ConfirmationDIDPerHour); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ConfirmationDevicePerHour, err = boundedIntEnv("INSTAGRAM_CONFIRMATION_DEVICE_PER_HOUR", limits.ConfirmationDevicePerHour, 1, limits.ConfirmationDevicePerHour); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ImportsDIDPerHour, err = boundedIntEnv("INSTAGRAM_IMPORTS_DID_PER_HOUR", limits.ImportsDIDPerHour, 1, limits.ImportsDIDPerHour); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ImportsDevicePerHour, err = boundedIntEnv("INSTAGRAM_IMPORTS_DEVICE_PER_HOUR", limits.ImportsDevicePerHour, 1, limits.ImportsDevicePerHour); err != nil {
		return InstagramLimits{}, err
	}
	if limits.ImportMaxEntries, err = boundedIntEnv("INSTAGRAM_IMPORT_MAX_ENTRIES", limits.ImportMaxEntries, 1, limits.ImportMaxEntries); err != nil {
		return InstagramLimits{}, err
	}
	if limits.PageMax, err = boundedIntEnv("INSTAGRAM_PAGE_MAX", limits.PageMax, 1, limits.PageMax); err != nil {
		return InstagramLimits{}, err
	}
	if limits.PageDefault, err = boundedIntEnv("INSTAGRAM_PAGE_DEFAULT", limits.PageDefault, 1, limits.PageDefault); err != nil {
		return InstagramLimits{}, err
	}
	if limits.PageDefault > limits.PageMax {
		return InstagramLimits{}, errors.New("INSTAGRAM_PAGE_DEFAULT must not exceed INSTAGRAM_PAGE_MAX")
	}
	if limits.MetaHTTPTimeout, err = boundedDurationEnv("INSTAGRAM_META_HTTP_TIMEOUT", limits.MetaHTTPTimeout, limits.MetaHTTPTimeout); err != nil {
		return InstagramLimits{}, err
	}
	if limits.MetaResponseLimitBytes, err = boundedInt64Env("INSTAGRAM_META_RESPONSE_LIMIT_BYTES", limits.MetaResponseLimitBytes, 1, limits.MetaResponseLimitBytes); err != nil {
		return InstagramLimits{}, err
	}
	if limits.MetaLookupConcurrency, err = boundedIntEnv("INSTAGRAM_META_LOOKUP_CONCURRENCY", limits.MetaLookupConcurrency, 1, limits.MetaLookupConcurrency); err != nil {
		return InstagramLimits{}, err
	}
	if limits.MetaLookupsPerIGSIDPerHour, err = boundedIntEnv("INSTAGRAM_META_LOOKUPS_PER_IGSID_HOUR", limits.MetaLookupsPerIGSIDPerHour, 1, limits.MetaLookupsPerIGSIDPerHour); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerConcurrency, err = boundedIntEnv("INSTAGRAM_WORKER_CONCURRENCY", limits.WorkerConcurrency, 1, limits.WorkerConcurrency); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerLeaseDuration, err = boundedDurationEnv("INSTAGRAM_WORKER_LEASE_DURATION", limits.WorkerLeaseDuration, limits.WorkerLeaseDuration); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerMaxAttempts, err = boundedIntEnv("INSTAGRAM_WORKER_MAX_ATTEMPTS", limits.WorkerMaxAttempts, 1, limits.WorkerMaxAttempts); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerBackoffMax, err = boundedDurationEnv("INSTAGRAM_WORKER_BACKOFF_MAX", limits.WorkerBackoffMax, limits.WorkerBackoffMax); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerBackoffInitial, err = boundedDurationEnv("INSTAGRAM_WORKER_BACKOFF_INITIAL", limits.WorkerBackoffInitial, limits.WorkerBackoffInitial); err != nil {
		return InstagramLimits{}, err
	}
	if limits.WorkerBackoffInitial > limits.WorkerBackoffMax {
		return InstagramLimits{}, errors.New("INSTAGRAM_WORKER_BACKOFF_INITIAL must not exceed INSTAGRAM_WORKER_BACKOFF_MAX")
	}
	if limits.WorkerMaxProcessingAge, err = boundedDurationEnv("INSTAGRAM_WORKER_MAX_PROCESSING_AGE", limits.WorkerMaxProcessingAge, limits.WorkerMaxProcessingAge); err != nil {
		return InstagramLimits{}, err
	}
	if limits.DMReplyWindow, err = boundedDurationEnv("INSTAGRAM_DM_REPLY_WINDOW", limits.DMReplyWindow, limits.DMReplyWindow); err != nil {
		return InstagramLimits{}, err
	}
	if limits.NotificationWindow, err = boundedDurationEnv("INSTAGRAM_NOTIFICATION_WINDOW", limits.NotificationWindow, limits.NotificationWindow); err != nil {
		return InstagramLimits{}, err
	}
	if limits.NotificationCountCap, err = boundedIntEnv("INSTAGRAM_NOTIFICATION_COUNT_CAP", limits.NotificationCountCap, 1, limits.NotificationCountCap); err != nil {
		return InstagramLimits{}, err
	}
	if limits.OperatorBatchMax, err = boundedIntEnv("INSTAGRAM_OPERATOR_BATCH_MAX", limits.OperatorBatchMax, 1, limits.OperatorBatchMax); err != nil {
		return InstagramLimits{}, err
	}
	return limits, nil
}

func loadInstagramDeploymentConfig(env Env, dataAvailable bool) (InstagramDeploymentConfig, error) {
	replicaCount, err := positiveIntEnv("APPVIEW_REPLICA_COUNT", 1)
	if err != nil {
		return InstagramDeploymentConfig{}, err
	}
	sharedRateLimits, err := boolEnv("INSTAGRAM_SHARED_RATE_LIMITS", false)
	if err != nil {
		return InstagramDeploymentConfig{}, err
	}
	trustedProxyCIDRs, err := parsePrefixListEnv("INSTAGRAM_TRUSTED_PROXY_CIDRS")
	if err != nil {
		return InstagramDeploymentConfig{}, err
	}
	if env == EnvProd && dataAvailable && replicaCount > 1 && !sharedRateLimits {
		return InstagramDeploymentConfig{}, errors.New("INSTAGRAM_SHARED_RATE_LIMITS=true is required for multi-replica Instagram operation")
	}
	return InstagramDeploymentConfig{
		replicaCount:      replicaCount,
		sharedRateLimits:  sharedRateLimits,
		trustedProxyCIDRs: trustedProxyCIDRs,
	}, nil
}

func boundedDurationEnv(key string, def, max time.Duration) (time.Duration, error) {
	value, err := durationEnv(key, def)
	if err != nil {
		return 0, err
	}
	if value <= 0 || value > max {
		return 0, fmt.Errorf("%s: must be positive and no greater than %s", key, max)
	}
	return value, nil
}

func positiveIntEnv(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	value, err := boundedIntEnv(key, def, 1, int(^uint(0)>>1))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parsePrefixListEnv(key string) ([]netip.Prefix, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, nil
	}
	values := strings.Split(raw, ",")
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("%s: invalid CIDR", key)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func parseInstagramHTTPSURL(key, raw string, rootOnly bool) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" {
		return nil, fmt.Errorf("%s must be an absolute HTTPS URL without credentials or a fragment", key)
	}
	if rootOnly && ((parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "") {
		return nil, fmt.Errorf("%s must not contain a path or query", key)
	}
	if rootOnly {
		parsed.Path = ""
	}
	return parsed, nil
}

func unsafeInstagramPlaceholder(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"change_me", "changeme", "replace_me", "replace-me", "placeholder"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func redactedInstagramAPIBase(value *url.URL) string {
	if value == nil {
		return ""
	}
	return value.Scheme + "://" + value.Host
}
