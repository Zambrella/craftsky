// appview/internal/api/profile_store.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

// ErrProfileNotFound is returned by ProfileStore.Read when the DID has
// neither a Craftsky profile nor readable Bluesky profile data. Handlers
// translate this into 404.
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
	pool            *pgxpool.Pool
	blueskyHydrator auth.PDSClient
}

func NewProfileStore(pool *pgxpool.Pool, blueskyHydrator ...auth.PDSClient) *ProfileStore {
	store := &ProfileStore{pool: pool}
	if len(blueskyHydrator) > 0 {
		store.blueskyHydrator = blueskyHydrator[0]
	}
	return store
}

// Read returns the joined profile for profileDID and relationship state from
// viewerDID. Returns ErrProfileNotFound if neither a Craftsky profile nor a
// readable/hydratable Bluesky profile exists.
func (s *ProfileStore) Read(ctx context.Context, profileDID string, viewerDID string) (*ProfileRow, error) {
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
			CASE
				WHEN $2 = '' OR $2 = cp.did THEN false
				ELSE EXISTS (
					SELECT 1
					FROM atproto_follows f
					WHERE f.did = $2 AND f.subject_did = cp.did
				)
			END AS viewer_is_following,
			bp.display_name, bp.description,
			bp.avatar_cid, bp.avatar_mime,
			bp.banner_cid, bp.banner_mime
		FROM craftsky_profiles cp
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE cp.did = $1
	`
	row := s.pool.QueryRow(ctx, q, profileDID, viewerDID)
	out := &ProfileRow{}
	var followerCount int
	var followingCount int
	err := row.Scan(
		&out.DID, &out.Crafts, &out.CreatedAt,
		&followerCount, &followingCount,
		&out.ViewerIsFollowing,
		&out.DisplayName, &out.Description,
		&out.AvatarCID, &out.AvatarMime,
		&out.BannerCID, &out.BannerMime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.readNonCraftsky(ctx, profileDID, viewerDID)
	}
	if err != nil {
		if strings.Contains(err.Error(), "atproto_follows") {
			return nil, fmt.Errorf("%w: %v", ErrProfileCountsUnavailable, err)
		}
		return nil, fmt.Errorf("profile read %s: %w", profileDID, err)
	}
	out.FollowerCount = &followerCount
	out.FollowingCount = &followingCount
	out.IsCraftskyProfile = true
	return out, nil
}

func (s *ProfileStore) readNonCraftsky(ctx context.Context, profileDID string, viewerDID string) (*ProfileRow, error) {
	out, err := s.readNonCraftskyCached(ctx, profileDID, viewerDID)
	if errors.Is(err, ErrProfileNotFound) && s.blueskyHydrator != nil {
		if hydrateErr := s.hydrateNonCraftsky(ctx, profileDID); hydrateErr != nil {
			return nil, hydrateErr
		}
		return s.readNonCraftskyCached(ctx, profileDID, viewerDID)
	}
	return out, err
}

func (s *ProfileStore) readNonCraftskyCached(ctx context.Context, profileDID string, viewerDID string) (*ProfileRow, error) {
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
	err := s.pool.QueryRow(ctx, q, profileDID).Scan(
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
		return nil, fmt.Errorf("non-craftsky profile read %s: %w", profileDID, err)
	}
	viewerIsFollowing, err := s.viewerFollows(ctx, viewerDID, profileDID)
	if err != nil {
		return nil, err
	}
	out.ViewerIsFollowing = viewerIsFollowing
	out.Crafts = []string{}
	out.IsCraftskyProfile = false
	return out, nil
}

func (s *ProfileStore) viewerFollows(ctx context.Context, viewerDID string, profileDID string) (bool, error) {
	if viewerDID == "" || viewerDID == profileDID {
		return false, nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM atproto_follows
			WHERE did = $1 AND subject_did = $2
		)
	`, viewerDID, profileDID).Scan(&exists); err != nil {
		if strings.Contains(err.Error(), "atproto_follows") {
			return false, fmt.Errorf("%w: %v", ErrProfileCountsUnavailable, err)
		}
		return false, fmt.Errorf("profile viewer follow %s->%s: %w", viewerDID, profileDID, err)
	}
	return exists, nil
}

type hydratedBlueskyProfile struct {
	DisplayName *string                 `json:"displayName,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Avatar      *hydratedBlueskyBlobRef `json:"avatar,omitempty"`
	Banner      *hydratedBlueskyBlobRef `json:"banner,omitempty"`
}

type hydratedBlueskyBlobRef struct {
	Ref struct {
		Link string `json:"$link"`
	} `json:"ref"`
	MimeType string `json:"mimeType"`
}

func (s *ProfileStore) hydrateNonCraftsky(ctx context.Context, profileDID string) error {
	did, err := syntax.ParseDID(profileDID)
	if err != nil {
		return fmt.Errorf("hydrate profile parse did %s: %w", profileDID, err)
	}

	var rec map[string]any
	cid, err := s.blueskyHydrator.GetRecord(ctx, did, blueskyProfileNSID, profileRecordKey, &rec)
	if errors.Is(err, auth.ErrRecordNotFound) {
		return ErrProfileNotFound
	}
	if err != nil {
		return fmt.Errorf("hydrate bluesky profile %s: %w", profileDID, err)
	}

	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("hydrate bluesky profile marshal %s: %w", profileDID, err)
	}
	var parsed hydratedBlueskyProfile
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("hydrate bluesky profile decode %s: %w", profileDID, err)
	}

	var avatarCID, avatarMime, bannerCID, bannerMime *string
	if parsed.Avatar != nil && parsed.Avatar.Ref.Link != "" {
		avatarCID = &parsed.Avatar.Ref.Link
		avatarMime = &parsed.Avatar.MimeType
	}
	if parsed.Banner != nil && parsed.Banner.Ref.Link != "" {
		bannerCID = &parsed.Banner.Ref.Link
		bannerMime = &parsed.Banner.MimeType
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO bluesky_profiles
			(did, display_name, description,
			 avatar_cid, avatar_mime, banner_cid, banner_mime, record_cid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (did) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description  = EXCLUDED.description,
			avatar_cid   = EXCLUDED.avatar_cid,
			avatar_mime  = EXCLUDED.avatar_mime,
			banner_cid   = EXCLUDED.banner_cid,
			banner_mime  = EXCLUDED.banner_mime,
			record_cid   = EXCLUDED.record_cid,
			indexed_at   = now()
		WHERE bluesky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
	`, profileDID, parsed.DisplayName, parsed.Description, avatarCID, avatarMime, bannerCID, bannerMime, cid); err != nil {
		return fmt.Errorf("hydrate bluesky profile upsert %s: %w", profileDID, err)
	}
	return nil
}
