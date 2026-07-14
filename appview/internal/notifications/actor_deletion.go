package notifications

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActorDeletionService struct{ pool *pgxpool.Pool }

func NewActorDeletionService(pool *pgxpool.Pool) *ActorDeletionService {
	return &ActorDeletionService{pool: pool}
}
func (s *ActorDeletionService) HandleIdentityDeleted(ctx context.Context, did syntax.DID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return s.HardDeleteByActor(ctx, tx, did)
	})
}

func (s *ActorDeletionService) HardDeleteByActor(ctx context.Context, tx pgx.Tx, did syntax.DID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM notification_events WHERE actor_did=$1`, did); err != nil {
		return fmt.Errorf("delete actor notifications: %w", err)
	}
	return nil
}
