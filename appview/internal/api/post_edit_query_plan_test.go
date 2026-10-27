package api_test

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

// REG-004 / IT-001: preserving chronology also preserves bounded indexed access
// for the actual chronological read query, rather than sorting the hot table.
func TestPostEditStableChronologyRetainsIndexedSearch(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	seedFeedQueryPlanCardinality(t, pool)
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_posts SET tags=ARRAY['sewing']; ANALYZE craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	capture := &feedQueryCapture{}
	capturedPool := newFeedQueryCapturePool(t, pool, capture)
	store := api.NewSearchStore(capturedPool, nil)
	if _, _, err := store.SearchHashtagPostsWithLanguages(t.Context(), "did:plc:plan-viewer", nil, "sewing", api.SearchSortChronological, 20, "", time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	query, args := capture.queryContaining(t, "0::double precision AS popularity_score")
	plan := explainCapturedFeedQuery(t, pool, query, args...)
	assertPlanHasIndexedAccess(t, plan, "craftsky_posts")
	assertNoSequentialScan(t, plan, "craftsky_posts")
}
