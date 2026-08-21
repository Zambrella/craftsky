package app

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/instagram"
)

// instagramStorageDependencies is the private, AppView-owned Instagram
// foundation shared by ingestion, account deletion, and the optional Meta
// runtime. It grants no remote Meta or PDS-write capability.
type instagramStorageDependencies struct {
	suggestionPolicy      *instagram.PostgresInstagramSuggestionEligibilityPolicy
	membership            *instagram.MembershipStore
	rateLimiter           *instagram.PostgresRateLimiter
	privateData           *instagram.PrivateDataService
	restoration           *instagram.ReconciliationTrigger
	moderationRestoration *instagram.ModerationRestorationRelay
	moderationStore       *api.ModerationStore
	privateSuggestions    *instagram.PrivateSuggestionStore
}

func newInstagramStorageDependencies(
	pool *pgxpool.Pool,
	owners *ownerDependencies,
	content *contentDependencies,
	cfg Config,
) (*instagramStorageDependencies, error) {
	suggestionPolicy := instagram.NewPostgresInstagramSuggestionEligibilityPolicy(
		pool,
		instagramRelationshipSafetyProvider{store: content.relationships},
		time.Now,
	)
	membership := instagram.NewMembershipStore(pool)
	var rateLimiter *instagram.PostgresRateLimiter
	var err error
	if cfg.InstagramData.Available() {
		rateLimiter, err = instagram.NewPostgresRateLimiter(
			pool,
			cfg.InstagramData.HMACKey(),
			time.Now,
		)
		if err != nil {
			return nil, fmt.Errorf("instagram persistent rate limiter: %w", err)
		}
	}
	privateData := instagram.NewPrivateDataService(pool, rateLimiter, time.Now)
	restoration := instagram.NewReconciliationTrigger(pool, time.Now)
	moderationRestoration, err := instagram.NewModerationRestorationRelay(
		pool,
		owners.lifecycles,
		time.Now,
	)
	if err != nil {
		return nil, fmt.Errorf("moderation restoration relay: %w", err)
	}
	moderationStore, err := api.NewModerationStore(pool, owners.lifecycles)
	if err != nil {
		return nil, fmt.Errorf("moderation store: %w", err)
	}
	privateSuggestions, err := instagram.NewPrivateSuggestionStore(
		pool,
		owners.lifecycles,
		content.notificationLifecycle,
		time.Now,
	)
	if err != nil {
		return nil, fmt.Errorf("instagram private suggestion store: %w", err)
	}
	return &instagramStorageDependencies{
		suggestionPolicy: suggestionPolicy, membership: membership,
		rateLimiter: rateLimiter, privateData: privateData,
		restoration: restoration, moderationRestoration: moderationRestoration,
		moderationStore: moderationStore, privateSuggestions: privateSuggestions,
	}, nil
}
