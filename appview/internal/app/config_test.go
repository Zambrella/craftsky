package app

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

const productionOAuthTestEnv = "OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n" +
	"OAUTH_CLIENT_SECRET_KEY=test-private-key\n" +
	"OAUTH_CLIENT_SECRET_KEY_ID=primary\n"

func withProductionOAuth(contents string) string {
	return contents + productionOAuthTestEnv
}

func TestConfigurationSecretsRedactFromFormatting(t *testing.T) {
	const raw = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	secret := Secret(raw)
	diagnostic := fmt.Sprintf("%v %+v %#v cfg=%+v", secret, secret, secret, Config{
		DevAuthSecret:          secret,
		OAuthHandoffReceiptKey: secret,
		OAuth:                  OAuthDeployment{ClientSecretKey: secret},
	})
	if strings.Contains(diagnostic, raw) || !strings.Contains(diagnostic, "[REDACTED]") {
		t.Fatal("secret diagnostic was not redacted")
	}
	if secret.Reveal() != raw {
		t.Fatal("Reveal did not return the configured secret")
	}
}

func TestLoadConfigRejectsMalformedHandoffReceiptKey(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	_, err := LoadConfig(EnvDev, testConfigFile(t, base+"OAUTH_HANDOFF_RECEIPT_KEY=not-a-256-bit-key\n"))
	if err == nil || !strings.Contains(err.Error(), "OAUTH_HANDOFF_RECEIPT_KEY") {
		t.Fatalf("LoadConfig malformed handoff key error = %v", err)
	}
}

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Env
		wantErr bool
	}{
		{"dev", "dev", EnvDev, false},
		{"prod", "prod", EnvProd, false},
		{"empty", "", "", true},
		{"unknown", "staging", "", true},
		{"caps", "DEV", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnv(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseEnv(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// UT-016: development enables previews by default while production requires
// an explicit true value; malformed booleans fail configuration loading.
func TestLoadConfigLinkPreviewEnablement(t *testing.T) {
	const devBase = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	const prodBase = "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n"
	tests := []struct {
		name    string
		env     Env
		value   string
		want    bool
		wantErr bool
	}{
		{name: "development default", env: EnvDev, want: true},
		{name: "development explicit false", env: EnvDev, value: "false", want: false},
		{name: "production default", env: EnvProd, want: false},
		{name: "production explicit true", env: EnvProd, value: "true", want: true},
		{name: "production explicit false", env: EnvProd, value: "false", want: false},
		{name: "invalid", env: EnvDev, value: "yes", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := devBase
			if tt.env == EnvProd {
				contents = withProductionOAuth(prodBase)
			}
			if tt.value != "" {
				contents += "LINK_PREVIEWS_ENABLED=" + tt.value + "\n"
			}
			cfg, err := LoadConfig(tt.env, testConfigFile(t, contents))
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && cfg.LinkPreviewsEnabled != tt.want {
				t.Fatalf("LinkPreviewsEnabled = %t, want %t", cfg.LinkPreviewsEnabled, tt.want)
			}
		})
	}
}

// UT-017: link previews have independent hourly token and device budgets with
// exact fixed-window rollover.
func TestDefaultRateLimitConfigLinkPreview(t *testing.T) {
	now := time.Unix(1_000, 0)
	limiter := middleware.NewLocalRateLimiter(DefaultRateLimitConfig(), func() time.Time { return now })
	for request := 1; request <= 61; request++ {
		decision := limiter.Allow(middleware.RateClassLinkPreview, middleware.RateKeys{TokenKey: "token-a", DeviceID: "device-" + fmt.Sprint(request)})
		if decision.Allowed != (request <= 60) {
			t.Fatalf("token request %d allowed = %t, want %t", request, decision.Allowed, request <= 60)
		}
		if request == 61 && decision.KeyType != "token" {
			t.Fatalf("token overflow key type = %q, want token", decision.KeyType)
		}
	}

	now = now.Add(time.Hour)
	for request := 1; request <= 121; request++ {
		decision := limiter.Allow(middleware.RateClassLinkPreview, middleware.RateKeys{TokenKey: "token-" + fmt.Sprint(request), DeviceID: "shared-device"})
		if decision.Allowed != (request <= 120) {
			t.Fatalf("device request %d allowed = %t, want %t", request, decision.Allowed, request <= 120)
		}
		if request == 121 && decision.KeyType != "device" {
			t.Fatalf("device overflow key type = %q, want device", decision.KeyType)
		}
	}

	now = now.Add(time.Hour)
	if decision := limiter.Allow(middleware.RateClassLinkPreview, middleware.RateKeys{TokenKey: "token-a", DeviceID: "shared-device"}); !decision.Allowed {
		t.Fatalf("first request after rollover rejected: %+v", decision)
	}
}

// REG-007: preview, write, and upload counters remain independent even when
// they use the same token and device keys in one limiter.
func TestDefaultRateLimitsKeepLinkPreviewQuotaIndependent(t *testing.T) {
	limiter := middleware.NewLocalRateLimiter(DefaultRateLimitConfig(), func() time.Time { return time.Unix(1_000, 0) })
	keys := middleware.RateKeys{TokenKey: "shared-token", DeviceID: "shared-device"}
	for request := 0; request < 60; request++ {
		if decision := limiter.Allow(middleware.RateClassLinkPreview, keys); !decision.Allowed {
			t.Fatalf("preview request %d rejected: %+v", request+1, decision)
		}
	}
	if decision := limiter.Allow(middleware.RateClassLinkPreview, keys); decision.Allowed {
		t.Fatal("61st preview request allowed")
	}
	if decision := limiter.Allow(middleware.RateClassWrite, keys); !decision.Allowed {
		t.Fatalf("first write consumed by preview traffic: %+v", decision)
	}
	if decision := limiter.Allow(middleware.RateClassUpload, keys); !decision.Allowed {
		t.Fatalf("first upload consumed by preview traffic: %+v", decision)
	}

	otherKeys := middleware.RateKeys{TokenKey: "other-token", DeviceID: "other-device"}
	for request := 1; request < 60; request++ {
		if decision := limiter.Allow(middleware.RateClassWrite, otherKeys); !decision.Allowed {
			t.Fatalf("write request %d rejected: %+v", request, decision)
		}
	}
	if decision := limiter.Allow(middleware.RateClassLinkPreview, otherKeys); !decision.Allowed {
		t.Fatalf("preview consumed by write traffic: %+v", decision)
	}
}

// testConfigFile writes a temporary .env-style file and returns its path.
// It also unsets the relevant env vars before the test so godotenv.Load
// will actually pick up the file's values. Setting them to "" instead of
// unsetting would leave them "present" from godotenv's perspective, which
// would cause Load to skip the file's value — the opposite of what we want.
func testConfigFile(t *testing.T, contents string) string {
	t.Helper()
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID",
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_ACK_SAFETY_MARGIN", "TAP_TERMINAL_TRANSACTION_BUDGET", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES",
		"TAP_PROJECTION_POLL_INTERVAL", "TAP_PROJECTION_LEASE_DURATION", "TAP_PROJECTION_BATCH_SIZE",
		"TAP_PROJECTION_BACKOFF_MIN", "TAP_PROJECTION_BACKOFF_MAX",
		"TAP_REPOSITORY_POLL_INTERVAL", "TAP_REPOSITORY_LEASE_DURATION", "TAP_REPOSITORY_BATCH_SIZE",
		"TAP_REPOSITORY_BACKOFF_MIN", "TAP_REPOSITORY_BACKOFF_MAX",
		"TAP_QUARANTINE_POLL_INTERVAL", "TAP_QUARANTINE_LEASE_DURATION",
		"TAP_QUARANTINE_OPERATION_TIMEOUT", "TAP_QUARANTINE_BATCH_SIZE",
		"OAUTH_HOSTNAME", "OAUTH_PUBLIC_ORIGIN", "OAUTH_REGISTRATION_PROVIDER_ORIGIN", "OAUTH_CALLBACK_URL", "OAUTH_CLIENT_SECRET_KEY", "OAUTH_CLIENT_SECRET_KEY_ID",
		"OAUTH_SCOPES", "OAUTH_SESSION_EXPIRY", "OAUTH_SESSION_ABSOLUTE_LIFETIME",
		"OAUTH_SESSION_INACTIVITY", "CRAFTSKY_SESSION_INACTIVITY",
		"OAUTH_AUTH_REQUEST_EXPIRY", "CRAFTSKY_SESSION_LAST_SEEN_THROTTLE",
		"CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL", "OAUTH_LOGIN_START_TIMEOUT",
		"OAUTH_CALLBACK_OPERATION_TIMEOUT", "OAUTH_SESSION_OPERATION_TIMEOUT", "OAUTH_HANDOFF_EXCHANGE_TTL",
		"OAUTH_HANDOFF_CONFIRMATION_TTL", "OAUTH_HANDOFF_RECEIPT_KEY",
		"OAUTH_HANDOFF_RECEIPT_KEY_VERSION", "APP_VERIFIED_LINK_ORIGIN", "APPVIEW_ENABLE_DEV_OAUTH_SCHEME",
		"APPVIEW_PUBLISH_HOST", "APPVIEW_EXPECTED_HOSTS", "APPVIEW_DEV_REMOTE_ACCESS",
		"APPVIEW_DEV_PROTECTED_TRANSPORT", "APPVIEW_DEV_AUTH_SECRET",
		"MAX_POST_IMAGES", "MAX_IMAGE_UPLOAD_BYTES", "APPVIEW_JSON_BODY_LIMIT_BYTES",
		"SCHEDULED_IMAGE_MAX_WIDTH", "SCHEDULED_IMAGE_MAX_HEIGHT", "SCHEDULED_IMAGE_MAX_PIXELS",
		"SCHEDULED_IMAGE_MAX_ASPECT_RATIO", "SCHEDULED_IMAGE_MAX_CONCURRENT_DECODES",
		"SCHEDULED_IMAGE_ADMISSION_WAIT",
		"SCHEDULED_POSTS_S3_ENDPOINT", "SCHEDULED_POSTS_S3_REGION", "SCHEDULED_POSTS_S3_BUCKET",
		"SCHEDULED_POSTS_S3_ACCESS_KEY_ID", "SCHEDULED_POSTS_S3_SECRET_ACCESS_KEY",
		"SENTRY_DSN", "SENTRY_RELEASE", "SENTRY_TRACING_ENABLED", "SENTRY_TRACES_SAMPLE_RATE",
		"SENTRY_LOGS_ENABLED", "SENTRY_METRICS_ENABLED", "SENTRY_TAP_TRACING_ENABLED",
		"SENTRY_TAP_TRACES_SAMPLE_RATE",
		"APPVIEW_UNSAFE_LOG_RESPONSE_BODIES",
		"INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES",
		"APPVIEW_ENABLE_DEV_MODERATION",
		"APPVIEW_DEV_MODERATION_TOKEN", "CRAFTSKY_DEV_LABELER_DID",
		"APPVIEW_TRUSTED_MODERATION_SOURCE_DIDS",
		"PUSH_ENABLED", "FIREBASE_PROJECT_ID", "PUSH_BATCH_SIZE", "PUSH_CONCURRENCY", "PUSH_POLL_INTERVAL", "PUSH_LEASE_DURATION", "PUSH_SEND_TIMEOUT", "PUSH_FINALIZATION_MARGIN",
		"OWNER_FENCE_ACQUIRE_TIMEOUT", "PDS_EFFECT_TIMEOUT", "SCHEDULED_MEDIA_PUT_TIMEOUT",
		"HTTP_MAX_CONNECTIONS", "HTTP_MAX_IN_FLIGHT_REQUESTS", "HTTP_READ_HEADER_TIMEOUT",
		"HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT", "HTTP_MAX_HEADER_BYTES",
		"HTTP_TRUSTED_PROXY_CIDRS", "HTTP_CLIENT_IPV6_PREFIX_BITS", "HTTP_OUTER_RATE_WINDOW",
		"HTTP_OUTER_GLOBAL_LIMIT", "HTTP_OUTER_CLIENT_LIMIT", "HTTP_LIMITER_CAPACITY",
		"HTTP_LIMITER_IDLE_TTL", "HTTP_JSON_BODY_READ_TIMEOUT", "HTTP_UPLOAD_BODY_READ_TIMEOUT",
		"OAUTH_PENDING_AUTH_REQUEST_CAPACITY", "OAUTH_AUTH_REQUEST_TERMINAL_RETENTION",
		"OAUTH_AUTH_REQUEST_SWEEP_INTERVAL", "OAUTH_AUTH_REQUEST_SWEEP_BATCH",
		"OAUTH_REVOCATION_POLL_INTERVAL", "OAUTH_REVOCATION_BATCH_SIZE",
		"OAUTH_REVOCATION_LEASE_DURATION", "OAUTH_REVOCATION_OPERATION_TIMEOUT",
		"OAUTH_REVOCATION_MAX_ATTEMPTS", "OAUTH_REVOCATION_BACKOFF_MIN",
		"OAUTH_REVOCATION_BACKOFF_MAX", "OAUTH_REVOCATION_MAX_CREDENTIAL_RETENTION",
		"AUTH_AUXILIARY_CLEANUP_POLL_INTERVAL", "AUTH_AUXILIARY_CLEANUP_BATCH_SIZE",
		"AUTH_AUXILIARY_CLEANUP_LEASE_DURATION", "AUTH_AUXILIARY_CLEANUP_OPERATION_TIMEOUT",
		"AUTH_AUXILIARY_CLEANUP_MAX_ATTEMPTS", "AUTH_AUXILIARY_CLEANUP_BACKOFF_MIN",
		"AUTH_AUXILIARY_CLEANUP_BACKOFF_MAX", "AUTH_SESSION_EXPIRY_SWEEP_INTERVAL",
		"AUTH_SESSION_EXPIRY_SWEEP_BATCH", "ACCOUNT_DELETION_INTENT_TTL",
		"ACCOUNT_DELETION_INTENT_SWEEP_INTERVAL", "ACCOUNT_DELETION_INTENT_SWEEP_BATCH",
		"TERMINAL_PURGE_POLL_INTERVAL", "TERMINAL_PURGE_COMPONENT_LIMIT",
		"TERMINAL_PURGE_ROW_BATCH_SIZE", "TERMINAL_PURGE_LEASE_DURATION",
		"TERMINAL_PURGE_RETRY_DELAY",
		"IDENTITY_CACHE_REFRESH_POLL_INTERVAL", "IDENTITY_CACHE_REFRESH_BATCH_SIZE",
		"IDENTITY_CACHE_REFRESH_OPERATION_TIMEOUT", "IDENTITY_CACHE_REFRESH_RETRY_DELAY",
		"FEDERATED_HTTP_DIAL_TIMEOUT", "FEDERATED_HTTP_TLS_HANDSHAKE_TIMEOUT",
		"FEDERATED_HTTP_RESPONSE_HEADER_TIMEOUT", "FEDERATED_HTTP_EXPECT_CONTINUE_TIMEOUT",
		"FEDERATED_HTTP_IDLE_CONN_TIMEOUT", "FEDERATED_HTTP_MAX_RESPONSE_HEADER_BYTES",
		"FEDERATED_OAUTH_METADATA_TIMEOUT", "FEDERATED_OAUTH_REQUEST_TIMEOUT",
		"FEDERATED_PDS_JSON_TIMEOUT", "FEDERATED_PDS_UPLOAD_TIMEOUT",
		"FEDERATED_OAUTH_METADATA_RESPONSE_LIMIT_BYTES", "FEDERATED_OAUTH_RESPONSE_LIMIT_BYTES",
		"FEDERATED_PDS_JSON_RESPONSE_LIMIT_BYTES", "FEDERATED_PDS_UPLOAD_RESPONSE_LIMIT_BYTES",
		"INSTAGRAM_DATA_HMAC_KEY", "INSTAGRAM_META_ENABLED",
		"INSTAGRAM_META_APP_SECRET", "INSTAGRAM_META_VERIFY_TOKEN",
		"INSTAGRAM_META_ACCESS_TOKEN", "INSTAGRAM_META_ACCOUNT_ID",
		"INSTAGRAM_META_API_VERSION", "INSTAGRAM_META_API_BASE_URL",
		"INSTAGRAM_META_DM_URL", "INSTAGRAM_META_REPLIES_ENABLED",
		"APPVIEW_REPLICA_COUNT", "INSTAGRAM_SHARED_RATE_LIMITS",
		"INSTAGRAM_TRUSTED_PROXY_CIDRS", "INSTAGRAM_CHALLENGE_TTL",
		"INSTAGRAM_WEBHOOK_BODY_LIMIT_BYTES", "INSTAGRAM_WEBHOOK_MAX_EVENTS",
		"INSTAGRAM_WEBHOOK_GLOBAL_PER_MINUTE", "INSTAGRAM_WEBHOOK_IP_PER_MINUTE",
		"INSTAGRAM_CHALLENGE_DID_PER_15_MINUTES", "INSTAGRAM_CHALLENGE_DEVICE_PER_15_MINUTES",
		"INSTAGRAM_CHALLENGE_IP_PER_15_MINUTES", "INSTAGRAM_INVALID_IGSID_PER_15_MINUTES",
		"INSTAGRAM_INVALID_IP_PER_15_MINUTES", "INSTAGRAM_CONFIRMATION_DID_PER_HOUR",
		"INSTAGRAM_CONFIRMATION_DEVICE_PER_HOUR", "INSTAGRAM_IMPORTS_DID_PER_HOUR",
		"INSTAGRAM_IMPORTS_DEVICE_PER_HOUR", "INSTAGRAM_IMPORT_MAX_ENTRIES",
		"INSTAGRAM_PAGE_DEFAULT", "INSTAGRAM_PAGE_MAX", "INSTAGRAM_META_HTTP_TIMEOUT",
		"INSTAGRAM_META_RESPONSE_LIMIT_BYTES", "INSTAGRAM_META_LOOKUP_CONCURRENCY",
		"INSTAGRAM_META_LOOKUPS_PER_IGSID_HOUR", "INSTAGRAM_WORKER_CONCURRENCY",
		"INSTAGRAM_WORKER_LEASE_DURATION", "INSTAGRAM_WORKER_MAX_ATTEMPTS",
		"INSTAGRAM_WORKER_BACKOFF_INITIAL", "INSTAGRAM_WORKER_BACKOFF_MAX",
		"INSTAGRAM_WORKER_MAX_PROCESSING_AGE", "INSTAGRAM_DM_REPLY_WINDOW",
		"INSTAGRAM_OPERATOR_BATCH_MAX", "LINK_PREVIEWS_ENABLED"} {
		// Snapshot for restoration, then unset.
		prior, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, prior)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
	f, err := os.CreateTemp(t.TempDir(), "test-*.env")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	return f.Name()
}

func TestLoadConfig_ScheduledPostObjectStoreIsCompleteAndProductionUsesTLS(t *testing.T) {
	dev := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n")
	cfg, err := LoadConfig(EnvDev, dev)
	if err != nil {
		t.Fatalf("LoadConfig dev defaults: %v", err)
	}
	if cfg.ScheduledPostsS3.Endpoint != "http://minio:9000" || cfg.ScheduledPostsS3.Bucket != "private-scheduled-media" {
		t.Fatalf("scheduled object store defaults = %+v", cfg.ScheduledPostsS3)
	}

	prodBase := withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n")
	_, err = LoadConfig(EnvProd, testConfigFile(t, prodBase+"SCHEDULED_POSTS_S3_ENDPOINT=http://objects.example\nSCHEDULED_POSTS_S3_REGION=eu-west-2\nSCHEDULED_POSTS_S3_BUCKET=private\nSCHEDULED_POSTS_S3_ACCESS_KEY_ID=key\nSCHEDULED_POSTS_S3_SECRET_ACCESS_KEY=secret\n"))
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("insecure production object store error = %v", err)
	}

	cfg, err = LoadConfig(EnvProd, testConfigFile(t, prodBase+"SCHEDULED_POSTS_S3_ENDPOINT=https://objects.example\nSCHEDULED_POSTS_S3_REGION=eu-west-2\nSCHEDULED_POSTS_S3_BUCKET=private\nSCHEDULED_POSTS_S3_ACCESS_KEY_ID=key\nSCHEDULED_POSTS_S3_SECRET_ACCESS_KEY=secret\n"))
	if err != nil {
		t.Fatalf("LoadConfig production object store: %v", err)
	}
	if cfg.ScheduledPostsS3.SecretAccessKey != "secret" {
		t.Fatal("production object-store credentials were not loaded")
	}
}

func TestLoadConfigRejectsMultipleAppViewReplicasWithoutSharedAdmission(t *testing.T) {
	const production = "DATABASE_URL=postgres://prod\n" +
		"ALLOWED_ORIGINS=https://craftsky.social\n" +
		"TAP_WS_URL=ws://tap\n" +
		"SCHEDULED_POSTS_S3_ENDPOINT=https://objects.example\n" +
		"SCHEDULED_POSTS_S3_REGION=eu-west-2\n" +
		"SCHEDULED_POSTS_S3_BUCKET=private\n" +
		"SCHEDULED_POSTS_S3_ACCESS_KEY_ID=key\n" +
		"SCHEDULED_POSTS_S3_SECRET_ACCESS_KEY=secret\n"

	_, err := LoadConfig(EnvProd, testConfigFile(t, withProductionOAuth(production)+
		"APPVIEW_REPLICA_COUNT=2\n"+
		"INSTAGRAM_SHARED_RATE_LIMITS=true\n"))
	if err == nil || !strings.Contains(err.Error(), "APPVIEW_REPLICA_COUNT") {
		t.Fatalf("LoadConfig multi-replica AppView error = %v", err)
	}
}

func TestLoadConfigFederatedHTTPBudgetsArePositiveAndLowerOnly(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"

	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatalf("LoadConfig defaults: %v", err)
	}
	if cfg.FederatedHTTP.Transport.DialTimeout != 5*time.Second ||
		cfg.FederatedHTTP.OAuthMetadata.TotalTimeout != 10*time.Second ||
		cfg.FederatedHTTP.PDSUpload.TotalTimeout != 20*time.Second {
		t.Fatalf("federated defaults = %+v", cfg.FederatedHTTP)
	}

	cfg, err = LoadConfig(EnvDev, testConfigFile(t, base+
		"FEDERATED_HTTP_DIAL_TIMEOUT=2s\n"+
		"FEDERATED_OAUTH_METADATA_TIMEOUT=4s\n"+
		"FEDERATED_PDS_JSON_RESPONSE_LIMIT_BYTES=1048576\n"))
	if err != nil {
		t.Fatalf("LoadConfig lower budgets: %v", err)
	}
	if cfg.FederatedHTTP.Transport.DialTimeout != 2*time.Second ||
		cfg.FederatedHTTP.OAuthMetadata.TotalTimeout != 4*time.Second ||
		cfg.FederatedHTTP.PDSJSON.ResponseLimit != 1<<20 {
		t.Fatalf("lowered federated budgets = %+v", cfg.FederatedHTTP)
	}

	for _, override := range []string{
		"FEDERATED_HTTP_DIAL_TIMEOUT=6s\n",
		"FEDERATED_OAUTH_METADATA_TIMEOUT=0s\n",
		"FEDERATED_PDS_JSON_RESPONSE_LIMIT_BYTES=4194305\n",
	} {
		if _, err := LoadConfig(EnvDev, testConfigFile(t, base+override)); err == nil {
			t.Fatalf("LoadConfig accepted unsafe override %q", override)
		}
	}
}

func TestLoadConfig_ObservabilityDefaultsAndValidation(t *testing.T) {
	t.Run("empty sentry config disables export and unsafe body logging by default", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
		cfg, err := LoadConfig(EnvDev, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.SentryDSN != "" {
			t.Fatalf("SentryDSN = %q, want empty", cfg.SentryDSN)
		}
		if cfg.SentryTracingEnabled {
			t.Fatal("SentryTracingEnabled = true, want false")
		}
		if cfg.SentryTracesSampleRate != 0 {
			t.Fatalf("SentryTracesSampleRate = %v, want 0", cfg.SentryTracesSampleRate)
		}
		if cfg.SentryLogsEnabled {
			t.Fatal("SentryLogsEnabled = true, want false")
		}
		if cfg.SentryMetricsEnabled {
			t.Fatal("SentryMetricsEnabled = true, want false")
		}
		if cfg.SentryTapTracingEnabled {
			t.Fatal("SentryTapTracingEnabled = true, want false")
		}
		if cfg.SentryTapTracesSampleRate != 0 {
			t.Fatalf("SentryTapTracesSampleRate = %v, want 0", cfg.SentryTapTracesSampleRate)
		}
		if cfg.UnsafeLogResponseBodies {
			t.Fatal("UnsafeLogResponseBodies = true, want false")
		}
	})

	t.Run("dsn only enables errors and panics but not logs tracing metrics or tap traces", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_DSN=https://public@example.invalid/1\n")
		cfg, err := LoadConfig(EnvDev, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.SentryDSN == "" {
			t.Fatal("SentryDSN = empty, want configured")
		}
		if cfg.SentryTracingEnabled {
			t.Fatal("SentryTracingEnabled = true, want false")
		}
		if cfg.SentryLogsEnabled {
			t.Fatal("SentryLogsEnabled = true, want false")
		}
		if cfg.SentryMetricsEnabled {
			t.Fatal("SentryMetricsEnabled = true, want false")
		}
		if cfg.SentryTapTracingEnabled {
			t.Fatal("SentryTapTracingEnabled = true, want false")
		}
	})

	t.Run("explicit logs metrics and tap tracing flags are honored with dsn", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_DSN=https://public@example.invalid/1\nSENTRY_LOGS_ENABLED=true\nSENTRY_METRICS_ENABLED=true\nSENTRY_TAP_TRACING_ENABLED=true\nSENTRY_TAP_TRACES_SAMPLE_RATE=0.25\n")
		cfg, err := LoadConfig(EnvDev, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !cfg.SentryLogsEnabled {
			t.Fatal("SentryLogsEnabled = false, want true")
		}
		if !cfg.SentryMetricsEnabled {
			t.Fatal("SentryMetricsEnabled = false, want true")
		}
		if !cfg.SentryTapTracingEnabled {
			t.Fatal("SentryTapTracingEnabled = false, want true")
		}
		if cfg.SentryTapTracesSampleRate != 0.25 {
			t.Fatalf("SentryTapTracesSampleRate = %v, want 0.25", cfg.SentryTapTracesSampleRate)
		}
	})

	t.Run("explicit sentry pillars are disabled without dsn", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_LOGS_ENABLED=true\nSENTRY_METRICS_ENABLED=true\nSENTRY_TAP_TRACING_ENABLED=true\nSENTRY_TAP_TRACES_SAMPLE_RATE=1\n")
		cfg, err := LoadConfig(EnvDev, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.SentryLogsEnabled {
			t.Fatal("SentryLogsEnabled = true without dsn, want false")
		}
		if cfg.SentryMetricsEnabled {
			t.Fatal("SentryMetricsEnabled = true without dsn, want false")
		}
		if cfg.SentryTapTracingEnabled {
			t.Fatal("SentryTapTracingEnabled = true without dsn, want false")
		}
		if cfg.SentryTapTracesSampleRate != 0 {
			t.Fatalf("SentryTapTracesSampleRate = %v, want 0", cfg.SentryTapTracesSampleRate)
		}
	})

	t.Run("prod tracing enabled without explicit sample rate defaults conservatively", func(t *testing.T) {
		path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_DSN=https://public@example.invalid/1\nSENTRY_TRACING_ENABLED=true\n"))
		cfg, err := LoadConfig(EnvProd, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !cfg.SentryTracingEnabled {
			t.Fatal("SentryTracingEnabled = false, want true")
		}
		if cfg.SentryTracesSampleRate != 0.01 {
			t.Fatalf("SentryTracesSampleRate = %v, want 0.01", cfg.SentryTracesSampleRate)
		}
	})

	t.Run("sample rate must be between zero and one", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_DSN=https://public@example.invalid/1\nSENTRY_TRACING_ENABLED=true\nSENTRY_TRACES_SAMPLE_RATE=1.5\n")
		_, err := LoadConfig(EnvDev, path)
		if err == nil {
			t.Fatal("expected sample rate validation error")
		}
		if !strings.Contains(err.Error(), "SENTRY_TRACES_SAMPLE_RATE") {
			t.Fatalf("error = %v, want SENTRY_TRACES_SAMPLE_RATE", err)
		}
	})

	t.Run("tap trace sample rate must be between zero and one", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nSENTRY_DSN=https://public@example.invalid/1\nSENTRY_TAP_TRACING_ENABLED=true\nSENTRY_TAP_TRACES_SAMPLE_RATE=-0.1\n")
		_, err := LoadConfig(EnvDev, path)
		if err == nil {
			t.Fatal("expected tap trace sample rate validation error")
		}
		if !strings.Contains(err.Error(), "SENTRY_TAP_TRACES_SAMPLE_RATE") {
			t.Fatalf("error = %v, want SENTRY_TAP_TRACES_SAMPLE_RATE", err)
		}
	})

	t.Run("prod forces unsafe response body logging off", func(t *testing.T) {
		path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_UNSAFE_LOG_RESPONSE_BODIES=true\n"))
		cfg, err := LoadConfig(EnvProd, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.UnsafeLogResponseBodies {
			t.Fatal("UnsafeLogResponseBodies = true in prod, want forced false")
		}
	})

	t.Run("unsafe Instagram webhook logging is opt-in in dev", func(t *testing.T) {
		path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nINSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES=true\n")
		cfg, err := LoadConfig(EnvDev, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !cfg.UnsafeLogInstagramWebhookBodies {
			t.Fatal("UnsafeLogInstagramWebhookBodies = false, want true")
		}
	})

	t.Run("prod forces unsafe Instagram webhook logging off", func(t *testing.T) {
		path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap:2480/channel\nINSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES=true\n"))
		cfg, err := LoadConfig(EnvProd, path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.UnsafeLogInstagramWebhookBodies {
			t.Fatal("UnsafeLogInstagramWebhookBodies = true in prod, want forced false")
		}
	})
}

func TestLoadConfig_DevModerationRequiresTokenWhenEnabled(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_ENABLE_DEV_MODERATION=true\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected dev moderation token error")
	}
	if !strings.Contains(err.Error(), "APPVIEW_DEV_MODERATION_TOKEN") {
		t.Fatalf("error = %v, want APPVIEW_DEV_MODERATION_TOKEN", err)
	}
}

func TestLoadConfig_DevModerationConfig(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_ENABLE_DEV_MODERATION=true\nAPPVIEW_DEV_MODERATION_TOKEN=secret-token\nCRAFTSKY_DEV_LABELER_DID=did:plc:labeler\nAPPVIEW_TRUSTED_MODERATION_SOURCE_DIDS=did:plc:ozone,did:plc:labeler\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.EnableDevModeration {
		t.Fatal("EnableDevModeration = false, want true")
	}
	if cfg.DevModerationToken != "secret-token" {
		t.Fatalf("DevModerationToken = %q", cfg.DevModerationToken)
	}
	if cfg.DevLabelerDID != "did:plc:labeler" {
		t.Fatalf("DevLabelerDID = %q", cfg.DevLabelerDID)
	}
	if got := cfg.TrustedModerationSourceDIDs; len(got) != 2 || got[0] != "did:plc:ozone" || got[1] != "did:plc:labeler" {
		t.Fatalf("TrustedModerationSourceDIDs = %v", got)
	}
}

func TestLoadConfigDevOAuthSchemeIsExplicitAndDevelopmentOnly(t *testing.T) {
	const devBase = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n"

	disabled, err := LoadConfig(EnvDev, testConfigFile(t, devBase))
	if err != nil {
		t.Fatalf("LoadConfig disabled development scheme: %v", err)
	}
	if disabled.EnableDevOAuthScheme {
		t.Fatal("EnableDevOAuthScheme = true without explicit opt-in")
	}

	enabled, err := LoadConfig(EnvDev, testConfigFile(t, devBase+"APPVIEW_ENABLE_DEV_OAUTH_SCHEME=true\n"))
	if err != nil {
		t.Fatalf("LoadConfig enabled development scheme: %v", err)
	}
	if !enabled.EnableDevOAuthScheme {
		t.Fatal("EnableDevOAuthScheme = false after explicit development opt-in")
	}

	prod := withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_ENABLE_DEV_OAUTH_SCHEME=true\n")
	_, err = LoadConfig(EnvProd, testConfigFile(t, prod))
	if err == nil || !strings.Contains(err.Error(), "APPVIEW_ENABLE_DEV_OAUTH_SCHEME") {
		t.Fatalf("LoadConfig production dev scheme error = %v", err)
	}
}

func TestLoadConfig_ProdClearsDevModerationFields(t *testing.T) {
	path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_ENABLE_DEV_MODERATION=true\nAPPVIEW_DEV_MODERATION_TOKEN=secret-token\nCRAFTSKY_DEV_LABELER_DID=did:plc:labeler\nAPPVIEW_TRUSTED_MODERATION_SOURCE_DIDS=did:plc:ozone\n"))
	cfg, err := LoadConfig(EnvProd, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.EnableDevModeration || cfg.DevModerationToken != "" || cfg.DevLabelerDID != "" || len(cfg.TrustedModerationSourceDIDs) != 0 {
		t.Fatalf("prod dev moderation fields not cleared: %+v", cfg)
	}
}

func TestLoadConfig_DevValid(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Env != EnvDev {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDev)
	}
	if cfg.DatabaseURL != "postgres://dev" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
	if cfg.DevDID != "did:plc:test" {
		t.Errorf("DevDID = %q", cfg.DevDID)
	}
	if cfg.MaxPostImages != api.DefaultMaxPostImages {
		t.Errorf("MaxPostImages = %d, want %d", cfg.MaxPostImages, api.DefaultMaxPostImages)
	}
	if cfg.MaxImageUploadBytes != api.DefaultMaxImageUploadBytes {
		t.Errorf("MaxImageUploadBytes = %d, want %d", cfg.MaxImageUploadBytes, api.DefaultMaxImageUploadBytes)
	}
}

func TestLoadConfig_ProdValid(t *testing.T) {
	path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example,https://b.example\nTAP_WS_URL=ws://tap:2480/channel\n"))
	cfg, err := LoadConfig(EnvProd, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.AllowedOrigins; len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Errorf("AllowedOrigins = %v", got)
	}
	if cfg.DevDID != "" {
		t.Errorf("DevDID = %q, want empty in prod", cfg.DevDID)
	}
}

func TestLoadConfig_ProdRejectsWildcardOrigin(t *testing.T) {
	path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=*\nTAP_WS_URL=ws://tap:2480/channel\n"))
	_, err := LoadConfig(EnvProd, path)
	if err == nil {
		t.Fatal("expected prod wildcard origin error")
	}
	if !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("error = %v, want ALLOWED_ORIGINS", err)
	}
}

func TestLoadConfig_LimitDefaults(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.JSONBodyLimitBytes != 1024*1024 {
		t.Fatalf("JSONBodyLimitBytes = %d, want 1 MiB", cfg.JSONBodyLimitBytes)
	}
	read := cfg.RateLimits.Classes["read"]
	if read.Window != time.Minute || read.PerToken != 300 || read.PerDevice != 600 {
		t.Fatalf("read rate limit = %+v, want 300/min token and 600/min device", read)
	}
	upload := cfg.RateLimits.Classes["upload"]
	if upload.Window != time.Hour || upload.PerToken != 100 || upload.PerDevice != 200 {
		t.Fatalf("upload rate limit = %+v, want 100/hour token and 200/hour device", upload)
	}
}

func TestLoadConfig_JSONBodyLimitOverride(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_JSON_BODY_LIMIT_BYTES=2048\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.JSONBodyLimitBytes != 2048 {
		t.Fatalf("JSONBodyLimitBytes = %d, want 2048", cfg.JSONBodyLimitBytes)
	}
}

func TestLoadConfig_JSONBodyLimitInvalid(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_JSON_BODY_LIMIT_BYTES=0\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected JSON body limit error")
	}
	if !strings.Contains(err.Error(), "APPVIEW_JSON_BODY_LIMIT_BYTES") {
		t.Fatalf("error = %v, want APPVIEW_JSON_BODY_LIMIT_BYTES", err)
	}
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	path := testConfigFile(t, "ALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error should name DATABASE_URL, got %v", err)
	}
}

func TestLoadConfig_MissingDevDIDInDev(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nTAP_WS_URL=ws://tap:2480/channel\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected error for missing CRAFTSKY_DEV_DID in dev")
	}
	if !strings.Contains(err.Error(), "CRAFTSKY_DEV_DID") {
		t.Errorf("error should name CRAFTSKY_DEV_DID, got %v", err)
	}
}

func TestLoadConfig_OSEnvUsedWhenFileAbsent(t *testing.T) {
	path := testConfigFile(t, "ALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	t.Setenv("DATABASE_URL", "postgres://fromenv")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DatabaseURL != "postgres://fromenv" {
		t.Errorf("DatabaseURL = %q, want postgres://fromenv", cfg.DatabaseURL)
	}
}

func TestLoadConfig_OSEnvWinsOnConflict(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://fromfile\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	t.Setenv("DATABASE_URL", "postgres://fromenv")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DatabaseURL != "postgres://fromenv" {
		t.Errorf("DatabaseURL = %q, want postgres://fromenv (os.Getenv must win over .env file)", cfg.DatabaseURL)
	}
}

func TestLoadConfig_DevDIDIgnoredInProd(t *testing.T) {
	path := testConfigFile(t, withProductionOAuth("DATABASE_URL=postgres://p\nALLOWED_ORIGINS=https://a.example\nCRAFTSKY_DEV_DID=did:plc:leaked\nTAP_WS_URL=ws://tap:2480/channel\n"))
	cfg, err := LoadConfig(EnvProd, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DevDID != "" {
		t.Errorf("DevDID = %q, want empty in prod (leaked from .env)", cfg.DevDID)
	}
}

func TestLoadConfig_TapFields(t *testing.T) {
	dir := t.TempDir()
	envPath := dir + "/test.env"
	contents := "DATABASE_URL=postgres://x\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"TAP_ACK_TIMEOUT=7s\n" +
		"TAP_RECONNECT_MAX=45s\n" +
		"TAP_PROJECTION_POLL_INTERVAL=500ms\n" +
		"TAP_PROJECTION_LEASE_DURATION=20s\n" +
		"TAP_PROJECTION_BATCH_SIZE=12\n" +
		"TAP_PROJECTION_BACKOFF_MIN=2s\n" +
		"TAP_PROJECTION_BACKOFF_MAX=1m\n" +
		"TAP_REPOSITORY_POLL_INTERVAL=2s\n" +
		"TAP_REPOSITORY_LEASE_DURATION=30s\n" +
		"TAP_REPOSITORY_BATCH_SIZE=3\n" +
		"TAP_REPOSITORY_BACKOFF_MIN=3s\n" +
		"TAP_REPOSITORY_BACKOFF_MAX=2m\n"
	if err := os.WriteFile(envPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	// Clear env so file wins. godotenv.Load skips keys already set in env
	// (including to ""), so we unset rather than set-empty.
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID",
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_ACK_SAFETY_MARGIN", "TAP_TERMINAL_TRANSACTION_BUDGET", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES",
		"TAP_PROJECTION_POLL_INTERVAL", "TAP_PROJECTION_LEASE_DURATION", "TAP_PROJECTION_BATCH_SIZE",
		"TAP_PROJECTION_BACKOFF_MIN", "TAP_PROJECTION_BACKOFF_MAX",
		"TAP_REPOSITORY_POLL_INTERVAL", "TAP_REPOSITORY_LEASE_DURATION", "TAP_REPOSITORY_BATCH_SIZE",
		"TAP_REPOSITORY_BACKOFF_MIN", "TAP_REPOSITORY_BACKOFF_MAX",
		"MAX_POST_IMAGES", "MAX_IMAGE_UPLOAD_BYTES"} {
		prior, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, prior)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}

	cfg, err := LoadConfig(EnvDev, envPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.TapWSURL != "ws://tap:2480/channel" {
		t.Errorf("TapWSURL = %q", cfg.TapWSURL)
	}
	if cfg.TapAckTimeout != 7*time.Second {
		t.Errorf("TapAckTimeout = %v", cfg.TapAckTimeout)
	}
	if cfg.TapReconnectMax != 45*time.Second {
		t.Errorf("TapReconnectMax = %v", cfg.TapReconnectMax)
	}
	if cfg.TapProjectionPollInterval != 500*time.Millisecond || cfg.TapProjectionLeaseDuration != 20*time.Second ||
		cfg.TapProjectionBatchSize != 12 || cfg.TapProjectionBackoffMin != 2*time.Second || cfg.TapProjectionBackoffMax != time.Minute {
		t.Errorf("projection config = %+v", cfg)
	}
	if cfg.TapRepositoryPollInterval != 2*time.Second || cfg.TapRepositoryLeaseDuration != 30*time.Second ||
		cfg.TapRepositoryBatchSize != 3 || cfg.TapRepositoryBackoffMin != 3*time.Second || cfg.TapRepositoryBackoffMax != 2*time.Minute {
		t.Errorf("repository config = %+v", cfg)
	}
}

func TestLoadConfig_TapDefaults(t *testing.T) {
	envPath := testConfigFile(t, "DATABASE_URL=postgres://x\n"+
		"ALLOWED_ORIGINS=*\n"+
		"CRAFTSKY_DEV_DID=did:plc:test\n"+
		"TAP_WS_URL=ws://tap:2480/channel\n")

	cfg, err := LoadConfig(EnvDev, envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TapAckTimeout != 10*time.Second {
		t.Errorf("default TapAckTimeout = %v", cfg.TapAckTimeout)
	}
	if cfg.TapReconnectMax != 30*time.Second {
		t.Errorf("default TapReconnectMax = %v", cfg.TapReconnectMax)
	}
	if cfg.TapProjectionPollInterval != 250*time.Millisecond || cfg.TapProjectionLeaseDuration != 30*time.Second ||
		cfg.TapProjectionBatchSize != 32 || cfg.TapProjectionBackoffMin != time.Second || cfg.TapProjectionBackoffMax != 5*time.Minute {
		t.Errorf("default projection config = %+v", cfg)
	}
	if cfg.TapRepositoryPollInterval != time.Second || cfg.TapRepositoryLeaseDuration != 45*time.Second ||
		cfg.TapRepositoryBatchSize != 8 || cfg.TapRepositoryBackoffMin != 2*time.Second || cfg.TapRepositoryBackoffMax != 5*time.Minute {
		t.Errorf("default repository config = %+v", cfg)
	}
}

func TestLoadConfigRejectsRemovedTapRetryDropSetting(t *testing.T) {
	const base = "DATABASE_URL=postgres://x\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\nTAP_MAX_RETRIES=5\n"
	_, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err == nil || !strings.Contains(err.Error(), "TAP_MAX_RETRIES has been removed") {
		t.Fatalf("LoadConfig error = %v", err)
	}
}

func TestLoadConfig_MediaLimitOverrides(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nMAX_POST_IMAGES=2\nMAX_IMAGE_UPLOAD_BYTES=1048576\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MaxPostImages != 2 {
		t.Errorf("MaxPostImages = %d, want 2", cfg.MaxPostImages)
	}
	if cfg.MaxImageUploadBytes != 1048576 {
		t.Errorf("MaxImageUploadBytes = %d, want 1048576", cfg.MaxImageUploadBytes)
	}
}

func TestLoadConfig_MediaLimitOverridesCannotExceedContract(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nMAX_POST_IMAGES=5\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected MAX_POST_IMAGES error")
	}
	if !strings.Contains(err.Error(), "MAX_POST_IMAGES") {
		t.Errorf("error should mention MAX_POST_IMAGES, got: %v", err)
	}

	path = testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\nMAX_IMAGE_UPLOAD_BYTES=15728641\n")
	_, err = LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected MAX_IMAGE_UPLOAD_BYTES error")
	}
	if !strings.Contains(err.Error(), "MAX_IMAGE_UPLOAD_BYTES") {
		t.Errorf("error should mention MAX_IMAGE_UPLOAD_BYTES, got: %v", err)
	}
}

func TestLoadConfig_OAuthDevDefaults(t *testing.T) {
	// Only required dev vars set; OAUTH_* left at their defaults.
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.OAuth.Mode != OAuthModeLocalhost {
		t.Errorf("OAuth.Mode = %q, want %q", cfg.OAuth.Mode, OAuthModeLocalhost)
	}
	if got := cfg.OAuth.CallbackURL.String(); got != "http://127.0.0.1:18080/oauth/callback" {
		t.Errorf("OAuth.CallbackURL = %q, want localhost default", got)
	}
	want := []string{"atproto", "transition:generic"}
	if len(cfg.OAuth.Scopes) != len(want) {
		t.Errorf("OAuth.Scopes = %v, want %v", cfg.OAuth.Scopes, want)
	} else {
		for i, s := range want {
			if cfg.OAuth.Scopes[i] != s {
				t.Errorf("OAuth.Scopes[%d] = %q, want %q", i, cfg.OAuth.Scopes[i], s)
			}
		}
	}
	if cfg.OAuthSessionAbsoluteLifetime != 180*24*time.Hour {
		t.Errorf("OAuthSessionAbsoluteLifetime = %v, want %v", cfg.OAuthSessionAbsoluteLifetime, 180*24*time.Hour)
	}
	if cfg.CraftskySessionInactivity != 30*24*time.Hour {
		t.Errorf("CraftskySessionInactivity = %v, want %v", cfg.CraftskySessionInactivity, 30*24*time.Hour)
	}
	if cfg.OAuthAuthRequestExpiry != 30*time.Minute {
		t.Errorf("OAuthAuthRequestExpiry = %v, want %v", cfg.OAuthAuthRequestExpiry, 30*time.Minute)
	}
	if cfg.CraftskySessionActivityWriteInterval != 5*time.Minute {
		t.Errorf("CraftskySessionActivityWriteInterval = %v, want %v", cfg.CraftskySessionActivityWriteInterval, 5*time.Minute)
	}
	if cfg.OAuthLoginStartTimeout != 20*time.Second || cfg.OAuthCallbackOperationTimeout != 45*time.Second {
		t.Errorf("OAuth operation timeouts = %v/%v, want 20s/45s", cfg.OAuthLoginStartTimeout, cfg.OAuthCallbackOperationTimeout)
	}
	if cfg.OAuthSessionOperationTimeout != 30*time.Second {
		t.Errorf("OAuthSessionOperationTimeout = %v, want 30s", cfg.OAuthSessionOperationTimeout)
	}
	if cfg.OAuthHandoffExchangeTTL != 10*time.Minute || cfg.OAuthHandoffConfirmationTTL != 2*time.Minute {
		t.Errorf("OAuth handoff TTLs = %v/%v, want 10m/2m", cfg.OAuthHandoffExchangeTTL, cfg.OAuthHandoffConfirmationTTL)
	}
	if got := cfg.VerifiedLinkOrigin.String(); got != "https://app.craftsky.social" {
		t.Errorf("VerifiedLinkOrigin = %q", got)
	}
	if cfg.OAuth.ClientKeyID != "" || cfg.OAuth.ClientSecretKey != "" {
		t.Errorf("localhost OAuth unexpectedly contains confidential fields: %+v", cfg.OAuth)
	}
}

func TestLoadConfig_ProductionRequiresCanonicalOAuthOrigin(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap:2480/channel\nOAUTH_CLIENT_SECRET_KEY=secret\nOAUTH_CLIENT_SECRET_KEY_ID=primary\n")
	_, err := LoadConfig(EnvProd, path)
	if err == nil {
		t.Fatal("LoadConfig error = nil, want missing canonical OAuth origin")
	}
	if !strings.Contains(err.Error(), "OAUTH_PUBLIC_ORIGIN") {
		t.Fatalf("LoadConfig error = %v, want OAUTH_PUBLIC_ORIGIN", err)
	}
}

// UT-002: the registration provider is server-owned, defaults to Bluesky in
// production, and accepts only canonical public HTTPS origins.
func TestLoadConfigRegistrationProviderOrigin(t *testing.T) {
	const base = "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap:2480/channel\n"
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "production default", want: "https://bsky.social"},
		{name: "controlled public HTTPS fixture", value: "https://pds.registration.craftsky.social", want: "https://pds.registration.craftsky.social"},
		{name: "private origin", value: "https://localhost", wantErr: true},
		{name: "HTTP origin", value: "http://pds.registration.craftsky.social", wantErr: true},
		{name: "path", value: "https://pds.registration.craftsky.social/xrpc", wantErr: true},
		{name: "query", value: "https://pds.registration.craftsky.social?provider=bluesky", wantErr: true},
		{name: "credentials", value: "https://user@pds.registration.craftsky.social", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contents := withProductionOAuth(base)
			if test.value != "" {
				contents += "OAUTH_REGISTRATION_PROVIDER_ORIGIN=" + test.value + "\n"
			}

			cfg, err := LoadConfig(EnvProd, testConfigFile(t, contents))
			if test.wantErr {
				if err == nil {
					t.Fatal("LoadConfig error = nil, want unsafe registration provider origin rejected")
				}
				if !strings.Contains(err.Error(), "OAUTH_REGISTRATION_PROVIDER_ORIGIN") {
					t.Fatalf("LoadConfig error = %v, want OAUTH_REGISTRATION_PROVIDER_ORIGIN", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if got := cfg.OAuthRegistrationProviderOrigin.String(); got != test.want {
				t.Fatalf("OAuthRegistrationProviderOrigin = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoadConfigDevRejectsConfidentialOAuthFields(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n"
	for _, keyValue := range []string{
		"OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n",
		"OAUTH_CLIENT_SECRET_KEY=not-used-in-localhost-mode\n",
		"OAUTH_CLIENT_SECRET_KEY=\n",
		"OAUTH_CLIENT_SECRET_KEY_ID=primary\n",
	} {
		key := strings.SplitN(keyValue, "=", 2)[0]
		t.Run(key, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+keyValue))
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("LoadConfig error = %v, want %s", err, key)
			}
		})
	}
}

func TestParseCanonicalPublicOrigin(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "canonical", raw: "https://appview.craftsky.social", want: "https://appview.craftsky.social"},
		{name: "normalises case and root", raw: "https://APPVIEW.CRAFTSKY.SOCIAL/", want: "https://appview.craftsky.social"},
		{name: "http", raw: "http://appview.craftsky.social", wantErr: true},
		{name: "IP literal", raw: "https://127.0.0.1", wantErr: true},
		{name: "localhost", raw: "https://localhost", wantErr: true},
		{name: "local suffix", raw: "https://appview.craftsky.local", wantErr: true},
		{name: "userinfo", raw: "https://user@appview.craftsky.social", wantErr: true},
		{name: "path", raw: "https://appview.craftsky.social/oauth", wantErr: true},
		{name: "query", raw: "https://appview.craftsky.social?mode=prod", wantErr: true},
		{name: "empty query", raw: "https://appview.craftsky.social?", wantErr: true},
		{name: "fragment", raw: "https://appview.craftsky.social#section", wantErr: true},
		{name: "empty fragment", raw: "https://appview.craftsky.social#", wantErr: true},
		{name: "explicit port", raw: "https://appview.craftsky.social:443", wantErr: true},
		{name: "malformed port", raw: "https://appview.craftsky.social:notaport", wantErr: true},
		{name: "encoded path", raw: "https://appview.craftsky.social/%2Foauth", wantErr: true},
		{name: "special-use suffix", raw: "https://appview.craftsky.test", wantErr: true},
		{name: "reserved example domain", raw: "https://appview.example.net", wantErr: true},
		{name: "trailing dot", raw: "https://appview.craftsky.social.", wantErr: true},
		{name: "unicode ambiguity", raw: "https://appvıew.craftsky.social", wantErr: true},
		{name: "backslash ambiguity", raw: "https://appview.craftsky.social\\@evil.example", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			origin, err := parseCanonicalPublicOrigin(test.raw)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseCanonicalPublicOrigin(%q) error = nil", test.raw)
				}
				if !strings.Contains(err.Error(), "OAUTH_PUBLIC_ORIGIN") {
					t.Fatalf("error = %v, want OAUTH_PUBLIC_ORIGIN", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCanonicalPublicOrigin(%q): %v", test.raw, err)
			}
			if got := origin.String(); got != test.want {
				t.Fatalf("origin = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseLoopbackOAuthCallback(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid", raw: "http://127.0.0.1:18080/oauth/callback"},
		{name: "missing port", raw: "http://127.0.0.1/oauth/callback", wantErr: true},
		{name: "zero port", raw: "http://127.0.0.1:0/oauth/callback", wantErr: true},
		{name: "out of range port", raw: "http://127.0.0.1:65536/oauth/callback", wantErr: true},
		{name: "localhost alias", raw: "http://localhost:18080/oauth/callback", wantErr: true},
		{name: "IPv6 loopback", raw: "http://[::1]:18080/oauth/callback", wantErr: true},
		{name: "HTTPS", raw: "https://127.0.0.1:18080/oauth/callback", wantErr: true},
		{name: "wrong path", raw: "http://127.0.0.1:18080/callback", wantErr: true},
		{name: "query", raw: "http://127.0.0.1:18080/oauth/callback?x=1", wantErr: true},
		{name: "userinfo", raw: "http://user@127.0.0.1:18080/oauth/callback", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callback, err := parseLoopbackOAuthCallback(test.raw)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseLoopbackOAuthCallback(%q) error = nil", test.raw)
				}
				if !strings.Contains(err.Error(), "OAUTH_CALLBACK_URL") {
					t.Fatalf("error = %v, want OAUTH_CALLBACK_URL", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLoopbackOAuthCallback(%q): %v", test.raw, err)
			}
			if got := callback.String(); got != test.raw {
				t.Fatalf("callback = %q, want %q", got, test.raw)
			}
		})
	}
}

func TestParseOAuthScopes(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "valid", raw: "atproto transition:generic", want: []string{"atproto", "transition:generic"}},
		{name: "missing atproto", raw: "transition:generic", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "duplicate", raw: "atproto atproto", wantErr: true},
		{name: "control character", raw: "atproto transition:generic\u007f", wantErr: true},
		{name: "control separator", raw: "atproto\ntransition:generic", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOAuthScopes(test.raw)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseOAuthScopes(%q) error = nil", test.raw)
				}
				if !strings.Contains(err.Error(), "OAUTH_SCOPES") {
					t.Fatalf("error = %v, want OAUTH_SCOPES", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOAuthScopes(%q): %v", test.raw, err)
			}
			if strings.Join(got, " ") != strings.Join(test.want, " ") {
				t.Fatalf("scopes = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLoadConfigProductionOAuthCompleteness(t *testing.T) {
	const complete = "DATABASE_URL=postgres://prod\n" +
		"ALLOWED_ORIGINS=https://craftsky.social\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n" +
		"OAUTH_CLIENT_SECRET_KEY=test-private-key\n" +
		"OAUTH_CLIENT_SECRET_KEY_ID=primary\n" +
		"OAUTH_SCOPES=atproto transition:generic\n"

	tests := []struct {
		name    string
		config  string
		wantKey string
	}{
		{name: "missing origin", config: strings.Replace(complete, "OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n", "", 1), wantKey: "OAUTH_PUBLIC_ORIGIN"},
		{name: "missing private key", config: strings.Replace(complete, "OAUTH_CLIENT_SECRET_KEY=test-private-key\n", "", 1), wantKey: "OAUTH_CLIENT_SECRET_KEY"},
		{name: "missing key id", config: strings.Replace(complete, "OAUTH_CLIENT_SECRET_KEY_ID=primary\n", "", 1), wantKey: "OAUTH_CLIENT_SECRET_KEY_ID"},
		{name: "missing atproto scope", config: strings.Replace(complete, "OAUTH_SCOPES=atproto transition:generic", "OAUTH_SCOPES=transition:generic", 1), wantKey: "OAUTH_SCOPES"},
		{name: "legacy hostname", config: complete + "OAUTH_HOSTNAME=appview.craftsky.social\n", wantKey: "OAUTH_HOSTNAME"},
		{name: "legacy callback", config: complete + "OAUTH_CALLBACK_URL=https://appview.craftsky.social/oauth/callback\n", wantKey: "OAUTH_CALLBACK_URL"},
		{name: "empty legacy callback", config: complete + "OAUTH_CALLBACK_URL=\n", wantKey: "OAUTH_CALLBACK_URL"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvProd, testConfigFile(t, test.config))
			if err == nil {
				t.Fatalf("LoadConfig error = nil, want %s", test.wantKey)
			}
			if !strings.Contains(err.Error(), test.wantKey) {
				t.Fatalf("LoadConfig error = %v, want %s", err, test.wantKey)
			}
		})
	}

	cfg, err := LoadConfig(EnvProd, testConfigFile(t, complete))
	if err != nil {
		t.Fatalf("LoadConfig complete production OAuth: %v", err)
	}
	if cfg.OAuth.Mode != OAuthModeConfidential {
		t.Fatalf("OAuth mode = %q, want %q", cfg.OAuth.Mode, OAuthModeConfidential)
	}
	if got := cfg.OAuth.ClientID.String(); got != "https://appview.craftsky.social/oauth/client-metadata.json" {
		t.Fatalf("OAuth client ID = %q", got)
	}
	if got := cfg.OAuth.CallbackURL.String(); got != "https://appview.craftsky.social/oauth/callback" {
		t.Fatalf("OAuth callback = %q", got)
	}
	if got := cfg.OAuth.JWKSURL.String(); got != "https://appview.craftsky.social/oauth/jwks.json" {
		t.Fatalf("OAuth JWKS URL = %q", got)
	}
}

func TestLoadConfigExpectedHostPolicy(t *testing.T) {
	dev, err := LoadConfig(EnvDev, testConfigFile(t,
		"DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n",
	))
	if err != nil {
		t.Fatalf("LoadConfig dev: %v", err)
	}
	if strings.Join(dev.ExpectedHosts, ",") != "127.0.0.1,localhost" || !dev.ExpectedHostAllowAnyPort {
		t.Fatalf("dev expected Host policy = %v anyPort=%t", dev.ExpectedHosts, dev.ExpectedHostAllowAnyPort)
	}

	const production = "DATABASE_URL=postgres://prod\n" +
		"ALLOWED_ORIGINS=https://craftsky.social\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n" +
		"OAUTH_CLIENT_SECRET_KEY=test-private-key\n" +
		"OAUTH_CLIENT_SECRET_KEY_ID=primary\n" +
		"OAUTH_SCOPES=atproto transition:generic\n"
	prod, err := LoadConfig(EnvProd, testConfigFile(t, production))
	if err != nil {
		t.Fatalf("LoadConfig prod: %v", err)
	}
	if strings.Join(prod.ExpectedHosts, ",") != "appview.craftsky.social" || prod.ExpectedHostAllowAnyPort {
		t.Fatalf("prod expected Host policy = %v anyPort=%t", prod.ExpectedHosts, prod.ExpectedHostAllowAnyPort)
	}

	_, err = LoadConfig(EnvProd, testConfigFile(t, production+"APPVIEW_EXPECTED_HOSTS=attacker.invalid\n"))
	if err == nil || !strings.Contains(err.Error(), "APPVIEW_EXPECTED_HOSTS") {
		t.Fatalf("mismatched production expected hosts error = %v", err)
	}

	_, err = LoadConfig(EnvDev, testConfigFile(t,
		"DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\nAPPVIEW_EXPECTED_HOSTS=localhost,\n",
	))
	if err == nil || !strings.Contains(err.Error(), "APPVIEW_EXPECTED_HOSTS") {
		t.Fatalf("empty development expected host error = %v", err)
	}
}

func TestLoadConfigRemoteDevelopmentFailsClosed(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	const strongSecret = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	tests := []struct {
		name    string
		extra   string
		wantKey string
	}{
		{name: "non-loopback publish without remote mode", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\n", wantKey: "APPVIEW_DEV_REMOTE_ACCESS"},
		{name: "remote mode with loopback publish", extra: "APPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n", wantKey: "APPVIEW_PUBLISH_HOST"},
		{name: "remote mode without protected transport", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n", wantKey: "APPVIEW_DEV_PROTECTED_TRANSPORT"},
		{name: "remote mode without secret", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n", wantKey: "APPVIEW_DEV_AUTH_SECRET"},
		{name: "remote mode with weak secret", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=too-short\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n", wantKey: "APPVIEW_DEV_AUTH_SECRET"},
		{name: "remote mode with low-diversity secret", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n", wantKey: "APPVIEW_DEV_AUTH_SECRET"},
		{name: "remote mode without expected tunnel host", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\n", wantKey: "APPVIEW_EXPECTED_HOSTS"},
		{name: "remote mode with expected host port", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social:443\n", wantKey: "APPVIEW_EXPECTED_HOSTS"},
		{name: "remote mode with duplicate expected host", extra: "APPVIEW_PUBLISH_HOST=0.0.0.0\nAPPVIEW_DEV_REMOTE_ACCESS=true\nAPPVIEW_DEV_PROTECTED_TRANSPORT=true\nAPPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\nAPPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social,TUNNEL.CRAFTSKY.SOCIAL\n", wantKey: "APPVIEW_EXPECTED_HOSTS"},
		{name: "secret in local mode", extra: "APPVIEW_DEV_AUTH_SECRET=" + strongSecret + "\n", wantKey: "APPVIEW_DEV_REMOTE_ACCESS"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.extra))
			if err == nil {
				t.Fatalf("LoadConfig error = nil, want %s", test.wantKey)
			}
			if !strings.Contains(err.Error(), test.wantKey) {
				t.Fatalf("LoadConfig error = %v, want %s", err, test.wantKey)
			}
		})
	}

	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base+
		"APPVIEW_PUBLISH_HOST=0.0.0.0\n"+
		"APPVIEW_DEV_REMOTE_ACCESS=true\n"+
		"APPVIEW_DEV_PROTECTED_TRANSPORT=true\n"+
		"APPVIEW_DEV_AUTH_SECRET="+strongSecret+"\n"+
		"APPVIEW_EXPECTED_HOSTS=tunnel.craftsky.social\n"))
	if err != nil {
		t.Fatalf("LoadConfig valid remote development: %v", err)
	}
	if !cfg.DevRemoteAccess || !cfg.DevProtectedTransport || cfg.DevAuthSecret != strongSecret || cfg.ExpectedHostAllowAnyPort {
		t.Fatalf("remote dev flags: remote=%t protected=%t secretConfigured=%t anyPort=%t", cfg.DevRemoteAccess, cfg.DevProtectedTransport, cfg.DevAuthSecret != "", cfg.ExpectedHostAllowAnyPort)
	}
}

func TestLoadConfigProductionRejectsDevAuthorizationConfig(t *testing.T) {
	const production = "DATABASE_URL=postgres://prod\n" +
		"ALLOWED_ORIGINS=https://craftsky.social\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"OAUTH_PUBLIC_ORIGIN=https://appview.craftsky.social\n" +
		"OAUTH_CLIENT_SECRET_KEY=test-private-key\n" +
		"OAUTH_CLIENT_SECRET_KEY_ID=primary\n" +
		"APPVIEW_DEV_REMOTE_ACCESS=true\n" +
		"APPVIEW_DEV_PROTECTED_TRANSPORT=true\n" +
		"APPVIEW_DEV_AUTH_SECRET=0123456789abcdefghijklmnopqrstuvwxyzABCDEFG\n"
	_, err := LoadConfig(EnvProd, testConfigFile(t, production))
	if err == nil || !strings.Contains(err.Error(), "APPVIEW_DEV_REMOTE_ACCESS") {
		t.Fatalf("LoadConfig production dev-auth error = %v", err)
	}
}

func TestLoadConfigOperationalDurationsFailClosed(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	tests := []struct {
		name     string
		override string
		wantKey  string
	}{
		{name: "zero Tap ACK", override: "TAP_ACK_TIMEOUT=0s\n", wantKey: "TAP_ACK_TIMEOUT"},
		{name: "negative Tap reconnect", override: "TAP_RECONNECT_MAX=-1s\n", wantKey: "TAP_RECONNECT_MAX"},
		{name: "Tap ACK above maximum", override: "TAP_ACK_TIMEOUT=2m1ns\n", wantKey: "TAP_ACK_TIMEOUT"},
		{name: "Tap reconnect above maximum", override: "TAP_RECONNECT_MAX=10m1ns\n", wantKey: "TAP_RECONNECT_MAX"},
		{name: "zero absolute lifetime", override: "OAUTH_SESSION_ABSOLUTE_LIFETIME=0s\n", wantKey: "OAUTH_SESSION_ABSOLUTE_LIFETIME"},
		{name: "absolute lifetime above maximum", override: "OAUTH_SESSION_ABSOLUTE_LIFETIME=4320h1ns\n", wantKey: "OAUTH_SESSION_ABSOLUTE_LIFETIME"},
		{name: "negative inactivity", override: "CRAFTSKY_SESSION_INACTIVITY=-1s\n", wantKey: "CRAFTSKY_SESSION_INACTIVITY"},
		{name: "inactivity above maximum", override: "CRAFTSKY_SESSION_INACTIVITY=4320h1ns\n", wantKey: "CRAFTSKY_SESSION_INACTIVITY"},
		{name: "zero auth request expiry", override: "OAUTH_AUTH_REQUEST_EXPIRY=0s\n", wantKey: "OAUTH_AUTH_REQUEST_EXPIRY"},
		{name: "auth request above maximum", override: "OAUTH_AUTH_REQUEST_EXPIRY=1h1ns\n", wantKey: "OAUTH_AUTH_REQUEST_EXPIRY"},
		{name: "zero activity write interval", override: "CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL=0s\n", wantKey: "CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL"},
		{name: "activity write interval above maximum", override: "CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL=24h1ns\n", wantKey: "CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL"},
		{name: "zero OAuth login start", override: "OAUTH_LOGIN_START_TIMEOUT=0s\n", wantKey: "OAUTH_LOGIN_START_TIMEOUT"},
		{name: "OAuth login start above maximum", override: "OAUTH_LOGIN_START_TIMEOUT=1m1ns\n", wantKey: "OAUTH_LOGIN_START_TIMEOUT"},
		{name: "zero OAuth callback operation", override: "OAUTH_CALLBACK_OPERATION_TIMEOUT=0s\n", wantKey: "OAUTH_CALLBACK_OPERATION_TIMEOUT"},
		{name: "OAuth callback operation above maximum", override: "OAUTH_CALLBACK_OPERATION_TIMEOUT=1m30s1ns\n", wantKey: "OAUTH_CALLBACK_OPERATION_TIMEOUT"},
		{name: "zero OAuth session operation", override: "OAUTH_SESSION_OPERATION_TIMEOUT=0s\n", wantKey: "OAUTH_SESSION_OPERATION_TIMEOUT"},
		{name: "OAuth session operation above maximum", override: "OAUTH_SESSION_OPERATION_TIMEOUT=1m30s1ns\n", wantKey: "OAUTH_SESSION_OPERATION_TIMEOUT"},
		{name: "zero handoff exchange", override: "OAUTH_HANDOFF_EXCHANGE_TTL=0s\n", wantKey: "OAUTH_HANDOFF_EXCHANGE_TTL"},
		{name: "handoff exchange above maximum", override: "OAUTH_HANDOFF_EXCHANGE_TTL=30m1ns\n", wantKey: "OAUTH_HANDOFF_EXCHANGE_TTL"},
		{name: "zero handoff confirmation", override: "OAUTH_HANDOFF_CONFIRMATION_TTL=0s\n", wantKey: "OAUTH_HANDOFF_CONFIRMATION_TTL"},
		{name: "handoff confirmation above maximum", override: "OAUTH_HANDOFF_CONFIRMATION_TTL=10m1ns\n", wantKey: "OAUTH_HANDOFF_CONFIRMATION_TTL"},
		{name: "zero push poll", override: "PUSH_POLL_INTERVAL=0s\n", wantKey: "PUSH_POLL_INTERVAL"},
		{name: "push poll above maximum", override: "PUSH_POLL_INTERVAL=1h1ns\n", wantKey: "PUSH_POLL_INTERVAL"},
		{name: "zero push lease", override: "PUSH_LEASE_DURATION=0s\n", wantKey: "PUSH_LEASE_DURATION"},
		{name: "push lease above maximum", override: "PUSH_LEASE_DURATION=1h1ns\n", wantKey: "PUSH_LEASE_DURATION"},
		{name: "negative push send", override: "PUSH_SEND_TIMEOUT=-1s\n", wantKey: "PUSH_SEND_TIMEOUT"},
		{name: "push send above maximum", override: "PUSH_SEND_TIMEOUT=10m1ns\n", wantKey: "PUSH_SEND_TIMEOUT"},
		{name: "legacy session expiry", override: "OAUTH_SESSION_EXPIRY=1h\n", wantKey: "OAUTH_SESSION_EXPIRY"},
		{name: "legacy empty session expiry", override: "OAUTH_SESSION_EXPIRY=\n", wantKey: "OAUTH_SESSION_EXPIRY"},
		{name: "legacy session inactivity", override: "OAUTH_SESSION_INACTIVITY=1h\n", wantKey: "OAUTH_SESSION_INACTIVITY"},
		{name: "legacy activity throttle", override: "CRAFTSKY_SESSION_LAST_SEEN_THROTTLE=1m\n", wantKey: "CRAFTSKY_SESSION_LAST_SEEN_THROTTLE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil {
				t.Fatalf("LoadConfig error = nil, want %s", test.wantKey)
			}
			if !strings.Contains(err.Error(), test.wantKey) {
				t.Fatalf("LoadConfig error = %v, want %s", err, test.wantKey)
			}
		})
	}
}

func TestLoadConfigOperationalDurationMaximumsAreInclusive(t *testing.T) {
	const config = "DATABASE_URL=postgres://dev\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"TAP_ACK_TIMEOUT=2m\n" +
		"TAP_RECONNECT_MAX=10m\n" +
		"OAUTH_SESSION_ABSOLUTE_LIFETIME=4320h\n" +
		"CRAFTSKY_SESSION_INACTIVITY=4320h\n" +
		"OAUTH_AUTH_REQUEST_EXPIRY=1h\n" +
		"CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL=24h\n" +
		"OAUTH_LOGIN_START_TIMEOUT=1m\n" +
		"OAUTH_CALLBACK_OPERATION_TIMEOUT=1m30s\n" +
		"OAUTH_SESSION_OPERATION_TIMEOUT=1m30s\n" +
		"OAUTH_HANDOFF_EXCHANGE_TTL=30m\n" +
		"OAUTH_HANDOFF_CONFIRMATION_TTL=10m\n" +
		"PUSH_POLL_INTERVAL=1h\n" +
		"PUSH_LEASE_DURATION=1h\n" +
		"PUSH_SEND_TIMEOUT=10m\n"
	if _, err := LoadConfig(EnvDev, testConfigFile(t, config)); err != nil {
		t.Fatalf("LoadConfig exact maxima: %v", err)
	}
}

func TestLoadConfigDurationRelationships(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	tests := []struct {
		name     string
		override string
		want     []string
	}{
		{name: "inactivity exceeds absolute lifetime", override: "OAUTH_SESSION_ABSOLUTE_LIFETIME=1h\nCRAFTSKY_SESSION_INACTIVITY=2h\n", want: []string{"CRAFTSKY_SESSION_INACTIVITY", "OAUTH_SESSION_ABSOLUTE_LIFETIME"}},
		{name: "auth request reaches absolute lifetime", override: "OAUTH_SESSION_ABSOLUTE_LIFETIME=30m\nCRAFTSKY_SESSION_INACTIVITY=20m\nOAUTH_AUTH_REQUEST_EXPIRY=30m\n", want: []string{"OAUTH_AUTH_REQUEST_EXPIRY", "OAUTH_SESSION_ABSOLUTE_LIFETIME"}},
		{name: "activity write interval reaches inactivity", override: "CRAFTSKY_SESSION_INACTIVITY=5m\nCRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL=5m\n", want: []string{"CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL", "CRAFTSKY_SESSION_INACTIVITY"}},
		{name: "handoff confirmation reaches exchange expiry", override: "OAUTH_HANDOFF_EXCHANGE_TTL=2m\nOAUTH_HANDOFF_CONFIRMATION_TTL=2m\n", want: []string{"OAUTH_HANDOFF_CONFIRMATION_TTL", "OAUTH_HANDOFF_EXCHANGE_TTL"}},
		{name: "push timeout leaves no safety margin", override: "PUSH_LEASE_DURATION=15s\nPUSH_SEND_TIMEOUT=10s\n", want: []string{"PUSH_SEND_TIMEOUT", "PUSH_LEASE_DURATION"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil {
				t.Fatal("LoadConfig error = nil")
			}
			for _, name := range test.want {
				if !strings.Contains(err.Error(), name) {
					t.Fatalf("LoadConfig error = %v, want %s", err, name)
				}
			}
		})
	}
}

func TestLoadConfigScheduledImageDecodeLimits(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	defaults, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatalf("LoadConfig defaults: %v", err)
	}
	if defaults.ImageDecodeLimits != api.DefaultImageDecodeLimits() {
		t.Fatalf("default image decode limits = %+v", defaults.ImageDecodeLimits)
	}

	lowered, err := LoadConfig(EnvDev, testConfigFile(t, base+
		"SCHEDULED_IMAGE_MAX_WIDTH=4000\n"+
		"SCHEDULED_IMAGE_MAX_HEIGHT=4000\n"+
		"SCHEDULED_IMAGE_MAX_PIXELS=16000000\n"+
		"SCHEDULED_IMAGE_MAX_ASPECT_RATIO=10\n"+
		"SCHEDULED_IMAGE_MAX_CONCURRENT_DECODES=1\n"+
		"SCHEDULED_IMAGE_ADMISSION_WAIT=100ms\n"))
	if err != nil {
		t.Fatalf("LoadConfig lowered limits: %v", err)
	}
	if lowered.ImageDecodeLimits.MaxWidth != 4000 || lowered.ImageDecodeLimits.AdmissionWait != 100*time.Millisecond {
		t.Fatalf("lowered image decode limits = %+v", lowered.ImageDecodeLimits)
	}

	tests := []struct {
		name     string
		override string
		wantKey  string
	}{
		{name: "zero width", override: "SCHEDULED_IMAGE_MAX_WIDTH=0\n", wantKey: "SCHEDULED_IMAGE_MAX_WIDTH"},
		{name: "width above ceiling", override: "SCHEDULED_IMAGE_MAX_WIDTH=8193\n", wantKey: "SCHEDULED_IMAGE_MAX_WIDTH"},
		{name: "pixels above ceiling", override: "SCHEDULED_IMAGE_MAX_PIXELS=16000001\n", wantKey: "SCHEDULED_IMAGE_MAX_PIXELS"},
		{name: "concurrency above ceiling", override: "SCHEDULED_IMAGE_MAX_CONCURRENT_DECODES=2\n", wantKey: "SCHEDULED_IMAGE_MAX_CONCURRENT_DECODES"},
		{name: "zero admission wait", override: "SCHEDULED_IMAGE_ADMISSION_WAIT=0s\n", wantKey: "SCHEDULED_IMAGE_ADMISSION_WAIT"},
		{name: "admission wait above ceiling", override: "SCHEDULED_IMAGE_ADMISSION_WAIT=250ms1ns\n", wantKey: "SCHEDULED_IMAGE_ADMISSION_WAIT"},
		{name: "pixels exceed configured geometry", override: "SCHEDULED_IMAGE_MAX_WIDTH=100\nSCHEDULED_IMAGE_MAX_HEIGHT=100\nSCHEDULED_IMAGE_MAX_PIXELS=10001\n", wantKey: "SCHEDULED_IMAGE_MAX_PIXELS"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.wantKey) {
				t.Fatalf("LoadConfig error = %v, want %s", err, test.wantKey)
			}
		})
	}
}

func TestLoadConfig_OAuthCustomValues(t *testing.T) {
	// Localhost mode accepts only its callback, scopes, and session budgets.
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	t.Setenv("OAUTH_CALLBACK_URL", "http://127.0.0.1:19090/oauth/callback")
	t.Setenv("OAUTH_SCOPES", "atproto transition:chat.bsky")
	t.Setenv("OAUTH_SESSION_ABSOLUTE_LIFETIME", "720h")
	t.Setenv("CRAFTSKY_SESSION_INACTIVITY", "48h")
	t.Setenv("OAUTH_AUTH_REQUEST_EXPIRY", "15m")
	t.Setenv("CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL", "2m")

	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.OAuth.CallbackURL.String(); got != "http://127.0.0.1:19090/oauth/callback" {
		t.Errorf("OAuth.CallbackURL = %q", got)
	}
	wantScopes := []string{"atproto", "transition:chat.bsky"}
	if len(cfg.OAuth.Scopes) != len(wantScopes) {
		t.Errorf("OAuth.Scopes = %v, want %v", cfg.OAuth.Scopes, wantScopes)
	} else {
		for i, s := range wantScopes {
			if cfg.OAuth.Scopes[i] != s {
				t.Errorf("OAuth.Scopes[%d] = %q, want %q", i, cfg.OAuth.Scopes[i], s)
			}
		}
	}
	if cfg.OAuthSessionAbsoluteLifetime != 720*time.Hour {
		t.Errorf("OAuthSessionAbsoluteLifetime = %v", cfg.OAuthSessionAbsoluteLifetime)
	}
	if cfg.CraftskySessionInactivity != 48*time.Hour {
		t.Errorf("CraftskySessionInactivity = %v", cfg.CraftskySessionInactivity)
	}
	if cfg.OAuthAuthRequestExpiry != 15*time.Minute {
		t.Errorf("OAuthAuthRequestExpiry = %v", cfg.OAuthAuthRequestExpiry)
	}
	if cfg.CraftskySessionActivityWriteInterval != 2*time.Minute {
		t.Errorf("CraftskySessionActivityWriteInterval = %v", cfg.CraftskySessionActivityWriteInterval)
	}
}
