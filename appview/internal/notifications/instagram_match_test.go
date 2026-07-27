package notifications

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestServiceCreatesOneActorfulInstagramMatchPerSuccessfulOperation(t *testing.T) {
	pool := instagramNotificationPool(t)
	ctx := context.Background()
	recipient := syntax.DID("did:plc:instagram-notification-recipient")
	firstActor := syntax.DID("did:plc:instagram-notification-first")
	secondActor := syntax.DID("did:plc:instagram-notification-second")
	firstOperation := uuid.MustParse("00000000-0000-0000-0000-000000000801")
	secondOperation := uuid.MustParse("00000000-0000-0000-0000-000000000802")
	seedInstagramNotificationSubscription(t, pool, recipient)
	base := time.Date(2026, 7, 27, 14, 0, 0, 0, time.UTC)
	service := NewService()
	service.now = func() time.Time { return base }

	for _, activation := range []InstagramMatchActivation{
		{
			RecipientDID: recipient,
			ActorDID:     firstActor,
			OperationID:  firstOperation,
			ActivityAt:   base,
		},
		{
			RecipientDID: recipient,
			ActorDID:     secondActor,
			OperationID:  secondOperation,
			ActivityAt:   base.Add(time.Minute),
		},
		{
			RecipientDID: recipient,
			ActorDID:     firstActor,
			OperationID:  firstOperation,
			ActivityAt:   base.Add(2 * time.Minute),
		},
	} {
		activateInstagramMatch(t, pool, service, activation)
	}

	rows, err := pool.Query(ctx, `
		SELECT actor_did, subject_key, source_uri, source_cid, source_rkey,
		       recipient_followed_actor
		FROM notification_events
		WHERE recipient_did=$1 AND category='instagramMatch'
		ORDER BY activity_at, id
	`, recipient)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type event struct {
		actor        string
		subjectKey   string
		sourceURI    *string
		sourceCID    *string
		sourceRkey   *string
		followsActor bool
	}
	events := make([]event, 0, 2)
	for rows.Next() {
		var item event
		if err := rows.Scan(
			&item.actor,
			&item.subjectKey,
			&item.sourceURI,
			&item.sourceCID,
			&item.sourceRkey,
			&item.followsActor,
		); err != nil {
			t.Fatal(err)
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events=%+v, want two per-operation rows", events)
	}
	if events[0].actor != firstActor.String() ||
		events[0].subjectKey != firstOperation.String() ||
		events[1].actor != secondActor.String() ||
		events[1].subjectKey != secondOperation.String() {
		t.Fatalf("actorful events=%+v", events)
	}
	for _, item := range events {
		if item.sourceURI != nil || item.sourceCID != nil || item.sourceRkey != nil || !item.followsActor {
			t.Fatalf("invalid source-less event=%+v", item)
		}
	}
	var deliveries int
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT count(*), min(next_attempt_at)
		FROM push_deliveries
	`).Scan(&deliveries, &nextAttempt); err != nil {
		t.Fatal(err)
	}
	if deliveries != 2 || !nextAttempt.Equal(base) {
		t.Fatalf("deliveries=%d nextAttempt=%s", deliveries, nextAttempt)
	}
}

func TestSocialActivationRemainsIdempotentAfterActorfulInstagramMigration(t *testing.T) {
	pool := instagramNotificationPool(t)
	service := NewService()
	activity := time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC)
	activation := Activation{
		RecipientDID: syntax.DID("did:plc:social-recipient"),
		ActorDID:     syntax.DID("did:plc:social-actor"),
		Category:     Like,
		SubjectKey:   "at://did:plc:social-recipient/social.craftsky.feed.post/post",
		SourceURI:    syntax.ATURI("at://did:plc:social-actor/social.craftsky.feed.like/like"),
		SourceCID:    syntax.CID("synthetic-social-cid"),
		SourceRkey:   syntax.RecordKey("like"),
		SubjectURI:   syntax.ATURI("at://did:plc:social-recipient/social.craftsky.feed.post/post"),
		SubjectCID:   syntax.CID("synthetic-post-cid"),
		ActivityAt:   activity,
	}
	for range 2 {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Activate(context.Background(), tx, activation); err != nil {
			tx.Rollback(context.Background())
			t.Fatal(err)
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var rows int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM notification_events WHERE category='like'
	`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("social rows=%d", rows)
	}
}

func activateInstagramMatch(
	t *testing.T,
	pool *pgxpool.Pool,
	service *Service,
	activation InstagramMatchActivation,
) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if err := service.ActivateInstagramMatch(context.Background(), tx, activation); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func instagramNotificationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE actor_mutes(
			owner_did TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			PRIMARY KEY(owner_did, subject_did)
		);
		CREATE TABLE atproto_blocks(
			uri TEXT PRIMARY KEY,
			blocker_did TEXT NOT NULL,
			subject_did TEXT NOT NULL
		);
		CREATE TABLE atproto_follows(
			uri TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			subject_did TEXT NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func seedInstagramNotificationSubscription(
	t *testing.T,
	pool *pgxpool.Pool,
	recipient syntax.DID,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES (
			'00000000-0000-0000-0000-000000000891',
			'synthetic-instagram-device','ios','synthetic-instagram-token'
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_account_subscriptions (
			id, installation_id, account_did, routing_id
		) VALUES (
			'00000000-0000-0000-0000-000000000892',
			'00000000-0000-0000-0000-000000000891',
			$1,
			'00000000-0000-0000-0000-000000000893'
		)
	`, recipient); err != nil {
		t.Fatal(err)
	}
}
