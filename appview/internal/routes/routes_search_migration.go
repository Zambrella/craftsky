package routes

import (
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
)

type searchRouteBundle struct {
	mux            Registrar
	middleware     v1Middleware
	facetStore     *api.FacetStore
	searchStore    *api.SearchStore
	handleResolver api.HandleResolver
	languages      *languages.Store
	logger         *slog.Logger
}

func registerSearchRoutes(routes searchRouteBundle) {
	routes.mux.Handle("GET /v1/whoami", routes.middleware.wrap(mustPolicy("GET", "/v1/whoami"), api.WhoAmIHandler(routes.handleResolver, routes.logger)))
	routes.mux.Handle("GET /v1/facets/mentions", routes.middleware.wrap(mustPolicy("GET", "/v1/facets/mentions"), api.ListFacetMentionSuggestionsHandler(routes.facetStore, routes.logger)))
	routes.mux.Handle("GET /v1/facets/mentions/resolve", routes.middleware.wrap(mustPolicy("GET", "/v1/facets/mentions/resolve"), api.ResolveFacetMentionHandler(routes.facetStore, routes.logger)))
	routes.mux.Handle("GET /v1/facets/hashtags", routes.middleware.wrap(mustPolicy("GET", "/v1/facets/hashtags"), api.ListFacetHashtagSuggestionsHandler(routes.facetStore, routes.logger)))
	routes.mux.Handle("GET /v1/projects", routes.middleware.wrap(mustPolicy("GET", "/v1/projects"), api.ListProjectsHandler(routes.searchStore, routes.handleResolver, routes.logger, routes.languages)))
	routes.mux.Handle("GET /v1/search/hashtags/{tag}/posts", routes.middleware.wrap(mustPolicy("GET", "/v1/search/hashtags/{tag}/posts"), api.SearchHashtagPostsHandler(routes.searchStore, routes.handleResolver, routes.logger, routes.languages)))
	routes.mux.Handle("GET /v1/search/suggestions", routes.middleware.wrap(mustPolicy("GET", "/v1/search/suggestions"), api.SearchSuggestionsHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("GET /v1/search/hashtags", routes.middleware.wrap(mustPolicy("GET", "/v1/search/hashtags"), api.SearchHashtagsHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("GET /v1/search/profiles", routes.middleware.wrap(mustPolicy("GET", "/v1/search/profiles"), api.SearchProfilesHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("GET /v1/search/posts", routes.middleware.wrap(mustPolicy("GET", "/v1/search/posts"), api.SearchPostsHandler(routes.searchStore, routes.handleResolver, routes.logger, routes.languages)))
	routes.mux.Handle("GET /v1/search/projects", routes.middleware.wrap(mustPolicy("GET", "/v1/search/projects"), api.SearchProjectsHandler(routes.searchStore, routes.handleResolver, routes.logger, routes.languages)))
	routes.mux.Handle("GET /v1/search/hashtags/top", routes.middleware.wrap(mustPolicy("GET", "/v1/search/hashtags/top"), api.TopHashtagsHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("GET /v1/search/recent", routes.middleware.wrap(mustPolicy("GET", "/v1/search/recent"), api.ListRecentSearchesHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("POST /v1/search/recent", routes.middleware.wrap(mustPolicy("POST", "/v1/search/recent"), api.SaveRecentSearchHandler(routes.searchStore, routes.logger)))
	routes.mux.Handle("DELETE /v1/search/recent/{id}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/search/recent/{id}"), api.DeleteRecentSearchHandler(routes.searchStore, routes.logger)))
}

type migrationRouteBundle struct {
	mux                  Registrar
	middleware           v1Middleware
	limits               InstagramLimits
	trustedProxyCIDRs    []netip.Prefix
	integrationAvailable bool
	rateLimiter          *instagram.PostgresRateLimiter
	verification         *instagram.VerificationService
	account              *instagram.AccountStore
	imports              *instagram.ImportService
	suggestions          *instagram.SuggestionService
	profileStore         *api.ProfileStore
	handleResolver       api.HandleResolver
	logger               *slog.Logger
}

func registerMigrationRoutes(routes migrationRouteBundle) {
	instagramLimit := func(handler http.Handler, rules ...middleware.InstagramRateLimitRule) http.Handler {
		// A missing data-plane key means Instagram's rate-limited private write
		// plane is unavailable. Nil deliberately fails these writes closed while
		// unwrapped reads and privacy deletions remain available.
		var limiter middleware.InstagramPersistentLimiter
		if routes.rateLimiter != nil {
			limiter = routes.rateLimiter
		}
		return middleware.InstagramPersistentRateLimit(
			limiter,
			rules,
			routes.trustedProxyCIDRs,
			routes.logger,
		)(handler)
	}
	routes.mux.Handle("POST /v1/migrations/instagram/verifications", routes.middleware.wrap(mustPolicy("POST", "/v1/migrations/instagram/verifications"), instagramLimit(
		api.CreateInstagramVerificationHandler(routes.verification, routes.logger),
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitChallengeDID, Identity: middleware.InstagramRateIdentityDID, Window: 15 * time.Minute, Limit: routes.limits.ChallengeDIDPer15Minutes},
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitChallengeDevice, Identity: middleware.InstagramRateIdentityDevice, Window: 15 * time.Minute, Limit: routes.limits.ChallengeDevicePer15Minutes},
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitChallengeIP, Identity: middleware.InstagramRateIdentityClientIP, Window: 15 * time.Minute, Limit: routes.limits.ChallengeIPPer15Minutes},
	)))
	routes.mux.Handle("GET /v1/migrations/instagram/verifications/current", routes.middleware.wrap(mustPolicy("GET", "/v1/migrations/instagram/verifications/current"), api.GetCurrentInstagramVerificationHandler(routes.verification, routes.logger)))
	routes.mux.Handle("GET /v1/migrations/instagram/verifications/{verificationId}", routes.middleware.wrap(mustPolicy("GET", "/v1/migrations/instagram/verifications/{verificationId}"), api.GetInstagramVerificationHandler(routes.verification, routes.logger)))
	routes.mux.Handle("DELETE /v1/migrations/instagram/verifications/{verificationId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/migrations/instagram/verifications/{verificationId}"), api.DeleteInstagramVerificationHandler(routes.verification, routes.logger)))
	routes.mux.Handle("POST /v1/migrations/instagram/verifications/{verificationId}/confirm", routes.middleware.wrap(mustPolicy("POST", "/v1/migrations/instagram/verifications/{verificationId}/confirm"), instagramLimit(
		api.ConfirmInstagramVerificationHandler(routes.verification, routes.logger),
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitConfirmationDID, Identity: middleware.InstagramRateIdentityDID, Window: time.Hour, Limit: routes.limits.ConfirmationDIDPerHour},
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitConfirmationDevice, Identity: middleware.InstagramRateIdentityDevice, Window: time.Hour, Limit: routes.limits.ConfirmationDevicePerHour},
	)))
	integrationAvailable := func() bool { return routes.integrationAvailable }
	routes.mux.Handle("GET /v1/migrations/instagram/account", routes.middleware.wrap(mustPolicy("GET", "/v1/migrations/instagram/account"), api.GetInstagramAccountHandler(routes.account, integrationAvailable, routes.logger)))
	routes.mux.Handle("DELETE /v1/migrations/instagram/account", routes.middleware.wrap(mustPolicy("DELETE", "/v1/migrations/instagram/account"), api.DeleteInstagramAccountHandler(routes.account, routes.logger)))
	routes.mux.Handle("PATCH /v1/migrations/instagram/settings", routes.middleware.wrap(mustPolicy("PATCH", "/v1/migrations/instagram/settings"), api.PatchInstagramSettingsHandler(routes.account, integrationAvailable, routes.logger)))
	routes.mux.Handle("POST /v1/migrations/instagram/imports", routes.middleware.wrap(mustPolicy("POST", "/v1/migrations/instagram/imports"), instagramLimit(
		api.CreateInstagramImportHandler(routes.imports, routes.logger),
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitImportDID, Identity: middleware.InstagramRateIdentityDID, Window: time.Hour, Limit: routes.limits.ImportsDIDPerHour},
		middleware.InstagramRateLimitRule{Scope: instagram.RateLimitImportDevice, Identity: middleware.InstagramRateIdentityDevice, Window: time.Hour, Limit: routes.limits.ImportsDevicePerHour},
	)))
	routes.mux.Handle("GET /v1/migrations/instagram/imports", routes.middleware.wrap(mustPolicy("GET", "/v1/migrations/instagram/imports"), api.ListInstagramImportsHandler(routes.imports, routes.logger)))
	routes.mux.Handle("GET /v1/migrations/instagram/imports/{importId}", routes.middleware.wrap(mustPolicy("GET", "/v1/migrations/instagram/imports/{importId}"), api.GetInstagramImportHandler(routes.imports, routes.logger)))
	routes.mux.Handle("PATCH /v1/migrations/instagram/imports/{importId}", routes.middleware.wrap(mustPolicy("PATCH", "/v1/migrations/instagram/imports/{importId}"), api.PatchInstagramImportHandler(routes.imports, routes.logger)))
	routes.mux.Handle("DELETE /v1/migrations/instagram/imports/{importId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/migrations/instagram/imports/{importId}"), api.DeleteInstagramImportHandler(routes.imports, routes.logger)))
	routes.mux.Handle("GET /v1/migrations/instagram/suggestions", routes.middleware.wrap(
		mustPolicy("GET", "/v1/migrations/instagram/suggestions"),
		api.ListInstagramSuggestionsHandler(routes.suggestions, routes.profileStore, routes.handleResolver, routes.logger),
	))
	routes.mux.Handle("POST /v1/migrations/instagram/suggestions/{suggestionId}/accept", routes.middleware.wrap(
		mustPolicy("POST", "/v1/migrations/instagram/suggestions/{suggestionId}/accept"),
		api.AcceptInstagramSuggestionHandler(routes.suggestions, routes.logger),
	))
	routes.mux.Handle("DELETE /v1/migrations/instagram/suggestions/{suggestionId}", routes.middleware.wrap(
		mustPolicy("DELETE", "/v1/migrations/instagram/suggestions/{suggestionId}"),
		api.DismissInstagramSuggestionHandler(routes.suggestions, routes.logger),
	))
}
