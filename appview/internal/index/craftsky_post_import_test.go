package index_test

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-010, REG-001, and REG-006: provenance follows the authoritative record
// through create, replacement, unknown-source replacement, and deletion.
func TestCraftskyPostImportProvenanceFollowsAuthoritativeReplacement(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:author")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:author/social.craftsky.feed.post/imported",
		CID:        "bafy-imported-1",
		DID:        "did:plc:author",
		Rkey:       "imported",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type":"social.craftsky.feed.post",
			"text":"historical",
			"createdAt":"2019-03-04T05:06:07Z",
			"externalImport":{"source":"instagram"}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("create import: %v", err)
	}
	assertPostImportClassification(t, pool, string(ev.URI), stringPointer("instagram"), time.Date(2019, 3, 4, 5, 6, 7, 0, time.UTC), false)
	assertStoredPostRecord(t, pool, string(ev.URI), ev.Record)

	ev.Action = "update"
	ev.CID = "bafy-imported-2"
	ev.Record = json.RawMessage(`{
		"$type":"social.craftsky.feed.post",
		"text":"historical edit",
		"createdAt":"2019-03-04T05:06:07Z",
		"externalImport":{"source":"instagram"}
	}`)
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("retaining replacement: %v", err)
	}
	assertPostImportClassification(t, pool, string(ev.URI), stringPointer("instagram"), time.Date(2019, 3, 4, 5, 6, 7, 0, time.UTC), false)
	assertStoredPostRecord(t, pool, string(ev.URI), ev.Record)

	ev.CID = "bafy-ordinary-3"
	ev.Record = json.RawMessage(`{
		"$type":"social.craftsky.feed.post",
		"text":"ordinary replacement",
		"createdAt":"2019-03-04T05:06:07Z"
	}`)
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("omitting replacement: %v", err)
	}
	assertPostImportClassification(t, pool, string(ev.URI), nil, time.Time{}, true)
	assertStoredPostRecord(t, pool, string(ev.URI), ev.Record)

	ev.CID = "bafy-future-4"
	ev.Record = json.RawMessage(`{
		"$type":"social.craftsky.feed.post",
		"text":"future source replacement",
		"createdAt":"2019-03-04T05:06:07Z",
		"externalImport":{"source":"future-source"}
	}`)
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("unknown-source replacement: %v", err)
	}
	assertPostImportClassification(t, pool, string(ev.URI), nil, time.Time{}, true)
	assertStoredPostRecord(t, pool, string(ev.URI), ev.Record)

	ev.Action = "delete"
	ev.Record = nil
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("delete import: %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&count); err != nil {
		t.Fatalf("count deleted import: %v", err)
	}
	if count != 0 {
		t.Fatalf("deleted import count = %d, want 0", count)
	}
}

// AT-008, IT-013, and REG-004: the import source is silent while later
// like/repost/quote/reply activity retains the ordinary notification path.
func TestCraftskyPostImportSuppressesOnlySourceNotifications(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply notification migration: %v", err)
	}

	for _, did := range []string{"did:plc:author", "did:plc:recipient", "did:plc:actor"} {
		seedCraftskyMember(t, pool, did)
	}
	const targetURI = "at://did:plc:recipient/social.craftsky.feed.post/target"
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, profile_sort_at
		) VALUES (
			$1, 'did:plc:recipient', 'target', 'bafy-target', 'target', '{}', now(), now()
		)
	`, targetURI); err != nil {
		t.Fatalf("seed notification target: %v", err)
	}

	ev := tap.Event{
		URI:        "at://did:plc:author/social.craftsky.feed.post/imported",
		CID:        "bafy-imported",
		DID:        "did:plc:author",
		Rkey:       "imported",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type":"social.craftsky.feed.post",
			"text":"hello recipient",
			"createdAt":"2018-03-04T05:06:07Z",
			"externalImport":{"source":"instagram"},
			"reply":{
				"root":{"uri":"` + targetURI + `","cid":"bafy-target"},
				"parent":{"uri":"` + targetURI + `","cid":"bafy-target"}
			},
			"embed":{
				"$type":"social.craftsky.feed.post#quoteEmbed",
				"record":{"uri":"` + targetURI + `","cid":"bafy-target"}
			},
			"facets":[{
				"index":{"byteStart":6,"byteEnd":15},
				"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]
			}]
		}`),
	}
	postIndexer := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
	if err := postIndexer.Handle(context.Background(), ev); err != nil {
		t.Fatalf("index imported source: %v", err)
	}

	var mentionCount, sourceEventCount int
	var replyURI, quoteURI *string
	if err := pool.QueryRow(context.Background(), `
		SELECT reply_parent_uri, quote_uri
		FROM craftsky_posts
		WHERE uri = $1
	`, ev.URI).Scan(&replyURI, &quoteURI); err != nil {
		t.Fatalf("read imported relationships: %v", err)
	}
	if replyURI == nil || *replyURI != targetURI || quoteURI == nil || *quoteURI != targetURI {
		t.Fatalf("imported relationships reply=%v quote=%v", replyURI, quoteURI)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM craftsky_post_mentions WHERE post_uri = $1
	`, ev.URI).Scan(&mentionCount); err != nil {
		t.Fatalf("count imported mentions: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM notification_events WHERE source_uri = $1
	`, ev.URI).Scan(&sourceEventCount); err != nil {
		t.Fatalf("count imported notification events: %v", err)
	}
	if mentionCount != 1 || sourceEventCount != 0 {
		t.Fatalf("mention materialization=%d source notifications=%d, want 1/0", mentionCount, sourceEventCount)
	}

	likeIndexer := index.NewCraftskyLike(pool, testLogger(), notifications.NewService())
	like := interactionEvent(interactionIndexerCases()[0], "like-import", "bafy-like-import", string(ev.URI), string(ev.CID))
	if err := likeIndexer.Handle(context.Background(), like); err != nil {
		t.Fatalf("index later like: %v", err)
	}
	var recipient, category string
	if err := pool.QueryRow(context.Background(), `
		SELECT recipient_did, category
		FROM notification_events
		WHERE source_uri = $1
	`, like.URI).Scan(&recipient, &category); err != nil {
		t.Fatalf("read later interaction notification: %v", err)
	}
	if recipient != "did:plc:author" || category != "like" {
		t.Fatalf("later notification recipient/category = %q/%q, want author/like", recipient, category)
	}

	repostIndexer := index.NewCraftskyRepost(pool, testLogger(), notifications.NewService())
	repost := interactionEvent(interactionIndexerCases()[1], "repost-import", "bafy-repost-import", string(ev.URI), string(ev.CID))
	if err := repostIndexer.Handle(context.Background(), repost); err != nil {
		t.Fatalf("index later repost: %v", err)
	}

	laterPosts := []tap.Event{
		{
			URI:        "at://did:plc:actor/social.craftsky.feed.post/quote-import",
			CID:        "bafy-quote-import",
			DID:        "did:plc:actor",
			Rkey:       "quote-import",
			Collection: "social.craftsky.feed.post",
			Action:     "create",
			Record: json.RawMessage(`{
				"$type":"social.craftsky.feed.post",
				"text":"ordinary later quote",
				"createdAt":"2026-07-23T12:01:00Z",
				"embed":{
					"$type":"social.craftsky.feed.post#quoteEmbed",
					"record":{"uri":"` + string(ev.URI) + `","cid":"` + string(ev.CID) + `"}
				}
			}`),
		},
		{
			URI:        "at://did:plc:actor/social.craftsky.feed.post/reply-import",
			CID:        "bafy-reply-import",
			DID:        "did:plc:actor",
			Rkey:       "reply-import",
			Collection: "social.craftsky.feed.post",
			Action:     "create",
			Record: json.RawMessage(`{
				"$type":"social.craftsky.feed.post",
				"text":"ordinary later reply",
				"createdAt":"2026-07-23T12:02:00Z",
				"reply":{
					"root":{"uri":"` + string(ev.URI) + `","cid":"` + string(ev.CID) + `"},
					"parent":{"uri":"` + string(ev.URI) + `","cid":"` + string(ev.CID) + `"}
				}
			}`),
		},
	}
	for _, later := range laterPosts {
		if err := postIndexer.Handle(context.Background(), later); err != nil {
			t.Fatalf("index later ordinary post %s: %v", later.Rkey, err)
		}
	}

	rows, err := pool.Query(context.Background(), `
		SELECT source_uri, category
		FROM notification_events
		WHERE recipient_did = 'did:plc:author'
		  AND source_uri = ANY($1::text[])
	`, []string{string(like.URI), string(repost.URI), string(laterPosts[0].URI), string(laterPosts[1].URI)})
	if err != nil {
		t.Fatalf("read later notification set: %v", err)
	}
	defer rows.Close()
	gotCategories := make(map[string]string)
	for rows.Next() {
		var sourceURI, gotCategory string
		if err := rows.Scan(&sourceURI, &gotCategory); err != nil {
			t.Fatalf("scan later notification: %v", err)
		}
		gotCategories[sourceURI] = gotCategory
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate later notifications: %v", err)
	}
	wantCategories := map[string]string{
		string(like.URI):          "like",
		string(repost.URI):        "repost",
		string(laterPosts[0].URI): "quote",
		string(laterPosts[1].URI): "reply",
	}
	if !maps.Equal(gotCategories, wantCategories) {
		t.Fatalf("later notification categories = %v, want %v", gotCategories, wantCategories)
	}
}

func assertPostImportClassification(
	t *testing.T,
	pool *pgxpool.Pool,
	uri string,
	wantSource *string,
	wantProfileSortAt time.Time,
	wantOrdinarySort bool,
) {
	t.Helper()
	var source *string
	var profileSortAt, indexedAt time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT external_import_source, profile_sort_at, indexed_at
		FROM craftsky_posts
		WHERE uri = $1
	`, uri).Scan(&source, &profileSortAt, &indexedAt); err != nil {
		t.Fatalf("read import classification: %v", err)
	}
	if (source == nil) != (wantSource == nil) || source != nil && *source != *wantSource {
		t.Fatalf("external_import_source = %v, want %v", source, wantSource)
	}
	if wantOrdinarySort {
		if !profileSortAt.Equal(indexedAt) {
			t.Fatalf("ordinary profile_sort_at = %s, want indexed_at %s", profileSortAt, indexedAt)
		}
		return
	}
	if !profileSortAt.Equal(wantProfileSortAt) {
		t.Fatalf("profile_sort_at = %s, want %s", profileSortAt, wantProfileSortAt)
	}
}

func assertStoredPostRecord(
	t *testing.T,
	pool *pgxpool.Pool,
	uri string,
	want json.RawMessage,
) {
	t.Helper()
	var stored json.RawMessage
	if err := pool.QueryRow(context.Background(), `
		SELECT record
		FROM craftsky_posts
		WHERE uri = $1
	`, uri).Scan(&stored); err != nil {
		t.Fatalf("read stored record: %v", err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(stored, &gotValue); err != nil {
		t.Fatalf("decode stored record: %v", err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode expected record: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("stored record = %#v, want %#v", gotValue, wantValue)
	}
}
