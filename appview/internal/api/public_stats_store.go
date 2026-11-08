package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PublicStatsStore struct {
	pool *pgxpool.Pool
}

func NewPublicStatsStore(pool *pgxpool.Pool) *PublicStatsStore {
	return &PublicStatsStore{pool: pool}
}

// One statement gives the totals and craft breakdown the same database snapshot.
// The shared anonymous predicates exclude unavailable owners and content rather
// than borrowing authenticated viewer/owner visibility exceptions.
func (s *PublicStatsStore) PublicStats(ctx context.Context) (*PublicStats, error) {
	const finishedStatus = "social.craftsky.feed.defs#finished"
	query := `
 WITH public_accounts AS MATERIALIZED (
   SELECT cp.did FROM craftsky_profiles cp LEFT JOIN bluesky_profiles bp ON bp.did=cp.did WHERE TRUE ` + publicOwnerPredicate + `
 ), public_posts AS MATERIALIZED (
   SELECT p.is_project, p.quote_uri, pp.uri AS project_uri,
          pp.common_craft_type, pp.common_status
   FROM craftsky_posts p
   JOIN public_accounts cp ON cp.did=p.did
   LEFT JOIN craftsky_project_posts pp ON pp.uri=p.uri
   WHERE p.reply_root_uri IS NULL AND p.reply_parent_uri IS NULL
   ` + publicPostPredicate + `
 ), projects AS MATERIALIZED (
   SELECT common_craft_type, common_status FROM public_posts
   WHERE is_project AND project_uri IS NOT NULL AND quote_uri IS NULL
 ), crafts AS (
   SELECT common_craft_type, count(*) AS count
   FROM projects GROUP BY common_craft_type
 )
 SELECT statement_timestamp(),
        (SELECT count(*) FROM public_accounts),
        (SELECT count(*) FROM public_posts),
        (SELECT count(*) FROM projects),
        (SELECT count(*) FROM projects WHERE common_status=$1),
        COALESCE((SELECT jsonb_agg(jsonb_build_object('craft', common_craft_type, 'count', count)
                         ORDER BY count DESC, common_craft_type ASC) FROM crafts), '[]'::jsonb)
 `
	stats := &PublicStats{}
	var crafts []byte
	err := s.pool.QueryRow(ctx, query, finishedStatus).Scan(
		&stats.GeneratedAt, &stats.Accounts, &stats.Posts,
		&stats.ProjectPosts, &stats.FinishedProjectPosts, &crafts,
	)
	if err != nil {
		return nil, fmt.Errorf("public statistics read: %w", err)
	}
	if err := json.Unmarshal(crafts, &stats.ProjectsByCraft); err != nil {
		return nil, fmt.Errorf("public statistics craft counts: %w", err)
	}
	stats.GeneratedAt = stats.GeneratedAt.UTC()
	return stats, nil
}
