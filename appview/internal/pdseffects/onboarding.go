package pdseffects

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

const (
	onboardingProfileCollection syntax.NSID      = "social.craftsky.actor.profile"
	onboardingProfileRkey       syntax.RecordKey = "self"
)

type OnboardingExecutor struct {
	attempts *ownerlifecycle.Store
	timeout  time.Duration
	now      func() time.Time
}

func NewOnboardingExecutor(
	attempts *ownerlifecycle.Store,
	timeout time.Duration,
	now func() time.Time,
) (*OnboardingExecutor, error) {
	if attempts == nil || timeout <= 0 {
		return nil, errors.New("onboarding PDS effect executor dependencies and timeout are required")
	}
	if now == nil {
		now = time.Now
	}
	return &OnboardingExecutor{attempts: attempts, timeout: timeout, now: now}, nil
}

type OnboardingProfileRequest struct {
	// OperationID and MutationKey have the same contract as PutRecordRequest.
	OperationID     string
	MutationKey     string
	Owner           syntax.DID
	OwnerGeneration int64
	Record          any
	ExpectedCID     syntax.CID
}

// PutProfile is the single non-active ordinary-write exception. The caller
// must already be inside WithOnboardingAuth (or its WithExistingAuth retry
// counterpart) for the exact departed generation;
// WithOnboardingEffect converts that existing exclusive owner fence into a
// durable effect scope without acquiring another lock. Collection and rkey
// are hardcoded so this cannot become a general departed-owner writer.
func (executor *OnboardingExecutor) PutProfile(
	ctx context.Context,
	client auth.PDSClient,
	request OnboardingProfileRequest,
) (RecordResult, error) {
	if executor == nil || executor.attempts == nil || client == nil ||
		!validMutationIdentity(request.OperationID) || request.Owner == "" ||
		!validMutationIdentity(request.MutationKey) || request.OwnerGeneration <= 0 || request.Record == nil {
		return RecordResult{}, errors.New("onboarding profile effect identity is incomplete")
	}
	_, fingerprint, err := canonicalPutBody(
		request.Owner,
		onboardingProfileCollection,
		onboardingProfileRkey,
		request.Record,
		request.ExpectedCID,
	)
	if err != nil {
		return RecordResult{}, err
	}
	_, recordFingerprint, err := canonicalPutBody(
		request.Owner,
		onboardingProfileCollection,
		onboardingProfileRkey,
		request.Record,
		"",
	)
	if err != nil {
		return RecordResult{}, err
	}
	uri, err := deterministicRecordURI(
		request.Owner,
		onboardingProfileCollection,
		onboardingProfileRkey,
	)
	if err != nil {
		return RecordResult{}, err
	}
	putRequest := PutRecordRequest{
		OperationID:     request.OperationID,
		MutationKey:     request.MutationKey,
		Owner:           request.Owner,
		OwnerGeneration: request.OwnerGeneration,
		Collection:      onboardingProfileCollection,
		Rkey:            onboardingProfileRkey,
		Record:          request.Record,
		ExpectedCID:     request.ExpectedCID,
	}
	var result RecordResult
	err = executor.attempts.WithOnboardingEffect(
		ctx,
		request.Owner,
		request.OwnerGeneration,
		func(effectCtx context.Context) error {
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
						Cause:       errors.New("accepted onboarding profile Put has no authoritative CID"),
					}
				}
				result = RecordResult{URI: uri, CID: syntax.CID(attempt.ResultCID)}
				return nil
			case ownerlifecycle.OutcomePrepared:
				// Continue to the one permitted profile Put below.
			case ownerlifecycle.OutcomeDispatched:
				activeExecutor := &Executor{
					attempts: executor.attempts,
					owner:    request.Owner,
					timeout:  executor.timeout,
				}
				var reconcileErr error
				result, reconcileErr = activeExecutor.reconcilePutInsideActive(
					effectCtx,
					client,
					putRequest,
					attempt,
					uri,
					recordFingerprint,
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
					onboardingProfileCollection.String(),
					onboardingProfileRkey.String(),
					request.Record,
					request.ExpectedCID,
				)
			} else {
				err = client.PutRecord(
					callCtx,
					request.Owner,
					onboardingProfileCollection.String(),
					onboardingProfileRkey.String(),
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
			activeExecutor := &Executor{
				attempts: executor.attempts,
				owner:    request.Owner,
				timeout:  executor.timeout,
			}
			result, err = activeExecutor.reconcilePutInsideActive(
				effectCtx, client, putRequest, attempt, uri, recordFingerprint,
			)
			return err
		},
	)
	return result, err
}
