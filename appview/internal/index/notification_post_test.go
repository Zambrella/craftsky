package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestPostNotificationUsesReplyOverQuoteAndMentionForSameRecipient(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:recipient")
	seedCraftskyMember(t, pool, "did:plc:actor")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at)
		VALUES ('at://did:plc:recipient/social.craftsky.feed.post/original', 'did:plc:recipient', 'original', 'originalcid', 'original', '{}', now())
	`); err != nil {
		t.Fatal(err)
	}

	ev := tap.Event{
		URI: "at://did:plc:actor/social.craftsky.feed.post/response", CID: "responsecid",
		DID: "did:plc:actor", Rkey: "response", Collection: "social.craftsky.feed.post", Action: "create",
		Record: json.RawMessage(`{
			"text":"hello recipient",
			"createdAt":"2026-05-04T12:00:00Z",
			"reply":{"root":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/original","cid":"originalcid"},"parent":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/original","cid":"originalcid"}},
			"embed":{"$type":"social.craftsky.feed.post#quoteEmbed","record":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/original","cid":"originalcid"}},
			"facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]
		}`),
	}
	idx := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	var count int
	var category string
	if err := pool.QueryRow(context.Background(), `SELECT count(*), max(category) FROM notification_events`).Scan(&count, &category); err != nil {
		t.Fatal(err)
	}
	if count != 1 || category != "reply" {
		t.Fatalf("count=%d category=%s, want one reply", count, category)
	}
	ev.Action = "delete"
	ev.Record = nil
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "retracted" {
		t.Fatalf("state=%s, want retracted", state)
	}
}

func TestPostMentionOfNonMemberIndexesPostWithoutNotification(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:actor")
	ev := tap.Event{
		URI: "at://did:plc:actor/social.craftsky.feed.post/mention", CID: "mentioncid",
		DID: "did:plc:actor", Rkey: "mention", Collection: "social.craftsky.feed.post", Action: "create",
		Record: json.RawMessage(`{
			"text":"hello outsider",
			"createdAt":"2026-05-04T12:00:00Z",
			"facets":[{"index":{"byteStart":6,"byteEnd":14},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:outsider"}]}]
		}`),
	}
	idx := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	var posts, events, deliveries int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_posts`).Scan(&posts)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events`).Scan(&events)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveries)
	if posts != 1 || events != 0 || deliveries != 0 {
		t.Fatalf("posts=%d events=%d deliveries=%d", posts, events, deliveries)
	}
}

func TestPostNotificationDeliveryWindow(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskyMember(t, pool, "did:plc:recipient")
	seedNotificationSubscription(t, pool, "did:plc:recipient")
	ev := tap.Event{
		URI: "at://did:plc:actor/social.craftsky.feed.post/mention", CID: "mentioncid",
		DID: "did:plc:actor", Rkey: "mention", Collection: "social.craftsky.feed.post", Action: "create",
		Record: json.RawMessage(`{"text":"hello recipient","createdAt":"2099-05-04T12:00:00Z"` + mentionExtra("did:plc:recipient") + `}`),
	}
	idx := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	var bounded bool
	if err := pool.QueryRow(context.Background(), `
		SELECT deadline_at = next_attempt_at + interval '6 hours' AND status = 'pending'
		FROM push_deliveries
	`).Scan(&bounded); err != nil || !bounded {
		t.Fatalf("future-dated delivery window bounded=%t err=%v", bounded, err)
	}

	// Backfill still creates Activity history, but an expired delivery must
	// start terminal so the dispatcher never has to claim it.
	old := ev
	old.URI = "at://did:plc:actor/social.craftsky.feed.post/historical"
	old.Rkey = "historical"
	old.Record = json.RawMessage(`{"text":"hello recipient","createdAt":"2000-01-01T00:00:00Z"` + mentionExtra("did:plc:recipient") + `}`)
	if err := idx.Handle(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	var expired bool
	if err := pool.QueryRow(context.Background(), `
		SELECT delivery.status = 'expired' AND notice.state = 'active'
		FROM push_deliveries delivery JOIN notification_events notice ON notice.id = delivery.notification_id
		WHERE notice.source_uri = $1
	`, old.URI).Scan(&expired); err != nil || !expired {
		t.Fatalf("historical delivery starts expired=%t err=%v", expired, err)
	}
}
