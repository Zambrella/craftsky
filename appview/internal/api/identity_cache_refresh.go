package api

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type IdentityCacheRefreshProcessorOptions struct {
	Store               *IdentityCacheStore
	Resolver            HandleResolver
	BatchSize           int
	OperationTimeout    time.Duration
	RetryDelay          time.Duration
	Now                 func() time.Time
	IdentityInvalidator IdentityInvalidator
}

type IdentityInvalidator interface {
	InvalidateIdentity(context.Context, syntax.DID, ...syntax.Handle)
}

type IdentityCacheRefreshProcessor struct {
	store               *IdentityCacheStore
	resolver            HandleResolver
	batchSize           int
	operationTimeout    time.Duration
	retryDelay          time.Duration
	now                 func() time.Time
	identityInvalidator IdentityInvalidator
}

func NewIdentityCacheRefreshProcessor(options IdentityCacheRefreshProcessorOptions) (*IdentityCacheRefreshProcessor, error) {
	if options.Store == nil || options.Resolver == nil {
		return nil, fmt.Errorf("identity cache refresh dependencies are unavailable")
	}
	if options.BatchSize < 1 || options.BatchSize > 1000 {
		return nil, fmt.Errorf("identity cache refresh batch size must be between 1 and 1000")
	}
	if options.OperationTimeout <= 0 || options.RetryDelay <= 0 {
		return nil, fmt.Errorf("identity cache refresh durations must be positive")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &IdentityCacheRefreshProcessor{
		store: options.Store, resolver: options.Resolver, batchSize: options.BatchSize,
		operationTimeout: options.OperationTimeout, retryDelay: options.RetryDelay,
		now: options.Now, identityInvalidator: options.IdentityInvalidator,
	}, nil
}

func (processor *IdentityCacheRefreshProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil {
		return 0, fmt.Errorf("identity cache refresh processor is unavailable")
	}
	now := processor.now().UTC()
	candidates, err := processor.store.refreshCandidates(ctx, processor.batchSize, now)
	if err != nil {
		return 0, err
	}
	for _, candidate := range candidates {
		did := candidate.DID
		age := time.Duration(0)
		if candidate.ResolvedAt != nil {
			age = now.Sub(*candidate.ResolvedAt)
		}
		operationCtx, cancel := context.WithTimeout(ctx, processor.operationTimeout)
		handle, valid, resolveErr := resolveAuthoritativeHandle(operationCtx, processor.resolver, did)
		cancel()
		if resolveErr != nil {
			deferred, err := processor.store.deferRefresh(ctx, candidate, now.Add(processor.retryDelay), now)
			if err != nil {
				return 0, err
			}
			if deferred && processor.store.observer != nil {
				processor.store.observer.ObserveIdentityCache("refresh_retry", age)
			}
			continue
		}
		if !valid {
			handle = syntax.HandleInvalid
		}
		commit, completed, err := processor.store.completeRefresh(ctx, candidate, handle, now)
		if err != nil {
			return 0, fmt.Errorf("identity cache refresh %s: %w", did, err)
		}
		if !completed {
			continue
		}
		invalidateIdentityCommit(ctx, processor.identityInvalidator, commit)
		if processor.store.observer != nil {
			processor.store.observer.ObserveIdentityCache("refresh_success", age)
		}
	}
	return len(candidates), nil
}

func resolveAuthoritativeHandle(ctx context.Context, resolver HandleResolver, did syntax.DID) (syntax.Handle, bool, error) {
	handle, err := resolver.ResolveHandle(ctx, did)
	if err != nil {
		return "", false, err
	}
	if handle == "" || handle.IsInvalidHandle() {
		return syntax.HandleInvalid, false, nil
	}
	resolvedDID, err := resolver.ResolveDID(ctx, handle)
	if err != nil {
		if isDefinitiveHandleResolutionError(err) {
			return syntax.HandleInvalid, false, nil
		}
		return "", false, err
	}
	if resolvedDID != did {
		return syntax.HandleInvalid, false, nil
	}
	return handle.Normalize(), true, nil
}

func invalidateIdentityCommit(ctx context.Context, invalidator IdentityInvalidator, commit identityRefreshCommit) {
	if invalidator == nil {
		return
	}
	invalidator.InvalidateIdentity(ctx, commit.DID, commit.Handles...)
	for _, did := range commit.DisplacedDID {
		invalidator.InvalidateIdentity(ctx, did, commit.Handles...)
	}
}
