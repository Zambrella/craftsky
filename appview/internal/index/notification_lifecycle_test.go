package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestPostProducerDeletionRetractsAndCancelsUnsentStates(t *testing.T) {
	const target = "at://did:plc:recipient/social.craftsky.feed.post/root"
	for producer, body := range map[string]string{
		"reply":   `{"text":"reply","createdAt":"2026-05-04T12:00:00Z","reply":{"root":{"uri":"` + target + `","cid":"rootcid"},"parent":{"uri":"` + target + `","cid":"rootcid"}}}`,
		"mention": `{"text":"mention recipient","createdAt":"2026-05-04T12:00:00Z","facets":[{"index":{"byteStart":8,"byteEnd":17},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`,
		"quote":   `{"text":"quote","createdAt":"2026-05-04T12:00:00Z","embed":{"$type":"social.craftsky.feed.post#quoteEmbed","record":{"uri":"` + target + `","cid":"rootcid"}}}`,
	} {
		for _, status := range []string{"pending", "retry", "leased"} {
			t.Run(producer+"_"+status, func(t *testing.T) {
				pool := testdb.WithSchema(t, craftskyPostsDDL)
				applyNotificationMigration(t, pool)
				seedCraftskyMember(t, pool, "did:plc:actor")
				seedCraftskyMember(t, pool, "did:plc:recipient")
				if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES($1,'did:plc:recipient','root','rootcid','root','{}',now())`, target); err != nil {
					t.Fatal(err)
				}
				seedNotificationSubscription(t, pool, "did:plc:recipient")
				ev := tap.Event{
					URI: syntax.ATURI("at://did:plc:actor/social.craftsky.feed.post/" + producer),
					CID: "cid1", DID: "did:plc:actor", Rkey: syntax.RecordKey(producer),
					Collection: "social.craftsky.feed.post", Action: "create", Record: json.RawMessage(body),
				}
				idx := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
				if err := idx.Handle(context.Background(), ev); err != nil {
					t.Fatal(err)
				}
				if status == "leased" {
					_, _ = pool.Exec(context.Background(), `UPDATE push_deliveries SET status='leased',lease_owner='worker',lease_expires_at=now()+interval '1 minute'`)
				} else {
					_, _ = pool.Exec(context.Background(), `UPDATE push_deliveries SET status=$1`, status)
				}
				ev.Action, ev.Record = "delete", nil
				if err := idx.Handle(context.Background(), ev); err != nil {
					t.Fatal(err)
				}
				var state, delivery string
				if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
					t.Fatal(err)
				}
				if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&delivery); err != nil {
					t.Fatal(err)
				}
				if state != "retracted" || delivery != "cancelled" {
					t.Fatalf("state=%s delivery=%s", state, delivery)
				}
			})
		}
	}
}
