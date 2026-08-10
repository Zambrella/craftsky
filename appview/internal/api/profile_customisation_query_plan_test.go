package api_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestIdentityCustomisationHydratorBatchCountDoesNotGrowWithPageSize(t *testing.T) {
	t.Parallel()
	for _, size := range []int{1, 25, 250} {
		t.Run(fmt.Sprintf("page_%d", size), func(t *testing.T) {
			t.Parallel()
			reader := &fakeProfileCustomisationBatchReader{}
			hydrator := api.NewIdentityCustomisationHydrator(reader)
			var body strings.Builder
			body.WriteString(`{"items":[`)
			for i := range size {
				if i > 0 {
					body.WriteByte(',')
				}
				_, _ = fmt.Fprintf(
					&body,
					`{"author":{"did":"did:plc:user%d","handle":"user%d.example"}}`,
					i%5,
					i%5,
				)
			}
			body.WriteString(`]}`)

			if _, err := hydrator.HydrateJSON(
				context.Background(),
				[]byte(body.String()),
			); err != nil {
				t.Fatalf("hydrate page: %v", err)
			}
			if reader.calls != 1 {
				t.Fatalf("batch calls = %d, want 1 for page size %d", reader.calls, size)
			}
			wantDIDs := min(size, 5)
			if len(reader.dids) != wantDIDs {
				t.Fatalf("deduplicated DIDs = %d, want %d", len(reader.dids), wantDIDs)
			}
		})
	}
}

func TestProfileCustomisationBatchQueryUsesOwnerIndexes(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read profile customisation migration: %v", err)
	}
	pool := testdb.WithSchema(t, profileCustomisationStoreTestDDL+string(migration))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `SET enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scans: %v", err)
	}

	rows, err := pool.Query(
		ctx,
		"EXPLAIN (FORMAT TEXT, COSTS OFF) "+api.ProfileCustomisationBatchQuery,
		[]string{"did:plc:alice", "did:plc:bob"},
	)
	if err != nil {
		t.Fatalf("explain batch query: %v", err)
	}
	defer rows.Close()
	var explained strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan query plan: %v", err)
		}
		explained.WriteString(line)
		explained.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read query plan: %v", err)
	}
	plan := explained.String()
	for _, index := range []string{
		"craftsky_profiles_pkey",
		"profile_customisations_pkey",
	} {
		if !strings.Contains(plan, index) {
			t.Fatalf("plan does not use %s:\n%s", index, plan)
		}
	}
}
