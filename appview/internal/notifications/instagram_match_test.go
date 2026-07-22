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

func TestServiceCoalescesInstagramMatchesIntoFixedWindowAndOneOutboxDelivery(t *testing.T) {
	pool := instagramNotificationPool(t)
	ctx := context.Background()
	recipient := syntax.DID("did:plc:instagram-notification-recipient")
	firstSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000801")
	secondSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000802")
	thirdSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000803")
	olderSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000804")
	seedInstagramNotificationSuggestion(t, pool, firstSuggestion)
	seedInstagramNotificationSuggestion(t, pool, secondSuggestion)
	seedInstagramNotificationSuggestion(t, pool, thirdSuggestion)
	seedInstagramNotificationSuggestion(t, pool, olderSuggestion)
	seedInstagramNotificationSubscription(t, pool, recipient)

	base := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	service := NewService()
	service.now = func() time.Time { return base }
	activateInstagramMatch(t, pool, service, InstagramMatchActivation{
		RecipientDID: recipient,
		SuggestionID: firstSuggestion,
		ActivityAt:   base,
	})

	var eventID uuid.UUID
	var count int
	var capped bool
	var createdAt, indexedAt, coalesceUntil time.Time
	var revision int64
	if err := pool.QueryRow(ctx, `
		SELECT id, system_count, system_count_capped, first_activity_at,
		       indexed_at, coalesce_until, newness_revision
		FROM notification_events
		WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
	`, recipient).Scan(&eventID, &count, &capped, &createdAt, &indexedAt, &coalesceUntil, &revision); err != nil {
		t.Fatal(err)
	}
	if count != 1 || capped || !createdAt.Equal(base) || !indexedAt.Equal(base) || !coalesceUntil.Equal(base.Add(5*time.Minute)) {
		t.Fatalf("first event count=%d capped=%t created=%s indexed=%s closes=%s", count, capped, createdAt, indexedAt, coalesceUntil)
	}

	service.now = func() time.Time { return base.Add(4 * time.Minute) }
	activateInstagramMatch(t, pool, service, InstagramMatchActivation{
		RecipientDID: recipient,
		SuggestionID: secondSuggestion,
		ActivityAt:   base.Add(4 * time.Minute),
	})
	var updatedID uuid.UUID
	var updatedRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT id, system_count, system_count_capped, indexed_at,
		       coalesce_until, newness_revision
		FROM notification_events
		WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
	`, recipient).Scan(&updatedID, &count, &capped, &indexedAt, &coalesceUntil, &updatedRevision); err != nil {
		t.Fatal(err)
	}
	if updatedID != eventID || count != 2 || capped || !indexedAt.Equal(base.Add(4*time.Minute)) || !coalesceUntil.Equal(base.Add(5*time.Minute)) || updatedRevision <= revision {
		t.Fatalf("coalesced id=%s count=%d capped=%t indexed=%s closes=%s revision=%d initialRevision=%d", updatedID, count, capped, indexedAt, coalesceUntil, updatedRevision, revision)
	}

	// An exact reconciliation replay is a no-op: it neither increments count nor
	// makes the same digest new again.
	activateInstagramMatch(t, pool, service, InstagramMatchActivation{
		RecipientDID: recipient,
		SuggestionID: secondSuggestion,
		ActivityAt:   base.Add(4*time.Minute + time.Second),
	})
	var replayRevision int64
	if err := pool.QueryRow(ctx, `SELECT system_count, newness_revision FROM notification_events WHERE id=$1`, eventID).Scan(&count, &replayRevision); err != nil {
		t.Fatal(err)
	}
	if count != 2 || replayRevision != updatedRevision {
		t.Fatalf("replay count=%d revision=%d, want 2/%d", count, replayRevision, updatedRevision)
	}

	// A delayed trigger whose source activity predates the digest's latest
	// activity still adds support without moving indexedAt backwards.
	activateInstagramMatch(t, pool, service, InstagramMatchActivation{
		RecipientDID: recipient,
		SuggestionID: olderSuggestion,
		ActivityAt:   base.Add(2 * time.Minute),
	})
	var delayedRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT system_count, indexed_at, coalesce_until, newness_revision
		FROM notification_events WHERE id=$1
	`, eventID).Scan(&count, &indexedAt, &coalesceUntil, &delayedRevision); err != nil {
		t.Fatal(err)
	}
	if count != 3 || !indexedAt.Equal(base.Add(4*time.Minute)) || !coalesceUntil.Equal(base.Add(5*time.Minute)) || delayedRevision <= replayRevision {
		t.Fatalf("delayed count=%d indexed=%s closes=%s revision=%d previous=%d", count, indexedAt, coalesceUntil, delayedRevision, replayRevision)
	}

	var supports, deliveries int
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions WHERE notification_id=$1`, eventID).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*), min(next_attempt_at) FROM push_deliveries WHERE notification_id=$1`, eventID).Scan(&deliveries, &nextAttempt); err != nil {
		t.Fatal(err)
	}
	if supports != 3 || deliveries != 1 || !nextAttempt.Equal(base.Add(5*time.Minute)) {
		t.Fatalf("supports=%d deliveries=%d nextAttempt=%s", supports, deliveries, nextAttempt)
	}

	service.now = func() time.Time { return base.Add(5 * time.Minute) }
	activateInstagramMatch(t, pool, service, InstagramMatchActivation{
		RecipientDID: recipient,
		SuggestionID: thirdSuggestion,
		ActivityAt:   base.Add(5 * time.Minute),
	})
	var events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events WHERE recipient_did=$1 AND kind='system'`, recipient).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM push_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if events != 2 || deliveries != 2 {
		t.Fatalf("exact-boundary events=%d deliveries=%d, want a new digest/outbox row", events, deliveries)
	}
}

func TestServiceHonoursTightenedInstagramNotificationLimits(t *testing.T) {
	pool := instagramNotificationPool(t)
	ctx := context.Background()
	recipient := syntax.DID("did:plc:instagram-notification-tightened")
	base := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	service, err := NewServiceWithOptions(ServiceOptions{
		InstagramCoalescingWindow: 2 * time.Minute,
		InstagramCountCap:         2,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return base }
	for index := range 3 {
		suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000082" + string(rune('1'+index)))
		seedInstagramNotificationSuggestion(t, pool, suggestionID)
		activateInstagramMatch(t, pool, service, InstagramMatchActivation{
			RecipientDID: recipient,
			SuggestionID: suggestionID,
			ActivityAt:   base.Add(time.Duration(index) * 10 * time.Second),
		})
	}

	var count int
	var capped bool
	var coalesceUntil time.Time
	if err := pool.QueryRow(ctx, `
		SELECT system_count, system_count_capped, coalesce_until
		FROM notification_events
		WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
	`, recipient).Scan(&count, &capped, &coalesceUntil); err != nil {
		t.Fatal(err)
	}
	if count != 2 || !capped || !coalesceUntil.Equal(base.Add(2*time.Minute)) {
		t.Fatalf("tightened notification count=%d capped=%t closes=%s", count, capped, coalesceUntil)
	}
}

func TestServiceRecountsAndRetractsInstagramMatchWithItsOutbox(t *testing.T) {
	pool := instagramNotificationPool(t)
	ctx := context.Background()
	recipient := syntax.DID("did:plc:instagram-retraction-recipient")
	firstSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000811")
	secondSuggestion := uuid.MustParse("00000000-0000-0000-0000-000000000812")
	seedInstagramNotificationSuggestion(t, pool, firstSuggestion)
	seedInstagramNotificationSuggestion(t, pool, secondSuggestion)
	seedInstagramNotificationSubscription(t, pool, recipient)

	base := time.Date(2026, 7, 19, 13, 0, 0, 0, time.UTC)
	service := NewService()
	service.now = func() time.Time { return base }
	for index, suggestionID := range []uuid.UUID{firstSuggestion, secondSuggestion} {
		activateInstagramMatch(t, pool, service, InstagramMatchActivation{
			RecipientDID: recipient,
			SuggestionID: suggestionID,
			ActivityAt:   base.Add(time.Duration(index) * time.Minute),
		})
	}

	var eventID uuid.UUID
	var beforeRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT id, newness_revision FROM notification_events
		WHERE recipient_did=$1 AND kind='system'
	`, recipient).Scan(&eventID, &beforeRevision); err != nil {
		t.Fatal(err)
	}

	service.now = func() time.Time { return base.Add(2 * time.Minute) }
	retractInstagramMatch(t, pool, service, InstagramMatchRetraction{
		SuggestionID: firstSuggestion,
		Reason:       "suggestion_invalidated",
	})
	var state, deliveryStatus string
	var count, supports int
	var afterPartialRevision int64
	if err := pool.QueryRow(ctx, `
		SELECT state, system_count, newness_revision
		FROM notification_events WHERE id=$1
	`, eventID).Scan(&state, &count, &afterPartialRevision); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions WHERE notification_id=$1`, eventID).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries WHERE notification_id=$1`, eventID).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if state != "active" || count != 1 || supports != 1 || deliveryStatus != "pending" || afterPartialRevision != beforeRevision {
		t.Fatalf("partial state=%s count=%d supports=%d delivery=%s revision=%d wantRevision=%d", state, count, supports, deliveryStatus, afterPartialRevision, beforeRevision)
	}

	service.now = func() time.Time { return base.Add(3 * time.Minute) }
	retractInstagramMatch(t, pool, service, InstagramMatchRetraction{
		SuggestionID: secondSuggestion,
		Reason:       "suggestion_invalidated",
	})
	if err := pool.QueryRow(ctx, `SELECT state FROM notification_events WHERE id=$1`, eventID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions WHERE notification_id=$1`, eventID).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries WHERE notification_id=$1`, eventID).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if state != "retracted" || supports != 0 || deliveryStatus != "cancelled" {
		t.Fatalf("zero-support state=%s supports=%d delivery=%s", state, supports, deliveryStatus)
	}
}

func TestSocialActivationRemainsIdempotentAfterSystemUnionMigration(t *testing.T) {
	pool := instagramNotificationPool(t)
	service := NewService()
	activity := time.Date(2026, 7, 19, 15, 0, 0, 0, time.UTC)
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
	var kind string
	var actorDID, sourceURI string
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*), min(kind), min(actor_did), min(source_uri)
		FROM notification_events
	`).Scan(&rows, &kind, &actorDID, &sourceURI); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || kind != "social" || actorDID != activation.ActorDID.String() || sourceURI != activation.SourceURI.String() {
		t.Fatalf("social rows=%d kind=%q actor=%q source=%q", rows, kind, actorDID, sourceURI)
	}
}

func activateInstagramMatch(t *testing.T, pool *pgxpool.Pool, service *Service, activation InstagramMatchActivation) {
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

func retractInstagramMatch(t *testing.T, pool *pgxpool.Pool, service *Service, retraction InstagramMatchRetraction) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if err := service.RetractInstagramMatch(context.Background(), tx, retraction); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func instagramNotificationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, `CREATE TABLE instagram_follow_suggestions(id UUID PRIMARY KEY);`)
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000024_system_notifications.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	return pool
}

func seedInstagramNotificationSuggestion(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO instagram_follow_suggestions(id) VALUES($1)`, id); err != nil {
		t.Fatal(err)
	}
}

func seedInstagramNotificationSubscription(t *testing.T, pool *pgxpool.Pool, recipient syntax.DID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ('10000000-0000-0000-0000-000000000801', 'instagram-device', 'ios', 'synthetic-token')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		VALUES (
			'20000000-0000-0000-0000-000000000801',
			'10000000-0000-0000-0000-000000000801', $1,
			'30000000-0000-0000-0000-000000000801'
		)
	`, recipient); err != nil {
		t.Fatal(err)
	}
}
