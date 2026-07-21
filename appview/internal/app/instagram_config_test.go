package app

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	instagramTestHMACKey   = "synthetic-hmac-key-32-bytes-long!!"
	instagramTestAppSecret = "synthetic-app-secret-32-bytes-long!"
	instagramTestVerify    = "synthetic-verify-token-32-bytes!!"
	instagramTestAccess    = "synthetic-access-token-32-bytes!!"
	instagramTestAccountID = "17841400000000000"
)

func TestLoadConfig_InstagramModesAndRedactedOptions(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		cfg, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, ""))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.InstagramData.Available() {
			t.Fatal("InstagramData.Available() = true, want false")
		}
		if cfg.InstagramMeta.Enabled() || cfg.InstagramMeta.Configured() {
			t.Fatalf("InstagramMeta = %v, want disabled and unconfigured", cfg.InstagramMeta)
		}
	})

	t.Run("private data can remain available while Meta is disabled", func(t *testing.T) {
		cfg, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd,
			"INSTAGRAM_DATA_HMAC_KEY="+instagramTestHMACKey+"\n"))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !cfg.InstagramData.Available() {
			t.Fatal("InstagramData.Available() = false, want true")
		}
		if cfg.InstagramMeta.Enabled() || cfg.InstagramMeta.Configured() {
			t.Fatalf("InstagramMeta = %v, want disabled and unconfigured", cfg.InstagramMeta)
		}
		key := cfg.InstagramData.HMACKey()
		if got := string(key); got != instagramTestHMACKey {
			t.Fatalf("HMACKey() = %q, want configured key", got)
		}
		key[0] = 'X'
		if got := string(cfg.InstagramData.HMACKey()); got != instagramTestHMACKey {
			t.Fatal("HMACKey returned mutable configuration storage")
		}
	})

	t.Run("complete Meta bundle enables verification", func(t *testing.T) {
		cfg, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, instagramFullBundle()))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if !cfg.InstagramData.Available() || !cfg.InstagramMeta.Configured() || !cfg.InstagramMeta.Enabled() {
			t.Fatalf("Instagram config = data %v meta %v, want full mode", cfg.InstagramData, cfg.InstagramMeta)
		}
		if got := cfg.InstagramMeta.InstagramAccountID(); got != instagramTestAccountID {
			t.Fatalf("InstagramAccountID() = %q", got)
		}
		if got := cfg.InstagramMeta.AppSecret(); got != instagramTestAppSecret {
			t.Fatalf("AppSecret() = %q", got)
		}
		if got := cfg.InstagramMeta.VerifyToken(); got != instagramTestVerify {
			t.Fatalf("VerifyToken() = %q", got)
		}
		if got := cfg.InstagramMeta.AccessToken(); got != instagramTestAccess {
			t.Fatalf("AccessToken() = %q", got)
		}
		if cfg.InstagramMeta.RepliesEnabled() {
			t.Fatal("RepliesEnabled() = true, want explicit false")
		}

		gotURL, err := cfg.InstagramMeta.GraphAPIURL(instagramTestAccountID, "messages")
		if err != nil {
			t.Fatalf("GraphAPIURL: %v", err)
		}
		wantURL := "https://graph.instagram.com/v23.0/17841400000000000/messages"
		if gotURL != wantURL {
			t.Fatalf("GraphAPIURL = %q, want %q", gotURL, wantURL)
		}

		diagnostic := fmt.Sprintf("%v %+v %#v full=%+v", cfg.InstagramData, cfg.InstagramMeta, cfg.InstagramMeta, cfg)
		for _, secret := range []string{
			instagramTestHMACKey,
			instagramTestAppSecret,
			instagramTestVerify,
			instagramTestAccess,
			instagramTestAccountID,
			"direct/t/123456789",
		} {
			if strings.Contains(diagnostic, secret) {
				t.Fatalf("formatted Instagram config leaked %q: %s", secret, diagnostic)
			}
		}
	})
}

func TestLoadConfig_InstagramRejectsPartialOrUnsafeProductionConfig(t *testing.T) {
	full := instagramFullBundle()
	tests := []struct {
		name       string
		config     string
		wantErrKey string
	}{
		{
			name:       "enabled without data key",
			config:     strings.Replace(full, "INSTAGRAM_DATA_HMAC_KEY="+instagramTestHMACKey+"\n", "", 1),
			wantErrKey: "INSTAGRAM_DATA_HMAC_KEY",
		},
		{
			name:       "enabled missing app secret",
			config:     strings.Replace(full, "INSTAGRAM_META_APP_SECRET="+instagramTestAppSecret+"\n", "", 1),
			wantErrKey: "INSTAGRAM_META_APP_SECRET",
		},
		{
			name:       "enabled missing verify token",
			config:     strings.Replace(full, "INSTAGRAM_META_VERIFY_TOKEN="+instagramTestVerify+"\n", "", 1),
			wantErrKey: "INSTAGRAM_META_VERIFY_TOKEN",
		},
		{
			name:       "enabled missing access token",
			config:     strings.Replace(full, "INSTAGRAM_META_ACCESS_TOKEN="+instagramTestAccess+"\n", "", 1),
			wantErrKey: "INSTAGRAM_META_ACCESS_TOKEN",
		},
		{
			name:       "enabled missing account",
			config:     strings.Replace(full, "INSTAGRAM_META_ACCOUNT_ID="+instagramTestAccountID+"\n", "", 1),
			wantErrKey: "INSTAGRAM_META_ACCOUNT_ID",
		},
		{
			name:       "enabled missing API version",
			config:     strings.Replace(full, "INSTAGRAM_META_API_VERSION=v23.0\n", "", 1),
			wantErrKey: "INSTAGRAM_META_API_VERSION",
		},
		{
			name:       "enabled missing API base URL",
			config:     strings.Replace(full, "INSTAGRAM_META_API_BASE_URL=https://graph.instagram.com\n", "", 1),
			wantErrKey: "INSTAGRAM_META_API_BASE_URL",
		},
		{
			name:       "enabled missing DM URL",
			config:     strings.Replace(full, "INSTAGRAM_META_DM_URL=https://www.instagram.com/direct/t/123456789/\n", "", 1),
			wantErrKey: "INSTAGRAM_META_DM_URL",
		},
		{
			name:       "enabled missing reply capability",
			config:     strings.Replace(full, "INSTAGRAM_META_REPLIES_ENABLED=false\n", "", 1),
			wantErrKey: "INSTAGRAM_META_REPLIES_ENABLED",
		},
		{
			name:       "partial bundle while disabled",
			config:     "INSTAGRAM_DATA_HMAC_KEY=" + instagramTestHMACKey + "\nINSTAGRAM_META_APP_SECRET=" + instagramTestAppSecret + "\n",
			wantErrKey: "partial Instagram Meta configuration",
		},
		{
			name:       "short data key",
			config:     "INSTAGRAM_DATA_HMAC_KEY=too-short\n",
			wantErrKey: "INSTAGRAM_DATA_HMAC_KEY",
		},
		{
			name:       "placeholder secret",
			config:     strings.Replace(full, instagramTestAppSecret, "CHANGE_ME_CHANGE_ME_CHANGE_ME_1234", 1),
			wantErrKey: "INSTAGRAM_META_APP_SECRET",
		},
		{
			name:       "non-numeric account ID",
			config:     strings.Replace(full, instagramTestAccountID, "not-an-account", 1),
			wantErrKey: "INSTAGRAM_META_ACCOUNT_ID",
		},
		{
			name:       "unversioned API",
			config:     strings.Replace(full, "v23.0", "latest", 1),
			wantErrKey: "INSTAGRAM_META_API_VERSION",
		},
		{
			name:       "non-HTTPS API base",
			config:     strings.Replace(full, "https://graph.instagram.com", "http://graph.instagram.com", 1),
			wantErrKey: "INSTAGRAM_META_API_BASE_URL",
		},
		{
			name:       "API base contains version path",
			config:     strings.Replace(full, "https://graph.instagram.com", "https://graph.instagram.com/v23.0", 1),
			wantErrKey: "INSTAGRAM_META_API_BASE_URL",
		},
		{
			name:       "non-HTTPS DM URL",
			config:     strings.Replace(full, "https://www.instagram.com/direct/t/123456789/", "http://www.instagram.com/direct/t/123456789/", 1),
			wantErrKey: "INSTAGRAM_META_DM_URL",
		},
		{
			name:       "unsafe multi-replica limiting",
			config:     full + "APPVIEW_REPLICA_COUNT=2\n",
			wantErrKey: "INSTAGRAM_SHARED_RATE_LIMITS",
		},
		{
			name:       "invalid trusted proxy CIDR",
			config:     full + "INSTAGRAM_TRUSTED_PROXY_CIDRS=not-a-cidr\n",
			wantErrKey: "INSTAGRAM_TRUSTED_PROXY_CIDRS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, tt.config))
			if err == nil || !strings.Contains(err.Error(), tt.wantErrKey) {
				t.Fatalf("LoadConfig error = %v, want key %q", err, tt.wantErrKey)
			}
		})
	}
}

func TestLoadConfig_InstagramLimitsDefaultToHardMaximaAndMayOnlyTighten(t *testing.T) {
	cfg, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, ""))
	if err != nil {
		t.Fatalf("LoadConfig defaults: %v", err)
	}
	want := InstagramLimits{
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
	if cfg.InstagramLimits != want {
		t.Fatalf("InstagramLimits = %+v, want %+v", cfg.InstagramLimits, want)
	}

	tightened := strings.Join([]string{
		"INSTAGRAM_CHALLENGE_TTL=5m",
		"INSTAGRAM_WEBHOOK_BODY_LIMIT_BYTES=131072",
		"INSTAGRAM_WEBHOOK_MAX_EVENTS=50",
		"INSTAGRAM_META_HTTP_TIMEOUT=3s",
		"INSTAGRAM_META_RESPONSE_LIMIT_BYTES=32768",
		"INSTAGRAM_META_LOOKUP_CONCURRENCY=10",
		"INSTAGRAM_WORKER_CONCURRENCY=2",
		"INSTAGRAM_WORKER_LEASE_DURATION=30s",
		"INSTAGRAM_WORKER_MAX_ATTEMPTS=3",
		"INSTAGRAM_WORKER_MAX_PROCESSING_AGE=10m",
		"INSTAGRAM_IMPORT_MAX_ENTRIES=5000",
		"INSTAGRAM_PAGE_MAX=25",
		"INSTAGRAM_PAGE_DEFAULT=10",
		"INSTAGRAM_OPERATOR_BATCH_MAX=250",
	}, "\n") + "\n"
	cfg, err = LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, tightened))
	if err != nil {
		t.Fatalf("LoadConfig tightened limits: %v", err)
	}
	if cfg.InstagramLimits.WebhookBodyLimitBytes != 128*1024 ||
		cfg.InstagramLimits.MetaHTTPTimeout != 3*time.Second ||
		cfg.InstagramLimits.WorkerConcurrency != 2 ||
		cfg.InstagramLimits.PageDefault != 10 || cfg.InstagramLimits.PageMax != 25 {
		t.Fatalf("tightened limits not applied: %+v", cfg.InstagramLimits)
	}

	tests := []struct {
		setting string
		key     string
	}{
		{"INSTAGRAM_CHALLENGE_TTL=10m1ns\n", "INSTAGRAM_CHALLENGE_TTL"},
		{"INSTAGRAM_WEBHOOK_BODY_LIMIT_BYTES=262145\n", "INSTAGRAM_WEBHOOK_BODY_LIMIT_BYTES"},
		{"INSTAGRAM_WEBHOOK_MAX_EVENTS=101\n", "INSTAGRAM_WEBHOOK_MAX_EVENTS"},
		{"INSTAGRAM_META_HTTP_TIMEOUT=5s1ns\n", "INSTAGRAM_META_HTTP_TIMEOUT"},
		{"INSTAGRAM_META_RESPONSE_LIMIT_BYTES=65537\n", "INSTAGRAM_META_RESPONSE_LIMIT_BYTES"},
		{"INSTAGRAM_META_LOOKUP_CONCURRENCY=21\n", "INSTAGRAM_META_LOOKUP_CONCURRENCY"},
		{"INSTAGRAM_WORKER_CONCURRENCY=5\n", "INSTAGRAM_WORKER_CONCURRENCY"},
		{"INSTAGRAM_WORKER_LEASE_DURATION=60s1ns\n", "INSTAGRAM_WORKER_LEASE_DURATION"},
		{"INSTAGRAM_WORKER_MAX_ATTEMPTS=6\n", "INSTAGRAM_WORKER_MAX_ATTEMPTS"},
		{"INSTAGRAM_WORKER_MAX_PROCESSING_AGE=15m1ns\n", "INSTAGRAM_WORKER_MAX_PROCESSING_AGE"},
		{"INSTAGRAM_IMPORT_MAX_ENTRIES=10001\n", "INSTAGRAM_IMPORT_MAX_ENTRIES"},
		{"INSTAGRAM_PAGE_MAX=51\n", "INSTAGRAM_PAGE_MAX"},
		{"INSTAGRAM_OPERATOR_BATCH_MAX=501\n", "INSTAGRAM_OPERATOR_BATCH_MAX"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			_, err := LoadConfig(EnvProd, instagramConfigFile(t, EnvProd, tt.setting))
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("LoadConfig error = %v, want key %s", err, tt.key)
			}
		})
	}
}

func instagramConfigFile(t *testing.T, env Env, extra string) string {
	t.Helper()
	base := "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://craftsky.social\nTAP_WS_URL=ws://tap\n"
	if env == EnvDev {
		base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	}
	return testConfigFile(t, base+extra)
}

func instagramFullBundle() string {
	return strings.Join([]string{
		"INSTAGRAM_DATA_HMAC_KEY=" + instagramTestHMACKey,
		"INSTAGRAM_META_ENABLED=true",
		"INSTAGRAM_META_APP_SECRET=" + instagramTestAppSecret,
		"INSTAGRAM_META_VERIFY_TOKEN=" + instagramTestVerify,
		"INSTAGRAM_META_ACCESS_TOKEN=" + instagramTestAccess,
		"INSTAGRAM_META_ACCOUNT_ID=" + instagramTestAccountID,
		"INSTAGRAM_META_API_VERSION=v23.0",
		"INSTAGRAM_META_API_BASE_URL=https://graph.instagram.com",
		"INSTAGRAM_META_DM_URL=https://www.instagram.com/direct/t/123456789/",
		"INSTAGRAM_META_REPLIES_ENABLED=false",
	}, "\n") + "\n"
}
