package api

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *SearchStore) searchSubmittedByRelevance(ctx context.Context, viewer string, languages []string, req ProjectSearchRequest, project bool) ([]SearchPostRow, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("search store unavailable")
	}
	kind := relevanceCursorKindPosts
	scope := `p.is_project = false AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL`
	if project {
		kind = relevanceCursorKindProjects
		scope = `p.is_project = true AND pp.uri IS NOT NULL AND p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL AND p.quote_uri IS NULL`
	}
	// Posts use empty filter arrays; project filters retain their existing semantics.
	scope += `
   AND (cardinality($8::text[]) = 0 OR lower(pp.common_craft_type) = ANY($8))
   AND (cardinality($9::text[]) = 0 OR lower(coalesce(pp.pattern_difficulty,'')) = ANY($9))
   AND (cardinality($10::text[]) = 0 OR lower(coalesce(pp.knitting_project_type,'')) = ANY($10) OR lower(coalesce(pp.crochet_project_type,'')) = ANY($10) OR lower(coalesce(pp.quilting_project_type,'')) = ANY($10) OR lower(coalesce(pp.sewing_project_type,'')) = ANY($10))
   AND (cardinality($11::text[]) = 0 OR EXISTS(SELECT 1 FROM unnest(pp.colors) v WHERE lower(v) = ANY($11)))
   AND (cardinality($12::text[]) = 0 OR EXISTS(SELECT 1 FROM unnest(pp.materials) v WHERE lower(v) = ANY($12)))
   AND (cardinality($13::text[]) = 0 OR EXISTS(SELECT 1 FROM unnest(pp.design_tags) v WHERE lower(v) = ANY($13)))
   AND (cardinality($14::text[]) = 0 OR EXISTS(SELECT 1 FROM unnest(pp.project_tags) v WHERE lower(v) = ANY($14)))`
	normalized := searchTSQuery(req.Query)
	cur, err := DecodeRelevanceSearchCursor(req.Cursor, kind, normalized)
	if err != nil {
		return nil, "", err
	}
	concepts, correctionWords := buildSearchMatchingPlan(req.Query)
	plan, err := json.Marshal(concepts)
	if err != nil {
		return nil, "", err
	}

	tags := `ARRAY(SELECT DISTINCT lower(tag) COLLATE "C" FROM unnest(coalesce(p.tags,'{}'::text[])) tag WHERE tag IS NOT NULL ORDER BY lower(tag) COLLATE "C")`
	if project {
		tags = `ARRAY(SELECT DISTINCT lower(tag) COLLATE "C" FROM unnest(coalesce(p.tags,'{}'::text[]) || coalesce(pp.project_tags,'{}'::text[]) || coalesce(pp.design_tags,'{}'::text[])) tag WHERE tag IS NOT NULL ORDER BY lower(tag) COLLATE "C")`
	}
	alts := `ARRAY(SELECT image->>'alt' FROM jsonb_array_elements(CASE WHEN jsonb_typeof(p.images)='array' THEN p.images ELSE '[]'::jsonb END) image WHERE jsonb_typeof(image->'alt')='string')`
	fields := `ARRAY[p.text,array_to_string(` + tags + `,' ')]::text[] || ` + alts
	weights := `ARRAY[1.0,0.2]::real[] || array_fill(0.1::real,ARRAY[cardinality(` + alts + `)])`
	if project {
		fields = `ARRAY[p.text,pp.common_title,pp.pattern_name,array_to_string(pp.materials,' '),array_to_string(` + tags + `,' ')]::text[] || ` + alts
		weights = `ARRAY[1.0,1.0,1.0,0.4,0.2]::real[] || array_fill(0.1::real,ARRAY[cardinality(` + alts + `)])`
	}
	corrections, _ := json.Marshal(correctionWords)
	dictionary, _ := json.Marshal(searchEquivalenceGroups)
	q := `WITH eligible AS MATERIALIZED (
  SELECT p.*, ` + fields + ` AS search_fields, ` + weights + ` AS search_weights,
   EXISTS(SELECT 1 FROM unnest(p.langs) lang WHERE lower(split_part(lang,'-',1))='en') AS search_english
  FROM craftsky_posts p
  LEFT JOIN craftsky_project_posts pp ON pp.uri=p.uri
  WHERE ` + scope + relationshipTopLevelPredicate("$6") + postVisibleModerationPredicate + languageVisibilityPredicate("p", "$6", "$7") + `
 ), reliable AS MATERIALIZED (
  SELECT p.*,craftsky_search_score(search_fields,search_weights,search_english,$1::jsonb) AS reliable_score FROM eligible p
 ), words AS (
  SELECT value AS correction FROM jsonb_array_elements($16::jsonb)
 ), vocabulary AS (
  SELECT DISTINCT correction, word COLLATE "C" AS word
  FROM words CROSS JOIN eligible p
  CROSS JOIN LATERAL unnest(tsvector_to_array(to_tsvector('simple',array_to_string(p.search_fields,' ')))) word
  WHERE (jsonb_array_length($1::jsonb)=1 OR craftsky_search_score(search_fields,search_weights,search_english,$1::jsonb - (correction->>'concept')::int) IS NOT NULL)
    AND craftsky_search_one_edit(correction->>'word',word)
    AND craftsky_search_score(search_fields,search_weights,search_english,
      jsonb_set($1::jsonb,ARRAY[correction->>'concept'],
        jsonb_build_array(concat_ws(' ',nullif(correction->>'before',''),word,nullif(correction->>'after',''))))) IS NOT NULL
 ), bounded AS (
  SELECT *,row_number() OVER (PARTITION BY correction ORDER BY word COLLATE "C") AS candidate_number FROM vocabulary
 ), replacements AS (
  SELECT correction,concat_ws(' ',nullif(correction->>'before',''),word,nullif(correction->>'after','')) AS replacement
  FROM bounded WHERE candidate_number<=8
 ), correction_plans AS (
  SELECT jsonb_set($1::jsonb,ARRAY[correction->>'concept'],
   coalesce((SELECT value FROM jsonb_array_elements($17::jsonb) groups WHERE value ? replacement LIMIT 1),jsonb_build_array(replacement))) AS plan
  FROM replacements
 ), ranked AS (
  SELECT p.*,
   CASE WHEN reliable_score IS NOT NULL THEN 1 ELSE 0 END AS match_tier,
   coalesce(reliable_score, corrected.score) AS relevance_score
  FROM reliable p
  LEFT JOIN LATERAL (
   SELECT max(craftsky_search_score(search_fields,search_weights,search_english,plan)) AS score
   FROM correction_plans WHERE reliable_score IS NULL
  ) corrected ON true
 )
 SELECT ` + postSelectColumns + `,relevance_score,match_tier
 FROM ranked p LEFT JOIN craftsky_project_posts pp ON pp.uri=p.uri LEFT JOIN bluesky_profiles bp ON bp.did=p.did
 WHERE relevance_score IS NOT NULL AND ($15::integer IS NULL OR (match_tier,relevance_score,p.created_at,p.uri)<($15::integer,$3::double precision,$4::timestamptz,$5::text))
 ORDER BY match_tier DESC,relevance_score DESC,p.created_at DESC,p.uri DESC LIMIT $2`

	args := []any{plan, req.Limit + 1, cur.ScorePtr(), cur.CreatedAtPtr(), cur.URIPtr(), viewer, languages}
	for _, key := range []string{"craftType", "patternDifficulty", "projectType", "color", "material", "designTag", "projectTag"} {
		args = append(args, projectFilterValues(req, key))
	}
	args = append(args, cur.TierPtr(), corrections, dictionary)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("submitted search: %w", err)
	}
	defer rows.Close()
	out := make([]SearchPostRow, 0, req.Limit)
	tiers := make([]int, 0, req.Limit)
	for rows.Next() {
		var tier int
		row, err := scanPostRowWithScoreAndTier(rows, &tier)
		if err != nil {
			return nil, "", err
		}
		out = append(out, row)
		tiers = append(tiers, tier)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(out) <= req.Limit {
		return out, "", nil
	}
	out = out[:req.Limit]
	last := out[len(out)-1]
	next, err := encodeTieredRelevanceSearchCursor(kind, normalized, tiers[req.Limit-1], last.Score, last.Post.CreatedAt, last.Post.URI)
	return out, next, err
}
