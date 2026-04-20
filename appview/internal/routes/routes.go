// Package routes wires the App View's HTTP routes onto a *http.ServeMux.
// Each handler factory in internal/api takes only the specific deps it
// needs; this package owns the mapping from URL → handler + middleware.
package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testpipeline"
)

// AddRoutes registers all App View routes on mux.
//
// ctx is the startup-scope context (used by future route-time validation,
// e.g. checking that a required table exists at boot). Per-request work
// inside handlers uses r.Context(), not this ctx.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))
	mux.Handle("GET /healthz", api.NewHealthHandler(deps.DB, deps.Consumer))

	// OAuth discovery endpoints.
	oauthHandlers := auth.NewHTTPHandlers(
		deps.OAuthApp,
		deps.CraftskySessionStore,
		deps.DB,
		deps.Logger,
		deps.Config.Env == app.EnvDev,
	)
	mux.Handle("GET /oauth/client-metadata.json", oauthHandlers.ClientMetadataHandler())
	mux.Handle("GET /oauth/jwks.json", oauthHandlers.JWKSHandler())
	mux.Handle("POST /auth/login", oauthHandlers.LoginHandler())
	mux.Handle("GET /oauth/callback", oauthHandlers.CallbackHandler())

	// Authenticated.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	mux.Handle("GET /whoami", authN(api.WhoAmIHandler()))
	mux.Handle("POST /auth/logout", authN(oauthHandlers.LogoutHandler()))

	// Disposable test pipeline (GET /test/feed). Dev only — see
	// docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
	if deps.Config.Env == app.EnvDev {
		mux.Handle("GET /test/feed", testpipeline.NewHandler(deps.DB))
	}

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
