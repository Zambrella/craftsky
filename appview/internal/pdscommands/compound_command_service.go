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

type CompoundCommandServiceConfig struct {
	Store       *Store
	Lifecycles  *ownerlifecycle.Store
	NewBoundary ActivePDSBoundaryFactory
	Now         func() time.Time
	Sleep       func(time.Duration)
	Jitter      func(int) int
}

type CompoundCommandService struct {
	store       *Store
	lifecycles  *ownerlifecycle.Store
	newBoundary ActivePDSBoundaryFactory
	now         func() time.Time
	sleep       func(time.Duration)
	jitter      func(int) int
}

type CompoundPutRecordRequest struct {
	URI         syntax.ATURI
	BuildRecord func(AuthoritativeRecord) (json.RawMessage, error)
}

type CompoundPutCommandRequest struct {
	Owner           syntax.DID
	OwnerGeneration int64
	SessionID       string
	OperationKind   string
	OperationKey    uuid.UUID
	Intent          json.RawMessage
	Blobs           []BlobReference
	Records         []CompoundPutRecordRequest
	Accepted        func([]AuthoritativeRecord) (TerminalResult, error)
	Rejected        func(error) TerminalResult
}

func NewCompoundCommandService(config CompoundCommandServiceConfig) (*CompoundCommandService, error) {
	if config.Store == nil || config.Lifecycles == nil || config.NewBoundary == nil {
		return nil, errors.New("compound command service dependencies are unavailable")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Sleep == nil {
		config.Sleep = time.Sleep
	}
	return &CompoundCommandService{
		store: config.Store, lifecycles: config.Lifecycles, newBoundary: config.NewBoundary,
		now: config.Now, sleep: config.Sleep, jitter: config.Jitter,
	}, nil
}

func (service *CompoundCommandService) Put(ctx context.Context, request CompoundPutCommandRequest) (CommandResult, error) {
	rkeys, err := validateCompoundPutRequest(request)
	if err != nil {
		return CommandResult{}, err
	}
	conditions, err := json.Marshal(struct {
		OwnerGeneration int64          `json:"ownerGeneration"`
		URIs            []syntax.ATURI `json:"uris"`
	}{OwnerGeneration: request.OwnerGeneration, URIs: []syntax.ATURI{request.Records[0].URI, request.Records[1].URI}})
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
		SelectedURI: request.Records[0].URI, SelectedRkey: rkeys[0],
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
		return service.reject(ctx, prepared.ID, request, err)
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
		result, err = service.putFenced(operationCtx, transport, prepared, fingerprint, request)
		return err
	})
	if err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func validateCompoundPutRequest(request CompoundPutCommandRequest) ([]syntax.RecordKey, error) {
	if request.Owner == "" || request.OwnerGeneration < 1 || request.SessionID == "" || request.OperationKind == "" ||
		request.OperationKey == uuid.Nil || !json.Valid(request.Intent) || len(request.Records) != 2 ||
		request.Accepted == nil || request.Rejected == nil {
		return nil, ErrMalformedCommand
	}
	rkeys := make([]syntax.RecordKey, len(request.Records))
	seen := make(map[syntax.ATURI]struct{}, len(request.Records))
	for index, record := range request.Records {
		if record.URI == "" || record.BuildRecord == nil {
			return nil, ErrMalformedCommand
		}
		parsed, err := syntax.ParseATURI(record.URI.String())
		if err != nil || parsed.Authority().DID() != request.Owner || parsed.Collection() == "" || parsed.RecordKey() == "" {
			return nil, ErrMalformedCommand
		}
		if _, duplicate := seen[record.URI]; duplicate {
			return nil, ErrMalformedCommand
		}
		seen[record.URI] = struct{}{}
		rkeys[index] = parsed.RecordKey()
	}
	for _, blob := range request.Blobs {
		if blob.CID == "" || blob.MIMEType == "" || blob.Size < 0 {
			return nil, ErrMalformedCommand
		}
	}
	return rkeys, nil
}

func (service *CompoundCommandService) putFenced(
	ctx context.Context,
	transport *PDSTransport,
	command PreparedCommand,
	requestFingerprint [32]byte,
	request CompoundPutCommandRequest,
) (CommandResult, error) {
	for attemptNumber := 1; attemptNumber <= 3; attemptNumber++ {
		head, current, err := readStableCompoundRecords(ctx, transport, request.Owner, request.Records)
		if err != nil {
			if errors.Is(err, ErrRepositoryConflict) {
				delay, retry := InvalidSwapRetryDelay(attemptNumber, service.jitter)
				if retry {
					service.sleep(delay)
					continue
				}
				return service.reject(ctx, command.ID, request, err)
			}
			return CommandResult{}, errors.Join(ErrDispatchUnavailable, err)
		}
		desired := make([]json.RawMessage, len(request.Records))
		allMatch := true
		for index, record := range request.Records {
			desired[index], err = record.BuildRecord(current[index])
			if err != nil || !json.Valid(desired[index]) {
				return service.reject(ctx, command.ID, request, errors.Join(ErrMalformedCommand, err))
			}
			if current[index].CID == "" {
				allMatch = false
				continue
			}
			matches, compareErr := equalCanonicalJSON(current[index].Record, desired[index])
			if compareErr != nil {
				return CommandResult{}, compareErr
			}
			allMatch = allMatch && matches
		}
		if allMatch {
			return service.acceptRecords(ctx, command.ID, request, current)
		}
		if command.State != CommandPrepared {
			if command.State == CommandDispatching {
				if err := service.store.MarkOpenDispatchAmbiguous(ctx, command.ID, 1, "recovery_required"); err != nil {
					return CommandResult{}, err
				}
			}
			return service.store.Result(ctx, command.ID)
		}
		steps := make([]DispatchStep, len(request.Records))
		for index, record := range request.Records {
			action := "update"
			if current[index].CID == "" {
				action = "create"
			}
			steps[index] = DispatchStep{
				Action: action, URI: record.URI, Body: desired[index], ExpectedCID: current[index].CID,
			}
		}
		dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
			AlgorithmVersion: 1, RequestFingerprint: requestFingerprint, RepositoryHead: head, Steps: steps,
		})
		if err != nil {
			return CommandResult{}, err
		}
		attempt, err := service.store.BeginDispatch(ctx, command.ID, ExactPlan{
			Version: command.ActivePlanVersion + attemptNumber, FingerprintVersion: 1,
			Fingerprint: dispatchFingerprint, RepositoryCID: head,
			RemoteDeadline: service.now().UTC().Add(30 * time.Second), Steps: steps,
		})
		if err != nil {
			if errors.Is(err, ErrIllegalTransition) {
				return service.store.Result(ctx, command.ID)
			}
			return CommandResult{}, err
		}
		err = transport.ApplyWrites(ctx, request.Owner, head, steps)
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
		_, authoritative, err := readStableCompoundRecords(ctx, transport, request.Owner, request.Records)
		if err != nil || !compoundRecordsMatch(authoritative, desired) {
			errorClass := "result_mismatch"
			if err != nil {
				errorClass = "result_read"
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: errorClass}); completeErr != nil {
				return CommandResult{}, completeErr
			}
			return service.store.Result(ctx, command.ID)
		}
		resultRecords, err := json.Marshal(authoritative)
		if err != nil {
			return CommandResult{}, err
		}
		terminal, err := request.Accepted(authoritative)
		if err != nil {
			return CommandResult{}, err
		}
		if err := service.store.CompleteDispatchAndCommand(
			ctx,
			attempt.ID,
			DispatchCompletion{Outcome: CommandAccepted, ResultRecords: resultRecords},
			terminal,
		); err != nil {
			return CommandResult{}, err
		}
		return service.store.Result(ctx, command.ID)
	}
	return service.reject(ctx, command.ID, request, ErrRepositoryConflict)
}

func readStableCompoundRecords(
	ctx context.Context,
	transport *PDSTransport,
	owner syntax.DID,
	requests []CompoundPutRecordRequest,
) (syntax.CID, []AuthoritativeRecord, error) {
	headBefore, err := transport.LatestCommit(ctx, owner)
	if err != nil {
		return "", nil, err
	}
	records := make([]AuthoritativeRecord, len(requests))
	for index, request := range requests {
		records[index], err = transport.GetRecord(ctx, owner, request.URI)
		if errors.Is(err, auth.ErrRecordNotFound) {
			records[index] = AuthoritativeRecord{URI: request.URI}
			continue
		}
		if err != nil {
			return "", nil, err
		}
	}
	headAfter, err := transport.LatestCommit(ctx, owner)
	if err != nil {
		return "", nil, err
	}
	if headBefore != headAfter {
		return "", nil, ErrRepositoryConflict
	}
	return headAfter, records, nil
}

func compoundRecordsMatch(records []AuthoritativeRecord, desired []json.RawMessage) bool {
	if len(records) != len(desired) {
		return false
	}
	for index := range records {
		if records[index].CID == "" {
			return false
		}
		matches, err := equalCanonicalJSON(records[index].Record, desired[index])
		if err != nil || !matches {
			return false
		}
	}
	return true
}

func (service *CompoundCommandService) acceptRecords(
	ctx context.Context,
	commandID uuid.UUID,
	request CompoundPutCommandRequest,
	records []AuthoritativeRecord,
) (CommandResult, error) {
	terminal, err := request.Accepted(records)
	if err != nil {
		return CommandResult{}, err
	}
	if err := service.store.CompleteCommand(ctx, commandID, terminal); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			return service.store.Result(ctx, commandID)
		}
		return CommandResult{}, err
	}
	return service.store.Result(ctx, commandID)
}

func (service *CompoundCommandService) reject(
	ctx context.Context,
	commandID uuid.UUID,
	request CompoundPutCommandRequest,
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

func (service *CompoundCommandService) rejectDispatch(
	ctx context.Context,
	attemptID uuid.UUID,
	commandID uuid.UUID,
	request CompoundPutCommandRequest,
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
