package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FacetStore struct {
	pool          *pgxpool.Pool
	resolver      HandleResolver
	identityCache *IdentityCacheStore
}

func NewFacetStore(pool *pgxpool.Pool, resolver ...HandleResolver) *FacetStore {
	var handleResolver HandleResolver
	if len(resolver) > 0 {
		handleResolver = resolver[0]
	}
	return &FacetStore{pool: pool, resolver: handleResolver, identityCache: NewIdentityCacheStore(pool)}
}

func RankMentionSuggestionRows(rows []MentionSuggestionRow, query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	sort.SliceStable(rows, func(i, j int) bool {
		a := rows[i]
		b := rows[j]
		if a.ViewerIsFollowing != b.ViewerIsFollowing {
			return a.ViewerIsFollowing
		}
		aPrefix := strings.HasPrefix(strings.ToLower(a.Handle), query)
		bPrefix := strings.HasPrefix(strings.ToLower(b.Handle), query)
		if aPrefix != bPrefix {
			return aPrefix
		}
		return strings.ToLower(a.Handle) < strings.ToLower(b.Handle)
	})
}

func NormalizeHashtagSuggestionRows(rows []HashtagSuggestionRow) []HashtagSuggestionRow {
	counts := map[string]int{}
	for _, row := range rows {
		tag := strings.ToLower(strings.TrimSpace(row.Tag))
		if tag == "" {
			continue
		}
		count := row.PostsLast28Days
		if count < 0 {
			count = 0
		}
		counts[tag] += count
	}
	out := make([]HashtagSuggestionRow, 0, len(counts))
	for tag, count := range counts {
		out = append(out, HashtagSuggestionRow{Tag: tag, PostsLast28Days: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PostsLast28Days != out[j].PostsLast28Days {
			return out[i].PostsLast28Days > out[j].PostsLast28Days
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

func EscapeFacetLikePattern(query string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(query)
}

func (s *FacetStore) SearchMentionSuggestions(ctx context.Context, viewerDID syntax.DID, query string, limit int, now time.Time) ([]MentionSuggestionRow, error) {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" || limit <= 0 {
		return []MentionSuggestionRow{}, nil
	}
	likeQuery := EscapeFacetLikePattern(queryLower)
	rows, err := s.pool.Query(ctx, `
		SELECT
			ic.did,
			ic.handle,
			bp.display_name,
			bp.avatar_cid,
			bp.avatar_mime,
			EXISTS (
				SELECT 1 FROM atproto_follows f
				WHERE f.did = $1 AND f.subject_did = ic.did
			) AS viewer_is_following
		FROM atproto_identity_cache ic
		JOIN craftsky_profiles cp ON cp.did = ic.did
		LEFT JOIN bluesky_profiles bp ON bp.did = ic.did
		WHERE ic.resolved_at >= $2
		  AND (
			ic.handle_lower LIKE '%' || $3 || '%' ESCAPE '\'
			OR lower(coalesce(bp.display_name, '')) LIKE '%' || $3 || '%' ESCAPE '\'
		  )
		ORDER BY
			EXISTS (
				SELECT 1 FROM atproto_follows f
				WHERE f.did = $1 AND f.subject_did = ic.did
			) DESC,
			CASE WHEN ic.handle_lower LIKE $3 || '%' ESCAPE '\' THEN 0 ELSE 1 END ASC,
			ic.handle_lower ASC
		LIMIT $4
	`, viewerDID.String(), now.Add(-identityCacheFreshness), likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("facet mention suggestions: %w", err)
	}
	defer rows.Close()

	out := make([]MentionSuggestionRow, 0, limit)
	for rows.Next() {
		var row MentionSuggestionRow
		if err := rows.Scan(&row.DID, &row.Handle, &row.DisplayName, &row.AvatarCID, &row.AvatarMime, &row.ViewerIsFollowing); err != nil {
			return nil, fmt.Errorf("facet mention suggestion scan: %w", err)
		}
		row.IsCraftskyProfile = true
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("facet mention suggestion rows: %w", err)
	}
	return out, nil
}

func (s *FacetStore) SearchHashtagSuggestions(ctx context.Context, query string, limit int, now time.Time) ([]HashtagSuggestionRow, error) {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" || limit <= 0 {
		return []HashtagSuggestionRow{}, nil
	}
	likeQuery := EscapeFacetLikePattern(queryLower)
	rows, err := s.pool.Query(ctx, `
		SELECT lower(trim(tag.raw_tag)) AS tag, COUNT(DISTINCT p.uri)::int AS posts_last_28_days
		FROM craftsky_posts p
		CROSS JOIN LATERAL unnest(p.tags) AS tag(raw_tag)
		WHERE p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND p.created_at >= $1
		  AND trim(tag.raw_tag) <> ''
		  AND lower(trim(tag.raw_tag)) LIKE '%' || $2 || '%' ESCAPE '\'
		GROUP BY lower(trim(tag.raw_tag))
		ORDER BY posts_last_28_days DESC, tag ASC
		LIMIT $3
	`, now.Add(-28*24*time.Hour), likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("facet hashtag suggestions: %w", err)
	}
	defer rows.Close()
	out := make([]HashtagSuggestionRow, 0, limit)
	for rows.Next() {
		var row HashtagSuggestionRow
		if err := rows.Scan(&row.Tag, &row.PostsLast28Days); err != nil {
			return nil, fmt.Errorf("facet hashtag suggestion scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("facet hashtag suggestion rows: %w", err)
	}
	return out, nil
}

func (s *FacetStore) ResolveMention(ctx context.Context, handle syntax.Handle, now time.Time) (IdentityCacheRow, error) {
	if s.identityCache == nil {
		return IdentityCacheRow{}, ErrMentionNotFound
	}
	if cached, err := s.identityCache.FreshByHandle(ctx, handle, now); err != nil {
		return IdentityCacheRow{}, err
	} else if cached != nil {
		return *cached, nil
	}
	if s.resolver == nil {
		return IdentityCacheRow{}, ErrMentionNotFound
	}
	did, err := s.resolver.ResolveDID(ctx, handle)
	if err != nil || did.String() == "" {
		return IdentityCacheRow{}, ErrMentionNotFound
	}
	isCraftsky, err := s.identityCache.IsCraftskyProfile(ctx, did)
	if err != nil {
		return IdentityCacheRow{}, err
	}
	if !isCraftsky {
		return IdentityCacheRow{}, ErrMentionNotFound
	}
	canonicalHandle, err := s.resolver.ResolveHandle(ctx, did)
	if err != nil || canonicalHandle.String() == "" {
		return IdentityCacheRow{}, ErrMentionNotFound
	}
	if err := s.identityCache.Upsert(ctx, did, canonicalHandle, now); err != nil {
		return IdentityCacheRow{}, err
	}
	return IdentityCacheRow{DID: did, Handle: canonicalHandle, ResolvedAt: now}, nil
}
