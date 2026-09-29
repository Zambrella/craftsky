package routes

import (
	"log/slog"

	"social.craftsky/appview/internal/api"
)

type ageEligibilityRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	store      api.AgeEligibilityStatusReader
	logger     *slog.Logger
}

func registerAgeEligibilityRoutes(routes ageEligibilityRouteBundle) {
	routes.mux.Handle(
		"GET /v1/account/eligibility",
		routes.middleware.wrap(
			mustPolicy("GET", "/v1/account/eligibility"),
			api.AgeEligibilityStatusHandler(routes.store, routes.logger),
		),
	)
}
