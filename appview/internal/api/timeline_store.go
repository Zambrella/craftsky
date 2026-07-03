// appview/internal/api/timeline_store.go
package api

import (
	"context"
	"fmt"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

// ListTimeline returns a viewer's basic home timeline from indexed AppView
// rows. Eligible posts are authored by the viewer or by accounts actively
// followed by the viewer, are top-level Craftsky post rows, and are ordered by
// AppView index chronology.
func (s *PostStore) ListTimeline(ctx context.Context, viewerDID string, limit int, cursor string) ([]*PostRow, string, error) {
	var rows []*PostRow
	var nextCursor string
	err := s.observeDB(ctx, "feed.timeline", "/v1/feed/timeline", func(ctx context.Context) error {
		var err error
		rows, nextCursor, err = s.listTimelineObserved(ctx, viewerDID, limit, cursor)
		return err
	})
	return rows, nextCursor, err
}

func (s *PostStore) listTimelineObserved(ctx context.Context, viewerDID string, limit int, cursor string) ([]*PostRow, string, error) {
	curIndexedAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND (
			p.did = $1
			OR EXISTS (
				SELECT 1
				FROM atproto_follows f
				WHERE f.did = $1 AND f.subject_did = p.did
			)
		  )
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.indexed_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.indexed_at DESC, p.uri DESC
		LIMIT $4
	`
	queryLimit := limit + 1
	rows, err := s.pool.Query(ctx, q, viewerDID, curIndexedAt, curURI, queryLimit)
	if err != nil {
		return nil, "", fmt.Errorf("timeline list %s: %w", viewerDID, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, queryLimit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
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
		"indexedAt": last.IndexedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode timeline cursor: %w", err)
	}
	return out, next, nil
}
