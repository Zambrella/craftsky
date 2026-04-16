package main

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/routes"
)

// NewServer constructs the App View's HTTP handler. main.go wraps it in
// a *http.Server; this function stays focused on routing and middleware.
//
// Middleware stack (outside-in):
//
//	Logging  (assigns run_id, logs every request)
//	CORS     (origin check, preflight handling)
//	mux      (routing — Authenticated is applied per-route)
func NewServer(ctx context.Context, deps *app.Deps) http.Handler {
	mux := http.NewServeMux()
	routes.AddRoutes(ctx, mux, deps)

	var h http.Handler = mux
	h = middleware.CORS(deps.Config.AllowedOrigins)(h)
	h = middleware.Logging(deps.Logger)(h)
	return h
}
