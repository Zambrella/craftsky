package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckoutExpirer interface {
	ExpireCheckout(context.Context, string) (CheckoutSession, error)
}

// CancelPendingWebCheckout releases the durable fence only after Stripe proves
// the original session expired. A payment racing cancellation remains blocked.
func (s *Store) CancelPendingWebCheckout(ctx context.Context, did syntax.DID, now time.Time) error {
	expirer, ok := s.checkoutProvider.(CheckoutExpirer)
	if !ok {
		return ErrWebCheckoutUnavailable
	}
	return s.withCheckoutFence(ctx, did, false, func(conn *pgxpool.Conn) error {
		state, err := s.ownerStateUsing(ctx, conn, did, now)
		if err != nil {
			return err
		}
		op, err := s.readCheckoutOperation(ctx, conn, state.BillingAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		session, resolved, err := s.recoverCheckout(ctx, conn, did, state, op, now)
		if err != nil {
			return err
		}
		if resolved {
			return nil
		}
		if session.Status != "open" {
			return ErrSubscriptionAlreadyExists
		}
		var expired CheckoutSession
		err = s.withCheckoutStanding(ctx, conn, did, func(effectCtx context.Context) error {
			expired, err = expirer.ExpireCheckout(effectCtx, session.ID)
			return err
		})
		if err != nil {
			return err
		}
		if expired.ID != session.ID || expired.Status != "expired" || !validCheckoutSession(expired, state.RevenueCatAppUserID, s.accessEnvironment) {
			return ErrWebCheckoutUnavailable
		}
		return s.deleteCheckoutOperation(ctx, conn, state.BillingAccountID)
	})
}
