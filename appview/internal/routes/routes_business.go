package routes

import (
	"log/slog"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
)

type businessRouteBundle struct {
	mux               Registrar
	middleware        v1Middleware
	store             *business.Store
	handleResolver    api.HandleResolver
	appendCommands    api.AppendCommandExecutor
	addressedCommands api.AddressedCommandExecutor
	reportStore       *api.ReportStore
	reportForwarder   api.ReportForwarder
	cursors           *api.EventCursorCodec
	now               func() time.Time
	logger            *slog.Logger
}

func registerBusinessRoutes(routes businessRouteBundle) {
	routes.mux.Handle("PUT /v1/profiles/me/account-type", routes.middleware.wrap(mustPolicy("PUT", "/v1/profiles/me/account-type"), api.PutBusinessAccountTypeHandler(routes.store)))
	routes.mux.Handle("PUT /v1/profiles/me/business", routes.middleware.wrap(mustPolicy("PUT", "/v1/profiles/me/business"), api.PutBusinessProfileHandler(nil, api.BusinessProfileHandlerOptions{Commands: routes.addressedCommands})))
	routes.mux.Handle("DELETE /v1/profiles/me/business", routes.middleware.wrap(mustPolicy("DELETE", "/v1/profiles/me/business"), api.DeleteBusinessProfileHandler(nil, api.BusinessProfileHandlerOptions{DeleteCommands: routes.addressedCommands})))
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}/events", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/events"), api.GetProfileBusinessEventsHandler(routes.store, routes.handleResolver, routes.cursors, routes.now)))
	routes.mux.Handle("POST /v1/events", routes.middleware.wrap(mustPolicy("POST", "/v1/events"), api.PostBusinessEventHandler(nil, routes.now, api.BusinessEventHandlerOptions{Commands: routes.appendCommands})))
	routes.mux.Handle("GET /v1/events", routes.middleware.wrap(mustPolicy("GET", "/v1/events"), api.GetOwnerBusinessEventsHandler(routes.store, routes.cursors, routes.now)))
	routes.mux.Handle("GET /v1/events/{did}/{rkey}", routes.middleware.wrap(mustPolicy("GET", "/v1/events/{did}/{rkey}"), api.GetBusinessEventHandler(routes.store, routes.now)))
	routes.mux.Handle("PUT /v1/events/{did}/{rkey}", routes.middleware.wrap(mustPolicy("PUT", "/v1/events/{did}/{rkey}"), api.PutBusinessEventHandler(nil, routes.now, api.BusinessEventHandlerOptions{AddressedCommands: routes.addressedCommands})))
	routes.mux.Handle("DELETE /v1/events/{did}/{rkey}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/events/{did}/{rkey}"), api.DeleteBusinessEventHandler(nil, api.BusinessEventHandlerOptions{DeleteCommands: routes.addressedCommands})))
	routes.mux.Handle("POST /v1/events/{did}/{rkey}/reports", routes.middleware.wrap(mustPolicy("POST", "/v1/events/{did}/{rkey}/reports"), api.ReportBusinessEventHandler(routes.store, routes.reportStore, routes.reportForwarder, routes.logger, routes.now)))
}
