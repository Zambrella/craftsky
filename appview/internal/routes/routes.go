package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// AddRoutes registers all App View routes on mux.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public ops.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))
	mux.Handle("GET /healthz", api.NewHealthHandler(deps.DB, deps.Consumer))

	// OAuth discovery endpoints (contracts with the AS; not versioned).
	oauthHandlers := auth.NewHTTPHandlers(
		deps.OAuthApp,
		deps.CraftskySessionStore,
		deps.DB,
		deps.Logger,
		deps.Config.Env == app.EnvDev,
		deps.NewPDSClient,
	)
	mux.Handle("GET /oauth/client-metadata.json", oauthHandlers.ClientMetadataHandler())
	mux.Handle("GET /oauth/jwks.json", oauthHandlers.JWKSHandler())
	mux.Handle("GET /oauth/callback", oauthHandlers.CallbackHandler())

	// Middleware stacks.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	deviceID := middleware.DeviceID(deps.CraftskySessionStore, deps.Logger)

	// v1 — unauthenticated but device-id required.
	mux.Handle("POST /v1/auth/login", deviceID(oauthHandlers.LoginHandler()))

	// v1 — authenticated + device-id required.
	mediaLimits := api.MediaLimits{
		MaxPostImages:       deps.Config.MaxPostImages,
		MaxImageUploadBytes: deps.Config.MaxImageUploadBytes,
	}
	mux.Handle("GET /v1/whoami", authN(deviceID(api.WhoAmIHandler(deps.HandleResolver, deps.Logger))))
	mux.Handle("POST /v1/auth/logout", authN(deviceID(oauthHandlers.LogoutHandler())))
	mux.Handle("GET /v1/profiles/{handleOrDid}",
		authN(deviceID(api.GetProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/profiles/me",
		authN(deviceID(api.GetMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/profiles/me/followers",
		authN(deviceID(api.GetMeFollowersHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/profiles/me/following",
		authN(deviceID(api.GetMeFollowingHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("PUT /v1/profiles/me",
		authN(deviceID(api.PutMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, mediaLimits, deps.Logger))))
	mux.Handle("GET /v1/profiles/{handleOrDid}/mutual-followers",
		authN(deviceID(api.GetMutualFollowersHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("POST /v1/profiles/{handleOrDid}/follows",
		authN(deviceID(api.FollowProfileHandler(deps.FollowStore, deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, deps.Logger))))
	mux.Handle("DELETE /v1/profiles/{handleOrDid}/follows",
		authN(deviceID(api.UnfollowProfileHandler(deps.FollowStore, deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, deps.Logger))))
	mux.Handle("POST /v1/profiles/{handleOrDid}/reports",
		authN(deviceID(api.ReportProfileHandler(api.NewProfileReportTargetResolver(deps.ProfileStore, deps.HandleResolver), deps.ReportStore, deps.ReportForwarder, deps.Logger))))

	// v1 — post handlers (authenticated + device-id required).
	postStore := api.NewPostStore(deps.DB)
	mux.Handle("GET /v1/feed/timeline",
		authN(deviceID(api.ListTimelineHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/notifications",
		authN(deviceID(api.ListNotificationsHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("POST /v1/blobs/images",
		authN(deviceID(api.ImageBlobUploadHandler(deps.NewPDSClient, mediaLimits, deps.Logger))))
	mux.Handle("POST /v1/posts",
		authN(deviceID(api.CreatePostHandler(postStore, deps.NewPDSClient, deps.HandleResolver, mediaLimits, deps.Logger))))
	mux.Handle("GET /v1/posts/{did}/{rkey}",
		authN(deviceID(api.GetPostHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/posts/{did}/{rkey}/replies",
		authN(deviceID(api.ListCommentRepliesHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/posts/{did}/{rkey}/comments",
		authN(deviceID(api.GetPostCommentsHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("POST /v1/posts/{did}/{rkey}/likes",
		authN(deviceID(api.LikePostHandler(postStore, deps.NewPDSClient, deps.Logger))))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}/likes",
		authN(deviceID(api.UnlikePostHandler(postStore, deps.NewPDSClient, deps.Logger))))
	mux.Handle("POST /v1/posts/{did}/{rkey}/reposts",
		authN(deviceID(api.RepostPostHandler(postStore, deps.NewPDSClient, deps.Logger))))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}/reposts",
		authN(deviceID(api.UnrepostPostHandler(postStore, deps.NewPDSClient, deps.Logger))))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}",
		authN(deviceID(api.DeletePostHandler(deps.NewPDSClient, deps.Logger))))
	mux.Handle("POST /v1/posts/{did}/{rkey}/reports",
		authN(deviceID(api.ReportPostHandler(postStore, deps.ReportStore, deps.ReportForwarder, deps.Logger))))
	if deps.Config.Env == app.EnvDev && deps.Config.EnableDevModeration && deps.Config.DevModerationToken != "" {
		mux.Handle("POST /v1/dev/moderation/ozone-events",
			authN(deviceID(api.DevModerationOzoneEventsHandler(
				deps.Config.DevModerationToken,
				api.ModerationRequestConfig{
					DefaultSourceDID:  deps.Config.DevLabelerDID,
					TrustedSourceDIDs: deps.Config.TrustedModerationSourceDIDs,
				},
				deps.ModerationStore,
				deps.Logger,
			))))
	}
	mux.Handle("GET /v1/profiles/{handleOrDid}/posts",
		authN(deviceID(api.ListPostsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger))))
	mux.Handle("GET /v1/profiles/{handleOrDid}/comments",
		authN(deviceID(api.ListCommentsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger))))

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
