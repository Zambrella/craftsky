package followergrowth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

func TestStoreCaptureUsesOneSetBasedStatement(t *testing.T) {
	source, err := os.ReadFile("store.go")
	if err != nil {
		t.Fatalf("read store source: %v", err)
	}
	captureSource := string(source)
	if got := strings.Count(captureSource, "INSERT INTO follower_growth_snapshots"); got != 1 {
		t.Fatalf("capture snapshot insert statements = %d, want 1", got)
	}
	if !strings.Contains(captureSource, "FROM craftsky_profile_follower_counts") {
		t.Fatal("capture does not select all canonical profile counts")
	}

	pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
	applyFollowerGrowthMigration(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid');
	`); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}

	var plan string
	if err := pool.QueryRow(ctx, `
		EXPLAIN (FORMAT JSON)
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		)
		SELECT profile_did, $1, follower_count, $2
		FROM craftsky_profile_follower_counts
	`, time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, time.August, 25, 0, 0, 2, 0, time.UTC)).Scan(&plan); err != nil {
		t.Fatalf("explain set-based capture: %v", err)
	}
	if !strings.Contains(plan, `"Operation": "Insert"`) {
		t.Fatalf("capture plan is not one insert operation: %s", plan)
	}
	if strings.Contains(plan, `"Parent Relationship": "SubPlan"`) {
		t.Fatalf("capture plan contains repeated subplan work: %s", plan)
	}
}
