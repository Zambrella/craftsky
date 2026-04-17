// Package routes wires the App View's HTTP routes onto a *http.ServeMux.
// Each handler factory in internal/api takes only the specific deps it
// needs; this package owns the mapping from URL → handler + middleware.
package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
)

// AddRoutes registers all App View routes on mux.
//
// ctx is the startup-scope context (used by future route-time validation,
// e.g. checking that a required table exists at boot). Per-request work
// inside handlers uses r.Context(), not this ctx.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))

	// Authenticated.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	mux.Handle("GET /whoami", authN(api.WhoAmIHandler()))

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
