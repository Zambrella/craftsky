package routes

import (
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/tap"
)

// Environment is the deployment mode relevant to HTTP route composition.
// The app package adapts its validated process configuration into this narrow
// route-owned representation.
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvProd Environment = "prod"
)

// Secret keeps route configuration diagnostics from exposing credentials.
type Secret string

func (Secret) String() string   { return "[REDACTED]" }
func (Secret) GoString() string { return "routes.Secret([REDACTED])" }

func (secret Secret) reveal() string { return string(secret) }

// InstagramLimits contains only the request-plane limits consumed by routes.
type InstagramLimits struct {
	ChallengeDIDPer15Minutes    int
	ChallengeDevicePer15Minutes int
	ChallengeIPPer15Minutes     int
	ConfirmationDIDPerHour      int
	ConfirmationDevicePerHour   int
	ImportsDIDPerHour           int
	ImportsDevicePerHour        int
}

// Config is the route and server policy subset of the process configuration.
// It deliberately excludes storage, worker, and external-client settings.
type Config struct {
	Env                           Environment
	AllowedOrigins                []string
	ExpectedHosts                 []string
	ExpectedHostAllowAnyPort      bool
	DevRemoteAccess               bool
	DevAuthSecret                 Secret
	JSONBodyLimitBytes            int64
	MaxPostImages                 int
	MaxImageUploadBytes           int64
	ImageDecodeLimits             api.ImageDecodeLimits
	HTTPJSONBodyReadTimeout       time.Duration
	HTTPUploadBodyReadTimeout     time.Duration
	InstagramLimits               InstagramLimits
	InstagramTrustedProxyCIDRs    []netip.Prefix
	InstagramIntegrationAvailable bool
	EnableDevOAuthScheme          bool
	EnableDevModeration           bool
	DevModerationToken            string
	DevLabelerDID                 string
	TrustedModerationSourceDIDs   []string
}

// Dependencies is the route-composition boundary. AddRoutes is the only
// consumer of the complete set; capability registrars receive smaller bundles.
type Dependencies struct {
	Config        Config
	Logger        *slog.Logger
	DB            *pgxpool.Pool
	AuthService   auth.AuthService
	RateLimiter   *middleware.LocalRateLimiter
	Observability *observability.Observer

	AccountDeletion             accountdeletion.Service
	AccountDeletionOAuth        auth.AccountDeletionOAuthCallbacks
	AccountDeletionPendingLogin auth.AccountDeletionPendingLoginPolicy

	OAuthApp             *oauth.ClientApp
	OAuthArtifacts       auth.ClientArtifacts
	OAuthFlow            auth.OAuthFlowCoordinator
	HandoffCoordinator   auth.HandoffCoordinator
	SessionLifecycle     *auth.SessionLifecycleService
	CraftskySessionStore *auth.CraftskySessionStore
	OwnerLifecycles      *ownerlifecycle.Store
	NewPendingPDSClient  auth.PendingOnboardingPDSClientFactory
	OnboardingProfile    auth.OnboardingProfileWriter
	LoginCompleteURL     string
	DeletionCompleteURL  string
	IdentityCacheUpdater auth.IdentityCacheUpdater
	RepositoryTracker    auth.RepositoryTracker

	HandleResolver              api.HandleResolver
	AuthoritativeHandleResolver api.HandleResolver
	Consumer                    tap.Consumer

	InstagramMembership   *instagram.MembershipStore
	InstagramRateLimiter  *instagram.PostgresRateLimiter
	InstagramVerification *instagram.VerificationService
	InstagramWebhook      http.Handler
	InstagramSuggestions  *instagram.SuggestionService
	InstagramAccount      *instagram.AccountStore
	InstagramImports      *instagram.ImportService

	ProfileStore              *api.ProfileStore
	ProfileCustomisationStore *api.ProfileCustomisationStore
	FollowStore               *api.FollowStore
	RelationshipStore         *relationships.Store
	RelationshipMutations     api.RelationshipMutationService
	ReportStore               *api.ReportStore
	ReportForwarder           api.ReportForwarder
	ModerationStore           *api.ModerationStore
	LanguagePreferences       *languages.Store
	NewPDSEffects             pdseffects.ExecutorFactory

	ScheduledPosts           *scheduledposts.Store
	ScheduledMedia           *scheduledposts.PrivateMediaService
	ScheduledManualPublisher scheduledposts.ManualPublisher
}
