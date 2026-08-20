package api

import (
	"context"
	"fmt"
	"time"

	"social.craftsky/appview/internal/middleware"
)

func (s *SearchStore) SearchHashtags(ctx context.Context, req HashtagSearchRequest, now time.Time) ([]HashtagSearchResult, string, error) {
	var items []HashtagSearchResult
	var cursor string
	err := s.observeDB(ctx, "search.hashtags", "/v1/search/hashtags", func(ctx context.Context) error {
		var err error
		items, cursor, err = s.searchHashtagsObserved(ctx, req, now)
		return err
	})
	return items, cursor, err
}

func (s *SearchStore) searchHashtagsObserved(ctx context.Context, req HashtagSearchRequest, now time.Time) ([]HashtagSearchResult, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	query := normalizeHashtagSearchTerm(req.Query)
	if query == "" {
		return []HashtagSearchResult{}, "", nil
	}
	limit := req.Limit
	if limit <= 0 {
		limit = SearchDefaultLimit
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cur, err := DecodeHashtagSearchCursor(req.Cursor, query)
	if err != nil {
		return nil, "", err
	}
	likeQuery := EscapeFacetLikePattern(query)
	viewerDID, _ := middleware.GetDID(ctx)
	rows, err := s.pool.Query(ctx, `
		SELECT lower(trim(tag.raw_tag)) AS tag, COUNT(DISTINCT p.uri)::int AS posts_last_28_days
		FROM craftsky_posts p
		CROSS JOIN LATERAL unnest(p.tags) AS tag(raw_tag)
		WHERE p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND p.quote_uri IS NULL
		  AND p.created_at >= $1
		  AND trim(tag.raw_tag) <> ''
		  AND lower(trim(tag.raw_tag)) LIKE '%' || $3 || '%' ESCAPE '\'
		`+relationshipTopLevelPredicate("$6")+`
		`+postVisibleModerationPredicate+`
		GROUP BY lower(trim(tag.raw_tag))
		ORDER BY
		  CASE
		    WHEN lower(trim(tag.raw_tag)) = $2 THEN 0
		    WHEN lower(trim(tag.raw_tag)) LIKE $3 || '%' ESCAPE '\' THEN 1
		    ELSE 2
		  END ASC,
		  posts_last_28_days DESC,
		  tag ASC
		LIMIT $4 OFFSET $5
	`, now.Add(-28*24*time.Hour), query, likeQuery, limit+1, cur.Offset, viewerDID.String())
	if err != nil {
		return nil, "", fmt.Errorf("hashtag search: %w", err)
	}
	defer rows.Close()
	out := make([]HashtagSearchResult, 0, limit+1)
	for rows.Next() {
		var row HashtagSearchResult
		if err := rows.Scan(&row.Tag, &row.PostsLast28Days); err != nil {
			return nil, "", err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(out) <= limit {
		return out, "", nil
	}
	out = out[:limit]
	next, err := EncodeHashtagSearchCursor(query, cur.Offset+limit)
	return out, next, err
}

func (s *SearchStore) SearchHashtagPosts(ctx context.Context, tag string, sort SearchSort, limit int, cursor string, now time.Time) ([]SearchPostRow, string, error) {
	viewerDID, _ := middleware.GetDID(ctx)
	return s.SearchHashtagPostsWithLanguages(ctx, viewerDID.String(), []string{}, tag, sort, limit, cursor, now)
}

func (s *SearchStore) SearchHashtagPostsWithLanguages(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	tag string,
	sort SearchSort,
	limit int,
	cursor string,
	now time.Time,
) ([]SearchPostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	var rows []SearchPostRow
	var nextCursor string
	err := s.observeDB(ctx, "search.hashtag_posts", "/v1/search/hashtags/{tag}/posts", func(ctx context.Context) error {
		var err error
		rows, nextCursor, err = s.searchPosts(ctx, searchPostQuery{
			Tag:              tag,
			Sort:             sort,
			Limit:            limit,
			Cursor:           cursor,
			Now:              now,
			ViewerDID:        viewerDID,
			ContentLanguages: contentLanguages,
		})
		return err
	})
	return rows, nextCursor, err
}

func (s *SearchStore) TopHashtags(ctx context.Context, req TopHashtagsRequest, now time.Time) ([]TopHashtagGroup, error) {
	var groups []TopHashtagGroup
	err := s.observeDB(ctx, "search.hashtags.top", "/v1/search/hashtags/top", func(ctx context.Context) error {
		var err error
		groups, err = s.topHashtagsObserved(ctx, req, now)
		return err
	})
	return groups, err
}

func (s *SearchStore) topHashtagsObserved(ctx context.Context, req TopHashtagsRequest, now time.Time) ([]TopHashtagGroup, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("search store unavailable")
	}
	crafts, err := CanonicalCraftTypes(req.CraftTypes, true)
	if err != nil {
		return nil, err
	}
	groups := make([]TopHashtagGroup, 0, len(crafts))
	viewerDID, _ := middleware.GetDID(ctx)
	for _, craft := range crafts {
		q := `
			SELECT lower(tag) AS tag, count(DISTINCT p.uri)::int AS count
			FROM craftsky_posts p
			JOIN craftsky_project_posts pp ON pp.uri = p.uri
			CROSS JOIN LATERAL unnest(p.tags) AS tag
			WHERE p.is_project = true
			  AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL AND p.quote_uri IS NULL
			  AND p.created_at >= $2
			  AND lower(pp.common_craft_type) = $1
			` + relationshipTopLevelPredicate("$4") + `
			` + postVisibleModerationPredicate + `
			GROUP BY lower(tag)
			ORDER BY count DESC, tag ASC
			LIMIT $3`
		rows, err := s.pool.Query(ctx, q, craft, now.Add(-28*24*time.Hour), req.Limit, viewerDID.String())
		if err != nil {
			return nil, fmt.Errorf("top hashtags: %w", err)
		}
		items := []TopHashtagItem{}
		for rows.Next() {
			var item TopHashtagItem
			if err := rows.Scan(&item.Tag, &item.Count); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		groups = append(groups, TopHashtagGroup{CraftType: craft, Items: items})
	}
	return groups, nil
}
