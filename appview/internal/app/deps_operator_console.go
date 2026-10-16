package app

import (
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/operatorconsole"
	"strings"
	"time"
)

func newOperatorConsole(pool *pgxpool.Pool, federated *federatedClients, cfg Config, logger *slog.Logger) (*operatorconsole.Handlers, error) {
	if !cfg.ModerationAdminEnabled {
		return nil, nil
	}
	origin := cfg.OAuth.PublicOrigin.String()
	if cfg.Env == EnvDev {
		u := cfg.OAuth.CallbackURL
		u.Path = ""
		origin = strings.TrimSuffix(u.String(), "/")
	}
	endpoint := func(path string) url.URL { u, _ := url.Parse(origin + path); return *u }
	input := auth.ClientConfigInput{Purpose: auth.ClientPurposeOperator, Mode: auth.ClientMode(cfg.OAuth.Mode), CallbackURL: endpoint("/oauth/admin/callback"), Scopes: []string{"atproto"}}
	if auth.ClientMode(cfg.OAuth.Mode) == auth.ClientModeConfidential {
		input.ClientID = endpoint("/oauth/admin/client-metadata.json")
		input.JWKSURL = endpoint("/oauth/admin/jwks.json")
		input.ClientSecretKey = cfg.OAuth.ClientSecretKey.Reveal()
		input.ClientKeyID = cfg.OAuth.ClientKeyID
	}
	artifacts, err := auth.BuildClientArtifacts(input)
	if err != nil {
		return nil, err
	}
	app := oauth.NewClientApp(&artifacts.Config, nil)
	app.Client = federated.oauth
	app.Resolver.Client = federated.metadata
	app.Dir = federated.authoritativeDirectory
	return operatorconsole.NewHandlers(operatorconsole.NewStore(pool, time.Now), app, origin, logger)
}
