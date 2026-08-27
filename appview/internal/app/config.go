// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

const (
	defaultJSONBodyLimitBytes               int64 = 1024 * 1024
	maxTapAckTimeout                              = 2 * time.Minute
	maxTapTerminalTransactionBudget               = 30 * time.Second
	maxTapAckSafetyMargin                         = 30 * time.Second
	maxTapReconnect                               = 10 * time.Minute
	maxTapWorkerPollInterval                      = time.Minute
	maxTapWorkerLeaseDuration                     = 10 * time.Minute
	maxTapWorkerBackoff                           = time.Hour
	maxOAuthSessionAbsoluteLifetime               = 180 * 24 * time.Hour
	maxCraftskySessionInactivity                  = 180 * 24 * time.Hour
	maxOAuthAuthRequestExpiry                     = time.Hour
	maxCraftskyActivityWriteInterval              = 24 * time.Hour
	maxPushPollInterval                           = time.Hour
	maxPushLeaseDuration                          = time.Hour
	maxPushSendTimeout                            = 10 * time.Minute
	maxPushFinalizationMargin                     = time.Minute
	maxOwnerFenceAcquireTimeout                   = time.Minute
	maxPDSEffectTimeout                           = 10 * time.Minute
	maxScheduledMediaPutTimeout                   = 10 * time.Minute
	maxHTTPConnections                            = 10_000
	maxHTTPInFlightRequests                       = 10_000
	maxHTTPReadHeaderTimeout                      = 30 * time.Second
	maxHTTPReadTimeout                            = 5 * time.Minute
	maxHTTPWriteTimeout                           = 20 * time.Minute
	maxHTTPIdleTimeout                            = 5 * time.Minute
	maxHTTPHeaderBytes                            = 1 << 20
	maxHTTPOuterRateWindow                        = time.Hour
	maxHTTPOuterLimit                             = 1_000_000
	maxHTTPLimiterCapacity                        = 1_000_000
	maxHTTPLimiterIdleTTL                         = 24 * time.Hour
	maxHTTPJSONBodyReadTimeout                    = 90 * time.Second
	maxHTTPUploadBodyReadTimeout                  = 5 * time.Minute
	maxOAuthPendingAuthRequestCapacity            = 1_000_000
	maxOAuthAuthRequestTerminalRetention          = 30 * 24 * time.Hour
	maxOAuthAuthRequestSweepInterval              = time.Hour
	maxOAuthAuthRequestSweepBatch                 = 10_000
	maxAuthLifecycleWorkerPollInterval            = time.Hour
	maxAuthLifecycleWorkerLeaseDuration           = time.Hour
	maxAuthLifecycleWorkerOperationTimeout        = 10 * time.Minute
	maxAuthLifecycleWorkerAttempts                = 100
	maxAuthLifecycleWorkerBackoff                 = 24 * time.Hour
	maxAuthLifecycleWorkerBatch                   = 1000
	maxAccountDeletionIntentTTL                   = time.Hour
	maxAccountDeletionIntentSweepInterval         = time.Hour
	maxAccountDeletionIntentSweepBatch            = 1000
	maxTerminalPurgePollInterval                  = time.Hour
	maxTerminalPurgeLeaseDuration                 = time.Hour
	maxTerminalPurgeRetryDelay                    = time.Hour
	maxTerminalPurgeBatch                         = 1000
	maxIdentityCacheRefreshPollInterval           = time.Hour
	maxIdentityCacheRefreshOperationTimeout       = time.Minute
	maxIdentityCacheRefreshRetryDelay             = 24 * time.Hour
	maxIdentityCacheRefreshBatch                  = 1000
	httpWriteResponseSafetyMargin                 = 5 * time.Second
)

// OAuthMode selects the one OAuth client shape a deployment is allowed to
// construct. It is derived from Env rather than inferred from an empty URL.
type OAuthMode string

const (
	OAuthModeLocalhost    OAuthMode = "localhost"
	OAuthModeConfidential OAuthMode = "confidential"
)

// Secret keeps accidental fmt-based diagnostics from exposing credential
// material. Reveal is reserved for the narrow dependency/request boundary
// that must consume the value.
type Secret string

func (Secret) String() string   { return "[REDACTED]" }
func (Secret) GoString() string { return "app.Secret([REDACTED])" }

// Reveal returns the credential for the narrow boundary that consumes it.
func (secret Secret) Reveal() string {
	return string(secret)
}

// OAuthDeployment is the immutable, validated client identity used for one
// process. Endpoint URLs are derived from one canonical origin in production.
type OAuthDeployment struct {
	Mode            OAuthMode
	PublicOrigin    url.URL
	ClientID        url.URL
	CallbackURL     url.URL
	JWKSURL         url.URL
	ClientSecretKey Secret
	ClientKeyID     string
	Scopes          []string
}

// FederatedHTTPConfig contains only lowerable operational budgets. The
// federatedhttp package owns the hard security ceilings; configuration cannot
// disable or enlarge them.
type FederatedHTTPConfig struct {
	Transport     federatedhttp.TransportProfile
	OAuthMetadata federatedhttp.Profile
	OAuthRequest  federatedhttp.Profile
	PDSJSON       federatedhttp.Profile
	PDSUpload     federatedhttp.Profile
}

// Env identifies which deployment environment the process is running in.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// ParseEnv converts the string form of an environment into an Env.
// Case-sensitive: "DEV" returns an error.
func ParseEnv(s string) (Env, error) {
	switch s {
	case string(EnvDev):
		return EnvDev, nil
	case string(EnvProd):
		return EnvProd, nil
	default:
		return "", fmt.Errorf("unknown env %q: expected %q or %q", s, EnvDev, EnvProd)
	}
}

// Config is the validated, fully-resolved configuration for one process.
//
// It is produced by LoadConfig, which merges an .env file with os.Getenv
// (with os.Getenv winning on conflicts). Required fields cause LoadConfig
// to fail loudly; the Config that reaches Deps is always complete.
type Config struct {
	Env            Env
	DatabaseURL    string
	AllowedOrigins []string
	ExpectedHosts  []string
	// ExpectedHostAllowAnyPort is true only for loopback development, where
	// Compose assigns a checkout-specific host port.
	ExpectedHostAllowAnyPort bool
	DevDID                   string // populated in dev only; empty in prod
	PublishHost              string
	DevRemoteAccess          bool
	DevProtectedTransport    bool
	DevAuthSecret            Secret

	// HTTP admission and OAuth request retention. These values are explicit
	// deployment budgets rather than hidden constructor defaults.
	HTTPMaxConnections                    int
	HTTPMaxInFlightRequests               int
	HTTPReadHeaderTimeout                 time.Duration
	HTTPReadTimeout                       time.Duration
	HTTPWriteTimeout                      time.Duration
	HTTPIdleTimeout                       time.Duration
	HTTPMaxHeaderBytes                    int
	HTTPTrustedProxyCIDRs                 []string
	HTTPClientIPv6PrefixBits              int
	HTTPOuterRateWindow                   time.Duration
	HTTPOuterGlobalLimit                  int
	HTTPOuterClientLimit                  int
	HTTPLimiterCapacity                   int
	HTTPLimiterIdleTTL                    time.Duration
	HTTPJSONBodyReadTimeout               time.Duration
	HTTPUploadBodyReadTimeout             time.Duration
	OAuthPendingAuthRequestCapacity       int
	OAuthAuthRequestTerminalRetention     time.Duration
	OAuthAuthRequestSweepInterval         time.Duration
	OAuthAuthRequestSweepBatch            int
	OAuthRevocationPollInterval           time.Duration
	OAuthRevocationBatchSize              int
	OAuthRevocationLeaseDuration          time.Duration
	OAuthRevocationOperationTimeout       time.Duration
	OAuthRevocationMaxAttempts            int
	OAuthRevocationBackoffMin             time.Duration
	OAuthRevocationBackoffMax             time.Duration
	OAuthRevocationMaxCredentialRetention time.Duration
	AuthAuxiliaryCleanupPollInterval      time.Duration
	AuthAuxiliaryCleanupBatchSize         int
	AuthAuxiliaryCleanupLeaseDuration     time.Duration
	AuthAuxiliaryCleanupOperationTimeout  time.Duration
	AuthAuxiliaryCleanupMaxAttempts       int
	AuthAuxiliaryCleanupBackoffMin        time.Duration
	AuthAuxiliaryCleanupBackoffMax        time.Duration
	SessionExpirySweepInterval            time.Duration
	SessionExpirySweepBatch               int
	AccountDeletionIntentTTL              time.Duration
	AccountDeletionIntentSweepInterval    time.Duration
	AccountDeletionIntentSweepBatch       int
	TerminalPurgePollInterval             time.Duration
	TerminalPurgeComponentLimit           int
	TerminalPurgeRowBatchSize             int
	TerminalPurgeLeaseDuration            time.Duration
	TerminalPurgeRetryDelay               time.Duration
	IdentityCacheRefreshPollInterval      time.Duration
	IdentityCacheRefreshBatchSize         int
	IdentityCacheRefreshOperationTimeout  time.Duration
	IdentityCacheRefreshRetryDelay        time.Duration

	TapWSURL                      string
	TapAckTimeout                 time.Duration
	TapTerminalTransactionBudget  time.Duration
	TapAckSafetyMargin            time.Duration
	TapReconnectMax               time.Duration
	TapProjectionPollInterval     time.Duration
	TapProjectionLeaseDuration    time.Duration
	TapProjectionBatchSize        int
	TapProjectionBackoffMin       time.Duration
	TapProjectionBackoffMax       time.Duration
	TapRepositoryPollInterval     time.Duration
	TapRepositoryLeaseDuration    time.Duration
	TapRepositoryBatchSize        int
	TapRepositoryBackoffMin       time.Duration
	TapRepositoryBackoffMax       time.Duration
	TapQuarantinePollInterval     time.Duration
	TapQuarantineLeaseDuration    time.Duration
	TapQuarantineOperationTimeout time.Duration
	TapQuarantineBatchSize        int
	PushEnabled                   bool
	FirebaseProjectID             string
	PushBatchSize                 int
	PushConcurrency               int
	PushPollInterval              time.Duration
	PushLeaseDuration             time.Duration
	PushSendTimeout               time.Duration
	PushFinalizationMargin        time.Duration

	// Owner effect boundaries. The object backend currently has no proven
	// finite server-side settlement bound, so cleanup retains exact-key
	// tombstones rather than inferring completion from elapsed time.
	OwnerFenceAcquireTimeout time.Duration
	PDSEffectTimeout         time.Duration
	ScheduledMediaPutTimeout time.Duration
	FederatedHTTP            FederatedHTTPConfig

	// Instagram migration has separate private-data and external Meta
	// availability so a provider outage cannot lock members out of retained
	// imports or privacy controls.
	InstagramData       InstagramDataConfig
	InstagramMeta       InstagramMetaConfig
	InstagramLimits     InstagramLimits
	InstagramDeployment InstagramDeploymentConfig

	// OAuth-related.
	OAuth                                OAuthDeployment
	OAuthSessionAbsoluteLifetime         time.Duration // default 180d
	CraftskySessionInactivity            time.Duration // default 30d
	OAuthAuthRequestExpiry               time.Duration // default 30m
	CraftskySessionActivityWriteInterval time.Duration // default 5m
	OAuthLoginStartTimeout               time.Duration // default 20s
	OAuthCallbackOperationTimeout        time.Duration // default 45s
	OAuthSessionOperationTimeout         time.Duration // default 30s
	OAuthHandoffExchangeTTL              time.Duration // default 10m
	OAuthHandoffConfirmationTTL          time.Duration // default 2m
	OAuthHandoffReceiptKey               Secret
	OAuthHandoffReceiptKeyVersion        int
	VerifiedLinkOrigin                   url.URL
	EnableDevOAuthScheme                 bool

	// Media policy. Defaults preserve the approved image-posting contract;
	// env overrides may lower but not raise these ceilings.
	JSONBodyLimitBytes  int64 // default 1 MiB
	MaxPostImages       int   // default 4, maximum 4
	MaxImageUploadBytes int64 // default 15MB, maximum 15MB
	LinkPreviewsEnabled bool
	ImageDecodeLimits   api.ImageDecodeLimits
	RateLimits          middleware.RateLimitConfig
	ScheduledPostsS3    scheduledposts.S3ObjectStoreConfig

	// Observability. Sentry export/tracing stays disabled unless explicitly
	// configured, and unsafe body logging is local-dev only.
	SentryDSN                 string
	SentryRelease             string
	SentryLogsEnabled         bool
	SentryTracingEnabled      bool
	SentryTracesSampleRate    float64
	SentryMetricsEnabled      bool
	SentryTapTracingEnabled   bool
	SentryTapTracesSampleRate float64
	UnsafeLogResponseBodies   bool
	// UnsafeLogInstagramWebhookBodies is a temporary local capability-spike
	// diagnostic. Production validation always forces it off.
	UnsafeLogInstagramWebhookBodies bool

	// Dev-only synthetic moderation controls. These fields are cleared in prod.
	EnableDevModeration         bool
	DevModerationToken          string
	DevLabelerDID               string
	TrustedModerationSourceDIDs []string
}

// LoadConfig reads environments/<env>.env from envFilePath, layers os.Getenv
// on top (so shell env vars override the file), and validates that every
// required field is set. Missing required fields produce an error naming
// the specific key.
//
// envFilePath is passed explicitly (not derived from env) so tests can
// point at a temp file. Callers in main code will pass
// "environments/<env>.env".
func LoadConfig(env Env, envFilePath string) (Config, error) {
	// godotenv.Load merges the file into os.Environ without overwriting
	// existing values — exactly the "os.Getenv wins" semantics we want.
	// A missing file is not fatal: os.Getenv alone may have everything.
	_ = godotenv.Load(envFilePath)

	cfg := Config{
		Env:         env,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DevDID:      os.Getenv("CRAFTSKY_DEV_DID"),
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		for _, o := range strings.Split(origins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
			}
		}
	}

	cfg.TapWSURL = os.Getenv("TAP_WS_URL")

	var err error
	if cfg.TapAckTimeout, err = boundedPositiveDurationEnv("TAP_ACK_TIMEOUT", 10*time.Second, maxTapAckTimeout); err != nil {
		return Config{}, err
	}
	if cfg.TapTerminalTransactionBudget, err = boundedPositiveDurationEnv("TAP_TERMINAL_TRANSACTION_BUDGET", time.Second, maxTapTerminalTransactionBudget); err != nil {
		return Config{}, err
	}
	if cfg.TapAckSafetyMargin, err = boundedPositiveDurationEnv("TAP_ACK_SAFETY_MARGIN", 500*time.Millisecond, maxTapAckSafetyMargin); err != nil {
		return Config{}, err
	}
	if cfg.TapReconnectMax, err = boundedPositiveDurationEnv("TAP_RECONNECT_MAX", 30*time.Second, maxTapReconnect); err != nil {
		return Config{}, err
	}

	if _, removed := os.LookupEnv("TAP_MAX_RETRIES"); removed {
		return Config{}, fmt.Errorf("TAP_MAX_RETRIES has been removed; retryable events are never dropped")
	}
	if cfg.TapProjectionPollInterval, err = boundedPositiveDurationEnv("TAP_PROJECTION_POLL_INTERVAL", 250*time.Millisecond, maxTapWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.TapProjectionLeaseDuration, err = boundedPositiveDurationEnv("TAP_PROJECTION_LEASE_DURATION", 30*time.Second, maxTapWorkerLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.TapProjectionBatchSize, err = boundedIntEnv("TAP_PROJECTION_BATCH_SIZE", 32, 1, 1000); err != nil {
		return Config{}, err
	}
	if cfg.TapProjectionBackoffMin, err = boundedPositiveDurationEnv("TAP_PROJECTION_BACKOFF_MIN", time.Second, maxTapWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.TapProjectionBackoffMax, err = boundedPositiveDurationEnv("TAP_PROJECTION_BACKOFF_MAX", 5*time.Minute, maxTapWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.TapRepositoryPollInterval, err = boundedPositiveDurationEnv("TAP_REPOSITORY_POLL_INTERVAL", time.Second, maxTapWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.TapRepositoryLeaseDuration, err = boundedPositiveDurationEnv("TAP_REPOSITORY_LEASE_DURATION", 45*time.Second, maxTapWorkerLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.TapRepositoryBatchSize, err = boundedIntEnv("TAP_REPOSITORY_BATCH_SIZE", 8, 1, 1000); err != nil {
		return Config{}, err
	}
	if cfg.TapRepositoryBackoffMin, err = boundedPositiveDurationEnv("TAP_REPOSITORY_BACKOFF_MIN", 2*time.Second, maxTapWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.TapRepositoryBackoffMax, err = boundedPositiveDurationEnv("TAP_REPOSITORY_BACKOFF_MAX", 5*time.Minute, maxTapWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.TapQuarantinePollInterval, err = boundedPositiveDurationEnv("TAP_QUARANTINE_POLL_INTERVAL", time.Second, maxTapWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.TapQuarantineLeaseDuration, err = boundedPositiveDurationEnv("TAP_QUARANTINE_LEASE_DURATION", 30*time.Second, maxTapWorkerLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.TapQuarantineOperationTimeout, err = boundedPositiveDurationEnv("TAP_QUARANTINE_OPERATION_TIMEOUT", 10*time.Second, maxTapAckTimeout); err != nil {
		return Config{}, err
	}
	if cfg.TapQuarantineBatchSize, err = boundedIntEnv("TAP_QUARANTINE_BATCH_SIZE", 8, 1, 1000); err != nil {
		return Config{}, err
	}

	if _, legacy := os.LookupEnv("OAUTH_HOSTNAME"); legacy {
		return Config{}, fmt.Errorf("OAUTH_HOSTNAME has been removed; set OAUTH_PUBLIC_ORIGIN")
	}
	oauthScopes, err := parseOAuthScopes(getEnvWithDefault("OAUTH_SCOPES", "atproto transition:generic"))
	if err != nil {
		return Config{}, err
	}
	switch env {
	case EnvDev:
		for _, key := range []string{"OAUTH_PUBLIC_ORIGIN", "OAUTH_CLIENT_SECRET_KEY", "OAUTH_CLIENT_SECRET_KEY_ID"} {
			if _, exists := os.LookupEnv(key); exists {
				return Config{}, fmt.Errorf("%s is not allowed in localhost OAuth mode", key)
			}
		}
		callback, parseErr := parseLoopbackOAuthCallback(getEnvWithDefault("OAUTH_CALLBACK_URL", "http://127.0.0.1:18080/oauth/callback"))
		if parseErr != nil {
			return Config{}, parseErr
		}
		cfg.OAuth = OAuthDeployment{
			Mode:        OAuthModeLocalhost,
			CallbackURL: callback,
			Scopes:      append([]string(nil), oauthScopes...),
		}
		cfg.ExpectedHosts, err = expectedHostsEnv()
		if err != nil {
			return Config{}, err
		}
		if len(cfg.ExpectedHosts) == 0 {
			cfg.ExpectedHosts = []string{"127.0.0.1", "localhost"}
		}
		cfg.ExpectedHostAllowAnyPort = true
		cfg.PublishHost = getEnvWithDefault("APPVIEW_PUBLISH_HOST", "127.0.0.1")
		publishIP := net.ParseIP(cfg.PublishHost)
		if publishIP == nil {
			return Config{}, fmt.Errorf("APPVIEW_PUBLISH_HOST must be an IP literal")
		}
		if cfg.DevRemoteAccess, err = boolEnv("APPVIEW_DEV_REMOTE_ACCESS", false); err != nil {
			return Config{}, err
		}
		if cfg.DevProtectedTransport, err = boolEnv("APPVIEW_DEV_PROTECTED_TRANSPORT", false); err != nil {
			return Config{}, err
		}
		cfg.DevAuthSecret = Secret(os.Getenv("APPVIEW_DEV_AUTH_SECRET"))
		remotePublish := !publishIP.IsLoopback()
		switch {
		case remotePublish && !cfg.DevRemoteAccess:
			return Config{}, fmt.Errorf("APPVIEW_DEV_REMOTE_ACCESS=true is required for a non-loopback APPVIEW_PUBLISH_HOST")
		case cfg.DevRemoteAccess && !remotePublish:
			return Config{}, fmt.Errorf("APPVIEW_PUBLISH_HOST must be non-loopback when APPVIEW_DEV_REMOTE_ACCESS=true")
		case !cfg.DevRemoteAccess && (cfg.DevProtectedTransport || cfg.DevAuthSecret != ""):
			return Config{}, fmt.Errorf("APPVIEW_DEV_REMOTE_ACCESS=true is required before configuring remote dev authorization")
		case cfg.DevRemoteAccess && !cfg.DevProtectedTransport:
			return Config{}, fmt.Errorf("APPVIEW_DEV_PROTECTED_TRANSPORT=true is required for remote development")
		case cfg.DevRemoteAccess && !isHeaderSafeSecret(cfg.DevAuthSecret.Reveal()):
			return Config{}, fmt.Errorf("APPVIEW_DEV_AUTH_SECRET must contain at least 32 printable ASCII bytes with sufficient character diversity")
		case cfg.DevRemoteAccess && os.Getenv("APPVIEW_EXPECTED_HOSTS") == "":
			return Config{}, fmt.Errorf("APPVIEW_EXPECTED_HOSTS is required for remote development")
		}
		if cfg.DevRemoteAccess {
			cfg.ExpectedHostAllowAnyPort = false
			if err := normalizeRemoteExpectedHosts(cfg.ExpectedHosts); err != nil {
				return Config{}, err
			}
			for i := range cfg.ExpectedHosts {
				cfg.ExpectedHosts[i] = strings.ToLower(cfg.ExpectedHosts[i])
			}
		} else if err := normalizeLocalExpectedHosts(cfg.ExpectedHosts); err != nil {
			return Config{}, err
		}
	case EnvProd:
		for _, key := range []string{
			"APPVIEW_PUBLISH_HOST",
			"APPVIEW_DEV_REMOTE_ACCESS",
			"APPVIEW_DEV_PROTECTED_TRANSPORT",
			"APPVIEW_DEV_AUTH_SECRET",
		} {
			if _, exists := os.LookupEnv(key); exists {
				return Config{}, fmt.Errorf("%s is not allowed in prod", key)
			}
		}
		if _, exists := os.LookupEnv("OAUTH_CALLBACK_URL"); exists {
			return Config{}, fmt.Errorf("OAUTH_CALLBACK_URL is not allowed in prod; it is derived from OAUTH_PUBLIC_ORIGIN")
		}
		origin, parseErr := parseCanonicalPublicOrigin(os.Getenv("OAUTH_PUBLIC_ORIGIN"))
		if parseErr != nil {
			return Config{}, parseErr
		}
		cfg.OAuth = OAuthDeployment{
			Mode:            OAuthModeConfidential,
			PublicOrigin:    origin,
			ClientID:        resolveOriginPath(origin, "/oauth/client-metadata.json"),
			CallbackURL:     resolveOriginPath(origin, "/oauth/callback"),
			JWKSURL:         resolveOriginPath(origin, "/oauth/jwks.json"),
			ClientSecretKey: Secret(os.Getenv("OAUTH_CLIENT_SECRET_KEY")),
			ClientKeyID:     os.Getenv("OAUTH_CLIENT_SECRET_KEY_ID"),
			Scopes:          append([]string(nil), oauthScopes...),
		}
		cfg.ExpectedHosts, err = expectedHostsEnv()
		if err != nil {
			return Config{}, err
		}
		if len(cfg.ExpectedHosts) == 0 {
			cfg.ExpectedHosts = []string{origin.Host}
		}
		if len(cfg.ExpectedHosts) != 1 || !strings.EqualFold(cfg.ExpectedHosts[0], origin.Host) {
			return Config{}, fmt.Errorf("APPVIEW_EXPECTED_HOSTS must contain only the OAUTH_PUBLIC_ORIGIN host in prod")
		}
		cfg.ExpectedHosts[0] = origin.Host
	default:
		return Config{}, fmt.Errorf("unknown env %q: expected %q or %q", env, EnvDev, EnvProd)
	}

	for _, removed := range []string{
		"OAUTH_SESSION_EXPIRY",
		"OAUTH_SESSION_INACTIVITY",
		"CRAFTSKY_SESSION_LAST_SEEN_THROTTLE",
	} {
		if _, exists := os.LookupEnv(removed); exists {
			return Config{}, fmt.Errorf("%s has been removed", removed)
		}
	}
	if cfg.OAuthSessionAbsoluteLifetime, err = boundedPositiveDurationEnv("OAUTH_SESSION_ABSOLUTE_LIFETIME", 180*24*time.Hour, maxOAuthSessionAbsoluteLifetime); err != nil {
		return Config{}, err
	}
	if cfg.CraftskySessionInactivity, err = boundedPositiveDurationEnv("CRAFTSKY_SESSION_INACTIVITY", 30*24*time.Hour, maxCraftskySessionInactivity); err != nil {
		return Config{}, err
	}
	if cfg.OAuthAuthRequestExpiry, err = boundedPositiveDurationEnv("OAUTH_AUTH_REQUEST_EXPIRY", 30*time.Minute, maxOAuthAuthRequestExpiry); err != nil {
		return Config{}, err
	}
	if cfg.CraftskySessionActivityWriteInterval, err = boundedPositiveDurationEnv("CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL", 5*time.Minute, maxCraftskyActivityWriteInterval); err != nil {
		return Config{}, err
	}
	if cfg.OAuthLoginStartTimeout, err = boundedPositiveDurationEnv(
		"OAUTH_LOGIN_START_TIMEOUT",
		auth.DefaultOAuthLoginStartTimeout,
		auth.MaximumOAuthLoginStartTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.OAuthCallbackOperationTimeout, err = boundedPositiveDurationEnv(
		"OAUTH_CALLBACK_OPERATION_TIMEOUT",
		auth.DefaultOAuthCallbackOperationTimeout,
		auth.MaximumOAuthCallbackOperationTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.OAuthSessionOperationTimeout, err = boundedPositiveDurationEnv(
		"OAUTH_SESSION_OPERATION_TIMEOUT",
		auth.DefaultOAuthSessionOperationTimeout,
		auth.MaximumOAuthSessionOperationTimeout,
	); err != nil {
		return Config{}, err
	}
	if cfg.OAuthHandoffExchangeTTL, err = boundedPositiveDurationEnv("OAUTH_HANDOFF_EXCHANGE_TTL", 10*time.Minute, 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.OAuthHandoffConfirmationTTL, err = boundedPositiveDurationEnv("OAUTH_HANDOFF_CONFIRMATION_TTL", 2*time.Minute, 10*time.Minute); err != nil {
		return Config{}, err
	}
	cfg.OAuthHandoffReceiptKey = Secret(strings.TrimSpace(os.Getenv("OAUTH_HANDOFF_RECEIPT_KEY")))
	if cfg.OAuthHandoffReceiptKeyVersion, err = boundedIntEnv("OAUTH_HANDOFF_RECEIPT_KEY_VERSION", 1, 1, 1_000_000); err != nil {
		return Config{}, err
	}
	if cfg.OAuthHandoffReceiptKey != "" {
		if _, err := decodeHandoffReceiptKey(cfg.OAuthHandoffReceiptKey); err != nil {
			return Config{}, err
		}
	}
	verifiedOrigin := strings.TrimSpace(os.Getenv("APP_VERIFIED_LINK_ORIGIN"))
	if verifiedOrigin == "" {
		if env == EnvProd {
			verifiedOrigin = cfg.OAuth.PublicOrigin.String()
		} else {
			verifiedOrigin = "https://app.craftsky.social"
		}
	}
	cfg.VerifiedLinkOrigin, err = parseCanonicalPublicOrigin(verifiedOrigin)
	if err != nil {
		return Config{}, fmt.Errorf("APP_VERIFIED_LINK_ORIGIN must be a canonical public HTTPS origin")
	}
	if cfg.EnableDevOAuthScheme, err = boolEnv("APPVIEW_ENABLE_DEV_OAUTH_SCHEME", false); err != nil {
		return Config{}, err
	}
	if env == EnvProd && cfg.EnableDevOAuthScheme {
		return Config{}, fmt.Errorf("APPVIEW_ENABLE_DEV_OAUTH_SCHEME is not allowed in prod")
	}
	if cfg.PushEnabled, err = boolEnv("PUSH_ENABLED", false); err != nil {
		return Config{}, err
	}
	cfg.FirebaseProjectID = strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if cfg.PushBatchSize, err = boundedIntEnv("PUSH_BATCH_SIZE", 100, 1, 500); err != nil {
		return Config{}, err
	}
	if cfg.PushConcurrency, err = boundedIntEnv("PUSH_CONCURRENCY", 4, 1, 64); err != nil {
		return Config{}, err
	}
	if cfg.PushPollInterval, err = boundedPositiveDurationEnv("PUSH_POLL_INTERVAL", time.Second, maxPushPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.PushLeaseDuration, err = boundedPositiveDurationEnv("PUSH_LEASE_DURATION", time.Minute, maxPushLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.PushSendTimeout, err = boundedPositiveDurationEnv("PUSH_SEND_TIMEOUT", 10*time.Second, maxPushSendTimeout); err != nil {
		return Config{}, err
	}
	if cfg.PushFinalizationMargin, err = boundedPositiveDurationEnv("PUSH_FINALIZATION_MARGIN", 5*time.Second, maxPushFinalizationMargin); err != nil {
		return Config{}, err
	}
	if cfg.OwnerFenceAcquireTimeout, err = boundedPositiveDurationEnv("OWNER_FENCE_ACQUIRE_TIMEOUT", 5*time.Second, maxOwnerFenceAcquireTimeout); err != nil {
		return Config{}, err
	}
	if cfg.PDSEffectTimeout, err = boundedPositiveDurationEnv("PDS_EFFECT_TIMEOUT", 10*time.Second, maxPDSEffectTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ScheduledMediaPutTimeout, err = boundedPositiveDurationEnv("SCHEDULED_MEDIA_PUT_TIMEOUT", 30*time.Second, maxScheduledMediaPutTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPMaxConnections, err = boundedIntEnv("HTTP_MAX_CONNECTIONS", 512, 1, maxHTTPConnections); err != nil {
		return Config{}, err
	}
	if cfg.HTTPMaxInFlightRequests, err = boundedIntEnv("HTTP_MAX_IN_FLIGHT_REQUESTS", 256, 1, maxHTTPInFlightRequests); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadHeaderTimeout, err = boundedPositiveDurationEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second, maxHTTPReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPReadTimeout, err = boundedPositiveDurationEnv("HTTP_READ_TIMEOUT", 90*time.Second, maxHTTPReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPWriteTimeout, err = boundedPositiveDurationEnv("HTTP_WRITE_TIMEOUT", 2*time.Minute, maxHTTPWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPIdleTimeout, err = boundedPositiveDurationEnv("HTTP_IDLE_TIMEOUT", time.Minute, maxHTTPIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPMaxHeaderBytes, err = boundedIntEnv("HTTP_MAX_HEADER_BYTES", 32<<10, 1, maxHTTPHeaderBytes); err != nil {
		return Config{}, err
	}
	if cfg.HTTPTrustedProxyCIDRs, err = trustedProxyCIDRsEnv("HTTP_TRUSTED_PROXY_CIDRS"); err != nil {
		return Config{}, err
	}
	if cfg.HTTPClientIPv6PrefixBits, err = boundedIntEnv("HTTP_CLIENT_IPV6_PREFIX_BITS", 64, 32, 128); err != nil {
		return Config{}, err
	}
	if cfg.HTTPOuterRateWindow, err = boundedPositiveDurationEnv("HTTP_OUTER_RATE_WINDOW", time.Minute, maxHTTPOuterRateWindow); err != nil {
		return Config{}, err
	}
	if cfg.HTTPOuterGlobalLimit, err = boundedIntEnv("HTTP_OUTER_GLOBAL_LIMIT", 6000, 1, maxHTTPOuterLimit); err != nil {
		return Config{}, err
	}
	if cfg.HTTPOuterClientLimit, err = boundedIntEnv("HTTP_OUTER_CLIENT_LIMIT", 600, 1, maxHTTPOuterLimit); err != nil {
		return Config{}, err
	}
	if cfg.HTTPLimiterCapacity, err = boundedIntEnv("HTTP_LIMITER_CAPACITY", 4096, 1, maxHTTPLimiterCapacity); err != nil {
		return Config{}, err
	}
	if cfg.HTTPLimiterIdleTTL, err = boundedPositiveDurationEnv("HTTP_LIMITER_IDLE_TTL", 10*time.Minute, maxHTTPLimiterIdleTTL); err != nil {
		return Config{}, err
	}
	if cfg.HTTPJSONBodyReadTimeout, err = boundedPositiveDurationEnv("HTTP_JSON_BODY_READ_TIMEOUT", 10*time.Second, maxHTTPJSONBodyReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPUploadBodyReadTimeout, err = boundedPositiveDurationEnv("HTTP_UPLOAD_BODY_READ_TIMEOUT", time.Minute, maxHTTPUploadBodyReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.OAuthPendingAuthRequestCapacity, err = boundedIntEnv("OAUTH_PENDING_AUTH_REQUEST_CAPACITY", 4096, 1, maxOAuthPendingAuthRequestCapacity); err != nil {
		return Config{}, err
	}
	if cfg.OAuthAuthRequestTerminalRetention, err = boundedPositiveDurationEnv("OAUTH_AUTH_REQUEST_TERMINAL_RETENTION", 24*time.Hour, maxOAuthAuthRequestTerminalRetention); err != nil {
		return Config{}, err
	}
	if cfg.OAuthAuthRequestSweepInterval, err = boundedPositiveDurationEnv("OAUTH_AUTH_REQUEST_SWEEP_INTERVAL", time.Minute, maxOAuthAuthRequestSweepInterval); err != nil {
		return Config{}, err
	}
	if cfg.OAuthAuthRequestSweepBatch, err = boundedIntEnv("OAUTH_AUTH_REQUEST_SWEEP_BATCH", 100, 1, maxOAuthAuthRequestSweepBatch); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationPollInterval, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_POLL_INTERVAL", 5*time.Second, maxAuthLifecycleWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationBatchSize, err = boundedIntEnv("OAUTH_REVOCATION_BATCH_SIZE", 20, 1, maxAuthLifecycleWorkerBatch); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationLeaseDuration, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_LEASE_DURATION", time.Minute, maxAuthLifecycleWorkerLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationOperationTimeout, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_OPERATION_TIMEOUT", 20*time.Second, maxAuthLifecycleWorkerOperationTimeout); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationMaxAttempts, err = boundedIntEnv("OAUTH_REVOCATION_MAX_ATTEMPTS", 5, 1, maxAuthLifecycleWorkerAttempts); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationBackoffMin, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_BACKOFF_MIN", time.Minute, maxAuthLifecycleWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationBackoffMax, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_BACKOFF_MAX", time.Hour, maxAuthLifecycleWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.OAuthRevocationMaxCredentialRetention, err = boundedPositiveDurationEnv("OAUTH_REVOCATION_MAX_CREDENTIAL_RETENTION", 24*time.Hour, maxOAuthSessionAbsoluteLifetime); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupPollInterval, err = boundedPositiveDurationEnv("AUTH_AUXILIARY_CLEANUP_POLL_INTERVAL", 5*time.Second, maxAuthLifecycleWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupBatchSize, err = boundedIntEnv("AUTH_AUXILIARY_CLEANUP_BATCH_SIZE", 20, 1, maxAuthLifecycleWorkerBatch); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupLeaseDuration, err = boundedPositiveDurationEnv("AUTH_AUXILIARY_CLEANUP_LEASE_DURATION", time.Minute, maxAuthLifecycleWorkerLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupOperationTimeout, err = boundedPositiveDurationEnv("AUTH_AUXILIARY_CLEANUP_OPERATION_TIMEOUT", 20*time.Second, maxAuthLifecycleWorkerOperationTimeout); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupMaxAttempts, err = boundedIntEnv("AUTH_AUXILIARY_CLEANUP_MAX_ATTEMPTS", 5, 1, maxAuthLifecycleWorkerAttempts); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupBackoffMin, err = boundedPositiveDurationEnv("AUTH_AUXILIARY_CLEANUP_BACKOFF_MIN", time.Minute, maxAuthLifecycleWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.AuthAuxiliaryCleanupBackoffMax, err = boundedPositiveDurationEnv("AUTH_AUXILIARY_CLEANUP_BACKOFF_MAX", time.Hour, maxAuthLifecycleWorkerBackoff); err != nil {
		return Config{}, err
	}
	if cfg.SessionExpirySweepInterval, err = boundedPositiveDurationEnv("AUTH_SESSION_EXPIRY_SWEEP_INTERVAL", time.Minute, maxAuthLifecycleWorkerPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.SessionExpirySweepBatch, err = boundedIntEnv("AUTH_SESSION_EXPIRY_SWEEP_BATCH", 100, 1, maxAuthLifecycleWorkerBatch); err != nil {
		return Config{}, err
	}
	if cfg.AccountDeletionIntentTTL, err = boundedPositiveDurationEnv("ACCOUNT_DELETION_INTENT_TTL", 10*time.Minute, maxAccountDeletionIntentTTL); err != nil {
		return Config{}, err
	}
	if cfg.AccountDeletionIntentSweepInterval, err = boundedPositiveDurationEnv("ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL", time.Minute, maxAccountDeletionIntentSweepInterval); err != nil {
		return Config{}, err
	}
	if cfg.AccountDeletionIntentSweepBatch, err = boundedIntEnv("ACCOUNT_DELETION_INTENT_SWEEP_BATCH", 100, 1, maxAccountDeletionIntentSweepBatch); err != nil {
		return Config{}, err
	}
	if cfg.TerminalPurgePollInterval, err = boundedPositiveDurationEnv("TERMINAL_PURGE_POLL_INTERVAL", time.Second, maxTerminalPurgePollInterval); err != nil {
		return Config{}, err
	}
	if cfg.TerminalPurgeComponentLimit, err = boundedIntEnv("TERMINAL_PURGE_COMPONENT_LIMIT", 16, 1, maxTerminalPurgeBatch); err != nil {
		return Config{}, err
	}
	if cfg.TerminalPurgeRowBatchSize, err = boundedIntEnv("TERMINAL_PURGE_ROW_BATCH_SIZE", 100, 1, maxTerminalPurgeBatch); err != nil {
		return Config{}, err
	}
	if cfg.TerminalPurgeLeaseDuration, err = boundedPositiveDurationEnv("TERMINAL_PURGE_LEASE_DURATION", time.Minute, maxTerminalPurgeLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.TerminalPurgeRetryDelay, err = boundedPositiveDurationEnv("TERMINAL_PURGE_RETRY_DELAY", time.Second, maxTerminalPurgeRetryDelay); err != nil {
		return Config{}, err
	}
	if cfg.IdentityCacheRefreshPollInterval, err = boundedPositiveDurationEnv("IDENTITY_CACHE_REFRESH_POLL_INTERVAL", 5*time.Minute, maxIdentityCacheRefreshPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.IdentityCacheRefreshBatchSize, err = boundedIntEnv("IDENTITY_CACHE_REFRESH_BATCH_SIZE", 100, 1, maxIdentityCacheRefreshBatch); err != nil {
		return Config{}, err
	}
	if cfg.IdentityCacheRefreshOperationTimeout, err = boundedPositiveDurationEnv("IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT", 10*time.Second, maxIdentityCacheRefreshOperationTimeout); err != nil {
		return Config{}, err
	}
	if cfg.IdentityCacheRefreshRetryDelay, err = boundedPositiveDurationEnv("IDENTITY_CACHE_REFRESH_RETRY_DELAY", 15*time.Minute, maxIdentityCacheRefreshRetryDelay); err != nil {
		return Config{}, err
	}
	if cfg.FederatedHTTP, err = federatedHTTPConfigFromEnv(); err != nil {
		return Config{}, err
	}
	if cfg.MaxPostImages, err = boundedIntEnv("MAX_POST_IMAGES", api.DefaultMaxPostImages, 1, api.DefaultMaxPostImages); err != nil {
		return Config{}, err
	}
	if cfg.MaxImageUploadBytes, err = boundedInt64Env("MAX_IMAGE_UPLOAD_BYTES", api.DefaultMaxImageUploadBytes, 1, api.DefaultMaxImageUploadBytes); err != nil {
		return Config{}, err
	}
	if cfg.JSONBodyLimitBytes, err = boundedInt64Env("APPVIEW_JSON_BODY_LIMIT_BYTES", defaultJSONBodyLimitBytes, 1, defaultJSONBodyLimitBytes); err != nil {
		return Config{}, err
	}
	if cfg.LinkPreviewsEnabled, err = boolEnv("LINK_PREVIEWS_ENABLED", env != EnvProd); err != nil {
		return Config{}, err
	}
	imageDefaults := api.DefaultImageDecodeLimits()
	if cfg.ImageDecodeLimits.MaxWidth, err = boundedIntEnv("SCHEDULED_IMAGE_MAX_WIDTH", imageDefaults.MaxWidth, 1, imageDefaults.MaxWidth); err != nil {
		return Config{}, err
	}
	if cfg.ImageDecodeLimits.MaxHeight, err = boundedIntEnv("SCHEDULED_IMAGE_MAX_HEIGHT", imageDefaults.MaxHeight, 1, imageDefaults.MaxHeight); err != nil {
		return Config{}, err
	}
	if cfg.ImageDecodeLimits.MaxPixels, err = boundedUint64Env("SCHEDULED_IMAGE_MAX_PIXELS", imageDefaults.MaxPixels, 1, imageDefaults.MaxPixels); err != nil {
		return Config{}, err
	}
	if cfg.ImageDecodeLimits.MaxAspectRatio, err = boundedIntEnv("SCHEDULED_IMAGE_MAX_ASPECT_RATIO", imageDefaults.MaxAspectRatio, 1, imageDefaults.MaxAspectRatio); err != nil {
		return Config{}, err
	}
	if cfg.ImageDecodeLimits.MaxConcurrentDecodes, err = boundedIntEnv("SCHEDULED_IMAGE_MAX_CONCURRENT_DECODES", imageDefaults.MaxConcurrentDecodes, 1, imageDefaults.MaxConcurrentDecodes); err != nil {
		return Config{}, err
	}
	if cfg.ImageDecodeLimits.AdmissionWait, err = boundedPositiveDurationEnv("SCHEDULED_IMAGE_ADMISSION_WAIT", imageDefaults.AdmissionWait, imageDefaults.AdmissionWait); err != nil {
		return Config{}, err
	}
	cfg.RateLimits = DefaultRateLimitConfig()
	cfg.ScheduledPostsS3 = scheduledposts.S3ObjectStoreConfig{
		Endpoint:        strings.TrimSpace(os.Getenv("SCHEDULED_POSTS_S3_ENDPOINT")),
		Region:          strings.TrimSpace(os.Getenv("SCHEDULED_POSTS_S3_REGION")),
		Bucket:          strings.TrimSpace(os.Getenv("SCHEDULED_POSTS_S3_BUCKET")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("SCHEDULED_POSTS_S3_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("SCHEDULED_POSTS_S3_SECRET_ACCESS_KEY")),
		Environment:     string(env),
	}
	if cfg.ScheduledPostsS3.Endpoint == "" {
		if env == EnvDev {
			cfg.ScheduledPostsS3.Endpoint = "http://minio:9000"
		} else {
			// Deliberately non-routable: production still fails its startup
			// connectivity check until a real private store is configured.
			cfg.ScheduledPostsS3.Endpoint = "https://scheduled-media.invalid"
		}
	}
	if cfg.ScheduledPostsS3.Region == "" {
		cfg.ScheduledPostsS3.Region = "us-east-1"
	}
	if cfg.ScheduledPostsS3.Bucket == "" {
		cfg.ScheduledPostsS3.Bucket = "private-scheduled-media"
	}
	if cfg.ScheduledPostsS3.AccessKeyID == "" {
		cfg.ScheduledPostsS3.AccessKeyID = "not-configured"
	}
	if cfg.ScheduledPostsS3.SecretAccessKey == "" {
		cfg.ScheduledPostsS3.SecretAccessKey = "not-configured"
	}
	cfg.InstagramData, cfg.InstagramMeta, cfg.InstagramLimits, cfg.InstagramDeployment, err = loadInstagramConfig(env)
	if err != nil {
		return Config{}, err
	}
	cfg.SentryDSN = os.Getenv("SENTRY_DSN")
	cfg.SentryRelease = os.Getenv("SENTRY_RELEASE")
	if cfg.SentryLogsEnabled, err = boolEnv("SENTRY_LOGS_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.SentryTracingEnabled, err = boolEnv("SENTRY_TRACING_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.SentryTracesSampleRate, err = sampleRateEnv("SENTRY_TRACES_SAMPLE_RATE"); err != nil {
		return Config{}, err
	}
	if cfg.SentryMetricsEnabled, err = boolEnv("SENTRY_METRICS_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.SentryTapTracingEnabled, err = boolEnv("SENTRY_TAP_TRACING_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.SentryTapTracesSampleRate, err = sampleRateEnv("SENTRY_TAP_TRACES_SAMPLE_RATE"); err != nil {
		return Config{}, err
	}
	if cfg.SentryDSN == "" {
		cfg.SentryLogsEnabled = false
		cfg.SentryTracingEnabled = false
		cfg.SentryTracesSampleRate = 0
		cfg.SentryMetricsEnabled = false
		cfg.SentryTapTracingEnabled = false
		cfg.SentryTapTracesSampleRate = 0
	} else if cfg.SentryTracingEnabled && os.Getenv("SENTRY_TRACES_SAMPLE_RATE") == "" {
		if env == EnvProd {
			cfg.SentryTracesSampleRate = 0.01
		} else {
			cfg.SentryTracesSampleRate = 1
		}
	}
	if cfg.SentryDSN != "" && cfg.SentryTapTracingEnabled && os.Getenv("SENTRY_TAP_TRACES_SAMPLE_RATE") == "" {
		if env == EnvProd {
			cfg.SentryTapTracesSampleRate = 0.01
		} else {
			cfg.SentryTapTracesSampleRate = 1
		}
	}
	if cfg.UnsafeLogResponseBodies, err = boolEnv("APPVIEW_UNSAFE_LOG_RESPONSE_BODIES", false); err != nil {
		return Config{}, err
	}
	if cfg.UnsafeLogInstagramWebhookBodies, err = boolEnv("INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES", false); err != nil {
		return Config{}, err
	}
	if cfg.EnableDevModeration, err = boolEnv("APPVIEW_ENABLE_DEV_MODERATION", false); err != nil {
		return Config{}, err
	}
	if env == EnvDev {
		cfg.DevModerationToken = os.Getenv("APPVIEW_DEV_MODERATION_TOKEN")
		cfg.DevLabelerDID = getEnvWithDefault("CRAFTSKY_DEV_LABELER_DID", "did:plc:labeler")
		cfg.TrustedModerationSourceDIDs = splitCommaEnv("APPVIEW_TRUSTED_MODERATION_SOURCE_DIDS")
		cfg.TrustedModerationSourceDIDs = appendUniqueNonEmpty(cfg.TrustedModerationSourceDIDs, cfg.DevLabelerDID)
	}

	// Required everywhere.
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_ORIGINS is required (comma-separated list)")
	}
	if cfg.TapWSURL == "" {
		return Config{}, fmt.Errorf("TAP_WS_URL is required")
	}

	// Required in dev only.
	if env == EnvDev && cfg.DevDID == "" {
		return Config{}, fmt.Errorf("CRAFTSKY_DEV_DID is required in dev")
	}
	// In prod, DevDID is intentionally ignored; clear it so callers don't
	// accidentally use a leftover value.
	if env == EnvProd {
		for _, origin := range cfg.AllowedOrigins {
			if origin == "*" {
				return Config{}, fmt.Errorf("ALLOWED_ORIGINS: wildcard origin is not allowed in prod")
			}
		}
		cfg.DevDID = ""
		cfg.EnableDevModeration = false
		cfg.DevModerationToken = ""
		cfg.DevLabelerDID = ""
		cfg.TrustedModerationSourceDIDs = nil
		cfg.UnsafeLogResponseBodies = false
		cfg.UnsafeLogInstagramWebhookBodies = false
	}
	if env == EnvDev && cfg.EnableDevModeration && strings.TrimSpace(cfg.DevModerationToken) == "" {
		return Config{}, fmt.Errorf("APPVIEW_DEV_MODERATION_TOKEN is required when APPVIEW_ENABLE_DEV_MODERATION=true")
	}

	if cfg.Env == EnvProd {
		if cfg.OAuth.ClientSecretKey == "" || cfg.OAuth.ClientSecretKey.Reveal() != strings.TrimSpace(cfg.OAuth.ClientSecretKey.Reveal()) {
			return Config{}, fmt.Errorf("OAUTH_CLIENT_SECRET_KEY is required in prod")
		}
		if cfg.OAuth.ClientKeyID == "" || cfg.OAuth.ClientKeyID != strings.TrimSpace(cfg.OAuth.ClientKeyID) || containsControl(cfg.OAuth.ClientKeyID) {
			return Config{}, fmt.Errorf("OAUTH_CLIENT_SECRET_KEY_ID is required in prod")
		}
	}
	if cfg.PushEnabled && cfg.FirebaseProjectID == "" {
		return Config{}, fmt.Errorf("FIREBASE_PROJECT_ID is required when PUSH_ENABLED=true")
	}
	if cfg.ScheduledPostsS3.Endpoint == "" || cfg.ScheduledPostsS3.Region == "" ||
		cfg.ScheduledPostsS3.Bucket == "" || cfg.ScheduledPostsS3.AccessKeyID == "" ||
		cfg.ScheduledPostsS3.SecretAccessKey == "" {
		return Config{}, fmt.Errorf("scheduled-post private object store configuration is incomplete")
	}
	objectStoreURL, parseErr := url.Parse(cfg.ScheduledPostsS3.Endpoint)
	if parseErr != nil || objectStoreURL.Host == "" {
		return Config{}, fmt.Errorf("SCHEDULED_POSTS_S3_ENDPOINT is invalid")
	}
	if env == EnvProd && objectStoreURL.Scheme != "https" {
		return Config{}, fmt.Errorf("SCHEDULED_POSTS_S3_ENDPOINT must use HTTPS in production")
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate applies cross-field invariants after both defaults and overrides
// have been parsed. New audit-remediation durations should join this single
// pass rather than validating only their individual syntax.
func (cfg Config) Validate() error {
	switch {
	case cfg.InstagramDeployment.ReplicaCount() > 1:
		return fmt.Errorf("APPVIEW_REPLICA_COUNT must be 1 until a shared HTTP admission boundary is configured and verified")
	case cfg.CraftskySessionInactivity > cfg.OAuthSessionAbsoluteLifetime:
		return fmt.Errorf("CRAFTSKY_SESSION_INACTIVITY must not exceed OAUTH_SESSION_ABSOLUTE_LIFETIME")
	case cfg.TapProjectionLeaseDuration <= cfg.TapProjectionPollInterval:
		return fmt.Errorf("TAP_PROJECTION_LEASE_DURATION must exceed TAP_PROJECTION_POLL_INTERVAL")
	case cfg.TapProjectionBackoffMax < cfg.TapProjectionBackoffMin:
		return fmt.Errorf("TAP_PROJECTION_BACKOFF_MAX must not be shorter than TAP_PROJECTION_BACKOFF_MIN")
	case cfg.TapRepositoryLeaseDuration <= cfg.TapRepositoryPollInterval:
		return fmt.Errorf("TAP_REPOSITORY_LEASE_DURATION must exceed TAP_REPOSITORY_POLL_INTERVAL")
	case cfg.TapRepositoryBackoffMax < cfg.TapRepositoryBackoffMin:
		return fmt.Errorf("TAP_REPOSITORY_BACKOFF_MAX must not be shorter than TAP_REPOSITORY_BACKOFF_MIN")
	case cfg.TapRepositoryLeaseDuration <= cfg.FederatedHTTP.PDSJSON.TotalTimeout+5*time.Second:
		return fmt.Errorf("TAP_REPOSITORY_LEASE_DURATION must exceed FEDERATED_PDS_JSON_TIMEOUT plus a 5s finalization margin")
	case cfg.TapQuarantineLeaseDuration <= cfg.TapQuarantinePollInterval:
		return fmt.Errorf("TAP_QUARANTINE_LEASE_DURATION must exceed TAP_QUARANTINE_POLL_INTERVAL")
	case cfg.TapQuarantineOperationTimeout >= cfg.TapQuarantineLeaseDuration:
		return fmt.Errorf("TAP_QUARANTINE_OPERATION_TIMEOUT must be shorter than TAP_QUARANTINE_LEASE_DURATION")
	case cfg.TapAckTimeout <= cfg.OwnerFenceAcquireTimeout+cfg.TapTerminalTransactionBudget+cfg.TapAckSafetyMargin:
		return fmt.Errorf("OWNER_FENCE_ACQUIRE_TIMEOUT plus TAP_TERMINAL_TRANSACTION_BUDGET plus TAP_ACK_SAFETY_MARGIN must be shorter than TAP_ACK_TIMEOUT")
	case cfg.OAuthAuthRequestExpiry >= cfg.OAuthSessionAbsoluteLifetime:
		return fmt.Errorf("OAUTH_AUTH_REQUEST_EXPIRY must be shorter than OAUTH_SESSION_ABSOLUTE_LIFETIME")
	case cfg.CraftskySessionActivityWriteInterval >= cfg.CraftskySessionInactivity:
		return fmt.Errorf("CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL must be shorter than CRAFTSKY_SESSION_INACTIVITY")
	case cfg.OAuthHandoffConfirmationTTL >= cfg.OAuthHandoffExchangeTTL:
		return fmt.Errorf("OAUTH_HANDOFF_CONFIRMATION_TTL must be shorter than OAUTH_HANDOFF_EXCHANGE_TTL")
	case cfg.PushConcurrency > cfg.PushBatchSize:
		return fmt.Errorf("PUSH_CONCURRENCY must not exceed PUSH_BATCH_SIZE")
	case cfg.HTTPMaxInFlightRequests > cfg.HTTPMaxConnections:
		return fmt.Errorf("HTTP_MAX_IN_FLIGHT_REQUESTS must not exceed HTTP_MAX_CONNECTIONS")
	case cfg.HTTPReadHeaderTimeout > cfg.HTTPReadTimeout:
		return fmt.Errorf("HTTP_READ_HEADER_TIMEOUT must not exceed HTTP_READ_TIMEOUT")
	case cfg.HTTPWriteTimeout <= cfg.HTTPReadTimeout:
		return fmt.Errorf("HTTP_WRITE_TIMEOUT must exceed HTTP_READ_TIMEOUT")
	case cfg.HTTPJSONBodyReadTimeout > cfg.HTTPReadTimeout:
		return fmt.Errorf("HTTP_JSON_BODY_READ_TIMEOUT must not exceed HTTP_READ_TIMEOUT")
	case cfg.HTTPUploadBodyReadTimeout > cfg.HTTPReadTimeout:
		return fmt.Errorf("HTTP_UPLOAD_BODY_READ_TIMEOUT must not exceed HTTP_READ_TIMEOUT")
	case cfg.HTTPWriteTimeout <= cfg.HTTPUploadBodyReadTimeout ||
		cfg.HTTPWriteTimeout-cfg.HTTPUploadBodyReadTimeout <= cfg.ScheduledMediaPutTimeout ||
		cfg.HTTPWriteTimeout-cfg.HTTPUploadBodyReadTimeout-cfg.ScheduledMediaPutTimeout <= httpWriteResponseSafetyMargin:
		// Compare by subtraction so a manually constructed Config cannot make a
		// duration sum overflow before validation rejects the unsafe geometry.
		return fmt.Errorf("HTTP_WRITE_TIMEOUT must exceed HTTP_UPLOAD_BODY_READ_TIMEOUT plus SCHEDULED_MEDIA_PUT_TIMEOUT plus a 5s response margin")
	case cfg.HTTPOuterClientLimit > cfg.HTTPOuterGlobalLimit:
		return fmt.Errorf("HTTP_OUTER_CLIENT_LIMIT must not exceed HTTP_OUTER_GLOBAL_LIMIT")
	case cfg.OAuthAuthRequestSweepBatch > cfg.OAuthPendingAuthRequestCapacity:
		return fmt.Errorf("OAUTH_AUTH_REQUEST_SWEEP_BATCH must not exceed OAUTH_PENDING_AUTH_REQUEST_CAPACITY")
	case cfg.OAuthAuthRequestSweepInterval > cfg.OAuthAuthRequestTerminalRetention:
		return fmt.Errorf("OAUTH_AUTH_REQUEST_SWEEP_INTERVAL must not exceed OAUTH_AUTH_REQUEST_TERMINAL_RETENTION")
	case cfg.OAuthAuthRequestTerminalRetention <= cfg.OAuthAuthRequestExpiry:
		return fmt.Errorf("OAUTH_AUTH_REQUEST_TERMINAL_RETENTION must exceed OAUTH_AUTH_REQUEST_EXPIRY")
	case cfg.OAuthRevocationOperationTimeout >= cfg.OAuthRevocationLeaseDuration:
		return fmt.Errorf("OAUTH_REVOCATION_OPERATION_TIMEOUT must be less than OAUTH_REVOCATION_LEASE_DURATION")
	case cfg.OAuthRevocationPollInterval > cfg.OAuthRevocationLeaseDuration:
		return fmt.Errorf("OAUTH_REVOCATION_POLL_INTERVAL must not exceed OAUTH_REVOCATION_LEASE_DURATION")
	case cfg.OAuthRevocationBackoffMax < cfg.OAuthRevocationBackoffMin:
		return fmt.Errorf("OAUTH_REVOCATION_BACKOFF_MAX must not be less than OAUTH_REVOCATION_BACKOFF_MIN")
	case cfg.OAuthRevocationMaxCredentialRetention > cfg.OAuthSessionAbsoluteLifetime:
		return fmt.Errorf("OAUTH_REVOCATION_MAX_CREDENTIAL_RETENTION must not exceed OAUTH_SESSION_ABSOLUTE_LIFETIME")
	case cfg.AuthAuxiliaryCleanupOperationTimeout >= cfg.AuthAuxiliaryCleanupLeaseDuration:
		return fmt.Errorf("AUTH_AUXILIARY_CLEANUP_OPERATION_TIMEOUT must be less than AUTH_AUXILIARY_CLEANUP_LEASE_DURATION")
	case cfg.AuthAuxiliaryCleanupPollInterval > cfg.AuthAuxiliaryCleanupLeaseDuration:
		return fmt.Errorf("AUTH_AUXILIARY_CLEANUP_POLL_INTERVAL must not exceed AUTH_AUXILIARY_CLEANUP_LEASE_DURATION")
	case cfg.AuthAuxiliaryCleanupBackoffMax < cfg.AuthAuxiliaryCleanupBackoffMin:
		return fmt.Errorf("AUTH_AUXILIARY_CLEANUP_BACKOFF_MAX must not be less than AUTH_AUXILIARY_CLEANUP_BACKOFF_MIN")
	case cfg.SessionExpirySweepInterval > cfg.CraftskySessionInactivity:
		return fmt.Errorf("AUTH_SESSION_EXPIRY_SWEEP_INTERVAL must not exceed CRAFTSKY_SESSION_INACTIVITY")
	case cfg.AccountDeletionIntentSweepInterval > cfg.AccountDeletionIntentTTL:
		return fmt.Errorf("ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL must not exceed ACCOUNT_DELETION_INTENT_TTL")
	case cfg.TerminalPurgeLeaseDuration <= cfg.TerminalPurgePollInterval:
		return fmt.Errorf("TERMINAL_PURGE_LEASE_DURATION must exceed TERMINAL_PURGE_POLL_INTERVAL")
	case cfg.PushSendTimeout >= cfg.PushLeaseDuration ||
		cfg.PushFinalizationMargin >= cfg.PushLeaseDuration-cfg.PushSendTimeout:
		return fmt.Errorf("PUSH_SEND_TIMEOUT plus PUSH_FINALIZATION_MARGIN must be shorter than PUSH_LEASE_DURATION")
	case cfg.ImageDecodeLimits.MaxPixels > uint64(cfg.ImageDecodeLimits.MaxWidth)*uint64(cfg.ImageDecodeLimits.MaxHeight):
		return fmt.Errorf("SCHEDULED_IMAGE_MAX_PIXELS must not exceed SCHEDULED_IMAGE_MAX_WIDTH multiplied by SCHEDULED_IMAGE_MAX_HEIGHT")
	default:
		if err := cfg.ImageDecodeLimits.Validate(); err != nil {
			return fmt.Errorf("scheduled image decode limits: %w", err)
		}
		return nil
	}
}

func (cfg Config) tapIngestionTimeout() time.Duration {
	return cfg.TapAckTimeout - cfg.TapAckSafetyMargin
}

func (cfg Config) tapTerminalCommitTimeout() time.Duration {
	return cfg.OwnerFenceAcquireTimeout + cfg.TapTerminalTransactionBudget
}

func decodeHandoffReceiptKey(secret Secret) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(secret.Reveal())
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("OAUTH_HANDOFF_RECEIPT_KEY must be an unpadded base64url 256-bit key")
	}
	return key, nil
}

func federatedHTTPConfigFromEnv() (FederatedHTTPConfig, error) {
	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		return FederatedHTTPConfig{}, err
	}

	durationSettings := []struct {
		key    string
		value  *time.Duration
		secure time.Duration
	}{
		{key: "FEDERATED_HTTP_DIAL_TIMEOUT", value: &config.Transport.DialTimeout, secure: config.Transport.DialTimeout},
		{key: "FEDERATED_HTTP_TLS_HANDSHAKE_TIMEOUT", value: &config.Transport.TLSHandshakeTimeout, secure: config.Transport.TLSHandshakeTimeout},
		{key: "FEDERATED_HTTP_RESPONSE_HEADER_TIMEOUT", value: &config.Transport.ResponseHeaderTimeout, secure: config.Transport.ResponseHeaderTimeout},
		{key: "FEDERATED_HTTP_EXPECT_CONTINUE_TIMEOUT", value: &config.Transport.ExpectContinueTimeout, secure: config.Transport.ExpectContinueTimeout},
		{key: "FEDERATED_HTTP_IDLE_CONN_TIMEOUT", value: &config.Transport.IdleConnTimeout, secure: config.Transport.IdleConnTimeout},
		{key: "FEDERATED_OAUTH_METADATA_TIMEOUT", value: &config.OAuthMetadata.TotalTimeout, secure: config.OAuthMetadata.TotalTimeout},
		{key: "FEDERATED_OAUTH_REQUEST_TIMEOUT", value: &config.OAuthRequest.TotalTimeout, secure: config.OAuthRequest.TotalTimeout},
		{key: "FEDERATED_PDS_JSON_TIMEOUT", value: &config.PDSJSON.TotalTimeout, secure: config.PDSJSON.TotalTimeout},
		{key: "FEDERATED_PDS_UPLOAD_TIMEOUT", value: &config.PDSUpload.TotalTimeout, secure: config.PDSUpload.TotalTimeout},
	}
	for _, setting := range durationSettings {
		parsed, err := boundedPositiveDurationEnv(setting.key, setting.secure, setting.secure)
		if err != nil {
			return FederatedHTTPConfig{}, err
		}
		*setting.value = parsed
	}

	limitSettings := []struct {
		key    string
		value  *int64
		secure int64
	}{
		{key: "FEDERATED_HTTP_MAX_RESPONSE_HEADER_BYTES", value: &config.Transport.MaxResponseHeaderBytes, secure: config.Transport.MaxResponseHeaderBytes},
		{key: "FEDERATED_OAUTH_METADATA_RESPONSE_LIMIT_BYTES", value: &config.OAuthMetadata.ResponseLimit, secure: config.OAuthMetadata.ResponseLimit},
		{key: "FEDERATED_OAUTH_RESPONSE_LIMIT_BYTES", value: &config.OAuthRequest.ResponseLimit, secure: config.OAuthRequest.ResponseLimit},
		{key: "FEDERATED_PDS_JSON_RESPONSE_LIMIT_BYTES", value: &config.PDSJSON.ResponseLimit, secure: config.PDSJSON.ResponseLimit},
		{key: "FEDERATED_PDS_UPLOAD_RESPONSE_LIMIT_BYTES", value: &config.PDSUpload.ResponseLimit, secure: config.PDSUpload.ResponseLimit},
	}
	for _, setting := range limitSettings {
		parsed, err := boundedInt64Env(setting.key, setting.secure, 1, setting.secure)
		if err != nil {
			return FederatedHTTPConfig{}, err
		}
		*setting.value = parsed
	}
	return config, nil
}

func defaultFederatedHTTPConfig() (FederatedHTTPConfig, error) {
	metadata, err := federatedhttp.DefaultProfile(federatedhttp.PurposeOAuthMetadata)
	if err != nil {
		return FederatedHTTPConfig{}, err
	}
	oauthRequest, err := federatedhttp.DefaultProfile(federatedhttp.PurposeOAuthRequest)
	if err != nil {
		return FederatedHTTPConfig{}, err
	}
	pdsJSON, err := federatedhttp.DefaultProfile(federatedhttp.PurposePDSJSON)
	if err != nil {
		return FederatedHTTPConfig{}, err
	}
	pdsUpload, err := federatedhttp.DefaultProfile(federatedhttp.PurposePDSUpload)
	if err != nil {
		return FederatedHTTPConfig{}, err
	}
	config := FederatedHTTPConfig{
		Transport:     federatedhttp.DefaultTransportProfile(),
		OAuthMetadata: metadata,
		OAuthRequest:  oauthRequest,
		PDSJSON:       pdsJSON,
		PDSUpload:     pdsUpload,
	}
	return config, nil
}

func parseCanonicalPublicOrigin(raw string) (url.URL, error) {
	if raw == "" {
		return url.URL{}, fmt.Errorf("OAUTH_PUBLIC_ORIGIN is required in prod")
	}
	if raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\\#\r\n\t") {
		return url.URL{}, fmt.Errorf("OAUTH_PUBLIC_ORIGIN must be a canonical HTTPS origin")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return url.URL{}, fmt.Errorf("OAUTH_PUBLIC_ORIGIN: %w", err)
	}
	hostname := strings.ToLower(parsed.Hostname())
	if parsed.Scheme != "https" || parsed.Opaque != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" ||
		parsed.Port() != "" || hostname == "" || net.ParseIP(hostname) != nil ||
		!isPublicDNSName(hostname) {
		return url.URL{}, fmt.Errorf("OAUTH_PUBLIC_ORIGIN must be a public canonical HTTPS origin")
	}
	parsed.Scheme = "https"
	parsed.Host = hostname
	parsed.Path = ""
	return *parsed, nil
}

func resolveOriginPath(origin url.URL, path string) url.URL {
	reference := &url.URL{Path: path}
	return *origin.ResolveReference(reference)
}

func parseLoopbackOAuthCallback(raw string) (url.URL, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\\?#\r\n\t") {
		return url.URL{}, fmt.Errorf("OAUTH_CALLBACK_URL must be an exact loopback callback URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return url.URL{}, fmt.Errorf("OAUTH_CALLBACK_URL: %w", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 || parsed.Scheme != "http" ||
		parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() != "127.0.0.1" ||
		parsed.Path != "/oauth/callback" || parsed.RawPath != "" || parsed.RawQuery != "" ||
		parsed.ForceQuery || parsed.Fragment != "" {
		return url.URL{}, fmt.Errorf("OAUTH_CALLBACK_URL must match http://127.0.0.1:<port>/oauth/callback")
	}
	return *parsed, nil
}

func parseOAuthScopes(raw string) ([]string, error) {
	for _, char := range raw {
		if (char < 0x20 && char != ' ') || char == 0x7f {
			return nil, fmt.Errorf("OAUTH_SCOPES contains an invalid control character")
		}
	}
	scopes := strings.Fields(raw)
	if len(scopes) == 0 {
		return nil, fmt.Errorf("OAUTH_SCOPES must include atproto")
	}
	seen := make(map[string]struct{}, len(scopes))
	hasATProto := false
	for _, scope := range scopes {
		for _, char := range scope {
			if char < 0x21 || char > 0x7e {
				return nil, fmt.Errorf("OAUTH_SCOPES contains an invalid scope")
			}
		}
		if _, exists := seen[scope]; exists {
			return nil, fmt.Errorf("OAUTH_SCOPES contains duplicate scope %q", scope)
		}
		seen[scope] = struct{}{}
		hasATProto = hasATProto || scope == "atproto"
	}
	if !hasATProto {
		return nil, fmt.Errorf("OAUTH_SCOPES must include atproto")
	}
	return scopes, nil
}

func containsControl(value string) bool {
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return true
		}
	}
	return false
}

func isHeaderSafeSecret(value string) bool {
	if len([]byte(value)) < 32 {
		return false
	}
	distinct := make(map[rune]struct{})
	for _, char := range value {
		if char < 0x21 || char > 0x7e {
			return false
		}
		distinct[char] = struct{}{}
	}
	return len(distinct) >= 8
}

func normalizeLocalExpectedHosts(hosts []string) error {
	seen := make(map[string]struct{}, len(hosts))
	for i, host := range hosts {
		normalized := strings.ToLower(host)
		if normalized != "localhost" && normalized != "127.0.0.1" {
			return fmt.Errorf("APPVIEW_EXPECTED_HOSTS must contain only loopback hosts in local development")
		}
		if _, duplicate := seen[normalized]; duplicate {
			return fmt.Errorf("APPVIEW_EXPECTED_HOSTS contains duplicate host %q", normalized)
		}
		seen[normalized] = struct{}{}
		hosts[i] = normalized
	}
	return nil
}

func normalizeRemoteExpectedHosts(hosts []string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("APPVIEW_EXPECTED_HOSTS must contain at least one public DNS host in remote development")
	}
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		normalized := strings.ToLower(host)
		if host != strings.TrimSpace(host) || !isPublicDNSName(normalized) {
			return fmt.Errorf("APPVIEW_EXPECTED_HOSTS must contain public DNS hosts without ports in remote development")
		}
		if _, duplicate := seen[normalized]; duplicate {
			return fmt.Errorf("APPVIEW_EXPECTED_HOSTS contains duplicate host %q", normalized)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func expectedHostsEnv() ([]string, error) {
	raw := os.Getenv("APPVIEW_EXPECTED_HOSTS")
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		host := strings.TrimSpace(part)
		if host == "" {
			return nil, fmt.Errorf("APPVIEW_EXPECTED_HOSTS contains an empty host")
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}

func trustedProxyCIDRsEnv(key string) ([]string, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, nil
	}
	seen := make(map[netip.Prefix]struct{})
	prefixes := make([]string, 0, strings.Count(raw, ",")+1)
	for _, part := range strings.Split(raw, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(part))
		if err != nil || prefix.Bits() == 0 {
			return nil, fmt.Errorf("%s must contain valid CIDR prefixes narrower than /0", key)
		}
		prefix = prefix.Masked()
		if _, duplicate := seen[prefix]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate prefix %q", key, prefix)
		}
		seen[prefix] = struct{}{}
		prefixes = append(prefixes, prefix.String())
	}
	return prefixes, nil
}

func isPublicDNSName(hostname string) bool {
	if hostname != strings.TrimSuffix(hostname, ".") || !strings.Contains(hostname, ".") {
		return false
	}
	for _, suffix := range []string{
		"localhost", ".localhost", ".local", ".internal", ".home.arpa",
		".test", ".invalid", ".example", ".onion", ".alt", ".arpa",
	} {
		if hostname == suffix || strings.HasSuffix(hostname, suffix) {
			return false
		}
	}
	for _, reserved := range []string{"example.com", "example.net", "example.org"} {
		if hostname == reserved || strings.HasSuffix(hostname, "."+reserved) {
			return false
		}
	}
	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
				return false
			}
		}
	}
	return len(hostname) <= 253
}

func DefaultRateLimitConfig() middleware.RateLimitConfig {
	return middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
		middleware.RateClassAuth:        {Window: time.Minute, PerDevice: 10},
		middleware.RateClassRead:        {Window: time.Minute, PerToken: 300, PerDevice: 600},
		middleware.RateClassWrite:       {Window: time.Minute, PerToken: 60, PerDevice: 120},
		middleware.RateClassSearch:      {Window: time.Minute, PerToken: 60, PerDevice: 120},
		middleware.RateClassUpload:      {Window: time.Hour, PerToken: 100, PerDevice: 200},
		middleware.RateClassLinkPreview: {Window: time.Hour, PerToken: 60, PerDevice: 120},
	}}
}

// getEnvWithDefault returns os.Getenv(key), or def if empty.
func getEnvWithDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// durationEnv parses os.Getenv(key) as a time.Duration, or returns def if empty.
// Returns a wrapped error mentioning key if parsing fails.
func durationEnv(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func boundedPositiveDurationEnv(key string, def, max time.Duration) (time.Duration, error) {
	value, err := durationEnv(key, def)
	if err != nil {
		return 0, err
	}
	if value <= 0 || value > max {
		return 0, fmt.Errorf("%s: must be greater than zero and at most %s", key, max)
	}
	return value, nil
}

func boundedIntEnv(key string, def, min, max int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s: must be integer between %d and %d, got %q", key, min, max, raw)
	}
	return n, nil
}

func boundedInt64Env(key string, def, min, max int64) (int64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s: must be integer between %d and %d, got %q", key, min, max, raw)
	}
	return n, nil
}

func boundedUint64Env(key string, def, min, max uint64) (uint64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s: must be integer between %d and %d, got %q", key, min, max, raw)
	}
	return n, nil
}

func sampleRateEnv(key string) (float64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n < 0 || n > 1 {
		return 0, fmt.Errorf("%s: must be number between 0 and 1, got %q", key, raw)
	}
	return n, nil
}

func boolEnv(key string, def bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: must be boolean, got %q", key, raw)
	}
	return parsed, nil
}

func splitCommaEnv(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	out := []string{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func appendUniqueNonEmpty(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
