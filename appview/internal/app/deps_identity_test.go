package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type countingIdentityDirectory struct {
	did         syntax.DID
	current     syntax.Handle
	didLookups  int
	handleCalls map[syntax.Handle]int
}

func (directory *countingIdentityDirectory) LookupHandle(_ context.Context, handle syntax.Handle) (*identity.Identity, error) {
	if directory.handleCalls == nil {
		directory.handleCalls = make(map[syntax.Handle]int)
	}
	directory.handleCalls[handle]++
	if handle != directory.current {
		return nil, fmt.Errorf("%w: %s", identity.ErrHandleMismatch, handle)
	}
	return &identity.Identity{DID: directory.did, Handle: handle, AlsoKnownAs: []string{"at://" + handle.String()}}, nil
}

func (directory *countingIdentityDirectory) LookupDID(context.Context, syntax.DID) (*identity.Identity, error) {
	directory.didLookups++
	return &identity.Identity{DID: directory.did, Handle: directory.current, AlsoKnownAs: []string{"at://" + directory.current.String()}}, nil
}

func (directory *countingIdentityDirectory) Lookup(ctx context.Context, atid syntax.AtIdentifier) (*identity.Identity, error) {
	if did, err := atid.AsDID(); err == nil {
		return directory.LookupDID(ctx, did)
	}
	handle, err := atid.AsHandle()
	if err != nil {
		return nil, err
	}
	return directory.LookupHandle(ctx, handle)
}

func (*countingIdentityDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

func TestTapIdentityInvalidationPurgesDIDAndKnownHandleEntriesWithoutLookup(t *testing.T) {
	ctx := context.Background()
	did := syntax.DID("did:plc:identity-cache-owner")
	oldHandle := syntax.Handle("old.example")
	newHandle := syntax.Handle("new.example")
	inner := &countingIdentityDirectory{did: did, current: oldHandle}
	cache := identity.NewCacheDirectory(inner, 10, time.Hour, time.Minute, time.Minute)

	if _, err := cache.LookupHandle(ctx, oldHandle); err != nil {
		t.Fatal(err)
	}
	if err := cache.Purge(ctx, syntax.AtIdentifier(did.String())); err != nil {
		t.Fatal(err)
	}
	inner.current = newHandle
	if _, err := cache.LookupHandle(ctx, newHandle); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.LookupDID(ctx, did); err != nil {
		t.Fatal(err)
	}
	beforeDID := inner.didLookups
	beforeOld := inner.handleCalls[oldHandle]
	beforeNew := inner.handleCalls[newHandle]

	invalidator := newIdentityCacheInvalidator(cache)
	invalidator.InvalidateIdentity(ctx, did, oldHandle, newHandle)
	if inner.didLookups != beforeDID || inner.handleCalls[oldHandle] != beforeOld || inner.handleCalls[newHandle] != beforeNew {
		t.Fatal("cache invalidation performed an identity lookup")
	}

	if _, err := cache.LookupDID(ctx, did); err != nil {
		t.Fatal(err)
	}
	invalidator.InvalidateIdentity(ctx, did, oldHandle, newHandle)
	if _, err := cache.LookupHandle(ctx, oldHandle); err == nil {
		t.Fatal("invalidated old handle unexpectedly remained resolvable")
	}
	invalidator.InvalidateIdentity(ctx, did, oldHandle, newHandle)
	if _, err := cache.LookupHandle(ctx, newHandle); err != nil {
		t.Fatal(err)
	}
	if inner.didLookups != beforeDID+1 || inner.handleCalls[oldHandle] != beforeOld+1 || inner.handleCalls[newHandle] != beforeNew+1 {
		t.Fatalf("post-invalidation lookups did=%d old=%d new=%d", inner.didLookups, inner.handleCalls[oldHandle], inner.handleCalls[newHandle])
	}
}
