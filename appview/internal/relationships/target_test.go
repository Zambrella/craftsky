package relationships

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type targetResolverFake struct {
	did   syntax.DID
	err   error
	calls []syntax.Handle
}

func (f *targetResolverFake) ResolveDID(_ context.Context, handle syntax.Handle) (syntax.DID, error) {
	f.calls = append(f.calls, handle)
	return f.did, f.err
}

func TestResolveTargetCanonicalizesAndRequiresEligibleNonSelfMember(t *testing.T) {
	caller := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	t.Run("current member DID is accepted without handle resolution", func(t *testing.T) {
		resolver := &targetResolverFake{did: syntax.DID("did:plc:wrong")}
		membership := &membershipLookupFake{current: map[syntax.DID]bool{bob: true}}
		got, err := ResolveTarget(context.Background(), bob.String(), caller, resolver, membership)
		if err != nil {
			t.Fatalf("ResolveTarget: %v", err)
		}
		if got != bob {
			t.Fatalf("DID = %s, want %s", got, bob)
		}
		if len(resolver.calls) != 0 {
			t.Fatalf("DID target used handle resolver %d times", len(resolver.calls))
		}
	})

	t.Run("handle resolves once and canonicalizes to member DID", func(t *testing.T) {
		resolver := &targetResolverFake{did: bob}
		membership := &membershipLookupFake{current: map[syntax.DID]bool{bob: true}}
		got, err := ResolveTarget(context.Background(), "@bob.example", caller, resolver, membership)
		if err != nil {
			t.Fatalf("ResolveTarget: %v", err)
		}
		if got != bob {
			t.Fatalf("DID = %s, want %s", got, bob)
		}
		if len(resolver.calls) != 1 || resolver.calls[0] != syntax.Handle("bob.example") {
			t.Fatalf("resolver calls = %v, want [bob.example]", resolver.calls)
		}
	})

	t.Run("invalid identifier stops before resolution and membership", func(t *testing.T) {
		resolver := &targetResolverFake{}
		membership := &membershipLookupFake{}
		_, err := ResolveTarget(context.Background(), "not a handle", caller, resolver, membership)
		if !errors.Is(err, ErrInvalidIdentifier) {
			t.Fatalf("error = %v, want ErrInvalidIdentifier", err)
		}
		if len(resolver.calls) != 0 || len(membership.calls) != 0 {
			t.Fatalf("invalid target performed resolver=%d membership=%d calls", len(resolver.calls), len(membership.calls))
		}
	})

	t.Run("self target stops before membership", func(t *testing.T) {
		resolver := &targetResolverFake{}
		membership := &membershipLookupFake{current: map[syntax.DID]bool{caller: true}}
		_, err := ResolveTarget(context.Background(), caller.String(), caller, resolver, membership)
		if !errors.Is(err, ErrSelfRelationship) {
			t.Fatalf("error = %v, want ErrSelfRelationship", err)
		}
		if len(membership.calls) != 0 {
			t.Fatalf("self target performed %d membership calls", len(membership.calls))
		}
	})

	t.Run("resolvable nonmember is indistinguishable from unknown", func(t *testing.T) {
		for _, target := range []syntax.DID{"did:plc:resolvable", "did:plc:unknown"} {
			resolver := &targetResolverFake{did: target}
			membership := &membershipLookupFake{}
			_, err := ResolveTarget(context.Background(), "target.example", caller, resolver, membership)
			if !errors.Is(err, ErrProfileNotFound) {
				t.Fatalf("target %s error = %v, want ErrProfileNotFound", target, err)
			}
		}
	})

	t.Run("resolver failure remains an identity failure", func(t *testing.T) {
		want := errors.New("directory unavailable")
		resolver := &targetResolverFake{err: want}
		membership := &membershipLookupFake{}
		_, err := ResolveTarget(context.Background(), "bob.example", caller, resolver, membership)
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want wrapped resolver failure", err)
		}
		if errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("resolver failure was misclassified as profile not found: %v", err)
		}
		if len(membership.calls) != 0 {
			t.Fatalf("resolver failure performed %d membership calls", len(membership.calls))
		}
	})
}
