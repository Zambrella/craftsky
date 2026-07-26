package relationships

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var (
	ErrProfileNotFound       = errors.New("profile not found")
	ErrMembershipUnavailable = errors.New("membership unavailable")
)

// MembershipLookup exposes the sole Craftsky membership predicate: the
// presence of a craftsky_profiles row.
type MembershipLookup interface {
	IsCurrentMember(context.Context, syntax.DID) (bool, error)
}

// RequireCurrentMember deliberately maps every absent identity to the same
// decision without attempting external profile hydration.
func RequireCurrentMember(ctx context.Context, lookup MembershipLookup, did syntax.DID) error {
	current, err := lookup.IsCurrentMember(ctx, did)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMembershipUnavailable, err)
	}
	if !current {
		return ErrProfileNotFound
	}
	return nil
}

// IsCurrentMember checks only the canonical Craftsky profile table.
func (s *Store) IsCurrentMember(ctx context.Context, did syntax.DID) (bool, error) {
	var current bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM craftsky_profiles
			WHERE did = $1
		)
	`, did).Scan(&current); err != nil {
		return false, fmt.Errorf("read current membership: %w", err)
	}
	return current, nil
}
