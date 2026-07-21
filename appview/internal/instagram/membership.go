package instagram

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MembershipStore struct {
	pool *pgxpool.Pool
}

func NewMembershipStore(pool *pgxpool.Pool) *MembershipStore {
	return &MembershipStore{pool: pool}
}

func (s *MembershipStore) IsCurrentMember(ctx context.Context, did syntax.DID) (bool, error) {
	if s == nil || s.pool == nil {
		return false, errors.New("membership store is unavailable")
	}
	var current bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did,
	).Scan(&current); err != nil {
		return false, err
	}
	return current, nil
}
