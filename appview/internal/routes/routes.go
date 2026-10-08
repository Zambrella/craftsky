package routes

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/eligibility"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/subscriptions"
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
	subscriptionAccess subscriptionAccessReader
	authCurrentMember  func(http.Handler) http.Handler
	authRecovery       func(http.Handler) http.Handler
	deviceID           func(http.Handler) http.Handler
	member             func(http.Handler) http.Handler
	bodyLimit          middleware.BodyLimitConfig
	uploadAdmission    *middleware.UploadBodyAdmission
	rateLimit          map[RateClass]func(http.Handler) http.Handler
	observer           *observability.Observer
	hydrator           *api.IdentityCustomisationHydrator
	folderRedactor     interface {
		Handler(http.Handler) http.Handler
	}
	accountTypeHydrator *api.IdentityAccountTypeHydrator
	moderator           func(http.Handler) http.Handler
	suspension          middleware.SuspensionReader
	eligibility         middleware.AgeEligibilityReader
	handlerDecorator    func(RoutePolicy, http.Handler) http.Handler
}

type subscriptionAccessReader interface {
	SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error)
}

func (m v1Middleware) requirePlus(next http.Handler) http.Handler {
	return m.requireTier(next, subscriptions.SelfAccess.AllowsPlus, "Plus")
}

func (m v1Middleware) requireBusiness(next http.Handler) http.Handler {
	return m.requireTier(next, subscriptions.SelfAccess.AllowsBusiness, "Business")
}

func (m v1Middleware) requireTier(next http.Handler, allowed func(subscriptions.SelfAccess) bool, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.subscriptionAccess == nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		access, err := m.subscriptionAccess.SelfAccess(r.Context(), did, time.Now())
		if err != nil {
			envelope.WriteError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		if !allowed(access) {
			envelope.WriteError(w, http.StatusForbidden, "subscription_required", name+" subscription required", middleware.GetRunID(r.Context()), nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m v1Middleware) wrap(policy RoutePolicy, handler http.Handler) http.Handler {
	accessClass := policy.AccessClass
	if !accessClass.Valid() {
		// Catalogue construction rejects invalid classes. Keep direct wrapper
		// use fail-closed as current-member authorization too.
		accessClass = AccessCurrentMember
	}
	wrapped := handler
	if m.handlerDecorator != nil {
		wrapped = m.handlerDecorator(policy, wrapped)
	}
	if plusRoutes[policyKey(policy.Method, policy.PathPattern)] {
		wrapped = m.requirePlus(wrapped)
	}
	if businessOwnerRoutes[policyKey(policy.Method, policy.PathPattern)] {
		wrapped = m.requireBusiness(wrapped)
	}
	if policy.Method == http.MethodGet && policy.PathPattern == "/v1/events/{did}/{rkey}" {
		ownerHandler := m.requireBusiness(wrapped)
		visitorHandler := wrapped
		wrapped = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, ok := middleware.GetDID(r.Context())
			if ok && caller.String() == r.PathValue("did") {
				ownerHandler.ServeHTTP(w, r)
				return
			}
			visitorHandler.ServeHTTP(w, r)
		})
	}
	if m.hydrator != nil {
		wrapped = m.hydrator.Handler(wrapped)
	}
	if m.accountTypeHydrator != nil {
		wrapped = m.accountTypeHydrator.Handler(wrapped)
	}
	if m.folderRedactor != nil {
		wrapped = m.folderRedactor.Handler(wrapped)
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
		wrapped = middleware.AgeEligibilityEnforcement(m.eligibility, policy.EligibilityClass.AllowedWhenRestricted(), nil)(wrapped)
	}
	if accessClass == AccessCurrentMember {
		wrapped = middleware.ModerationEnforcement(m.suspension, policy.SuspensionClass.AllowedWhenSuspended(), nil)(wrapped)
	}
	if accessClass == AccessCurrentMember {
		wrapped = m.member(wrapped)
	}
	if rl := m.rateLimit[policy.RateClass]; rl != nil {
		wrapped = rl(wrapped)
	}
	switch accessClass {
	case AccessModerator:
		wrapped = m.moderator(wrapped)
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

var plusRoutes = map[string]bool{
	"GET /v1/saved-post-folders":                true,
	"POST /v1/saved-post-folders":               true,
	"PATCH /v1/saved-post-folders/{folderId}":   true,
	"DELETE /v1/saved-post-folders/{folderId}":  true,
	"POST /v1/scheduled-posts":                  true,
	"PUT /v1/scheduled-posts/{id}":              true,
	"POST /v1/scheduled-posts/{id}/publication": true,
	"PUT /v1/scheduled-post-media/{mediaId}":    true,
	"PUT /v1/posts/{did}/{rkey}/pin":            true,
	"DELETE /v1/posts/{did}/{rkey}/pin":         true,
	"GET /v1/profiles/me/follower-growth":       true,
	"PUT /v1/profiles/me/customisation":         true,
}

var businessOwnerRoutes = map[string]bool{
	"PUT /v1/profiles/me/business":    true,
	"DELETE /v1/profiles/me/business": true,
	"POST /v1/events":                 true,
	"GET /v1/events":                  true,
	"PUT /v1/events/{did}/{rkey}":     true,
	"DELETE /v1/events/{did}/{rkey}":  true,
}

type Registrar interface {
	Handle(string, http.Handler)
}

type middlewareDependencies struct {
	Subscriptions             *subscriptions.Store
	Config                    Config
	Logger                    *slog.Logger
	DB                        *pgxpool.Pool
	AuthService               auth.AuthService
	CraftskySessionStore      *auth.CraftskySessionStore
	InstagramMembership       *instagram.MembershipStore
	OwnerLifecycles           *ownerlifecycle.Store
	RateLimiter               *middleware.LocalRateLimiter
	ProfileCustomisationStore *api.ProfileCustomisationStore
	BusinessStore             *business.Store
	Suspension                middleware.SuspensionReader
	Eligibility               middleware.AgeEligibilityReader
	ModeratorAuthenticator    middleware.ModeratorAuthenticator
	HandlerDecorator          func(RoutePolicy, http.Handler) http.Handler
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
		if deps.Subscriptions != nil {
			hydrator = api.NewIdentityCustomisationHydrator(profileCustomisationStore, deps.Subscriptions)
		}
	}
	var accountTypeHydrator *api.IdentityAccountTypeHydrator
	if deps.Subscriptions != nil {
		accountTypeHydrator = api.NewIdentityAccountTypeHydrator(nil, deps.Subscriptions)
	} else if deps.BusinessStore != nil {
		accountTypeHydrator = api.NewIdentityAccountTypeHydrator(deps.BusinessStore)
	}
	moderatorAuthentication := middleware.ModeratorAuthentication(deps.Config.ModerationAdminToken.reveal(), deps.Config.ModerationAdminActorID, deps.Config.ModerationAdminSourceSystem, deps.Logger, observer)
	if deps.ModeratorAuthenticator != nil {
		moderatorAuthentication = middleware.ModeratorDatabaseAuthentication(deps.ModeratorAuthenticator, deps.Logger, observer)
	}
	var subscriptionAccess subscriptionAccessReader
	if deps.Subscriptions != nil {
		subscriptionAccess = deps.Subscriptions
	}
	var folderRedactor interface {
		Handler(http.Handler) http.Handler
	}
	if deps.Subscriptions != nil {
		folderRedactor = api.SavedFolderRedactor(deps.Subscriptions)
	}
	return v1Middleware{
		subscriptionAccess:  subscriptionAccess,
		authCurrentMember:   authCurrentMember,
		authRecovery:        authRecovery,
		deviceID:            deviceID,
		member:              currentMember,
		bodyLimit:           bodyLimitCfg,
		uploadAdmission:     uploadAdmission,
		rateLimit:           rateLimits,
		observer:            observer,
		hydrator:            hydrator,
		folderRedactor:      folderRedactor,
		accountTypeHydrator: accountTypeHydrator,
		moderator:           moderatorAuthentication,
		suspension:          deps.Suspension,
		eligibility:         deps.Eligibility,
		handlerDecorator:    deps.HandlerDecorator,
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
		imageSafety: deps.ImageSafetyReadiness,
	})

	oauthHandlers := newOAuthHandlers(oauthRouteDependencies{
		app: deps.OAuthApp, artifacts: deps.OAuthArtifacts,
		sessionStore: deps.CraftskySessionStore, db: deps.DB, logger: deps.Logger,
		identityCacheUpdater: deps.IdentityCacheUpdater,
		deletionOAuth:        deps.AccountDeletionOAuth,
		deletionPendingLogin: deps.AccountDeletionPendingLogin,
		oauthFlow:            deps.OAuthFlow, handoffs: deps.HandoffCoordinator,
		sessionLifecycle:         deps.SessionLifecycle,
		newPendingPDSClient:      deps.NewPendingPDSClient,
		onboardingProfile:        deps.OnboardingProfile,
		blueskyProfileProjector:  deps.BlueskyProfileProjector,
		craftskyProfileProjector: deps.CraftskyProfileProjector,
		loginCompleteURL:         deps.LoginCompleteURL,
		deletionCompleteURL:      deps.DeletionCompleteURL,
		allowDevScheme:           deps.Config.EnableDevOAuthScheme,
	})
	registerPublicOAuthRoutes(publicOAuthRouteBundle{
		mux: mux, inFlight: inFlight, handlers: oauthHandlers,
		instagramWebhook: deps.InstagramWebhook,
	})
	registerRevenueCatWebhookRoute(mux, inFlight, deps.RevenueCatWebhook)

	profileCustomisationStore := deps.ProfileCustomisationStore
	if profileCustomisationStore == nil && deps.DB != nil {
		profileCustomisationStore = api.NewProfileCustomisationStore(deps.DB)
	}
	businessStore := deps.BusinessStore
	if businessStore == nil && deps.DB != nil {
		if deps.Subscriptions != nil {
			businessStore = business.NewStoreForEnvironment(deps.DB, deps.Subscriptions.AccessEnvironment())
		} else {
			businessStore = business.NewStore(deps.DB)
		}
	}
	moderationCases := deps.ModerationCases
	if moderationCases == nil && deps.DB != nil {
		moderationCases = moderation.NewStore(deps.DB)
	}
	var suspension middleware.SuspensionReader
	if deps.SuspensionReader != nil {
		suspension = deps.SuspensionReader
	} else if moderationCases != nil {
		suspension = moderationCases
	}
	ageEligibility := eligibility.NewStore(deps.DB)
	eligibilityReader := deps.EligibilityReader
	if eligibilityReader == nil {
		eligibilityReader = ageEligibility
	}
	v1mw := buildV1Middleware(middlewareDependencies{
		Config: deps.Config, Logger: deps.Logger, DB: deps.DB,
		AuthService: deps.AuthService, CraftskySessionStore: deps.CraftskySessionStore,
		InstagramMembership: deps.InstagramMembership, OwnerLifecycles: deps.OwnerLifecycles,
		RateLimiter: deps.RateLimiter, ProfileCustomisationStore: profileCustomisationStore,
		BusinessStore: businessStore, Suspension: suspension,
		Eligibility:            eligibilityReader,
		ModeratorAuthenticator: deps.ModeratorAuthenticator,
		Subscriptions:          deps.Subscriptions,
		HandlerDecorator:       deps.routeHandlerDecorator,
	}, observer)
	mediaLimits := api.MediaLimits{
		MaxPostImages:       deps.Config.MaxPostImages,
		MaxImageUploadBytes: deps.Config.MaxImageUploadBytes,
	}

	registerAuthRoutes(authRouteBundle{mux: mux, middleware: v1mw, handlers: oauthHandlers})
	registerModerationRoutes(moderationRouteBundle{
		mux: mux, middleware: v1mw, store: moderationCases,
		commands: deps.ModerationCommands, sourceDID: syntax.DID(deps.Config.ModerationSourceDID),
		config: deps.Config, imageHealth: deps.ImageSafetyHealth, safetyWork: deps.SafetyWork,
		incidents: deps.SafetyIncidents, intake: deps.SafetyIntake,
		evidence: deps.SafetyEvidence, holds: deps.SafetyHolds, workflows: deps.SafetyWorkflows, csea: deps.SafetyCSEA,
		eligibility: ageEligibility, now: deps.Now,
	})
	registerVideoRoutes(videoRouteBundle{
		mux: mux, middleware: v1mw, authorization: deps.VideoUploadAuthorization,
		limits: deps.VideoUploadLimits, logger: deps.Logger, observer: observer,
		enabled: deps.Config.VideoEnabled,
	})
	scheduledImageValidator := newScheduledImageValidator(deps.Config.ImageDecodeLimits, observer)
	registerSearchRoutes(searchRouteBundle{
		mux: mux, middleware: v1mw,
		facetStore:     api.NewFacetStoreWithInvalidator(deps.DB, deps.AuthoritativeHandleResolver, deps.IdentityInvalidator),
		searchStore:    api.NewSearchStoreWithPlayback(deps.DB, observer, deps.VideoPlayback),
		handleResolver: deps.HandleResolver, languages: deps.LanguagePreferences,
		logger: deps.Logger,
	})
	registerLogoutRoute(logoutRouteBundle{mux: mux, middleware: v1mw, handlers: oauthHandlers})
	registerAccountDeletionRoutes(accountDeletionRouteBundle{
		mux: mux, middleware: v1mw, service: deps.AccountDeletion,
	})
	registerSubscriptionRoutes(subscriptionRouteBundle{
		mux: mux, middleware: v1mw, store: deps.Subscriptions, now: deps.Now,
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
	registerOnboardingRoutes(onboardingRouteBundle{
		mux: mux, middleware: v1mw, store: api.NewOnboardingStatusStore(deps.DB, deps.Config.RequiredPolicyVersion), logger: deps.Logger,
	})
	registerAgeEligibilityRoutes(ageEligibilityRouteBundle{
		mux: mux, middleware: v1mw, store: ageEligibility, logger: deps.Logger,
	})
	registerProfileRelationshipRoutes(profileRelationshipRouteBundle{
		mux: mux, middleware: v1mw, profileStore: deps.ProfileStore,
		businessProfiles:          businessStore,
		followerGrowth:            deps.FollowerGrowth,
		profileCustomisationStore: profileCustomisationStore,
		relationshipStore:         deps.RelationshipStore,
		relationshipMutations:     deps.RelationshipMutations,
		handleResolver:            deps.HandleResolver,
		authoritativeResolver:     deps.AuthoritativeHandleResolver,
		pdsCommands:               deps.PDSCommands,
		compoundCommands:          deps.PDSCompoundCommands,
		reportStore:               deps.ReportStore, reportForwarder: deps.ReportForwarder,
		mediaLimits: mediaLimits, logger: deps.Logger,
	})
	registerBusinessRoutes(businessRouteBundle{
		mux: mux, middleware: v1mw, store: businessStore,
		handleResolver: deps.HandleResolver,
		appendCommands: deps.PDSAppendCommands, addressedCommands: deps.PDSAddressedCommands,
		reportStore: deps.ReportStore, reportForwarder: deps.ReportForwarder,
		cursors: deps.EventCursorCodec, now: deps.Now, logger: deps.Logger,
	})

	postStore := api.NewPostStoreWithPlayback(deps.DB, observer, deps.VideoPlayback)
	savedPostStore := api.NewSavedPostStore(deps.DB)
	profilePinOptions := api.ProfilePinStoreOptions{Observer: observer, RequirePlus: deps.Subscriptions != nil}
	if deps.Subscriptions != nil {
		profilePinOptions.AccessEnvironment = deps.Subscriptions.AccessEnvironment()
	}
	profilePinStore := api.NewProfilePinStore(deps.DB, profilePinOptions)
	savedPostService := api.NewSavedPostService(savedPostStore, postStore, deps.HandleResolver)
	oauthHandlers.NotificationSubscriptions = postStore
	registerNotificationRoutes(notificationRouteBundle{
		mux: mux, middleware: v1mw, postStore: postStore,
		handleResolver: deps.HandleResolver, languages: deps.LanguagePreferences,
		logger: deps.Logger,
	})
	registerScheduledPostRoutes(scheduledPostRouteBundle{
		mux: mux, middleware: v1mw, newBlobEffects: deps.NewBlobEffects,
		mediaLimits:    mediaLimits,
		imageValidator: scheduledImageValidator,
		posts:          deps.ScheduledPosts, media: deps.ScheduledMedia,
		manualPublisher: deps.ScheduledManualPublisher, logger: deps.Logger,
	})
	registerPostRoutes(postRouteBundle{
		mux: mux, middleware: v1mw,
		subscriptionAccess: deps.Subscriptions,
		moderation: devModerationRouteConfig{
			env: deps.Config.Env, enabled: deps.Config.EnableDevModeration,
			token:             deps.Config.DevModerationToken,
			defaultSourceDID:  deps.Config.DevLabelerDID,
			trustedSourceDIDs: deps.Config.TrustedModerationSourceDIDs,
		},
		postStore: postStore, savedPostStore: savedPostStore,
		savedPostService: savedPostService, profilePinStore: profilePinStore,
		handleResolver: deps.HandleResolver,
		pdsCommands:    deps.PDSCommands, appendCommands: deps.PDSAppendCommands, addressedCommands: deps.PDSAddressedCommands,
		reportStore: deps.ReportStore, reportForwarder: deps.ReportForwarder,
		moderationStore: deps.ModerationStore, languages: deps.LanguagePreferences,
		mediaLimits: mediaLimits, videoVerifier: deps.VideoCompletionVerifier, videoEnabled: deps.Config.VideoEnabled, videoCaptions: deps.VideoCaptionFetcher, videoObserver: observer, logger: deps.Logger,
	})
	registerLinkPreviewRoute(linkPreviewRouteBundle{
		mux: mux, middleware: v1mw, service: deps.LinkPreviews,
		enabled: deps.Config.LinkPreviewsEnabled, observer: observer,
	})
	registerFallbackRoutes(fallbackRouteBundle{mux: mux, inFlight: inFlight})
}
