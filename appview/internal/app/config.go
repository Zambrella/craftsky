// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

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
	DevDID         string // populated in dev only; empty in prod

	TapWSURL        string
	TapAckTimeout   time.Duration
	TapReconnectMax time.Duration
	TapMaxRetries   int

	// OAuth-related.
	OAuthHostname                   string        // empty in dev (localhost mode)
	OAuthClientSecretKey            string        // multibase-encoded P-256 private key; empty in dev
	OAuthClientKeyID                string        // default "primary"
	OAuthScopes                     []string      // default ["atproto", "transition:generic"]
	OAuthSessionExpiry              time.Duration // default 180d
	OAuthSessionInactivity          time.Duration // default 30d
	OAuthAuthRequestExpiry          time.Duration // default 30m
	CraftskySessionLastSeenThrottle time.Duration // default 5m
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
	if cfg.TapAckTimeout, err = durationEnv("TAP_ACK_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.TapReconnectMax, err = durationEnv("TAP_RECONNECT_MAX", 30*time.Second); err != nil {
		return Config{}, err
	}

	maxRetries := os.Getenv("TAP_MAX_RETRIES")
	if maxRetries == "" {
		cfg.TapMaxRetries = 5
	} else {
		n, err := strconv.Atoi(maxRetries)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("TAP_MAX_RETRIES: must be non-negative integer, got %q", maxRetries)
		}
		cfg.TapMaxRetries = n
	}

	cfg.OAuthHostname = os.Getenv("OAUTH_HOSTNAME")
	cfg.OAuthClientSecretKey = os.Getenv("OAUTH_CLIENT_SECRET_KEY")
	cfg.OAuthClientKeyID = getEnvWithDefault("OAUTH_CLIENT_SECRET_KEY_ID", "primary")

	scopesStr := getEnvWithDefault("OAUTH_SCOPES", "atproto transition:generic")
	cfg.OAuthScopes = strings.Fields(scopesStr)

	if cfg.OAuthSessionExpiry, err = durationEnv("OAUTH_SESSION_EXPIRY", 180*24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.OAuthSessionInactivity, err = durationEnv("OAUTH_SESSION_INACTIVITY", 30*24*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.OAuthAuthRequestExpiry, err = durationEnv("OAUTH_AUTH_REQUEST_EXPIRY", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.CraftskySessionLastSeenThrottle, err = durationEnv("CRAFTSKY_SESSION_LAST_SEEN_THROTTLE", 5*time.Minute); err != nil {
		return Config{}, err
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
		cfg.DevDID = ""
	}

	// Prod validation: if hostname is set, confidential client requires the key.
	if cfg.Env == EnvProd && cfg.OAuthHostname != "" && cfg.OAuthClientSecretKey == "" {
		return Config{}, fmt.Errorf("OAUTH_CLIENT_SECRET_KEY is required in prod when OAUTH_HOSTNAME is set")
	}

	return cfg, nil
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
