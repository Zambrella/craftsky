package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultTerminalPurgeComponentLimit = 16
	defaultTerminalPurgeRowBatchSize   = 100
	maximumTerminalPurgeBatchSize      = 1000
)

type TerminalPurgeProcessorConfig struct {
	Store          *Store
	WorkerID       string
	PollInterval   time.Duration
	ComponentLimit int
	RowBatchSize   int
	LeaseDuration  time.Duration
	RetryDelay     time.Duration
	Observer       TerminalPurgeObserver
}

// TerminalPurgeObserver receives bounded, identifier-free health signals from
// the purge loop. Implementations must not perform blocking work in the
// callback; the processor invokes it synchronously after each operation.
type TerminalPurgeObserver interface {
	ObserveTerminalPurge(TerminalPurgeObservation)
}

// TerminalPurgeObservation deliberately omits owner identifiers and errors.
// ErrorCategory is a low-cardinality operational classification suitable for
// metrics and logs without leaking account data.
type TerminalPurgeObservation struct {
	Operation     string
	Result        string
	ErrorCategory string
	Component     string
	DIDRole       string
	Claims        int
	RowsAffected  int64
	Remaining     int64
	Complete      bool
}

type TerminalPurgeProcessor struct {
	store          *Store
	workerID       string
	pollInterval   time.Duration
	componentLimit int
	rowBatchSize   int
	leaseDuration  time.Duration
	retryDelay     time.Duration
	inventory      map[string]TerminalDIDEntry
	observer       TerminalPurgeObserver
}

type PurgeBatchResult struct {
	RowsAffected int64
	Complete     bool
}

func NewTerminalPurgeProcessor(config TerminalPurgeProcessorConfig) (*TerminalPurgeProcessor, error) {
	if config.Store == nil {
		return nil, errors.New("terminal purge processor requires a lifecycle store")
	}
	config.WorkerID = strings.TrimSpace(config.WorkerID)
	if config.WorkerID == "" || len(config.WorkerID) > 256 {
		return nil, errors.New("terminal purge worker ID must contain 1 to 256 characters")
	}
	if config.PollInterval == 0 {
		config.PollInterval = time.Second
	}
	if config.ComponentLimit == 0 {
		config.ComponentLimit = defaultTerminalPurgeComponentLimit
	}
	if config.RowBatchSize == 0 {
		config.RowBatchSize = defaultTerminalPurgeRowBatchSize
	}
	if config.LeaseDuration == 0 {
		config.LeaseDuration = time.Minute
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
	if config.PollInterval <= 0 || config.ComponentLimit < 1 ||
		config.ComponentLimit > maximumTerminalPurgeBatchSize ||
		config.RowBatchSize < 1 || config.RowBatchSize > maximumTerminalPurgeBatchSize ||
		config.LeaseDuration <= config.PollInterval || config.RetryDelay <= 0 {
		return nil, errors.New("invalid terminal purge worker timing or batch geometry")
	}

	inventory := make(map[string]TerminalDIDEntry, len(terminalDIDInventory))
	for _, entry := range TerminalDIDInventory() {
		key := purgeInventoryKey(entry.Component, entry.Role)
		if _, exists := inventory[key]; exists {
			return nil, fmt.Errorf("duplicate terminal purge inventory key %s", key)
		}
		inventory[key] = entry
	}
	return &TerminalPurgeProcessor{
		store: config.Store, workerID: config.WorkerID,
		pollInterval: config.PollInterval, componentLimit: config.ComponentLimit,
		rowBatchSize: config.RowBatchSize, leaseDuration: config.LeaseDuration,
		retryDelay: config.RetryDelay, inventory: inventory, observer: config.Observer,
	}, nil
}

func (processor *TerminalPurgeProcessor) Run(ctx context.Context) error {
	if processor == nil {
		return errors.New("terminal purge processor is nil")
	}
	ticker := time.NewTicker(processor.pollInterval)
	defer ticker.Stop()
	for {
		if _, err := processor.ProcessBatch(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (processor *TerminalPurgeProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil || processor.store == nil {
		return 0, errors.New("terminal purge processor is unavailable")
	}
	claims, err := processor.store.ClaimPurgeComponents(ctx, PurgeClaimRequest{
		Worker: processor.workerID, LeaseToken: uuid.New(),
		LeaseDuration: processor.leaseDuration, Limit: processor.componentLimit,
	})
	if err != nil {
		processor.observe(TerminalPurgeObservation{
			Operation: "claim", Result: "failure", ErrorCategory: "dependency_unavailable",
		})
		return 0, err
	}
	processor.observe(TerminalPurgeObservation{
		Operation: "claim", Result: "success", Claims: len(claims),
	})
	var batchErr error
	touched := make(map[string]PurgeClaim)
	for _, claim := range claims {
		touched[claim.Owner.String()+"\x00"+fmt.Sprint(claim.OwnerGeneration)] = claim
		result, err := processor.ProcessClaim(ctx, claim)
		if err != nil {
			processor.observe(TerminalPurgeObservation{
				Operation: "component", Result: "failure", ErrorCategory: "component_failure",
				Component: claim.Component, DIDRole: claim.DIDRole,
			})
			next := processor.store.now().UTC().Add(processor.retryDelay)
			rescheduleErr := processor.store.ReschedulePurgeComponent(
				ctx, claim, claim.LeaseToken, next, "component_failure",
			)
			batchErr = errors.Join(batchErr, err, rescheduleErr)
		} else {
			processor.observe(TerminalPurgeObservation{
				Operation: "component", Result: "success", Component: claim.Component,
				DIDRole: claim.DIDRole, RowsAffected: result.RowsAffected,
				Complete: result.Complete,
			})
		}
	}
	for _, claim := range touched {
		_, err := processor.store.FinalizeTerminalPurge(ctx, claim.Owner, claim.OwnerGeneration)
		if err != nil && !errors.Is(err, ErrPurgeIncomplete) {
			batchErr = errors.Join(batchErr, err)
		}
	}
	processor.observeBacklog(ctx)
	return len(claims), batchErr
}

func (processor *TerminalPurgeProcessor) observe(observation TerminalPurgeObservation) {
	if processor != nil && processor.observer != nil {
		processor.observer.ObserveTerminalPurge(observation)
	}
}

func (processor *TerminalPurgeProcessor) observeBacklog(ctx context.Context) {
	if processor == nil || processor.observer == nil || processor.store == nil {
		return
	}
	var remaining int64
	if err := processor.store.pool.QueryRow(ctx, `
		SELECT count(*) FROM owner_purge_components WHERE state <> 'complete'
	`).Scan(&remaining); err != nil {
		processor.observe(TerminalPurgeObservation{
			Operation: "backlog", Result: "failure", ErrorCategory: "dependency_unavailable",
		})
		return
	}
	processor.observe(TerminalPurgeObservation{
		Operation: "backlog", Result: "success", Remaining: remaining,
	})
}

// ProcessClaim applies at most one configured row batch and atomically moves
// the leased component back to pending or complete. Deleting the rows and
// advancing the ledger cannot split across a crash.
func (processor *TerminalPurgeProcessor) ProcessClaim(
	ctx context.Context,
	claim PurgeClaim,
) (PurgeBatchResult, error) {
	if processor == nil || processor.store == nil || claim.Owner == "" ||
		claim.OwnerGeneration <= 0 || claim.LeaseToken == uuid.Nil {
		return PurgeBatchResult{}, ErrPurgeLeaseLost
	}
	entry, ok := processor.inventory[purgeInventoryKey(claim.Component, claim.DIDRole)]
	if !ok {
		return PurgeBatchResult{}, fmt.Errorf("unknown terminal purge component %s/%s", claim.Component, claim.DIDRole)
	}
	moderationScope, err := processor.selectModerationPurgeScope(ctx, entry, claim.Owner)
	if err != nil {
		return PurgeBatchResult{}, err
	}
	fencedOwners := []syntax.DID{claim.Owner}
	if len(moderationScope.owners) > 0 {
		fencedOwners = moderationScope.owners
	}
	var result PurgeBatchResult
	err = processor.store.WithExclusiveOwnerStates(
		ctx,
		fencedOwners,
		func(fenceCtx context.Context, tx pgx.Tx, _ map[syntax.DID]Lifecycle) error {
			now := processor.store.now().UTC().Truncate(time.Microsecond)
			if err := lockPurgeClaimTx(fenceCtx, tx, claim, now); err != nil {
				return err
			}
			var err error
			switch entry.Action {
			case TerminalRetainTombstone:
				result.Complete = true
			case TerminalRetainCleanup:
				var exists bool
				exists, err = terminalRoleExistsTx(fenceCtx, tx, entry, claim.Owner)
				result.Complete = !exists
			case TerminalAnonymizeRow:
				result.RowsAffected, err = anonymizeTerminalRoleBatchTx(
					fenceCtx, tx, entry, claim.Owner, processor.rowBatchSize,
				)
				if err == nil {
					var exists bool
					exists, err = terminalRoleExistsTx(fenceCtx, tx, entry, claim.Owner)
					result.Complete = !exists
				}
			case TerminalDeleteRow:
				var blocked bool
				var cascadeTargets []pgtype.TID
				cascadeTargets, err = lockTerminalCascadeParentsTx(
					fenceCtx, tx, entry, claim.Owner, processor.rowBatchSize,
				)
				if err == nil {
					blocked, err = terminalPurgeDependencyBlockedTx(fenceCtx, tx, entry, claim.Owner)
				}
				if err == nil && !blocked {
					result.RowsAffected, err = drainTerminalCascadeBatchTx(
						fenceCtx, tx, entry, claim.Owner, cascadeTargets, processor.rowBatchSize,
					)
				}
				if err == nil && !blocked && result.RowsAffected == 0 {
					if entry.Table == "scheduled_post_media" && entry.Column == "owner_did" {
						result.RowsAffected, err = purgeScheduledMediaBatchTx(
							fenceCtx, tx, claim.Owner, processor.rowBatchSize, now,
						)
					} else if entry.Table == "push_account_subscriptions" && entry.Column == "account_did" {
						result.RowsAffected, err = purgePushSubscriptionsBatchTx(
							fenceCtx, tx, claim.Owner, cascadeTargets, processor.rowBatchSize,
						)
					} else if entry.Table == "moderation_outputs" {
						result.RowsAffected, err = purgeModerationOutputsBatchTx(
							fenceCtx, tx, entry, claim.Owner, moderationScope.outputIDs,
							processor.rowBatchSize, now,
						)
					} else if entry.Table == "moderation_restoration_outbox" {
						result.RowsAffected, err = purgeModerationRestorationOutboxBatchTx(
							fenceCtx, tx, claim.Owner, moderationScope.outputIDs,
							processor.rowBatchSize, now,
						)
					} else if _, classified := terminalCascadePolicies[entry.Table]; classified {
						result.RowsAffected, err = deleteLockedTerminalRoleBatchTx(
							fenceCtx, tx, entry, cascadeTargets,
						)
					} else {
						result.RowsAffected, err = deleteTerminalRoleBatchTx(
							fenceCtx, tx, entry, claim.Owner, processor.rowBatchSize,
						)
					}
				}
				if err == nil && !blocked {
					var exists bool
					exists, err = terminalRoleExistsTx(fenceCtx, tx, entry, claim.Owner)
					result.Complete = !exists
				}
			default:
				err = errors.New("invalid terminal purge action")
			}
			if err != nil {
				return err
			}
			return finishPurgeClaimTx(
				fenceCtx, tx, claim, now, result.Complete, processor.retryDelay,
			)
		},
	)
	return result, err
}

type moderationPurgeScope struct {
	outputIDs []string
	owners    []syntax.DID
}

// selectModerationPurgeScope fixes one bounded parent/outbox batch before any
// lifecycle fence is acquired. ProcessClaim then takes every participant fence
// in canonical DID order and restricts deletion to these IDs, so a shifted
// keyset cannot introduce an unfenced source or target into the transaction.
func (processor *TerminalPurgeProcessor) selectModerationPurgeScope(
	ctx context.Context,
	entry TerminalDIDEntry,
	owner syntax.DID,
) (moderationPurgeScope, error) {
	if entry.Table != "moderation_outputs" && entry.Table != "moderation_restoration_outbox" {
		return moderationPurgeScope{}, nil
	}
	query := `
		SELECT output.id,output.source_did,output.subject_did
		FROM moderation_outputs AS output
		WHERE output.` + quoteIdentifier(entry.Column) + `=$1
		ORDER BY output.id
		LIMIT $2
	`
	if entry.Table == "moderation_restoration_outbox" {
		query = `
			SELECT output.id,output.source_did,output.subject_did
			FROM moderation_restoration_outbox AS outbox
			JOIN moderation_outputs AS output ON output.id=outbox.moderation_output_id
			WHERE outbox.target_did=$1
			ORDER BY output.id
			LIMIT $2
		`
	}
	rows, err := processor.store.pool.Query(ctx, query, owner, processor.rowBatchSize)
	if err != nil {
		return moderationPurgeScope{}, fmt.Errorf("select terminal moderation fence scope: %w", err)
	}
	defer rows.Close()
	scope := moderationPurgeScope{}
	participants := []syntax.DID{owner}
	for rows.Next() {
		var (
			outputID string
			source   syntax.DID
			subject  syntax.DID
		)
		if err := rows.Scan(&outputID, &source, &subject); err != nil {
			return moderationPurgeScope{}, fmt.Errorf("scan terminal moderation fence scope: %w", err)
		}
		scope.outputIDs = append(scope.outputIDs, outputID)
		participants = append(participants, source, subject)
	}
	if err := rows.Err(); err != nil {
		return moderationPurgeScope{}, fmt.Errorf("iterate terminal moderation fence scope: %w", err)
	}
	scope.owners, err = CanonicalOwners(participants)
	if err != nil {
		return moderationPurgeScope{}, err
	}
	return scope, nil
}

func lockPurgeClaimTx(ctx context.Context, tx pgx.Tx, claim PurgeClaim, now time.Time) error {
	var locked bool
	err := tx.QueryRow(ctx, `
		SELECT true FROM owner_purge_components
		WHERE owner_did=$1 AND owner_generation=$2
		  AND component=$3 AND did_role=$4
		  AND state='running' AND lease_token=$5 AND lease_expires_at>$6
		FOR UPDATE
	`, claim.Owner, claim.OwnerGeneration, claim.Component, claim.DIDRole,
		claim.LeaseToken, now).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPurgeLeaseLost
	}
	if err != nil {
		return fmt.Errorf("lock terminal purge component: %w", err)
	}
	if !locked {
		return ErrPurgeLeaseLost
	}
	return nil
}

func finishPurgeClaimTx(
	ctx context.Context,
	tx pgx.Tx,
	claim PurgeClaim,
	now time.Time,
	complete bool,
	retryDelay time.Duration,
) error {
	state := PurgePending
	nextAttempt := any(now.Add(retryDelay))
	completedAt := any(nil)
	if complete {
		state = PurgeComplete
		nextAttempt = now
		completedAt = now
	}
	result, err := tx.Exec(ctx, `
		UPDATE owner_purge_components
		SET state=$6,next_attempt_at=$7,lease_owner=NULL,lease_token=NULL,
		    lease_expires_at=NULL,completed_at=$8,last_error_category=NULL,updated_at=$9
		WHERE owner_did=$1 AND owner_generation=$2 AND component=$3 AND did_role=$4
		  AND state='running' AND lease_token=$5 AND lease_expires_at>$9
	`, claim.Owner, claim.OwnerGeneration, claim.Component, claim.DIDRole,
		claim.LeaseToken, state, nextAttempt, completedAt, now)
	if err != nil {
		return fmt.Errorf("advance terminal purge component: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPurgeLeaseLost
	}
	return nil
}

func terminalRoleExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM " + quoteIdentifier(entry.Table) +
		" WHERE " + quoteIdentifier(entry.Column) + "=$1)"
	var exists bool
	if err := tx.QueryRow(ctx, query, owner).Scan(&exists); err != nil {
		return false, fmt.Errorf("inspect terminal purge role %s/%s: %w", entry.Component, entry.Role, err)
	}
	return exists, nil
}

func deleteTerminalRoleBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
	limit int,
) (int64, error) {
	query := terminalMutationBatchSQL(entry, "DELETE")
	result, err := tx.Exec(ctx, query, owner, limit)
	if err != nil {
		return 0, fmt.Errorf("delete terminal purge role %s/%s: %w", entry.Component, entry.Role, err)
	}
	return result.RowsAffected(), nil
}

func anonymizeTerminalRoleBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
	limit int,
) (int64, error) {
	query := terminalMutationBatchSQL(entry, "ANONYMIZE")
	result, err := tx.Exec(ctx, query, owner, limit)
	if err != nil {
		return 0, fmt.Errorf("anonymize terminal purge role %s/%s: %w", entry.Component, entry.Role, err)
	}
	return result.RowsAffected(), nil
}

func terminalMutationBatchSQL(entry TerminalDIDEntry, operation string) string {
	table := quoteIdentifier(entry.Table)
	keys := make([]string, 0, len(entry.KeyColumns))
	qualifiedMatches := make([]string, 0, len(entry.KeyColumns))
	for _, key := range entry.KeyColumns {
		quoted := quoteIdentifier(key)
		keys = append(keys, quoted)
		qualifiedMatches = append(qualifiedMatches, "row."+quoted+" IS NOT DISTINCT FROM target."+quoted)
	}
	selection := `WITH target AS (
		SELECT ` + strings.Join(keys, ",") + ` FROM ` + table + `
		WHERE ` + quoteIdentifier(entry.Column) + `=$1
		ORDER BY ` + strings.Join(keys, ",") + ` LIMIT $2
		FOR UPDATE SKIP LOCKED
	) `
	if operation == "ANONYMIZE" {
		set := quoteIdentifier(entry.Column) + "=NULL"
		if entry.Table == "instagram_audit_events" {
			set += ",subject_id=NULL"
		}
		return selection + "UPDATE " + table + " AS row SET " + set +
			" FROM target WHERE " + strings.Join(qualifiedMatches, " AND ")
	}
	return selection + "DELETE FROM " + table + " AS row USING target WHERE " +
		strings.Join(qualifiedMatches, " AND ")
}

func terminalPurgeDependencyBlockedTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
) (bool, error) {
	var query string
	switch entry.Table {
	case "account_deletion_operations":
		query = `SELECT EXISTS(SELECT 1 FROM account_deletion_safety_tombstones WHERE owner_did=$1)`
	case "craftsky_profiles":
		// These relations all reference craftsky_profiles with cascading
		// actions. Their own fixed ledger roles delete them in bounded batches;
		// the profile parent cannot run first and amplify one selected row.
		query = `SELECT EXISTS(
			SELECT 1 FROM actor_mutes WHERE owner_did=$1
			UNION ALL SELECT 1 FROM saved_post_folders WHERE owner_did=$1
			UNION ALL SELECT 1 FROM saved_posts WHERE owner_did=$1
			UNION ALL SELECT 1 FROM profile_customisations WHERE owner_did=$1
			UNION ALL SELECT 1 FROM profile_pins WHERE owner_did=$1
		)`
	case "saved_post_folders":
		// Folder deletion uses ON DELETE SET NULL. Drain the owner's saved
		// rows first so one folder cannot rewrite an unbounded number of rows.
		query = `SELECT EXISTS(SELECT 1 FROM saved_posts WHERE owner_did=$1 AND folder_id IS NOT NULL)`
	case "oauth_sessions":
		// Parent-session deletion cascades to children and handoff exchanges,
		// and accepted deletion operations also retain restrictive references.
		query = `SELECT EXISTS(
			SELECT 1 FROM craftsky_sessions WHERE account_did=$1
			UNION ALL SELECT 1 FROM oauth_handoff_exchanges WHERE owner_did=$1
			UNION ALL SELECT 1 FROM account_deletion_operations WHERE owner_did=$1
		)`
	case "scheduled_posts":
		query = `SELECT EXISTS(SELECT 1 FROM scheduled_post_media WHERE owner_did=$1)`
	case "scheduled_post_object_attempts":
		query = `SELECT EXISTS(
			SELECT 1 FROM scheduled_post_media WHERE owner_did=$1
			UNION ALL
			SELECT 1 FROM scheduled_post_cleanup_jobs WHERE owner_did=$1
		)`
	default:
		return false, nil
	}
	var blocked bool
	if err := tx.QueryRow(ctx, query, owner).Scan(&blocked); err != nil {
		return false, fmt.Errorf("inspect terminal purge dependency %s: %w", entry.Table, err)
	}
	return blocked, nil
}

func purgeScheduledMediaBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	limit int,
	now time.Time,
) (int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT id,object_key
		FROM scheduled_post_media
		WHERE owner_did=$1
		ORDER BY id
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, owner, limit)
	if err != nil {
		return 0, fmt.Errorf("select terminal scheduled media: %w", err)
	}
	type mediaTarget struct {
		id        uuid.UUID
		objectKey string
	}
	targets := make([]mediaTarget, 0, limit)
	for rows.Next() {
		var target mediaTarget
		if err := rows.Scan(&target.id, &target.objectKey); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan terminal scheduled media: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate terminal scheduled media: %w", err)
	}
	rows.Close()
	for _, target := range targets {
		if _, err := tx.Exec(ctx, `
			INSERT INTO scheduled_post_cleanup_jobs(
				id,object_key,owner_did,owner_generation,upload_generation,
				source_attempt_id,outcome_uncertain,settlement_not_before,
				next_attempt_at,created_at,updated_at
			)
			SELECT $1,attempt.object_key,attempt.owner_did,attempt.owner_generation,
			       attempt.upload_generation,attempt.upload_attempt_id,
			       attempt.remote_outcome='dispatched',attempt.settlement_not_before,
			       $3,$3,$3
			FROM scheduled_post_object_attempts AS attempt
			WHERE attempt.object_key=$2
			ON CONFLICT(object_key) DO NOTHING
		`, uuid.New(), target.objectKey, now); err != nil {
			return 0, fmt.Errorf("enqueue terminal scheduled object cleanup: %w", err)
		}
	}
	ids := make([]uuid.UUID, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.id)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := tx.Exec(ctx, `DELETE FROM scheduled_post_media WHERE id=ANY($1::uuid[])`, ids)
	if err != nil {
		return 0, fmt.Errorf("delete terminal scheduled media: %w", err)
	}
	return result.RowsAffected(), nil
}

func purgePushSubscriptionsBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	targets []pgtype.TID,
	limit int,
) (int64, error) {
	if len(targets) == 0 {
		return 0, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id,installation_id
		FROM push_account_subscriptions
		WHERE account_did=$1 AND ctid=ANY($3::tid[])
		ORDER BY id
		LIMIT $2
	`, owner, limit, targets)
	if err != nil {
		return 0, fmt.Errorf("select terminal push subscriptions: %w", err)
	}
	var ids, installationIDs []uuid.UUID
	for rows.Next() {
		var id, installationID uuid.UUID
		if err := rows.Scan(&id, &installationID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan terminal push subscriptions: %w", err)
		}
		ids = append(ids, id)
		installationIDs = append(installationIDs, installationID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate terminal push subscriptions: %w", err)
	}
	rows.Close()
	if len(ids) == 0 {
		return 0, nil
	}
	installationRows, err := tx.Query(ctx, `
		SELECT id FROM push_installations
		WHERE id=ANY($1::uuid[])
		ORDER BY id
		FOR UPDATE NOWAIT
	`, installationIDs)
	if err != nil {
		return 0, fmt.Errorf("lock terminal push installations: %w", err)
	}
	for installationRows.Next() {
		var id uuid.UUID
		if err := installationRows.Scan(&id); err != nil {
			installationRows.Close()
			return 0, fmt.Errorf("scan terminal push installation: %w", err)
		}
	}
	if err := installationRows.Err(); err != nil {
		installationRows.Close()
		return 0, fmt.Errorf("iterate terminal push installations: %w", err)
	}
	installationRows.Close()
	result, err := tx.Exec(ctx, `DELETE FROM push_account_subscriptions WHERE id=ANY($1::uuid[])`, ids)
	if err != nil {
		return 0, fmt.Errorf("delete terminal push subscriptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM push_installations installation
		WHERE installation.id=ANY($1::uuid[])
		  AND NOT EXISTS(
			SELECT 1 FROM push_account_subscriptions subscription
			WHERE subscription.installation_id=installation.id
		  )
	`, installationIDs); err != nil {
		return 0, fmt.Errorf("delete unowned terminal push installations: %w", err)
	}
	return result.RowsAffected(), nil
}

// purgeModerationOutputsBatchTx settles the ON DELETE RESTRICT restoration
// child before removing its parent moderation row. A claimed reconciliation
// job is allowed to finish under its own lease; queued/retryable work is
// cancelled before it can create new private state. The fixed component stays
// pending while any processing child exists, so a crash or live worker cannot
// turn FK pressure into data loss or an unbounded transaction.
func purgeModerationOutputsBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
	outputIDs []string,
	limit int,
	now time.Time,
) (int64, error) {
	if len(outputIDs) == 0 {
		return 0, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id,subject_did
		FROM moderation_outputs
		WHERE `+quoteIdentifier(entry.Column)+`=$1 AND id=ANY($3::text[])
		ORDER BY id
		LIMIT $2
		FOR UPDATE
	`, owner, limit, outputIDs)
	if err != nil {
		return 0, fmt.Errorf("select terminal moderation outputs %s: %w", entry.Role, err)
	}
	type moderationTarget struct {
		id      string
		subject syntax.DID
	}
	targets := make([]moderationTarget, 0, limit)
	for rows.Next() {
		var target moderationTarget
		if err := rows.Scan(&target.id, &target.subject); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan terminal moderation output: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate terminal moderation outputs: %w", err)
	}
	rows.Close()

	var deleted int64
	for _, target := range targets {
		blocked, err := settleModerationRestorationOutboxTx(
			ctx, tx, target.id, target.subject == owner, now,
		)
		if err != nil {
			return deleted, err
		}
		if blocked {
			continue
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM moderation_idempotency_receipts
			WHERE output_id=$1
		`, target.id); err != nil {
			return deleted, fmt.Errorf("delete terminal moderation idempotency receipt: %w", err)
		}
		result, err := tx.Exec(ctx, `
			DELETE FROM moderation_outputs
			WHERE id=$1 AND `+quoteIdentifier(entry.Column)+`=$2
		`, target.id, owner)
		if err != nil {
			return deleted, fmt.Errorf("delete terminal moderation output %s: %w", entry.Role, err)
		}
		deleted += result.RowsAffected()
	}
	return deleted, nil
}

// PurgeModerationOutputsForOwnerTx applies the same role-aware child-first
// settlement used by terminal purge to an accepted explicit deletion. A DID
// that is the target/subject wins over its source role, so its restoration is
// cancelled. Source-only deletion preserves or promotes target-owned work.
// A live processing reconciliation claim is reported as blocked and left
// untouched so the caller can roll back and retry the complete cleanup.
func PurgeModerationOutputsForOwnerTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	limit int,
	now time.Time,
) (deleted int64, blocked bool, err error) {
	if tx == nil || owner == "" || limit < 1 || limit > maximumTerminalPurgeBatchSize {
		return 0, false, errors.New("invalid moderation owner purge scope")
	}
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM moderation_outputs
		WHERE source_did=$1 OR subject_did=$1
		ORDER BY id
		LIMIT $2
	`, owner, limit)
	if err != nil {
		return 0, false, fmt.Errorf("select owner moderation outputs: %w", err)
	}
	var outputIDs []string
	for rows.Next() {
		var outputID string
		if err := rows.Scan(&outputID); err != nil {
			rows.Close()
			return 0, false, fmt.Errorf("scan owner moderation output ID: %w", err)
		}
		outputIDs = append(outputIDs, outputID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, false, fmt.Errorf("iterate owner moderation output IDs: %w", err)
	}
	rows.Close()
	return PurgeModerationOutputIDsForOwnerTx(ctx, tx, owner, outputIDs, now)
}

// PurgeModerationOutputIDsForOwnerTx settles only the batch whose participant
// owner fences the caller already holds. Restricting the keyset prevents a
// concurrent deletion from shifting an unfenced participant into this batch.
func PurgeModerationOutputIDsForOwnerTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	outputIDs []string,
	now time.Time,
) (deleted int64, blocked bool, err error) {
	if tx == nil || owner == "" || len(outputIDs) > maximumTerminalPurgeBatchSize {
		return 0, false, errors.New("invalid moderation owner purge batch")
	}
	if len(outputIDs) == 0 {
		return 0, false, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id,subject_did
		FROM moderation_outputs
		WHERE (source_did=$1 OR subject_did=$1)
		  AND id=ANY($2::text[])
		ORDER BY id
		FOR UPDATE
	`, owner, outputIDs)
	if err != nil {
		return 0, false, fmt.Errorf("lock owner moderation output batch: %w", err)
	}
	type target struct {
		id      string
		subject syntax.DID
	}
	var targets []target
	for rows.Next() {
		var selected target
		if err := rows.Scan(&selected.id, &selected.subject); err != nil {
			rows.Close()
			return 0, false, fmt.Errorf("scan owner moderation output: %w", err)
		}
		targets = append(targets, selected)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, false, fmt.Errorf("iterate owner moderation outputs: %w", err)
	}
	rows.Close()

	for _, selected := range targets {
		processing, err := settleModerationRestorationOutboxTx(
			ctx,
			tx,
			selected.id,
			selected.subject == owner,
			now,
		)
		if err != nil {
			return deleted, blocked, err
		}
		if processing {
			blocked = true
			continue
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM moderation_idempotency_receipts
			WHERE output_id=$1
		`, selected.id); err != nil {
			return deleted, blocked, fmt.Errorf("delete owner moderation idempotency receipt: %w", err)
		}
		result, err := tx.Exec(ctx, `
			DELETE FROM moderation_outputs
			WHERE id=$1 AND (source_did=$2 OR subject_did=$2)
		`, selected.id, owner)
		if err != nil {
			return deleted, blocked, fmt.Errorf("delete owner moderation output: %w", err)
		}
		deleted += result.RowsAffected()
	}
	return deleted, blocked, nil
}

func purgeModerationRestorationOutboxBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	outputIDs []string,
	limit int,
	now time.Time,
) (int64, error) {
	if len(outputIDs) == 0 {
		return 0, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT moderation_output_id
		FROM moderation_restoration_outbox
		WHERE target_did=$1 AND moderation_output_id=ANY($3::text[])
		ORDER BY moderation_output_id
		LIMIT $2
		FOR UPDATE
	`, owner, limit, outputIDs)
	if err != nil {
		return 0, fmt.Errorf("select terminal moderation restoration outbox: %w", err)
	}
	var selectedIDs []string
	for rows.Next() {
		var outputID string
		if err := rows.Scan(&outputID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan terminal moderation restoration outbox: %w", err)
		}
		selectedIDs = append(selectedIDs, outputID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate terminal moderation restoration outbox: %w", err)
	}
	rows.Close()

	var deleted int64
	for _, outputID := range selectedIDs {
		blocked, err := settleModerationRestorationOutboxTx(ctx, tx, outputID, true, now)
		if err != nil {
			return deleted, err
		}
		if !blocked {
			deleted++
		}
	}
	return deleted, nil
}

func settleModerationRestorationOutboxTx(
	ctx context.Context,
	tx pgx.Tx,
	outputID string,
	targetTerminal bool,
	now time.Time,
) (bool, error) {
	var (
		status          string
		jobID           *uuid.UUID
		processedAt     *time.Time
		targetState     string
		activeLinkID    *uuid.UUID
		activeLinkOwner *syntax.DID
	)
	err := tx.QueryRow(ctx, `
		SELECT outbox.status,outbox.reconciliation_job_id,outbox.processed_at,
		       COALESCE(lifecycle.state,''),link.id,link.owner_did
		FROM moderation_restoration_outbox AS outbox
		LEFT JOIN owner_lifecycles AS lifecycle
		  ON lifecycle.owner_did=outbox.target_did
		LEFT JOIN LATERAL (
			SELECT id,owner_did
			FROM instagram_account_links
			WHERE owner_did=outbox.target_did
			  AND state='active' AND discoverable AND NOT conflict_pending
			ORDER BY updated_at DESC,id DESC
			LIMIT 1
		) AS link ON true
		WHERE outbox.moderation_output_id=$1
		FOR UPDATE OF outbox
	`, outputID).Scan(
		&status,
		&jobID,
		&processedAt,
		&targetState,
		&activeLinkID,
		&activeLinkOwner,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock terminal moderation restoration outbox: %w", err)
	}
	targetTerminal = targetTerminal || targetState == string(StateTerminal)

	if jobID != nil {
		var jobStatus string
		err := tx.QueryRow(ctx, `
			SELECT status
			FROM instagram_reconciliation_jobs
			WHERE id=$1
			FOR UPDATE
		`, *jobID).Scan(&jobStatus)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("lock terminal moderation reconciliation job: %w", err)
		}
		if err == nil {
			if jobStatus == "processing" {
				return true, nil
			}
			switch {
			case targetTerminal && (jobStatus == "queued" || jobStatus == "retryable"):
				result, err := tx.Exec(ctx, `
					UPDATE instagram_reconciliation_jobs
					SET status='ignored',terminal_at=COALESCE(terminal_at,$2),
					    lease_token=NULL,lease_expires_at=NULL,updated_at=$2
					WHERE id=$1 AND status IN ('queued','retryable')
				`, *jobID, now)
				if err != nil {
					return false, fmt.Errorf("cancel terminal moderation reconciliation job: %w", err)
				}
				if result.RowsAffected() != 1 {
					return false, errors.New("terminal moderation reconciliation cancellation lost ownership")
				}
			case jobStatus == "queued" || jobStatus == "retryable" ||
				jobStatus == "completed" || jobStatus == "ignored" || jobStatus == "failed":
			default:
				return false, fmt.Errorf("unknown moderation reconciliation job state %q", jobStatus)
			}
		}
	}

	outcome := status
	archivedProcessedAt := processedAt
	switch {
	case targetTerminal:
		outcome = "cancelled_target_terminal"
		if archivedProcessedAt == nil {
			archivedProcessedAt = &now
		}
	case status == "pending":
		outcome = "no_work"
		archivedProcessedAt = &now
		if targetState == string(StateActive) && activeLinkID != nil &&
			activeLinkOwner != nil && *activeLinkOwner != "" {
			jobID := uuid.New()
			if _, err := tx.Exec(ctx, `
				INSERT INTO instagram_reconciliation_jobs(
					id,owner_did,link_id,reason,status,next_attempt_at,
					created_at,updated_at
				) VALUES($1,$2,$3,$4,'queued',$5,$5,$5)
			`, jobID, *activeLinkOwner, *activeLinkID, "moderationCleared:"+outputID, now); err != nil {
				return false, fmt.Errorf("promote terminal-source moderation restoration: %w", err)
			}
			outcome = "queued"
		}
	case status == "queued":
		if archivedProcessedAt == nil {
			archivedProcessedAt = &now
		}
	case status == "no_work" || status == "cancelled_target_terminal":
		if archivedProcessedAt == nil {
			archivedProcessedAt = &now
		}
	default:
		return false, fmt.Errorf("unknown moderation restoration outbox state %q", status)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_restoration_history(
			moderation_output_id,outcome,processed_at,archived_at
		) VALUES($1,$2,$3,$4)
		ON CONFLICT(moderation_output_id) DO NOTHING
	`, outputID, outcome, *archivedProcessedAt, now); err != nil {
		return false, fmt.Errorf("archive terminal moderation restoration outbox: %w", err)
	}
	result, err := tx.Exec(ctx, `
		DELETE FROM moderation_restoration_outbox
		WHERE moderation_output_id=$1
	`, outputID)
	if err != nil {
		return false, fmt.Errorf("delete terminal moderation restoration outbox: %w", err)
	}
	if result.RowsAffected() != 1 {
		return false, errors.New("terminal moderation restoration delete lost ownership")
	}
	return false, nil
}

func purgeInventoryKey(component, role string) string {
	return component + "\x00" + role
}

func quoteIdentifier(value string) string {
	return pgx.Identifier{value}.Sanitize()
}
