package notifications_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationNewnessRevisionTracksGenuineActivations(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
	} {
		migration, err := os.ReadFile(path)
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
	if err := service.Retract(ctx, tx, notifications.Retraction{SourceURI: activation.SourceURI, Reason: "source_deleted"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if retracted := revision(); retracted != first {
		t.Fatalf("retraction revision = %d, want %d", retracted, first)
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
