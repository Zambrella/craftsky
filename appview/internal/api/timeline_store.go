// appview/internal/api/timeline_store.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

// TimelineFeedItemRow is a mixed home-timeline row. Authored posts and
// straight repost activity both carry a post; only repost items have Repost.
type TimelineFeedItemRow struct {
	ItemKind   string
	ItemKey    string
	ActivityAt time.Time
	Post       *PostRow
	Repost     *TimelineRepostReasonRow
}

// TimelineRepostReasonRow is the store-level attribution for a straight
// repost timeline item.
type TimelineRepostReasonRow struct {
	URI               string
	CID               string
	DID               string
	CreatedAt         time.Time
	IndexedAt         time.Time
	AuthorDisplayName *string
	AuthorAvatarCID   *string
	AuthorAvatarMime  *string
}

// ListTimeline returns a viewer's home timeline from indexed AppView rows.
// Eligible authored posts and straight repost activity are authored by the
// viewer or by accounts actively followed by the viewer, are ordered by AppView
// activity chronology, and use ItemKey as the stable feed-item identity.
func (s *PostStore) ListTimeline(ctx context.Context, viewerDID string, limit int, cursor string) ([]*TimelineFeedItemRow, string, error) {
	var rows []*TimelineFeedItemRow
	var nextCursor string
	err := s.observeDB(ctx, "feed.timeline", "/v1/feed/timeline", func(ctx context.Context) error {
		var err error
		rows, nextCursor, err = s.listTimelineObserved(ctx, viewerDID, limit, cursor)
		return err
	})
	return rows, nextCursor, err
}

func (s *PostStore) listTimelineObserved(ctx context.Context, viewerDID string, limit int, cursor string) ([]*TimelineFeedItemRow, string, error) {
	curActivityAt, curItemKey, err := decodeTimelineCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	q := `
		WITH eligible_authors AS (
			SELECT $1::text AS did
			UNION
			SELECT f.subject_did
			FROM atproto_follows f
			JOIN craftsky_profiles followed_cp ON followed_cp.did = f.subject_did
			WHERE f.did = $1
			  AND NOT EXISTS (
				SELECT 1 FROM actor_mutes m
				WHERE m.owner_did = $1 AND m.subject_did = f.subject_did
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM atproto_blocks b
				WHERE (b.blocker_did = $1 AND b.subject_did = f.subject_did)
				   OR (b.blocker_did = f.subject_did AND b.subject_did = $1)
			  )
		), feed AS (
			SELECT
				'post'::text AS item_kind,
				'post:' || p.uri AS item_key,
				p.indexed_at AS activity_at,
				p.uri AS post_uri,
				NULL::text AS repost_uri,
				NULL::text AS repost_cid,
				NULL::text AS repost_did,
				NULL::timestamptz AS repost_created_at,
				NULL::timestamptz AS repost_indexed_at
			FROM craftsky_posts p
			JOIN eligible_authors a ON a.did = p.did
			WHERE p.reply_root_uri IS NULL
			  AND p.reply_parent_uri IS NULL
			` + postVisibleModerationPredicate + `

			UNION ALL

			SELECT
				'repost'::text AS item_kind,
				'repost:' || r.uri AS item_key,
				r.indexed_at AS activity_at,
				r.subject_uri AS post_uri,
				r.uri AS repost_uri,
				r.cid AS repost_cid,
				r.did AS repost_did,
				r.created_at AS repost_created_at,
				r.indexed_at AS repost_indexed_at
			FROM craftsky_reposts r
			JOIN eligible_authors a ON a.did = r.did
			JOIN craftsky_posts p ON p.uri = r.subject_uri
			JOIN craftsky_profiles subject_cp ON subject_cp.did = p.did
			WHERE r.deleted_at IS NULL
			  AND p.reply_root_uri IS NULL
			  AND p.reply_parent_uri IS NULL
			  AND NOT EXISTS (
				SELECT 1 FROM actor_mutes m
				WHERE m.owner_did = $1 AND m.subject_did = p.did
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM atproto_blocks b
				WHERE (b.blocker_did = $1 AND b.subject_did = p.did)
				   OR (b.blocker_did = p.did AND b.subject_did = $1)
			  )
			` + postVisibleModerationPredicate + `
		)
		SELECT
			feed.item_kind, feed.item_key, feed.activity_at,
			feed.repost_uri, feed.repost_cid, feed.repost_did,
			feed.repost_created_at, feed.repost_indexed_at,
			rbp.display_name, rbp.avatar_cid, rbp.avatar_mime,
			` + postSelectColumns + `
		FROM feed
		JOIN craftsky_posts p ON p.uri = feed.post_uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		LEFT JOIN bluesky_profiles rbp ON rbp.did = feed.repost_did
		WHERE ($2::timestamptz IS NULL
		       OR (feed.activity_at, feed.item_key) < ($2::timestamptz, $3::text))
		ORDER BY feed.activity_at DESC, feed.item_key DESC
		LIMIT $4
	`
	queryLimit := limit + 1
	rows, err := s.pool.Query(ctx, q, viewerDID, curActivityAt, curItemKey, queryLimit)
	if err != nil {
		return nil, "", fmt.Errorf("timeline list %s: %w", viewerDID, err)
	}
	defer rows.Close()

	out := make([]*TimelineFeedItemRow, 0, queryLimit)
	for rows.Next() {
		row, scanErr := scanTimelineFeedItemRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("timeline list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("timeline list iter: %w", err)
	}
	if len(out) <= limit {
		return out, "", nil
	}
	out = out[:limit]
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"activityAt": last.ActivityAt.UTC().Format(time.RFC3339Nano),
		"itemKey":    last.ItemKey,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode timeline cursor: %w", err)
	}
	return out, next, nil
}

func decodeTimelineCursor(cursor string) (any, any, error) {
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	if cursor == "" {
		return nil, nil, nil
	}
	if len(cur) != 2 {
		return nil, nil, envelope.ErrInvalidCursor
	}
	timeValue, ok := cur["activityAt"].(string)
	if !ok || timeValue == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, timeValue)
	if err != nil {
		return nil, nil, envelope.ErrInvalidCursor
	}
	itemKey, ok := cur["itemKey"].(string)
	if !ok || itemKey == "" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	return parsedTime, itemKey, nil
}

func scanTimelineFeedItemRow(scanner interface{ Scan(...any) error }) (*TimelineFeedItemRow, error) {
	out := &TimelineFeedItemRow{Post: &PostRow{}}
	var repostURI, repostCID, repostDID *string
	var repostCreatedAt, repostIndexedAt *time.Time
	var repostAuthorDisplayName, repostAuthorAvatarCID, repostAuthorAvatarMime *string
	var rawProject *json.RawMessage
	err := scanner.Scan(
		&out.ItemKind, &out.ItemKey, &out.ActivityAt,
		&repostURI, &repostCID, &repostDID,
		&repostCreatedAt, &repostIndexedAt,
		&repostAuthorDisplayName, &repostAuthorAvatarCID, &repostAuthorAvatarMime,
		&out.Post.URI, &out.Post.DID, &out.Post.Rkey, &out.Post.CID, &out.Post.Text, &out.Post.Facets, &out.Post.Images,
		&out.Post.ReplyRootURI, &out.Post.ReplyRootCID, &out.Post.ReplyParentURI, &out.Post.ReplyParentCID,
		&out.Post.QuoteURI, &out.Post.QuoteCID, &out.Post.Tags, &out.Post.CreatedAt, &out.Post.IndexedAt,
		&out.Post.IsProject, &out.Post.ProjectCraftType, &rawProject,
		&out.Post.AuthorDisplayName, &out.Post.AuthorAvatarCID, &out.Post.AuthorAvatarMime,
		&out.Post.ModerationWarningKind,
	)
	if err != nil {
		return out, err
	}
	if repostURI != nil && repostCID != nil && repostDID != nil && repostCreatedAt != nil && repostIndexedAt != nil {
		out.Repost = &TimelineRepostReasonRow{
			URI:               *repostURI,
			CID:               *repostCID,
			DID:               *repostDID,
			CreatedAt:         repostCreatedAt.UTC(),
			IndexedAt:         repostIndexedAt.UTC(),
			AuthorDisplayName: repostAuthorDisplayName,
			AuthorAvatarCID:   repostAuthorAvatarCID,
			AuthorAvatarMime:  repostAuthorAvatarMime,
		}
	}
	out.ActivityAt = out.ActivityAt.UTC()
	if rawProject != nil && len(*rawProject) > 0 {
		out.Post.RawProject = append(json.RawMessage(nil), (*rawProject)...)
		var project Project
		if err := json.Unmarshal(*rawProject, &project); err != nil {
			return out, err
		}
		out.Post.Project = &project
	}
	return out, nil
}
