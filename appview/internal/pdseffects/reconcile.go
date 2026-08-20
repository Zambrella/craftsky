package pdseffects

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// RecordReader is the deliberately read-only remote capability used to
// reconcile a request after ordinary OAuth authority or membership has gone
// away. It cannot repeat PutRecord or DeleteRecord.
type RecordReader interface {
	GetRecord(
		context.Context,
		syntax.DID,
		string,
		string,
		any,
	) (string, error)
}

func (executor *Executor) ReconcilePutRecord(
	ctx context.Context,
	request PutRecordRequest,
	reader RecordReader,
) (RecordResult, error) {
	if executor == nil || executor.attempts == nil || reader == nil {
		return RecordResult{}, errors.New("PDS effect read reconciler is unavailable")
	}
	if err := validateEffectOwner(
		executor.owner,
		request.OperationID,
		request.MutationKey,
		request.Owner,
		request.OwnerGeneration,
		request.ExpectedOwners,
	); err != nil {
		return RecordResult{}, err
	}
	_, fingerprint, err := canonicalPutBody(
		request.Owner,
		request.Collection,
		request.Rkey,
		request.Record,
		request.ExpectedCID,
	)
	if err != nil {
		return RecordResult{}, err
	}
	_, recordFingerprint, err := canonicalPutBody(
		request.Owner,
		request.Collection,
		request.Rkey,
		request.Record,
		"",
	)
	if err != nil {
		return RecordResult{}, err
	}
	uri, err := deterministicRecordURI(request.Owner, request.Collection, request.Rkey)
	if err != nil {
		return RecordResult{}, err
	}
	attempt, err := executor.attempts.GetEffectAttempt(ctx, request.OperationID)
	if err != nil {
		return RecordResult{}, err
	}
	if err := validateStoredAttempt(
		attempt,
		request.MutationKey,
		request.Owner,
		request.OwnerGeneration,
		ownerlifecycle.EffectPDSRecord,
		ownerlifecycle.EffectActionPutRecord,
		uri.String(),
		fingerprint,
		recordFingerprint,
		request.ExpectedCID.String(),
	); err != nil {
		return RecordResult{}, &ConflictError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       err,
		}
	}
	if result, done, err := reconciledRecordResult(attempt, uri); done {
		return result, err
	}
	var current map[string]any
	callCtx, cancel := context.WithTimeout(ctx, executor.timeout)
	cid, readErr := reader.GetRecord(
		callCtx,
		request.Owner,
		request.Collection.String(),
		request.Rkey.String(),
		&current,
	)
	cancel()
	if readErr != nil {
		return RecordResult{}, &OutcomeAmbiguousError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       readErr,
		}
	}
	_, currentFingerprint, err := canonicalPutBody(
		request.Owner,
		request.Collection,
		request.Rkey,
		current,
		"",
	)
	if err != nil {
		return RecordResult{}, err
	}
	if currentFingerprint != recordFingerprint {
		_, reconcileErr := executor.attempts.ReconcileEffectAttempt(
			ctx,
			ownerlifecycle.ReconcileEffectRequest{
				OperationID: request.OperationID,
				Owner:       request.Owner,
				Outcome:     ownerlifecycle.OutcomeReconciliationMismatch,
				ResultCID:   cid,
			},
		)
		return RecordResult{}, &ConflictError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       reconcileErr,
		}
	}
	reconciled, err := executor.attempts.ReconcileEffectAttempt(
		ctx,
		ownerlifecycle.ReconcileEffectRequest{
			OperationID: request.OperationID,
			Owner:       request.Owner,
			Outcome:     ownerlifecycle.OutcomeReconciledAccepted,
			ResultCID:   cid,
		},
	)
	if err != nil {
		return RecordResult{}, err
	}
	return RecordResult{URI: uri, CID: syntax.CID(reconciled.ResultCID)}, nil
}

func (executor *Executor) ReconcileDeleteRecord(
	ctx context.Context,
	request DeleteRecordRequest,
	reader RecordReader,
) (RecordResult, error) {
	if executor == nil || executor.attempts == nil || reader == nil {
		return RecordResult{}, errors.New("PDS effect read reconciler is unavailable")
	}
	if err := validateEffectOwner(
		executor.owner,
		request.OperationID,
		request.MutationKey,
		request.Owner,
		request.OwnerGeneration,
		request.ExpectedOwners,
	); err != nil {
		return RecordResult{}, err
	}
	_, fingerprint, uri, err := canonicalDeleteBody(
		request.Owner,
		request.Collection,
		request.Rkey,
		request.ExpectedCID,
	)
	if err != nil {
		return RecordResult{}, err
	}
	attempt, err := executor.attempts.GetEffectAttempt(ctx, request.OperationID)
	if err != nil {
		return RecordResult{}, err
	}
	if err := validateStoredAttempt(
		attempt,
		request.MutationKey,
		request.Owner,
		request.OwnerGeneration,
		ownerlifecycle.EffectPDSRecord,
		ownerlifecycle.EffectActionDeleteRecord,
		uri.String(),
		fingerprint,
		[32]byte{},
		request.ExpectedCID.String(),
	); err != nil {
		return RecordResult{}, &ConflictError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       err,
		}
	}
	if result, done, err := reconciledRecordResult(attempt, uri); done {
		return result, err
	}

	var current map[string]any
	callCtx, cancel := context.WithTimeout(ctx, executor.timeout)
	currentCID, readErr := reader.GetRecord(
		callCtx,
		request.Owner,
		request.Collection.String(),
		request.Rkey.String(),
		&current,
	)
	cancel()
	if !errors.Is(readErr, auth.ErrRecordNotFound) {
		if readErr == nil && request.ExpectedCID != "" &&
			currentCID != request.ExpectedCID.String() {
			_, reconcileErr := executor.attempts.ReconcileEffectAttempt(
				ctx,
				ownerlifecycle.ReconcileEffectRequest{
					OperationID: request.OperationID,
					Owner:       request.Owner,
					Outcome:     ownerlifecycle.OutcomeReconciliationMismatch,
					ResultCID:   currentCID,
				},
			)
			return RecordResult{}, &ConflictError{
				OperationID: request.OperationID,
				ExactKey:    uri.String(),
				Cause:       errors.Join(auth.ErrRecordSwapConflict, reconcileErr),
			}
		}
		return RecordResult{}, &OutcomeAmbiguousError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       readErr,
		}
	}
	if _, err := executor.attempts.ReconcileEffectAttempt(
		ctx,
		ownerlifecycle.ReconcileEffectRequest{
			OperationID: request.OperationID,
			Owner:       request.Owner,
			Outcome:     ownerlifecycle.OutcomeReconciledAccepted,
		},
	); err != nil {
		return RecordResult{}, err
	}
	return RecordResult{URI: uri}, nil
}

func validateStoredAttempt(
	attempt ownerlifecycle.EffectAttempt,
	mutationKey string,
	owner syntax.DID,
	generation int64,
	kind ownerlifecycle.EffectKind,
	action ownerlifecycle.EffectAction,
	exactKey string,
	fingerprint [32]byte,
	recordFingerprint [32]byte,
	expectedCID string,
) error {
	if attempt.Owner != owner || attempt.OwnerGeneration != generation ||
		attempt.Kind != kind || attempt.Action != action ||
		attempt.MutationKey != mutationKey ||
		attempt.DeterministicKey != exactKey || attempt.RequestFingerprint != fingerprint ||
		attempt.RecordFingerprint != recordFingerprint ||
		attempt.ExpectedCID != expectedCID {
		return ownerlifecycle.ErrAttemptConflict
	}
	return nil
}

func reconciledRecordResult(
	attempt ownerlifecycle.EffectAttempt,
	uri syntax.ATURI,
) (RecordResult, bool, error) {
	switch attempt.Outcome {
	case ownerlifecycle.OutcomeAccepted, ownerlifecycle.OutcomeReconciledAccepted:
		return RecordResult{URI: uri, CID: syntax.CID(attempt.ResultCID)}, true, nil
	case ownerlifecycle.OutcomeDispatched, ownerlifecycle.OutcomeUnknownPreTransition:
		return RecordResult{}, false, nil
	default:
		return RecordResult{}, true, ErrEffectRejected
	}
}
