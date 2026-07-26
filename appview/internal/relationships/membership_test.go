package relationships

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type membershipLookupFake struct {
	current map[syntax.DID]bool
	err     error
	calls   []syntax.DID
}

func (f *membershipLookupFake) IsCurrentMember(_ context.Context, did syntax.DID) (bool, error) {
	f.calls = append(f.calls, did)
	if f.err != nil {
		return false, f.err
	}
	return f.current[did], nil
}

func TestRequireCurrentMemberUsesOnlyProfileMembership(t *testing.T) {
	current := syntax.DID("did:plc:current")
	lookup := &membershipLookupFake{current: map[syntax.DID]bool{current: true}}

	if err := RequireCurrentMember(context.Background(), lookup, current); err != nil {
		t.Fatalf("current member rejected: %v", err)
	}

	for _, did := range []syntax.DID{
		"did:plc:former",
		"did:plc:resolvable-never-member",
		"did:plc:unknown",
	} {
		if err := RequireCurrentMember(context.Background(), lookup, did); !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("RequireCurrentMember(%s) error = %v, want ErrProfileNotFound", did, err)
		}
	}

	if len(lookup.calls) != 4 {
		t.Fatalf("membership lookup calls = %d, want 4", len(lookup.calls))
	}
}

func TestRequireCurrentMemberPreservesLookupFailure(t *testing.T) {
	want := errors.New("database unavailable")
	lookup := &membershipLookupFake{err: want}
	err := RequireCurrentMember(context.Background(), lookup, syntax.DID("did:plc:target"))
	if !errors.Is(err, want) {
		t.Fatalf("RequireCurrentMember error = %v, want wrapped lookup failure", err)
	}
	if errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("lookup failure was misclassified as profile not found: %v", err)
	}
}
