package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"social.craftsky/appview/internal/ownerlifecycle"
)

const assignmentCooldown = 7 * 24 * time.Hour

func CanChangeAssignmentTarget(lastTargetChange *time.Time, now time.Time) bool {
	return lastTargetChange == nil || !now.Before(lastTargetChange.Add(assignmentCooldown))
}

func (s *Store) Assign(ctx context.Context, params AssignParams) (assignment Assignment, resultErr error) {
	defer func() {
		if s.observer != nil {
			s.observer.ObserveSubscriptionAssignment(ctx, "assign", assignmentOutcome(resultErr))
		}
	}()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Assignment{}, fmt.Errorf("begin billing assignment: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	lifecycles, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, []syntax.DID{params.TargetDID})
	if err != nil {
		return Assignment{}, fmt.Errorf("lock billing assignment target lifecycle: %w", err)
	}
	if lifecycle, ok := lifecycles[params.TargetDID]; !ok || lifecycle.State != ownerlifecycle.StateActive {
		return Assignment{}, ErrAssignmentTargetIneligible
	}

	var accountID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM billing_accounts
		WHERE owner_did=$1 AND state='active'
		FOR UPDATE
	`, params.OwnerDID).Scan(&accountID); errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrLicenseNotFound
	} else if err != nil {
		return Assignment{}, fmt.Errorf("lock billing owner: %w", err)
	}

	var currentDevice *string
	if err := tx.QueryRow(ctx, `
		SELECT session.last_device_id
		FROM craftsky_profiles profile
		JOIN craftsky_sessions session ON session.account_did=profile.did
		WHERE profile.did=$1
		  AND appview_owner_is_active(profile.did)
		  AND session.lifecycle_state='active'
		  AND session.revoked_at IS NULL
		  AND session.idle_expires_at > $2
		ORDER BY session.last_seen_at DESC
		LIMIT 1
		FOR UPDATE OF profile, session
	`, params.TargetDID, params.Now).Scan(&currentDevice); errors.Is(err, pgx.ErrNoRows) || currentDevice == nil || *currentDevice != params.DeviceID {
		return Assignment{}, ErrAssignmentTargetIneligible
	} else if err != nil {
		return Assignment{}, fmt.Errorf("revalidate billing assignment target: %w", err)
	}

	var assignedDID *syntax.DID
	var lastTargetChange *time.Time
	var assignable bool
	if err := tx.QueryRow(ctx, `
		SELECT license.assigned_did, license.last_target_change_at, license.assignable
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE license.id=$1 AND subscription.billing_account_id=$2
		FOR UPDATE OF subscription, license
	`, params.LicenseID, accountID).Scan(&assignedDID, &lastTargetChange, &assignable); errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrLicenseNotFound
	} else if err != nil {
		return Assignment{}, fmt.Errorf("lock billing license: %w", err)
	}
	if !assignable && (assignedDID == nil || *assignedDID != params.TargetDID) {
		return Assignment{}, ErrLicenseNotFound
	}
	if assignedDID != nil && *assignedDID != params.TargetDID && !CanChangeAssignmentTarget(lastTargetChange, params.Now) {
		return Assignment{}, ErrAssignmentCooldown
	}
	locks := []syntax.DID{params.TargetDID}
	if assignedDID != nil && *assignedDID != params.TargetDID {
		locks = append(locks, *assignedDID)
	}
	slices.Sort(locks)
	for _, did := range locks {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, accessFenceKey(did)); err != nil {
			return Assignment{}, fmt.Errorf("fence billing assignment access: %w", err)
		}
	}
	var oldPaid bool
	if assignedDID != nil && *assignedDID != params.TargetDID {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM billing_licenses license
			JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
			WHERE license.id=$1 AND license.assigned_did=$2 AND license.tier IN ('plus','business')
			 AND subscription.gives_access AND subscription.environment=$3
			 AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
			 AND subscription.anomaly='none'
		)`, params.LicenseID, *assignedDID, s.accessEnvironment).Scan(&oldPaid); err != nil {
			return Assignment{}, fmt.Errorf("read prior assignment access: %w", err)
		}
	}

	var targetChange any
	if assignedDID != nil && *assignedDID != params.TargetDID {
		targetChange = params.Now
	} else {
		targetChange = lastTargetChange
	}
	_, err = tx.Exec(ctx, `
		UPDATE billing_licenses
		SET assigned_did=$2, assigned_at=$3, last_target_change_at=$4, updated_at=$3
		WHERE id=$1
	`, params.LicenseID, params.TargetDID, params.Now, targetChange)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Assignment{}, ErrDIDAlreadyAssigned
		}
		return Assignment{}, fmt.Errorf("assign billing license: %w", err)
	}
	if oldPaid {
		if _, err := tx.Exec(ctx, `DELETE FROM profile_pins WHERE owner_did=$1`, *assignedDID); err != nil {
			return Assignment{}, fmt.Errorf("clear reassigned profile pins: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Assignment{}, fmt.Errorf("commit billing assignment: %w", err)
	}
	return Assignment{LicenseID: params.LicenseID, TargetDID: params.TargetDID, AssignedAt: params.Now}, nil
}

func (s *Store) Unassign(ctx context.Context, params UnassignParams) (resultErr error) {
	defer func() {
		if s.observer != nil {
			s.observer.ObserveSubscriptionAssignment(ctx, "unassign", assignmentOutcome(resultErr))
		}
	}()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin billing unassignment: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var accountID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM billing_accounts WHERE owner_did=$1 AND state='active' FOR UPDATE`, params.OwnerDID).Scan(&accountID); errors.Is(err, pgx.ErrNoRows) {
		return ErrLicenseNotFound
	} else if err != nil {
		return fmt.Errorf("lock billing owner for unassignment: %w", err)
	}
	var previous *syntax.DID
	if err := tx.QueryRow(ctx, `
		SELECT license.assigned_did
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE license.id=$1 AND subscription.billing_account_id=$2
		FOR UPDATE OF subscription, license
	`, params.LicenseID, accountID).Scan(&previous); errors.Is(err, pgx.ErrNoRows) {
		return ErrLicenseNotFound
	} else if err != nil {
		return fmt.Errorf("lock billing license for unassignment: %w", err)
	}
	if previous != nil {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, accessFenceKey(*previous)); err != nil {
			return fmt.Errorf("fence subscription unassignment: %w", err)
		}
	}
	var previouslyPaid bool
	if previous != nil {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM billing_licenses license
			JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
			WHERE license.id=$1 AND license.assigned_did=$2 AND license.tier IN ('plus','business')
			  AND subscription.gives_access AND subscription.environment=$3
			  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
			  AND subscription.anomaly='none'
		)`, params.LicenseID, *previous, s.accessEnvironment).Scan(&previouslyPaid); err != nil {
			return fmt.Errorf("read prior subscription access: %w", err)
		}
	}
	command, err := tx.Exec(ctx, `
		UPDATE billing_licenses license
		SET assigned_did=NULL, assigned_at=NULL, updated_at=$3
		FROM provider_subscriptions subscription
		JOIN billing_accounts account ON account.id=subscription.billing_account_id
		WHERE license.id=$1
		  AND license.provider_subscription_id=subscription.id
		  AND account.owner_did=$2
		  AND account.state='active'
		  AND license.assigned_did IS NOT DISTINCT FROM $4
	`, params.LicenseID, params.OwnerDID, params.Now, previous)
	if err != nil {
		return fmt.Errorf("unassign billing license: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrLicenseNotFound
	}
	if previouslyPaid {
		if _, err := tx.Exec(ctx, `DELETE FROM profile_pins WHERE owner_did=$1`, *previous); err != nil {
			return fmt.Errorf("clear lapsed profile pins: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit billing unassignment: %w", err)
	}
	return nil
}

func assignmentOutcome(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, ErrLicenseNotFound):
		return "not_found"
	case errors.Is(err, ErrAssignmentTargetIneligible):
		return "target_ineligible"
	case errors.Is(err, ErrDIDAlreadyAssigned):
		return "already_assigned"
	case errors.Is(err, ErrAssignmentCooldown):
		return "cooldown"
	default:
		return "store_error"
	}
}
