package routes

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
)

const defaultJSONBodyLimitBytes int64 = 1024 * 1024

func newScheduledImageValidator(
	limits api.ImageDecodeLimits,
	observer api.ImageValidationObserver,
) api.ImageValidator {
	limits = normalizedImageDecodeLimits(limits)
	validator, err := api.NewImageValidatorWithObserver(limits, observer)
	if err != nil {
		panic("invalid scheduled image decode limits")
	}
	return validator
}

func normalizedImageDecodeLimits(limits api.ImageDecodeLimits) api.ImageDecodeLimits {
	if limits.MaxWidth == 0 {
		// Route tests sometimes construct Config directly. Real startup always
		// receives the fully validated limits from LoadConfig.
		return api.DefaultImageDecodeLimits()
	}
	return limits
}

type v1Middleware struct {
	authCurrentMember func(http.Handler) http.Handler
	authRecovery      func(http.Handler) http.Handler
	deviceID          func(http.Handler) http.Handler
	member            func(http.Handler) http.Handler
	bodyLimit         middleware.BodyLimitConfig
	uploadAdmission   *middleware.UploadBodyAdmission
	rateLimit         map[RateClass]func(http.Handler) http.Handler
	observer          *observability.Observer
	hydrator          *api.IdentityCustomisationHydrator
}

func (m v1Middleware) wrap(policy RoutePolicy, handler http.Handler) http.Handler {
	accessClass := policy.AccessClass
	if !accessClass.Valid() {
		// Catalogue construction rejects invalid classes. Keep direct wrapper
		// use fail-closed as current-member authorization too.
		accessClass = AccessCurrentMember
	}
	wrapped := handler
	if m.hydrator != nil {
		wrapped = m.hydrator.Handler(wrapped)
	}
	// Keep BodyLimit outside response decorators so ResponseController reaches
	// net/http's writer and can install the route-specific read deadline.
	wrapped = middleware.BodyLimit(m.bodyLimit, middleware.BodyKind(policy.BodyKind), nil)(wrapped)
	if policy.BodyKind == BodyUpload {
		// Acquire the shared encoded-body permit before either upload handler can
		// read and retain its bounded body. Holding it until handler completion
		// also accounts for decode and remote-write work retaining those bytes.
		wrapped = m.uploadAdmission.Handler(wrapped)
	}
	if accessClass == AccessCurrentMember {
		wrapped = m.member(wrapped)
	}
	if rl := m.rateLimit[policy.RateClass]; rl != nil {
		wrapped = rl(wrapped)
	}
	switch accessClass {
	case AccessAuthenticatedRecovery:
		wrapped = m.deviceID(wrapped)
		wrapped = m.authRecovery(wrapped)
	case AccessCurrentMember:
		wrapped = m.deviceID(wrapped)
		wrapped = m.authCurrentMember(wrapped)
	case AccessAnonymous:
		if policy.RateClass == RateClassAuth {
			wrapped = m.deviceID(wrapped)
		}
	}
	wrapped = middleware.BodyPrecheck(m.bodyLimit, middleware.BodyKind(policy.BodyKind), nil)(wrapped)
	return middleware.HTTPInFlight(m.observer)(wrapped)
}

type Registrar interface {
	Handle(string, http.Handler)
}

type middlewareDependencies struct {
	Config                    Config
	Logger                    *slog.Logger
	DB                        *pgxpool.Pool
	AuthService               auth.AuthService
	CraftskySessionStore      *auth.CraftskySessionStore
	InstagramMembership       *instagram.MembershipStore
	OwnerLifecycles           *ownerlifecycle.Store
	RateLimiter               *middleware.LocalRateLimiter
	ProfileCustomisationStore *api.ProfileCustomisationStore
}

func buildV1Middleware(deps middlewareDependencies, observer *observability.Observer) v1Middleware {
	devAuthPolicy := middleware.DevAuthPolicy{Mode: middleware.DevAuthDisabled}
	if deps.Config.Env == EnvDev {
		devAuthPolicy.Mode = middleware.DevAuthLocal
		if deps.Config.DevRemoteAccess {
			devAuthPolicy.Mode = middleware.DevAuthRemote
			devAuthPolicy.Secret = deps.Config.DevAuthSecret.reveal()
		}
	}
	recoveryAuthService, ok := deps.AuthService.(auth.RecoveryAuthService)
	if !ok {
		panic("routes: auth service does not implement recovery authentication")
	}
	authCurrentMember := middleware.Authenticated(deps.AuthService, deps.Logger, devAuthPolicy)
	authRecovery := middleware.AuthenticatedRecovery(recoveryAuthService, deps.Logger, devAuthPolicy)
	deviceID := middleware.DeviceID(deps.CraftskySessionStore, deps.Logger)
	membership := deps.InstagramMembership
	if membership == nil {
		membership = instagram.NewMembershipStore(deps.DB)
	}
	currentMember := middleware.CurrentMember(membership, deps.Logger)
	if deps.OwnerLifecycles != nil {
		currentMember = middleware.CurrentMember(membership, deps.Logger, deps.OwnerLifecycles)
	}
	rateLimits := map[RateClass]func(http.Handler) http.Handler{}
	if deps.RateLimiter != nil {
		rateLimits[RateClassAuth] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassAuth, deps.Logger)
		rateLimits[RateClassRead] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassRead, deps.Logger)
		rateLimits[RateClassWrite] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassWrite, deps.Logger)
		rateLimits[RateClassSearch] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassSearch, deps.Logger)
		rateLimits[RateClassUpload] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassUpload, deps.Logger)
		rateLimits[RateClassLinkPreview] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassLinkPreview, deps.Logger)
	}
	bodyLimitCfg := middleware.BodyLimitConfig{
		DefaultJSONBytes:       deps.Config.JSONBodyLimitBytes,
		UploadBytes:            deps.Config.MaxImageUploadBytes,
		DefaultJSONReadTimeout: deps.Config.HTTPJSONBodyReadTimeout,
		UploadReadTimeout:      deps.Config.HTTPUploadBodyReadTimeout,
	}
	if bodyLimitCfg.DefaultJSONBytes == 0 {
		bodyLimitCfg.DefaultJSONBytes = defaultJSONBodyLimitBytes
	}
	imageLimits := normalizedImageDecodeLimits(deps.Config.ImageDecodeLimits)
	uploadAdmission, err := middleware.NewUploadBodyAdmission(
		imageLimits.MaxConcurrentDecodes,
		imageLimits.AdmissionWait,
	)
	if err != nil {
		panic("routes: invalid upload body admission")
	}
	profileCustomisationStore := deps.ProfileCustomisationStore
	if profileCustomisationStore == nil && deps.DB != nil {
		profileCustomisationStore = api.NewProfileCustomisationStore(deps.DB)
	}
	var hydrator *api.IdentityCustomisationHydrator
	if profileCustomisationStore != nil {
		hydrator = api.NewIdentityCustomisationHydrator(profileCustomisationStore)
	}
	return v1Middleware{
		authCurrentMember: authCurrentMember,
		authRecovery:      authRecovery,
		deviceID:          deviceID,
		member:            currentMember,
		bodyLimit:         bodyLimitCfg,
		uploadAdmission:   uploadAdmission,
		rateLimit:         rateLimits,
		observer:          observer,
		hydrator:          hydrator,
	}
}

// AddRoutes is the sole route composer. Registration stays in capability
// functions whose bundles expose only the dependencies that capability uses.
func AddRoutes(_ context.Context, mux Registrar, deps *Dependencies) {
	observer := deps.Observability
	if observer == nil {
		observer = observability.New(observability.Config{Env: string(deps.Config.Env)})
	}
	inFlight := middleware.HTTPInFlight(observer)

	registerPublicOperationsRoutes(publicOperationsRouteBundle{
		mux: mux, inFlight: inFlight, env: deps.Config.Env,
		db: deps.DB, consumer: deps.Consumer, logger: deps.Logger,
	})

	oauthHandlers := newOAuthHandlers(oauthRouteDependencies{
		app: deps.OAuthApp, artifacts: deps.OAuthArtifacts,
		sessionStore: deps.CraftskySessionStore, db: deps.DB, logger: deps.Logger,
		identityCacheUpdater: deps.IdentityCacheUpdater,
		repositoryTracker:    deps.RepositoryTracker,
		deletionOAuth:        deps.AccountDeletionOAuth,
		deletionPendingLogin: deps.AccountDeletionPendingLogin,
		oauthFlow:            deps.OAuthFlow, handoffs: deps.HandoffCoordinator,
		sessionLifecycle:    deps.SessionLifecycle,
		newPendingPDSClient: deps.NewPendingPDSClient,
		onboardingProfile:   deps.OnboardingProfile,
		loginCompleteURL:    deps.LoginCompleteURL,
		deletionCompleteURL: deps.DeletionCompleteURL,
		allowDevScheme:      deps.Config.EnableDevOAuthScheme,
	})
	registerPublicOAuthRoutes(publicOAuthRouteBundle{
		mux: mux, inFlight: inFlight, handlers: oauthHandlers,
		instagramWebhook: deps.InstagramWebhook,
	})

	profileCustomisationStore := deps.ProfileCustomisationStore
	if profileCustomisationStore == nil && deps.DB != nil {
		profileCustomisationStore = api.NewProfileCustomisationStore(deps.DB)
	}
	v1mw := buildV1Middleware(middlewareDependencies{
		Config: deps.Config, Logger: deps.Logger, DB: deps.DB,
		AuthService: deps.AuthService, CraftskySessionStore: deps.CraftskySessionStore,
		InstagramMembership: deps.InstagramMembership, OwnerLifecycles: deps.OwnerLifecycles,
		RateLimiter: deps.RateLimiter, ProfileCustomisationStore: profileCustomisationStore,
	}, observer)
	mediaLimits := api.MediaLimits{
		MaxPostImages:       deps.Config.MaxPostImages,
		MaxImageUploadBytes: deps.Config.MaxImageUploadBytes,
	}

	registerAuthRoutes(authRouteBundle{mux: mux, middleware: v1mw, handlers: oauthHandlers})
	scheduledImageValidator := newScheduledImageValidator(deps.Config.ImageDecodeLimits, observer)
	registerSearchRoutes(searchRouteBundle{
		mux: mux, middleware: v1mw,
		facetStore:     api.NewFacetStore(deps.DB, deps.AuthoritativeHandleResolver),
		searchStore:    api.NewSearchStore(deps.DB, observer),
		handleResolver: deps.HandleResolver, languages: deps.LanguagePreferences,
		logger: deps.Logger,
	})
	registerLogoutRoute(logoutRouteBundle{mux: mux, middleware: v1mw, handlers: oauthHandlers})
	registerAccountDeletionRoutes(accountDeletionRouteBundle{
		mux: mux, middleware: v1mw, service: deps.AccountDeletion,
	})
	registerMigrationRoutes(migrationRouteBundle{
		mux: mux, middleware: v1mw, limits: deps.Config.InstagramLimits,
		trustedProxyCIDRs:    deps.Config.InstagramTrustedProxyCIDRs,
		integrationAvailable: deps.Config.InstagramIntegrationAvailable,
		rateLimiter:          deps.InstagramRateLimiter, verification: deps.InstagramVerification,
		account: deps.InstagramAccount, imports: deps.InstagramImports,
		suggestions: deps.InstagramSuggestions, profileStore: deps.ProfileStore,
		handleResolver: deps.HandleResolver, logger: deps.Logger,
	})
	registerProfileRelationshipRoutes(profileRelationshipRouteBundle{
		mux: mux, middleware: v1mw, profileStore: deps.ProfileStore,
		profileCustomisationStore: profileCustomisationStore,
		followStore:               deps.FollowStore, relationshipStore: deps.RelationshipStore,
		relationshipMutations: deps.RelationshipMutations,
		handleResolver:        deps.HandleResolver,
		authoritativeResolver: deps.AuthoritativeHandleResolver,
		newPDSEffects:         deps.NewPDSEffects,
		reportStore:           deps.ReportStore, reportForwarder: deps.ReportForwarder,
		mediaLimits: mediaLimits, logger: deps.Logger,
	})

	postStore := api.NewPostStore(deps.DB, observer)
	savedPostStore := api.NewSavedPostStore(deps.DB)
	profilePinStore := api.NewProfilePinStore(deps.DB, api.ProfilePinStoreOptions{Observer: observer})
	savedPostService := api.NewSavedPostService(savedPostStore, postStore, deps.HandleResolver)
	oauthHandlers.NotificationSubscriptions = postStore
	registerNotificationRoutes(notificationRouteBundle{
		mux: mux, middleware: v1mw, postStore: postStore,
		handleResolver: deps.HandleResolver, languages: deps.LanguagePreferences,
		logger: deps.Logger,
	})
	registerScheduledPostRoutes(scheduledPostRouteBundle{
		mux: mux, middleware: v1mw, newPDSEffects: deps.NewPDSEffects,
		mediaLimits:    mediaLimits,
		imageValidator: scheduledImageValidator,
		posts:          deps.ScheduledPosts, media: deps.ScheduledMedia,
		manualPublisher: deps.ScheduledManualPublisher, logger: deps.Logger,
	})
	registerPostRoutes(postRouteBundle{
		mux: mux, middleware: v1mw,
		moderation: devModerationRouteConfig{
			env: deps.Config.Env, enabled: deps.Config.EnableDevModeration,
			token:             deps.Config.DevModerationToken,
			defaultSourceDID:  deps.Config.DevLabelerDID,
			trustedSourceDIDs: deps.Config.TrustedModerationSourceDIDs,
		},
		postStore: postStore, savedPostStore: savedPostStore,
		savedPostService: savedPostService, profilePinStore: profilePinStore,
		handleResolver: deps.HandleResolver, newPDSEffects: deps.NewPDSEffects,
		reportStore: deps.ReportStore, reportForwarder: deps.ReportForwarder,
		moderationStore: deps.ModerationStore, languages: deps.LanguagePreferences,
		mediaLimits: mediaLimits, logger: deps.Logger,
	})
	registerLinkPreviewRoute(linkPreviewRouteBundle{
		mux: mux, middleware: v1mw, service: deps.LinkPreviews,
		enabled: deps.Config.LinkPreviewsEnabled, observer: observer,
	})
	registerFallbackRoutes(fallbackRouteBundle{mux: mux, inFlight: inFlight})
}
