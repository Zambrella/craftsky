package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleResolver resolves between atproto DIDs and handles. Production
// impl wraps indigo's identity.Directory; tests commonly stub the
// interface directly.
type HandleResolver interface {
	ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error)
	ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error)
}

// DirectoryHandleResolver is the indigo-backed implementation.
type DirectoryHandleResolver struct {
	Directory identity.Directory
}

var _ HandleResolver = DirectoryHandleResolver{}

// ErrHandleUnavailable wraps every failure mode (directory error, empty
// handle, etc). Handlers convert this to 502 identity_unavailable.
var ErrHandleUnavailable = errors.New("handle unavailable")

// ResolveHandle returns the handle for did.
func (r DirectoryHandleResolver) ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error) {
	id, err := r.Directory.LookupDID(ctx, did)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	if id.Handle == "" || id.Handle == syntax.HandleInvalid {
		return "", fmt.Errorf("%w: empty handle for %s", ErrHandleUnavailable, did)
	}
	return id.Handle, nil
}

// ResolveDID returns the DID for handle.
func (r DirectoryHandleResolver) ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error) {
	id, err := r.Directory.LookupHandle(ctx, handle)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	return id.DID, nil
}
