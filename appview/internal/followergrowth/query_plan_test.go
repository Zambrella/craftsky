package followergrowth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

func TestStoreCapturePlanIsSingleSetBasedInsert(t *testing.T) {
	pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
	applyFollowerGrowthMigration(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid');
		CREATE TABLE captured_follower_growth_queries (query TEXT NOT NULL);
		CREATE FUNCTION capture_follower_growth_query() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			INSERT INTO captured_follower_growth_queries(query) VALUES (current_query());
			RETURN NULL;
		END;
		$$;
		CREATE TRIGGER capture_follower_growth_query
		BEFORE INSERT ON follower_growth_snapshots
		FOR EACH STATEMENT EXECUTE FUNCTION capture_follower_growth_query();
	`); err != nil {
		t.Fatalf("seed profiles and query observer: %v", err)
	}

	snapshotDate := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	capturedAt := snapshotDate.Add(2 * time.Second)
	result, err := NewStore(pool).Capture(ctx, snapshotDate, capturedAt)
	if err != nil {
		t.Fatalf("capture follower growth: %v", err)
	}
	if result.CapturedProfileCount != 2 {
		t.Fatalf("captured profiles = %d, want 2", result.CapturedProfileCount)
	}
	var captureQuery string
	if err := pool.QueryRow(ctx, `SELECT query FROM captured_follower_growth_queries`).Scan(&captureQuery); err != nil {
		t.Fatalf("read observed capture query: %v", err)
	}
	var encodedPlan []byte
	if err := pool.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+captureQuery, snapshotDate, capturedAt).Scan(&encodedPlan); err != nil {
		t.Fatalf("explain set-based capture: %v", err)
	}
	var plans []struct {
		Plan queryPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal(encodedPlan, &plans); err != nil {
		t.Fatalf("decode capture plan: %v", err)
	}
	if len(plans) != 1 || plans[0].Plan.NodeType != "ModifyTable" || plans[0].Plan.Operation != "Insert" {
		t.Fatalf("capture root plan = %+v, want one Insert ModifyTable", plans)
	}
	if got := countPlanRelationship(plans[0].Plan, "SubPlan"); got != 0 {
		t.Fatalf("capture plan contains %d repeated subplans: %s", got, encodedPlan)
	}
}

type queryPlanNode struct {
	NodeType           string          `json:"Node Type"`
	Operation          string          `json:"Operation"`
	ParentRelationship string          `json:"Parent Relationship"`
	Plans              []queryPlanNode `json:"Plans"`
}

func countPlanRelationship(node queryPlanNode, relationship string) int {
	count := 0
	if node.ParentRelationship == relationship {
		count++
	}
	for _, child := range node.Plans {
		count += countPlanRelationship(child, relationship)
	}
	return count
}
