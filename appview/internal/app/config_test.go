package app

import (
	"os"
	"strings"
	"testing"
	"time"
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
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES"} {
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
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES"} {
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
	dir := t.TempDir()
	envPath := dir + "/test.env"
	contents := "DATABASE_URL=postgres://x\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	if err := os.WriteFile(envPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES"} {
		t.Setenv(k, "")
	}

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
