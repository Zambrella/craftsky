package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

// IT-014 / NFR-001, RULE-002 / AC-011,015.
// Exercise the public Tap indexer while reusing migrated search fixtures.
func TestSearchImprovementsProjectionConvergence(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			pool, store, now := searchFixture(t)
			ctx := context.Background()
			idx := index.NewCraftskyPost(pool, nilLogger())
			uri := syntax.ATURI("at://did:plc:author/social.craftsky.feed.post/convergence")
			event := tap.Event{URI: uri, DID: "did:plc:author", Rkey: "convergence", Collection: "social.craftsky.feed.post", Action: "create"}
			makeRecord := func(prefix string) json.RawMessage {
				text := prefix + "caption #" + prefix + "tag"
				record := map[string]any{
					"$type": "social.craftsky.feed.post", "text": text, "createdAt": now.Format("2006-01-02T15:04:05Z07:00"), "langs": []string{"en"},
					"facets": []any{map[string]any{"index": map[string]int{"byteStart": strings.Index(text, "#"), "byteEnd": len(text)}, "features": []any{map[string]any{"$type": "app.bsky.richtext.facet#tag", "tag": prefix + "tag"}}}},
					"images": []any{map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]string{"$link": "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"}, "mimeType": "image/jpeg", "size": 123}, "alt": prefix + "alt"}},
				}
				if project {
					record["project"] = map[string]any{"common": map[string]any{"craftType": "social.craftsky.feed.defs#knitting", "title": prefix + "title", "materials": []any{map[string]string{"text": prefix + "material"}}, "tags": []string{prefix + "projecttag"}}}
				}
				raw, err := json.Marshal(record)
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
			for _, prefix := range []string{"old", "new"} {
				event.CID = syntax.CID(prefix + "-cid")
				event.Record = makeRecord(prefix)
				if prefix == "new" {
					event.Action = "update"
				}
				if err := idx.Handle(ctx, event); err != nil {
					t.Fatal(err)
				}
				if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES($1,'post',$2,'clear') ON CONFLICT(subject_uri) DO UPDATE SET source_cid=excluded.source_cid,visibility_state='clear'`, uri, event.CID); err != nil {
					t.Fatal(err)
				}
				if err := idx.Handle(ctx, event); err != nil {
					t.Fatalf("replay: %v", err)
				}
				fields := []string{"caption", "tag", "alt"}
				if project {
					fields = append(fields, "title", "material", "projecttag")
				}
				for _, field := range fields {
					rows, _ := submittedSearch(t, store, project, prefix+field, 10, "", now)
					if len(rows) != 1 || rows[0].Post.URI != uri.String() {
						t.Fatalf("%s%s not searchable: %v", prefix, field, searchURIs(rows))
					}
					if prefix == "new" {
						rows, _ = submittedSearch(t, store, project, "old"+field, 10, "", now)
						if len(rows) != 0 {
							t.Fatalf("obsolete %s remains", field)
						}
					}
				}
				var cid string
				var count int
				if err := pool.QueryRow(ctx, `SELECT count(*),min(cid) FROM craftsky_posts WHERE uri=$1`, uri).Scan(&count, &cid); err != nil {
					t.Fatal(err)
				}
				if count != 1 || cid != event.CID.String() {
					t.Fatal("replay/search changed identity")
				}
			}
			event.Action = "delete"
			event.Record = nil
			if err := idx.Handle(ctx, event); err != nil {
				t.Fatal(err)
			}
			rows, _ := submittedSearch(t, store, project, "newcaption", 10, "", now)
			if len(rows) != 0 {
				t.Fatal("deleted result remains")
			}
		})
	}
}
