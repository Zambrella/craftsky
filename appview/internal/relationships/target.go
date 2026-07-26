package relationships

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var (
	ErrInvalidIdentifier = errors.New("invalid identifier")
	ErrSelfRelationship  = errors.New("self relationship not allowed")
)

type TargetDIDResolver interface {
	ResolveDID(context.Context, syntax.Handle) (syntax.DID, error)
}

// ResolveTarget parses one boundary identifier, resolves a handle at most
// once, rejects self, then enforces current Craftsky membership.
func ResolveTarget(
	ctx context.Context,
	raw string,
	caller syntax.DID,
	resolver TargetDIDResolver,
	membership MembershipLookup,
) (syntax.DID, error) {
	raw = strings.TrimPrefix(raw, "@")
	var target syntax.DID
	if strings.HasPrefix(raw, "did:") {
		did, err := syntax.ParseDID(raw)
		if err != nil {
			return "", ErrInvalidIdentifier
		}
		target = did
	} else {
		handle, err := syntax.ParseHandle(raw)
		if err != nil {
			return "", ErrInvalidIdentifier
		}
		did, err := resolver.ResolveDID(ctx, handle)
		if err != nil {
			return "", fmt.Errorf("resolve target identity: %w", err)
		}
		target = did
	}

	if target == caller {
		return "", ErrSelfRelationship
	}
	if err := RequireCurrentMember(ctx, membership, target); err != nil {
		return "", err
	}
	return target, nil
}
