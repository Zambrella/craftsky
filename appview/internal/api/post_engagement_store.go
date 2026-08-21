// appview/internal/api/post_engagement_store.go
package api

import (
	"context"
	"fmt"
)

func (s *PostStore) countActiveInteractions(ctx context.Context, table, label string, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	q := `
		SELECT subject_uri, count(*)::int
		FROM ` + table + `
		WHERE deleted_at IS NULL AND subject_uri = ANY($1::text[])
		  AND NOT appview_owner_is_terminal(did)
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("%s count active: %w", label, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("%s count scan: %w", label, err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s count iter: %w", label, err)
	}
	return out, nil
}

// CountActiveLikes returns active like counts keyed by post URI.
func (s *PostStore) CountActiveLikes(ctx context.Context, postURIs []string) (map[string]int, error) {
	return s.countActiveInteractions(ctx, "craftsky_likes", "like", postURIs)
}

// CountActiveReposts returns active repost counts keyed by post URI.
func (s *PostStore) CountActiveReposts(ctx context.Context, postURIs []string) (map[string]int, error) {
	return s.countActiveInteractions(ctx, "craftsky_reposts", "repost", postURIs)
}

// CountVisibleQuotes returns visible top-level quote-post counts keyed by the
// quoted subject URI.
func (s *PostStore) CountVisibleQuotes(ctx context.Context, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	q := `
		SELECT p.quote_uri, count(*)::int
		FROM craftsky_posts p
		WHERE p.quote_uri = ANY($1::text[])
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		` + postVisibleModerationPredicate + `
		GROUP BY p.quote_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("quote count visible: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("quote count visible scan: %w", err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("quote count visible iter: %w", err)
	}
	return out, nil
}

// CountDescendantReplies returns all descendant reply counts keyed by ancestor
// post URI. Traversal is depth-capped to match branch rendering.
func (s *PostStore) CountDescendantReplies(ctx context.Context, postURIs []string) (map[string]int, error) {
	out := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = 0
	}
	const q = `
		WITH RECURSIVE descendants(subject_uri, uri, depth) AS (
			SELECT subjects.subject_uri, p.uri, 1
			FROM unnest($1::text[]) AS subjects(subject_uri)
			JOIN craftsky_posts p ON p.reply_parent_uri = subjects.subject_uri
			WHERE NOT appview_owner_is_terminal(p.did)
			UNION ALL
			SELECT descendants.subject_uri, child.uri, descendants.depth + 1
			FROM descendants
			JOIN craftsky_posts child ON child.reply_parent_uri = descendants.uri
			WHERE descendants.depth < 64
			  AND NOT appview_owner_is_terminal(child.did)
		)
		SELECT subject_uri, count(*)::int
		FROM descendants
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, postURIs)
	if err != nil {
		return nil, fmt.Errorf("reply count descendants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, fmt.Errorf("reply count descendant scan: %w", err)
		}
		out[uri] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reply count descendant iter: %w", err)
	}
	return out, nil
}

// ViewerInteractionStates returns current-viewer active like/repost booleans keyed by post URI.
func (s *PostStore) ViewerInteractionStates(ctx context.Context, viewerDID string, postURIs []string) (map[string]ViewerInteractionState, error) {
	out := make(map[string]ViewerInteractionState, len(postURIs))
	if len(postURIs) == 0 || viewerDID == "" {
		for _, uri := range postURIs {
			out[uri] = ViewerInteractionState{}
		}
		return out, nil
	}
	for _, uri := range postURIs {
		out[uri] = ViewerInteractionState{}
	}
	const q = `
		SELECT subject_uri, bool_or(kind = 'like'), bool_or(kind = 'repost')
		FROM (
			SELECT subject_uri, 'like' AS kind
			FROM craftsky_likes
			WHERE did = $1 AND deleted_at IS NULL AND subject_uri = ANY($2::text[])
			  AND NOT appview_owner_is_terminal(did)
			UNION ALL
			SELECT subject_uri, 'repost' AS kind
			FROM craftsky_reposts
			WHERE did = $1 AND deleted_at IS NULL AND subject_uri = ANY($2::text[])
			  AND NOT appview_owner_is_terminal(did)
		) interactions
		GROUP BY subject_uri
	`
	rows, err := s.pool.Query(ctx, q, viewerDID, postURIs)
	if err != nil {
		return nil, fmt.Errorf("viewer interaction states %s: %w", viewerDID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var liked, reposted bool
		if err := rows.Scan(&uri, &liked, &reposted); err != nil {
			return nil, fmt.Errorf("viewer interaction states scan: %w", err)
		}
		out[uri] = ViewerInteractionState{HasLiked: liked, HasReposted: reposted}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("viewer interaction states iter: %w", err)
	}
	return out, nil
}

// ViewerReplyStates returns whether the current viewer authored a direct child reply for each post URI.
func (s *PostStore) ViewerReplyStates(ctx context.Context, viewerDID string, postURIs []string) (map[string]ViewerReplyState, error) {
	out := make(map[string]ViewerReplyState, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = ViewerReplyState{}
	}
	if len(postURIs) == 0 || viewerDID == "" {
		return out, nil
	}
	const q = `
		WITH RECURSIVE subjects(uri, reply_parent_uri) AS (
			SELECT uri, reply_parent_uri
			FROM craftsky_posts
			WHERE uri = ANY($2::text[])
			  AND NOT appview_owner_is_terminal(did)
		), descendants(subject_uri, uri, depth) AS (
			SELECT subjects.uri, child.uri, 1
			FROM subjects
			JOIN craftsky_posts child ON child.reply_parent_uri = subjects.uri
			WHERE NOT appview_owner_is_terminal(child.did)
			UNION ALL
			SELECT descendants.subject_uri, child.uri, descendants.depth + 1
			FROM descendants
			JOIN craftsky_posts child ON child.reply_parent_uri = descendants.uri
			WHERE descendants.depth < 64
			  AND NOT appview_owner_is_terminal(child.did)
		)
		SELECT descendants.subject_uri, true
		FROM descendants
		JOIN subjects ON subjects.uri = descendants.subject_uri
		JOIN craftsky_posts viewer_reply ON viewer_reply.uri = descendants.uri
		WHERE viewer_reply.did = $1
		  AND NOT appview_owner_is_terminal(viewer_reply.did)
		  AND (subjects.reply_parent_uri IS NOT NULL OR descendants.depth = 1)
		GROUP BY descendants.subject_uri
	`
	rows, err := s.pool.Query(ctx, q, viewerDID, postURIs)
	if err != nil {
		return nil, fmt.Errorf("viewer reply states %s: %w", viewerDID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var replied bool
		if err := rows.Scan(&uri, &replied); err != nil {
			return nil, fmt.Errorf("viewer reply states scan: %w", err)
		}
		out[uri] = ViewerReplyState{HasReplied: replied}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("viewer reply states iter: %w", err)
	}
	return out, nil
}

// EngagementSummaries returns counts and current-viewer state keyed by post URI.
func (s *PostStore) EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	var summaries map[string]EngagementSummary
	err := s.observeDB(ctx, "post.engagement_summaries", "unmatched", func(ctx context.Context) error {
		var err error
		summaries, err = s.engagementSummariesObserved(ctx, viewerDID, postURIs)
		return err
	})
	return summaries, err
}

func (s *PostStore) viewerSavedStates(ctx context.Context, viewerDID string, postURIs []string) (map[string]ViewerSavedState, error) {
	out := make(map[string]ViewerSavedState, len(postURIs))
	rows, err := s.pool.Query(ctx, `
		SELECT post_uri, folder_id::text
		FROM saved_posts
		WHERE owner_did = $1
		  AND post_uri = ANY($2::text[])
	`, viewerDID, postURIs)
	if err != nil {
		return nil, fmt.Errorf("viewer saved states: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uri string
		var folderID *string
		if err := rows.Scan(&uri, &folderID); err != nil {
			return nil, fmt.Errorf("viewer saved states scan: %w", err)
		}
		out[uri] = ViewerSavedState{HasSaved: true, FolderID: folderID}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("viewer saved states iter: %w", err)
	}
	return out, nil
}

func (s *PostStore) engagementSummariesObserved(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	out := make(map[string]EngagementSummary, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = EngagementSummary{}
	}
	if len(postURIs) == 0 {
		return out, nil
	}
	likeCounts, err := s.CountActiveLikes(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	repostCounts, err := s.CountActiveReposts(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	quoteCounts, err := s.CountVisibleQuotes(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	replyCounts, err := s.CountDescendantReplies(ctx, postURIs)
	if err != nil {
		return nil, err
	}
	viewerStates, err := s.ViewerInteractionStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	viewerReplyStates, err := s.ViewerReplyStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	viewerSavedStates, err := s.viewerSavedStates(ctx, viewerDID, postURIs)
	if err != nil {
		return nil, err
	}
	for _, uri := range postURIs {
		state := viewerStates[uri]
		replyState := viewerReplyStates[uri]
		savedState := viewerSavedStates[uri]
		out[uri] = EngagementSummary{
			LikeCount:           likeCounts[uri],
			RepostCount:         repostCounts[uri],
			QuoteCount:          quoteCounts[uri],
			ReplyCount:          replyCounts[uri],
			ViewerHasLiked:      state.HasLiked,
			ViewerHasReposted:   state.HasReposted,
			ViewerHasReplied:    replyState.HasReplied,
			ViewerHasSaved:      savedState.HasSaved,
			ViewerSavedFolderID: savedState.FolderID,
		}
	}
	return out, nil
}

// QuoteViewRows returns compact quote-preview hydration rows keyed by quoted
// URI. Missing/unindexed/deleted refs are unavailable; indexed rows hidden by
// moderation are hidden; visible rows include the target PostRow.
func (s *PostStore) QuoteViewRows(ctx context.Context, refs []ResponseStrongRef) (map[string]*QuoteViewRow, error) {
	out := make(map[string]*QuoteViewRow, len(refs))
	uris := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.URI == "" {
			continue
		}
		out[ref.URI] = &QuoteViewRow{State: "unavailable"}
		if _, ok := seen[ref.URI]; ok {
			continue
		}
		seen[ref.URI] = struct{}{}
		uris = append(uris, ref.URI)
	}
	if len(uris) == 0 {
		return out, nil
	}

	q := `
		SELECT ` + postSelectColumns + `
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.uri = ANY($1::text[])
		` + postVisibleModerationPredicate + `
	`
	rows, err := s.pool.Query(ctx, q, uris)
	if err != nil {
		return nil, fmt.Errorf("quote view rows: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		row, scanErr := scanPostRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("quote view rows scan: %w", scanErr)
		}
		out[row.URI] = &QuoteViewRow{State: "visible", Post: row}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("quote view rows iter: %w", err)
	}

	hiddenRows, err := s.pool.Query(ctx, `
		SELECT uri FROM craftsky_posts
		WHERE uri = ANY($1::text[])
		  AND NOT appview_owner_is_terminal(did)
	`, uris)
	if err != nil {
		return nil, fmt.Errorf("quote view hidden rows: %w", err)
	}
	defer hiddenRows.Close()
	for hiddenRows.Next() {
		var uri string
		if err := hiddenRows.Scan(&uri); err != nil {
			return nil, fmt.Errorf("quote view hidden rows scan: %w", err)
		}
		if current := out[uri]; current != nil && current.State == "unavailable" {
			out[uri] = &QuoteViewRow{State: "hidden"}
		}
	}
	if err := hiddenRows.Err(); err != nil {
		return nil, fmt.Errorf("quote view hidden rows iter: %w", err)
	}
	return out, nil
}
