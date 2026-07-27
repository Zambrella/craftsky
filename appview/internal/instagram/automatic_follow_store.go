package instagram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
)

const (
	AutomaticFollowLeaseDuration  = time.Minute
	AutomaticFollowInitialBackoff = time.Second
	AutomaticFollowMaxBackoff     = 5 * time.Minute
)

var ErrAutomaticFollowLeaseLost = errors.New("automatic follow lease lost")

type AutomaticFollowStoreOptions struct {
	LeaseDuration  time.Duration
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultAutomaticFollowStoreOptions() AutomaticFollowStoreOptions {
	return AutomaticFollowStoreOptions{
		LeaseDuration:  AutomaticFollowLeaseDuration,
		InitialBackoff: AutomaticFollowInitialBackoff,
		MaxBackoff:     AutomaticFollowMaxBackoff,
	}
}

func (options AutomaticFollowStoreOptions) valid() bool {
	return options.LeaseDuration > 0 &&
		options.LeaseDuration <= AutomaticFollowLeaseDuration &&
		options.InitialBackoff > 0 &&
		options.InitialBackoff <= AutomaticFollowInitialBackoff &&
		options.MaxBackoff >= options.InitialBackoff &&
		options.MaxBackoff <= AutomaticFollowMaxBackoff
}

type automaticFollowRetryPolicy struct {
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

func defaultAutomaticFollowRetryPolicy() automaticFollowRetryPolicy {
	return automaticFollowRetryPolicy{
		initialBackoff: AutomaticFollowInitialBackoff,
		maxBackoff:     AutomaticFollowMaxBackoff,
	}
}

func nextAutomaticFollowRetry(
	policy automaticFollowRetryPolicy,
	now time.Time,
	attempts int,
) time.Time {
	backoff := policy.initialBackoff
	for attempt := 1; attempt < attempts && backoff < policy.maxBackoff; attempt++ {
		if backoff > policy.maxBackoff/2 {
			backoff = policy.maxBackoff
			break
		}
		backoff *= 2
	}
	return now.Add(backoff).UTC()
}

type ReconcileAutomaticFollowParams struct {
	ID          uuid.UUID
	ImporterDID syntax.DID
	TargetDID   syntax.DID
	ImportID    uuid.UUID
	Username    string
	Rkey        syntax.RecordKey
	Now         time.Time
}

type ReconcileAutomaticFollowResult struct {
	Operation  AutomaticFollowOperation
	Created    bool
	Queued     bool
	Suppressed bool
}

type AutomaticFollowStore struct {
	pool          *pgxpool.Pool
	notifications AutomaticFollowNotificationService
	leaseDuration time.Duration
	retryPolicy   automaticFollowRetryPolicy
}

type AutomaticFollowNotificationService interface {
	ActivateInstagramMatch(
		context.Context,
		pgx.Tx,
		notifications.InstagramMatchActivation,
	) error
}

func NewAutomaticFollowStore(
	pool *pgxpool.Pool,
	notificationServices ...AutomaticFollowNotificationService,
) *AutomaticFollowStore {
	store, err := NewAutomaticFollowStoreWithOptions(
		pool,
		DefaultAutomaticFollowStoreOptions(),
		notificationServices...,
	)
	if err != nil {
		panic(err)
	}
	return store
}

func NewAutomaticFollowStoreWithOptions(
	pool *pgxpool.Pool,
	options AutomaticFollowStoreOptions,
	notificationServices ...AutomaticFollowNotificationService,
) (*AutomaticFollowStore, error) {
	if !options.valid() {
		return nil, errors.New("invalid automatic follow store limits")
	}
	var notificationService AutomaticFollowNotificationService
	if len(notificationServices) > 0 {
		notificationService = notificationServices[0]
	}
	return &AutomaticFollowStore{
		pool: pool, notifications: notificationService,
		leaseDuration: options.LeaseDuration,
		retryPolicy: automaticFollowRetryPolicy{
			initialBackoff: options.InitialBackoff,
			maxBackoff:     options.MaxBackoff,
		},
	}, nil
}

func (s *AutomaticFollowStore) ClaimBatch(
	ctx context.Context,
	limit int,
	now time.Time,
) ([]AutomaticFollowOperation, error) {
	if s == nil || s.pool == nil || limit < 1 || limit > AutomaticFollowBatchMax || now.IsZero() {
		return nil, errors.New("invalid automatic follow claim")
	}
	now = now.UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status='pending',
		    lease_token=NULL,
		    lease_expires_at=NULL,
		    next_attempt_at=$1,
		    last_error_code='leaseExpired',
		    updated_at=$1
		WHERE status='writing'
		  AND lease_expires_at <= $1
	`, now); err != nil {
		return nil, fmt.Errorf("recover automatic follow leases: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT
			operation.id,
			operation.owner_did,
			operation.target_did,
			COALESCE((
				SELECT handle.username_normalized
				FROM instagram_suggestion_sources source
				JOIN instagram_graph_imports import
				  ON import.id=source.import_id
				 AND import.owner_did=operation.owner_did
				 AND import.state='active'
				JOIN instagram_graph_handles handle
				  ON handle.import_id=import.id
				JOIN instagram_account_links link
				  ON link.owner_did=operation.target_did
				 AND link.username_normalized=handle.username_normalized
				WHERE source.suggestion_id=operation.suggestion_id
				ORDER BY import.created_at, import.id, handle.id
				LIMIT 1
			), ''),
			operation.rkey,
			operation.attempt_count,
			operation.created_at
		FROM pds_follow_operations operation
		WHERE operation.status='pending'
		  AND operation.next_attempt_at <= $2
		ORDER BY operation.next_attempt_at, operation.id
		LIMIT $1
		FOR UPDATE OF operation SKIP LOCKED
	`, limit, now)
	if err != nil {
		return nil, fmt.Errorf("select automatic follow claims: %w", err)
	}
	operations := make([]AutomaticFollowOperation, 0, limit)
	for rows.Next() {
		var operation AutomaticFollowOperation
		if err := rows.Scan(
			&operation.ID,
			&operation.OwnerDID,
			&operation.TargetDID,
			&operation.ImportedUsername,
			&operation.Rkey,
			&operation.AttemptCount,
			&operation.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}
		operations = append(operations, operation)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for index := range operations {
		operations[index].LeaseToken = uuid.New()
		operations[index].AttemptCount++
		if _, err := tx.Exec(ctx, `
			UPDATE pds_follow_operations
			SET status='writing',
			    lease_token=$2,
			    lease_expires_at=$3,
			    attempt_count=$4,
			    updated_at=$5
			WHERE id=$1 AND status='pending'
			`, operations[index].ID, operations[index].LeaseToken,
			now.Add(s.leaseDuration), operations[index].AttemptCount, now); err != nil {
			return nil, fmt.Errorf("lease automatic follow: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_follow_suggestions
			SET state='writing', updated_at=$2
			WHERE id=(
				SELECT suggestion_id
				FROM pds_follow_operations
				WHERE id=$1
			)
			  AND state='pending'
		`, operations[index].ID, now); err != nil {
			return nil, fmt.Errorf("mark automatic follow ledger writing: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return operations, nil
}

func (s *AutomaticFollowStore) CompleteFollowed(
	ctx context.Context,
	operation AutomaticFollowOperation,
	now time.Time,
) error {
	return s.complete(ctx, operation, AutomaticFollowFollowed, now)
}

func (s *AutomaticFollowStore) CompleteAlreadyFollowing(
	ctx context.Context,
	operation AutomaticFollowOperation,
	now time.Time,
) error {
	return s.complete(ctx, operation, AutomaticFollowAlreadyFollowing, now)
}

func (s *AutomaticFollowStore) complete(
	ctx context.Context,
	operation AutomaticFollowOperation,
	state AutomaticFollowState,
	now time.Time,
) error {
	if s == nil || s.pool == nil || operation.ID == uuid.Nil ||
		operation.LeaseToken == uuid.Nil || now.IsZero() ||
		(state != AutomaticFollowFollowed && state != AutomaticFollowAlreadyFollowing) {
		return errors.New("invalid automatic follow completion")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status=$3,
		    record_uri=CASE
		        WHEN $3='followed'
		        THEN COALESCE(record_uri, 'at://' || owner_did || '/app.bsky.graph.follow/' || rkey)
		        ELSE record_uri
		    END,
		    completed_at=COALESCE(completed_at,$4),
		    lease_token=NULL,
		    lease_expires_at=NULL,
		    last_error_code=NULL,
		    updated_at=$4
		WHERE id=$1
		  AND status='writing'
		  AND lease_token=$2
	`, operation.ID, operation.LeaseToken, state, now.UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAutomaticFollowLeaseLost
	}
	if state == AutomaticFollowFollowed && s.notifications != nil {
		if err := s.notifications.ActivateInstagramMatch(
			ctx,
			tx,
			notifications.InstagramMatchActivation{
				RecipientDID: operation.OwnerDID,
				ActorDID:     operation.TargetDID,
				OperationID:  operation.ID,
				ActivityAt:   now.UTC(),
			},
		); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state=$2,
		    terminal_at=COALESCE(terminal_at,$3),
		    updated_at=$3
		WHERE id=(
			SELECT suggestion_id FROM pds_follow_operations WHERE id=$1
		)
		  AND state='writing'
	`, operation.ID, state, now.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AutomaticFollowStore) Retry(
	ctx context.Context,
	operation AutomaticFollowOperation,
	code string,
	now time.Time,
) error {
	return s.release(
		ctx,
		operation,
		AutomaticFollowPending,
		code,
		nextAutomaticFollowRetry(s.retryPolicy, now, operation.AttemptCount),
	)
}

func (s *AutomaticFollowStore) Invalidate(
	ctx context.Context,
	operation AutomaticFollowOperation,
	code string,
	now time.Time,
) error {
	return s.release(ctx, operation, AutomaticFollowInvalidated, code, now)
}

func (s *AutomaticFollowStore) release(
	ctx context.Context,
	operation AutomaticFollowOperation,
	state AutomaticFollowState,
	code string,
	at time.Time,
) error {
	if s == nil || s.pool == nil || operation.ID == uuid.Nil ||
		operation.LeaseToken == uuid.Nil || at.IsZero() || code == "" ||
		(state != AutomaticFollowPending && state != AutomaticFollowInvalidated) {
		return errors.New("invalid automatic follow release")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status=$3,
		    lease_token=NULL,
		    lease_expires_at=NULL,
		    next_attempt_at=$4,
		    last_error_code=$5,
		    completed_at=CASE WHEN $3='invalidated' THEN COALESCE(completed_at,$4) ELSE NULL END,
		    updated_at=$4
		WHERE id=$1
		  AND status='writing'
		  AND lease_token=$2
	`, operation.ID, operation.LeaseToken, state, at.UTC(), code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrAutomaticFollowLeaseLost
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state=$2,
		    terminal_at=CASE WHEN $2='invalidated' THEN COALESCE(terminal_at,$3) ELSE NULL END,
		    updated_at=$3
		WHERE id=(
			SELECT suggestion_id FROM pds_follow_operations WHERE id=$1
		)
		  AND state='writing'
	`, operation.ID, state, at.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AutomaticFollowStore) ReconcileCandidate(
	ctx context.Context,
	params ReconcileAutomaticFollowParams,
) (ReconcileAutomaticFollowResult, error) {
	if s == nil || s.pool == nil ||
		params.ID == uuid.Nil ||
		params.ImporterDID == "" ||
		params.TargetDID == "" ||
		params.ImporterDID == params.TargetDID ||
		params.ImportID == uuid.Nil ||
		params.Rkey == "" ||
		params.Now.IsZero() {
		return ReconcileAutomaticFollowResult{}, errors.New("invalid automatic follow reconciliation")
	}
	username, err := NormalizeInstagramUsername(params.Username)
	if err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	if _, err := syntax.ParseRecordKey(string(params.Rkey)); err != nil {
		return ReconcileAutomaticFollowResult{}, errors.New("invalid automatic follow rkey")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 7))`,
		params.ImporterDID,
		params.TargetDID,
	); err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	var handleID int64
	if err := tx.QueryRow(ctx, `
		SELECT handle.id
		FROM instagram_graph_imports import
		JOIN instagram_graph_handles handle ON handle.import_id=import.id
		JOIN instagram_account_links link
		  ON link.owner_did=$4
		 AND link.username_normalized=handle.username_normalized
		 AND link.state='active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE import.id=$1
		  AND import.owner_did=$2
		  AND import.state='active'
		  AND handle.username_normalized=$3
		LIMIT 1
		FOR UPDATE OF import, handle, link
	`, params.ImportID, params.ImporterDID, username, params.TargetDID).Scan(&handleID); errors.Is(err, pgx.ErrNoRows) {
		return ReconcileAutomaticFollowResult{}, ErrInstagramResourceNotFound
	} else if err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}

	var (
		ledgerID uuid.UUID
		state    AutomaticFollowState
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO instagram_follow_suggestions (
			id, importer_did, target_did, state, reason, created_at, updated_at
		) VALUES ($1,$2,$3,'pending','verifiedInstagramFollow',$4,$4)
		ON CONFLICT (importer_did, target_did, reason) DO NOTHING
		RETURNING id, state
	`, params.ID, params.ImporterDID, params.TargetDID, params.Now).Scan(&ledgerID, &state)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id, state
			FROM instagram_follow_suggestions
			WHERE importer_did=$1
			  AND target_did=$2
			  AND reason='verifiedInstagramFollow'
			FOR UPDATE
		`, params.ImporterDID, params.TargetDID).Scan(&ledgerID, &state)
	}
	if err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE instagram_graph_handles SET matched=true WHERE id=$1
	`, handleID); err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_suggestion_sources (suggestion_id, import_id, created_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (suggestion_id, import_id) DO NOTHING
	`, ledgerID, params.ImportID, params.Now); err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}

	if state.SuppressesReconciliation() {
		operation, err := loadAutomaticFollowOperation(ctx, tx, ledgerID)
		if err != nil {
			return ReconcileAutomaticFollowResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReconcileAutomaticFollowResult{}, err
		}
		return ReconcileAutomaticFollowResult{
			Operation: operation, Suppressed: true,
		}, nil
	}
	if state == AutomaticFollowInvalidated {
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_follow_suggestions
			SET state='pending', terminal_at=NULL, updated_at=$2
			WHERE id=$1
		`, ledgerID, params.Now); err != nil {
			return ReconcileAutomaticFollowResult{}, err
		}
		state = AutomaticFollowPending
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO pds_follow_operations (
			id, suggestion_id, owner_did, target_did, rkey, status,
			attempt_count, next_attempt_at, created_at, updated_at
		) VALUES ($1,$1,$2,$3,$4,'pending',0,$5,$5,$5)
		ON CONFLICT (suggestion_id) DO UPDATE SET
			status=CASE
				WHEN pds_follow_operations.status='invalidated' THEN 'pending'
				ELSE pds_follow_operations.status
			END,
			next_attempt_at=CASE
				WHEN pds_follow_operations.status='invalidated' THEN EXCLUDED.next_attempt_at
				ELSE pds_follow_operations.next_attempt_at
			END,
			last_error_code=CASE
				WHEN pds_follow_operations.status='invalidated' THEN NULL
				ELSE pds_follow_operations.last_error_code
			END,
			updated_at=CASE
				WHEN pds_follow_operations.status='invalidated' THEN EXCLUDED.updated_at
				ELSE pds_follow_operations.updated_at
			END
	`, ledgerID, params.ImporterDID, params.TargetDID, params.Rkey, params.Now); err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	operation, err := loadAutomaticFollowOperation(ctx, tx, ledgerID)
	if err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReconcileAutomaticFollowResult{}, err
	}
	return ReconcileAutomaticFollowResult{
		Operation: operation,
		Created:   created,
		Queued:    created || state == AutomaticFollowPending,
	}, nil
}

func (s *AutomaticFollowStore) DeleteVerificationLedger(ctx context.Context, owner syntax.DID) error {
	if s == nil || s.pool == nil || owner == "" {
		return errors.New("invalid automatic follow ledger deletion")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		DELETE FROM pds_follow_operations WHERE owner_did=$1
	`, owner); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM instagram_follow_suggestions WHERE importer_did=$1
	`, owner); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func loadAutomaticFollowOperation(
	ctx context.Context,
	tx pgx.Tx,
	ledgerID uuid.UUID,
) (AutomaticFollowOperation, error) {
	var operation AutomaticFollowOperation
	err := tx.QueryRow(ctx, `
		SELECT id, owner_did, target_did, rkey, created_at
		FROM pds_follow_operations
		WHERE suggestion_id=$1
	`, ledgerID).Scan(
		&operation.ID,
		&operation.OwnerDID,
		&operation.TargetDID,
		&operation.Rkey,
		&operation.CreatedAt,
	)
	return operation, err
}

var _ AutomaticFollowOperationStore = (*AutomaticFollowStore)(nil)
