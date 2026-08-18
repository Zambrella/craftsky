package ownerlifecycle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// CreateEffectAttempt persists a deterministic pre-call record. It is valid
// only inside WithActiveEffects so creation and the later remote call share the
// same owner fence and generation recheck.
func (store *Store) CreateEffectAttempt(ctx context.Context, request NewEffectAttempt) (EffectAttempt, error) {
	if request.Action == "" {
		request.Action = defaultEffectAction(request.Kind)
	}
	if request.MutationKey == "" {
		request.MutationKey = request.OperationID
	}
	if err := requireActiveEffectScope(ctx, request.Owner, request.OwnerGeneration); err != nil {
		return EffectAttempt{}, err
	}
	if !validEffectRequest(request, store.now().UTC()) {
		return EffectAttempt{}, errors.New("invalid owner effect attempt")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	deadline := request.RemoteDeadline.UTC().Truncate(time.Microsecond)
	expectedCID := nullableString(request.ExpectedCID)
	if err := store.exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,expected_cid,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'prepared','pending',false,$10,$11,$11)
		ON CONFLICT DO NOTHING
	`, request.OperationID, request.Owner, request.OwnerGeneration, request.Kind, request.Action,
		request.MutationKey, request.DeterministicKey, request.RequestFingerprint[:], expectedCID, deadline, now); err != nil {
		return EffectAttempt{}, fmt.Errorf("create owner effect attempt: %w", err)
	}
	attempt, err := store.GetEffectAttempt(ctx, request.OperationID)
	if errors.Is(err, pgx.ErrNoRows) {
		attempt, err = store.getEffectAttemptByRemoteIdentity(
			ctx,
			request.Owner,
			request.OwnerGeneration,
			request.Kind,
			request.Action,
			request.MutationKey,
			request.DeterministicKey,
		)
	}
	if err != nil {
		return EffectAttempt{}, err
	}
	if attempt.Owner != request.Owner || attempt.OwnerGeneration != request.OwnerGeneration ||
		attempt.Kind != request.Kind || attempt.Action != request.Action ||
		attempt.MutationKey != request.MutationKey ||
		attempt.DeterministicKey != request.DeterministicKey ||
		!bytes.Equal(attempt.RequestFingerprint[:], request.RequestFingerprint[:]) ||
		attempt.ExpectedCID != request.ExpectedCID {
		return EffectAttempt{}, ErrAttemptConflict
	}
	return attempt, nil
}

func (store *Store) getEffectAttemptByRemoteIdentity(
	ctx context.Context,
	owner syntax.DID,
	ownerGeneration int64,
	kind EffectKind,
	action EffectAction,
	mutationKey string,
	deterministicKey string,
) (EffectAttempt, error) {
	return scanEffectAttempt(store.queryRow(ctx, effectAttemptSelect+`
		WHERE owner_did=$1 AND owner_generation=$2 AND effect_kind=$3
		  AND effect_action=$4 AND mutation_key=$5 AND deterministic_key=$6
	`, owner, ownerGeneration, kind, action, mutationKey, deterministicKey))
}

func (store *Store) GetEffectAttempt(ctx context.Context, operationID string) (EffectAttempt, error) {
	if !validBoundedString(operationID, 512) {
		return EffectAttempt{}, errors.New("invalid owner effect operation ID")
	}
	return scanEffectAttempt(store.queryRow(ctx, effectAttemptSelect+` WHERE operation_id=$1`, operationID))
}

// MarkAttemptDispatched is the durable no-repeat boundary immediately before
// the caller performs remote I/O. A crash after this commit is outcome-unknown,
// never permission to repeat the request.
func (store *Store) MarkAttemptDispatched(
	ctx context.Context,
	operationID string,
	owner syntax.DID,
	ownerGeneration int64,
) (EffectAttempt, error) {
	if err := requireActiveEffectScope(ctx, owner, ownerGeneration); err != nil {
		return EffectAttempt{}, err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	attempt, err := scanEffectAttempt(store.queryRow(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome='dispatched',repeat_forbidden=true,
		    dispatched_at=$4,updated_at=$4
		WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
		  AND remote_outcome='prepared' AND remote_deadline>$4
		RETURNING operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
		          request_fingerprint,expected_cid,result_cid,remote_outcome,
		          projection_disposition,repeat_forbidden,remote_deadline,
		          dispatched_at,completed_at,reconciled_at,created_at,updated_at
	`, operationID, owner, ownerGeneration, now))
	if err == nil {
		return attempt, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return EffectAttempt{}, err
	}
	return EffectAttempt{}, ErrAttemptState
}

// CompleteEffectAttempt records a definite synchronous remote result while
// the active owner fence is still held.
func (store *Store) CompleteEffectAttempt(
	ctx context.Context,
	operationID string,
	owner syntax.DID,
	ownerGeneration int64,
	outcome EffectOutcome,
	resultCID string,
) (EffectAttempt, error) {
	if err := requireActiveEffectScope(ctx, owner, ownerGeneration); err != nil {
		return EffectAttempt{}, err
	}
	if outcome != OutcomeAccepted && outcome != OutcomeRejected {
		return EffectAttempt{}, ErrAttemptState
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	disposition := ProjectionEligibleCurrent
	if outcome == OutcomeRejected {
		disposition = ProjectionNotApplicable
	}
	attempt, err := scanEffectAttempt(store.queryRow(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome=$4,result_cid=$5,projection_disposition=$6,
		    completed_at=$7,updated_at=$7
		WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
		  AND remote_outcome='dispatched'
		RETURNING operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
		          request_fingerprint,expected_cid,result_cid,remote_outcome,
		          projection_disposition,repeat_forbidden,remote_deadline,
		          dispatched_at,completed_at,reconciled_at,created_at,updated_at
	`, operationID, owner, ownerGeneration, outcome, nullableString(resultCID), disposition, now))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EffectAttempt{}, ErrAttemptState
		}
		return EffectAttempt{}, err
	}
	return attempt, nil
}

// ReconcileEffectAttempt classifies an outcome-unknown request without issuing
// another remote write. It shares the owner fence with transitions so terminal
// denial and non-active hiding cannot race the local classification.
func (store *Store) ReconcileEffectAttempt(
	ctx context.Context,
	request ReconcileEffectRequest,
) (EffectAttempt, error) {
	if request.Owner == "" || !validBoundedString(request.OperationID, 512) {
		return EffectAttempt{}, ErrAttemptState
	}
	switch request.Outcome {
	case OutcomeReconciledAccepted, OutcomeReconciledNotAccepted, OutcomeReconciliationMismatch:
	default:
		return EffectAttempt{}, ErrAttemptState
	}
	var reconciled EffectAttempt
	err := store.fencer.WithShared(ctx, []syntax.DID{request.Owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			lifecycle, err := scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, request.Owner,
			))
			if err != nil {
				return err
			}
			disposition := ProjectionHiddenNonActive
			if lifecycle.State == StateTerminal {
				disposition = ProjectionDeniedTerminal
			} else if lifecycle.State == StateActive && request.Outcome == OutcomeReconciledAccepted {
				disposition = ProjectionEligibleCurrent
			} else if request.Outcome != OutcomeReconciledAccepted {
				disposition = ProjectionNotApplicable
			}
			now := store.now().UTC().Truncate(time.Microsecond)
			reconciled, err = scanEffectAttempt(tx.QueryRow(fenceCtx, `
				UPDATE owner_effect_attempts
				SET remote_outcome=$3,result_cid=$4,projection_disposition=$5,
				    completed_at=$6,reconciled_at=$6,updated_at=$6
				WHERE operation_id=$1 AND owner_did=$2
				  AND remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
				RETURNING operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
				          request_fingerprint,expected_cid,result_cid,remote_outcome,
				          projection_disposition,repeat_forbidden,remote_deadline,
				          dispatched_at,completed_at,reconciled_at,created_at,updated_at
			`, request.OperationID, request.Owner, request.Outcome,
				nullableString(request.ResultCID), disposition, now))
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAttemptState
			}
			return err
		})
	})
	return reconciled, err
}

// ConfirmReconciledEffectCurrent promotes a proved accepted old-generation PDS
// record only after rejoin, under the current active-generation fence, and only
// when the authoritative current CID matches the reconciled result. Rejoin by
// itself never revives an old attempt.
func (store *Store) ConfirmReconciledEffectCurrent(
	ctx context.Context,
	operationID string,
	owner syntax.DID,
	currentGeneration int64,
	currentCID string,
) (EffectAttempt, error) {
	if err := requireActiveEffectScope(ctx, owner, currentGeneration); err != nil {
		return EffectAttempt{}, err
	}
	if !validBoundedString(operationID, 512) || strings.TrimSpace(currentCID) == "" {
		return EffectAttempt{}, ErrAttemptState
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	attempt, err := scanEffectAttempt(store.queryRow(ctx, `
		UPDATE owner_effect_attempts
		SET projection_disposition='eligible_current',updated_at=$4
		WHERE operation_id=$1 AND owner_did=$2
		  AND remote_outcome='reconciled_accepted'
		  AND result_cid=$3
		  AND projection_disposition='hidden_non_active'
		RETURNING operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
		          request_fingerprint,expected_cid,result_cid,remote_outcome,
		          projection_disposition,repeat_forbidden,remote_deadline,
		          dispatched_at,completed_at,reconciled_at,created_at,updated_at
	`, operationID, owner, currentCID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		existing, readErr := store.GetEffectAttempt(ctx, operationID)
		if readErr == nil && existing.Owner == owner && existing.Outcome == OutcomeReconciledAccepted &&
			existing.ResultCID == currentCID && existing.ProjectionDisposition == ProjectionEligibleCurrent {
			return existing, nil
		}
		return EffectAttempt{}, ErrAttemptState
	}
	return attempt, err
}

func closeOwnerEffectsTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	generation int64,
	terminal bool,
	now time.Time,
) error {
	disposition := ProjectionHiddenNonActive
	if terminal {
		disposition = ProjectionDeniedTerminal
		_, err := tx.Exec(ctx, `
			UPDATE owner_effect_attempts
			SET remote_outcome=CASE
			        WHEN remote_outcome='prepared' THEN 'abandoned_pre_transition'
			        WHEN remote_outcome='dispatched' THEN 'outcome_unknown_pre_transition'
			        ELSE remote_outcome
			    END,
			    projection_disposition=$3,
			    repeat_forbidden=true,
			    completed_at=CASE
			        WHEN remote_outcome='prepared' THEN $2
			        ELSE completed_at
			    END,
			    updated_at=$2
			WHERE owner_did=$1
		`, owner, now, disposition)
		if err != nil {
			return fmt.Errorf("close terminal owner effects: %w", err)
		}
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome=CASE
		        WHEN remote_outcome='prepared' THEN 'abandoned_pre_transition'
		        WHEN remote_outcome='dispatched' THEN 'outcome_unknown_pre_transition'
		        ELSE remote_outcome
		    END,
		    projection_disposition=$4,
		    repeat_forbidden=true,
		    completed_at=CASE
		        WHEN remote_outcome='prepared' THEN $3
		        ELSE completed_at
		    END,
		    updated_at=$3
		WHERE owner_did=$1 AND owner_generation=$2
	`, owner, generation, now, disposition)
	if err != nil {
		return fmt.Errorf("close owner effect generation: %w", err)
	}
	return nil
}

func requireActiveEffectScope(ctx context.Context, owner syntax.DID, generation int64) error {
	scope, ok := ctx.Value(activeEffectsContextKey{}).(activeEffectScope)
	if !ok {
		return ErrFenceRequired
	}
	if expected, exists := scope[owner]; !exists {
		return ErrFenceRequired
	} else if expected != generation {
		return ErrGenerationChanged
	}
	return nil
}

func validEffectRequest(request NewEffectAttempt, now time.Time) bool {
	return validBoundedString(request.OperationID, 512) && request.Owner != "" &&
		request.OwnerGeneration > 0 && request.Kind.valid() && request.Action.validFor(request.Kind) &&
		validBoundedString(request.MutationKey, 512) &&
		validBoundedString(request.DeterministicKey, 2048) &&
		(request.ExpectedCID == "" || strings.TrimSpace(request.ExpectedCID) != "") &&
		request.RemoteDeadline.After(now)
}

func validBoundedString(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= limit
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

const effectAttemptSelect = `
	SELECT operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
	       request_fingerprint,expected_cid,result_cid,remote_outcome,
	       projection_disposition,repeat_forbidden,remote_deadline,
	       dispatched_at,completed_at,reconciled_at,created_at,updated_at
	FROM owner_effect_attempts`

type effectAttemptRowScanner interface {
	Scan(dest ...any) error
}

func scanEffectAttempt(row effectAttemptRowScanner) (EffectAttempt, error) {
	var (
		attempt     EffectAttempt
		fingerprint []byte
		expectedCID *string
		resultCID   *string
	)
	err := row.Scan(
		&attempt.OperationID,
		&attempt.Owner,
		&attempt.OwnerGeneration,
		&attempt.Kind,
		&attempt.Action,
		&attempt.MutationKey,
		&attempt.DeterministicKey,
		&fingerprint,
		&expectedCID,
		&resultCID,
		&attempt.Outcome,
		&attempt.ProjectionDisposition,
		&attempt.RepeatForbidden,
		&attempt.RemoteDeadline,
		&attempt.DispatchedAt,
		&attempt.CompletedAt,
		&attempt.ReconciledAt,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	)
	if err != nil {
		return EffectAttempt{}, err
	}
	if len(fingerprint) != sha256.Size {
		return EffectAttempt{}, errors.New("invalid stored owner effect fingerprint")
	}
	copy(attempt.RequestFingerprint[:], fingerprint)
	if expectedCID != nil {
		attempt.ExpectedCID = *expectedCID
	}
	if resultCID != nil {
		attempt.ResultCID = *resultCID
	}
	return attempt, nil
}
