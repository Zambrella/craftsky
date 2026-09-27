package pdscommands

import (
	"context"
	"errors"
	"sync"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type FencedAppendCommandRequest struct {
	Command      AppendCommandRequest
	SelectedURI  syntax.ATURI
	SelectedRkey syntax.RecordKey
}

type AlreadyFencedCommandExecutor interface {
	ExecuteAppend(context.Context, FencedAppendCommandRequest) (CommandResult, error)
}

type alreadyFencedCommandExecutor struct {
	append   *AppendCommandService
	protocol PDSProtocol
	expected []ownerlifecycle.ExpectedOwner
}

type ScopedAlreadyFencedCommandExecutor struct {
	delegate AlreadyFencedCommandExecutor
	mu       sync.Mutex
	closed   bool
	inFlight int
	done     *sync.Cond
}

func NewAlreadyFencedCommandExecutor(
	appendService *AppendCommandService,
	client auth.PDSClient,
	expected []ownerlifecycle.ExpectedOwner,
) (AlreadyFencedCommandExecutor, error) {
	protocol, ok := client.(PDSProtocol)
	if appendService == nil || !ok || protocol == nil || len(expected) == 0 {
		return nil, errors.New("already-fenced PDS command executor is unavailable")
	}
	return &alreadyFencedCommandExecutor{
		append: appendService, protocol: protocol,
		expected: append([]ownerlifecycle.ExpectedOwner(nil), expected...),
	}, nil
}

func NewScopedAlreadyFencedCommandExecutor(
	delegate AlreadyFencedCommandExecutor,
) (*ScopedAlreadyFencedCommandExecutor, error) {
	if delegate == nil {
		return nil, errors.New("already-fenced PDS command executor is unavailable")
	}
	executor := &ScopedAlreadyFencedCommandExecutor{delegate: delegate}
	executor.done = sync.NewCond(&executor.mu)
	return executor, nil
}

func (executor *ScopedAlreadyFencedCommandExecutor) ExecuteAppend(
	ctx context.Context,
	request FencedAppendCommandRequest,
) (CommandResult, error) {
	if executor == nil || !executor.begin() {
		return CommandResult{}, ownerlifecycle.ErrFenceRequired
	}
	defer executor.end()
	return executor.delegate.ExecuteAppend(ctx, request)
}

func (executor *ScopedAlreadyFencedCommandExecutor) CloseAndWait() {
	if executor == nil {
		return
	}
	executor.mu.Lock()
	executor.closed = true
	for executor.inFlight > 0 {
		executor.done.Wait()
	}
	executor.mu.Unlock()
}

func (executor *ScopedAlreadyFencedCommandExecutor) begin() bool {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.closed {
		return false
	}
	executor.inFlight++
	return true
}

func (executor *ScopedAlreadyFencedCommandExecutor) end() {
	executor.mu.Lock()
	executor.inFlight--
	if executor.inFlight == 0 {
		executor.done.Broadcast()
	}
	executor.mu.Unlock()
}

func (executor *alreadyFencedCommandExecutor) ExecuteAppend(
	ctx context.Context,
	request FencedAppendCommandRequest,
) (CommandResult, error) {
	if executor == nil || executor.append == nil || executor.protocol == nil {
		return CommandResult{}, errors.New("already-fenced PDS command executor is unavailable")
	}
	return executor.append.executeAlreadyFenced(
		ctx,
		executor.protocol,
		executor.expected,
		request,
	)
}
