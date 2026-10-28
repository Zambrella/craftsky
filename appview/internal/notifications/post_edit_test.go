package notifications_test

import (
	"context"
	"testing"
	"time"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

// IT-012 / REG-007: edits do not create activity or reactivate publication alerts.
func TestPostPublicationActivationIsInsertOnce(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	service := notifications.NewService()
	originalTime := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	activation := notifications.Activation{RecipientDID: "did:plc:recipient", ActorDID: "did:plc:actor", Category: notifications.Mention, SubjectKey: "at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2", SourceURI: "at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2", SourceCID: "cid-original", SourceRkey: "3aaaaaaaaaaa2", ActivityAt: originalTime}
	activate := func() {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if err := notifications.ActivatePublication(ctx, service, tx, activation); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	activate()
	var originalRevision int64
	if err := pool.QueryRow(ctx, `SELECT newness_revision FROM notification_events`).Scan(&originalRevision); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO notification_seen_state(account_did,last_seen_revision) VALUES($1,$2)`, activation.RecipientDID, originalRevision); err != nil {
		t.Fatal(err)
	}
	activation.SourceCID = "cid-edited"
	activation.ActivityAt = originalTime.Add(time.Hour)
	activate()
	var revision int64
	var activity time.Time
	var read bool
	var state string
	var count int
	check := func(wantState string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT newness_revision,activity_at,newness_revision <= (SELECT last_seen_revision FROM notification_seen_state WHERE account_did=notification_events.recipient_did),state,(SELECT count(*) FROM notification_events) FROM notification_events`).Scan(&revision, &activity, &read, &state, &count); err != nil {
			t.Fatal(err)
		}
		if revision != originalRevision || !activity.Equal(originalTime) || !read || state != wantState || count != 1 {
			t.Fatalf("publication changed: revision=%d original=%d activity=%s read=%v state=%s count=%d", revision, originalRevision, activity, read, state, count)
		}
	}
	check("active")
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := service.Retract(ctx, tx, notifications.Retraction{SourceURI: activation.SourceURI, Reason: "removedReason"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	activate()
	check("retracted")
	activation.Category = notifications.Quote
	activate()
	check("retracted")
}
