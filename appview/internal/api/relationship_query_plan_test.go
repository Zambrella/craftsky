package api_test

import (
	"context"
	"strings"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestRelationshipFilteringQueryPlanUsesBidirectionalIndexes(t *testing.T) {
	pool := testdb.WithSchema(t, postStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `SET enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scans: %v", err)
	}

	rows, err := pool.Query(ctx, `
		EXPLAIN (FORMAT TEXT, COSTS OFF)
		SELECT cp.did
		FROM craftsky_profiles cp
		WHERE cp.did = ANY($2::text[])
		  AND NOT EXISTS (
			SELECT 1 FROM actor_mutes m
			WHERE m.owner_did = $1 AND m.subject_did = cp.did
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM atproto_blocks b
			WHERE (b.blocker_did = $1 AND b.subject_did = cp.did)
			   OR (b.subject_did = $1 AND b.blocker_did = cp.did)
		  )
	`, "did:plc:viewer", []string{"did:plc:alice", "did:plc:bob"})
	if err != nil {
		t.Fatalf("explain relationship-filtered selection: %v", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan query plan: %v", err)
		}
		planLines = append(planLines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read query plan: %v", err)
	}
	plan := strings.Join(planLines, "\n")
	for _, index := range []string{
		"actor_mutes_pkey",
		"atproto_blocks_blocker_subject_idx",
		"atproto_blocks_subject_blocker_idx",
	} {
		if !strings.Contains(plan, index) {
			t.Fatalf("query plan does not use %s:\n%s", index, plan)
		}
	}
}
