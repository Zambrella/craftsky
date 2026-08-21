// appview/internal/api/post_conversation_store.go
package api

import (
	"context"
	"fmt"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// ListRootComments returns direct replies to the root post in comment-section
// render order: viewer-authored comments first, then the selected sort within
// each group. The follows sort currently uses oldest-first ordering.
func (s *PostStore) ListRootComments(ctx context.Context, rootURI, viewerDID, sortValue string, limit int, cursor string) ([]*PostRow, string, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", err
	}
	orderDirection := "ASC"
	seekComparator := ">"
	if sortValue == "newest" {
		orderDirection = "DESC"
		seekComparator = "<"
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.reply_parent_uri = $1
		  AND NOT ` + postAuthorBlockedPredicate("p", "$5") + `
		  AND NOT ` + postReplyAuthorBlockedPredicate("p") + `
		  AND NOT ` + postMentionAuthorBlockedPredicate("p") + `
		` + postVisibleModerationPredicate + `
		  AND ($2::timestamptz IS NULL
		       OR (p.created_at, p.uri) ` + seekComparator + ` ($2::timestamptz, $3::text))
		ORDER BY CASE WHEN p.did = $5 THEN 0 ELSE 1 END ASC, p.created_at ` + orderDirection + `, p.uri ` + orderDirection + `
		LIMIT $4
	`
	rows, err := s.pool.Query(ctx, q, rootURI, curCreatedAt, curURI, limit, viewerDID)
	if err != nil {
		return nil, "", fmt.Errorf("comment list root %s: %w", rootURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("comment list scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("comment list iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode comment cursor: %w", err)
	}
	return out, next, nil
}

func (s *PostStore) commentBranchHasRepliesAfter(ctx context.Context, commentURI, rootURI string, createdAt time.Time, uri string) (bool, error) {
	viewerDID, _ := middleware.GetDID(ctx)
	q := `
		WITH RECURSIVE branch(uri, created_at, depth, protected, muted_ancestor_uri) AS (
			SELECT p.uri, p.created_at, 1,
				` + postAuthorBlockedPredicate("p", "$5") + ` OR ` + postReplyAuthorBlockedPredicate("p") + ` OR ` + postMentionAuthorBlockedPredicate("p") + `,
				CASE WHEN ` + postAuthorMutedPredicate("p", "$5") + ` THEN p.uri END
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1,
				parent.protected OR ` + postAuthorBlockedPredicate("child", "$5") + ` OR ` + postReplyAuthorBlockedPredicate("child") + ` OR ` + postMentionAuthorBlockedPredicate("child") + `,
				COALESCE(
					parent.muted_ancestor_uri,
					CASE WHEN ` + postAuthorMutedPredicate("child", "$5") + ` THEN child.uri END
				)
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		)
		SELECT EXISTS (
			SELECT 1
			FROM branch
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE (branch.created_at, branch.uri) > ($3::timestamptz, $4::text)
			  AND NOT branch.protected
			  AND (branch.muted_ancestor_uri IS NULL OR branch.muted_ancestor_uri = branch.uri)
			` + postVisibleModerationPredicate + `
		)
	`
	var hasMore bool
	if err := s.pool.QueryRow(ctx, q, commentURI, rootURI, createdAt, uri, viewerDID).Scan(&hasMore); err != nil {
		return false, fmt.Errorf("reply list branch has more comment=%s root=%s: %w", commentURI, rootURI, err)
	}
	return hasMore, nil
}

// ListCommentBranchReplies returns visual replies under a top-level comment,
// including deeper descendants flattened into chronological branch order.
func (s *PostStore) ListCommentBranchReplies(ctx context.Context, commentURI, rootURI string, limit int, cursor string) ([]*PostRow, string, error) {
	curCreatedAt, curURI, err := decodeSeekCursor(cursor, "createdAt")
	if err != nil {
		return nil, "", err
	}

	viewerDID, _ := middleware.GetDID(ctx)
	q := `
		WITH RECURSIVE branch(uri, created_at, depth, protected, muted_ancestor_uri) AS (
			SELECT p.uri, p.created_at, 1,
				` + postAuthorBlockedPredicate("p", "$6") + ` OR ` + postReplyAuthorBlockedPredicate("p") + ` OR ` + postMentionAuthorBlockedPredicate("p") + `,
				CASE WHEN ` + postAuthorMutedPredicate("p", "$6") + ` THEN p.uri END
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1,
				parent.protected OR ` + postAuthorBlockedPredicate("child", "$6") + ` OR ` + postReplyAuthorBlockedPredicate("child") + ` OR ` + postMentionAuthorBlockedPredicate("child") + `,
				COALESCE(
					parent.muted_ancestor_uri,
					CASE WHEN ` + postAuthorMutedPredicate("child", "$6") + ` THEN child.uri END
				)
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		), page AS (
			SELECT branch.uri, branch.created_at
			FROM branch
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE ($3::timestamptz IS NULL
			       OR (branch.created_at, branch.uri) > ($3::timestamptz, $4::text))
			  AND NOT branch.protected
			  AND (branch.muted_ancestor_uri IS NULL OR branch.muted_ancestor_uri = branch.uri)
			` + postVisibleModerationPredicate + `
			ORDER BY branch.created_at ASC, branch.uri ASC
			LIMIT $5
		)
		SELECT ` + postSelectColumns + `
		FROM page
		JOIN craftsky_posts p ON p.uri = page.uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE true
		` + postVisibleModerationPredicate + `
		ORDER BY page.created_at ASC, page.uri ASC
	`
	rows, err := s.pool.Query(ctx, q, commentURI, rootURI, curCreatedAt, curURI, limit, viewerDID)
	if err != nil {
		return nil, "", fmt.Errorf("reply list branch comment=%s root=%s: %w", commentURI, rootURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("reply list branch scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reply list branch iter: %w", err)
	}
	if len(out) < limit {
		return out, "", nil
	}
	last := out[len(out)-1]
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode reply cursor: %w", err)
	}
	return out, next, nil
}

// ListCommentBranchRepliesAround returns a bounded visual reply page that
// includes focusURI, ending at the focused reply so deep links can render the
// target without loading every earlier branch reply.
func (s *PostStore) ListCommentBranchRepliesAround(ctx context.Context, commentURI, rootURI, focusURI string, limit int) ([]*PostRow, string, error) {
	viewerDID, _ := middleware.GetDID(ctx)
	q := `
		WITH RECURSIVE branch(uri, created_at, depth, protected, muted_ancestor_uri) AS (
			SELECT p.uri, p.created_at, 1,
				` + postAuthorBlockedPredicate("p", "$5") + ` OR ` + postReplyAuthorBlockedPredicate("p") + ` OR ` + postMentionAuthorBlockedPredicate("p") + `,
				CASE WHEN ` + postAuthorMutedPredicate("p", "$5") + ` THEN p.uri END
			FROM craftsky_posts p
			WHERE p.reply_parent_uri = $1
			  AND p.reply_root_uri = $2
			UNION ALL
			SELECT child.uri, child.created_at, parent.depth + 1,
				parent.protected OR ` + postAuthorBlockedPredicate("child", "$5") + ` OR ` + postReplyAuthorBlockedPredicate("child") + ` OR ` + postMentionAuthorBlockedPredicate("child") + `,
				COALESCE(
					parent.muted_ancestor_uri,
					CASE WHEN ` + postAuthorMutedPredicate("child", "$5") + ` THEN child.uri END
				)
			FROM craftsky_posts child
			JOIN branch parent ON child.reply_parent_uri = parent.uri
			WHERE child.reply_root_uri = $2
			  AND parent.depth < 64
		), focus_target AS (
			SELECT COALESCE(muted_ancestor_uri, uri) AS uri
			FROM branch
			WHERE uri = $3
		), focus AS (
			SELECT branch.uri, branch.created_at
			FROM branch
			JOIN focus_target ON focus_target.uri = branch.uri
			WHERE NOT branch.protected
		), page AS (
			SELECT branch.uri, branch.created_at
			FROM branch
			JOIN focus ON true
			JOIN craftsky_posts p ON p.uri = branch.uri
			WHERE (branch.created_at, branch.uri) <= (focus.created_at, focus.uri)
			  AND NOT branch.protected
			  AND (branch.muted_ancestor_uri IS NULL OR branch.muted_ancestor_uri = branch.uri)
			` + postVisibleModerationPredicate + `
			ORDER BY branch.created_at DESC, branch.uri DESC
			LIMIT $4
		), ordered_page AS (
			SELECT uri, created_at
			FROM page
			ORDER BY created_at ASC, uri ASC
		)
		SELECT ` + postSelectColumns + `
		FROM ordered_page
		JOIN craftsky_posts p ON p.uri = ordered_page.uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE true
		` + postVisibleModerationPredicate + `
		ORDER BY ordered_page.created_at ASC, ordered_page.uri ASC
	`
	rows, err := s.pool.Query(ctx, q, commentURI, rootURI, focusURI, limit, viewerDID)
	if err != nil {
		return nil, "", fmt.Errorf("reply list branch around comment=%s root=%s focus=%s: %w", commentURI, rootURI, focusURI, err)
	}
	defer rows.Close()

	out := make([]*PostRow, 0, limit)
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("reply list branch around scan: %w", scanErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reply list branch around iter: %w", err)
	}
	if len(out) == 0 {
		return out, "", nil
	}
	last := out[len(out)-1]
	hasMore, err := s.commentBranchHasRepliesAfter(ctx, commentURI, rootURI, last.CreatedAt, last.URI)
	if err != nil {
		return nil, "", err
	}
	if !hasMore {
		return out, "", nil
	}
	next, err := envelope.EncodeCursor(map[string]any{
		"createdAt": last.CreatedAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encode reply cursor: %w", err)
	}
	return out, next, nil
}
