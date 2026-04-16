// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import (
	"fmt"
	"os"
	"strings"

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

	// Required everywhere.
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_ORIGINS is required (comma-separated list)")
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

	return cfg, nil
}
