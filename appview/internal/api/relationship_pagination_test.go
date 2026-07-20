package api_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

func TestDenseRelationshipFilteredTimelineFillsThreeOpaquePages(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	for _, did := range []string{
		"did:plc:viewer",
		"did:plc:alice",
		"did:plc:bob",
		"did:plc:carol",
		"did:plc:dana",
	} {
		seedMember(t, pool, did)
		if did != "did:plc:viewer" {
			seedFollow(t, pool, "did:plc:viewer", did, "follow-"+did[len("did:plc:"):])
		}
	}

	base := time.Date(2026, 7, 19, 16, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:bob", "hidden-before", "hidden", base.Add(10*time.Minute))
	want := []string{
		seedPost(t, pool, "did:plc:alice", "visible-1", "visible", base.Add(9*time.Minute)),
		seedPost(t, pool, "did:plc:alice", "visible-2", "visible", base.Add(8*time.Minute)),
	}
	seedPost(t, pool, "did:plc:carol", "hidden-between-1", "hidden", base.Add(7*time.Minute))
	want = append(want,
		seedPost(t, pool, "did:plc:alice", "visible-3", "visible", base.Add(6*time.Minute)),
		seedPost(t, pool, "did:plc:alice", "visible-4", "visible", base.Add(5*time.Minute)),
	)
	seedPost(t, pool, "did:plc:bob", "hidden-between-2", "hidden", base.Add(4*time.Minute))
	want = append(want,
		seedPost(t, pool, "did:plc:alice", "visible-5", "visible", base.Add(3*time.Minute)),
		seedPost(t, pool, "did:plc:alice", "visible-6", "visible", base.Add(2*time.Minute)),
	)
	seedPost(t, pool, "did:plc:dana", "hidden-after", "hidden", base.Add(time.Minute))

	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ('did:plc:viewer', 'did:plc:bob');
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES
		  ('at://did:plc:carol/app.bsky.graph.block/viewer', 'did:plc:carol', 'viewer', 'cid-carol', 'did:plc:viewer', '{}', now()),
		  ('at://did:plc:viewer/app.bsky.graph.block/dana', 'did:plc:viewer', 'dana', 'cid-dana', 'did:plc:dana', '{}', now());
	`); err != nil {
		t.Fatalf("seed protected actors: %v", err)
	}

	store := api.NewPostStore(pool)
	var got []string
	cursor := ""
	for page := 1; page <= 3; page++ {
		rows, next, err := store.ListTimeline(ctx, "did:plc:viewer", 2, cursor)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if len(rows) != 2 {
			t.Fatalf("page %d length = %d, want full page of 2", page, len(rows))
		}
		got = append(got, timelineURIs(rows)...)
		if page < 3 && next == "" {
			t.Fatalf("page %d cursor is empty before eligible rows are exhausted", page)
		}
		if page == 3 && next != "" {
			t.Fatalf("final cursor = %q, want empty after hidden trailing row", next)
		}
		if next != "" {
			payload, err := envelope.DecodeCursor(next)
			if err != nil {
				t.Fatalf("decode page %d cursor: %v", page, err)
			}
			encoded := next + fmt.Sprint(payload)
			for _, hidden := range []string{"did:plc:bob", "did:plc:carol", "did:plc:dana", "hidden-"} {
				if strings.Contains(encoded, hidden) {
					t.Fatalf("page %d cursor leaked protected identity %q: %s", page, hidden, encoded)
				}
			}
		}
		cursor = next
	}

	if !slices.Equal(got, want) {
		t.Fatalf("eligible union = %v, want stable once-only order %v", got, want)
	}
}
