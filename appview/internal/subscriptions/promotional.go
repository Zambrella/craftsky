package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func promotionalExpiry(snapshot ProviderSubscriptionSnapshot) *time.Time {
	var expiry *time.Time
	for _, bound := range []*time.Time{snapshot.EndsAt, snapshot.CurrentPeriodEndsAt} {
		if bound != nil && (expiry == nil || bound.Before(*expiry)) {
			value := bound.UTC()
			expiry = &value
		}
	}
	return expiry
}

func applyPromotionalGrants(ctx context.Context, tx pgx.Tx, claim SnapshotClaim, snapshots []ProviderSubscriptionSnapshot, catalog *Catalog) error {
	seen := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		tier, allowed := catalog.AuthorizePromotional(snapshot.Environment, snapshot.Entitlements)
		if !allowed {
			continue
		}
		if snapshot.ID == "" {
			return errors.New("promotional grant identity missing")
		}
		seen = append(seen, snapshot.ID)
		command, err := tx.Exec(ctx, `
			INSERT INTO billing_promotional_grants(
				billing_account_id,project_id,revenuecat_subscription_id,environment,tier,gives_access,expires_at,accepted_generation
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT(project_id,revenuecat_subscription_id) DO UPDATE SET
				environment=EXCLUDED.environment,tier=EXCLUDED.tier,gives_access=EXCLUDED.gives_access,
				expires_at=EXCLUDED.expires_at,accepted_generation=EXCLUDED.accepted_generation,updated_at=now()
			WHERE billing_promotional_grants.billing_account_id=EXCLUDED.billing_account_id
				AND billing_promotional_grants.accepted_generation<=EXCLUDED.accepted_generation
		`, claim.BillingAccountID, catalog.projectID, snapshot.ID, snapshot.Environment,
			tier, snapshot.GivesAccess, promotionalExpiry(snapshot), claim.Generation)
		if err != nil {
			return fmt.Errorf("store promotional grant: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrStaleSnapshot
		}
	}
	_, err := tx.Exec(ctx, `UPDATE billing_promotional_grants
		SET gives_access=false,accepted_generation=$3,updated_at=now()
		WHERE billing_account_id=$1 AND environment=$2 AND NOT (revenuecat_subscription_id=ANY($4))
	`, claim.BillingAccountID, catalog.environment, claim.Generation, seen)
	if err != nil {
		return fmt.Errorf("close absent promotional grants: %w", err)
	}
	return nil
}
