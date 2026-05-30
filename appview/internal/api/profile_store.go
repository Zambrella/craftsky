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

	"social.craftsky/appview/internal/api/envelope"
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
	DID                   string
	Crafts                []string
	CreatedAt             time.Time
	FollowerCount         *int
	FollowingCount        *int
	MutualFollowerCount   *int
	PostCount             *int
	PostsLast7Days        *int
	ProjectCount          *int
	ViewerIsFollowing     bool
	IsCraftskyProfile     bool
	DisplayName           *string
	Description           *string
	AvatarCID             *string
	AvatarMime            *string
	BannerCID             *string
	BannerMime            *string
	ModerationWarningKind *string
}

// ProfileAccountRow is the display-ready account summary used by social graph
// list endpoints before the handler resolves each DID to its current handle.
type ProfileAccountRow struct {
	DID               string
	DisplayName       *string
	Description       *string
	AvatarCID         *string
	AvatarMime        *string
	IsCraftskyProfile bool
	FollowCreatedAt   time.Time
	FollowURI         string
}

const profileVisibleModerationPredicate = `
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs mo
			WHERE mo.action = 'apply'
			  AND mo.subject_type = 'account'
			  AND mo.subject_did = cp.did
			  AND mo.value IN ('hide', 'takedown')
			  AND (mo.expires_at IS NULL OR mo.expires_at > now())
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs neg
				WHERE neg.action = 'negate'
				  AND neg.source_did = mo.source_did
				  AND neg.subject_type = mo.subject_type
				  AND neg.subject_did = mo.subject_did
				  AND neg.value = mo.value
				  AND (neg.expires_at IS NULL OR neg.expires_at > now())
				  AND neg.indexed_at > mo.indexed_at
			  )
		  )
`

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

// ResolveAccountReportTarget returns a canonical account DID snapshot for
// report eligibility. It checks indexed Craftsky profile existence without
// applying moderation visibility filters.
func (s *ProfileStore) ResolveAccountReportTarget(ctx context.Context, handleOrDID string) (*AccountReportTarget, error) {
	did, err := syntax.ParseDID(strings.TrimPrefix(handleOrDID, "@"))
	if err != nil {
		return nil, ErrProfileNotFound
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did.String()).Scan(&exists); err != nil {
		return nil, fmt.Errorf("profile report target %s: %w", did.String(), err)
	}
	if !exists {
		return nil, ErrProfileNotFound
	}
	return &AccountReportTarget{DID: did.String()}, nil
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
				WHEN $2 = '' OR $2 = cp.did THEN NULL
				ELSE (
					SELECT COUNT(*)
					FROM atproto_follows viewer_follow
					JOIN atproto_follows mutual_follow
					  ON mutual_follow.did = viewer_follow.subject_did
					WHERE viewer_follow.did = $2
					  AND mutual_follow.subject_did = cp.did
				)
			END AS mutual_follower_count,
			(
				SELECT COUNT(*)
				FROM craftsky_posts p
				WHERE p.did = cp.did
				  AND p.reply_root_uri IS NULL
				  AND p.reply_parent_uri IS NULL
			) AS post_count,
			(
				SELECT COUNT(*)
				FROM craftsky_posts p
				WHERE p.did = cp.did
				  AND p.reply_root_uri IS NULL
				  AND p.reply_parent_uri IS NULL
				  AND p.created_at >= now() - interval '7 days'
			) AS posts_last_7_days,
			0 AS project_count,
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
			bp.banner_cid, bp.banner_mime,
			CASE
				WHEN EXISTS (
					SELECT 1
					FROM moderation_outputs mo
					WHERE mo.action = 'apply'
					  AND mo.subject_type = 'account'
					  AND mo.subject_did = cp.did
					  AND mo.value = 'warn'
					  AND (mo.expires_at IS NULL OR mo.expires_at > now())
					  AND NOT EXISTS (
						SELECT 1
						FROM moderation_outputs neg
						WHERE neg.action = 'negate'
						  AND neg.source_did = mo.source_did
						  AND neg.subject_type = mo.subject_type
						  AND neg.subject_did = mo.subject_did
						  AND neg.value = mo.value
						  AND (neg.expires_at IS NULL OR neg.expires_at > now())
						  AND neg.indexed_at > mo.indexed_at
					  )
				) THEN 'profile'
				ELSE NULL
			END AS moderation_warning_kind
		FROM craftsky_profiles cp
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE cp.did = $1
		` + profileVisibleModerationPredicate + `
	`
	row := s.pool.QueryRow(ctx, q, profileDID, viewerDID)
	out := &ProfileRow{}
	var followerCount int
	var followingCount int
	var mutualFollowerCount *int
	var postCount int
	var postsLast7Days int
	var projectCount int
	err := row.Scan(
		&out.DID, &out.Crafts, &out.CreatedAt,
		&followerCount, &followingCount,
		&mutualFollowerCount,
		&postCount, &postsLast7Days, &projectCount,
		&out.ViewerIsFollowing,
		&out.DisplayName, &out.Description,
		&out.AvatarCID, &out.AvatarMime,
		&out.BannerCID, &out.BannerMime,
		&out.ModerationWarningKind,
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
	out.MutualFollowerCount = mutualFollowerCount
	out.PostCount = &postCount
	out.PostsLast7Days = &postsLast7Days
	out.ProjectCount = &projectCount
	out.IsCraftskyProfile = true
	return out, nil
}

// ListMutualFollowers returns accounts where viewerDID follows the account and
// that account follows profileDID, ordered by the mutual account's follow of the
// profile newest-first.
func (s *ProfileStore) ListMutualFollowers(ctx context.Context, viewerDID string, profileDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", 0, err
	}

	var total int
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM atproto_follows viewer_follow
		JOIN atproto_follows mutual_follow
		  ON mutual_follow.did = viewer_follow.subject_did
		WHERE viewer_follow.did = $1
		  AND mutual_follow.subject_did = $2
	`, viewerDID, profileDID).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("mutual follower count %s->%s: %w", viewerDID, profileDID, err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			viewer_follow.subject_did,
			bp.display_name,
			bp.description,
			bp.avatar_cid,
			bp.avatar_mime,
			(cp.did IS NOT NULL) AS is_craftsky_profile,
			mutual_follow.created_at,
			mutual_follow.uri
		FROM atproto_follows viewer_follow
		JOIN atproto_follows mutual_follow
		  ON mutual_follow.did = viewer_follow.subject_did
		LEFT JOIN bluesky_profiles bp ON bp.did = viewer_follow.subject_did
		LEFT JOIN craftsky_profiles cp ON cp.did = viewer_follow.subject_did
		WHERE viewer_follow.did = $1
		  AND mutual_follow.subject_did = $2
		  AND ($3::timestamptz IS NULL
		       OR (mutual_follow.created_at, mutual_follow.uri) < ($3::timestamptz, $4::text))
		ORDER BY mutual_follow.created_at DESC, mutual_follow.uri DESC
		LIMIT $5
	`, viewerDID, profileDID, curCreatedAt, curURI, limit)
	if err != nil {
		return nil, "", 0, fmt.Errorf("mutual follower list %s->%s: %w", viewerDID, profileDID, err)
	}
	defer rows.Close()

	out := make([]*ProfileAccountRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanProfileAccountRow(rows)
		if scanErr != nil {
			return nil, "", 0, fmt.Errorf("mutual follower scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("mutual follower iter: %w", err)
	}
	next, err := encodeProfileAccountCursor(out, limit)
	if err != nil {
		return nil, "", 0, err
	}
	return out, next, total, nil
}

// ListFollowers returns accounts that follow subjectDID, ordered by newest
// follow record first.
func (s *ProfileStore) ListFollowers(ctx context.Context, subjectDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error) {
	return s.listFollowAccounts(ctx, "followers", subjectDID, limit, cursor)
}

// ListFollowing returns accounts did follows, ordered by newest follow record
// first.
func (s *ProfileStore) ListFollowing(ctx context.Context, did string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error) {
	return s.listFollowAccounts(ctx, "following", did, limit, cursor)
}

func (s *ProfileStore) listFollowAccounts(ctx context.Context, kind string, did string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", 0, err
	}

	queryConfig := followAccountQueryConfig(kind)

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM atproto_follows f `+queryConfig.craftskyJoin+` WHERE `+queryConfig.whereExpr, did).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("%s count %s: %w", kind, did, err)
	}

	q := `
		SELECT
			` + queryConfig.accountExpr + ` AS account_did,
			bp.display_name,
			bp.description,
			bp.avatar_cid,
			bp.avatar_mime,
			(cp.did IS NOT NULL) AS is_craftsky_profile,
			f.created_at,
			f.uri
		FROM atproto_follows f
		` + queryConfig.craftskyJoin + `
		LEFT JOIN bluesky_profiles bp ON bp.did = ` + queryConfig.accountExpr + `
		LEFT JOIN craftsky_profiles cp ON cp.did = ` + queryConfig.accountExpr + `
		WHERE ` + queryConfig.whereExpr + `
		  AND ($2::timestamptz IS NULL OR (f.created_at, f.uri) < ($2::timestamptz, $3::text))
		ORDER BY f.created_at DESC, f.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, did, curCreatedAt, curURI, limit)
	if err != nil {
		return nil, "", 0, fmt.Errorf("%s list %s: %w", kind, did, err)
	}
	defer rows.Close()

	out := make([]*ProfileAccountRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanProfileAccountRow(rows)
		if scanErr != nil {
			return nil, "", 0, fmt.Errorf("%s scan: %w", kind, scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("%s iter: %w", kind, err)
	}
	next, err := encodeProfileAccountCursor(out, limit)
	if err != nil {
		return nil, "", 0, err
	}
	return out, next, total, nil
}

type followAccountQueryConfigSpec struct {
	accountExpr  string
	whereExpr    string
	craftskyJoin string
}

func followAccountQueryConfig(kind string) followAccountQueryConfigSpec {
	if kind == "following" {
		return followAccountQueryConfigSpec{
			accountExpr:  "f.subject_did",
			whereExpr:    "f.did = $1",
			craftskyJoin: "JOIN craftsky_profiles followed_cp ON followed_cp.did = f.subject_did",
		}
	}
	return followAccountQueryConfigSpec{
		accountExpr: "f.did",
		whereExpr:   "f.subject_did = $1",
	}
}

func scanProfileAccountRow(scanner pgx.Row) (*ProfileAccountRow, error) {
	out := &ProfileAccountRow{}
	err := scanner.Scan(
		&out.DID,
		&out.DisplayName,
		&out.Description,
		&out.AvatarCID,
		&out.AvatarMime,
		&out.IsCraftskyProfile,
		&out.FollowCreatedAt,
		&out.FollowURI,
	)
	return out, err
}

func encodeProfileAccountCursor(rows []*ProfileAccountRow, limit int) (string, error) {
	if len(rows) < limit {
		return "", nil
	}
	last := rows[len(rows)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.FollowCreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.FollowURI,
	})
	if err != nil {
		return "", fmt.Errorf("encode profile account cursor: %w", err)
	}
	return next, nil
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
