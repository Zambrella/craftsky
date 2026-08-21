package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"social.craftsky/appview/internal/middleware"
)

func (s *SearchStore) SearchProjects(ctx context.Context, req ProjectSearchRequest, now time.Time) ([]SearchPostRow, string, error) {
	viewerDID, _ := middleware.GetDID(ctx)
	return s.SearchProjectsWithLanguages(ctx, viewerDID.String(), []string{}, req, now)
}

func (s *SearchStore) SearchProjectsWithLanguages(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req ProjectSearchRequest,
	now time.Time,
) ([]SearchPostRow, string, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	var rows []SearchPostRow
	var cursor string
	err := s.observeDB(ctx, "search.projects", "/v1/search/projects", func(ctx context.Context) error {
		var err error
		rows, cursor, err = s.searchProjectsObserved(ctx, viewerDID, contentLanguages, req, now)
		return err
	})
	return rows, cursor, err
}

func (s *SearchStore) searchProjectsObserved(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req ProjectSearchRequest,
	now time.Time,
) ([]SearchPostRow, string, error) {
	if strings.TrimSpace(req.Query) != "" {
		return s.searchProjectsByRelevance(ctx, viewerDID, contentLanguages, req)
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
	relationshipParam := "$13"
	languagesParam := "$14"
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
		relationshipParam = "$14"
		languagesParam = "$15"
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
		` + relationshipTopLevelPredicate(relationshipParam) + `
		` + postVisibleModerationPredicate + `
		` + languageVisibilityPredicate("p", relationshipParam, languagesParam) + `
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
	args = append(args, viewerDID, contentLanguages)
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

func (s *SearchStore) searchProjectsByRelevance(
	ctx context.Context,
	viewerDID string,
	contentLanguages []string,
	req ProjectSearchRequest,
) ([]SearchPostRow, string, error) {
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
		`+relationshipTopLevelPredicate("$13")+`
		`+postVisibleModerationPredicate+`
		`+languageVisibilityPredicate("p", "$13", "$14")+`
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
		cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr(), viewerDID, contentLanguages)
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
