package pdseffects

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// ExecutorFactory binds a durable executor to one OAuth parent session. The
// returned executor still accepts multi-owner ExpectedOwners, but its remote
// repository owner can only be the owner supplied here.
type ExecutorFactory func(context.Context, syntax.DID, string) (EffectExecutor, error)

// GuardedExecutorFactory builds the coordinator-only form used by workflows
// that must hold an object/work lock across the durable remote effect and
// local finalization. It never returns a raw PDS client or an ordinary
// recursively-enterable executor.
type GuardedExecutorFactory func(
	context.Context,
	syntax.DID,
	string,
) (GuardedEffectCoordinator, error)

// NewExecutorFactory composes the durable attempt store with the single
// combined owner-generation/OAuth-session boundary. Application wiring should
// construct this once and inject the resulting factory into handlers.
func NewExecutorFactory(
	attempts *ownerlifecycle.Store,
	clients auth.PDSClientFactory,
	timeout time.Duration,
	now func() time.Time,
) (ExecutorFactory, error) {
	build, err := newExecutorBuilder(attempts, clients, timeout, now)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, owner syntax.DID, sessionID string) (EffectExecutor, error) {
		return build(ctx, owner, sessionID)
	}, nil
}

// NewGuardedExecutorFactory uses the same owner/session binding as
// NewExecutorFactory but exposes only callback-scoped effect composition.
func NewGuardedExecutorFactory(
	attempts *ownerlifecycle.Store,
	clients auth.PDSClientFactory,
	timeout time.Duration,
	now func() time.Time,
) (GuardedExecutorFactory, error) {
	build, err := newExecutorBuilder(attempts, clients, timeout, now)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		owner syntax.DID,
		sessionID string,
	) (GuardedEffectCoordinator, error) {
		executor, err := build(ctx, owner, sessionID)
		if err != nil {
			return nil, err
		}
		return guardedEffectCoordinator{executor: executor}, nil
	}, nil
}

type concreteExecutorBuilder func(
	context.Context,
	syntax.DID,
	string,
) (*Executor, error)

func newExecutorBuilder(
	attempts *ownerlifecycle.Store,
	clients auth.PDSClientFactory,
	timeout time.Duration,
	now func() time.Time,
) (concreteExecutorBuilder, error) {
	if attempts == nil || clients == nil || timeout <= 0 {
		return nil, errors.New("durable PDS effect factory dependencies are unavailable")
	}
	if now == nil {
		now = time.Now
	}
	return func(ctx context.Context, owner syntax.DID, sessionID string) (*Executor, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, err := clients(ctx, owner, sessionID)
		if err != nil {
			return nil, err
		}
		boundary, ok := client.(auth.ActiveEffectPDSBoundary)
		if !ok || boundary == nil {
			return nil, errors.New("PDS client factory dropped the combined active-effect boundary")
		}
		return NewExecutor(attempts, boundary, owner, timeout, now)
	}, nil
}
