package routes

import (
	"log/slog"

	"social.craftsky/appview/internal/api"
)

type videoRouteBundle struct {
	mux           Registrar
	middleware    v1Middleware
	authorization api.VideoUploadAuthorizationIssuer
	limits        api.VideoUploadLimitsService
	logger        *slog.Logger
	observer      api.VideoOperationObserver
	enabled       bool
}

func registerVideoRoutes(routes videoRouteBundle) {
	authorizationHandler := api.VideoUploadAuthorizationHandler(routes.authorization, routes.logger, routes.observer)
	limitsHandler := api.VideoUploadLimitsHandler(routes.limits, routes.logger, routes.observer)
	if !routes.enabled {
		authorizationHandler = api.VideoLaunchDisabledHandler()
		limitsHandler = api.VideoLaunchDisabledHandler()
	}
	routes.mux.Handle(
		"POST /v1/blobs/videos/authorization",
		routes.middleware.wrap(
			mustPolicy("POST", "/v1/blobs/videos/authorization"),
			authorizationHandler,
		),
	)
	routes.mux.Handle(
		"GET /v1/blobs/videos/limits",
		routes.middleware.wrap(
			mustPolicy("GET", "/v1/blobs/videos/limits"),
			limitsHandler,
		),
	)
}
