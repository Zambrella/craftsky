package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"social.craftsky/appview/internal/app"
)

// loadDeps is the boilerplate a subcommand runs when it genuinely needs
// a DB pool: parse --env, load config, build *app.Deps.
// Subcommands that don't need DB access (request, migrate, stubs) use
// parseEnvFlag + loadCfgLight instead, avoiding a wasted connect.
func loadDeps(ctx context.Context) (*app.Deps, func(), error) {
	env, err := app.ParseEnv(envFlag)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := app.LoadConfig(env, fmt.Sprintf("environments/%s.env", env))
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	switch env {
	case app.EnvDev:
		return app.NewDevDeps(ctx, cfg)
	case app.EnvProd:
		return app.NewProdDeps(ctx, cfg)
	default:
		return nil, nil, fmt.Errorf("unreachable: unknown env %q", env)
	}
}

// devEnvMarker is re-exposed so other cmd/cli files don't need their
// own internal/app import just to compare against EnvDev.
var devEnvMarker = app.EnvDev

// parseEnvFlag returns app.Env for the current --env flag value.
func parseEnvFlag() (app.Env, error) {
	return app.ParseEnv(envFlag)
}

// loadCfgLight loads Config without building Deps (no DB connection).
func loadCfgLight(env app.Env) (app.Config, error) {
	return app.LoadConfig(env, "environments/"+string(env)+".env")
}

// resolveBaseURL picks the URL `cli request` should hit.
//
//	Dev  → http://localhost:8080
//	Prod → $APPVIEW_BASE_URL, which must start with https://.
//
// A permissive http:// in prod is disallowed to prevent sending dev DIDs
// or future real tokens over cleartext.
func resolveBaseURL(env app.Env) (string, error) {
	if env == app.EnvDev {
		return "http://localhost:8080", nil
	}
	v := os.Getenv("APPVIEW_BASE_URL")
	if v == "" {
		return "", errors.New("APPVIEW_BASE_URL must be set to hit prod")
	}
	if !strings.HasPrefix(v, "https://") {
		return "", errors.New("APPVIEW_BASE_URL must use https:// in prod")
	}
	return v, nil
}
