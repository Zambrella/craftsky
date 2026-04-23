// appview/internal/api/profile_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProfileNotFound is returned by ProfileStore.Read when the DID has
// no Craftsky membership row. Handlers translate this into 404.
var ErrProfileNotFound = errors.New("profile: not found")

// ProfileRow is the joined read view of craftsky_profiles and
// bluesky_profiles for a single DID. Nullable bluesky fields are pointers
// so "present but empty string" and "absent entirely" are distinguishable.
type ProfileRow struct {
	DID         string
	Crafts      []string
	CreatedAt   time.Time
	DisplayName *string
	Description *string
	AvatarCID   *string
	AvatarMime  *string
	BannerCID   *string
	BannerMime  *string
}

// ProfileStore is the Postgres-backed read/write surface used by the
// /v1/profiles/* handlers. It owns no business logic; merges, validation,
// and URL synthesis live in the handler layer.
type ProfileStore struct {
	pool *pgxpool.Pool
}

func NewProfileStore(pool *pgxpool.Pool) *ProfileStore {
	return &ProfileStore{pool: pool}
}

// Read returns the joined profile for did. Returns ErrProfileNotFound if
// the DID is not a Craftsky member (absent from craftsky_profiles).
func (s *ProfileStore) Read(ctx context.Context, did string) (*ProfileRow, error) {
	const q = `
		SELECT
			cp.did, cp.crafts, cp.created_at,
			bp.display_name, bp.description,
			bp.avatar_cid, bp.avatar_mime,
			bp.banner_cid, bp.banner_mime
		FROM craftsky_profiles cp
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE cp.did = $1
	`
	row := s.pool.QueryRow(ctx, q, did)
	out := &ProfileRow{}
	err := row.Scan(
		&out.DID, &out.Crafts, &out.CreatedAt,
		&out.DisplayName, &out.Description,
		&out.AvatarCID, &out.AvatarMime,
		&out.BannerCID, &out.BannerMime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("profile read %s: %w", did, err)
	}
	return out, nil
}
