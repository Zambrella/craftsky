package index_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

type failingNotificationLifecycle struct{}

type failingRetractionLifecycle struct{ service *notifications.Service }

func (f failingRetractionLifecycle) Activate(ctx context.Context, tx pgx.Tx, activation notifications.Activation) error {
	return f.service.Activate(ctx, tx, activation)
}

func (f failingRetractionLifecycle) Retract(ctx context.Context, tx pgx.Tx, retraction notifications.Retraction) error {
	if err := f.service.Retract(ctx, tx, retraction); err != nil {
		return err
	}
	return errors.New("forced retraction failure")
}

func (failingNotificationLifecycle) Activate(ctx context.Context, tx pgx.Tx, activation notifications.Activation) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			source_uri, source_cid, source_rkey,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
			state, first_activity_at, activity_at, initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000001', $1, $2, $3, $4,
			$5, $6, $7, 'everyone', false, true,
			'active', $8, $8, $8
		)
	`, activation.RecipientDID, activation.ActorDID, activation.Category, activation.SubjectKey,
		activation.SourceURI, activation.SourceCID, activation.SourceRkey, activation.ActivityAt)
	if err != nil {
		return err
	}
	return errors.New("forced lifecycle failure")
}

func TestFollowCreationUsesSameNotificationTransactionAndEventTimeScope(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
		('did:plc:alice', 'alice-profile'),
		('did:plc:bob', 'bob-profile');
		INSERT INTO notification_preferences (account_did, category, scope, push_enabled)
		VALUES ('did:plc:bob', 'follow', 'peopleIFollow', true);
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:bob/app.bsky.graph.follow/mutual', 'did:plc:bob', 'mutual', 'c', 'did:plc:alice', '{}', now())
	`); err != nil {
		t.Fatal(err)
	}

	idx := index.NewBlueskyFollow(pool, notifications.NewService())
	ev := followEvent("follow1", "bafyfollow1", "did:plc:alice", "did:plc:bob")
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	var recipient, actor, category string
	if err := pool.QueryRow(context.Background(), `SELECT recipient_did, actor_did, category FROM notification_events`).Scan(&recipient, &actor, &category); err != nil {
		t.Fatal(err)
	}
	if recipient != "did:plc:bob" || actor != "did:plc:alice" || category != "follow" {
		t.Fatalf("recipient=%s actor=%s category=%s", recipient, actor, category)
	}
}

func TestFollowToNonMemberIndexesSourceWithoutNotification(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:alice','profile')`); err != nil {
		t.Fatal(err)
	}
	idx := index.NewBlueskyFollow(pool, notifications.NewService())
	if err := idx.Handle(context.Background(), followEvent("follow1", "bafyfollow1", "did:plc:alice", "did:plc:outsider")); err != nil {
		t.Fatal(err)
	}
	var follows, events, deliveries int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM atproto_follows`).Scan(&follows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if follows != 1 || events != 0 || deliveries != 0 {
		t.Fatalf("follows=%d events=%d deliveries=%d", follows, events, deliveries)
	}
}

func followEvent(rkey, cid, actor, subject string) tap.Event {
	return tap.Event{
		URI:        syntax.ATURI("at://" + actor + "/app.bsky.graph.follow/" + rkey),
		CID:        syntax.CID(cid),
		DID:        syntax.DID(actor),
		Rkey:       syntax.RecordKey(rkey),
		Collection: "app.bsky.graph.follow",
		Action:     "create",
		Record:     json.RawMessage(fmt.Sprintf(`{"subject":%q,"createdAt":"2026-05-25T12:00:00Z"}`, subject)),
	}
}

func (failingNotificationLifecycle) Retract(context.Context, pgx.Tx, notifications.Retraction) error {
	return nil
}

func TestLikeCreateRollsBackSourceAndNotificationTogether(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatalf("read notification migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply notification migration: %v", err)
	}
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")

	idx := index.NewCraftskyLike(pool, testLogger(), failingNotificationLifecycle{})
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Fatal("Handle create succeeded, want forced lifecycle failure")
	}

	for _, table := range []string{"craftsky_likes", "notification_events"} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s count = %d after rollback, want 0", table, count)
		}
	}
}

func TestInteractionDeletionRollsBackSourceNotificationAndDeliveryTogether(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyInteractionsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
	seedNotificationSubscription(t, pool, "did:plc:author")
	service := notifications.NewService()
	ev := interactionEvent(interactionIndexerCases()[0], "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
	if err := index.NewCraftskyLike(pool, testLogger(), service).Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	ev.Action, ev.Record = "delete", nil
	if err := index.NewCraftskyLike(pool, testLogger(), failingRetractionLifecycle{service}).Handle(context.Background(), ev); err == nil {
		t.Fatal("delete succeeded, want forced rollback")
	}
	assertDeletionRollbackState(t, pool, `SELECT count(*) FROM craftsky_likes WHERE deleted_at IS NULL`)
}

func TestFollowDeletionRollsBackSourceNotificationAndDeliveryTogether(t *testing.T) {
	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	applyNotificationMigration(t, pool)
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:alice','a'),('did:plc:bob','b')`); err != nil {
		t.Fatal(err)
	}
	seedNotificationSubscription(t, pool, "did:plc:bob")
	service := notifications.NewService()
	ev := followEvent("r1", "bafy1", "did:plc:alice", "did:plc:bob")
	if err := index.NewBlueskyFollow(pool, service).Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	ev.Action, ev.Record = "delete", nil
	if err := index.NewBlueskyFollow(pool, failingRetractionLifecycle{service}).Handle(context.Background(), ev); err == nil {
		t.Fatal("delete succeeded, want forced rollback")
	}
	assertDeletionRollbackState(t, pool, `SELECT count(*) FROM atproto_follows`)
}

func TestPostDeletionRollsBackSourceNotificationAndDeliveryTogether(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	applyNotificationMigration(t, pool)
	seedCraftskyMember(t, pool, "did:plc:actor")
	seedCraftskyMember(t, pool, "did:plc:recipient")
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES('at://did:plc:recipient/social.craftsky.feed.post/root','did:plc:recipient','root','rootcid','root','{}',now())`); err != nil {
		t.Fatal(err)
	}
	seedNotificationSubscription(t, pool, "did:plc:recipient")
	service := notifications.NewService()
	ev := tap.Event{URI: "at://did:plc:actor/social.craftsky.feed.post/reply", CID: "replycid", DID: "did:plc:actor", Rkey: "reply", Collection: "social.craftsky.feed.post", Action: "create", Record: json.RawMessage(`{"text":"reply","createdAt":"2026-05-04T12:00:00Z","reply":{"root":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"},"parent":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/root","cid":"rootcid"}}}`)}
	if err := index.NewCraftskyPost(pool, testLogger(), service).Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	ev.Action, ev.Record = "delete", nil
	if err := index.NewCraftskyPost(pool, testLogger(), failingRetractionLifecycle{service}).Handle(context.Background(), ev); err == nil {
		t.Fatal("delete succeeded, want forced rollback")
	}
	assertDeletionRollbackState(t, pool, `SELECT count(*) FROM craftsky_posts WHERE uri='at://did:plc:actor/social.craftsky.feed.post/reply'`)
}

func applyNotificationMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
}

func seedNotificationSubscription(t *testing.T, pool *pgxpool.Pool, accountDID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES('10000000-0000-0000-0000-000000000001','device','ios','token')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001',$1,'30000000-0000-0000-0000-000000000001')`, accountDID); err != nil {
		t.Fatal(err)
	}
}

func assertDeletionRollbackState(t *testing.T, pool *pgxpool.Pool, sourceCountSQL string) {
	t.Helper()
	var sourceCount int
	if err := pool.QueryRow(context.Background(), sourceCountSQL).Scan(&sourceCount); err != nil {
		t.Fatal(err)
	}
	var state, deliveryStatus string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM notification_events`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if sourceCount != 1 || state != "active" || deliveryStatus != "pending" {
		t.Fatalf("source=%d state=%s delivery=%s", sourceCount, state, deliveryStatus)
	}
}
