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

type ActivePDSBoundaryFactory func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error)

type SetCommandServiceConfig struct {
	Store       *Store
	Lifecycles  *ownerlifecycle.Store
	NewBoundary ActivePDSBoundaryFactory
	Now         func() time.Time
	Sleep       func(time.Duration)
	Jitter      func(int) int
}

type SetCommandService struct {
	store       *Store
	lifecycles  *ownerlifecycle.Store
	newBoundary ActivePDSBoundaryFactory
	now         func() time.Time
	sleep       func(time.Duration)
	jitter      func(int) int
}

type SetCommandRequest struct {
	Owner            syntax.DID
	OwnerGeneration  int64
	Target           syntax.DID
	TargetGeneration int64
	SessionID        string
	OperationKind    string
	OperationKey     uuid.UUID
	Collection       syntax.NSID
	DesiredActive    bool
	Intent           json.RawMessage
	SelectedRkey     syntax.RecordKey
	Matches          func(AuthoritativeRecord) bool
	CreateRecord     func(time.Time) (json.RawMessage, error)
	AcceptedPresent  func(AuthoritativeRecord, bool) (TerminalResult, error)
	AcceptedAbsent   func() TerminalResult
	Rejected         func(error) TerminalResult
}

func NewSetCommandService(config SetCommandServiceConfig) (*SetCommandService, error) {
	if config.Store == nil || config.Lifecycles == nil || config.NewBoundary == nil {
		return nil, errors.New("set command service dependencies are unavailable")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Sleep == nil {
		config.Sleep = time.Sleep
	}
	return &SetCommandService{
		store: config.Store, lifecycles: config.Lifecycles, newBoundary: config.NewBoundary,
		now: config.Now, sleep: config.Sleep, jitter: config.Jitter,
	}, nil
}

func (service *SetCommandService) LookupCommand(ctx context.Context, owner syntax.DID, kind string, key uuid.UUID) (*ReplayCommand, error) {
	return service.store.LookupCommand(ctx, owner, kind, key)
}

func (service *SetCommandService) Execute(ctx context.Context, request SetCommandRequest) (CommandResult, error) {
	if err := validateSetCommandRequest(request); err != nil {
		return CommandResult{}, err
	}
	conditions := json.RawMessage(fmt.Sprintf(`{"ownerGeneration":%d}`, request.OwnerGeneration))
	if request.TargetGeneration > 0 {
		conditions = json.RawMessage(fmt.Sprintf(
			`{"ownerGeneration":%d,"targetGeneration":%d}`,
			request.OwnerGeneration, request.TargetGeneration,
		))
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            request.Owner,
		OperationKind:    request.OperationKind,
		Intent:           request.Intent,
		Conditions:       conditions,
	})
	if err != nil {
		return CommandResult{}, err
	}
	selectedURI := syntax.ATURI("")
	if request.DesiredActive {
		selectedURI = syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + request.SelectedRkey.String())
	}
	prepared, err := service.store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		OperationKind: request.OperationKind, OperationKey: request.OperationKey,
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: request.Intent,
		SelectedURI: selectedURI, SelectedRkey: request.SelectedRkey,
		ExpectedOwner: request.Owner, ExpectedOwnerGeneration: request.OwnerGeneration,
		ExpectedTarget: request.Target,
	})
	if err != nil {
		return CommandResult{}, err
	}
	if prepared.State == CommandAccepted || prepared.State == CommandRejected {
		return service.store.Result(ctx, prepared.ID)
	}
	expected, err := service.resolveExpectedOwners(ctx, prepared, request.TargetGeneration)
	if err != nil {
		if isDefinitiveLifecycleError(err) {
			return service.reject(ctx, prepared.ID, request, err)
		}
		return CommandResult{}, err
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
		result, err = service.executeFenced(operationCtx, transport, prepared, fingerprint, request)
		return err
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func isDefinitiveLifecycleError(err error) bool {
	return errors.Is(err, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(err, ownerlifecycle.ErrOwnerNotActive) ||
		errors.Is(err, ownerlifecycle.ErrGenerationChanged)
}

func validateSetCommandRequest(request SetCommandRequest) error {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.SessionID == "" || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || request.Collection == "" || !json.Valid(request.Intent) ||
		request.Matches == nil || request.AcceptedPresent == nil || request.AcceptedAbsent == nil || request.Rejected == nil {
		return ErrMalformedCommand
	}
	if request.DesiredActive && (request.SelectedRkey == "" || request.CreateRecord == nil) {
		return ErrMalformedCommand
	}
	if request.TargetGeneration < 0 || (request.TargetGeneration > 0 && request.Target == "") {
		return ErrMalformedCommand
	}
	return nil
}

func (service *SetCommandService) executeFenced(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	request SetCommandRequest,
) (CommandResult, error) {
	createdRecord := json.RawMessage(nil)
	if request.DesiredActive {
		var err error
		createdRecord, err = request.CreateRecord(command.CreatedAt)
		if err != nil || !json.Valid(createdRecord) {
			return service.reject(ctx, command.ID, request, errors.Join(ErrMalformedCommand, err))
		}
	}
	for attemptNumber := 1; attemptNumber <= 3; attemptNumber++ {
		head, records, err := service.readCollection(ctx, transport, request)
		if err != nil {
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		if request.DesiredActive {
			result, done, err := service.reconcilePresent(ctx, command, request, createdRecord, records)
			if done || err != nil {
				return result, err
			}
		} else {
			steps, err := PlanSetRemoval(records, request.Matches)
			if err != nil {
				return service.reject(ctx, command.ID, request, err)
			}
			if len(steps) == 0 {
				return service.accept(ctx, command.ID, request.AcceptedAbsent())
			}
			if command.State == CommandDispatching {
				if err := service.store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
					return CommandResult{}, err
				}
				return service.store.Result(ctx, command.ID)
			}
			result, retry, err := service.dispatch(ctx, transport, command, requestFingerprint, request, head, steps, attemptNumber)
			if !retry || err != nil {
				return result, err
			}
			continue
		}

		steps := []DispatchStep{{Action: "create", URI: command.SelectedURI, Body: createdRecord}}
		result, retry, err := service.dispatch(ctx, transport, command, requestFingerprint, request, head, steps, attemptNumber)
		if retry {
			continue
		}
		return result, err
	}
	return service.reject(ctx, command.ID, request, ErrRepositoryConflict)
}

func (service *SetCommandService) readCollection(
	ctx context.Context,
	transport *PDSTransport,
	request SetCommandRequest,
) (syntax.CID, []AuthoritativeRecord, error) {
	reader, err := NewAuthoritativeReader(transport)
	if err != nil {
		return "", nil, err
	}
	return reader.CompleteCollection(ctx, request.Owner, request.Collection)
}

func (service *SetCommandService) reconcilePresent(
	ctx context.Context,
	command PreparedCommand,
	request SetCommandRequest,
	createdRecord json.RawMessage,
	records []AuthoritativeRecord,
) (CommandResult, bool, error) {
	for _, record := range records {
		if record.URI != command.SelectedURI {
			continue
		}
		equal, err := equalCanonicalJSON(record.Record, createdRecord)
		if err != nil || !equal || !request.Matches(record) {
			result, rejectErr := service.reject(ctx, command.ID, request, ErrIdempotencyConflict)
			return result, true, rejectErr
		}
		terminal, err := request.AcceptedPresent(record, command.Replay || command.State != CommandPrepared)
		if err != nil {
			return CommandResult{}, true, err
		}
		result, err := service.accept(ctx, command.ID, terminal)
		return result, true, err
	}
	for _, record := range records {
		if request.Matches(record) {
			terminal, err := request.AcceptedPresent(record, false)
			if err != nil {
				return CommandResult{}, true, err
			}
			result, err := service.accept(ctx, command.ID, terminal)
			return result, true, err
		}
	}
	if command.State == CommandDispatching {
		if err := service.store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
			return CommandResult{}, true, err
		}
		result, err := service.store.Result(ctx, command.ID)
		return result, true, err
	}
	return CommandResult{}, false, nil
}

func (service *SetCommandService) dispatch(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	request SetCommandRequest,
	head syntax.CID,
	steps []DispatchStep,
	attemptNumber int,
) (CommandResult, bool, error) {
	planVersion := command.ActivePlanVersion + attemptNumber
	dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
		AlgorithmVersion: 1, RequestFingerprint: requestFingerprint, RepositoryHead: head, Steps: steps,
	})
	if err != nil {
		return CommandResult{}, false, err
	}
	attempt, err := service.store.BeginDispatch(ctx, command.ID, ExactPlan{
		Version: planVersion, FingerprintVersion: 1, Fingerprint: dispatchFingerprint,
		RepositoryCID: head, RemoteDeadline: service.now().UTC().Add(30 * time.Second), Steps: steps,
	})
	if err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			current, resultErr := service.store.Result(ctx, command.ID)
			if resultErr != nil {
				return CommandResult{}, false, resultErr
			}
			if current.State != CommandAccepted && current.State != CommandRejected {
				current.State = CommandAmbiguous
				current.RetryAfterSeconds = 1
			}
			return current, false, nil
		}
		return CommandResult{}, false, err
	}
	err = transport.ApplyWrites(ctx, request.Owner, head, steps)
	if errors.Is(err, ErrAtomicWritesUnsupported) && len(steps) == 1 && steps[0].Action == "delete" {
		err = transport.DeleteRecordWithRepositorySwap(ctx, request.Owner, steps[0].URI, head, steps[0].ExpectedCID)
	}
	if errors.Is(err, ErrRepositorySwapConflict) {
		delay, retry := InvalidSwapRetryDelay(attemptNumber, service.jitter)
		if !retry {
			result, rejectErr := service.rejectDispatch(
				ctx,
				attempt.ID,
				command.ID,
				request,
				DispatchCompletion{Outcome: CommandRejected, ErrorClass: "invalid_swap"},
				ErrRepositoryConflict,
			)
			return result, false, rejectErr
		}
		if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: DispatchInvalidSwap, ErrorClass: "invalid_swap"}); completeErr != nil {
			return CommandResult{}, false, completeErr
		}
		service.sleep(delay)
		return CommandResult{}, true, nil
	}
	if err != nil {
		if errors.Is(err, ErrAtomicWritesUnsupported) || errors.Is(err, ErrDispatchRejected) {
			errorClass := "rejected"
			if errors.Is(err, ErrAtomicWritesUnsupported) {
				errorClass = "unsupported"
			}
			result, rejectErr := service.rejectDispatch(
				ctx,
				attempt.ID,
				command.ID,
				request,
				DispatchCompletion{Outcome: CommandRejected, ErrorClass: errorClass},
				err,
			)
			return result, false, rejectErr
		}
		if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "transport"}); completeErr != nil {
			return CommandResult{}, false, completeErr
		}
		result, resultErr := service.store.Result(ctx, command.ID)
		return result, false, resultErr
	}
	if request.DesiredActive {
		record, readErr := transport.GetRecord(ctx, request.Owner, command.SelectedURI)
		if readErr != nil {
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "result_read"}); completeErr != nil {
				return CommandResult{}, false, completeErr
			}
			result, resultErr := service.store.Result(ctx, command.ID)
			return result, false, resultErr
		}
		terminal, terminalErr := request.AcceptedPresent(record, true)
		if terminalErr != nil {
			return CommandResult{}, false, terminalErr
		}
		if err := service.store.CompleteDispatchAndCommand(
			ctx,
			attempt.ID,
			DispatchCompletion{Outcome: CommandAccepted, ResultRecords: record.Record},
			terminal,
		); err != nil {
			return CommandResult{}, false, err
		}
		result, terminalErr := service.store.Result(ctx, command.ID)
		return result, false, terminalErr
	}
	terminal := request.AcceptedAbsent()
	if err := service.store.CompleteDispatchAndCommand(
		ctx,
		attempt.ID,
		DispatchCompletion{Outcome: CommandAccepted},
		terminal,
	); err != nil {
		return CommandResult{}, false, err
	}
	result, err := service.store.Result(ctx, command.ID)
	return result, false, err
}

func (service *SetCommandService) accept(ctx context.Context, commandID uuid.UUID, terminal TerminalResult) (CommandResult, error) {
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *SetCommandService) reject(ctx context.Context, commandID uuid.UUID, request SetCommandRequest, cause error) (CommandResult, error) {
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

func (service *SetCommandService) rejectDispatch(
	ctx context.Context,
	attemptID uuid.UUID,
	commandID uuid.UUID,
	request SetCommandRequest,
	completion DispatchCompletion,
	cause error,
) (CommandResult, error) {
	terminal := request.Rejected(cause)
	if terminal.State == "" {
		terminal.State = CommandRejected
	}
	if err := service.store.CompleteDispatchAndCommand(ctx, attemptID, completion, terminal); err != nil {
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *SetCommandService) resolveExpectedOwners(
	ctx context.Context,
	command PreparedCommand,
	targetGeneration int64,
) ([]ownerlifecycle.ExpectedOwner, error) {
	owners := []syntax.DID{command.ExpectedOwner}
	if command.ExpectedTarget != "" {
		owners = append(owners, command.ExpectedTarget)
	}
	canonical, err := ownerlifecycle.CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	resolved := make([]ownerlifecycle.ExpectedOwner, 0, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := service.lifecycles.Get(ctx, owner)
		if errors.Is(err, pgx.ErrNoRows) && owner != command.ExpectedOwner {
			if owner == command.ExpectedTarget && targetGeneration > 0 {
				return nil, ownerlifecycle.ErrGenerationChanged
			}
			resolved = append(resolved, ownerlifecycle.ExpectedOwner{Owner: owner, AllowMissing: true})
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
		if owner == command.ExpectedOwner && lifecycle.Generation != command.ExpectedOwnerGeneration {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		if owner == command.ExpectedTarget && targetGeneration > 0 && lifecycle.Generation != targetGeneration {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		resolved = append(resolved, ownerlifecycle.ExpectedOwner{Owner: owner, Generation: lifecycle.Generation})
	}
	return resolved, nil
}

func equalCanonicalJSON(left, right json.RawMessage) (bool, error) {
	leftCanonical, err := canonicalJSON(json.RawMessage(left))
	if err != nil {
		return false, err
	}
	rightCanonical, err := canonicalJSON(json.RawMessage(right))
	if err != nil {
		return false, err
	}
	return string(leftCanonical) == string(rightCanonical), nil
}
