// appview/internal/api/profile_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProfileNotFound is returned by ProfileStore.Read when the DID has
// no Craftsky membership row. Handlers translate this into 404.
var ErrProfileNotFound = errors.New("profile: not found")

// ErrProfileCountsUnavailable is returned when required Craftsky profile
// counts cannot be calculated.
var ErrProfileCountsUnavailable = errors.New("profile: counts unavailable")

// ProfileRow is the joined read view of craftsky_profiles and
// bluesky_profiles for a single DID. Nullable bluesky fields are pointers
// so "present but empty string" and "absent entirely" are distinguishable.
type ProfileRow struct {
	DID               string
	Crafts            []string
	CreatedAt         time.Time
	FollowerCount     *int
	FollowingCount    *int
	ViewerIsFollowing bool
	IsCraftskyProfile bool
	DisplayName       *string
	Description       *string
	AvatarCID         *string
	AvatarMime        *string
	BannerCID         *string
	BannerMime        *string
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
			(
				SELECT COUNT(*)
				FROM atproto_follows f
				JOIN craftsky_profiles follower_cp ON follower_cp.did = f.did
				WHERE f.subject_did = cp.did
			) AS follower_count,
			(
				SELECT COUNT(*)
				FROM atproto_follows f
				JOIN craftsky_profiles target_cp ON target_cp.did = f.subject_did
				WHERE f.did = cp.did
			) AS following_count,
			bp.display_name, bp.description,
			bp.avatar_cid, bp.avatar_mime,
			bp.banner_cid, bp.banner_mime
		FROM craftsky_profiles cp
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE cp.did = $1
	`
	row := s.pool.QueryRow(ctx, q, did)
	out := &ProfileRow{}
	var followerCount int
	var followingCount int
	err := row.Scan(
		&out.DID, &out.Crafts, &out.CreatedAt,
		&followerCount, &followingCount,
		&out.DisplayName, &out.Description,
		&out.AvatarCID, &out.AvatarMime,
		&out.BannerCID, &out.BannerMime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.readNonCraftsky(ctx, did)
	}
	if err != nil {
		if strings.Contains(err.Error(), "atproto_follows") {
			return nil, fmt.Errorf("%w: %v", ErrProfileCountsUnavailable, err)
		}
		return nil, fmt.Errorf("profile read %s: %w", did, err)
	}
	out.FollowerCount = &followerCount
	out.FollowingCount = &followingCount
	out.IsCraftskyProfile = true
	return out, nil
}

func (s *ProfileStore) readNonCraftsky(ctx context.Context, did string) (*ProfileRow, error) {
	const q = `
		SELECT
			did,
			display_name,
			description,
			avatar_cid,
			avatar_mime,
			banner_cid,
			banner_mime
		FROM bluesky_profiles
		WHERE did = $1
	`
	out := &ProfileRow{}
	err := s.pool.QueryRow(ctx, q, did).Scan(
		&out.DID,
		&out.DisplayName,
		&out.Description,
		&out.AvatarCID,
		&out.AvatarMime,
		&out.BannerCID,
		&out.BannerMime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("non-craftsky profile read %s: %w", did, err)
	}
	out.Crafts = []string{}
	out.IsCraftskyProfile = false
	return out, nil
}
