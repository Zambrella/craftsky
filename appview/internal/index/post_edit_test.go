package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-001: accepted content replacement does not move a post in the author's
// chronology or remove its save and pin relationships.
func TestPostEditPreservesPublicationPositionAndRelationships(t *testing.T) {
	for _, imported := range []bool{false, true} {
		name := "ordinary"
		if imported {
			name = "future-dated import"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			pool := testdb.WithMigratedSchema(t)
			seedCraftskyMember(t, pool, "did:plc:author")
			idx := index.NewCraftskyPost(pool, testLogger())
			record := map[string]any{"$type": "social.craftsky.feed.post", "text": "before", "sponsored": false, "createdAt": "2026-01-01T00:00:00Z"}
			if imported {
				record["externalImport"] = map[string]string{"source": "instagram"}
				record["createdAt"] = "2099-01-01T00:00:00Z"
			}
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			event := tap.Event{URI: "at://did:plc:author/social.craftsky.feed.post/3existingpost", DID: "did:plc:author", Rkey: "3existingpost", Collection: "social.craftsky.feed.post", CID: "bafy-before", Action: "create", Record: raw}
			if err := idx.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			var originalCreated, originalSort time.Time
			if err := pool.QueryRow(ctx, `SELECT created_at,profile_sort_at FROM craftsky_posts WHERE uri=$1`, event.URI).Scan(&originalCreated, &originalSort); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO saved_posts(owner_did,post_uri,saved_at) VALUES('did:plc:author',$1,now())`, event.URI); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at) VALUES('did:plc:author','standard',$1,$2,now(),now())`, event.URI, uuid.New()); err != nil {
				t.Fatal(err)
			}
			record["text"] = "after"
			record["editedAt"] = "2026-10-26T12:00:00Z"
			event.Record, _ = json.Marshal(record)
			event.CID = "bafy-after"
			event.Action = "update"
			if err := idx.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			var created, sort time.Time
			var text string
			var posts, saves, pins int
			if err := pool.QueryRow(ctx, `SELECT text,created_at,profile_sort_at,(SELECT count(*) FROM craftsky_posts),(SELECT count(*) FROM saved_posts),(SELECT count(*) FROM profile_pins) FROM craftsky_posts WHERE uri=$1`, event.URI).Scan(&text, &created, &sort, &posts, &saves, &pins); err != nil {
				t.Fatal(err)
			}
			if text != "after" || !created.Equal(originalCreated) || !sort.Equal(originalSort) || posts != 1 || saves != 1 || pins != 1 {
				t.Fatalf("edit changed publication: text=%s created=%s/%s sort=%s/%s counts=%d,%d,%d", text, created, originalCreated, sort, originalSort, posts, saves, pins)
			}
		})
	}
}
