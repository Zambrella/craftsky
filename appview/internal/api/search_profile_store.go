package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type ProfileSearchRow struct {
	DID               string
	Handle            string
	HandleLower       string
	DisplayName       *string
	Description       *string
	AvatarCID         *string
	AvatarMime        *string
	Crafts            []string
	IsCraftskyProfile bool
	ViewerIsFollowing bool
	Muted             bool
	Blocking          bool
	BlockedBy         bool
	FollowedRank      int
	RelevanceRank     int
}

func (s *SearchStore) SearchProfiles(ctx context.Context, viewerDID string, req ProfileSearchRequest) ([]ProfileSearchRow, string, error) {
	var rows []ProfileSearchRow
	var cursor string
	err := s.observeDB(ctx, "search.profiles", "/v1/search/profiles", func(ctx context.Context) error {
		var err error
		rows, cursor, err = s.searchProfilesObserved(ctx, viewerDID, req)
		return err
	})
	return rows, cursor, err
}

func (s *SearchStore) searchProfilesObserved(ctx context.Context, viewerDID string, req ProfileSearchRequest) ([]ProfileSearchRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	queryLower := strings.ToLower(strings.TrimSpace(req.Query))
	cur, err := DecodeProfileSearchCursor(req.Cursor)
	if err != nil {
		return nil, "", err
	}
	// Profile search is deliberately stale-tolerant presentation. The bounded
	// identity refresh worker repairs rows older than the authoritative 24-hour
	// target; no durable action uses this result without a fresh lookup.
	q := `
		WITH ranked AS (
		SELECT cp.did, ic.handle, ic.handle_lower, bp.display_name, bp.description, bp.avatar_cid, bp.avatar_mime, cp.crafts,
			true AS is_craftsky_profile,
			EXISTS (
				SELECT 1 FROM atproto_follows f
				WHERE f.did = $2 AND f.subject_did = cp.did
				  AND NOT appview_owner_is_terminal(f.did)
				  AND NOT appview_owner_is_terminal(f.subject_did)
				  AND NOT EXISTS (
					SELECT 1 FROM atproto_blocks b
					WHERE ((b.blocker_did = $2 AND b.subject_did = cp.did)
					   OR (b.blocker_did = cp.did AND b.subject_did = $2))
					  AND NOT appview_owner_is_terminal(b.blocker_did)
					  AND NOT appview_owner_is_terminal(b.subject_did)
				  )
			) AS viewer_is_following,
			EXISTS (SELECT 1 FROM actor_mutes m WHERE m.owner_did = $2 AND m.subject_did = cp.did
			        AND NOT appview_owner_is_terminal(m.owner_did) AND NOT appview_owner_is_terminal(m.subject_did)) AS muted,
			EXISTS (SELECT 1 FROM atproto_blocks b WHERE b.blocker_did = $2 AND b.subject_did = cp.did
			        AND NOT appview_owner_is_terminal(b.blocker_did) AND NOT appview_owner_is_terminal(b.subject_did)) AS blocking,
			EXISTS (SELECT 1 FROM atproto_blocks b WHERE b.blocker_did = cp.did AND b.subject_did = $2
			        AND NOT appview_owner_is_terminal(b.blocker_did) AND NOT appview_owner_is_terminal(b.subject_did)) AS blocked_by,
			CASE WHEN EXISTS (SELECT 1 FROM atproto_follows f WHERE f.did = $2 AND f.subject_did = cp.did
			                 AND NOT appview_owner_is_terminal(f.did) AND NOT appview_owner_is_terminal(f.subject_did)) THEN 0 ELSE 1 END AS followed_rank,
			CASE
				WHEN ic.handle_lower = $1 THEN 0
				WHEN ic.handle_lower LIKE $1 || '%' THEN 1
				WHEN ic.handle_lower LIKE '%' || $1 || '%' THEN 2
				WHEN lower(coalesce(bp.display_name, '')) LIKE '%' || $1 || '%' THEN 3
				WHEN lower(coalesce(bp.description, '')) LIKE '%' || $1 || '%' THEN 4
				ELSE 99
			END AS relevance_rank
		FROM craftsky_profiles cp
		JOIN atproto_identity_cache ic ON ic.did = cp.did
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE (
			ic.handle_lower LIKE '%' || $1 || '%'
			OR lower(coalesce(bp.display_name, '')) LIKE '%' || $1 || '%'
			OR lower(coalesce(bp.description, '')) LIKE '%' || $1 || '%'
		)
		AND (
			ic.handle_lower = $1
			OR NOT EXISTS (
				SELECT 1 FROM atproto_blocks b
				WHERE ((b.blocker_did = $2 AND b.subject_did = cp.did)
				   OR (b.blocker_did = cp.did AND b.subject_did = $2))
				  AND NOT appview_owner_is_terminal(b.blocker_did)
				  AND NOT appview_owner_is_terminal(b.subject_did)
			)
		)
		` + profileVisibleModerationPredicate + `
		)
		SELECT did, handle, handle_lower, display_name, description, avatar_cid, avatar_mime, crafts, is_craftsky_profile, viewer_is_following, muted, blocking, blocked_by, followed_rank, relevance_rank
		FROM ranked
		WHERE ($4::int IS NULL OR (followed_rank, relevance_rank, handle_lower, did) > ($4::int, $5::int, $6::text, $7::text))
		ORDER BY followed_rank ASC, relevance_rank ASC, handle_lower ASC, did ASC
		LIMIT $3`
	rows, err := s.pool.Query(ctx, q, queryLower, viewerDID, req.Limit+1, profileCursorArg(cur, "followed"), profileCursorArg(cur, "relevance"), profileCursorArg(cur, "handle"), profileCursorArg(cur, "did"))
	if err != nil {
		return nil, "", fmt.Errorf("profile search: %w", err)
	}
	defer rows.Close()
	out := make([]ProfileSearchRow, 0, req.Limit+1)
	for rows.Next() {
		var row ProfileSearchRow
		if err := rows.Scan(&row.DID, &row.Handle, &row.HandleLower, &row.DisplayName, &row.Description, &row.AvatarCID, &row.AvatarMime, &row.Crafts, &row.IsCraftskyProfile, &row.ViewerIsFollowing, &row.Muted, &row.Blocking, &row.BlockedBy, &row.FollowedRank, &row.RelevanceRank); err != nil {
			return nil, "", err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(out) <= req.Limit {
		return out, "", nil
	}
	out = out[:req.Limit]
	last := out[len(out)-1]
	next, err := EncodeProfileSearchCursor(last.FollowedRank, last.RelevanceRank, last.HandleLower, last.DID)
	return out, next, err
}

func profileCursorArg(cur ProfileCursor, field string) any {
	if cur.DID == "" {
		return nil
	}
	switch field {
	case "followed":
		return cur.FollowedRank
	case "relevance":
		return cur.RelevanceRank
	case "handle":
		return cur.HandleLower
	case "did":
		return cur.DID
	default:
		return nil
	}
}

func BuildProfileSearchSummary(row ProfileSearchRow) ProfileSearchSummary {
	out := ProfileSearchSummary{
		ProfileAccountSummary: ProfileAccountSummary{
			DID:               syntax.DID(row.DID),
			Handle:            syntax.Handle(row.Handle),
			DisplayName:       row.DisplayName,
			Description:       row.Description,
			IsCraftskyProfile: row.IsCraftskyProfile,
			Muted:             row.Muted,
			Blocking:          row.Blocking,
			BlockedBy:         row.BlockedBy,
		},
		ViewerIsFollowing: row.ViewerIsFollowing,
		Crafts:            append([]string(nil), row.Crafts...),
	}
	if row.IsCraftskyProfile {
		value := DefaultProfileCustomisation
		out.Customisation = &value
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	return out
}
