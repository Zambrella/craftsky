package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"reflect"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

func TestPostCraftsPreservedAcrossProjectionUpdates(t *testing.T) {
	pool := hashtagIndexerTestPool(t)
	seedCraftskyMember(t, pool, "did:plc:m")
	projector := index.NewCraftskyPost(pool, testLogger())
	event := tap.Event{URI: "at://did:plc:m/social.craftsky.feed.post/crafts", DID: "did:plc:m", Rkey: "crafts", Collection: "social.craftsky.feed.post", Action: "create"}
	for i, crafts := range []string{
		`[{"craftType":"social.craftsky.feed.defs#sewing","futureContext":"retained"}]`,
		`[{"craftType":"social.craftsky.feed.defs#embroidery"}]`,
		`[]`,
		``,
	} {
		event.CID = syntax.CID(fmt.Sprintf("bafy-crafts-%d", i))
		record := map[string]any{"$type": "social.craftsky.feed.post", "text": "hello", "sponsored": false, "createdAt": fixedCreatedAt}
		var wanted any
		if crafts != "" {
			if err := json.Unmarshal([]byte(crafts), &wanted); err != nil {
				t.Fatal(err)
			}
			record["crafts"] = wanted
		}
		var err error
		event.Record, err = json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := projector.Handle(context.Background(), event); err != nil {
			t.Fatal(err)
		}
		var raw []byte
		var isProject bool
		if err := pool.QueryRow(context.Background(), `SELECT record,is_project FROM craftsky_posts WHERE uri=$1`, event.URI).Scan(&raw, &isProject); err != nil {
			t.Fatal(err)
		}
		var stored map[string]any
		if err := json.Unmarshal(raw, &stored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored["crafts"], wanted) || isProject {
			t.Fatalf("projection changed crafts/classification: %s", raw)
		}
		event.Action = "update"
	}
}
