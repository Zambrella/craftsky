package api

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type fakeDirectory struct {
	identity *identity.Identity
	err      error
}

func (f *fakeDirectory) LookupDID(ctx context.Context, did syntax.DID) (*identity.Identity, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.identity, nil
}

// Stubs for the rest of the identity.Directory interface. We only
// care about LookupDID; everything else panics if exercised.
func (f *fakeDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	panic("unexpected LookupHandle")
}
func (f *fakeDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	panic("unexpected Lookup")
}
func (f *fakeDirectory) Purge(context.Context, syntax.AtIdentifier) error {
	panic("unexpected Purge")
}

func TestHandleResolver_HappyPath(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{
		identity: &identity.Identity{Handle: syntax.Handle("alice.bsky.social")},
	}}
	h, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err != nil {
		t.Fatalf("ResolveHandle: %v", err)
	}
	if h != "alice.bsky.social" {
		t.Errorf("handle = %q, want %q", h, "alice.bsky.social")
	}
}

func TestHandleResolver_MalformedDID(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{}}
	_, err := r.ResolveHandle(context.Background(), "not-a-did")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleResolver_DirectoryError(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{err: errors.New("network down")}}
	_, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleResolver_EmptyHandle(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{
		identity: &identity.Identity{Handle: syntax.HandleInvalid},
	}}
	_, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
