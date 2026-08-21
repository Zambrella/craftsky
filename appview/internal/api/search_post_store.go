package api

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/middleware"
)

func (s *SearchStore) SearchPosts(ctx context.Context, req PostSearchRequest, now time.Time) ([]SearchPostRow, string, error) {
	viewerDID, _ := middleware.GetDID(ctx)
	return s.SearchPostsWithLanguages(ctx, viewerDID.String(), []string{}, req, now)
}

func (s *SearchStore) SearchPostsWithLanguages(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req PostSearchRequest,
	now time.Time,
) ([]SearchPostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	var rows []SearchPostRow
	var cursor string
	err := s.observeDB(ctx, "search.posts", "/v1/search/posts", func(ctx context.Context) error {
		var err error
		rows, cursor, err = s.searchPostsObserved(ctx, viewerDID, contentLanguages, req, now)
		return err
	})
	return rows, cursor, err
}

func (s *SearchStore) searchPostsObserved(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req PostSearchRequest,
	now time.Time,
) ([]SearchPostRow, string, error) {
	if strings.TrimSpace(req.Query) != "" {
		return s.searchPostsByRelevance(ctx, viewerDID, contentLanguages, req)
	}
	return s.searchPosts(ctx, searchPostQuery{
		Query:            req.Query,
		Sort:             req.Sort,
		Limit:            req.Limit,
		Cursor:           req.Cursor,
		Now:              now,
		ViewerDID:        viewerDID,
		ContentLanguages: contentLanguages,
	})
}

func (s *SearchStore) searchPostsByRelevance(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req PostSearchRequest,
) ([]SearchPostRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	queryLimit := req.Limit + 1
	fts := searchTSQuery(req.Query)
	cur, err := DecodeRelevanceSearchCursor(req.Cursor, relevanceCursorKindPosts, fts)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
		SELECT p.*,
		  ts_rank_cd(to_tsvector('simple', coalesce(p.text, '')), plainto_tsquery('simple', $1))::double precision AS relevance_score
		FROM craftsky_posts p
		WHERE p.is_project = false
		  AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
		  AND to_tsvector('simple', coalesce(p.text, '')) @@ plainto_tsquery('simple', $1)
		`+relationshipTopLevelPredicate("$6")+`
		`+postVisibleModerationPredicate+`
		`+languageVisibilityPredicate("p", "$6", "$7")+`
		)
		SELECT `+postSelectColumns+`, relevance_score
		FROM ranked p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE ($3::double precision IS NULL OR (relevance_score, p.created_at, p.uri) < ($3::double precision, $4::timestamptz, $5::text))
		ORDER BY relevance_score DESC, p.created_at DESC, p.uri DESC
		LIMIT $2
	`, fts, queryLimit, cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr(), viewerDID, contentLanguages)
	if err != nil {
		return nil, "", fmt.Errorf("post relevance search: %w", err)
	}
	defer rows.Close()
	out := make([]SearchPostRow, 0, req.Limit)
	for rows.Next() {
		row, scanErr := scanSearchPostRow(rows)
		if scanErr != nil {
			return nil, "", scanErr
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
	next, err := EncodeRelevanceSearchCursor(relevanceCursorKindPosts, fts, last.Score, last.Post.CreatedAt, last.Post.URI)
	return out, next, err
}

type searchPostQuery struct {
	Tag              string
	Query            string
	Sort             SearchSort
	Limit            int
	Cursor           string
	Now              time.Time
	ViewerDID        string
	ContentLanguages []string
}

func (s *SearchStore) searchPosts(ctx context.Context, req searchPostQuery) ([]SearchPostRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	queryLimit := req.Limit + 1
	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	var rows pgx.Rows
	if req.Sort == SearchSortPopular {
		cur, decodeErr := DecodePopularityCursor(req.Cursor)
		if decodeErr != nil {
			return nil, "", decodeErr
		}
		if !cur.RankedAt.IsZero() {
			req.Now = cur.RankedAt
		}
		q := `
		WITH candidate AS (
			SELECT p.*, pp.raw_project, bp.display_name, bp.avatar_cid,
				COALESCE(l.like_count, 0)::int AS like_count,
				COALESCE(r.repost_count, 0)::int AS repost_count,
				COALESCE(re.reply_count, 0)::int AS reply_count,
				(COALESCE(l.like_count, 0) + (2 * COALESCE(re.reply_count, 0)) + (3 * COALESCE(r.repost_count, 0))) /
					pow(1 + greatest(extract(epoch from ($3::timestamptz - p.created_at)) / 3600, 0) / 72, 1.5) AS popularity_score
			FROM craftsky_posts p
			LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
			LEFT JOIN bluesky_profiles bp ON bp.did = p.did
			LEFT JOIN (SELECT subject_uri, count(*) AS like_count FROM craftsky_likes WHERE deleted_at IS NULL AND NOT appview_owner_is_terminal(did) GROUP BY subject_uri) l ON l.subject_uri = p.uri
			LEFT JOIN (SELECT subject_uri, count(*) AS repost_count FROM craftsky_reposts WHERE deleted_at IS NULL AND NOT appview_owner_is_terminal(did) GROUP BY subject_uri) r ON r.subject_uri = p.uri
			LEFT JOIN (
				SELECT reply_root_uri AS subject_uri, count(*) AS reply_count
				FROM craftsky_posts rp
				WHERE rp.reply_root_uri IS NOT NULL
				  AND NOT appview_owner_is_terminal(rp.did)
				  AND NOT EXISTS (
					SELECT 1 FROM moderation_outputs mo
					WHERE mo.action = 'apply'
					  AND NOT appview_owner_is_terminal(mo.source_did)
					  AND mo.value IN ('hide', 'takedown')
					  AND (mo.expires_at IS NULL OR mo.expires_at > now())
					  AND ((mo.subject_type = 'post' AND mo.subject_uri = rp.uri) OR (mo.subject_type = 'account' AND mo.subject_did = rp.did))
				  )
				GROUP BY reply_root_uri
			) re ON re.subject_uri = p.uri
			WHERE p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
			  AND (
				($1 <> '' AND lower($1) = ANY(p.tags))
				OR ($7 <> '' AND (
					to_tsvector('simple', coalesce(p.text, '')) @@ plainto_tsquery('simple', $7)
					OR to_tsvector('simple', coalesce(pp.common_title, '') || ' ' || coalesce(pp.pattern_name, '') || ' ' || coalesce(craftsky_text_array_to_string(pp.materials, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.project_tags, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.design_tags, ' '), '')) @@ plainto_tsquery('simple', $7)
				))
			  )
			` + relationshipTopLevelPredicate("$8") + `
			` + postVisibleModerationPredicate + `
			` + languageVisibilityPredicate("p", "$8", "$9") + `
		)
		SELECT ` + postSelectColumns + `, popularity_score
		FROM candidate p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE ($4::double precision IS NULL OR (popularity_score, p.created_at, p.uri) < ($4::double precision, $5::timestamptz, $6::text))
		ORDER BY popularity_score DESC, p.created_at DESC, p.uri DESC
		LIMIT $2`
		var queryErr error
		rows, queryErr = s.pool.Query(ctx, q, req.Tag, queryLimit, req.Now, cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr(), strings.ToLower(req.Query), req.ViewerDID, req.ContentLanguages)
		if queryErr != nil {
			return nil, "", fmt.Errorf("search hashtag posts: %w", queryErr)
		}
	} else {
		curCreatedAt, curURI, decodeErr := DecodeChronologicalSearchCursor(req.Cursor)
		if decodeErr != nil {
			return nil, "", decodeErr
		}
		q := `
		SELECT ` + postSelectColumns + `, 0::double precision AS popularity_score
		FROM craftsky_posts p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
		  AND (
			($1 <> '' AND lower($1) = ANY(p.tags))
			OR ($5 <> '' AND (
				to_tsvector('simple', coalesce(p.text, '')) @@ plainto_tsquery('simple', $5)
				OR to_tsvector('simple', coalesce(pp.common_title, '') || ' ' || coalesce(pp.pattern_name, '') || ' ' || coalesce(craftsky_text_array_to_string(pp.materials, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.project_tags, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.design_tags, ' '), '')) @@ plainto_tsquery('simple', $5)
			))
		  )
		` + relationshipTopLevelPredicate("$6") + `
		` + languageVisibilityPredicate("p", "$6", "$7") + `
		  AND ($3::timestamptz IS NULL OR (p.created_at, p.uri) < ($3::timestamptz, $4::text))
		` + postVisibleModerationPredicate + `
		ORDER BY p.created_at DESC, p.uri DESC
		LIMIT $2`
		var queryErr error
		rows, queryErr = s.pool.Query(ctx, q, req.Tag, queryLimit, curCreatedAt, curURI, strings.ToLower(req.Query), req.ViewerDID, req.ContentLanguages)
		if queryErr != nil {
			return nil, "", fmt.Errorf("search hashtag posts: %w", queryErr)
		}
	}
	defer rows.Close()

	out := make([]SearchPostRow, 0, req.Limit)
	for rows.Next() {
		post, scanErr := scanSearchPostRow(rows)
		if scanErr != nil {
			return nil, "", scanErr
		}
		out = append(out, post)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(out) <= req.Limit {
		return out, "", nil
	}
	out = out[:req.Limit]
	last := out[len(out)-1]
	if req.Sort == SearchSortPopular {
		next, err := EncodePopularityCursor(req.Now, last.Score, last.Post.CreatedAt, last.Post.URI)
		return out, next, err
	}
	next, err := EncodeChronologicalSearchCursor(last.Post.CreatedAt, last.Post.URI)
	return out, next, err
}

func PopularityScore(likes, visibleReplies, reposts int, createdAt, rankedAt time.Time) float64 {
	ageHours := rankedAt.Sub(createdAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	weighted := float64(likes + 2*visibleReplies + 3*reposts)
	return weighted / math.Pow(1+ageHours/72, 1.5)
}
