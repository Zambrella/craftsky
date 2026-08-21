package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/push"
)

// newPushDependencies builds the optional provider client and the bounded
// delivery dispatcher as one capability. A disabled deployment has no Firebase
// dependency and returns a nil dispatcher.
func newPushDependencies(
	ctx context.Context,
	pool *pgxpool.Pool,
	owners *ownerDependencies,
	observer *observability.Observer,
	cfg Config,
) (*push.Dispatcher, error) {
	if !cfg.PushEnabled {
		return nil, nil
	}
	sender, err := push.NewFirebaseSender(ctx, cfg.FirebaseProjectID)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging init: %w", err)
	}
	dispatcher, err := push.NewDispatcherValidated(pool, sender, push.DispatcherOptions{
		BatchSize:          cfg.PushBatchSize,
		Concurrency:        cfg.PushConcurrency,
		LeaseDuration:      cfg.PushLeaseDuration,
		SendTimeout:        cfg.PushSendTimeout,
		FinalizationMargin: cfg.PushFinalizationMargin,
		Observer:           observer,
		LifecycleFence:     pushOwnerLifecycleFence{store: owners.lifecycles},
	})
	if err != nil {
		return nil, fmt.Errorf("push dispatcher init: %w", err)
	}
	return dispatcher, nil
}
