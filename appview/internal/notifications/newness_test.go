package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationNewnessRevisionTracksGenuineActivations(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did, subject_did));
		CREATE TABLE pds_set_aggregates(kind TEXT NOT NULL, actor_did TEXT NOT NULL, subject_did TEXT);
	`)
	ctx := context.Background()
	for _, path := range []string{
		"000021_appview_notifications.up.sql",
		"000022_notification_newness.up.sql",
	} {
		migration, err := testdb.ReadMigration(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}

	service := notifications.NewService()
	activation := notifications.Activation{
		RecipientDID: syntax.DID("did:plc:recipient"),
		ActorDID:     syntax.DID("did:plc:actor"),
		Category:     notifications.Like,
		SubjectKey:   "at://did:plc:recipient/social.craftsky.feed.post/post1",
		SourceURI:    syntax.ATURI("at://did:plc:actor/social.craftsky.feed.like/like1"),
		SourceCID:    syntax.CID("bafy-like-1"),
		SourceRkey:   syntax.RecordKey("like1"),
		SubjectURI:   syntax.ATURI("at://did:plc:recipient/social.craftsky.feed.post/post1"),
		SubjectCID:   syntax.CID("bafy-post-1"),
		ActivityAt:   time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC),
	}

	activate := func(value notifications.Activation) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if err := service.Activate(ctx, tx, value); err != nil {
			t.Fatalf("activate: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit activation: %v", err)
		}
	}
	revision := func() int64 {
		t.Helper()
		var value int64
		if err := pool.QueryRow(ctx, `SELECT newness_revision FROM notification_events`).Scan(&value); err != nil {
			t.Fatalf("read revision: %v", err)
		}
		return value
	}

	activate(activation)
	first := revision()
	activate(activation)
	if replayed := revision(); replayed != first {
		t.Fatalf("exact replay revision = %d, want %d", replayed, first)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Retract(ctx, tx, notifications.Retraction{
		RecipientDID: activation.RecipientDID,
		ActorDID:     activation.ActorDID,
		Category:     activation.Category,
		SubjectKey:   activation.SubjectKey,
		Reason:       "set_deactivated",
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if retracted := revision(); retracted != first {
		t.Fatalf("retraction revision = %d, want %d", retracted, first)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT state FROM notification_events`).Scan(&state); err != nil || state != "retracted" {
		t.Fatalf("state after logical retraction = %q, err=%v", state, err)
	}

	activation.SourceURI = syntax.ATURI("at://did:plc:actor/social.craftsky.feed.like/like2")
	activation.SourceCID = syntax.CID("bafy-like-2")
	activation.SourceRkey = syntax.RecordKey("like2")
	activation.ActivityAt = activation.ActivityAt.Add(time.Minute)
	activate(activation)
	if reactivated := revision(); reactivated <= first {
		t.Fatalf("reactivation revision = %d, want greater than %d", reactivated, first)
	}
}

func TestNotificationSuppressionEmitsBoundedRelationshipOutcome(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did, subject_did));
		CREATE TABLE pds_set_aggregates(kind TEXT NOT NULL, actor_did TEXT NOT NULL, subject_did TEXT);
	`)
	ctx := context.Background()
	migration, err := testdb.ReadMigration("000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes(owner_did,subject_did) VALUES('did:plc:recipient','did:plc:private-target')`); err != nil {
		t.Fatal(err)
	}
	observer := &notificationRelationshipObserver{}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := notifications.NewService(observer).Activate(ctx, tx, notifications.Activation{
		RecipientDID: "did:plc:recipient",
		ActorDID:     "did:plc:private-target",
		Category:     notifications.Like,
		SubjectKey:   "private-subject",
		SourceURI:    "at://did:plc:private-target/social.craftsky.feed.like/private-rkey",
		ActivityAt:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if observer.operation != "notification_suppression" || observer.stage != "policy" || observer.result != "suppressed" || observer.errorClass != "none" {
		t.Fatalf("relationship outcome = %+v", observer)
	}
}

func TestNotificationPeopleIFollowReadsLogicalFollowAggregate(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did, subject_did));
		CREATE TABLE pds_set_aggregates(kind TEXT NOT NULL, actor_did TEXT NOT NULL, subject_did TEXT);
	`)
	ctx := context.Background()
	migration, err := testdb.ReadMigration("000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES('did:plc:recipient','like','peopleIFollow',true);
		INSERT INTO pds_set_aggregates(kind,actor_did,subject_did)
		VALUES('follow','did:plc:recipient','did:plc:actor');
	`); err != nil {
		t.Fatal(err)
	}

	activate := func(subjectKey string) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if err := notifications.NewService().Activate(ctx, tx, notifications.Activation{
			RecipientDID: "did:plc:recipient",
			ActorDID:     "did:plc:actor",
			Category:     notifications.Like,
			SubjectKey:   subjectKey,
			SourceURI:    syntax.ATURI("at://did:plc:actor/social.craftsky.feed.like/" + subjectKey),
			SourceCID:    syntax.CID("bafy-" + subjectKey),
			SourceRkey:   syntax.RecordKey(subjectKey),
			ActivityAt:   time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	activate("followed")
	if _, err := pool.Exec(ctx, `DELETE FROM pds_set_aggregates WHERE kind='follow'`); err != nil {
		t.Fatal(err)
	}
	activate("unfollowed")

	var count int
	var followed bool
	if err := pool.QueryRow(ctx, `SELECT count(*), bool_and(recipient_followed_actor) FROM notification_events`).Scan(&count, &followed); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !followed {
		t.Fatalf("events=%d recipientFollowedActor=%t, want one followed event", count, followed)
	}
}

type notificationRelationshipObserver struct {
	operation, stage, result, errorClass string
}

func (o *notificationRelationshipObserver) ObserveNotificationDecision(string, string) {}

func (o *notificationRelationshipObserver) ObserveRelationshipOutcome(operation, stage, result, errorClass string, _ time.Duration) {
	o.operation = operation
	o.stage = stage
	o.result = result
	o.errorClass = errorClass
}
