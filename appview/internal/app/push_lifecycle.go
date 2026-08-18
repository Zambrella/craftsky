package app

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/push"
)

type pushOwnerLifecycleFence struct {
	store *ownerlifecycle.Store
}

func (fence pushOwnerLifecycleFence) WithActiveOwners(
	ctx context.Context,
	owners []syntax.DID,
	callback func(context.Context) error,
) error {
	if fence.store == nil || callback == nil {
		return errors.New("push owner lifecycle fence unavailable")
	}
	canonical, err := ownerlifecycle.CanonicalOwners(owners)
	if err != nil {
		return err
	}
	expected := make([]ownerlifecycle.ExpectedOwner, 0, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := fence.store.Get(ctx, owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return push.ErrDeliveryLifecycleInactive
		}
		if err != nil {
			return err
		}
		if lifecycle.State != ownerlifecycle.StateActive {
			return push.ErrDeliveryLifecycleInactive
		}
		expected = append(expected, ownerlifecycle.ExpectedOwner{
			Owner: owner, Generation: lifecycle.Generation,
		})
	}
	err = fence.store.WithActiveEffects(ctx, expected, callback)
	if errors.Is(err, ownerlifecycle.ErrOwnerNotActive) ||
		errors.Is(err, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(err, ownerlifecycle.ErrGenerationChanged) ||
		errors.Is(err, pgx.ErrNoRows) {
		return push.ErrDeliveryLifecycleInactive
	}
	return err
}

var _ push.DeliveryLifecycleFence = pushOwnerLifecycleFence{}
