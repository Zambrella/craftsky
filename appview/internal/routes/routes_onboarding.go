package routes

import (
	"log/slog"

	"social.craftsky/appview/internal/api"
)

type onboardingRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	store      api.OnboardingStatusService
	logger     *slog.Logger
}

func registerOnboardingRoutes(routes onboardingRouteBundle) {
	routes.mux.Handle(
		"GET /v1/onboarding/status",
		routes.middleware.wrap(
			mustPolicy("GET", "/v1/onboarding/status"),
			api.GetOnboardingStatusHandler(routes.store, routes.logger),
		),
	)
	routes.mux.Handle(
		"POST /v1/onboarding/completion",
		routes.middleware.wrap(
			mustPolicy("POST", "/v1/onboarding/completion"),
			api.CompleteOnboardingHandler(routes.store, routes.logger),
		),
	)
}
