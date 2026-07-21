package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
)

type instagramMembershipInactivator interface {
	InactivateMembershipTx(context.Context, pgx.Tx, syntax.DID, time.Time) error
}

// profileMembershipDeletion keeps notification cleanup, Instagram private-data
// inactivation, and removal of the current-membership row in one transaction.
// Rejoining therefore cannot silently reactivate a link or retained import.
type profileMembershipDeletion struct {
	notifications notifications.ActorDeletion
	instagram     instagramMembershipInactivator
	now           func() time.Time
}

var _ notifications.ActorDeletion = (*profileMembershipDeletion)(nil)

func (d *profileMembershipDeletion) HardDeleteByActor(ctx context.Context, tx pgx.Tx, did syntax.DID) error {
	if d == nil || d.notifications == nil || d.instagram == nil || d.now == nil {
		return errors.New("profile membership deletion lifecycle is unavailable")
	}
	if err := d.notifications.HardDeleteByActor(ctx, tx, did); err != nil {
		return fmt.Errorf("delete actor notifications: %w", err)
	}
	if err := d.instagram.InactivateMembershipTx(ctx, tx, did, d.now().UTC()); err != nil {
		return fmt.Errorf("inactivate Instagram membership: %w", err)
	}
	return nil
}

// terminalIdentityDeletion composes idempotent terminal cleanup services at
// Tap's identity-deletion boundary. A failure prevents acknowledgement, so a
// replay safely retries every handler.
type terminalIdentityDeletion struct {
	handlers []tap.IdentityDeletionHandler
}

var _ tap.IdentityDeletionHandler = (*terminalIdentityDeletion)(nil)

func (d *terminalIdentityDeletion) HandleIdentityDeleted(ctx context.Context, did syntax.DID) error {
	if d == nil || len(d.handlers) == 0 {
		return errors.New("terminal identity deletion lifecycle is unavailable")
	}
	for _, handler := range d.handlers {
		if handler == nil {
			return errors.New("terminal identity deletion handler is unavailable")
		}
		if err := handler.HandleIdentityDeleted(ctx, did); err != nil {
			return fmt.Errorf("terminal identity deletion: %w", err)
		}
	}
	return nil
}
