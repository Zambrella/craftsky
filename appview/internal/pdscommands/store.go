package pdscommands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing command")

var ErrIllegalTransition = errors.New("illegal command state transition")

type StoreConfig struct {
	Pool       *pgxpool.Pool
	Lifecycles *ownerlifecycle.Store
	Now        func() time.Time
	NewUUID    func() uuid.UUID
}

type Store struct {
	pool       *pgxpool.Pool
	lifecycles *ownerlifecycle.Store
	now        func() time.Time
	newUUID    func() uuid.UUID
}

type PrepareRequest struct {
	Owner                   syntax.DID
	OwnerGeneration         int64
	OperationKind           string
	OperationKey            uuid.UUID
	FingerprintVersion      int
	Fingerprint             [32]byte
	ImmutableRequest        json.RawMessage
	SelectedURI             syntax.ATURI
	SelectedRkey            syntax.RecordKey
	ExpectedOwner           syntax.DID
	ExpectedOwnerGeneration int64
	ExpectedTarget          syntax.DID
}

type PreparedCommand struct {
	ID                      uuid.UUID
	ScopedKey               ScopedCommandKey
	OwnerGeneration         int64
	Fingerprint             [32]byte
	State                   CommandState
	Replay                  bool
	SelectedURI             syntax.ATURI
	SelectedRkey            syntax.RecordKey
	ExpectedOwner           syntax.DID
	ExpectedOwnerGeneration int64
	ExpectedTarget          syntax.DID
	CreatedAt               time.Time
	ActivePlanVersion       int
}

type ExactPlan struct {
	Version            int
	FingerprintVersion int
	Fingerprint        [32]byte
	RepositoryCID      syntax.CID
	RepositoryRevision syntax.TID
	RemoteDeadline     time.Time
	Steps              []DispatchStep
}

type DispatchAttempt struct {
	ID             uuid.UUID
	CommandID      uuid.UUID
	AttemptOrdinal int
	PlanVersion    int
}

type DispatchCompletion struct {
	Outcome           CommandState
	RetryAfterSeconds int
	ErrorClass        string
	ResultCommitCID   syntax.CID
	ResultRecords     json.RawMessage
}

type TerminalResult struct {
	State           CommandState
	HTTPStatus      int
	ResponseBody    json.RawMessage
	ResponseHeaders json.RawMessage
}

type CommandResult struct {
	TerminalResult
	RetryAfterSeconds int
}

// ReplayCommand carries the immutable inputs needed to resume an existing
// command without consulting a potentially changed serving projection.
type ReplayCommand struct {
	OwnerGeneration int64
	Intent          json.RawMessage
	SelectedRkey    syntax.RecordKey
	Result          CommandResult
}

func (store *Store) LookupCommand(ctx context.Context, owner syntax.DID, kind string, key uuid.UUID) (*ReplayCommand, error) {
	var replay ReplayCommand
	var commandID uuid.UUID
	var rkey *string
	err := store.pool.QueryRow(ctx, `
		SELECT id,owner_generation,immutable_request,selected_rkey
		FROM pds_commands WHERE owner_did=$1 AND operation_kind=$2 AND operation_key=$3
	`, owner, kind, key).Scan(&commandID, &replay.OwnerGeneration, &replay.Intent, &rkey)
	if errors.Is(err, pgx.ErrNoRows) {
		hash := ScopedKeyHash(owner, kind, key)
		var tombstoned bool
		if err := store.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM pds_command_tombstones
			WHERE owner_did=$1 AND operation_kind=$2 AND scoped_key_hash=$3)
		`, owner, kind, hash[:]).Scan(&tombstoned); err != nil {
			return nil, fmt.Errorf("check PDS command tombstone: %w", err)
		}
		if tombstoned {
			return nil, ErrIdempotencyConflict
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("look up PDS command: %w", err)
	}
	if rkey != nil {
		replay.SelectedRkey = syntax.RecordKey(*rkey)
	}
	replay.Result, err = store.Result(ctx, commandID)
	if err != nil {
		return nil, err
	}
	return &replay, nil
}

func NewStore(config StoreConfig) (*Store, error) {
	if config.Pool == nil {
		return nil, errors.New("PDS command store requires a database pool")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.NewUUID == nil {
		config.NewUUID = uuid.New
	}
	return &Store{
		pool: config.Pool, lifecycles: config.Lifecycles,
		now: config.Now, newUUID: config.NewUUID,
	}, nil
}

func (store *Store) PrepareOrReplay(ctx context.Context, request PrepareRequest) (PreparedCommand, error) {
	return store.prepareOrReplay(ctx, request, nil)
}

// PrepareOrReplayWithSelection invokes selectIdentity only when no command exists
// for the scoped operation key. The scoped advisory lock makes append identity
// allocation part of first preparation rather than every replay attempt.
func (store *Store) PrepareOrReplayWithSelection(
	ctx context.Context,
	request PrepareRequest,
	selectIdentity func() (syntax.ATURI, syntax.RecordKey, error),
) (PreparedCommand, error) {
	if selectIdentity == nil {
		return PreparedCommand{}, errors.New("command identity selector is unavailable")
	}
	return store.prepareOrReplay(ctx, request, selectIdentity)
}

func (store *Store) prepareOrReplay(
	ctx context.Context,
	request PrepareRequest,
	selectIdentity func() (syntax.ATURI, syntax.RecordKey, error),
) (PreparedCommand, error) {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || request.FingerprintVersion < 1 || !json.Valid(request.ImmutableRequest) {
		return PreparedCommand{}, errors.New("invalid command preparation request")
	}
	if (request.ExpectedOwner == "") != (request.ExpectedOwnerGeneration == 0) || request.ExpectedOwnerGeneration < 0 {
		return PreparedCommand{}, errors.New("invalid command lifecycle expectation")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	commandID := store.newUUID()
	var prepared PreparedCommand
	prepare := func(operationCtx context.Context) error {
		return store.withTransaction(operationCtx, func(tx pgx.Tx) error {
			if err := lockScopedCommandKey(operationCtx, tx, request.Owner, request.OperationKind, request.OperationKey); err != nil {
				return err
			}
			scopedHash := ScopedKeyHash(request.Owner, request.OperationKind, request.OperationKey)
			var tombstoned bool
			if err := tx.QueryRow(operationCtx, `
			SELECT EXISTS(
				SELECT 1 FROM pds_command_tombstones
				WHERE owner_did=$1 AND operation_kind=$2 AND scoped_key_hash=$3
			)
		`, request.Owner, request.OperationKind, scopedHash[:]).Scan(&tombstoned); err != nil {
				return fmt.Errorf("check command tombstone: %w", err)
			}
			if tombstoned {
				return ErrIdempotencyConflict
			}
			if selectIdentity != nil {
				var exists bool
				if err := tx.QueryRow(operationCtx, `
				SELECT EXISTS(
					SELECT 1 FROM pds_commands
					WHERE owner_did=$1 AND operation_kind=$2 AND operation_key=$3
				)
			`, request.Owner, request.OperationKind, request.OperationKey).Scan(&exists); err != nil {
					return fmt.Errorf("check existing PDS command: %w", err)
				}
				if !exists {
					selectedURI, selectedRkey, err := selectIdentity()
					if err != nil {
						return err
					}
					request.SelectedURI = selectedURI
					request.SelectedRkey = selectedRkey
				}
			}
			result, err := tx.Exec(operationCtx, `
			INSERT INTO pds_commands(
				id,owner_did,owner_generation,operation_kind,operation_key,
				request_fingerprint_version,request_fingerprint,immutable_request,
				selected_uri,selected_rkey,expected_owner_did,expected_owner_generation,
				expected_target_did,created_at,updated_at
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)
			ON CONFLICT(owner_did,operation_kind,operation_key) DO NOTHING
		`, commandID, request.Owner, request.OwnerGeneration, request.OperationKind,
				request.OperationKey, request.FingerprintVersion, request.Fingerprint[:],
				request.ImmutableRequest, nullableCommandString(request.SelectedURI.String()),
				nullableCommandString(request.SelectedRkey.String()), nullableCommandString(request.ExpectedOwner.String()),
				nullableCommandInt64(request.ExpectedOwnerGeneration), nullableCommandString(request.ExpectedTarget.String()), now)
			if err != nil {
				return fmt.Errorf("insert PDS command: %w", err)
			}
			inserted := result.RowsAffected() == 1
			var fingerprint []byte
			var state string
			var selectedURI, selectedRkey, expectedOwner, expectedTarget *string
			var expectedOwnerGeneration *int64
			if err := tx.QueryRow(operationCtx, `
			SELECT id,owner_generation,request_fingerprint,state,selected_uri,selected_rkey,
				expected_owner_did,expected_owner_generation,expected_target_did,created_at,active_plan_version
			FROM pds_commands
			WHERE owner_did=$1 AND operation_kind=$2 AND operation_key=$3
			FOR UPDATE
		`, request.Owner, request.OperationKind, request.OperationKey).Scan(
				&prepared.ID, &prepared.OwnerGeneration, &fingerprint, &state, &selectedURI, &selectedRkey,
				&expectedOwner, &expectedOwnerGeneration, &expectedTarget, &prepared.CreatedAt, &prepared.ActivePlanVersion,
			); err != nil {
				return fmt.Errorf("read scoped PDS command: %w", err)
			}
			if prepared.OwnerGeneration != request.OwnerGeneration || !bytes.Equal(fingerprint, request.Fingerprint[:]) {
				return ErrIdempotencyConflict
			}
			if len(fingerprint) != len(prepared.Fingerprint) {
				return errors.New("invalid persisted request fingerprint")
			}
			copy(prepared.Fingerprint[:], fingerprint)
			prepared.ScopedKey = ScopedCommandKey{Owner: request.Owner, OperationKind: request.OperationKind, OperationKey: request.OperationKey}
			prepared.State = CommandState(state)
			prepared.Replay = !inserted
			if selectedURI != nil {
				prepared.SelectedURI = syntax.ATURI(*selectedURI)
			}
			if selectedRkey != nil {
				prepared.SelectedRkey = syntax.RecordKey(*selectedRkey)
			}
			if expectedOwner != nil {
				prepared.ExpectedOwner = syntax.DID(*expectedOwner)
			}
			if expectedOwnerGeneration != nil {
				prepared.ExpectedOwnerGeneration = *expectedOwnerGeneration
			}
			if expectedTarget != nil {
				prepared.ExpectedTarget = syntax.DID(*expectedTarget)
			}
			return nil
		})
	}
	var err error
	if store.lifecycles != nil && !store.lifecycles.HasActiveEffectScope(ctx) {
		err = store.lifecycles.WithActiveEffects(
			ctx,
			[]ownerlifecycle.ExpectedOwner{{Owner: request.Owner, Generation: request.OwnerGeneration}},
			prepare,
		)
	} else {
		err = prepare(ctx)
	}
	if err != nil {
		return PreparedCommand{}, err
	}
	return prepared, nil
}

func (store *Store) CompactExpired(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		return 0, errors.New("command compaction limit must be positive")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id FROM pds_commands
		WHERE state IN ('accepted','rejected') AND replay_expires_at <= $1
		ORDER BY replay_expires_at,id
		LIMIT $2
	`, store.now().UTC().Truncate(time.Microsecond), limit)
	if err != nil {
		return 0, fmt.Errorf("list expired PDS commands: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	compacted := 0
	for _, id := range ids {
		didCompact := false
		err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
			var owner syntax.DID
			var kind string
			var key uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT owner_did,operation_kind,operation_key FROM pds_commands WHERE id=$1`, id).Scan(&owner, &kind, &key); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return err
			}
			if err := lockScopedCommandKey(ctx, tx, owner, kind, key); err != nil {
				return err
			}
			var state CommandState
			var replayExpiresAt time.Time
			if err := tx.QueryRow(ctx, `SELECT state,replay_expires_at FROM pds_commands WHERE id=$1 FOR UPDATE`, id).Scan(&state, &replayExpiresAt); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return err
			}
			now := store.now().UTC().Truncate(time.Microsecond)
			if !ShouldCompact(state, replayExpiresAt, now) {
				return nil
			}
			hash := ScopedKeyHash(owner, kind, key)
			if _, err := tx.Exec(ctx, `
				INSERT INTO pds_command_tombstones(owner_did,operation_kind,scoped_key_hash,compacted_at)
				VALUES($1,$2,$3,$4)
				ON CONFLICT DO NOTHING
			`, owner, kind, hash[:], now); err != nil {
				return fmt.Errorf("insert PDS command tombstone: %w", err)
			}
			if _, err := tx.Exec(ctx, `DELETE FROM pds_commands WHERE id=$1`, id); err != nil {
				return fmt.Errorf("delete compacted PDS command: %w", err)
			}
			didCompact = true
			return nil
		})
		if err != nil {
			return compacted, err
		}
		if didCompact {
			compacted++
		}
	}
	return compacted, nil
}

func ScopedKeyHash(owner syntax.DID, operationKind string, operationKey uuid.UUID) [32]byte {
	return sha256.Sum256([]byte("craftsky-pds-command-scoped-key-v1\n" + owner.String() + "\n" + operationKind + "\n" + operationKey.String()))
}

func lockScopedCommandKey(ctx context.Context, tx pgx.Tx, owner syntax.DID, operationKind string, operationKey uuid.UUID) error {
	key := owner.String() + "\n" + operationKind + "\n" + operationKey.String()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
		return fmt.Errorf("lock scoped PDS command key: %w", err)
	}
	return nil
}

func (store *Store) BeginDispatch(ctx context.Context, commandID uuid.UUID, plan ExactPlan) (DispatchAttempt, error) {
	if commandID == uuid.Nil || plan.Version < 1 || plan.FingerprintVersion < 1 || len(plan.Steps) == 0 {
		return DispatchAttempt{}, errors.New("invalid exact dispatch plan")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	remoteDeadline := plan.RemoteDeadline.UTC().Truncate(time.Microsecond)
	if remoteDeadline.IsZero() {
		remoteDeadline = now
	}
	attempt := DispatchAttempt{ID: store.newUUID(), CommandID: commandID, PlanVersion: plan.Version}
	err := store.withTransaction(ctx, func(tx pgx.Tx) error {
		var rawState string
		var activePlanVersion int
		if err := tx.QueryRow(ctx, `SELECT state,active_plan_version FROM pds_commands WHERE id=$1 FOR UPDATE`, commandID).Scan(&rawState, &activePlanVersion); err != nil {
			return fmt.Errorf("lock PDS command for dispatch: %w", err)
		}
		canDispatch := CanTransition(CommandState(rawState), CommandDispatching)
		if CommandState(rawState) == CommandDispatching && plan.Version > activePlanVersion {
			var lastOutcome string
			if err := tx.QueryRow(ctx, `
				SELECT outcome FROM pds_command_dispatches
				WHERE command_id=$1 ORDER BY attempt_ordinal DESC LIMIT 1
			`, commandID).Scan(&lastOutcome); err != nil {
				return fmt.Errorf("read previous PDS dispatch outcome: %w", err)
			}
			canDispatch = lastOutcome == string(DispatchInvalidSwap)
		}
		if !canDispatch || plan.Version < activePlanVersion {
			return ErrIllegalTransition
		}
		for ordinal, step := range plan.Steps {
			repo, collection, rkey, err := dispatchStepIdentity(step.URI)
			if err != nil {
				return err
			}
			var record any
			if step.Action != "delete" {
				if !json.Valid(step.Body) {
					return errors.New("dispatch create/update step requires valid record JSON")
				}
				record = step.Body
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO pds_command_steps(
					command_id,plan_version,ordinal,action,repo_did,collection,rkey,
					record,expected_cid,selected_uri
				) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
				ON CONFLICT(command_id,plan_version,ordinal) DO NOTHING
			`, commandID, plan.Version, ordinal, step.Action, repo, collection, rkey,
				record, nullableCommandString(step.ExpectedCID.String()), step.URI); err != nil {
				return fmt.Errorf("persist PDS command step: %w", err)
			}
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(attempt_ordinal),0)+1 FROM pds_command_dispatches WHERE command_id=$1`, commandID).Scan(&attempt.AttemptOrdinal); err != nil {
			return fmt.Errorf("allocate dispatch attempt ordinal: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO pds_command_dispatches(
				id,command_id,attempt_ordinal,plan_version,
				dispatch_fingerprint_version,dispatch_fingerprint,
				repository_cid,repository_revision,remote_deadline,started_at
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, attempt.ID, commandID, attempt.AttemptOrdinal, plan.Version,
			plan.FingerprintVersion, plan.Fingerprint[:], nullableCommandString(plan.RepositoryCID.String()),
			nullableCommandString(plan.RepositoryRevision.String()), remoteDeadline, now); err != nil {
			return fmt.Errorf("persist PDS dispatch intent: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE pds_commands
			SET state='dispatching',active_plan_version=$2,retry_after_seconds=NULL,updated_at=$3
			WHERE id=$1
		`, commandID, plan.Version, now); err != nil {
			return fmt.Errorf("mark PDS command dispatching: %w", err)
		}
		return nil
	})
	if err != nil {
		return DispatchAttempt{}, err
	}
	return attempt, nil
}

func (store *Store) CompleteDispatch(ctx context.Context, attemptID uuid.UUID, completion DispatchCompletion) error {
	if completion.Outcome != CommandAccepted && completion.Outcome != CommandAmbiguous && completion.Outcome != CommandRejected && completion.Outcome != DispatchInvalidSwap {
		return errors.New("invalid dispatch completion outcome")
	}
	if completion.Outcome == CommandAmbiguous && (completion.RetryAfterSeconds < 1 || completion.RetryAfterSeconds > 5) {
		return errors.New("ambiguous dispatch requires bounded retry guidance")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	return store.withTransaction(ctx, func(tx pgx.Tx) error {
		return store.completeDispatchTx(ctx, tx, attemptID, completion, now)
	})
}

func (store *Store) CompleteDispatchAndCommand(
	ctx context.Context,
	attemptID uuid.UUID,
	completion DispatchCompletion,
	result TerminalResult,
) error {
	if completion.Outcome != CommandAccepted && completion.Outcome != CommandRejected {
		return errors.New("terminal dispatch completion must be accepted or rejected")
	}
	if completion.Outcome != result.State {
		return errors.New("dispatch and command terminal outcomes must match")
	}
	if err := validateTerminalResult(result); err != nil {
		return err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	return store.withTransaction(ctx, func(tx pgx.Tx) error {
		var commandID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT command_id FROM pds_command_dispatches WHERE id=$1`, attemptID).Scan(&commandID); err != nil {
			return fmt.Errorf("read terminal PDS dispatch command: %w", err)
		}
		if err := store.completeDispatchTx(ctx, tx, attemptID, completion, now); err != nil {
			return err
		}
		return completeCommandTx(ctx, tx, commandID, result, now)
	})
}

func (store *Store) completeDispatchTx(
	ctx context.Context,
	tx pgx.Tx,
	attemptID uuid.UUID,
	completion DispatchCompletion,
	now time.Time,
) error {
	var commandID uuid.UUID
	var dispatchOutcome, rawState string
	if err := tx.QueryRow(ctx, `
			SELECT dispatch.command_id,dispatch.outcome,command.state
			FROM pds_command_dispatches dispatch
			JOIN pds_commands command ON command.id=dispatch.command_id
			WHERE dispatch.id=$1
			FOR UPDATE OF dispatch,command
		`, attemptID).Scan(&commandID, &dispatchOutcome, &rawState); err != nil {
		return fmt.Errorf("lock PDS dispatch completion: %w", err)
	}
	validTransition := CanTransition(CommandState(rawState), completion.Outcome)
	if completion.Outcome == DispatchInvalidSwap {
		validTransition = CommandState(rawState) == CommandDispatching
	}
	if dispatchOutcome != "dispatching" || !validTransition {
		return ErrIllegalTransition
	}
	if _, err := tx.Exec(ctx, `
			UPDATE pds_command_dispatches
			SET outcome=$2,error_class=$3,result_commit_cid=$4,result_records=$5,completed_at=$6
			WHERE id=$1
		`, attemptID, completion.Outcome, nullableCommandString(completion.ErrorClass),
		nullableCommandString(completion.ResultCommitCID.String()), nullableCommandJSON(completion.ResultRecords), now); err != nil {
		return fmt.Errorf("complete PDS dispatch: %w", err)
	}
	if completion.Outcome == CommandAmbiguous {
		if _, err := tx.Exec(ctx, `UPDATE pds_commands SET state='ambiguous',retry_after_seconds=$2,updated_at=$3 WHERE id=$1`, commandID, completion.RetryAfterSeconds, now); err != nil {
			return fmt.Errorf("mark PDS command ambiguous: %w", err)
		}
		return nil
	}
	return nil
}

func (store *Store) MarkOpenDispatchAmbiguous(
	ctx context.Context,
	commandID uuid.UUID,
	retryAfterSeconds int,
	errorClass string,
) error {
	if commandID == uuid.Nil || retryAfterSeconds < 1 || retryAfterSeconds > 5 {
		return errors.New("invalid open dispatch ambiguity")
	}
	return store.withTransaction(ctx, func(tx pgx.Tx) error {
		var attemptID uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT id FROM pds_command_dispatches
			WHERE command_id=$1 AND outcome='dispatching'
			ORDER BY attempt_ordinal DESC LIMIT 1
			FOR UPDATE
		`, commandID).Scan(&attemptID); errors.Is(err, pgx.ErrNoRows) {
			var commandState, latestOutcome string
			if err := tx.QueryRow(ctx, `
				SELECT command.state,dispatch.outcome
				FROM pds_commands command
				JOIN pds_command_dispatches dispatch ON dispatch.command_id=command.id
				WHERE command.id=$1
				ORDER BY dispatch.attempt_ordinal DESC LIMIT 1
				FOR UPDATE OF command,dispatch
			`, commandID).Scan(&commandState, &latestOutcome); err != nil {
				return fmt.Errorf("read completed PDS dispatch for recovery: %w", err)
			}
			if CommandState(commandState) != CommandDispatching || latestOutcome != string(DispatchInvalidSwap) {
				return ErrIllegalTransition
			}
			if _, err := tx.Exec(ctx, `
				UPDATE pds_commands
				SET state='ambiguous',retry_after_seconds=$2,updated_at=$3
				WHERE id=$1
			`, commandID, retryAfterSeconds, store.now().UTC().Truncate(time.Microsecond)); err != nil {
				return fmt.Errorf("mark completed invalid-swap dispatch ambiguous: %w", err)
			}
			return nil
		} else if err != nil {
			return fmt.Errorf("read open PDS dispatch: %w", err)
		}
		return store.completeDispatchTx(ctx, tx, attemptID, DispatchCompletion{
			Outcome: CommandAmbiguous, RetryAfterSeconds: retryAfterSeconds, ErrorClass: errorClass,
		}, store.now().UTC().Truncate(time.Microsecond))
	})
}

// UnresolvedResult preserves the exact attempted plan after response loss.
// Current PDS state alone cannot prove a later external change did not occur.
func (store *Store) UnresolvedResult(ctx context.Context, command PreparedCommand) (CommandResult, error) {
	if command.State == CommandDispatching {
		if err := store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
			return CommandResult{}, err
		}
	}
	return store.Result(ctx, command.ID)
}

// ResumeKnownInvalidSwap permits a fresh head-bound plan only when the last
// dispatch was definitively rejected by the repository-head guard. Uncertain
// remote outcomes must continue through read-only reconciliation.
func (store *Store) ResumeKnownInvalidSwap(ctx context.Context, command PreparedCommand) (PreparedCommand, int, error) {
	if command.State == CommandPrepared {
		return command, 1, nil
	}
	var ordinal int
	var outcome string
	err := store.pool.QueryRow(ctx, `
		SELECT attempt_ordinal,outcome FROM pds_command_dispatches
		WHERE command_id=$1 ORDER BY attempt_ordinal DESC LIMIT 1
	`, command.ID).Scan(&ordinal, &outcome)
	if errors.Is(err, pgx.ErrNoRows) {
		return command, 1, nil
	}
	if err != nil {
		return command, 0, fmt.Errorf("read last PDS dispatch for swap recovery: %w", err)
	}
	if outcome == string(DispatchInvalidSwap) {
		command.State = CommandPrepared // only the local planning decision; durable state is unchanged
		return command, ordinal + 1, nil
	}
	return command, 1, nil
}

func (store *Store) CompleteCommand(ctx context.Context, commandID uuid.UUID, result TerminalResult) error {
	if err := validateTerminalResult(result); err != nil {
		return err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	return store.withTransaction(ctx, func(tx pgx.Tx) error {
		return completeCommandTx(ctx, tx, commandID, result, now)
	})
}

func validateTerminalResult(result TerminalResult) error {
	if result.State != CommandAccepted && result.State != CommandRejected || result.HTTPStatus < 100 || result.HTTPStatus > 599 {
		return errors.New("invalid terminal command result")
	}
	return nil
}

func completeCommandTx(
	ctx context.Context,
	tx pgx.Tx,
	commandID uuid.UUID,
	result TerminalResult,
	now time.Time,
) error {
	var rawState string
	if err := tx.QueryRow(ctx, `SELECT state FROM pds_commands WHERE id=$1 FOR UPDATE`, commandID).Scan(&rawState); err != nil {
		return fmt.Errorf("lock terminal PDS command: %w", err)
	}
	if !CanTransition(CommandState(rawState), result.State) {
		return ErrIllegalTransition
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pds_commands
		SET state=$2,terminal_http_status=$3,terminal_response_body=$4,
			terminal_response_headers=$5,retry_after_seconds=NULL,
			terminal_at=$6,replay_expires_at=$7,updated_at=$6
		WHERE id=$1
	`, commandID, result.State, result.HTTPStatus, nullableCommandJSON(result.ResponseBody),
		nullableCommandJSON(result.ResponseHeaders), now, ReplayExpiresAt(now)); err != nil {
		return fmt.Errorf("complete terminal PDS command: %w", err)
	}
	return nil
}

func (store *Store) Result(ctx context.Context, commandID uuid.UUID) (CommandResult, error) {
	var result CommandResult
	err := store.withTransaction(ctx, func(tx pgx.Tx) error {
		var rawState string
		var body, headers []byte
		var status, retryAfter *int
		if err := tx.QueryRow(ctx, `
			SELECT state,terminal_http_status,terminal_response_body,terminal_response_headers,retry_after_seconds
			FROM pds_commands WHERE id=$1
		`, commandID).Scan(&rawState, &status, &body, &headers, &retryAfter); err != nil {
			return err
		}
		result.State = CommandState(rawState)
		if status != nil {
			result.HTTPStatus = *status
		}
		result.ResponseBody = json.RawMessage(body)
		result.ResponseHeaders = json.RawMessage(headers)
		if retryAfter != nil {
			result.RetryAfterSeconds = *retryAfter
		}
		return nil
	})
	if err != nil {
		return CommandResult{}, fmt.Errorf("read PDS command result: %w", err)
	}
	return result, nil
}

func (store *Store) withTransaction(
	ctx context.Context,
	callback func(pgx.Tx) error,
) error {
	if store.lifecycles != nil && store.lifecycles.HasActiveEffectScope(ctx) {
		return store.lifecycles.WithActiveEffectTransaction(ctx, callback)
	}
	return pgx.BeginFunc(ctx, store.pool, callback)
}

func dispatchStepIdentity(uri syntax.ATURI) (syntax.DID, syntax.NSID, syntax.RecordKey, error) {
	parsed, err := syntax.ParseATURI(uri.String())
	if err != nil {
		return "", "", "", err
	}
	return parsed.Authority().DID(), parsed.Collection(), parsed.RecordKey(), nil
}

func nullableCommandString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableCommandJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullableCommandInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
