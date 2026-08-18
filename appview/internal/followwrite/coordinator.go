package followwrite

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var (
	ErrOutcomeUncertain = errors.New("follow outcome is uncertain")
	ErrEffectRejected   = errors.New("follow effect was rejected")
)

type DeterministicRequest struct {
	OperationID     string
	Owner           syntax.DID
	Target          syntax.DID
	OwnerGeneration int64
	SessionID       string
	Rkey            syntax.RecordKey
	CreatedAt       time.Time
}

type DeterministicResult struct {
	RecordURI syntax.ATURI
	RecordCID string
}

// Coordinator is the ordinary deterministic follow effect boundary. Its
// caller must already hold the active owner/participant fences through
// ownerlifecycle.Store.WithActiveEffects. It persists the no-repeat attempt
// before PutRecord and reconciles a dispatched attempt with a read; it never
// treats an ambiguous response as permission to repeat the write.
type Coordinator struct {
	lifecycles *ownerlifecycle.Store
	writer     *Service
	now        func() time.Time
	timeout    time.Duration
}

func NewCoordinator(
	lifecycles *ownerlifecycle.Store,
	writer *Service,
	timeout time.Duration,
	now func() time.Time,
) (*Coordinator, error) {
	if lifecycles == nil || writer == nil || timeout <= 0 {
		return nil, errors.New("follow coordinator dependencies and timeout are required")
	}
	if now == nil {
		now = time.Now
	}
	return &Coordinator{lifecycles: lifecycles, writer: writer, timeout: timeout, now: now}, nil
}

func (coordinator *Coordinator) ExecuteDeterministic(
	ctx context.Context,
	request DeterministicRequest,
) (DeterministicResult, error) {
	if coordinator == nil || request.OperationID == "" || request.Owner == "" ||
		request.Target == "" || request.Owner == request.Target || request.OwnerGeneration <= 0 ||
		request.SessionID == "" || request.Rkey == "" || request.CreatedAt.IsZero() {
		return DeterministicResult{}, errors.New("invalid deterministic follow request")
	}
	recordURI := syntax.ATURI(fmt.Sprintf(
		"at://%s/%s/%s",
		request.Owner,
		Collection,
		request.Rkey,
	))
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%s",
		request.Owner,
		request.Target,
		request.Rkey,
		request.CreatedAt.UTC().Format(time.RFC3339Nano),
	)))
	attempt, err := coordinator.lifecycles.CreateEffectAttempt(ctx, ownerlifecycle.NewEffectAttempt{
		OperationID:        request.OperationID,
		Owner:              request.Owner,
		OwnerGeneration:    request.OwnerGeneration,
		Kind:               ownerlifecycle.EffectPDSRecord,
		DeterministicKey:   recordURI.String(),
		RequestFingerprint: fingerprint,
		RemoteDeadline:     coordinator.now().UTC().Add(coordinator.timeout),
	})
	if err != nil {
		return DeterministicResult{}, err
	}

	switch attempt.Outcome {
	case ownerlifecycle.OutcomeAccepted, ownerlifecycle.OutcomeReconciledAccepted:
		return DeterministicResult{RecordURI: recordURI, RecordCID: attempt.ResultCID}, nil
	case ownerlifecycle.OutcomeRejected, ownerlifecycle.OutcomeReconciledNotAccepted,
		ownerlifecycle.OutcomeAbandonedPreTransition, ownerlifecycle.OutcomeReconciliationMismatch:
		return DeterministicResult{}, ErrEffectRejected
	case ownerlifecycle.OutcomeDispatched, ownerlifecycle.OutcomeUnknownPreTransition:
		return coordinator.reconcile(ctx, request, attempt, recordURI)
	case ownerlifecycle.OutcomePrepared:
		// Continue below.
	default:
		return DeterministicResult{}, ownerlifecycle.ErrAttemptState
	}

	attempt, err = coordinator.lifecycles.MarkAttemptDispatched(
		ctx,
		attempt.OperationID,
		request.Owner,
		request.OwnerGeneration,
	)
	if err != nil {
		return DeterministicResult{}, err
	}
	callCtx, cancel := context.WithTimeout(ctx, coordinator.timeout)
	err = coordinator.writer.Write(
		callCtx,
		request.Owner,
		request.Target,
		request.SessionID,
		&request.Rkey,
		request.CreatedAt,
	)
	cancel()
	if err != nil {
		// The request crossed the remote boundary. The durable dispatched
		// state forbids a second PutRecord; a later explicit replay reads the
		// deterministic key to resolve the outcome.
		return DeterministicResult{}, fmt.Errorf("%w: %v", ErrOutcomeUncertain, err)
	}
	attempt, err = coordinator.lifecycles.CompleteEffectAttempt(
		ctx,
		attempt.OperationID,
		request.Owner,
		request.OwnerGeneration,
		ownerlifecycle.OutcomeAccepted,
		"",
	)
	if err != nil {
		return DeterministicResult{}, err
	}
	return DeterministicResult{RecordURI: recordURI, RecordCID: attempt.ResultCID}, nil
}

func (coordinator *Coordinator) reconcile(
	ctx context.Context,
	request DeterministicRequest,
	attempt ownerlifecycle.EffectAttempt,
	recordURI syntax.ATURI,
) (DeterministicResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, coordinator.timeout)
	exists, err := coordinator.writer.HasDeterministicFollow(
		callCtx,
		request.Owner,
		request.Target,
		request.SessionID,
		request.Rkey,
	)
	cancel()
	if err != nil {
		return DeterministicResult{}, fmt.Errorf("%w: %v", ErrOutcomeUncertain, err)
	}
	if !exists {
		return DeterministicResult{}, ErrOutcomeUncertain
	}
	if attempt.Outcome == ownerlifecycle.OutcomeDispatched {
		completed, err := coordinator.lifecycles.CompleteEffectAttempt(
			ctx,
			attempt.OperationID,
			request.Owner,
			request.OwnerGeneration,
			ownerlifecycle.OutcomeAccepted,
			"",
		)
		if err != nil {
			return DeterministicResult{}, err
		}
		return DeterministicResult{RecordURI: recordURI, RecordCID: completed.ResultCID}, nil
	}
	return DeterministicResult{RecordURI: recordURI, RecordCID: attempt.ResultCID}, nil
}
