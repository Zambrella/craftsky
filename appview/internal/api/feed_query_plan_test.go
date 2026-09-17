package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestPrimaryFeedPaginationQueriesAvoidFullHotTableScans(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	seedFeedQueryPlanCardinality(t, pool)

	tracer := &feedQueryCapture{}
	capturedPool := newFeedQueryCapturePool(t, pool, tracer)
	store := api.NewPostStore(capturedPool)

	t.Run("timeline", func(t *testing.T) {
		_, cursor, err := store.ListTimeline(ctx, "did:plc:plan-viewer", 20, "")
		if err != nil {
			t.Fatalf("list first timeline page: %v", err)
		}
		if cursor == "" {
			t.Fatal("first timeline page did not produce a cursor")
		}

		tracer.reset()
		if _, _, err := store.ListTimeline(ctx, "did:plc:plan-viewer", 20, cursor); err != nil {
			t.Fatalf("list timeline cursor page: %v", err)
		}
		query, args := tracer.queryContaining(t, "WITH eligible_authors AS")
		plan := explainCapturedFeedQuery(t, pool, query, args...)

		assertPlanHasIndexedAccess(t, plan, "craftsky_posts")
		assertPlanHasIndexedAccess(t, plan, "craftsky_reposts")
		assertNoSequentialScan(t, plan, "craftsky_posts", "craftsky_reposts")
	})

	t.Run("notifications", func(t *testing.T) {
		_, cursor, err := store.ListNotifications(ctx, "did:plc:plan-viewer", 20, "")
		if err != nil {
			t.Fatalf("list first notification page: %v", err)
		}
		if cursor == "" {
			t.Fatal("first notification page did not produce a cursor")
		}

		tracer.reset()
		if _, _, err := store.ListNotifications(ctx, "did:plc:plan-viewer", 20, cursor); err != nil {
			t.Fatalf("list notification cursor page: %v", err)
		}
		query, args := tracer.queryContaining(t, "WITH page_events AS MATERIALIZED")
		plan := explainCapturedFeedQuery(t, pool, query, args...)

		assertPlanHasIndexedAccess(t, plan, "notification_events")
		assertNoSequentialScan(t, plan, "notification_events")
	})
}

func seedFeedQueryPlanCardinality(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:plan-viewer', 'plan-viewer-cid');
		INSERT INTO craftsky_profiles (did, record_cid)
		SELECT 'did:plc:plan-actor-' || n, 'plan-actor-cid-' || n
		FROM generate_series(1, 1000) AS n;

		INSERT INTO owner_lifecycles (
			owner_did, state, generation, auth_epoch, transition_reason,
			transitioned_at, created_at, updated_at
		)
		SELECT did, 'active', 1, 1, 'queryPlanTest',
		       '2026-09-17T00:00:00Z', '2026-09-17T00:00:00Z', '2026-09-17T00:00:00Z'
		FROM craftsky_profiles;

		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at, indexed_at)
		SELECT 'at://did:plc:plan-viewer/app.bsky.graph.follow/' || n,
		       'did:plc:plan-viewer', n::text, 'follow-cid-' || n,
		       'did:plc:plan-actor-' || n, '{}',
		       '2026-09-17T00:00:00Z'::timestamptz + n * interval '1 second',
		       '2026-09-17T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM generate_series(1, 20) AS n;

		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, indexed_at, profile_sort_at
		)
		SELECT 'at://did:plc:plan-actor-' || author || '/social.craftsky.feed.post/' || post,
		       'did:plc:plan-actor-' || author, post::text,
		       'post-cid-' || author || '-' || post, 'query plan post', '{}',
		       '2026-09-17T00:00:00Z'::timestamptz + (author * 20 + post) * interval '1 second',
		       '2026-09-17T00:00:00Z'::timestamptz + (author * 20 + post) * interval '1 second',
		       '2026-09-17T00:00:00Z'::timestamptz + (author * 20 + post) * interval '1 second'
		FROM generate_series(1, 1000) AS author
		CROSS JOIN generate_series(1, 20) AS post;

		INSERT INTO craftsky_reposts (
			uri, did, rkey, cid, subject_uri, subject_cid, record, created_at, indexed_at
		)
		SELECT 'at://did:plc:plan-actor-' || actor || '/social.craftsky.feed.repost/' || n,
		       'did:plc:plan-actor-' || actor, n::text, 'repost-cid-' || n,
		       'at://did:plc:plan-actor-' || subject_actor || '/social.craftsky.feed.post/' || subject_post,
		       'post-cid-' || subject_actor || '-' || subject_post, '{}',
		       '2026-09-18T00:00:00Z'::timestamptz + n * interval '1 second',
		       '2026-09-18T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM (
			SELECT n, ((n - 1) % 1000) + 1 AS actor,
			       (((n - 1) % 1000 + (n - 1) / 1000 + 1) % 1000) + 1 AS subject_actor,
			       ((n - 1) / 1000) + 1 AS subject_post
			FROM generate_series(1, 10000) AS n
		) AS seeded_reposts;

		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			source_uri, source_cid, source_rkey, eligibility_scope,
			recipient_followed_actor, push_enabled_snapshot, state,
			first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
		)
		SELECT md5(n::text)::uuid,
		       CASE WHEN n % 200 = 0 THEN 'did:plc:plan-viewer' ELSE 'did:plc:plan-recipient-' || (n % 200) END,
		       'did:plc:plan-actor-' || ((n - 1) % 1000 + 1),
		       'follow', 'plan-subject-' || n,
		       'at://did:plc:plan-actor-' || ((n - 1) % 1000 + 1) || '/app.bsky.graph.follow/' || n,
		       'notification-cid-' || n, n::text, 'everyone', false, true, 'active',
		       '2026-09-19T00:00:00Z'::timestamptz + n * interval '1 second',
		       '2026-09-19T00:00:00Z'::timestamptz + n * interval '1 second',
		       '2026-09-19T00:00:00Z'::timestamptz + n * interval '1 second',
		       '2026-09-19T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM generate_series(1, 12000) AS n;

		ANALYZE craftsky_profiles;
		ANALYZE owner_lifecycles;
		ANALYZE atproto_follows;
		ANALYZE craftsky_posts;
		ANALYZE craftsky_reposts;
		ANALYZE notification_events;
	`); err != nil {
		t.Fatalf("seed realistic feed cardinality: %v", err)
	}
}

type capturedFeedQuery struct {
	sql  string
	args []any
}

type feedQueryCapture struct {
	queries []capturedFeedQuery
}

func (capture *feedQueryCapture) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	capture.queries = append(capture.queries, capturedFeedQuery{
		sql:  data.SQL,
		args: append([]any(nil), data.Args...),
	})
	return ctx
}

func (*feedQueryCapture) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (capture *feedQueryCapture) reset() {
	capture.queries = nil
}

func (capture *feedQueryCapture) queryContaining(t *testing.T, fragment string) (string, []any) {
	t.Helper()
	for _, query := range capture.queries {
		if strings.Contains(query.sql, fragment) {
			return query.sql, query.args
		}
	}
	t.Fatalf("did not capture query containing %q", fragment)
	return "", nil
}

func newFeedQueryCapturePool(t *testing.T, source *pgxpool.Pool, tracer pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	config := source.Config()
	config.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("create query-capture pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func explainCapturedFeedQuery(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var encoded []byte
	if err := pool.QueryRow(t.Context(), "EXPLAIN (FORMAT JSON, COSTS OFF) "+query, args...).Scan(&encoded); err != nil {
		t.Fatalf("explain captured feed query: %v", err)
	}
	return string(encoded)
}

func assertPlanHasIndexedAccess(t *testing.T, encodedPlan string, relation string) {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(encodedPlan), &decoded); err != nil {
		t.Fatalf("decode query plan: %v", err)
	}
	found := false
	visitQueryPlan(decoded, func(node map[string]any) {
		if node["Relation Name"] != relation {
			return
		}
		switch node["Node Type"] {
		case "Index Scan", "Index Only Scan", "Bitmap Heap Scan":
			found = true
		}
	})
	if !found {
		t.Fatalf("query plan has no indexed access for %s:\n%s", relation, encodedPlan)
	}
}

func assertNoSequentialScan(t *testing.T, encodedPlan string, relations ...string) {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(encodedPlan), &decoded); err != nil {
		t.Fatalf("decode query plan: %v", err)
	}
	forbidden := make(map[string]bool, len(relations))
	for _, relation := range relations {
		forbidden[relation] = true
	}
	var scanned []string
	visitQueryPlan(decoded, func(node map[string]any) {
		relation, _ := node["Relation Name"].(string)
		if node["Node Type"] == "Seq Scan" && forbidden[relation] {
			scanned = append(scanned, relation)
		}
	})
	if len(scanned) != 0 {
		t.Fatalf("query plan sequentially scans hot relations %v:\n%s", scanned, encodedPlan)
	}
}

func visitQueryPlan(value any, visit func(map[string]any)) {
	switch value := value.(type) {
	case []any:
		for _, child := range value {
			visitQueryPlan(child, visit)
		}
	case map[string]any:
		visit(value)
		for _, child := range value {
			visitQueryPlan(child, visit)
		}
	}
}
