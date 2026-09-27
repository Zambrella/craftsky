// appview/internal/api/post_author_feed_store.go
package api

import (
	"context"
	"fmt"
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

// ListByAuthor returns up to limit posts authored by did, ordered by
// (profile_sort_at DESC, uri DESC), starting after the cursor if non-empty.
// Returns the encoded next-page cursor when the result is full; empty
// string when this is the final page.
func (s *PostStore) ListByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	return s.ListByAuthorWithLanguages(ctx, "", did, []string{}, limit, cursor)
}

func (s *PostStore) ListByAuthorWithLanguages(
	ctx context.Context,
	viewerDID string,
	authorDID string,
	contentLanguages []string,
	limit int,
	cursor string,
) ([]*PostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	curProfileSortAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.is_project = false
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		` + postVisibleModerationPredicate + `
		` + languageVisibilityPredicate("p", "$5", "$6") + `
		  AND ($2::timestamptz IS NULL
		       OR (p.profile_sort_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.profile_sort_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, authorDID, curProfileSortAt, curURI, limit, viewerDID, contentLanguages)
	if err != nil {
		return nil, "", fmt.Errorf("post list %s: %w", authorDID, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("post list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("post list iter: %w", err)
	}

	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.ProfileSortAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode cursor: %w", err)
	}
	return out, next, nil
}

// ListProjectsByAuthor returns root project posts authored by did, ordered by
// (profile_sort_at DESC, uri DESC), starting after the cursor if non-empty.
func (s *PostStore) ListProjectsByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	return s.ListProjectsByAuthorWithLanguages(ctx, "", did, []string{}, limit, cursor)
}

func (s *PostStore) ListProjectsByAuthorWithLanguages(
	ctx context.Context,
	viewerDID string,
	authorDID string,
	contentLanguages []string,
	limit int,
	cursor string,
) ([]*PostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	curProfileSortAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.is_project = true
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND p.quote_uri IS NULL
		` + postVisibleModerationPredicate + `
		` + languageVisibilityPredicate("p", "$5", "$6") + `
		  AND ($2::timestamptz IS NULL
		       OR (p.profile_sort_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.profile_sort_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, authorDID, curProfileSortAt, curURI, limit, viewerDID, contentLanguages)
	if err != nil {
		return nil, "", fmt.Errorf("project list %s: %w", authorDID, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("project list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("project list iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.ProfileSortAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode project cursor: %w", err)
	}
	return out, next, nil
}

// ListCommentsByAuthor returns authored comments and nested replies, ordered by
// (profile_sort_at DESC, uri DESC), starting after the cursor if non-empty.
func (s *PostStore) ListCommentsByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	return s.ListCommentsByAuthorWithLanguages(ctx, "", did, []string{}, limit, cursor)
}

func (s *PostStore) ListCommentsByAuthorWithLanguages(
	ctx context.Context,
	viewerDID string,
	authorDID string,
	contentLanguages []string,
	limit int,
	cursor string,
) ([]*PostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	curProfileSortAt, curURI, err := decodeSeekCursor(cursor, "indexedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1
		  AND p.reply_root_uri IS NOT NULL
		  AND p.reply_parent_uri IS NOT NULL
		` + postVisibleModerationPredicate + `
		` + languageVisibilityPredicate("p", "$5", "$6") + `
		  AND ($2::timestamptz IS NULL
		       OR (p.profile_sort_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY p.profile_sort_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, authorDID, curProfileSortAt, curURI, limit, viewerDID, contentLanguages)
	if err != nil {
		return nil, "", fmt.Errorf("comment list author %s: %w", authorDID, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("comment list author scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("comment list author iter: %w", err)
	}

	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.ProfileSortAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode comment author cursor: %w", err)
	}
	return out, next, nil
}

// ListRepostsByAuthor returns the active logical repost subjects for an actor,
// ordered by (repost activated_at DESC, subject post URI DESC).
func (s *PostStore) ListRepostsByAuthor(ctx context.Context, did string, limit int, cursor string) ([]*PostRow, string, error) {
	return s.ListRepostsByAuthorWithLanguages(ctx, "", did, []string{}, limit, cursor)
}

func (s *PostStore) ListRepostsByAuthorWithLanguages(
	ctx context.Context,
	viewerDID string,
	authorDID string,
	contentLanguages []string,
	limit int,
	cursor string,
) ([]*PostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	curActivatedAt, curURI, err := decodeSeekCursor(cursor, "activatedAt")
	if err != nil {
		return nil, "", err
	}

	q := `
		SELECT ` + postSelectColumns + `, repost.activated_at
		FROM pds_set_aggregates repost
		JOIN craftsky_posts p ON p.uri = repost.subject_uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE repost.kind = 'repost'
		  AND repost.actor_did = $1
		  ` + postVisibleModerationPredicate + `
		  ` + languageVisibilityPredicate("p", "$5", "$6") + `
		  AND NOT ` + postAuthorBlockedPredicate("p", "$5") + `
		  AND ($2::timestamptz IS NULL
		       OR (repost.activated_at, p.uri) < ($2::timestamptz, $3::text))
		ORDER BY repost.activated_at DESC, p.uri DESC
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, authorDID, curActivatedAt, curURI, limit, viewerDID, contentLanguages)
	if err != nil {
		return nil, "", fmt.Errorf("repost list author %s: %w", authorDID, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	activatedAt := make([]time.Time, 0, limit)
	for rows.Next() {
		var activated time.Time
		row, scanErr := scanPostRowWithExtra(rows, &activated)
		if scanErr != nil {
			return nil, "", fmt.Errorf("repost list author scan: %w", scanErr)
		}
		out = append(out, row)
		activatedAt = append(activatedAt, activated)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("repost list author iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := len(out) - 1
	next, err := envelope.EncodeCursor(map[string]any{
		"activatedAt": activatedAt[last].UTC().Format(time.RFC3339Nano),
		"uri":         out[last].URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode repost author cursor: %w", err)
	}
	return out, next, nil
}
