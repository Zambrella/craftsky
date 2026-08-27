package routes

import (
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/observability"
)

type linkPreviewRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	service    api.LinkPreviewService
	enabled    bool
	observer   *observability.Observer
}

func registerLinkPreviewRoute(bundle linkPreviewRouteBundle) {
	policy := mustPolicy("POST", "/v1/link-previews")
	bundle.mux.Handle("POST /v1/link-previews", bundle.middleware.wrap(
		policy,
		api.LinkPreviewHandler(bundle.service, bundle.enabled, bundle.observer),
	))
}
