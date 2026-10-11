package index_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const notificationTarget = "at://did:plc:recipient/social.craftsky.feed.post/3aaaaaaaaaaa2"

type notificationPipeline struct {
	t          *testing.T
	pool       *pgxpool.Pool
	store      *ingestion.Store
	dispatcher *index.TransactionalDispatcher
}

func newNotificationPipeline(t *testing.T) *notificationPipeline {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	f := &notificationPipeline{t: t, pool: pool, store: store, dispatcher: index.NewTransactionalDispatcher()}
	for _, did := range []string{"did:plc:actor", "did:plc:recipient"} {
		seedCraftskyMember(t, pool, did)
		f.exec(`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',now(),now(),now())`, did)
	}
	post := index.NewCraftskyPost(pool, testLogger(), notifications.NewService())
	f.dispatcher.Register("social.craftsky.feed.post", index.NewImageSafetyCraftskyPost(post, imagesafety.ScanKey{ScannerID: "test", PolicyVersion: "1", CorpusVersion: "1"}))
	return f
}

func (f *notificationPipeline) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *notificationPipeline) ingest(ev tap.Event) {
	f.t.Helper()
	if _, err := f.store.IngestRecord(context.Background(), ev); err != nil {
		f.t.Fatal(err)
	}
	f.exec(`UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1`, ev.URI)
}

func (f *notificationPipeline) drain() {
	f.t.Helper()
	for range 20 {
		claims, err := f.store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{Worker: "notifications-test", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1})
		if err != nil {
			f.t.Fatal(err)
		}
		if len(claims) == 0 {
			return
		}
		if err := f.store.Project(context.Background(), claims[0], f.dispatcher.Project); err != nil {
			f.t.Fatal(err)
		}
	}
	f.t.Fatal("projection jobs did not settle")
}

func notificationEvent(id uint64, actor, key, extra string) tap.Event {
	return tap.Event{ID: id, URI: syntax.ATURI("at://" + actor + "/social.craftsky.feed.post/" + key), DID: syntax.DID(actor), Rkey: syntax.RecordKey(key), Collection: "social.craftsky.feed.post", Rev: "3aaaaaaaaaaa2", CID: syntax.CID(fmt.Sprintf("cid-%d", id)), Action: "create", Record: json.RawMessage(`{"text":"hello recipient","sponsored":false,"createdAt":"2026-05-04T12:00:00Z"` + extra + `}`)}
}

func replyExtra(target string) string {
	return `,"reply":{"root":{"uri":"` + target + `","cid":"originalcid"},"parent":{"uri":"` + target + `","cid":"originalcid"}}`
}
func quoteExtra(target string) string {
	return `,"embed":{"$type":"social.craftsky.feed.post#quoteEmbed","record":{"uri":"` + target + `","cid":"originalcid"}}`
}
func mentionExtra(did string) string {
	return `,"facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"` + did + `"}]}]`
}

func (f *notificationPipeline) assertCount(sql string, want int) {
	f.t.Helper()
	var got int
	if err := f.pool.QueryRow(context.Background(), sql).Scan(&got); err != nil {
		f.t.Fatal(err)
	}
	if got != want {
		f.t.Fatalf("%s: got %d want %d", sql, got, want)
	}
}

func TestPostNotificationsConvergeAcrossBackfillOrder(t *testing.T) {
	for _, category := range []string{"reply", "quote", "replyOverQuoteAndMention"} {
		for _, targetFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/targetFirst=%t", category, targetFirst), func(t *testing.T) {
				f := newNotificationPipeline(t)
				target := notificationEvent(1, "did:plc:recipient", "3aaaaaaaaaaa2", "")
				extra := replyExtra(notificationTarget)
				if category == "quote" {
					extra = quoteExtra(notificationTarget)
				}
				if category == "replyOverQuoteAndMention" {
					extra += quoteExtra(notificationTarget) + mentionExtra("did:plc:recipient")
				}
				response := notificationEvent(2, "did:plc:actor", "3aaaaaaaaaaa3", extra)
				if targetFirst {
					f.ingest(target)
					f.drain()
				}
				f.ingest(response)
				f.drain()
				f.assertCount(`SELECT count(*) FROM craftsky_posts WHERE rkey='3aaaaaaaaaaa3'`, 1)
				if !targetFirst {
					f.assertCount(`SELECT count(*) FROM notification_events`, 0)
					f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE projection_kind='social_craftsky_feed_post' AND state='complete'`, 1)
					f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE projection_kind='post_notifications' AND state='blocked' AND dependency_kind='subject_uri'`, 1)
					f.ingest(target)
					f.drain()
				}
				want := category
				if category == "replyOverQuoteAndMention" {
					want = "reply"
				}
				f.assertCount(`SELECT count(*) FROM notification_events WHERE category='`+want+`'`, 1)
				f.assertCount(`SELECT count(*) FROM notification_events WHERE activity_at='2026-05-04T12:00:00Z'`, 1)
				// Explicitly replay notification work and duplicate source delivery.
				f.exec(`UPDATE tap_projection_jobs SET state='pending',completed_at=NULL WHERE projection_kind='post_notifications'`)
				f.drain()
				f.ingest(response)
				f.drain()
				f.assertCount(`SELECT count(*) FROM notification_events`, 1)
			})
		}
	}
}

func TestPostNotificationWaitsForRecipientMembership(t *testing.T) {
	f := newNotificationPipeline(t)
	f.exec(`DELETE FROM craftsky_profiles WHERE did='did:plc:recipient'`)
	f.ingest(notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa5", mentionExtra("did:plc:recipient")))
	f.drain()
	f.assertCount(`SELECT count(*) FROM craftsky_posts`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE projection_kind='post_notifications' AND state='blocked' AND dependency_kind='member_did'`, 1)
	seedCraftskyMember(t, f.pool, "did:plc:recipient")
	if err := f.store.WakeDependency(context.Background(), tap.Dependency{Kind: "member_did", Key: "did:plc:recipient"}); err != nil {
		t.Fatal(err)
	}
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='mention'`, 1)
}

func TestPostNotificationDeleteWhileWaitingCannotNotify(t *testing.T) {
	f := newNotificationPipeline(t)
	ev := notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget))
	f.ingest(ev)
	f.drain()
	ev.ID = 2
	ev.Rev = "3aaaaaaaaaaa3"
	ev.Action = "delete"
	ev.Record = nil
	ev.CID = ""
	f.ingest(ev)
	f.drain()
	f.ingest(notificationEvent(3, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
	f.drain()
	f.assertCount(`SELECT count(*) FROM craftsky_posts WHERE rkey='3aaaaaaaaaaa3'`, 0)
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE projection_kind='post_notifications' AND state='blocked'`, 0)
}

func TestPostNotificationUpdateReplacesWaitingDependency(t *testing.T) {
	f := newNotificationPipeline(t)
	ev := notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget))
	f.ingest(ev)
	f.drain()
	updated := notificationEvent(2, "did:plc:actor", "3aaaaaaaaaaa3", quoteExtra("at://did:plc:recipient/social.craftsky.feed.post/3aaaaaaaaaaa4"))
	updated.Action = "update"
	updated.Rev = "3aaaaaaaaaaa3"
	f.ingest(updated)
	f.drain()
	f.ingest(notificationEvent(3, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	f.ingest(notificationEvent(4, "did:plc:recipient", "3aaaaaaaaaaa4", ""))
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='quote' AND source_cid='cid-2'`, 1)
}

func TestPostNotificationRetryRespectsPreferences(t *testing.T) {
	f := newNotificationPipeline(t)
	f.exec(`INSERT INTO notification_preferences(account_did,category,scope,push_enabled) VALUES('did:plc:recipient','reply','peopleIFollow',false)`)
	f.ingest(notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget)))
	f.drain()
	f.ingest(notificationEvent(2, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE state='blocked'`, 0)
}

func TestPostNotificationTimerRecoversMissedDependencyWake(t *testing.T) {
	f := newNotificationPipeline(t)
	f.ingest(notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget)))
	f.drain()
	// Model the dependency becoming visible during the blocked-job commit,
	// when the dependency's wake query cannot yet see the waiting job.
	if err := index.NewCraftskyPost(f.pool, testLogger()).Handle(context.Background(), notificationEvent(2, "did:plc:recipient", "3aaaaaaaaaaa2", "")); err != nil {
		t.Fatal(err)
	}
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	f.exec(`UPDATE tap_projection_jobs SET next_attempt_at=now()-interval '1 second' WHERE projection_kind='post_notifications'`)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='reply'`, 1)
}

func TestPostNotificationUpdateInvalidatesClaimedLease(t *testing.T) {
	f := newNotificationPipeline(t)
	ev := notificationEvent(1, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget))
	f.ingest(ev)
	claim := func() ingestion.ProjectionClaim {
		t.Helper()
		claims, err := f.store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{Worker: "stale-test", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1})
		if err != nil || len(claims) != 1 {
			t.Fatalf("claims=%v err=%v", claims, err)
		}
		return claims[0]
	}
	if err := f.store.Project(context.Background(), claim(), f.dispatcher.Project); err != nil {
		t.Fatal(err)
	}
	stale := claim()
	if stale.ProjectionKind != ingestion.PostNotificationsProjection {
		t.Fatalf("unexpected claim: %+v", stale)
	}
	ev.ID = 2
	ev.Rev = "3aaaaaaaaaaa3"
	ev.Action = "delete"
	ev.Record = nil
	ev.CID = ""
	f.ingest(ev)
	if err := f.store.Project(context.Background(), stale, f.dispatcher.Project); !errors.Is(err, ingestion.ErrProjectionLeaseLost) {
		t.Fatalf("stale projection err=%v", err)
	}
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
}

func TestPostNotificationAvailableRecipientDoesNotWaitForOtherMember(t *testing.T) {
	f := newNotificationPipeline(t)
	f.ingest(notificationEvent(1, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
	f.drain()
	f.ingest(notificationEvent(2, "did:plc:actor", "3aaaaaaaaaaa3", replyExtra(notificationTarget)+mentionExtra("did:plc:outsider")))
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='reply'`, 1)
	f.assertCount(`SELECT count(*) FROM tap_projection_jobs WHERE state='blocked' AND dependency_key='did:plc:outsider'`, 1)
	seedCraftskyMember(t, f.pool, "did:plc:outsider")
	f.exec(`UPDATE tap_projection_jobs SET next_attempt_at=now()-interval '1 second' WHERE projection_kind='post_notifications' AND state='blocked'`)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='mention'`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='reply'`, 1)
}
