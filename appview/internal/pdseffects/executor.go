package pdseffects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type Executor struct {
	attempts *ownerlifecycle.Store
	boundary auth.ActiveEffectPDSBoundary
	owner    syntax.DID
	timeout  time.Duration
	now      func() time.Time
}

// EffectExecutor is the ordinary authenticated mutation capability exposed to
// handlers. It deliberately omits raw PDS access and read-only reconciliation.
type EffectExecutor interface {
	ResolveExpectedOwners(
		context.Context,
		int64,
		[]syntax.DID,
	) ([]ownerlifecycle.ExpectedOwner, error)
	PutRecord(context.Context, PutRecordRequest) (RecordResult, error)
	DeleteRecord(context.Context, DeleteRecordRequest) (RecordResult, error)
	UploadBlob(context.Context, UploadBlobRequest) (*auth.UploadedBlob, error)
}

func NewExecutor(
	attempts *ownerlifecycle.Store,
	boundary auth.ActiveEffectPDSBoundary,
	owner syntax.DID,
	timeout time.Duration,
	now func() time.Time,
) (*Executor, error) {
	if attempts == nil || boundary == nil || owner == "" || timeout <= 0 {
		return nil, errors.New("durable PDS effect executor dependencies and timeout are required")
	}
	if now == nil {
		now = time.Now
	}
	return &Executor{
		attempts: attempts,
		boundary: boundary,
		owner:    owner,
		timeout:  timeout,
		now:      now,
	}, nil
}

type PutRecordRequest struct {
	// OperationID is the globally unique durable attempt-row identifier.
	OperationID string
	// MutationKey is the caller-stable idempotency/version key. Retries reuse
	// it; every later intended mutation, including A-to-B-to-A, uses a new key.
	MutationKey     string
	Owner           syntax.DID
	OwnerGeneration int64
	ExpectedOwners  []ownerlifecycle.ExpectedOwner
	Collection      syntax.NSID
	Rkey            syntax.RecordKey
	Record          any
	ExpectedCID     syntax.CID
}

type RecordResult struct {
	URI syntax.ATURI
	CID syntax.CID
}

type DeleteRecordRequest struct {
	// OperationID and MutationKey have the same contract as PutRecordRequest.
	OperationID     string
	MutationKey     string
	Owner           syntax.DID
	OwnerGeneration int64
	ExpectedOwners  []ownerlifecycle.ExpectedOwner
	Collection      syntax.NSID
	Rkey            syntax.RecordKey
	ExpectedCID     syntax.CID
}

func (executor *Executor) PutRecord(
	ctx context.Context,
	request PutRecordRequest,
) (RecordResult, error) {
	if executor == nil || executor.attempts == nil || executor.boundary == nil {
		return RecordResult{}, errors.New("durable PDS effect executor is unavailable")
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
	var result RecordResult
	err = executor.boundary.WithActiveEffects(
		ctx,
		request.ExpectedOwners,
		func(effectCtx context.Context, client auth.PDSClient) error {
			attempt, err := executor.attempts.CreateEffectAttempt(
				effectCtx,
				ownerlifecycle.NewEffectAttempt{
					OperationID:        request.OperationID,
					MutationKey:        request.MutationKey,
					Owner:              request.Owner,
					OwnerGeneration:    request.OwnerGeneration,
					Kind:               ownerlifecycle.EffectPDSRecord,
					Action:             ownerlifecycle.EffectActionPutRecord,
					DeterministicKey:   uri.String(),
					RequestFingerprint: fingerprint,
					ExpectedCID:        request.ExpectedCID.String(),
					RemoteDeadline:     executor.now().UTC().Add(executor.timeout),
				},
			)
			if errors.Is(err, ownerlifecycle.ErrAttemptConflict) {
				return &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}
			}
			if err != nil {
				return err
			}
			switch attempt.Outcome {
			case ownerlifecycle.OutcomeAccepted, ownerlifecycle.OutcomeReconciledAccepted:
				if attempt.ResultCID == "" {
					return &OutcomeAmbiguousError{
						OperationID: request.OperationID,
						ExactKey:    uri.String(),
						Cause:       errors.New("accepted PDS Put has no authoritative CID"),
					}
				}
				result = RecordResult{URI: uri, CID: syntax.CID(attempt.ResultCID)}
				return nil
			case ownerlifecycle.OutcomePrepared:
				// Continue to the one permitted remote call below.
			case ownerlifecycle.OutcomeDispatched:
				var reconcileErr error
				result, reconcileErr = executor.reconcilePutInsideActive(
					effectCtx, client, request, attempt, uri, recordFingerprint,
				)
				return reconcileErr
			case ownerlifecycle.OutcomeUnknownPreTransition:
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
				}
			default:
				return ErrEffectRejected
			}

			var conditionalPutter auth.ConditionalPDSRecordPutter
			if request.ExpectedCID != "" {
				var ok bool
				conditionalPutter, ok = client.(auth.ConditionalPDSRecordPutter)
				if !ok || conditionalPutter == nil {
					return auth.ErrConditionalPutUnsupported
				}
			}

			attempt, err = executor.attempts.MarkAttemptDispatched(
				effectCtx,
				attempt.OperationID,
				request.Owner,
				request.OwnerGeneration,
			)
			if err != nil {
				return err
			}
			callCtx, cancel := context.WithTimeout(effectCtx, executor.timeout)
			if conditionalPutter != nil {
				err = conditionalPutter.PutRecordWithSwap(
					callCtx,
					request.Owner,
					request.Collection.String(),
					request.Rkey.String(),
					request.Record,
					request.ExpectedCID,
				)
			} else {
				err = client.PutRecord(
					callCtx,
					request.Owner,
					request.Collection.String(),
					request.Rkey.String(),
					request.Record,
				)
			}
			cancel()
			if errors.Is(err, auth.ErrRecordSwapConflict) {
				_, completeErr := executor.attempts.CompleteEffectAttempt(
					effectCtx,
					attempt.OperationID,
					request.Owner,
					request.OwnerGeneration,
					ownerlifecycle.OutcomeRejected,
					"",
				)
				return errors.Join(&ConflictError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}, completeErr)
			}
			if err != nil {
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}
			}
			result, err = executor.reconcilePutInsideActive(
				effectCtx, client, request, attempt, uri, recordFingerprint,
			)
			return err
		},
	)
	return result, err
}

func (executor *Executor) reconcilePutInsideActive(
	ctx context.Context,
	client auth.PDSClient,
	request PutRecordRequest,
	attempt ownerlifecycle.EffectAttempt,
	uri syntax.ATURI,
	fingerprint [32]byte,
) (RecordResult, error) {
	var current map[string]any
	callCtx, cancel := context.WithTimeout(ctx, executor.timeout)
	cid, err := client.GetRecord(
		callCtx,
		request.Owner,
		request.Collection.String(),
		request.Rkey.String(),
		&current,
	)
	cancel()
	if err != nil {
		return RecordResult{}, &OutcomeAmbiguousError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       err,
		}
	}
	if strings.TrimSpace(cid) == "" {
		return RecordResult{}, &OutcomeAmbiguousError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       errors.New("authoritative PDS record has no CID"),
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
	if currentFingerprint != fingerprint {
		return RecordResult{}, &ConflictError{
			OperationID: request.OperationID,
			ExactKey:    uri.String(),
			Cause:       fmt.Errorf("authoritative record CID or canonical body differs"),
		}
	}
	completed, err := executor.attempts.CompleteEffectAttempt(
		ctx,
		attempt.OperationID,
		request.Owner,
		request.OwnerGeneration,
		ownerlifecycle.OutcomeAccepted,
		cid,
	)
	if err != nil {
		return RecordResult{}, err
	}
	return RecordResult{URI: uri, CID: syntax.CID(completed.ResultCID)}, nil
}

func (executor *Executor) DeleteRecord(
	ctx context.Context,
	request DeleteRecordRequest,
) (RecordResult, error) {
	if executor == nil || executor.attempts == nil || executor.boundary == nil {
		return RecordResult{}, errors.New("durable PDS effect executor is unavailable")
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
	var result RecordResult
	err = executor.boundary.WithActiveEffects(
		ctx,
		request.ExpectedOwners,
		func(effectCtx context.Context, client auth.PDSClient) error {
			attempt, err := executor.attempts.CreateEffectAttempt(
				effectCtx,
				ownerlifecycle.NewEffectAttempt{
					OperationID:        request.OperationID,
					MutationKey:        request.MutationKey,
					Owner:              request.Owner,
					OwnerGeneration:    request.OwnerGeneration,
					Kind:               ownerlifecycle.EffectPDSRecord,
					Action:             ownerlifecycle.EffectActionDeleteRecord,
					DeterministicKey:   uri.String(),
					RequestFingerprint: fingerprint,
					ExpectedCID:        request.ExpectedCID.String(),
					RemoteDeadline:     executor.now().UTC().Add(executor.timeout),
				},
			)
			if errors.Is(err, ownerlifecycle.ErrAttemptConflict) {
				return &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}
			}
			if err != nil {
				return err
			}
			switch attempt.Outcome {
			case ownerlifecycle.OutcomeAccepted, ownerlifecycle.OutcomeReconciledAccepted:
				result = RecordResult{URI: uri}
				return nil
			case ownerlifecycle.OutcomePrepared:
				// Continue to the one permitted delete below.
			case ownerlifecycle.OutcomeDispatched:
				callCtx, cancel := context.WithTimeout(effectCtx, executor.timeout)
				var current map[string]any
				currentCID, readErr := client.GetRecord(
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
						return &ConflictError{
							OperationID: request.OperationID,
							ExactKey:    uri.String(),
							Cause:       auth.ErrRecordSwapConflict,
						}
					}
					return &OutcomeAmbiguousError{
						OperationID: request.OperationID,
						ExactKey:    uri.String(),
						Cause:       readErr,
					}
				}
				if _, err := executor.attempts.CompleteEffectAttempt(
					effectCtx,
					attempt.OperationID,
					request.Owner,
					request.OwnerGeneration,
					ownerlifecycle.OutcomeAccepted,
					"",
				); err != nil {
					return err
				}
				result = RecordResult{URI: uri}
				return nil
			case ownerlifecycle.OutcomeUnknownPreTransition:
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
				}
			case ownerlifecycle.OutcomeRejected:
				return &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       auth.ErrRecordSwapConflict,
				}
			default:
				return ErrEffectRejected
			}
			var conditionalDeleter auth.ConditionalPDSRecordDeleter
			if request.ExpectedCID != "" {
				var ok bool
				conditionalDeleter, ok = client.(auth.ConditionalPDSRecordDeleter)
				if !ok || conditionalDeleter == nil {
					return auth.ErrConditionalDeleteUnsupported
				}
			}

			attempt, err = executor.attempts.MarkAttemptDispatched(
				effectCtx,
				attempt.OperationID,
				request.Owner,
				request.OwnerGeneration,
			)
			if err != nil {
				return err
			}
			callCtx, cancel := context.WithTimeout(effectCtx, executor.timeout)
			if conditionalDeleter != nil {
				err = conditionalDeleter.DeleteRecordWithSwap(
					callCtx,
					request.Owner,
					request.Collection.String(),
					request.Rkey.String(),
					request.ExpectedCID,
				)
			} else {
				err = client.DeleteRecord(
					callCtx,
					request.Owner,
					request.Collection.String(),
					request.Rkey.String(),
				)
			}
			cancel()
			if errors.Is(err, auth.ErrRecordSwapConflict) {
				_, completeErr := executor.attempts.CompleteEffectAttempt(
					effectCtx,
					attempt.OperationID,
					request.Owner,
					request.OwnerGeneration,
					ownerlifecycle.OutcomeRejected,
					"",
				)
				conflictErr := &ConflictError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}
				return errors.Join(conflictErr, completeErr)
			}
			if err != nil && !errors.Is(err, auth.ErrRecordNotFound) {
				return &OutcomeAmbiguousError{
					OperationID: request.OperationID,
					ExactKey:    uri.String(),
					Cause:       err,
				}
			}
			if _, err := executor.attempts.CompleteEffectAttempt(
				effectCtx,
				attempt.OperationID,
				request.Owner,
				request.OwnerGeneration,
				ownerlifecycle.OutcomeAccepted,
				"",
			); err != nil {
				return err
			}
			result = RecordResult{URI: uri}
			return nil
		},
	)
	return result, err
}

func validateEffectOwner(
	executorOwner syntax.DID,
	operationID string,
	mutationKey string,
	owner syntax.DID,
	generation int64,
	expected []ownerlifecycle.ExpectedOwner,
) error {
	if executorOwner == "" || owner != executorOwner {
		return ErrExecutorOwnerMismatch
	}
	if !validMutationIdentity(operationID) || !validMutationIdentity(mutationKey) ||
		owner == "" || generation <= 0 || len(expected) == 0 {
		return errors.New("durable PDS effect identity is incomplete")
	}
	found := false
	for _, item := range expected {
		if item.Owner == owner {
			if item.Generation != generation {
				return ownerlifecycle.ErrGenerationChanged
			}
			found = true
		}
	}
	if !found {
		return ownerlifecycle.ErrFenceRequired
	}
	return nil
}

func validMutationIdentity(value string) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= 512
}

var _ EffectExecutor = (*Executor)(nil)
