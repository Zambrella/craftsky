package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

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

// testConfigFile writes a temporary .env-style file and returns its path.
// It also unsets the relevant env vars before the test so godotenv.Load
// will actually pick up the file's values. Setting them to "" instead of
// unsetting would leave them "present" from godotenv's perspective, which
// would cause Load to skip the file's value — the opposite of what we want.
func testConfigFile(t *testing.T, contents string) string {
	t.Helper()
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID",
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES",
		"OAUTH_HOSTNAME", "OAUTH_CLIENT_SECRET_KEY", "OAUTH_CLIENT_SECRET_KEY_ID",
		"OAUTH_SCOPES", "OAUTH_SESSION_EXPIRY", "OAUTH_SESSION_INACTIVITY",
		"OAUTH_AUTH_REQUEST_EXPIRY", "CRAFTSKY_SESSION_LAST_SEEN_THROTTLE",
		"MAX_POST_IMAGES", "MAX_IMAGE_UPLOAD_BYTES", "APPVIEW_JSON_BODY_LIMIT_BYTES",
		"APPVIEW_ENABLE_DEV_MODERATION",
		"APPVIEW_DEV_MODERATION_TOKEN", "CRAFTSKY_DEV_LABELER_DID",
		"APPVIEW_TRUSTED_MODERATION_SOURCE_DIDS"} {
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

func TestLoadConfig_ProdClearsDevModerationFields(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example\nTAP_WS_URL=ws://tap:2480/channel\nAPPVIEW_ENABLE_DEV_MODERATION=true\nAPPVIEW_DEV_MODERATION_TOKEN=secret-token\nCRAFTSKY_DEV_LABELER_DID=did:plc:labeler\nAPPVIEW_TRUSTED_MODERATION_SOURCE_DIDS=did:plc:ozone\n")
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
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example,https://b.example\nTAP_WS_URL=ws://tap:2480/channel\n")
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
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=*\nTAP_WS_URL=ws://tap:2480/channel\n")
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
	path := testConfigFile(t, "DATABASE_URL=postgres://p\nALLOWED_ORIGINS=https://a.example\nCRAFTSKY_DEV_DID=did:plc:leaked\nTAP_WS_URL=ws://tap:2480/channel\n")
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
		"TAP_MAX_RETRIES=3\n"
	if err := os.WriteFile(envPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	// Clear env so file wins. godotenv.Load skips keys already set in env
	// (including to ""), so we unset rather than set-empty.
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID",
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES",
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
	if cfg.TapMaxRetries != 3 {
		t.Errorf("TapMaxRetries = %d", cfg.TapMaxRetries)
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
	if cfg.TapMaxRetries != 5 {
		t.Errorf("default TapMaxRetries = %d", cfg.TapMaxRetries)
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
	if cfg.OAuthHostname != "" {
		t.Errorf("OAuthHostname = %q, want empty", cfg.OAuthHostname)
	}
	want := []string{"atproto", "transition:generic"}
	if len(cfg.OAuthScopes) != len(want) {
		t.Errorf("OAuthScopes = %v, want %v", cfg.OAuthScopes, want)
	} else {
		for i, s := range want {
			if cfg.OAuthScopes[i] != s {
				t.Errorf("OAuthScopes[%d] = %q, want %q", i, cfg.OAuthScopes[i], s)
			}
		}
	}
	if cfg.OAuthSessionExpiry != 180*24*time.Hour {
		t.Errorf("OAuthSessionExpiry = %v, want %v", cfg.OAuthSessionExpiry, 180*24*time.Hour)
	}
	if cfg.OAuthSessionInactivity != 30*24*time.Hour {
		t.Errorf("OAuthSessionInactivity = %v, want %v", cfg.OAuthSessionInactivity, 30*24*time.Hour)
	}
	if cfg.OAuthAuthRequestExpiry != 30*time.Minute {
		t.Errorf("OAuthAuthRequestExpiry = %v, want %v", cfg.OAuthAuthRequestExpiry, 30*time.Minute)
	}
	if cfg.CraftskySessionLastSeenThrottle != 5*time.Minute {
		t.Errorf("CraftskySessionLastSeenThrottle = %v, want %v", cfg.CraftskySessionLastSeenThrottle, 5*time.Minute)
	}
	if cfg.OAuthClientKeyID != "primary" {
		t.Errorf("OAuthClientKeyID = %q, want %q", cfg.OAuthClientKeyID, "primary")
	}
}

func TestLoadConfig_OAuthRequiredInProd(t *testing.T) {
	// env = prod, OAUTH_HOSTNAME set, OAUTH_CLIENT_SECRET_KEY unset.
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example\nTAP_WS_URL=ws://tap:2480/channel\n")
	t.Setenv("OAUTH_HOSTNAME", "https://craftsky.social")
	_, err := LoadConfig(EnvProd, path)
	if err == nil {
		t.Fatal("expected error when OAUTH_CLIENT_SECRET_KEY is unset in prod with OAUTH_HOSTNAME set")
	}
	if !strings.Contains(err.Error(), "OAUTH_CLIENT_SECRET_KEY") {
		t.Errorf("error should mention OAUTH_CLIENT_SECRET_KEY, got: %v", err)
	}
}

func TestLoadConfig_OAuthCustomValues(t *testing.T) {
	// Set all OAUTH_* vars to non-default values and assert each lands on cfg.
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap:2480/channel\n")
	t.Setenv("OAUTH_HOSTNAME", "https://craftsky.example")
	t.Setenv("OAUTH_CLIENT_SECRET_KEY", "zQ3shtest...")
	t.Setenv("OAUTH_CLIENT_SECRET_KEY_ID", "secondary")
	t.Setenv("OAUTH_SCOPES", "atproto transition:chat.bsky")
	t.Setenv("OAUTH_SESSION_EXPIRY", "720h")
	t.Setenv("OAUTH_SESSION_INACTIVITY", "48h")
	t.Setenv("OAUTH_AUTH_REQUEST_EXPIRY", "15m")
	t.Setenv("CRAFTSKY_SESSION_LAST_SEEN_THROTTLE", "2m")

	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.OAuthHostname != "https://craftsky.example" {
		t.Errorf("OAuthHostname = %q", cfg.OAuthHostname)
	}
	if cfg.OAuthClientSecretKey != "zQ3shtest..." {
		t.Errorf("OAuthClientSecretKey = %q", cfg.OAuthClientSecretKey)
	}
	if cfg.OAuthClientKeyID != "secondary" {
		t.Errorf("OAuthClientKeyID = %q", cfg.OAuthClientKeyID)
	}
	wantScopes := []string{"atproto", "transition:chat.bsky"}
	if len(cfg.OAuthScopes) != len(wantScopes) {
		t.Errorf("OAuthScopes = %v, want %v", cfg.OAuthScopes, wantScopes)
	} else {
		for i, s := range wantScopes {
			if cfg.OAuthScopes[i] != s {
				t.Errorf("OAuthScopes[%d] = %q, want %q", i, cfg.OAuthScopes[i], s)
			}
		}
	}
	if cfg.OAuthSessionExpiry != 720*time.Hour {
		t.Errorf("OAuthSessionExpiry = %v", cfg.OAuthSessionExpiry)
	}
	if cfg.OAuthSessionInactivity != 48*time.Hour {
		t.Errorf("OAuthSessionInactivity = %v", cfg.OAuthSessionInactivity)
	}
	if cfg.OAuthAuthRequestExpiry != 15*time.Minute {
		t.Errorf("OAuthAuthRequestExpiry = %v", cfg.OAuthAuthRequestExpiry)
	}
	if cfg.CraftskySessionLastSeenThrottle != 2*time.Minute {
		t.Errorf("CraftskySessionLastSeenThrottle = %v", cfg.CraftskySessionLastSeenThrottle)
	}
}
