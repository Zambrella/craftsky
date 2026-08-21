package ownerlifecycle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

var (
	ErrEffectSourceAmbiguous = errors.New("PDS effect source match is ambiguous")
	ErrEffectSourceMismatch  = errors.New("PDS effect source does not match its locked attempt")
)

type EffectSourceMatch string

const (
	EffectSourceUnmatched EffectSourceMatch = "unmatched"
	EffectSourceMatched   EffectSourceMatch = "matched"
	EffectSourceAmbiguous EffectSourceMatch = "ambiguous"
	EffectSourceMismatch  EffectSourceMatch = "mismatch"
)

// PDSRecordSourceObservation contains only the exact, non-secret provenance
// fields needed to join one public PDS source to one durable Put attempt.
// Authoritative means the caller obtained the still-current record through a
// read-only PDS reconciliation, rather than merely receiving a Tap delivery.
type PDSRecordSourceObservation struct {
	Owner             syntax.DID
	URI               syntax.ATURI
	CID               syntax.CID
	RecordFingerprint [32]byte
	LockedOperationID string
	Authoritative     bool
}

type EffectSourceResolution struct {
	Match              EffectSourceMatch
	Attempt            EffectAttempt
	NeedsAuthoritative bool
}

// ResolvePDSRecordSourceTx locks and resolves at most one exact Put attempt.
// It never crosses a remote boundary. A live source with multiple identical
// A-to-B-to-A candidates stays ambiguous; an authoritative read may select
// only the newest local mutation when that mutation is also newest for the
// exact URI. Once a source has LockedOperationID, reconciliation can validate
// or detach it but can never exchange it for another attempt.
func ResolvePDSRecordSourceTx(
	ctx context.Context,
	tx pgx.Tx,
	lifecycle Lifecycle,
	observation PDSRecordSourceObservation,
	now time.Time,
) (EffectSourceResolution, error) {
	if tx == nil || lifecycle.Owner == "" || observation.Owner != lifecycle.Owner ||
		observation.URI == "" || observation.CID == "" ||
		observation.RecordFingerprint == ([32]byte{}) || now.IsZero() {
		return EffectSourceResolution{}, errors.New("invalid PDS record source observation")
	}
	if observation.URI.Authority().DID() != observation.Owner {
		return EffectSourceResolution{}, ErrEffectSourceMismatch
	}

	if strings.TrimSpace(observation.LockedOperationID) != "" {
		attempt, err := scanEffectAttempt(tx.QueryRow(ctx, effectAttemptSelect+`
			WHERE operation_id=$1 FOR UPDATE
		`, observation.LockedOperationID))
		if err != nil {
			return EffectSourceResolution{}, err
		}
		if !effectAttemptMatchesSource(attempt, observation) {
			if err := markEffectSourceMismatchTx(ctx, tx, attempt, lifecycle, observation.CID, now); err != nil {
				return EffectSourceResolution{}, err
			}
			return EffectSourceResolution{Match: EffectSourceMismatch}, nil
		}
		return reconcileMatchedEffectSourceTx(ctx, tx, attempt, lifecycle, observation, now)
	}

	latest, err := scanEffectAttempt(tx.QueryRow(ctx, effectAttemptSelect+`
		WHERE owner_did=$1 AND deterministic_key=$2
		  AND effect_kind='pds_record' AND effect_action='put_record'
		ORDER BY mutation_sequence DESC
		LIMIT 1
		FOR UPDATE
	`, observation.Owner, observation.URI))
	if errors.Is(err, pgx.ErrNoRows) {
		return EffectSourceResolution{Match: EffectSourceUnmatched}, nil
	}
	if err != nil {
		return EffectSourceResolution{}, fmt.Errorf("read latest PDS effect source candidate: %w", err)
	}

	rows, err := tx.Query(ctx, effectAttemptSelect+`
		WHERE owner_did=$1 AND deterministic_key=$2
		  AND effect_kind='pds_record' AND effect_action='put_record'
		  AND record_fingerprint=$3
		  AND (result_cid IS NULL OR result_cid=$4)
		  AND remote_outcome NOT IN ('prepared','abandoned_pre_transition')
		ORDER BY mutation_sequence DESC
		LIMIT 2
		FOR UPDATE
	`, observation.Owner, observation.URI, observation.RecordFingerprint[:], observation.CID)
	if err != nil {
		return EffectSourceResolution{}, fmt.Errorf("lock PDS effect source candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]EffectAttempt, 0, 2)
	for rows.Next() {
		attempt, err := scanEffectAttempt(rows)
		if err != nil {
			return EffectSourceResolution{}, err
		}
		candidates = append(candidates, attempt)
	}
	if err := rows.Err(); err != nil {
		return EffectSourceResolution{}, fmt.Errorf("iterate PDS effect source candidates: %w", err)
	}
	if len(candidates) == 0 {
		if observation.Authoritative {
			if unresolvedAttemptMayStillComplete(latest, now) {
				return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
			}
			if err := markEffectSourceMismatchTx(
				ctx, tx, latest, lifecycle, observation.CID, now,
			); err != nil {
				return EffectSourceResolution{}, err
			}
			if err := markOtherUnresolvedPDSAttemptsMismatchTx(
				ctx, tx, latest.Owner, latest.DeterministicKey, "", lifecycle,
				observation.CID, now,
			); err != nil {
				return EffectSourceResolution{}, err
			}
			return EffectSourceResolution{Match: EffectSourceMismatch}, nil
		}
		// A live delivery for a URI with a known Put attempt may represent a
		// later user rewrite, but it may also be a reordered/misidentified
		// effect result. Do not make that distinction from Tap delivery order;
		// the existing read-only repository worker must observe current state.
		return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
	}
	if candidates[0].MutationSequence != latest.MutationSequence {
		if observation.Authoritative {
			if unresolvedAttemptMayStillComplete(latest, now) {
				return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
			}
			if err := markEffectSourceMismatchTx(
				ctx, tx, latest, lifecycle, observation.CID, now,
			); err != nil {
				return EffectSourceResolution{}, err
			}
			if err := markOtherUnresolvedPDSAttemptsMismatchTx(
				ctx, tx, latest.Owner, latest.DeterministicKey, "", lifecycle,
				observation.CID, now,
			); err != nil {
				return EffectSourceResolution{}, err
			}
			return EffectSourceResolution{Match: EffectSourceMismatch}, nil
		}
		return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
	}
	if len(candidates) > 1 {
		// Identical record content (and therefore often an identical CID) is
		// insufficient to distinguish A-to-B-to-A or duplicate Put attempts.
		// Even an authoritative current read must not assign one attempt's
		// lifecycle disposition to another attempt.
		return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
	}
	resolution, err := reconcileMatchedEffectSourceTx(
		ctx, tx, candidates[0], lifecycle, observation, now,
	)
	if err != nil {
		return EffectSourceResolution{}, err
	}
	if observation.Authoritative {
		if err := markOtherUnresolvedPDSAttemptsMismatchTx(
			ctx, tx, candidates[0].Owner, candidates[0].DeterministicKey,
			candidates[0].OperationID, lifecycle, observation.CID, now,
		); err != nil {
			return EffectSourceResolution{}, err
		}
	}
	return resolution, nil
}

// LockedPDSRecordEffectTx rechecks a persisted source link while holding the
// attempt row lock. Projection code must use the returned attempt disposition
// instead of trusting cached source metadata.
func LockedPDSRecordEffectTx(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	observation PDSRecordSourceObservation,
) (EffectAttempt, error) {
	if tx == nil || strings.TrimSpace(operationID) == "" || observation.Owner == "" ||
		observation.URI == "" || observation.CID == "" ||
		observation.RecordFingerprint == ([32]byte{}) {
		return EffectAttempt{}, ErrEffectSourceMismatch
	}
	attempt, err := scanEffectAttempt(tx.QueryRow(ctx, effectAttemptSelect+`
		WHERE operation_id=$1 FOR SHARE
	`, operationID))
	if err != nil {
		return EffectAttempt{}, err
	}
	if !effectAttemptMatchesSource(attempt, observation) {
		return EffectAttempt{}, ErrEffectSourceMismatch
	}
	return attempt, nil
}

func effectAttemptMatchesSource(attempt EffectAttempt, observation PDSRecordSourceObservation) bool {
	return attempt.Owner == observation.Owner && attempt.Kind == EffectPDSRecord &&
		attempt.Action == EffectActionPutRecord &&
		attempt.DeterministicKey == observation.URI.String() &&
		bytes.Equal(attempt.RecordFingerprint[:], observation.RecordFingerprint[:]) &&
		(attempt.ResultCID == "" || attempt.ResultCID == observation.CID.String())
}

func reconcileMatchedEffectSourceTx(
	ctx context.Context,
	tx pgx.Tx,
	attempt EffectAttempt,
	lifecycle Lifecycle,
	observation PDSRecordSourceObservation,
	now time.Time,
) (EffectSourceResolution, error) {
	disposition := ProjectionHiddenNonActive
	needsAuthoritative := false
	if lifecycle.State == StateTerminal {
		// Terminal is the permanent visibility override even when the remote
		// result had already been classified as rejected or mismatched.
		disposition = ProjectionDeniedTerminal
	} else {
		switch attempt.Outcome {
		case OutcomeRejected, OutcomeAbandonedPreTransition,
			OutcomeReconciledNotAccepted, OutcomeReconciliationMismatch:
			disposition = ProjectionNotApplicable
		case OutcomeDispatched, OutcomeUnknownPreTransition,
			OutcomeAccepted, OutcomeReconciledAccepted:
			switch {
			case lifecycle.State == StateActive &&
				(attempt.OwnerGeneration == lifecycle.Generation || observation.Authoritative):
				disposition = ProjectionEligibleCurrent
			case lifecycle.State == StateActive:
				needsAuthoritative = true
			}
		default:
			return EffectSourceResolution{Match: EffectSourceAmbiguous}, nil
		}
	}

	remoteOutcome := attempt.Outcome
	completedAt := attempt.CompletedAt
	reconciledAt := attempt.ReconciledAt
	if attempt.Outcome == OutcomeDispatched || attempt.Outcome == OutcomeUnknownPreTransition {
		remoteOutcome = OutcomeReconciledAccepted
		completedAt = &now
		reconciledAt = &now
	}
	resultCID := attempt.ResultCID
	if resultCID == "" {
		resultCID = observation.CID.String()
	}
	updated, err := scanEffectAttempt(tx.QueryRow(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome=$3,result_cid=$4,projection_disposition=$5,
		    repeat_forbidden=true,completed_at=$6,reconciled_at=$7,updated_at=$8
		WHERE operation_id=$1 AND owner_did=$2
		RETURNING operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
		          request_fingerprint,record_fingerprint,mutation_sequence,expected_cid,result_cid,remote_outcome,
		          projection_disposition,repeat_forbidden,remote_deadline,
		          dispatched_at,completed_at,reconciled_at,created_at,updated_at
	`, attempt.OperationID, attempt.Owner, remoteOutcome, resultCID, disposition,
		completedAt, reconciledAt, now))
	if err != nil {
		return EffectSourceResolution{}, fmt.Errorf("reconcile PDS effect source: %w", err)
	}
	return EffectSourceResolution{
		Match: EffectSourceMatched, Attempt: updated, NeedsAuthoritative: needsAuthoritative,
	}, nil
}

func markEffectSourceMismatchTx(
	ctx context.Context,
	tx pgx.Tx,
	attempt EffectAttempt,
	lifecycle Lifecycle,
	observedCID syntax.CID,
	now time.Time,
) error {
	if attempt.Outcome != OutcomeDispatched && attempt.Outcome != OutcomeUnknownPreTransition &&
		attempt.Outcome != OutcomeAccepted && attempt.Outcome != OutcomeReconciledAccepted {
		return nil
	}
	disposition := ProjectionNotApplicable
	if lifecycle.State == StateTerminal {
		disposition = ProjectionDeniedTerminal
	}
	_, err := tx.Exec(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome='reconciliation_mismatch',result_cid=$3,
		    projection_disposition=$4,repeat_forbidden=true,
		    completed_at=$5,reconciled_at=$5,updated_at=$5
		WHERE operation_id=$1 AND owner_did=$2
		  AND remote_outcome IN (
		      'dispatched','outcome_unknown_pre_transition','accepted','reconciled_accepted'
		  )
	`, attempt.OperationID, attempt.Owner, observedCID, disposition, now)
	if err != nil {
		return fmt.Errorf("mark PDS effect source mismatch: %w", err)
	}
	return nil
}

func unresolvedAttemptMayStillComplete(attempt EffectAttempt, now time.Time) bool {
	return (attempt.Outcome == OutcomeDispatched || attempt.Outcome == OutcomeUnknownPreTransition) &&
		attempt.RemoteDeadline.After(now)
}

func markOtherUnresolvedPDSAttemptsMismatchTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	deterministicKey string,
	exceptOperationID string,
	lifecycle Lifecycle,
	observedCID syntax.CID,
	now time.Time,
) error {
	disposition := ProjectionNotApplicable
	if lifecycle.State == StateTerminal {
		disposition = ProjectionDeniedTerminal
	}
	_, err := tx.Exec(ctx, `
		UPDATE owner_effect_attempts
		SET remote_outcome='reconciliation_mismatch',result_cid=$4,
		    projection_disposition=$5,repeat_forbidden=true,
		    completed_at=$6,reconciled_at=$6,updated_at=$6
		WHERE owner_did=$1 AND deterministic_key=$2
		  AND operation_id<>$3
		  AND effect_kind='pds_record' AND effect_action='put_record'
		  AND remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
	`, owner, deterministicKey, exceptOperationID, observedCID, disposition, now)
	if err != nil {
		return fmt.Errorf("mark superseded unresolved PDS effects: %w", err)
	}
	return nil
}
