package app

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/observability"
)

type identityResolutionDependencies struct {
	cached        api.HandleResolver
	authoritative api.HandleResolver
	invalidator   *identityCacheInvalidator
}

type identityCacheInvalidator struct{ directory identity.Directory }

func newIdentityCacheInvalidator(directory identity.Directory) *identityCacheInvalidator {
	return &identityCacheInvalidator{directory: directory}
}

func (invalidator *identityCacheInvalidator) InvalidateIdentity(
	ctx context.Context,
	did syntax.DID,
	handles ...syntax.Handle,
) {
	if invalidator == nil || invalidator.directory == nil {
		return
	}
	_ = invalidator.directory.Purge(ctx, syntax.AtIdentifier(did.String()))
	seen := make(map[syntax.Handle]struct{}, len(handles))
	for _, handle := range handles {
		handle = handle.Normalize()
		if handle == "" || handle.IsInvalidHandle() {
			continue
		}
		if _, exists := seen[handle]; exists {
			continue
		}
		seen[handle] = struct{}{}
		_ = invalidator.directory.Purge(ctx, syntax.AtIdentifier(handle.String()))
	}
}

func newIdentityResolutionDependencies(
	cachedDirectory identity.Directory,
	authoritativeDirectory identity.Directory,
	pool *pgxpool.Pool,
	env Env,
	observer *observability.Observer,
) *identityResolutionDependencies {
	cached := api.HandleResolver(api.DirectoryHandleResolver{Directory: cachedDirectory})
	authoritative := api.HandleResolver(api.DirectoryHandleResolver{Directory: authoritativeDirectory})
	if env == EnvDev {
		cached = api.DevHandleResolver{Primary: cached, Pool: pool}
		authoritative = api.DevHandleResolver{Primary: authoritative, Pool: pool}
	}
	return &identityResolutionDependencies{
		cached:        api.NewObservedHandleResolver(cached, "cached", observer),
		authoritative: api.NewObservedHandleResolver(authoritative, "authoritative", observer),
		invalidator:   newIdentityCacheInvalidator(cachedDirectory),
	}
}
