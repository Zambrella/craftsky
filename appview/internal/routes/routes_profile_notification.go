package routes

import (
	"log/slog"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
)

type profileRelationshipRouteBundle struct {
	mux                       Registrar
	middleware                v1Middleware
	profileStore              *api.ProfileStore
	profileCustomisationStore *api.ProfileCustomisationStore
	followStore               *api.FollowStore
	relationshipStore         *relationships.Store
	relationshipMutations     api.RelationshipMutationService
	handleResolver            api.HandleResolver
	authoritativeResolver     api.HandleResolver
	newPDSEffects             pdseffects.ExecutorFactory
	reportStore               *api.ReportStore
	reportForwarder           api.ReportForwarder
	mediaLimits               api.MediaLimits
	logger                    *slog.Logger
}

func registerProfileRelationshipRoutes(routes profileRelationshipRouteBundle) {
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}"), api.GetProfileHandler(routes.profileStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/profiles/me", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me"), api.GetMeProfileHandler(routes.profileStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/profiles/me/followers", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me/followers"), api.GetMeFollowersHandler(routes.profileStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/profiles/me/following", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me/following"), api.GetMeFollowingHandler(routes.profileStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("PUT /v1/profiles/me", routes.middleware.wrap(mustPolicy("PUT", "/v1/profiles/me"), api.PutMeProfileHandler(routes.profileStore, routes.handleResolver, routes.newPDSEffects, routes.mediaLimits, routes.logger)))
	routes.mux.Handle("PUT /v1/profiles/me/customisation", routes.middleware.wrap(
		mustPolicy("PUT", "/v1/profiles/me/customisation"),
		api.PutProfileCustomisationHandler(routes.profileCustomisationStore),
	))
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}/mutual-followers", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/mutual-followers"), api.GetMutualFollowersHandler(routes.profileStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("POST /v1/profiles/{handleOrDid}/follows", routes.middleware.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/follows"), api.FollowProfileHandler(routes.followStore, routes.profileStore, routes.authoritativeResolver, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("DELETE /v1/profiles/{handleOrDid}/follows", routes.middleware.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/follows"), api.UnfollowProfileHandler(routes.followStore, routes.profileStore, routes.authoritativeResolver, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("POST /v1/profiles/{handleOrDid}/mutes", routes.middleware.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/mutes"), api.MuteProfileHandler(routes.relationshipMutations, routes.relationshipStore, routes.authoritativeResolver, routes.logger)))
	routes.mux.Handle("DELETE /v1/profiles/{handleOrDid}/mutes", routes.middleware.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/mutes"), api.UnmuteProfileHandler(routes.relationshipMutations, routes.relationshipStore, routes.authoritativeResolver, routes.logger)))
	routes.mux.Handle("POST /v1/profiles/{handleOrDid}/blocks", routes.middleware.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/blocks"), api.BlockProfileHandler(routes.relationshipMutations, routes.relationshipStore, routes.authoritativeResolver, routes.logger)))
	routes.mux.Handle("DELETE /v1/profiles/{handleOrDid}/blocks", routes.middleware.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/blocks"), api.UnblockProfileHandler(routes.relationshipMutations, routes.relationshipStore, routes.authoritativeResolver, routes.logger)))
	routes.mux.Handle("GET /v1/profiles/me/mutes", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me/mutes"), api.ListMutedProfilesHandler(routes.relationshipStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/profiles/me/blocks", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me/blocks"), api.ListBlockedProfilesHandler(routes.relationshipStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("POST /v1/profiles/{handleOrDid}/reports", routes.middleware.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/reports"), api.ReportProfileHandler(api.NewProfileReportTargetResolver(routes.profileStore, routes.authoritativeResolver), routes.reportStore, routes.reportForwarder, routes.logger)))
}

type notificationRouteBundle struct {
	mux            Registrar
	middleware     v1Middleware
	postStore      *api.PostStore
	handleResolver api.HandleResolver
	languages      *languages.Store
	logger         *slog.Logger
}

func registerNotificationRoutes(routes notificationRouteBundle) {
	routes.mux.Handle("GET /v1/feed/timeline", routes.middleware.wrap(mustPolicy("GET", "/v1/feed/timeline"), api.ListTimelineHandler(routes.postStore, routes.handleResolver, routes.logger, routes.languages)))
	routes.mux.Handle("GET /v1/notifications", routes.middleware.wrap(mustPolicy("GET", "/v1/notifications"), api.ListNotificationsHandler(routes.postStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/notifications/new-count", routes.middleware.wrap(mustPolicy("GET", "/v1/notifications/new-count"), api.NotificationNewCountHandler(routes.postStore, routes.logger)))
	routes.mux.Handle("POST /v1/notifications/seen", routes.middleware.wrap(mustPolicy("POST", "/v1/notifications/seen"), api.MarkNotificationsSeenHandler(routes.postStore, routes.logger)))
	routes.mux.Handle("GET /v1/notifications/preferences", routes.middleware.wrap(mustPolicy("GET", "/v1/notifications/preferences"), api.GetNotificationPreferencesHandler(routes.postStore, routes.logger)))
	routes.mux.Handle("PATCH /v1/notifications/preferences", routes.middleware.wrap(mustPolicy("PATCH", "/v1/notifications/preferences"), api.PatchNotificationPreferencesHandler(routes.postStore, routes.logger)))
	routes.mux.Handle("GET /v1/languages/preferences", routes.middleware.wrap(mustPolicy("GET", "/v1/languages/preferences"), api.GetLanguagePreferencesHandler(routes.languages)))
	routes.mux.Handle("PUT /v1/languages/preferences", routes.middleware.wrap(mustPolicy("PUT", "/v1/languages/preferences"), api.PutLanguagePreferencesHandler(routes.languages)))
	routes.mux.Handle("POST /v1/languages/preferences/initialize", routes.middleware.wrap(mustPolicy("POST", "/v1/languages/preferences/initialize"), api.InitializeLanguagePreferencesHandler(routes.languages)))
	routes.mux.Handle("POST /v1/notifications/devices", routes.middleware.wrap(mustPolicy("POST", "/v1/notifications/devices"), api.RegisterNotificationDeviceHandler(routes.postStore, routes.logger)))
	routes.mux.Handle("DELETE /v1/notifications/devices/{accountSubscriptionId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/notifications/devices/{accountSubscriptionId}"), api.RemoveNotificationDeviceHandler(routes.postStore, routes.logger)))
}
