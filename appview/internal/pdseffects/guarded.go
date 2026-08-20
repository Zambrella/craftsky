package pdseffects

import (
	"context"
	"errors"
	"sync"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// GuardedEffectOperation receives an effect-only executor whose authority is
// tied to the callback context and lifetime. The executor cannot expose or
// retain the purpose PDS client.
type GuardedEffectOperation func(context.Context, EffectExecutor) error

// GuardedEffectCoordinator acquires the canonical owner/session boundary once
// so a caller may acquire a later object/work lock and keep it across durable
// effect execution plus local finalization without owner-fence reentry.
type GuardedEffectCoordinator interface {
	WithGuardedEffects(
		context.Context,
		[]ownerlifecycle.ExpectedOwner,
		GuardedEffectOperation,
	) error
}

type guardedEffectCoordinator struct {
	executor *Executor
}

func (coordinator guardedEffectCoordinator) WithGuardedEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation GuardedEffectOperation,
) error {
	if coordinator.executor == nil {
		return errors.New("guarded PDS effect coordinator is unavailable")
	}
	return coordinator.executor.WithGuardedEffects(ctx, expected, operation)
}

func (executor *Executor) WithGuardedEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation GuardedEffectOperation,
) error {
	if executor == nil || executor.attempts == nil || executor.boundary == nil ||
		executor.owner == "" || operation == nil {
		return errors.New("guarded PDS effect coordinator is unavailable")
	}
	canonical, err := canonicalExpectedOwners(expected)
	if err != nil {
		return err
	}
	ownerIncluded := false
	for _, item := range canonical {
		if item.Owner == executor.owner {
			ownerIncluded = true
			break
		}
	}
	if !ownerIncluded {
		return ErrExecutorOwnerMismatch
	}
	return executor.boundary.WithActiveEffects(
		ctx,
		canonical,
		func(effectCtx context.Context, client auth.PDSClient) error {
			if client == nil {
				return errors.New("guarded PDS purpose client is unavailable")
			}
			scope := &callbackEffectBoundary{
				client:   client,
				token:    &struct{}{},
				expected: canonical,
			}
			scope.activate()
			defer scope.closeAndWait()
			callbackCtx := context.WithValue(effectCtx, guardedEffectContextKey{}, scope.token)
			scopedExecutor := &callbackEffectExecutor{
				executor: &Executor{
					attempts: executor.attempts,
					boundary: scope,
					owner:    executor.owner,
					timeout:  executor.timeout,
					now:      executor.now,
				},
				scope: scope,
			}
			return operation(callbackCtx, scopedExecutor)
		},
	)
}

type guardedEffectContextKey struct{}

type callbackEffectBoundary struct {
	client   auth.PDSClient
	token    *struct{}
	expected []ownerlifecycle.ExpectedOwner

	mu       sync.Mutex
	idle     *sync.Cond
	active   bool
	inFlight int
}

func (scope *callbackEffectBoundary) activate() {
	if scope == nil {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.idle == nil {
		scope.idle = sync.NewCond(&scope.mu)
	}
	scope.active = true
}

func (scope *callbackEffectBoundary) enter(ctx context.Context) error {
	if scope == nil || ctx == nil {
		return ErrGuardedEffectScopeExpired
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if !scope.active || ctx.Value(guardedEffectContextKey{}) != scope.token {
		return ErrGuardedEffectScopeExpired
	}
	scope.inFlight++
	return nil
}

func (scope *callbackEffectBoundary) leave() {
	if scope == nil {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.inFlight > 0 {
		scope.inFlight--
	}
	if scope.inFlight == 0 && scope.idle != nil {
		scope.idle.Broadcast()
	}
}

func (scope *callbackEffectBoundary) closeAndWait() {
	if scope == nil {
		return
	}
	scope.mu.Lock()
	if scope.idle == nil {
		scope.idle = sync.NewCond(&scope.mu)
	}
	scope.active = false
	for scope.inFlight > 0 {
		scope.idle.Wait()
	}
	scope.mu.Unlock()
}

func (scope *callbackEffectBoundary) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	if err := scope.enter(ctx); err != nil {
		return err
	}
	defer scope.leave()
	canonical, err := canonicalExpectedOwners(expected)
	if err != nil {
		return err
	}
	if !sameExpectedOwners(canonical, scope.expected) {
		return ErrGuardedEffectScopeMismatch
	}
	if operation == nil || scope.client == nil {
		return errors.New("guarded PDS effect operation is unavailable")
	}
	return operation(ctx, scope.client)
}

type callbackEffectExecutor struct {
	executor *Executor
	scope    *callbackEffectBoundary
}

func (executor *callbackEffectExecutor) ResolveExpectedOwners(
	ctx context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if err := executor.scope.enter(ctx); err != nil {
		return nil, err
	}
	defer executor.scope.leave()
	return executor.executor.ResolveExpectedOwners(ctx, ownerGeneration, targets)
}

func (executor *callbackEffectExecutor) ReadRecord(
	ctx context.Context,
	request ReadRecordRequest,
	out any,
) (syntax.CID, error) {
	if err := executor.scope.enter(ctx); err != nil {
		return "", err
	}
	defer executor.scope.leave()
	return executor.executor.ReadRecord(ctx, request, out)
}

func (executor *callbackEffectExecutor) PutRecord(
	ctx context.Context,
	request PutRecordRequest,
) (RecordResult, error) {
	if err := executor.scope.enter(ctx); err != nil {
		return RecordResult{}, err
	}
	defer executor.scope.leave()
	return executor.executor.PutRecord(ctx, request)
}

func (executor *callbackEffectExecutor) DeleteRecord(
	ctx context.Context,
	request DeleteRecordRequest,
) (RecordResult, error) {
	if err := executor.scope.enter(ctx); err != nil {
		return RecordResult{}, err
	}
	defer executor.scope.leave()
	return executor.executor.DeleteRecord(ctx, request)
}

func (executor *callbackEffectExecutor) UploadBlob(
	ctx context.Context,
	request UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	if err := executor.scope.enter(ctx); err != nil {
		return nil, err
	}
	defer executor.scope.leave()
	return executor.executor.UploadBlob(ctx, request)
}

func canonicalExpectedOwners(
	expected []ownerlifecycle.ExpectedOwner,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if len(expected) == 0 {
		return nil, ErrGuardedEffectScopeMismatch
	}
	byOwner := make(map[syntax.DID]ownerlifecycle.ExpectedOwner, len(expected))
	owners := make([]syntax.DID, 0, len(expected))
	for _, item := range expected {
		if item.Owner == "" {
			return nil, ownerlifecycle.ErrInvalidOwner
		}
		if item.AllowMissing {
			if item.Generation != 0 {
				return nil, ownerlifecycle.ErrGenerationChanged
			}
		} else if item.Generation <= 0 {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		if existing, exists := byOwner[item.Owner]; exists && existing != item {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		if _, exists := byOwner[item.Owner]; !exists {
			owners = append(owners, item.Owner)
		}
		byOwner[item.Owner] = item
	}
	canonicalOwners, err := ownerlifecycle.CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	canonical := make([]ownerlifecycle.ExpectedOwner, 0, len(canonicalOwners))
	for _, owner := range canonicalOwners {
		canonical = append(canonical, byOwner[owner])
	}
	return canonical, nil
}

func sameExpectedOwners(left, right []ownerlifecycle.ExpectedOwner) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

var _ GuardedEffectCoordinator = guardedEffectCoordinator{}
var _ auth.ActiveEffectPDSBoundary = (*callbackEffectBoundary)(nil)
var _ EffectExecutor = (*callbackEffectExecutor)(nil)
