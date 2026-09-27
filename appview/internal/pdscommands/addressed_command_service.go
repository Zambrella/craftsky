package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrRecordConflict = errors.New("PDS record changed")

type AddressedCommandServiceConfig struct {
	Store       *Store
	Lifecycles  *ownerlifecycle.Store
	NewBoundary ActivePDSBoundaryFactory
	Now         func() time.Time
	Sleep       func(time.Duration)
	Jitter      func(int) int
}

type AddressedCommandService struct {
	store       *Store
	lifecycles  *ownerlifecycle.Store
	newBoundary ActivePDSBoundaryFactory
	now         func() time.Time
	sleep       func(time.Duration)
	jitter      func(int) int
}

type AddressedDeleteCommandRequest struct {
	Owner           syntax.DID
	OwnerGeneration int64
	SessionID       string
	OperationKind   string
	OperationKey    uuid.UUID
	URI             syntax.ATURI
	ExpectedCID     syntax.CID
	Intent          json.RawMessage
	AcceptedAbsent  func() TerminalResult
	Rejected        func(error) TerminalResult
}

type AddressedPutCommandRequest struct {
	Owner           syntax.DID
	OwnerGeneration int64
	SessionID       string
	OperationKind   string
	OperationKey    uuid.UUID
	URI             syntax.ATURI
	ExpectedCID     syntax.CID
	Intent          json.RawMessage
	Blobs           []BlobReference
	BuildRecord     func(AuthoritativeRecord) (json.RawMessage, error)
	Accepted        func(AuthoritativeRecord) (TerminalResult, error)
	Rejected        func(error) TerminalResult
}

func NewAddressedCommandService(config AddressedCommandServiceConfig) (*AddressedCommandService, error) {
	if config.Store == nil || config.Lifecycles == nil || config.NewBoundary == nil {
		return nil, errors.New("addressed command service dependencies are unavailable")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Sleep == nil {
		config.Sleep = time.Sleep
	}
	return &AddressedCommandService{
		store: config.Store, lifecycles: config.Lifecycles, newBoundary: config.NewBoundary,
		now: config.Now, sleep: config.Sleep, jitter: config.Jitter,
	}, nil
}

func (service *AddressedCommandService) Delete(ctx context.Context, request AddressedDeleteCommandRequest) (CommandResult, error) {
	rkey, err := validateAddressedDeleteRequest(request)
	if err != nil {
		return CommandResult{}, err
	}
	conditions, err := json.Marshal(struct {
		OwnerGeneration int64      `json:"ownerGeneration"`
		ExpectedCID     syntax.CID `json:"expectedCid"`
	}{OwnerGeneration: request.OwnerGeneration, ExpectedCID: request.ExpectedCID})
	if err != nil {
		return CommandResult{}, err
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: request.Owner, OperationKind: request.OperationKind,
		Intent: request.Intent, Conditions: conditions,
	})
	if err != nil {
		return CommandResult{}, err
	}
	prepared, err := service.store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		OperationKind: request.OperationKind, OperationKey: request.OperationKey,
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: request.Intent,
		SelectedURI: request.URI, SelectedRkey: rkey,
		ExpectedOwner: request.Owner, ExpectedOwnerGeneration: request.OwnerGeneration,
	})
	if err != nil {
		return CommandResult{}, err
	}
	if prepared.State == CommandAccepted || prepared.State == CommandRejected {
		return service.store.Result(ctx, prepared.ID)
	}
	lifecycle, err := service.lifecycles.Get(ctx, request.Owner)
	if err != nil {
		return CommandResult{}, err
	}
	if lifecycle.State == ownerlifecycle.StateTerminal {
		return service.reject(ctx, prepared.ID, request, ownerlifecycle.ErrTerminalOwner)
	}
	if lifecycle.State != ownerlifecycle.StateActive {
		return service.reject(ctx, prepared.ID, request, ownerlifecycle.ErrOwnerNotActive)
	}
	if lifecycle.Generation != request.OwnerGeneration {
		return service.reject(ctx, prepared.ID, request, ownerlifecycle.ErrGenerationChanged)
	}
	boundary, err := service.newBoundary(ctx, request.Owner, request.SessionID)
	if err != nil {
		return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
	}
	var result CommandResult
	err = boundary.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{Owner: request.Owner, Generation: request.OwnerGeneration}}, func(operationCtx context.Context, client auth.PDSClient) error {
		protocol, ok := client.(PDSProtocol)
		if !ok {
			return ErrDispatchUnavailable
		}
		transport, err := NewPDSTransport(protocol)
		if err != nil {
			return err
		}
		result, err = service.deleteFenced(operationCtx, transport, prepared, fingerprint, request)
		return err
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func (service *AddressedCommandService) Put(ctx context.Context, request AddressedPutCommandRequest) (CommandResult, error) {
	rkey, err := validateAddressedPutRequest(request)
	if err != nil {
		return CommandResult{}, err
	}
	conditions, err := json.Marshal(struct {
		OwnerGeneration int64      `json:"ownerGeneration"`
		ExpectedCID     syntax.CID `json:"expectedCid"`
	}{OwnerGeneration: request.OwnerGeneration, ExpectedCID: request.ExpectedCID})
	if err != nil {
		return CommandResult{}, err
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: request.Owner, OperationKind: request.OperationKind,
		Intent: request.Intent, Conditions: conditions, Blobs: request.Blobs,
	})
	if err != nil {
		return CommandResult{}, err
	}
	prepared, err := service.store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		OperationKind: request.OperationKind, OperationKey: request.OperationKey,
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: request.Intent,
		SelectedURI: request.URI, SelectedRkey: rkey,
		ExpectedOwner: request.Owner, ExpectedOwnerGeneration: request.OwnerGeneration,
	})
	if err != nil {
		return CommandResult{}, err
	}
	if prepared.State == CommandAccepted || prepared.State == CommandRejected {
		return service.store.Result(ctx, prepared.ID)
	}
	lifecycle, err := service.lifecycles.Get(ctx, request.Owner)
	if err != nil {
		return CommandResult{}, err
	}
	if lifecycle.State == ownerlifecycle.StateTerminal {
		return service.rejectPut(ctx, prepared.ID, request, ownerlifecycle.ErrTerminalOwner)
	}
	if lifecycle.State != ownerlifecycle.StateActive {
		return service.rejectPut(ctx, prepared.ID, request, ownerlifecycle.ErrOwnerNotActive)
	}
	if lifecycle.Generation != request.OwnerGeneration {
		return service.rejectPut(ctx, prepared.ID, request, ownerlifecycle.ErrGenerationChanged)
	}
	boundary, err := service.newBoundary(ctx, request.Owner, request.SessionID)
	if err != nil {
		return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
	}
	var result CommandResult
	err = boundary.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{Owner: request.Owner, Generation: request.OwnerGeneration}}, func(operationCtx context.Context, client auth.PDSClient) error {
		protocol, ok := client.(PDSProtocol)
		if !ok {
			return ErrDispatchUnavailable
		}
		transport, err := NewPDSTransport(protocol)
		if err != nil {
			return err
		}
		result, err = service.putFenced(operationCtx, transport, prepared, fingerprint, request)
		return err
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func validateAddressedPutRequest(request AddressedPutCommandRequest) (syntax.RecordKey, error) {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.SessionID == "" || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || request.URI == "" || request.ExpectedCID == "" ||
		!json.Valid(request.Intent) || request.BuildRecord == nil || request.Accepted == nil || request.Rejected == nil {
		return "", ErrMalformedCommand
	}
	for _, blob := range request.Blobs {
		if blob.CID == "" || blob.MIMEType == "" || blob.Size < 0 {
			return "", ErrMalformedCommand
		}
	}
	parsed, err := syntax.ParseATURI(request.URI.String())
	if err != nil || parsed.Authority().DID() != request.Owner || parsed.Collection() == "" || parsed.RecordKey() == "" {
		return "", ErrMalformedCommand
	}
	return parsed.RecordKey(), nil
}

func validateAddressedDeleteRequest(request AddressedDeleteCommandRequest) (syntax.RecordKey, error) {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.SessionID == "" || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || request.URI == "" || request.ExpectedCID == "" ||
		!json.Valid(request.Intent) || request.AcceptedAbsent == nil || request.Rejected == nil {
		return "", ErrMalformedCommand
	}
	parsed, err := syntax.ParseATURI(request.URI.String())
	if err != nil || parsed.Authority().DID() != request.Owner || parsed.Collection() == "" || parsed.RecordKey() == "" {
		return "", ErrMalformedCommand
	}
	return parsed.RecordKey(), nil
}

func (service *AddressedCommandService) deleteFenced(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	request AddressedDeleteCommandRequest,
) (CommandResult, error) {
	for attemptNumber := 1; attemptNumber <= 3; attemptNumber++ {
		record, err := transport.GetRecord(ctx, request.Owner, request.URI)
		if errors.Is(err, auth.ErrRecordNotFound) {
			return service.accept(ctx, command.ID, request.AcceptedAbsent())
		}
		if err != nil {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		if record.CID != request.ExpectedCID {
			return service.reject(ctx, command.ID, request, ErrRecordConflict)
		}
		if command.State != CommandPrepared {
			if command.State == CommandDispatching {
				if err := service.store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
					return CommandResult{}, err
				}
			}
			return service.store.Result(ctx, command.ID)
		}
		head, err := transport.LatestCommit(ctx, request.Owner)
		if err != nil {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		step := DispatchStep{Action: "delete", URI: request.URI, ExpectedCID: request.ExpectedCID}
		dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
			AlgorithmVersion: 1, RequestFingerprint: requestFingerprint, RepositoryHead: head, Steps: []DispatchStep{step},
		})
		if err != nil {
			return CommandResult{}, err
		}
		attempt, err := service.store.BeginDispatch(ctx, command.ID, ExactPlan{
			Version: command.ActivePlanVersion + attemptNumber, FingerprintVersion: 1,
			Fingerprint: dispatchFingerprint, RepositoryCID: head,
			RemoteDeadline: service.now().UTC().Add(30 * time.Second), Steps: []DispatchStep{step},
		})
		if err != nil {
			if errors.Is(err, ErrIllegalTransition) {
				return service.store.Result(ctx, command.ID)
			}
			return CommandResult{}, err
		}
		err = transport.ApplyWrites(ctx, request.Owner, head, []DispatchStep{step})
		if errors.Is(err, ErrAtomicWritesUnsupported) {
			err = transport.DeleteRecordWithRepositorySwap(ctx, request.Owner, request.URI, head, request.ExpectedCID)
		}
		if errors.Is(err, ErrRepositorySwapConflict) {
			delay, retry := InvalidSwapRetryDelay(attemptNumber, service.jitter)
			if !retry {
				return service.rejectDeleteDispatch(ctx, attempt.ID, command.ID, request, "invalid_swap", ErrRepositoryConflict)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: DispatchInvalidSwap, ErrorClass: "invalid_swap"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			service.sleep(delay)
			command.ActivePlanVersion++
			continue
		}
		if err != nil {
			if errors.Is(err, ErrDispatchRejected) {
				return service.rejectDeleteDispatch(ctx, attempt.ID, command.ID, request, "rejected", err)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "transport"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		terminal := request.AcceptedAbsent()
		if err := service.store.CompleteDispatchAndCommand(
			ctx,
			attempt.ID,
			DispatchCompletion{Outcome: CommandAccepted},
			terminal,
		); err != nil {
			return CommandResult{}, err
		}
		return service.store.Result(ctx, command.ID)
	}
	return service.reject(ctx, command.ID, request, ErrRepositoryConflict)
}

func (service *AddressedCommandService) putFenced(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	request AddressedPutCommandRequest,
) (CommandResult, error) {
	for attemptNumber := 1; attemptNumber <= 3; attemptNumber++ {
		record, err := transport.GetRecord(ctx, request.Owner, request.URI)
		absent := errors.Is(err, auth.ErrRecordNotFound)
		if absent && request.ExpectedCID != "*" {
			return service.rejectPut(ctx, command.ID, request, ErrRecordConflict)
		}
		if err != nil && !absent {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		if absent {
			record = AuthoritativeRecord{URI: request.URI}
		}
		desired, buildErr := request.BuildRecord(record)
		if buildErr != nil || !json.Valid(desired) {
			return service.rejectPut(ctx, command.ID, request, errors.Join(ErrMalformedCommand, buildErr))
		}
		if !absent {
			matches, compareErr := equalCanonicalJSON(record.Record, desired)
			if compareErr != nil {
				return CommandResult{}, compareErr
			}
			if matches {
				return service.acceptRecord(ctx, command.ID, request, record)
			}
			if request.ExpectedCID == "*" || record.CID != request.ExpectedCID {
				return service.rejectPut(ctx, command.ID, request, ErrRecordConflict)
			}
		}
		if command.State != CommandPrepared {
			if command.State == CommandDispatching {
				if err := service.store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
					return CommandResult{}, err
				}
			}
			return service.store.Result(ctx, command.ID)
		}
		head, err := transport.LatestCommit(ctx, request.Owner)
		if err != nil {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		action := "update"
		expectedCID := request.ExpectedCID
		if request.ExpectedCID == "*" {
			action = "create"
			expectedCID = ""
		}
		step := DispatchStep{Action: action, URI: request.URI, Body: desired, ExpectedCID: expectedCID}
		dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
			AlgorithmVersion: 1, RequestFingerprint: requestFingerprint, RepositoryHead: head, Steps: []DispatchStep{step},
		})
		if err != nil {
			return CommandResult{}, err
		}
		attempt, err := service.store.BeginDispatch(ctx, command.ID, ExactPlan{
			Version: command.ActivePlanVersion + attemptNumber, FingerprintVersion: 1,
			Fingerprint: dispatchFingerprint, RepositoryCID: head,
			RemoteDeadline: service.now().UTC().Add(30 * time.Second), Steps: []DispatchStep{step},
		})
		if err != nil {
			if errors.Is(err, ErrIllegalTransition) {
				return service.store.Result(ctx, command.ID)
			}
			return CommandResult{}, err
		}
		err = transport.ApplyWrites(ctx, request.Owner, head, []DispatchStep{step})
		if errors.Is(err, ErrRepositorySwapConflict) {
			delay, retry := InvalidSwapRetryDelay(attemptNumber, service.jitter)
			if !retry {
				return service.rejectPutDispatch(ctx, attempt.ID, command.ID, request, "invalid_swap", ErrRepositoryConflict)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: DispatchInvalidSwap, ErrorClass: "invalid_swap"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			service.sleep(delay)
			command.ActivePlanVersion++
			continue
		}
		if err != nil {
			if errors.Is(err, ErrAtomicWritesUnsupported) || errors.Is(err, ErrDispatchRejected) {
				errorClass := "rejected"
				if errors.Is(err, ErrAtomicWritesUnsupported) {
					errorClass = "unsupported"
				}
				return service.rejectPutDispatch(ctx, attempt.ID, command.ID, request, errorClass, err)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "transport"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		authoritative, err := transport.GetRecord(ctx, request.Owner, request.URI)
		if err != nil {
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "result_read"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		matches, compareErr := equalCanonicalJSON(authoritative.Record, desired)
		if compareErr != nil || !matches {
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "result_mismatch"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		terminal, err := request.Accepted(authoritative)
		if err != nil {
			return CommandResult{}, err
		}
		if err := service.store.CompleteDispatchAndCommand(
			ctx,
			attempt.ID,
			DispatchCompletion{Outcome: CommandAccepted, ResultRecords: authoritative.Record},
			terminal,
		); err != nil {
			return CommandResult{}, err
		}
		return service.store.Result(ctx, command.ID)
	}
	return service.rejectPut(ctx, command.ID, request, ErrRepositoryConflict)
}

func (service *AddressedCommandService) acceptRecord(
	ctx context.Context,
	commandID uuid.UUID,
	request AddressedPutCommandRequest,
	record AuthoritativeRecord,
) (CommandResult, error) {
	terminal, err := request.Accepted(record)
	if err != nil {
		return CommandResult{}, err
	}
	return service.accept(ctx, commandID, terminal)
}

func (service *AddressedCommandService) accept(ctx context.Context, commandID uuid.UUID, terminal TerminalResult) (CommandResult, error) {
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *AddressedCommandService) reject(
	ctx context.Context,
	commandID uuid.UUID,
	request AddressedDeleteCommandRequest,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *AddressedCommandService) rejectPut(
	ctx context.Context,
	commandID uuid.UUID,
	request AddressedPutCommandRequest,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *AddressedCommandService) rejectDeleteDispatch(
	ctx context.Context,
	attemptID uuid.UUID,
	commandID uuid.UUID,
	request AddressedDeleteCommandRequest,
	errorClass string,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteDispatchAndCommand(ctx, attemptID, DispatchCompletion{Outcome: CommandRejected, ErrorClass: errorClass}, terminal); err != nil {
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *AddressedCommandService) rejectPutDispatch(
	ctx context.Context,
	attemptID uuid.UUID,
	commandID uuid.UUID,
	request AddressedPutCommandRequest,
	errorClass string,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteDispatchAndCommand(ctx, attemptID, DispatchCompletion{Outcome: CommandRejected, ErrorClass: errorClass}, terminal); err != nil {
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}
