package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

// ListExtendedNetworkWithLanguages returns original posts by second-degree
// accounts in the indexed CraftSky graph. Membership is evaluated before the
// page limit, and each post contributes once regardless of shared paths.
func (s *PostStore) ListExtendedNetworkWithLanguages(ctx context.Context, viewerDID string, contentLanguages []string, limit int, cursor string) ([]*TimelineFeedItemRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	var items []*TimelineFeedItemRow
	var next string
	err := s.observeDB(ctx, "feed.extended_network", "/v1/feed/extended-network", func(ctx context.Context) error {
		var err error
		items, next, err = s.listExtendedNetworkObserved(ctx, viewerDID, contentLanguages, limit, cursor)
		return err
	})
	return items, next, err
}

func (s *PostStore) listExtendedNetworkObserved(ctx context.Context, viewerDID string, contentLanguages []string, limit int, cursor string) ([]*TimelineFeedItemRow, string, error) {
	activityAt, itemKey, err := decodeExtendedNetworkCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	// The graph is local: intermediaries must be tracked CraftSky members.
	// Collapse shared paths by author before joining posts, so sparse networks
	// do not require scanning unrelated post history.
	q := extendedNetworkQuery()
	rows, err := s.pool.Query(ctx, q, viewerDID, contentLanguages, activityAt, itemKey, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("extended network list: %w", err)
	}
	defer rows.Close()
	out := make([]*TimelineFeedItemRow, 0, limit+1)
	for rows.Next() {
		item := &TimelineFeedItemRow{ItemKind: "post", FollowedBy: &TimelineFollowedByRow{}}
		by := item.FollowedBy
		item.Post, err = scanPostRowWithExtra(rows, &item.ActivityAt, &by.DID, &by.DisplayName, &by.AvatarCID, &by.AvatarMime)
		if err != nil {
			return nil, "", fmt.Errorf("extended network scan: %w", err)
		}
		item.ItemKey = "post:" + item.Post.URI
		item.ActivityAt = item.ActivityAt.UTC()
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("extended network rows: %w", err)
	}
	if len(out) <= limit {
		return out, "", nil
	}
	out = out[:limit]
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{"feed": "extended-network", "activityAt": last.ActivityAt.Format(time.RFC3339Nano), "itemKey": last.ItemKey})
	if err != nil {
		return nil, "", fmt.Errorf("extended network cursor: %w", err)
	}
	return out, next, nil
}

func decodeExtendedNetworkCursor(cursor string) (any, any, error) {
	if cursor == "" {
		return nil, nil, nil
	}
	cur, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	if len(cur) != 3 || cur["feed"] != "extended-network" {
		return nil, nil, envelope.ErrInvalidCursor
	}
	key, ok := cur["itemKey"].(string)
	if !ok || !strings.HasPrefix(key, "post:at://") {
		return nil, nil, envelope.ErrInvalidCursor
	}
	delete(cur, "feed")
	return decodeTimelineCursorPayload(cur)
}

func extendedNetworkQuery() string {
	return `
        WITH connections AS MATERIALIZED (
            SELECT f.subject_did AS did
            FROM pds_set_aggregates f
            JOIN craftsky_profiles cp ON cp.did = f.subject_did
            WHERE f.kind = 'follow' AND f.actor_did = $1
              AND f.subject_did <> $1
              AND NOT appview_owner_is_terminal(f.actor_did)
              AND NOT appview_owner_is_terminal(f.subject_did)
              AND NOT EXISTS (SELECT 1 FROM actor_mutes m WHERE m.owner_did = $1 AND m.subject_did = f.subject_did)
              AND NOT EXISTS (
                SELECT 1 FROM pds_set_aggregates b WHERE b.kind = 'block'
                AND ((b.actor_did = $1 AND b.subject_did = f.subject_did)
                  OR (b.actor_did = f.subject_did AND b.subject_did = $1))
              )
              ` + profileVisibleModerationPredicate + `
        ), candidate_authors AS MATERIALIZED (
            SELECT f.subject_did AS did, min(c.did) AS followed_by_did
            FROM connections c
            JOIN pds_set_aggregates f ON f.actor_did = c.did AND f.kind = 'follow'
            WHERE EXISTS (SELECT 1 FROM connections)
              AND f.subject_did <> $1
              AND NOT EXISTS (
                SELECT 1 FROM pds_set_aggregates b WHERE b.kind = 'block'
                AND ((b.actor_did = c.did AND b.subject_did = f.subject_did)
                  OR (b.actor_did = f.subject_did AND b.subject_did = c.did))
              )
            GROUP BY f.subject_did
        ), eligible_authors AS MATERIALIZED (
            SELECT a.did, a.followed_by_did
            FROM candidate_authors a
            JOIN craftsky_profiles cp ON cp.did = a.did
            WHERE NOT EXISTS (
                SELECT 1 FROM pds_set_aggregates direct
                WHERE direct.kind = 'follow' AND direct.actor_did = $1 AND direct.subject_did = a.did
            )
              AND NOT EXISTS (SELECT 1 FROM actor_mutes m WHERE m.owner_did = $1 AND m.subject_did = a.did)
              AND NOT EXISTS (
                SELECT 1 FROM pds_set_aggregates b WHERE b.kind = 'block'
                AND ((b.actor_did = $1 AND b.subject_did = a.did)
                  OR (b.actor_did = a.did AND b.subject_did = $1))
              )
              ` + profileVisibleModerationPredicate + `
        )
        SELECT ` + postSelectColumns + `,
            LEAST(p.created_at, p.indexed_at, p.profile_sort_at), a.followed_by_did,
            fb.display_name, fb.avatar_cid, fb.avatar_mime
        FROM eligible_authors a
        JOIN craftsky_posts p ON p.did = a.did
        LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
        LEFT JOIN bluesky_profiles bp ON bp.did = p.did
        LEFT JOIN bluesky_profiles fb ON fb.did = a.followed_by_did
        WHERE EXISTS (SELECT 1 FROM eligible_authors)
          AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
          AND p.external_import_source IS DISTINCT FROM 'instagram'
          ` + postVisibleModerationPredicate + `
          ` + languageVisibilityPredicate("p", "$1", "$2") + `
          AND ($3::timestamptz IS NULL OR
            (LEAST(p.created_at, p.indexed_at, p.profile_sort_at), p.uri) < ($3::timestamptz, substring($4::text FROM 6)))
        ORDER BY LEAST(p.created_at, p.indexed_at, p.profile_sort_at) DESC, p.uri DESC
        LIMIT $5
    `
}
