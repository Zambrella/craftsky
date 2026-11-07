package routes

import (
	"log/slog"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/video"
)

type videoRouteBundle struct {
	mux           Registrar
	middleware    v1Middleware
	authorization api.VideoUploadAuthorizationIssuer
	limits        api.VideoUploadLimitsService
	logger        *slog.Logger
	observer      api.VideoOperationObserver
	enabled       bool
	bunny         *video.Bunny
}

func registerVideoRoutes(routes videoRouteBundle) {
	cancelHandler := api.BunnyVideoCancelHandler(routes.bunny)
	if !routes.enabled {
		cancelHandler = api.VideoLaunchDisabledHandler()
	}
	routes.mux.Handle("POST /v1/blobs/videos/{jobId}/cancel", routes.middleware.wrap(mustPolicy("POST", "/v1/blobs/videos/{jobId}/cancel"), cancelHandler))
	statusHandler := api.BunnyVideoStatusHandler(routes.bunny)
	if !routes.enabled {
		statusHandler = api.VideoLaunchDisabledHandler()
	}
	routes.mux.Handle("POST /v1/blobs/videos/{jobId}/status", routes.middleware.wrap(mustPolicy("POST", "/v1/blobs/videos/{jobId}/status"), statusHandler))
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
