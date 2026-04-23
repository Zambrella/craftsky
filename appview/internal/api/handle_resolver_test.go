package api

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// fakeDirectory lets us inject canned responses for both directions.
// LookupDID and LookupHandle each return the paired (identity, err); any
// unused direction returns a sentinel panic to catch accidental calls.
type fakeDirectory struct {
	didResult    *identity.Identity
	didErr       error
	handleResult *identity.Identity
	handleErr    error
}

func (f *fakeDirectory) LookupDID(_ context.Context, _ syntax.DID) (*identity.Identity, error) {
	return f.didResult, f.didErr
}
func (f *fakeDirectory) LookupHandle(_ context.Context, _ syntax.Handle) (*identity.Identity, error) {
	return f.handleResult, f.handleErr
}
func (f *fakeDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	panic("unexpected Lookup")
}
func (f *fakeDirectory) Purge(context.Context, syntax.AtIdentifier) error {
	panic("unexpected Purge")
}

func TestHandleResolver_ResolveHandle_HappyPath(t *testing.T) {
	r := DirectoryHandleResolver{Directory: &fakeDirectory{
		didResult: &identity.Identity{Handle: syntax.Handle("alice.bsky.social")},
	}}
	h, err := r.ResolveHandle(context.Background(), syntax.DID("did:plc:abc"))
	if err != nil {
		t.Fatalf("ResolveHandle: %v", err)
	}
	if h != syntax.Handle("alice.bsky.social") {
		t.Errorf("handle = %q", h)
	}
}

func TestHandleResolver_ResolveHandle_DirectoryError(t *testing.T) {
	r := DirectoryHandleResolver{Directory: &fakeDirectory{
		didErr: errors.New("plc down"),
	}}
	_, err := r.ResolveHandle(context.Background(), syntax.DID("did:plc:abc"))
	if !errors.Is(err, ErrHandleUnavailable) {
		t.Errorf("want ErrHandleUnavailable; got %v", err)
	}
}

func TestHandleResolver_ResolveHandle_EmptyHandle(t *testing.T) {
	r := DirectoryHandleResolver{Directory: &fakeDirectory{
		didResult: &identity.Identity{Handle: syntax.HandleInvalid},
	}}
	_, err := r.ResolveHandle(context.Background(), syntax.DID("did:plc:abc"))
	if !errors.Is(err, ErrHandleUnavailable) {
		t.Errorf("want ErrHandleUnavailable; got %v", err)
	}
}

func TestHandleResolver_ResolveDID_HappyPath(t *testing.T) {
	r := DirectoryHandleResolver{Directory: &fakeDirectory{
		handleResult: &identity.Identity{DID: syntax.DID("did:plc:xyz")},
	}}
	got, err := r.ResolveDID(context.Background(), syntax.Handle("alice.example"))
	if err != nil {
		t.Fatal(err)
	}
	if got != syntax.DID("did:plc:xyz") {
		t.Errorf("did = %q", got)
	}
}

func TestHandleResolver_ResolveDID_DirectoryError(t *testing.T) {
	r := DirectoryHandleResolver{Directory: &fakeDirectory{
		handleErr: errors.New("plc down"),
	}}
	_, err := r.ResolveDID(context.Background(), syntax.Handle("alice.example"))
	if !errors.Is(err, ErrHandleUnavailable) {
		t.Errorf("want ErrHandleUnavailable; got %v", err)
	}
}
