package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleResolver resolves a DID to its current handle via indigo's
// identity.Directory. v1 does no caching beyond what the directory
// provides internally — every /v1/whoami call pays one lookup.
//
// A nil Directory is a programmer error and will panic on use.
type HandleResolver struct {
	Directory identity.Directory
}

// ErrHandleUnavailable wraps every failure mode (malformed DID,
// directory error, empty handle) into a single sentinel. Handlers
// convert this to 502 identity_unavailable.
var ErrHandleUnavailable = errors.New("handle unavailable")

// ResolveHandle returns the handle for did.
func (r HandleResolver) ResolveHandle(ctx context.Context, did string) (string, error) {
	parsed, err := syntax.ParseDID(did)
	if err != nil {
		return "", fmt.Errorf("%w: parse did: %v", ErrHandleUnavailable, err)
	}
	id, err := r.Directory.LookupDID(ctx, parsed)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	h := id.Handle.String()
	if h == "" || h == syntax.HandleInvalid.String() {
		// syntax.HandleInvalid ("handle.invalid") is the indigo
		// sentinel for DIDs with no valid handle (deactivated,
		// mid-migration, etc.).
		return "", fmt.Errorf("%w: empty handle for %s", ErrHandleUnavailable, did)
	}
	return h, nil
}
