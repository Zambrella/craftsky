package routes

import (
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/subscriptions"
)

type subscriptionRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	store      *subscriptions.Store
	now        func() time.Time
}

func registerSubscriptionRoutes(routes subscriptionRouteBundle) {
	now := routes.now
	if now == nil {
		now = time.Now
	}
	routes.mux.Handle("GET /v1/subscriptions/access", routes.middleware.wrap(
		mustPolicy("GET", "/v1/subscriptions/access"),
		api.GetSubscriptionAccessHandler(routes.store, now),
	))
	routes.mux.Handle("PUT /v1/billing/account", routes.middleware.wrap(
		mustPolicy("PUT", "/v1/billing/account"),
		api.EnsureBillingAccountHandler(routes.store, now),
	))
	routes.mux.Handle("GET /v1/billing/account", routes.middleware.wrap(
		mustPolicy("GET", "/v1/billing/account"),
		api.GetBillingAccountHandler(routes.store, now),
	))
	routes.mux.Handle("POST /v1/billing/reconciliation", routes.middleware.wrap(
		mustPolicy("POST", "/v1/billing/reconciliation"),
		api.RequestSubscriptionReconciliationHandler(routes.store, now),
	))
	routes.mux.Handle("PUT /v1/billing/licenses/{licenseId}/assignment", routes.middleware.wrap(
		mustPolicy("PUT", "/v1/billing/licenses/{licenseId}/assignment"),
		api.AssignSubscriptionLicenseHandler(routes.store, now),
	))
	routes.mux.Handle("DELETE /v1/billing/licenses/{licenseId}/assignment", routes.middleware.wrap(
		mustPolicy("DELETE", "/v1/billing/licenses/{licenseId}/assignment"),
		api.UnassignSubscriptionLicenseHandler(routes.store, now),
	))
}
