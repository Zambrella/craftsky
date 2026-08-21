package app

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

// authDependencies owns persisted OAuth parents, CraftSky child sessions, and
// the lifecycle coordinators that must share one owner authority boundary.
// Remote-purpose clients remain supplied by the foundation bundle.
type authDependencies struct {
	store              *auth.PostgresAuthStore
	app                *oauth.ClientApp
	craftskySessions   *auth.CraftskySessionStore
	flow               *auth.OAuthFlowService
	handoffs           *auth.HandoffService
	sessionLifecycle   *auth.SessionLifecycleService
	sessionCoordinator *auth.OAuthSessionCoordinator
}

func newAuthDependencies(
	pool *pgxpool.Pool,
	federated *federatedClients,
	owners *ownerDependencies,
	artifacts auth.ClientArtifacts,
	handoffReceiptKey []byte,
	cfg Config,
	logger *slog.Logger,
) (*authDependencies, error) {
	oauthStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		SessionExpiry:                cfg.OAuthSessionAbsoluteLifetime,
		SessionInactivity:            cfg.CraftskySessionInactivity,
		SessionAbsoluteLifetime:      cfg.OAuthSessionAbsoluteLifetime,
		AuthRequestExpiry:            cfg.OAuthAuthRequestExpiry,
		PendingAuthRequestCapacity:   cfg.OAuthPendingAuthRequestCapacity,
		AuthRequestTerminalRetention: cfg.OAuthAuthRequestTerminalRetention,
		OwnerLifecycles:              owners.lifecycles,
		EndpointValidator:            federated.boundary,
		Logger:                       logger,
	})
	oauthApp := oauth.NewClientApp(&artifacts.Config, oauthStore)
	oauthApp.Client = federated.oauth
	oauthApp.Resolver.Client = federated.metadata
	// OAuth identity routing is authoritative: a cached handle mapping must not
	// choose the DID/PDS that receives credentials. Ordinary display reads keep
	// using federated.directory.
	oauthApp.Dir = federated.authoritativeDirectory
	craftskyStore, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            cfg.CraftskySessionInactivity,
		ActivityWriteInterval: cfg.CraftskySessionActivityWriteInterval,
		RecoveryAuthorization: owners.deletionStore,
	})
	if err != nil {
		return nil, fmt.Errorf("CraftSky session store: %w", err)
	}
	oauthFlow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: oauthStore, Owners: owners.lifecycles,
		StartOperationTimeout:    cfg.OAuthLoginStartTimeout,
		CallbackOperationTimeout: cfg.OAuthCallbackOperationTimeout,
		DeletionRequests:         owners.deletionStore,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuth flow service: %w", err)
	}
	handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
		Pool: pool, Owners: owners.lifecycles, Sessions: craftskyStore,
		ExchangeTTL: cfg.OAuthHandoffExchangeTTL, ConfirmationTTL: cfg.OAuthHandoffConfirmationTTL,
		ReceiptKey: handoffReceiptKey, ReceiptKeyVersion: cfg.OAuthHandoffReceiptKeyVersion,
		Random: rand.Reader, Now: time.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuth handoff service: %w", err)
	}
	sessionLifecycle, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners.lifecycles, Sessions: craftskyStore,
		DeletionExemption: owners.deletionStore,
		Now:               time.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("session lifecycle service: %w", err)
	}
	oauthSessionCoordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: oauthApp, Store: oauthStore, Owners: owners.lifecycles,
		OperationTimeout: cfg.OAuthSessionOperationTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuth session coordinator: %w", err)
	}
	return &authDependencies{
		store: oauthStore, app: oauthApp, craftskySessions: craftskyStore,
		flow: oauthFlow, handoffs: handoffs, sessionLifecycle: sessionLifecycle,
		sessionCoordinator: oauthSessionCoordinator,
	}, nil
}
