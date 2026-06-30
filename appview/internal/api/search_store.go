package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type SearchPostRow struct {
	Post  *PostRow
	Score float64
}

const (
	relevanceCursorKindPosts    = "searchPostsRelevance"
	relevanceCursorKindProjects = "searchProjectsRelevance"
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
	FollowedRank      int
	RelevanceRank     int
}

func (s *SearchStore) SearchProfiles(ctx context.Context, viewerDID string, req ProfileSearchRequest) ([]ProfileSearchRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	queryLower := strings.ToLower(strings.TrimSpace(req.Query))
	cur, err := DecodeProfileSearchCursor(req.Cursor)
	if err != nil {
		return nil, "", err
	}
	q := `
		WITH ranked AS (
		SELECT cp.did, ic.handle, ic.handle_lower, bp.display_name, bp.description, bp.avatar_cid, bp.avatar_mime, cp.crafts,
			true AS is_craftsky_profile,
			EXISTS (SELECT 1 FROM atproto_follows f WHERE f.did = $2 AND f.subject_did = cp.did) AS viewer_is_following,
			CASE WHEN EXISTS (SELECT 1 FROM atproto_follows f WHERE f.did = $2 AND f.subject_did = cp.did) THEN 0 ELSE 1 END AS followed_rank,
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
		` + profileVisibleModerationPredicate + `
		)
		SELECT did, handle, handle_lower, display_name, description, avatar_cid, avatar_mime, crafts, is_craftsky_profile, viewer_is_following, followed_rank, relevance_rank
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
		if err := rows.Scan(&row.DID, &row.Handle, &row.HandleLower, &row.DisplayName, &row.Description, &row.AvatarCID, &row.AvatarMime, &row.Crafts, &row.IsCraftskyProfile, &row.ViewerIsFollowing, &row.FollowedRank, &row.RelevanceRank); err != nil {
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

func (s *SearchStore) SearchHashtags(ctx context.Context, req HashtagSearchRequest, now time.Time) ([]HashtagSearchResult, string, error) {
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
	`, now.Add(-28*24*time.Hour), query, likeQuery, limit+1, cur.Offset)
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
		},
		ViewerIsFollowing: row.ViewerIsFollowing,
		Crafts:            append([]string(nil), row.Crafts...),
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	return out
}

func (s *SearchStore) SearchHashtagPosts(ctx context.Context, tag string, sort SearchSort, limit int, cursor string, now time.Time) ([]SearchPostRow, string, error) {
	return s.searchPosts(ctx, searchPostQuery{Tag: tag, Sort: sort, Limit: limit, Cursor: cursor, Now: now})
}

func (s *SearchStore) SearchPosts(ctx context.Context, req PostSearchRequest, now time.Time) ([]SearchPostRow, string, error) {
	if s != nil && s.observer != nil {
		var rows []SearchPostRow
		var cursor string
		err := s.observer.ObserveDB(ctx, observability.DBOperation{
			Operation:    "search.posts",
			RoutePattern: "/v1/search/posts",
			RunID:        middleware.GetRunID(ctx),
		}, func(ctx context.Context) error {
			var err error
			rows, cursor, err = s.searchPostsObserved(ctx, req, now)
			return err
		})
		return rows, cursor, err
	}
	return s.searchPostsObserved(ctx, req, now)
}

func (s *SearchStore) searchPostsObserved(ctx context.Context, req PostSearchRequest, now time.Time) ([]SearchPostRow, string, error) {
	if strings.TrimSpace(req.Query) != "" {
		return s.searchPostsByRelevance(ctx, req)
	}
	return s.searchPosts(ctx, searchPostQuery{Query: req.Query, Sort: req.Sort, Limit: req.Limit, Cursor: req.Cursor, Now: now})
}

func (s *SearchStore) SearchProjects(ctx context.Context, req ProjectSearchRequest, now time.Time) ([]SearchPostRow, string, error) {
	if strings.TrimSpace(req.Query) != "" {
		return s.searchProjectsByRelevance(ctx, req)
	}
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	query := strings.ToLower(strings.TrimSpace(req.Query))
	fts := searchTSQuery(query)
	queryLimit := req.Limit + 1
	var cursorWhere string
	var cursorArgs []any
	var rankedAt time.Time
	if req.Sort == SearchSortPopular {
		cur, err := DecodePopularityCursor(req.Cursor)
		if err != nil {
			return nil, "", err
		}
		rankedAt = now
		if !cur.RankedAt.IsZero() {
			rankedAt = cur.RankedAt
		}
		cursorWhere = `AND ($11::double precision IS NULL OR (popularity_score, p.created_at, p.uri) < ($11::double precision, $12::timestamptz, $13::text))`
		cursorArgs = []any{cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr()}
	} else {
		curCreatedAt, curURI, err := DecodeChronologicalSearchCursor(req.Cursor)
		if err != nil {
			return nil, "", err
		}
		cursorWhere = `AND ($11::timestamptz IS NULL OR (p.created_at, p.uri) < ($11::timestamptz, $12::text))`
		cursorArgs = []any{curCreatedAt, curURI}
	}
	orderBy := `ORDER BY p.created_at DESC, p.uri DESC`
	if req.Sort == SearchSortPopular {
		orderBy = `ORDER BY popularity_score DESC, p.created_at DESC, p.uri DESC`
	}
	q := `
		WITH candidate AS (
		SELECT p.*, pp.raw_project, bp.display_name, bp.avatar_cid,
			COALESCE(l.like_count, 0)::int AS like_count,
			COALESCE(r.repost_count, 0)::int AS repost_count,
			COALESCE(re.reply_count, 0)::int AS reply_count,
			(COALESCE(l.like_count, 0) + (2 * COALESCE(re.reply_count, 0)) + (3 * COALESCE(r.repost_count, 0))) /
				pow(1 + greatest(extract(epoch from ($10::timestamptz - p.created_at)) / 3600, 0) / 72, 1.5) AS popularity_score
		FROM craftsky_posts p
		JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		LEFT JOIN (SELECT subject_uri, count(*) AS like_count FROM craftsky_likes WHERE deleted_at IS NULL GROUP BY subject_uri) l ON l.subject_uri = p.uri
		LEFT JOIN (SELECT subject_uri, count(*) AS repost_count FROM craftsky_reposts WHERE deleted_at IS NULL GROUP BY subject_uri) r ON r.subject_uri = p.uri
		LEFT JOIN (
			SELECT reply_root_uri AS subject_uri, count(*) AS reply_count
			FROM craftsky_posts rp
			WHERE rp.reply_root_uri IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1 FROM moderation_outputs mo
				WHERE mo.action = 'apply'
				  AND mo.value IN ('hide', 'takedown')
				  AND (mo.expires_at IS NULL OR mo.expires_at > now())
				  AND ((mo.subject_type = 'post' AND mo.subject_uri = rp.uri) OR (mo.subject_type = 'account' AND mo.subject_did = rp.did))
			  )
			GROUP BY reply_root_uri
		) re ON re.subject_uri = p.uri
		WHERE p.is_project = true
		  AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL AND p.quote_uri IS NULL
		  AND (cardinality($2::text[]) = 0 OR lower(pp.common_craft_type) = ANY($2::text[]))
		  AND (cardinality($3::text[]) = 0 OR lower(coalesce(pp.pattern_difficulty, '')) = ANY($3::text[]))
		  AND (cardinality($4::text[]) = 0 OR lower(coalesce(pp.knitting_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.crochet_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.quilting_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.sewing_project_type, '')) = ANY($4::text[]))
		  AND (cardinality($5::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.colors, '{}')) AS v WHERE lower(v) = ANY($5::text[])))
		  AND (cardinality($6::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.materials, '{}')) AS v WHERE lower(v) = ANY($6::text[])))
		  AND (cardinality($7::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.design_tags, '{}')) AS v WHERE lower(v) = ANY($7::text[])))
		  AND (cardinality($8::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.project_tags, '{}')) AS v WHERE lower(v) = ANY($8::text[])))
		  AND ($9 = '' OR (
			to_tsvector('simple', coalesce(p.text, '')) @@ plainto_tsquery('simple', $9)
			OR to_tsvector('simple', coalesce(pp.common_title, '') || ' ' || coalesce(pp.pattern_name, '') || ' ' || coalesce(craftsky_text_array_to_string(pp.materials, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.project_tags, ' '), '') || ' ' || coalesce(craftsky_text_array_to_string(pp.design_tags, ' '), '')) @@ plainto_tsquery('simple', $9)
		  ))
		` + postVisibleModerationPredicate + `
		)
		SELECT ` + postSelectColumns + `, popularity_score
		FROM candidate p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE true ` + cursorWhere + `
		` + orderBy + `
		LIMIT $1`
	args := []any{queryLimit,
		projectFilterValues(req, "craftType"), projectFilterValues(req, "patternDifficulty"), projectFilterValues(req, "projectType"), projectFilterValues(req, "color"), projectFilterValues(req, "material"), projectFilterValues(req, "designTag"), projectFilterValues(req, "projectTag"), fts, rankedAt,
	}
	args = append(args, cursorArgs...)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("project search: %w", err)
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
	if req.Sort == SearchSortPopular {
		next, err := EncodePopularityCursor(rankedAt, last.Score, last.Post.CreatedAt, last.Post.URI)
		return out, next, err
	}
	next, err := EncodeChronologicalSearchCursor(last.Post.CreatedAt, last.Post.URI)
	return out, next, err
}

func (s *SearchStore) searchPostsByRelevance(ctx context.Context, req PostSearchRequest) ([]SearchPostRow, string, error) {
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
		`+postVisibleModerationPredicate+`
		)
		SELECT `+postSelectColumns+`, relevance_score
		FROM ranked p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE ($3::double precision IS NULL OR (relevance_score, p.created_at, p.uri) < ($3::double precision, $4::timestamptz, $5::text))
		ORDER BY relevance_score DESC, p.created_at DESC, p.uri DESC
		LIMIT $2
	`, fts, queryLimit, cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr())
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

func (s *SearchStore) searchProjectsByRelevance(ctx context.Context, req ProjectSearchRequest) ([]SearchPostRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	queryLimit := req.Limit + 1
	fts := searchTSQuery(req.Query)
	cur, err := DecodeRelevanceSearchCursor(req.Cursor, relevanceCursorKindProjects, fts)
	if err != nil {
		return nil, "", err
	}
	projectVector := `
		setweight(to_tsvector('simple', coalesce(pp.common_title, '')), 'A') ||
		setweight(to_tsvector('simple', coalesce(pp.pattern_name, '')), 'B') ||
		setweight(to_tsvector('simple', coalesce(craftsky_text_array_to_string(pp.materials, ' '), '')), 'C') ||
		setweight(to_tsvector('simple', coalesce(craftsky_text_array_to_string(pp.project_tags, ' '), '')), 'C') ||
		setweight(to_tsvector('simple', coalesce(craftsky_text_array_to_string(pp.design_tags, ' '), '')), 'C') ||
		setweight(to_tsvector('simple', coalesce(p.text, '')), 'D')`
	rows, err := s.pool.Query(ctx, `
		WITH ranked AS (
		SELECT p.*,
		  ts_rank_cd(`+projectVector+`, plainto_tsquery('simple', $9))::double precision AS relevance_score
		FROM craftsky_posts p
		JOIN craftsky_project_posts pp ON pp.uri = p.uri
		WHERE p.is_project = true
		  AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL AND p.quote_uri IS NULL
		  AND (cardinality($2::text[]) = 0 OR lower(pp.common_craft_type) = ANY($2::text[]))
		  AND (cardinality($3::text[]) = 0 OR lower(coalesce(pp.pattern_difficulty, '')) = ANY($3::text[]))
		  AND (cardinality($4::text[]) = 0 OR lower(coalesce(pp.knitting_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.crochet_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.quilting_project_type, '')) = ANY($4::text[]) OR lower(coalesce(pp.sewing_project_type, '')) = ANY($4::text[]))
		  AND (cardinality($5::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.colors, '{}')) AS v WHERE lower(v) = ANY($5::text[])))
		  AND (cardinality($6::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.materials, '{}')) AS v WHERE lower(v) = ANY($6::text[])))
		  AND (cardinality($7::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.design_tags, '{}')) AS v WHERE lower(v) = ANY($7::text[])))
		  AND (cardinality($8::text[]) = 0 OR EXISTS (SELECT 1 FROM unnest(coalesce(pp.project_tags, '{}')) AS v WHERE lower(v) = ANY($8::text[])))
		  AND (`+projectVector+`) @@ plainto_tsquery('simple', $9)
		`+postVisibleModerationPredicate+`
		)
		SELECT `+postSelectColumns+`, relevance_score
		FROM ranked p
		JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE ($10::double precision IS NULL OR (relevance_score, p.created_at, p.uri) < ($10::double precision, $11::timestamptz, $12::text))
		ORDER BY relevance_score DESC, p.created_at DESC, p.uri DESC
		LIMIT $1
	`, queryLimit,
		projectFilterValues(req, "craftType"), projectFilterValues(req, "patternDifficulty"), projectFilterValues(req, "projectType"), projectFilterValues(req, "color"), projectFilterValues(req, "material"), projectFilterValues(req, "designTag"), projectFilterValues(req, "projectTag"), fts,
		cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr())
	if err != nil {
		return nil, "", fmt.Errorf("project relevance search: %w", err)
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
	next, err := EncodeRelevanceSearchCursor(relevanceCursorKindProjects, fts, last.Score, last.Post.CreatedAt, last.Post.URI)
	return out, next, err
}

func searchTSQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func projectFilterValues(req ProjectSearchRequest, key string) []string {
	if req.Filters == nil || req.Filters[key] == nil {
		return []string{}
	}
	if key == "craftType" {
		crafts, err := CanonicalCraftTypes(req.Filters[key], false)
		if err == nil {
			return crafts
		}
	}
	return req.Filters[key]
}

func (s *SearchStore) TopHashtags(ctx context.Context, req TopHashtagsRequest, now time.Time) ([]TopHashtagGroup, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("search store unavailable")
	}
	crafts, err := CanonicalCraftTypes(req.CraftTypes, true)
	if err != nil {
		return nil, err
	}
	groups := make([]TopHashtagGroup, 0, len(crafts))
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
			` + postVisibleModerationPredicate + `
			GROUP BY lower(tag)
			ORDER BY count DESC, tag ASC
			LIMIT $3`
		rows, err := s.pool.Query(ctx, q, craft, now.Add(-28*24*time.Hour), req.Limit)
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

type searchPostQuery struct {
	Tag    string
	Query  string
	Sort   SearchSort
	Limit  int
	Cursor string
	Now    time.Time
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
	var err error
	if req.Sort == SearchSortPopular {
		cur, err := DecodePopularityCursor(req.Cursor)
		if err != nil {
			return nil, "", err
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
			LEFT JOIN (SELECT subject_uri, count(*) AS like_count FROM craftsky_likes WHERE deleted_at IS NULL GROUP BY subject_uri) l ON l.subject_uri = p.uri
			LEFT JOIN (SELECT subject_uri, count(*) AS repost_count FROM craftsky_reposts WHERE deleted_at IS NULL GROUP BY subject_uri) r ON r.subject_uri = p.uri
			LEFT JOIN (
				SELECT reply_root_uri AS subject_uri, count(*) AS reply_count
				FROM craftsky_posts rp
				WHERE rp.reply_root_uri IS NOT NULL
				  AND NOT EXISTS (
					SELECT 1 FROM moderation_outputs mo
					WHERE mo.action = 'apply'
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
			` + postVisibleModerationPredicate + `
		)
		SELECT ` + postSelectColumns + `, popularity_score
		FROM candidate p
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE ($4::double precision IS NULL OR (popularity_score, p.created_at, p.uri) < ($4::double precision, $5::timestamptz, $6::text))
		ORDER BY popularity_score DESC, p.created_at DESC, p.uri DESC
		LIMIT $2`
		rows, err = s.pool.Query(ctx, q, req.Tag, queryLimit, req.Now, cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr(), strings.ToLower(req.Query))
	} else {
		curCreatedAt, curURI, err := DecodeChronologicalSearchCursor(req.Cursor)
		if err != nil {
			return nil, "", err
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
		  AND ($3::timestamptz IS NULL OR (p.created_at, p.uri) < ($3::timestamptz, $4::text))
		` + postVisibleModerationPredicate + `
		ORDER BY p.created_at DESC, p.uri DESC
		LIMIT $2`
		rows, err = s.pool.Query(ctx, q, req.Tag, queryLimit, curCreatedAt, curURI, strings.ToLower(req.Query))
	}
	if err != nil {
		return nil, "", fmt.Errorf("search hashtag posts: %w", err)
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

func scanSearchPostRow(scanner pgx.Row) (SearchPostRow, error) {
	row, err := scanPostRowWithExtraScore(scanner)
	return row, err
}

func scanPostRowWithExtraScore(scanner pgx.Row) (SearchPostRow, error) {
	post := &PostRow{}
	var rawProject *[]byte
	var score float64
	err := scanner.Scan(
		&post.URI, &post.DID, &post.Rkey, &post.CID, &post.Text, &post.Facets, &post.Images,
		&post.ReplyRootURI, &post.ReplyRootCID, &post.ReplyParentURI, &post.ReplyParentCID,
		&post.QuoteURI, &post.QuoteCID, &post.Tags, &post.CreatedAt, &post.IndexedAt,
		&post.IsProject, &post.ProjectCraftType, &rawProject,
		&post.AuthorDisplayName, &post.AuthorAvatarCID, &post.AuthorAvatarMime,
		&post.ModerationWarningKind,
		&score,
	)
	if err != nil {
		return SearchPostRow{}, err
	}
	if rawProject != nil && len(*rawProject) > 0 {
		post.RawProject = append([]byte(nil), (*rawProject)...)
		var project Project
		if err := json.Unmarshal(post.RawProject, &project); err != nil {
			return SearchPostRow{}, err
		}
		post.Project = &project
	}
	return SearchPostRow{Post: post, Score: score}, nil
}

func (s *SearchStore) EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error) {
	if s.postStore == nil {
		s.postStore = NewPostStore(s.pool)
	}
	return s.postStore.EngagementSummaries(ctx, viewerDID, postURIs)
}

func PopularityScore(likes, visibleReplies, reposts int, createdAt, rankedAt time.Time) float64 {
	ageHours := rankedAt.Sub(createdAt).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	weighted := float64(likes + 2*visibleReplies + 3*reposts)
	return weighted / math.Pow(1+ageHours/72, 1.5)
}

func isInvalidCursor(err error) bool { return err == envelope.ErrInvalidCursor }
