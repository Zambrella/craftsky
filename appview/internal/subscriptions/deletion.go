package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DeletionParticipant struct{}

func NewDeletionParticipant() *DeletionParticipant {
	return &DeletionParticipant{}
}

func (*DeletionParticipant) BeginDeletion(ctx context.Context, tx pgx.Tx, owner syntax.DID, now time.Time) (bool, error) {
	command, err := tx.Exec(ctx, `
		UPDATE billing_accounts
		SET requested_generation=requested_generation+1,
			deletion_requested_generation=requested_generation+1,
			reconciliation_requested_at=$2,next_attempt_at=$2,updated_at=$2
		WHERE owner_did=$1 AND state='active'
	`, owner, now)
	if err != nil {
		return false, fmt.Errorf("request deletion subscription reconciliation: %w", err)
	}
	return command.RowsAffected() == 1, nil
}

func (*DeletionParticipant) ConfirmDeletion(ctx context.Context, tx pgx.Tx, owner syntax.DID, now time.Time) (bool, error) {
	var accountID uuid.UUID
	var deletionGeneration *int64
	var reconciledGeneration int64
	err := tx.QueryRow(ctx, `
		SELECT id,deletion_requested_generation,reconciled_generation
		FROM billing_accounts
		WHERE owner_did=$1 AND state='active'
		FOR UPDATE
	`, owner).Scan(&accountID, &deletionGeneration, &reconciledGeneration)
	billingOwner := err == nil
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return false, fmt.Errorf("lock deletion billing account: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE billing_licenses
		SET assigned_did=NULL,assigned_at=NULL,updated_at=$2
		WHERE assigned_did=$1
	`, owner, now); err != nil {
		return false, fmt.Errorf("clear deleted subscription target: %w", err)
	}
	if !billingOwner {
		return false, nil
	}
	if deletionGeneration == nil || reconciledGeneration < *deletionGeneration {
		return false, ErrProviderBillingMustBeResolved
	}
	var blocking bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM provider_subscriptions
			WHERE billing_account_id=$1
			  AND (
				pending_payment
				OR auto_renewal_status IS DISTINCT FROM 'will_not_renew'
			  )
		)
	`, accountID).Scan(&blocking); err != nil {
		return false, fmt.Errorf("check provider billing deletion state: %w", err)
	}
	if blocking {
		return false, ErrProviderBillingMustBeResolved
	}
	if _, err := tx.Exec(ctx, `DELETE FROM provider_subscriptions WHERE billing_account_id=$1`, accountID); err != nil {
		return false, fmt.Errorf("delete closed provider subscriptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE billing_accounts
		SET owner_did=NULL,state='closed',closed_at=$2,
			requested_generation=0,reconciled_generation=0,claimed_generation=NULL,
			reconciliation_requested_at=NULL,reconciled_at=NULL,
			reconciliation_attempts=0,next_attempt_at=NULL,
			lease_token=NULL,lease_expires_at=NULL,deletion_requested_generation=NULL,
			updated_at=$2
		WHERE id=$1
	`, accountID, now); err != nil {
		return false, fmt.Errorf("close billing account: %w", err)
	}
	return true, nil
}
