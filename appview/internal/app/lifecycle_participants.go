package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// composeTransitionParticipants preserves the repository-wide row-lock order
// while allowing independently owned lifecycle cleanup components to share one
// transition transaction. A missing participant is a startup wiring error,
// not a reason to silently skip invalidation.
func composeTransitionParticipants(
	participants ...ownerlifecycle.TransitionParticipant,
) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		for _, participant := range participants {
			if participant == nil {
				return errors.New("owner lifecycle transition participant is unavailable")
			}
			if err := participant(ctx, tx, before, after); err != nil {
				return err
			}
		}
		return nil
	}
}
