package routes

import (
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/moderation"
)

type moderationRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	store      *moderation.Store
	commands   api.ModerationCommander
	sourceDID  syntax.DID
	config     Config
}

func registerModerationRoutes(bundle moderationRouteBundle) {
	bundle.mux.Handle("GET /v1/moderation/standing", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/standing"), api.ModerationStandingHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/moderation/history", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/history"), api.ModerationHistoryHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/moderation/history/{caseReference}", bundle.middleware.wrap(mustPolicy("GET", "/v1/moderation/history/{caseReference}"), api.ModerationHistoryEntryHandler(bundle.store)))
	if !bundle.config.ModerationAdminEnabled {
		return
	}
	bundle.mux.Handle("GET /v1/admin/moderation/cases", bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/moderation/cases"), api.ModerationAdminQueueHandler(bundle.store)))
	bundle.mux.Handle("GET /v1/admin/moderation/cases/{caseReference}", bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "GET", "/v1/admin/moderation/cases/{caseReference}"), api.ModerationAdminDetailHandler(bundle.store)))
	commands := []struct{ path, kind string }{{"decisions", "decision"}, {"appeal-confirmations", "appealConfirmation"}, {"appeal-resolutions", "appealResolution"}, {"effect-changes", "effectChange"}, {"restorations", "restoration"}}
	for _, command := range commands {
		pattern := "/v1/admin/moderation/cases/{caseReference}/" + command.path
		bundle.mux.Handle("POST "+pattern, bundle.middleware.wrap(mustConfiguredPolicy(bundle.config.Env, bundle.config, "POST", pattern), api.ModerationAdminCommandHandler(bundle.commands, bundle.sourceDID, command.kind)))
	}
}
