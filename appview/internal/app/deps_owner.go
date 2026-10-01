package app

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// ownerDependencies is the shared authority boundary consumed by auth,
// ingestion, scheduled work, and account deletion. It is constructed before
// feature stores so those stores cannot invent a parallel lifecycle fence.
type ownerDependencies struct {
	fence         *ownerlifecycle.Fencer
	lifecycles    *ownerlifecycle.Store
	deletionStore *accountdeletion.Store
}

func newOwnerDependencies(pool *pgxpool.Pool, cfg Config) (*ownerDependencies, error) {
	ownerFence, err := ownerlifecycle.NewFencer(pool, cfg.OwnerFenceAcquireTimeout)
	if err != nil {
		return nil, fmt.Errorf("owner lifecycle fence: %w", err)
	}
	ownerLifecycles, err := ownerlifecycle.NewStore(pool, ownerFence, time.Now)
	if err != nil {
		return nil, fmt.Errorf("owner lifecycle store: %w", err)
	}
	return &ownerDependencies{
		fence: ownerFence, lifecycles: ownerLifecycles,
		deletionStore: accountdeletion.NewStore(pool, time.Now),
	}, nil
}
