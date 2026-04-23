package routes

import (
	"context"
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// AddRoutes registers all App View routes on mux.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public ops.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))
	mux.Handle("GET /healthz", api.NewHealthHandler(deps.DB, deps.Consumer))

	// Temporary factory; overwritten in Chunk 10 with a real indigo-backed one.
	placeholderPDS := func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return nil, errors.New("PDS client not wired; routes.go placeholder")
	}
	// OAuth discovery endpoints (contracts with the AS; not versioned).
	oauthHandlers := auth.NewHTTPHandlers(
		deps.OAuthApp,
		deps.CraftskySessionStore,
		deps.DB,
		deps.Logger,
		deps.Config.Env == app.EnvDev,
		placeholderPDS,
	)
	mux.Handle("GET /oauth/client-metadata.json", oauthHandlers.ClientMetadataHandler())
	mux.Handle("GET /oauth/jwks.json", oauthHandlers.JWKSHandler())
	mux.Handle("GET /oauth/callback", oauthHandlers.CallbackHandler())

	// Middleware stacks.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	deviceID := middleware.DeviceID(deps.CraftskySessionStore, deps.Logger)

	// v1 — unauthenticated but device-id required.
	mux.Handle("POST /v1/auth/login", deviceID(oauthHandlers.LoginHandler()))

	// v1 — authenticated + device-id required.
	mux.Handle("GET /v1/whoami", authN(deviceID(api.WhoAmIHandler(deps.HandleResolver, deps.Logger))))
	mux.Handle("POST /v1/auth/logout", authN(deviceID(oauthHandlers.LogoutHandler())))

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
