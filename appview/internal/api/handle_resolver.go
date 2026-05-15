package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
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

// DevHandleResolver falls back to a deterministic local handle for mirrored
// development profiles when the real atproto directory cannot resolve a DID.
// This keeps local-only fake seed data usable without changing prod behavior.
type DevHandleResolver struct {
	Primary HandleResolver
	Pool    *pgxpool.Pool
}

var _ HandleResolver = DevHandleResolver{}

func (r DevHandleResolver) ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error) {
	if r.Primary != nil {
		handle, err := r.Primary.ResolveHandle(ctx, did)
		if err == nil {
			return handle, nil
		}
	}
	if r.Pool == nil {
		return "", fmt.Errorf("%w: no dev fallback store", ErrHandleUnavailable)
	}

	var exists bool
	if err := r.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM bluesky_profiles WHERE did = $1)`, did.String()).Scan(&exists); err != nil {
		return "", fmt.Errorf("%w: dev fallback lookup: %v", ErrHandleUnavailable, err)
	}
	if !exists {
		return "", fmt.Errorf("%w: no local profile for %s", ErrHandleUnavailable, did)
	}
	handle, err := syntax.ParseHandle(localHandleForDID(did.String()))
	if err != nil {
		return "", fmt.Errorf("%w: build local handle: %v", ErrHandleUnavailable, err)
	}
	return handle, nil
}

func (r DevHandleResolver) ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error) {
	if r.Primary == nil {
		return "", fmt.Errorf("%w: no primary resolver", ErrHandleUnavailable)
	}
	return r.Primary.ResolveDID(ctx, handle)
}

func localHandleForDID(did string) string {
	_, id, ok := strings.Cut(strings.TrimSpace(strings.ToLower(did)), ":plc:")
	if !ok {
		_, id, ok = strings.Cut(strings.TrimSpace(strings.ToLower(did)), ":")
		if !ok {
			id = "user"
		}
	}
	var b strings.Builder
	lastWasHyphen := false
	for _, r := range id {
		valid := unicode.IsLetter(r) || unicode.IsDigit(r)
		if valid {
			b.WriteRune(r)
			lastWasHyphen = false
			continue
		}
		if !lastWasHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastWasHyphen = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "user"
	}
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
		if slug == "" {
			slug = "user"
		}
	}
	return slug + ".craftsky.test"
}
