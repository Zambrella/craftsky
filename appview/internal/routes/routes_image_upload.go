package routes

import "social.craftsky/appview/internal/api"

type imageUploadRouteBundle struct {
	mux           Registrar
	middleware    v1Middleware
	authorization api.ImageUploadAuthorizationIssuer
}

func registerImageUploadRoutes(routes imageUploadRouteBundle) {
	routes.mux.Handle("POST /v1/blobs/images/authorization", routes.middleware.wrap(mustPolicy("POST", "/v1/blobs/images/authorization"), api.ImageUploadAuthorizationHandler(routes.authorization)))
}
