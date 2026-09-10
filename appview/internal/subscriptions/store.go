package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool     *pgxpool.Pool
	observer BillingObserver
}

func NewStore(pool *pgxpool.Pool, observers ...BillingObserver) *Store {
	store := &Store{pool: pool}
	if len(observers) > 0 {
		store.observer = observers[0]
	}
	return store
}

func (s *Store) EnsureAccount(ctx context.Context, owner syntax.DID) (BillingAccount, bool, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO billing_accounts(owner_did, revenuecat_app_user_id)
		VALUES ($1, gen_random_uuid())
		ON CONFLICT (owner_did) DO NOTHING
		RETURNING id, owner_did, revenuecat_app_user_id,
			requested_generation, reconciled_generation,
			reconciliation_requested_at, reconciled_at
	`, owner)
	account, err := scanBillingAccount(row)
	if err == nil {
		return account, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BillingAccount{}, false, fmt.Errorf("create billing account: %w", err)
	}

	account, err = s.OwnerAccount(ctx, owner)
	return account, false, err
}

func (s *Store) OwnerAccount(ctx context.Context, owner syntax.DID) (BillingAccount, error) {
	account, err := scanBillingAccount(s.pool.QueryRow(ctx, `
		SELECT id, owner_did, revenuecat_app_user_id,
			requested_generation, reconciled_generation,
			reconciliation_requested_at, reconciled_at
		FROM billing_accounts
		WHERE owner_did=$1 AND state='active'
	`, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingAccount{}, ErrBillingAccountNotFound
	}
	if err != nil {
		return BillingAccount{}, fmt.Errorf("read billing account: %w", err)
	}
	return account, nil
}

func (s *Store) OwnerState(ctx context.Context, owner syntax.DID, _ time.Time) (BillingState, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BillingState{}, fmt.Errorf("begin billing state read: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	account, err := scanBillingAccount(tx.QueryRow(ctx, `
		SELECT id,owner_did,revenuecat_app_user_id,requested_generation,reconciled_generation,reconciliation_requested_at,reconciled_at
		FROM billing_accounts WHERE owner_did=$1 AND state='active'
	`, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingState{}, ErrBillingAccountNotFound
	}
	if err != nil {
		return BillingState{}, fmt.Errorf("read billing state account: %w", err)
	}
	state := BillingState{
		BillingAccountID: account.ID, RevenueCatAppUserID: account.RevenueCatAppUserID,
		RequestedGeneration: account.RequestedGeneration, ReconciledGeneration: account.ReconciledGeneration,
		ReconciliationRequested: account.ReconciliationRequested, ReconciledAt: account.ReconciledAt,
		ReconciliationStale: account.RequestedGeneration > account.ReconciledGeneration,
		Subscriptions:       []BillingSubscription{}, Licenses: []BillingLicense{},
	}
	rows, err := tx.Query(ctx, `
		SELECT id,product_id,store,status,gives_access,pending_payment,auto_renewal_status,
			current_period_starts_at,current_period_ends_at,ends_at,anomaly
		FROM provider_subscriptions WHERE billing_account_id=$1 ORDER BY created_at,id
	`, account.ID)
	if err != nil {
		return BillingState{}, fmt.Errorf("read owner subscriptions: %w", err)
	}
	for rows.Next() {
		var subscription BillingSubscription
		if err := rows.Scan(&subscription.ID, &subscription.ProductID, &subscription.Store, &subscription.Status,
			&subscription.GivesAccess, &subscription.PendingPayment, &subscription.AutoRenewalStatus,
			&subscription.CurrentPeriodStartsAt, &subscription.CurrentPeriodEndsAt, &subscription.EndsAt,
			&subscription.Anomaly); err != nil {
			rows.Close()
			return BillingState{}, err
		}
		state.Subscriptions = append(state.Subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return BillingState{}, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `
		SELECT license.id,license.tier,license.assigned_did,license.assigned_at,license.assignable,license.anomaly
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.billing_account_id=$1 ORDER BY license.created_at,license.id
	`, account.ID)
	if err != nil {
		return BillingState{}, fmt.Errorf("read owner licenses: %w", err)
	}
	for rows.Next() {
		var license BillingLicense
		if err := rows.Scan(&license.ID, &license.Tier, &license.AssignedDID, &license.AssignedAt, &license.Assignable, &license.Anomaly); err != nil {
			rows.Close()
			return BillingState{}, err
		}
		state.Licenses = append(state.Licenses, license)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return BillingState{}, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return BillingState{}, err
	}
	return state, nil
}

type billingAccountRow interface {
	Scan(...any) error
}

func scanBillingAccount(row billingAccountRow) (BillingAccount, error) {
	var account BillingAccount
	err := row.Scan(
		&account.ID,
		&account.OwnerDID,
		&account.RevenueCatAppUserID,
		&account.RequestedGeneration,
		&account.ReconciledGeneration,
		&account.ReconciliationRequested,
		&account.ReconciledAt,
	)
	return account, err
}

func (s *Store) SelfAccess(ctx context.Context, did syntax.DID, _ time.Time) (SelfAccess, error) {
	var assignment LocalAssignment
	err := s.pool.QueryRow(ctx, `
		SELECT license.tier,
			subscription.gives_access
				AND subscription.environment='production'
				AND subscription.app_id IS NOT NULL
				AND subscription.mapped_tier=license.tier
				AND subscription.anomaly = 'none',
			subscription.current_period_ends_at
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE license.assigned_did=$1
	`, did).Scan(&assignment.Tier, &assignment.GivesAccess, &assignment.AccessEndsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProjectSelfAccess(did, nil), nil
	}
	if err != nil {
		return SelfAccess{}, fmt.Errorf("read self subscription access: %w", err)
	}
	return ProjectSelfAccess(did, &assignment), nil
}

func (s *Store) ApplySnapshot(ctx context.Context, claim SnapshotClaim, snapshot CompleteSnapshot, catalog *Catalog) error {
	if !snapshot.Complete {
		return errors.New("incomplete subscription snapshot")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin subscription snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var reconciledGeneration int64
	var state string
	err = tx.QueryRow(ctx, `
		SELECT reconciled_generation, state
		FROM billing_accounts
		WHERE id=$1 AND claimed_generation=$2 AND lease_token=$3
		FOR UPDATE
	`, claim.BillingAccountID, claim.Generation, claim.LeaseToken).Scan(&reconciledGeneration, &state)
	if errors.Is(err, pgx.ErrNoRows) || state != "active" || claim.Generation <= reconciledGeneration {
		return ErrStaleSnapshot
	}
	if err != nil {
		return fmt.Errorf("lock subscription snapshot claim: %w", err)
	}

	assigned := make(map[string]*syntax.DID)
	existingTiers := make(map[string]*Tier)
	rows, err := tx.Query(ctx, `
		SELECT subscription.revenuecat_subscription_id, license.assigned_did, license.tier
		FROM provider_subscriptions subscription
		JOIN billing_licenses license ON license.provider_subscription_id=subscription.id
		WHERE subscription.billing_account_id=$1
	`, claim.BillingAccountID)
	if err != nil {
		return fmt.Errorf("read existing subscription assignments: %w", err)
	}
	for rows.Next() {
		var subscriptionID string
		var did *syntax.DID
		var tier Tier
		if err := rows.Scan(&subscriptionID, &did, &tier); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing subscription assignment: %w", err)
		}
		assigned[subscriptionID] = did
		existingTiers[subscriptionID] = &tier
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate existing subscription assignments: %w", err)
	}
	rows.Close()

	type mappedSubscription struct {
		snapshot ProviderSubscriptionSnapshot
		mapping  ProductMapping
		allowed  bool
	}
	mapped := make([]mappedSubscription, 0, len(snapshot.Subscriptions))
	candidates := make([]LicenseCandidate, 0, len(snapshot.Subscriptions))
	seen := make(map[string]struct{}, len(snapshot.Subscriptions))
	for _, provider := range snapshot.Subscriptions {
		mapping, allowed := catalog.Authorize(catalog.projectID, provider.Environment, provider.ProductID)
		item := mappedSubscription{snapshot: provider, mapping: mapping, allowed: allowed}
		mapped = append(mapped, item)
		seen[provider.ID] = struct{}{}
		if !allowed {
			continue
		}
		candidate := LicenseCandidate{
			SubscriptionID: provider.ID, Tier: mapping.Tier, GivesAccess: provider.GivesAccess,
			ExistingTier: existingTiers[provider.ID], AssignedDID: assigned[provider.ID],
		}
		if pending, ok := catalog.Authorize(catalog.projectID, provider.Environment, provider.PendingProductID); ok {
			candidate.PendingTier = &pending.Tier
		}
		candidates = append(candidates, candidate)
	}
	decisions := ClassifySnapshotAnomalies(candidates)
	anomalyCount := len(mapped) - len(candidates)
	for _, decision := range decisions {
		if decision.Anomaly != AnomalyNone {
			anomalyCount++
		}
	}

	for _, item := range mapped {
		var appID any
		var mappedTier any
		providerAnomaly := "unsupported"
		if item.allowed {
			appID = item.mapping.AppID
			mappedTier = item.mapping.Tier
			providerAnomaly = anomalyValue(decisions[item.snapshot.ID].Anomaly)
		}
		var providerID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO provider_subscriptions(
				billing_account_id, project_id, revenuecat_subscription_id,
				product_id, pending_product_id, app_id, store, environment, status,
				gives_access, pending_payment, auto_renewal_status, starts_at,
				current_period_starts_at, current_period_ends_at, ends_at,
				mapped_tier, accepted_generation, anomaly
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
			ON CONFLICT (project_id, revenuecat_subscription_id) DO UPDATE SET
				product_id=EXCLUDED.product_id,
				pending_product_id=EXCLUDED.pending_product_id,
				app_id=EXCLUDED.app_id,
				store=EXCLUDED.store,
				environment=EXCLUDED.environment,
				status=EXCLUDED.status,
				gives_access=EXCLUDED.gives_access,
				pending_payment=EXCLUDED.pending_payment,
				auto_renewal_status=EXCLUDED.auto_renewal_status,
				starts_at=EXCLUDED.starts_at,
				current_period_starts_at=EXCLUDED.current_period_starts_at,
				current_period_ends_at=EXCLUDED.current_period_ends_at,
				ends_at=EXCLUDED.ends_at,
				mapped_tier=EXCLUDED.mapped_tier,
				accepted_generation=EXCLUDED.accepted_generation,
				anomaly=EXCLUDED.anomaly,
				updated_at=now()
			WHERE provider_subscriptions.billing_account_id=EXCLUDED.billing_account_id
			  AND provider_subscriptions.accepted_generation <= EXCLUDED.accepted_generation
			RETURNING id
		`, claim.BillingAccountID, catalog.projectID, item.snapshot.ID,
			item.snapshot.ProductID, nullableString(item.snapshot.PendingProductID), appID,
			item.snapshot.Store, item.snapshot.Environment, item.snapshot.Status,
			item.snapshot.GivesAccess, item.snapshot.PendingPayment,
			nullableString(item.snapshot.AutoRenewalStatus), item.snapshot.StartsAt,
			item.snapshot.CurrentPeriodStartsAt, item.snapshot.CurrentPeriodEndsAt,
			item.snapshot.EndsAt, mappedTier, claim.Generation, providerAnomaly,
		).Scan(&providerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaleSnapshot
		}
		if err != nil {
			return fmt.Errorf("upsert provider subscription: %w", err)
		}
		if item.allowed {
			decision := decisions[item.snapshot.ID]
			licenseTier := item.mapping.Tier
			if existingTier := existingTiers[item.snapshot.ID]; decision.Anomaly == AnomalyCrossTierConflict && existingTier != nil && *existingTier != item.mapping.Tier {
				licenseTier = *existingTier
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO billing_licenses(provider_subscription_id,tier,assignable,anomaly)
				VALUES ($1,$2,$3,$4)
				ON CONFLICT (provider_subscription_id) DO UPDATE SET
					tier=EXCLUDED.tier, assignable=EXCLUDED.assignable,
					anomaly=EXCLUDED.anomaly, updated_at=now()
			`, providerID, licenseTier, decision.Assignable, anomalyValue(decision.Anomaly)); err != nil {
				return fmt.Errorf("upsert billing license: %w", err)
			}
		} else if _, err := tx.Exec(ctx, `
			UPDATE billing_licenses
			SET assignable=false,updated_at=now()
			WHERE provider_subscription_id=$1
		`, providerID); err != nil {
			return fmt.Errorf("disable unsupported billing license: %w", err)
		}
	}

	knownRows, err := tx.Query(ctx, `
		SELECT revenuecat_subscription_id, id FROM provider_subscriptions
		WHERE billing_account_id=$1
	`, claim.BillingAccountID)
	if err != nil {
		return fmt.Errorf("read known subscriptions: %w", err)
	}
	var absent []uuid.UUID
	for knownRows.Next() {
		var subscriptionID string
		var id uuid.UUID
		if err := knownRows.Scan(&subscriptionID, &id); err != nil {
			knownRows.Close()
			return err
		}
		if _, ok := seen[subscriptionID]; !ok {
			absent = append(absent, id)
		}
	}
	knownRows.Close()
	for _, id := range absent {
		if _, err := tx.Exec(ctx, `
			UPDATE provider_subscriptions
			SET gives_access=false, accepted_generation=$2, updated_at=now()
			WHERE id=$1
		`, id, claim.Generation); err != nil {
			return fmt.Errorf("mark absent subscription inaccessible: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE billing_licenses
			SET assignable=false, updated_at=now()
			WHERE provider_subscription_id=$1
		`, id); err != nil {
			return fmt.Errorf("mark absent license inaccessible: %w", err)
		}
	}

	command, err := tx.Exec(ctx, `
		UPDATE billing_accounts
		SET reconciled_generation=$2, reconciled_at=$3,
			claimed_generation=NULL, lease_token=NULL, lease_expires_at=NULL,
			reconciliation_attempts=0, next_attempt_at=NULL, updated_at=$3
		WHERE id=$1 AND claimed_generation=$2 AND lease_token=$4
	`, claim.BillingAccountID, claim.Generation, time.Now().UTC(), claim.LeaseToken)
	if err != nil || command.RowsAffected() != 1 {
		return ErrStaleSnapshot
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit subscription snapshot: %w", err)
	}
	if s.observer != nil {
		s.observer.ObserveSubscriptionAnomalies(ctx, anomalyCount)
	}
	return nil
}

func anomalyValue(anomaly Anomaly) string {
	if anomaly == AnomalyNone {
		return "none"
	}
	return string(anomaly)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) RequestReconciliation(ctx context.Context, accountID uuid.UUID, now time.Time) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE billing_accounts
		SET requested_generation=requested_generation+1,
			reconciliation_requested_at=$2, next_attempt_at=$2, updated_at=$2
		WHERE id=$1 AND state='active'
	`, accountID, now)
	if err != nil {
		return fmt.Errorf("request subscription reconciliation: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrBillingAccountNotFound
	}
	return nil
}

// ScheduleActiveAccounts advances every active billing account through the same
// generation-fenced queue used by webhook and owner refresh triggers.
func (s *Store) ScheduleActiveAccounts(ctx context.Context, now time.Time) (int64, error) {
	command, err := s.pool.Exec(ctx, `
		UPDATE billing_accounts
		SET requested_generation=requested_generation+1,
			reconciliation_requested_at=$1, next_attempt_at=$1, updated_at=$1
		WHERE state='active'
	`, now)
	if err != nil {
		return 0, fmt.Errorf("schedule subscription reconciliation: %w", err)
	}
	return command.RowsAffected(), nil
}

func (s *Store) ClaimReconciliation(ctx context.Context, now time.Time, leaseDuration time.Duration) (SnapshotClaim, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SnapshotClaim{}, false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var claim SnapshotClaim
	err = tx.QueryRow(ctx, `
		SELECT id, revenuecat_app_user_id, requested_generation
		FROM billing_accounts
		WHERE state='active'
		  AND requested_generation > reconciled_generation
		  AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
		  AND (lease_expires_at IS NULL OR lease_expires_at <= $1)
		ORDER BY reconciliation_requested_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, now).Scan(&claim.BillingAccountID, &claim.RevenueCatAppUserID, &claim.Generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return SnapshotClaim{}, false, nil
	}
	if err != nil {
		return SnapshotClaim{}, false, fmt.Errorf("claim subscription reconciliation: %w", err)
	}
	claim.LeaseToken = uuid.New()
	if _, err := tx.Exec(ctx, `
		UPDATE billing_accounts
		SET claimed_generation=$2, lease_token=$3, lease_expires_at=$4,
			reconciliation_attempts=reconciliation_attempts+1, updated_at=$1
		WHERE id=$5
	`, now, claim.Generation, claim.LeaseToken, now.Add(leaseDuration), claim.BillingAccountID); err != nil {
		return SnapshotClaim{}, false, fmt.Errorf("save subscription reconciliation claim: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SnapshotClaim{}, false, err
	}
	return claim, true, nil
}

func (s *Store) AcceptRevenueCatEvent(ctx context.Context, event RevenueCatEvent, acceptedAt time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var accountID *uuid.UUID
	var state string
	err = tx.QueryRow(ctx, `
		SELECT id, state FROM billing_accounts WHERE revenuecat_app_user_id=$1 FOR UPDATE
	`, event.RevenueCatAppUserID).Scan(&accountID, &state)
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return false, fmt.Errorf("map RevenueCat event account: %w", err)
	}
	outcome := "ignored_unmapped"
	if err == nil && state == "active" {
		outcome = "reconciliation_queued"
	} else if err == nil {
		outcome = "ignored_closed"
	}
	command, err := tx.Exec(ctx, `
		INSERT INTO revenuecat_events(
			event_id,event_type,billing_account_id,event_occurred_at,accepted_at,outcome
		) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT (event_id) DO NOTHING
	`, event.ID, event.Type, accountID, nullableTime(event.OccurredAt), acceptedAt, outcome)
	if err != nil {
		return false, fmt.Errorf("persist RevenueCat event: %w", err)
	}
	if command.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if outcome == "reconciliation_queued" {
		if _, err := tx.Exec(ctx, `
			UPDATE billing_accounts
			SET requested_generation=requested_generation+1,
				reconciliation_requested_at=$2, next_attempt_at=$2, updated_at=$2
			WHERE id=$1 AND state='active'
		`, accountID, acceptedAt); err != nil {
			return false, fmt.Errorf("queue RevenueCat event reconciliation: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) FailReconciliation(ctx context.Context, claim SnapshotClaim, now time.Time, retryDelay time.Duration) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE billing_accounts
		SET claimed_generation=NULL, lease_token=NULL, lease_expires_at=NULL,
			next_attempt_at=$4, updated_at=$3
		WHERE id=$1 AND claimed_generation=$2 AND lease_token=$5 AND state='active'
	`, claim.BillingAccountID, claim.Generation, now, now.Add(retryDelay), claim.LeaseToken)
	if err != nil {
		return fmt.Errorf("release failed reconciliation claim: %w", err)
	}
	return nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
