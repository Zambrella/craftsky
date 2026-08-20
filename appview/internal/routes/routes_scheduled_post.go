package routes

import (
	"log/slog"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledPostRouteBundle struct {
	mux             Registrar
	middleware      v1Middleware
	newPDSEffects   pdseffects.ExecutorFactory
	mediaLimits     api.MediaLimits
	imageValidator  api.ImageValidator
	posts           *scheduledposts.Store
	media           *scheduledposts.PrivateMediaService
	manualPublisher scheduledposts.ManualPublisher
	logger          *slog.Logger
}

func registerScheduledPostRoutes(routes scheduledPostRouteBundle) {
	routes.mux.Handle("POST /v1/blobs/images", routes.middleware.wrap(mustPolicy("POST", "/v1/blobs/images"), api.ImageBlobUploadHandler(routes.newPDSEffects, routes.mediaLimits, routes.logger)))
	routes.mux.Handle("PUT /v1/scheduled-post-media/{mediaId}", routes.middleware.wrap(mustPolicy("PUT", "/v1/scheduled-post-media/{mediaId}"), api.PutScheduledMediaHandler(routes.media, routes.mediaLimits, routes.imageValidator, routes.logger)))
	routes.mux.Handle("GET /v1/scheduled-post-media/{mediaId}", routes.middleware.wrap(mustPolicy("GET", "/v1/scheduled-post-media/{mediaId}"), api.GetScheduledMediaHandler(routes.media, routes.logger)))
	routes.mux.Handle("DELETE /v1/scheduled-post-media/{mediaId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/scheduled-post-media/{mediaId}"), api.DeleteScheduledMediaHandler(routes.media, time.Now, routes.logger)))
	routes.mux.Handle("POST /v1/scheduled-posts", routes.middleware.wrap(mustPolicy("POST", "/v1/scheduled-posts"), api.CreateScheduledPostHandler(routes.posts, routes.mediaLimits, time.Now, routes.logger)))
	routes.mux.Handle("GET /v1/scheduled-posts", routes.middleware.wrap(mustPolicy("GET", "/v1/scheduled-posts"), api.ListScheduledPostsHandler(routes.posts, routes.logger)))
	routes.mux.Handle("GET /v1/scheduled-posts/{id}", routes.middleware.wrap(mustPolicy("GET", "/v1/scheduled-posts/{id}"), api.GetScheduledPostHandler(routes.posts, routes.logger)))
	routes.mux.Handle("PUT /v1/scheduled-posts/{id}", routes.middleware.wrap(mustPolicy("PUT", "/v1/scheduled-posts/{id}"), api.UpdateScheduledPostHandler(routes.posts, routes.mediaLimits, time.Now, routes.logger)))
	routes.mux.Handle("DELETE /v1/scheduled-posts/{id}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/scheduled-posts/{id}"), api.DeleteScheduledPostHandler(routes.posts, time.Now, routes.logger)))
	routes.mux.Handle("POST /v1/scheduled-posts/{id}/publication", routes.middleware.wrap(mustPolicy("POST", "/v1/scheduled-posts/{id}/publication"), api.PublishScheduledPostHandler(routes.manualPublisher, routes.mediaLimits, time.Now, routes.logger)))
}

type devModerationRouteConfig struct {
	env               Environment
	enabled           bool
	token             string
	defaultSourceDID  string
	trustedSourceDIDs []string
}

type postRouteBundle struct {
	mux              Registrar
	middleware       v1Middleware
	moderation       devModerationRouteConfig
	postStore        *api.PostStore
	savedPostStore   *api.SavedPostStore
	savedPostService *api.SavedPostService
	profilePinStore  *api.ProfilePinStore
	handleResolver   api.HandleResolver
	newPDSEffects    pdseffects.ExecutorFactory
	reportStore      *api.ReportStore
	reportForwarder  api.ReportForwarder
	moderationStore  *api.ModerationStore
	languages        *languages.Store
	mediaLimits      api.MediaLimits
	logger           *slog.Logger
}

func registerPostRoutes(routes postRouteBundle) {
	routes.mux.Handle("POST /v1/posts", routes.middleware.wrap(mustPolicy("POST", "/v1/posts"), api.CreatePostHandler(routes.postStore, routes.newPDSEffects, routes.handleResolver, routes.mediaLimits, routes.logger)))
	routes.mux.Handle("GET /v1/posts/{did}/{rkey}", routes.middleware.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}"), api.GetPostHandler(routes.postStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("POST /v1/posts/{did}/{rkey}/saves", routes.middleware.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/saves"), api.SavePostHandler(routes.postStore, routes.savedPostStore)))
	routes.mux.Handle("DELETE /v1/posts/{did}/{rkey}/saves", routes.middleware.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/saves"), api.UnsavePostHandler(routes.savedPostStore)))
	routes.mux.Handle("GET /v1/profiles/me/pins", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/me/pins"), api.GetProfilePinsHandler(routes.profilePinStore)))
	routes.mux.Handle("PUT /v1/posts/{did}/{rkey}/pin", routes.middleware.wrap(mustPolicy("PUT", "/v1/posts/{did}/{rkey}/pin"), api.PinProfilePostHandler(routes.profilePinStore)))
	routes.mux.Handle("DELETE /v1/posts/{did}/{rkey}/pin", routes.middleware.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/pin"), api.UnpinProfilePostHandler(routes.profilePinStore)))
	routes.mux.Handle("GET /v1/saved-posts", routes.middleware.wrap(mustPolicy("GET", "/v1/saved-posts"), api.ListSavedPostsHandler(routes.savedPostService)))
	routes.mux.Handle("GET /v1/saved-post-folders", routes.middleware.wrap(mustPolicy("GET", "/v1/saved-post-folders"), api.ListSavedPostFoldersHandler(routes.savedPostStore)))
	routes.mux.Handle("POST /v1/saved-post-folders", routes.middleware.wrap(mustPolicy("POST", "/v1/saved-post-folders"), api.CreateSavedPostFolderHandler(routes.savedPostStore)))
	routes.mux.Handle("PATCH /v1/saved-post-folders/{folderId}", routes.middleware.wrap(mustPolicy("PATCH", "/v1/saved-post-folders/{folderId}"), api.RenameSavedPostFolderHandler(routes.savedPostStore)))
	routes.mux.Handle("DELETE /v1/saved-post-folders/{folderId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/saved-post-folders/{folderId}"), api.DeleteSavedPostFolderHandler(routes.savedPostStore)))
	routes.mux.Handle("GET /v1/posts/{did}/{rkey}/replies", routes.middleware.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}/replies"), api.ListCommentRepliesHandler(routes.postStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/posts/{did}/{rkey}/comments", routes.middleware.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}/comments"), api.GetPostCommentsHandler(routes.postStore, routes.handleResolver, routes.logger)))
	routes.mux.Handle("POST /v1/posts/{did}/{rkey}/likes", routes.middleware.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/likes"), api.LikePostHandler(routes.postStore, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("DELETE /v1/posts/{did}/{rkey}/likes", routes.middleware.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/likes"), api.UnlikePostHandler(routes.postStore, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("POST /v1/posts/{did}/{rkey}/reposts", routes.middleware.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/reposts"), api.RepostPostHandler(routes.postStore, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("DELETE /v1/posts/{did}/{rkey}/reposts", routes.middleware.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/reposts"), api.UnrepostPostHandler(routes.postStore, routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("DELETE /v1/posts/{did}/{rkey}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}"), api.DeletePostHandler(routes.newPDSEffects, routes.logger)))
	routes.mux.Handle("POST /v1/posts/{did}/{rkey}/reports", routes.middleware.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/reports"), api.ReportPostHandler(routes.postStore, routes.reportStore, routes.reportForwarder, routes.logger)))
	if routes.moderation.env == EnvDev && routes.moderation.enabled && routes.moderation.token != "" {
		config := Config{
			Env:                         routes.moderation.env,
			EnableDevModeration:         routes.moderation.enabled,
			DevModerationToken:          routes.moderation.token,
			DevLabelerDID:               routes.moderation.defaultSourceDID,
			TrustedModerationSourceDIDs: routes.moderation.trustedSourceDIDs,
		}
		routes.mux.Handle("POST /v1/dev/moderation/ozone-events",
			routes.middleware.wrap(mustConfiguredPolicy(config.Env, config, "POST", "/v1/dev/moderation/ozone-events"), api.DevModerationOzoneEventsHandler(
				routes.moderation.token,
				api.ModerationRequestConfig{
					DefaultSourceDID:  routes.moderation.defaultSourceDID,
					TrustedSourceDIDs: routes.moderation.trustedSourceDIDs,
				},
				routes.moderationStore,
				routes.logger,
			)))
	}
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}/posts", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/posts"), api.ListPostsByAuthorHandler(routes.postStore, routes.handleResolver, routes.logger, routes.profilePinStore, routes.languages)))
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}/projects", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/projects"), api.ListProjectsByAuthorHandler(routes.postStore, routes.handleResolver, routes.logger, routes.profilePinStore, routes.languages)))
	routes.mux.Handle("GET /v1/profiles/{handleOrDid}/comments", routes.middleware.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/comments"), api.ListCommentsByAuthorHandler(routes.postStore, routes.handleResolver, routes.logger, routes.languages)))
}
