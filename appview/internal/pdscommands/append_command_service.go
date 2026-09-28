package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type AppendCommandServiceConfig struct {
	Store        *Store
	Lifecycles   *ownerlifecycle.Store
	NewBoundary  ActivePDSBoundaryFactory
	NewRecordKey func() (syntax.RecordKey, error)
	Now          func() time.Time
	Sleep        func(time.Duration)
	Jitter       func(int) int
}

type AppendCommandService struct {
	store        *Store
	lifecycles   *ownerlifecycle.Store
	newBoundary  ActivePDSBoundaryFactory
	newRecordKey func() (syntax.RecordKey, error)
	now          func() time.Time
	sleep        func(time.Duration)
	jitter       func(int) int
}

type AppendCommandRequest struct {
	Owner           syntax.DID
	OwnerGeneration int64
	Targets         []syntax.DID
	SessionID       string
	OperationKind   string
	OperationKey    uuid.UUID
	Collection      syntax.NSID
	Intent          json.RawMessage
	Blobs           []BlobReference
	BuildRecord     func(time.Time) (json.RawMessage, error)
	Accepted        func(AuthoritativeRecord) (TerminalResult, error)
	Rejected        func(error) TerminalResult
}

func NewAppendCommandService(config AppendCommandServiceConfig) (*AppendCommandService, error) {
	if config.Store == nil || config.Lifecycles == nil || config.NewBoundary == nil || config.NewRecordKey == nil {
		return nil, errors.New("append command service dependencies are unavailable")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Sleep == nil {
		config.Sleep = time.Sleep
	}
	return &AppendCommandService{
		store: config.Store, lifecycles: config.Lifecycles, newBoundary: config.NewBoundary,
		newRecordKey: config.NewRecordKey, now: config.Now, sleep: config.Sleep, jitter: config.Jitter,
	}, nil
}

func (service *AppendCommandService) Execute(ctx context.Context, request AppendCommandRequest) (CommandResult, error) {
	if err := validateAppendCommandRequest(request); err != nil {
		return CommandResult{}, err
	}
	expected, err := service.resolveExpectedOwners(ctx, request)
	if err != nil {
		return CommandResult{}, err
	}
	fingerprint, err := appendRequestFingerprint(request, expected)
	if err != nil {
		return CommandResult{}, err
	}
	prepared, err := service.store.PrepareOrReplayWithSelection(ctx, PrepareRequest{
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		OperationKind: request.OperationKind, OperationKey: request.OperationKey,
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: request.Intent,
		ExpectedOwner: request.Owner, ExpectedOwnerGeneration: request.OwnerGeneration,
	}, func() (syntax.ATURI, syntax.RecordKey, error) {
		rkey, err := service.newRecordKey()
		if err != nil {
			return "", "", fmt.Errorf("allocate append record key: %w", err)
		}
		if _, err := syntax.ParseRecordKey(rkey.String()); err != nil {
			return "", "", ErrMalformedCommand
		}
		uri := syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + rkey.String())
		return uri, rkey, nil
	})
	if err != nil {
		return CommandResult{}, err
	}
	if prepared.State == CommandAccepted || prepared.State == CommandRejected {
		return service.store.Result(ctx, prepared.ID)
	}
	record, err := request.BuildRecord(prepared.CreatedAt.UTC())
	if err != nil || !json.Valid(record) {
		return service.reject(ctx, prepared.ID, request, errors.Join(ErrMalformedCommand, err))
	}
	boundary, err := service.newBoundary(ctx, request.Owner, request.SessionID)
	if err != nil {
		return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
	}
	var result CommandResult
	err = boundary.WithActiveEffects(ctx, expected, func(operationCtx context.Context, client auth.PDSClient) error {
		protocol, ok := client.(PDSProtocol)
		if !ok {
			return ErrDispatchUnavailable
		}
		transport, err := NewPDSTransport(protocol)
		if err != nil {
			return err
		}
		result, err = service.executeFenced(operationCtx, transport, prepared, fingerprint, record, request)
		return err
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func (service *AppendCommandService) executeAlreadyFenced(
	ctx context.Context,
	protocol PDSProtocol,
	expected []ownerlifecycle.ExpectedOwner,
	request FencedAppendCommandRequest,
) (CommandResult, error) {
	if err := validateAppendCommandRequest(request.Command); err != nil ||
		request.SelectedURI == "" || request.SelectedRkey == "" {
		return CommandResult{}, ErrMalformedCommand
	}
	if err := validateSelectedAppendIdentity(request); err != nil {
		return CommandResult{}, err
	}
	if !appendRequestMatchesFence(request.Command, expected) {
		return CommandResult{}, ownerlifecycle.ErrFenceRequired
	}
	fingerprint, err := appendRequestFingerprint(request.Command, expected)
	if err != nil {
		return CommandResult{}, err
	}
	prepared, err := service.store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: request.Command.Owner, OwnerGeneration: request.Command.OwnerGeneration,
		OperationKind: request.Command.OperationKind, OperationKey: request.Command.OperationKey,
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: request.Command.Intent,
		SelectedURI: request.SelectedURI, SelectedRkey: request.SelectedRkey,
		ExpectedOwner: request.Command.Owner, ExpectedOwnerGeneration: request.Command.OwnerGeneration,
	})
	if err != nil {
		return CommandResult{}, err
	}
	if prepared.SelectedURI != request.SelectedURI || prepared.SelectedRkey != request.SelectedRkey {
		return CommandResult{}, ErrIdempotencyConflict
	}
	if prepared.State == CommandAccepted || prepared.State == CommandRejected {
		return service.store.Result(ctx, prepared.ID)
	}
	record, err := request.Command.BuildRecord(prepared.CreatedAt.UTC())
	if err != nil || !json.Valid(record) {
		return service.reject(ctx, prepared.ID, request.Command, errors.Join(ErrMalformedCommand, err))
	}
	transport, err := NewPDSTransport(protocol)
	if err != nil {
		return CommandResult{}, err
	}
	return service.executeFenced(
		ctx,
		transport,
		prepared,
		fingerprint,
		record,
		request.Command,
	)
}

func appendRequestFingerprint(
	request AppendCommandRequest,
	expected []ownerlifecycle.ExpectedOwner,
) ([32]byte, error) {
	conditions, err := json.Marshal(struct {
		Owners []ownerlifecycle.ExpectedOwner `json:"owners"`
	}{Owners: expected})
	if err != nil {
		return [32]byte{}, err
	}
	return RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            request.Owner,
		OperationKind:    request.OperationKind,
		Intent:           request.Intent,
		Conditions:       conditions,
		Blobs:            request.Blobs,
	})
}

func appendRequestMatchesFence(
	request AppendCommandRequest,
	expected []ownerlifecycle.ExpectedOwner,
) bool {
	if len(request.Targets) != 0 || len(expected) != 1 {
		return false
	}
	return expected[0].Owner == request.Owner &&
		expected[0].Generation == request.OwnerGeneration &&
		!expected[0].AllowMissing
}

func validateAppendCommandRequest(request AppendCommandRequest) error {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.SessionID == "" || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || request.Collection == "" || !json.Valid(request.Intent) ||
		request.BuildRecord == nil || request.Accepted == nil || request.Rejected == nil {
		return ErrMalformedCommand
	}
	for _, blob := range request.Blobs {
		if blob.CID == "" || blob.MIMEType == "" || blob.Size < 0 {
			return ErrMalformedCommand
		}
	}
	return nil
}

func (service *AppendCommandService) executeFenced(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	record json.RawMessage,
	request AppendCommandRequest,
) (CommandResult, error) {
	var firstAttempt int
	var err error
	command, firstAttempt, err = service.store.ResumeKnownInvalidSwap(ctx, command)
	if err != nil {
		return CommandResult{}, err
	}
	if firstAttempt > 3 {
		return service.reject(ctx, command.ID, request, ErrRepositoryConflict)
	}
	for attemptNumber := firstAttempt; attemptNumber <= 3; attemptNumber++ {
		authoritative, err := transport.GetRecord(ctx, request.Owner, command.SelectedURI)
		switch {
		case err == nil:
			equal, compareErr := equalCanonicalJSON(authoritative.Record, record)
			if compareErr != nil || !equal {
				if command.State != CommandPrepared {
					return service.store.UnresolvedResult(ctx, command)
				}
				return service.reject(ctx, command.ID, request, ErrIdempotencyConflict)
			}
			return service.acceptRecord(ctx, command.ID, request, authoritative)
		case !errors.Is(err, auth.ErrRecordNotFound):
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		if command.State != CommandPrepared {
			return service.store.UnresolvedResult(ctx, command)
		}
		head, err := transport.LatestCommit(ctx, request.Owner)
		if err != nil {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		step := DispatchStep{Action: "create", URI: command.SelectedURI, Body: record}
		generated, err := json.Marshal(struct {
			Rkey      syntax.RecordKey `json:"rkey"`
			CreatedAt time.Time        `json:"createdAt"`
		}{Rkey: command.SelectedRkey, CreatedAt: command.CreatedAt.UTC()})
		if err != nil {
			return CommandResult{}, err
		}
		dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
			AlgorithmVersion: 1, RequestFingerprint: requestFingerprint,
			RepositoryHead: head, Generated: generated, Steps: []DispatchStep{step},
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
				return service.rejectDispatch(ctx, attempt.ID, command.ID, request, "invalid_swap", ErrRepositoryConflict)
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
				return service.rejectDispatch(ctx, attempt.ID, command.ID, request, errorClass, err)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "transport"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		authoritative, err = transport.GetRecord(ctx, request.Owner, command.SelectedURI)
		if err != nil {
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "result_read"}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		equal, compareErr := equalCanonicalJSON(authoritative.Record, record)
		if compareErr != nil || !equal {
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
	return service.reject(ctx, command.ID, request, ErrRepositoryConflict)
}

func validateSelectedAppendIdentity(request FencedAppendCommandRequest) error {
	parsed, err := syntax.ParseATURI(request.SelectedURI.String())
	if err != nil || parsed.Authority().DID() != request.Command.Owner ||
		parsed.Collection() != request.Command.Collection ||
		parsed.RecordKey() != request.SelectedRkey {
		return ErrMalformedCommand
	}
	return nil
}

func (service *AppendCommandService) resolveExpectedOwners(
	ctx context.Context,
	request AppendCommandRequest,
) ([]ownerlifecycle.ExpectedOwner, error) {
	owners := append([]syntax.DID{request.Owner}, request.Targets...)
	canonical, err := ownerlifecycle.CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	expected := make([]ownerlifecycle.ExpectedOwner, 0, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := service.lifecycles.Get(ctx, owner)
		if errors.Is(err, pgx.ErrNoRows) && owner != request.Owner {
			expected = append(expected, ownerlifecycle.ExpectedOwner{Owner: owner, AllowMissing: true})
			continue
		}
		if err != nil {
			return nil, err
		}
		if lifecycle.State == ownerlifecycle.StateTerminal {
			return nil, ownerlifecycle.ErrTerminalOwner
		}
		if lifecycle.State != ownerlifecycle.StateActive {
			return nil, ownerlifecycle.ErrOwnerNotActive
		}
		if owner == request.Owner && lifecycle.Generation != request.OwnerGeneration {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		expected = append(expected, ownerlifecycle.ExpectedOwner{Owner: owner, Generation: lifecycle.Generation})
	}
	return expected, nil
}

func (service *AppendCommandService) acceptRecord(
	ctx context.Context,
	commandID uuid.UUID,
	request AppendCommandRequest,
	record AuthoritativeRecord,
) (CommandResult, error) {
	terminal, err := request.Accepted(record)
	if err != nil {
		return CommandResult{}, err
	}
	return service.accept(ctx, commandID, terminal)
}

func (service *AppendCommandService) accept(ctx context.Context, commandID uuid.UUID, terminal TerminalResult) (CommandResult, error) {
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *AppendCommandService) reject(
	ctx context.Context,
	commandID uuid.UUID,
	request AppendCommandRequest,
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

func (service *AppendCommandService) rejectDispatch(
	ctx context.Context,
	attemptID uuid.UUID,
	commandID uuid.UUID,
	request AppendCommandRequest,
	errorClass string,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteDispatchAndCommand(
		ctx,
		attemptID,
		DispatchCompletion{Outcome: CommandRejected, ErrorClass: errorClass},
		terminal,
	); err != nil {
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}
