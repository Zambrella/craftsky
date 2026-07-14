package index_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestLikeDeleteRetractsNotificationAndCancelsUnsentDelivery(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ('10000000-0000-0000-0000-000000000001', 'device-1', 'ios', 'token-1');
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		VALUES ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'did:plc:author', '30000000-0000-0000-0000-000000000001')
	`); err != nil {
		t.Fatal(err)
	}

	idx := index.NewCraftskyLike(pool, testLogger(), notifications.NewService())
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	ev.Action = "delete"
	ev.Record = nil
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var state, deliveryStatus string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if state != "retracted" || deliveryStatus != "cancelled" {
		t.Fatalf("state=%s delivery=%s, want retracted/cancelled", state, deliveryStatus)
	}
}

func TestLikeRecreationReactivatesStableNotificationWithoutAnotherPush(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ('10000000-0000-0000-0000-000000000001', 'device-1', 'ios', 'token-1');
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		VALUES ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'did:plc:author', '30000000-0000-0000-0000-000000000001')
	`); err != nil {
		t.Fatal(err)
	}

	idx := index.NewCraftskyLike(pool, testLogger(), notifications.NewService())
	first := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	var firstID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM notification_events`).Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	first.Action = "delete"
	first.Record = nil
	if err := idx.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}

	second := interactionEvent(interactionIndexerCases()[0], "r2", "bafy2", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	var id, state, sourceURI string
	if err := pool.QueryRow(context.Background(), `SELECT id::text, state, source_uri FROM notification_events`).Scan(&id, &state, &sourceURI); err != nil {
		t.Fatal(err)
	}
	var deliveryCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveryCount); err != nil {
		t.Fatal(err)
	}
	if id != firstID || state != "active" || sourceURI != string(second.URI) || deliveryCount != 1 {
		t.Fatalf("id=%s state=%s source=%s deliveries=%d", id, state, sourceURI, deliveryCount)
	}
}

func TestActiveInteractionSourceReplacementIgnoresStaleDelete(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
				t.Fatal(err)
			}
			seedCraftskyMember(t, pool, "did:plc:actor")
			seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
			lifecycle := notifications.NewService()
			var idx index.Indexer
			if tc.name == "like" {
				idx = index.NewCraftskyLike(pool, testLogger(), lifecycle)
			} else {
				idx = index.NewCraftskyRepost(pool, testLogger(), lifecycle)
			}
			first := interactionEvent(tc, "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			second := interactionEvent(tc, "r2", "bafy2", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), first); err != nil {
				t.Fatal(err)
			}
			var stableID string
			if err := pool.QueryRow(context.Background(), `SELECT id::text FROM notification_events`).Scan(&stableID); err != nil {
				t.Fatal(err)
			}
			if err := idx.Handle(context.Background(), second); err != nil {
				t.Fatal(err)
			}
			first.Action = "delete"
			first.Record = nil
			if err := idx.Handle(context.Background(), first); err != nil {
				t.Fatal(err)
			}
			var id, state, sourceURI string
			if err := pool.QueryRow(context.Background(), `SELECT id::text,state,source_uri FROM notification_events`).Scan(&id, &state, &sourceURI); err != nil {
				t.Fatal(err)
			}
			if id != stableID || state != "active" || sourceURI != second.URI.String() {
				t.Fatalf("id=%s state=%s source=%s", id, state, sourceURI)
			}
		})
	}
}

func TestActiveFollowSourceReplacementIgnoresStaleDelete(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:alice','a'),('did:plc:bob','b')`); err != nil {
		t.Fatal(err)
	}
	idx := index.NewBlueskyFollow(pool, notifications.NewService())
	first := followEvent("r1", "bafy1", "did:plc:alice", "did:plc:bob")
	second := followEvent("r2", "bafy2", "did:plc:alice", "did:plc:bob")
	if err := idx.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	var stableID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM notification_events`).Scan(&stableID); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	first.Action = "delete"
	first.Record = nil
	if err := idx.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	var id, state, sourceURI string
	if err := pool.QueryRow(context.Background(), `SELECT id::text,state,source_uri FROM notification_events`).Scan(&id, &state, &sourceURI); err != nil {
		t.Fatal(err)
	}
	if id != stableID || state != "active" || sourceURI != second.URI.String() {
		t.Fatalf("id=%s state=%s source=%s", id, state, sourceURI)
	}
}

func TestDeletingRequiredDestinationRetractsInteractionNotification(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	lifecycle := notifications.NewService()
	like := index.NewCraftskyLike(pool, testLogger(), lifecycle)
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := like.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	post := index.NewCraftskyPost(pool, testLogger(), lifecycle)
	if err := post.Handle(context.Background(), tap.Event{URI: "at://did:plc:author/social.craftsky.feed.post/post1", DID: "did:plc:author", Rkey: "post1", Collection: "social.craftsky.feed.post", Action: "delete"}); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "retracted" {
		t.Fatalf("state=%s", state)
	}
}

func TestEveryProducerDeletionRetractsAndCancelsUnsentStates(t *testing.T) {
	for _, producer := range []string{"like", "repost", "follow", "reply", "mention", "quote"} {
		for _, status := range []string{"pending", "retry", "leased"} {
			t.Run(producer+"_"+status, func(t *testing.T) {
				pool, idx, ev := producerWithDelivery(t, producer)
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
				_ = pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state)
				_ = pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&delivery)
				if state != "retracted" || delivery != "cancelled" {
					t.Fatalf("state=%s delivery=%s", state, delivery)
				}
			})
		}
	}
}

func producerWithDelivery(t *testing.T, producer string) (*pgxpool.Pool, index.Indexer, tap.Event) {
	t.Helper()
	service := notifications.NewService()
	switch producer {
	case "like", "repost":
		pool := testdb.WithSchema(t, craftskyInteractionsDDL)
		applyNotificationMigration(t, pool)
		seedCraftskyMember(t, pool, "did:plc:actor")
		seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
		seedNotificationSubscription(t, pool, "did:plc:author")
		var tc interactionIndexerCase
		for _, candidate := range interactionIndexerCases() {
			if candidate.name == producer {
				tc = candidate
			}
		}
		var idx index.Indexer
		if producer == "like" {
			idx = index.NewCraftskyLike(pool, testLogger(), service)
		} else {
			idx = index.NewCraftskyRepost(pool, testLogger(), service)
		}
		ev := interactionEvent(tc, "r1", "cid1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
		if err := idx.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
		return pool, idx, ev
	case "follow":
		pool := testdb.WithSchema(t, atprotoFollowsDDL)
		applyNotificationMigration(t, pool)
		_, _ = pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:alice','a'),('did:plc:bob','b')`)
		seedNotificationSubscription(t, pool, "did:plc:bob")
		idx := index.NewBlueskyFollow(pool, service)
		ev := followEvent("r1", "cid1", "did:plc:alice", "did:plc:bob")
		if err := idx.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
		return pool, idx, ev
	default:
		pool := testdb.WithSchema(t, craftskyPostsDDL)
		applyNotificationMigration(t, pool)
		seedCraftskyMember(t, pool, "did:plc:actor")
		seedCraftskyMember(t, pool, "did:plc:recipient")
		_, _ = pool.Exec(context.Background(), `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES('at://did:plc:recipient/social.craftsky.feed.post/root','did:plc:recipient','root','rootcid','root','{}',now())`)
		seedNotificationSubscription(t, pool, "did:plc:recipient")
		body := `{"text":"event","createdAt":"2026-05-04T12:00:00Z"}`
		switch producer {
		case "reply":
			body = `{"text":"reply","createdAt":"2026-05-04T12:00:00Z","reply":{"root":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"},"parent":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"}}}`
		case "mention":
			body = `{"text":"mention recipient","createdAt":"2026-05-04T12:00:00Z","facets":[{"index":{"byteStart":8,"byteEnd":17},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`
		case "quote":
			body = `{"text":"quote","createdAt":"2026-05-04T12:00:00Z","embed":{"$type":"social.craftsky.feed.post#quoteEmbed","record":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"}}}`
		}
		ev := tap.Event{URI: syntax.ATURI("at://did:plc:actor/social.craftsky.feed.post/" + producer), CID: "cid1", DID: "did:plc:actor", Rkey: syntax.RecordKey(producer), Collection: "social.craftsky.feed.post", Action: "create", Record: json.RawMessage(body)}
		idx := index.NewCraftskyPost(pool, testLogger(), service)
		if err := idx.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
		return pool, idx, ev
	}
}
