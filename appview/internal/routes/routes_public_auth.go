package routes

import (
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/tap"
)

type publicOperationsRouteBundle struct {
	mux      Registrar
	inFlight func(http.Handler) http.Handler
	env      Environment
	db       *pgxpool.Pool
	consumer tap.Consumer
	logger   *slog.Logger
}

func registerPublicOperationsRoutes(routes publicOperationsRouteBundle) {
	routes.mux.Handle("GET /health", routes.inFlight(api.HealthHandler(routes.db, routes.logger)))
	routes.mux.Handle("GET /healthz", routes.inFlight(api.NewHealthHandler(routes.db, routes.consumer)))
	if routes.env == EnvDev {
		routes.mux.Handle("GET /v1/dev/media/{name}", routes.inFlight(api.DevMediaHandler()))
		routes.mux.Handle("GET /v1/dev/panic", routes.inFlight(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("synthetic appview dev panic")
		})))
	}
}

type oauthRouteDependencies struct {
	app                  *oauth.ClientApp
	artifacts            auth.ClientArtifacts
	sessionStore         *auth.CraftskySessionStore
	db                   *pgxpool.Pool
	logger               *slog.Logger
	identityCacheUpdater auth.IdentityCacheUpdater
	repositoryTracker    auth.RepositoryTracker
	deletionOAuth        auth.AccountDeletionOAuthCallbacks
	deletionPendingLogin auth.AccountDeletionPendingLoginPolicy
	oauthFlow            auth.OAuthFlowCoordinator
	handoffs             auth.HandoffCoordinator
	sessionLifecycle     *auth.SessionLifecycleService
	newPendingPDSClient  auth.PendingOnboardingPDSClientFactory
	onboardingProfile    auth.OnboardingProfileWriter
	loginCompleteURL     string
	deletionCompleteURL  string
	allowDevScheme       bool
}

func newOAuthHandlers(deps oauthRouteDependencies) *auth.HTTPHandlers {
	handlers := auth.NewHTTPHandlers(
		deps.app,
		deps.artifacts,
		deps.sessionStore,
		deps.db,
		deps.logger,
		deps.identityCacheUpdater,
	)
	handlers.RepositoryTracker = deps.repositoryTracker
	handlers.DeletionOAuthCallbacks = deps.deletionOAuth
	handlers.DeletionPendingLogin = deps.deletionPendingLogin
	handlers.OAuthFlow = deps.oauthFlow
	handlers.Handoffs = deps.handoffs
	handlers.SessionLifecycle = deps.sessionLifecycle
	handlers.NewPendingPDSClient = deps.newPendingPDSClient
	handlers.OnboardingProfile = deps.onboardingProfile
	handlers.LoginCompleteURL = deps.loginCompleteURL
	handlers.DeletionCompleteURL = deps.deletionCompleteURL
	handlers.AllowDevScheme = deps.allowDevScheme
	return handlers
}

type publicOAuthRouteBundle struct {
	mux              Registrar
	inFlight         func(http.Handler) http.Handler
	handlers         *auth.HTTPHandlers
	instagramWebhook http.Handler
}

func registerPublicOAuthRoutes(routes publicOAuthRouteBundle) {
	routes.mux.Handle("GET /oauth/client-metadata.json", routes.inFlight(routes.handlers.ClientMetadataHandler()))
	routes.mux.Handle("GET /oauth/jwks.json", routes.inFlight(routes.handlers.JWKSHandler()))
	routes.mux.Handle("GET /oauth/callback", routes.inFlight(routes.handlers.CallbackHandler()))
	// Meta callbacks are deliberately absent unless the complete integration
	// was validated and wired at startup. They never share Craftsky auth/body
	// middleware because signature verification covers the exact raw bytes.
	if routes.instagramWebhook != nil {
		routes.mux.Handle("GET /integrations/instagram/webhook", routes.inFlight(routes.instagramWebhook))
		routes.mux.Handle("POST /integrations/instagram/webhook", routes.inFlight(routes.instagramWebhook))
	}
}

type authRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	handlers   *auth.HTTPHandlers
}

func registerAuthRoutes(routes authRouteBundle) {
	routes.mux.Handle("POST /v1/auth/login", routes.middleware.wrap(mustPolicy("POST", "/v1/auth/login"), routes.handlers.LoginHandler()))
	routes.mux.Handle("POST /v1/auth/registrations", routes.middleware.wrap(mustPolicy("POST", "/v1/auth/registrations"), routes.handlers.RegistrationHandler()))
	routes.mux.Handle("POST /v1/auth/handoffs/exchange", routes.middleware.wrap(mustPolicy("POST", "/v1/auth/handoffs/exchange"), routes.handlers.HandoffExchangeHandler()))
	routes.mux.Handle("POST /v1/auth/handoffs/confirm", routes.middleware.wrap(mustPolicy("POST", "/v1/auth/handoffs/confirm"), routes.handlers.HandoffConfirmHandler()))
}

type logoutRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	handlers   *auth.HTTPHandlers
}

func registerLogoutRoute(routes logoutRouteBundle) {
	routes.mux.Handle("POST /v1/auth/logout", routes.middleware.wrap(mustPolicy("POST", "/v1/auth/logout"), routes.handlers.LogoutHandler()))
}

type accountDeletionRouteBundle struct {
	mux        Registrar
	middleware v1Middleware
	service    accountdeletion.Service
}

func registerAccountDeletionRoutes(routes accountDeletionRouteBundle) {
	routes.mux.Handle("POST /v1/account-deletion/intents", routes.middleware.wrap(mustPolicy("POST", "/v1/account-deletion/intents"), api.CreateAccountDeletionIntentHandler(routes.service)))
	routes.mux.Handle("DELETE /v1/account-deletion/intents/{jobId}", routes.middleware.wrap(mustPolicy("DELETE", "/v1/account-deletion/intents/{jobId}"), api.CancelAccountDeletionIntentHandler(routes.service)))
	routes.mux.Handle("POST /v1/account-deletions/{jobId}", routes.middleware.wrap(mustPolicy("POST", "/v1/account-deletions/{jobId}"), api.AcceptAccountDeletionHandler(routes.service)))
}

type fallbackRouteBundle struct {
	mux      Registrar
	inFlight func(http.Handler) http.Handler
}

func registerFallbackRoutes(routes fallbackRouteBundle) {
	routes.mux.Handle("/", routes.inFlight(http.NotFoundHandler()))
}
