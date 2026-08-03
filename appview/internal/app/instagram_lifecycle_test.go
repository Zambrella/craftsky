package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

type lifecycleActorDeletionFake struct {
	calls *[]string
	name  string
	err   error
}

func (f lifecycleActorDeletionFake) HardDeleteByActor(context.Context, pgx.Tx, syntax.DID) error {
	name := f.name
	if name == "" {
		name = "notifications"
	}
	*f.calls = append(*f.calls, name)
	return f.err
}

type lifecycleInstagramInactivatorFake struct {
	calls *[]string
	now   time.Time
	err   error
}

func (f *lifecycleInstagramInactivatorFake) InactivateMembershipTx(_ context.Context, _ pgx.Tx, _ syntax.DID, now time.Time) error {
	*f.calls = append(*f.calls, "instagram")
	f.now = now
	return f.err
}

func TestProfileMembershipDeletionComposesCleanupInOrder(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.FixedZone("test", 3600))
	calls := make([]string, 0, 3)
	instagram := &lifecycleInstagramInactivatorFake{calls: &calls}
	lifecycle := &profileMembershipDeletion{
		notifications: lifecycleActorDeletionFake{calls: &calls},
		scheduled: lifecycleActorDeletionFake{
			calls: &calls,
			name:  "scheduled",
		},
		instagram: instagram,
		now:       func() time.Time { return now },
	}
	if err := lifecycle.HardDeleteByActor(context.Background(), nil, "did:plc:member"); err != nil {
		t.Fatalf("HardDeleteByActor: %v", err)
	}
	if want := []string{"notifications", "scheduled", "instagram"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if want := now.UTC(); !instagram.now.Equal(want) || instagram.now.Location() != time.UTC {
		t.Fatalf("inactivation time = %v, want UTC %v", instagram.now, want)
	}
}

func TestProfileMembershipDeletionStopsBeforeInactivationOnNotificationFailure(t *testing.T) {
	calls := make([]string, 0, 1)
	wantErr := errors.New("synthetic notification failure")
	lifecycle := &profileMembershipDeletion{
		notifications: lifecycleActorDeletionFake{calls: &calls, err: wantErr},
		scheduled:     lifecycleActorDeletionFake{calls: &calls, name: "scheduled"},
		instagram:     &lifecycleInstagramInactivatorFake{calls: &calls},
		now:           time.Now,
	}
	err := lifecycle.HardDeleteByActor(context.Background(), nil, "did:plc:member")
	if !errors.Is(err, wantErr) {
		t.Fatalf("HardDeleteByActor error = %v, want %v", err, wantErr)
	}
	if want := []string{"notifications"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

type lifecycleBackfillerFake struct{}

func (lifecycleBackfillerFake) Backfill(context.Context, syntax.DID) error { return nil }

func TestProfileIndexerDeletionQueuesScheduledPrivateMediaCleanup(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000034_scheduled_posts.up.sql")
	if err != nil {
		t.Fatalf("read scheduled-post migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT NOT NULL PRIMARY KEY,
			crafts TEXT[] NOT NULL DEFAULT '{}',
			record_cid TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE bluesky_profiles (
			did TEXT NOT NULL PRIMARY KEY,
			display_name TEXT,
			description TEXT,
			avatar_cid TEXT,
			avatar_mime TEXT,
			banner_cid TEXT,
			banner_mime TEXT,
			record_cid TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`+string(migration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:scheduled-profile-delete")
	if _, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'profile-cid')`,
		owner); err != nil {
		t.Fatalf("seed Craftsky profile: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO bluesky_profiles (did, record_cid) VALUES ($1, 'bluesky-cid')`,
		owner); err != nil {
		t.Fatalf("seed Bluesky profile: %v", err)
	}

	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	scheduleID := uuid.MustParse("00000000-0000-4000-8000-000000000230")
	payload := []byte(`{"kind":"standard","text":"private"}`)
	payloadHash := sha256.Sum256(payload)
	requestHash := sha256.Sum256([]byte("scheduled-profile-delete"))
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_posts (
			id, owner_did, operation_id, request_hash, status, scheduled_at,
			next_attempt_at, payload_bytes, payload_hash
		) VALUES ($1, $2, $3, $4, 'scheduled', $5, $5, $6, $7)
	`, scheduleID, owner,
		uuid.MustParse("10000000-0000-4000-8000-000000000230"),
		requestHash[:], now.Add(time.Hour), payload, payloadHash[:]); err != nil {
		t.Fatalf("seed scheduled post: %v", err)
	}
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000231")
	objectKey := "scheduled-media/" + mediaID.String()
	mediaHash := sha256.Sum256([]byte("data"))
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_post_media (
			id, owner_did, object_key, state, schedule_id, ordinal,
			mime_type, size_bytes, sha256, blob_cid, unclaimed_expires_at
		) VALUES ($1, $2, $3, 'ready', $4, 0,
			'image/jpeg', 4, $5, 'bafk-private-profile-delete', $6)
	`, mediaID, owner, objectKey, scheduleID, mediaHash[:], now.Add(24*time.Hour)); err != nil {
		t.Fatalf("seed scheduled media: %v", err)
	}
	tombstoneHash := sha256.Sum256([]byte("published"))
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_post_publication_tombstones (
			schedule_id, owner_did, operation_id, request_hash,
			publication_uri, publication_cid, published_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, 'bafy-published', $6, $7)
	`, uuid.MustParse("00000000-0000-4000-8000-000000000232"), owner,
		uuid.MustParse("10000000-0000-4000-8000-000000000232"),
		tombstoneHash[:],
		"at://"+owner.String()+"/social.craftsky.feed.post/3mprofiledelete",
		now.Add(-time.Hour), now.Add(30*24*time.Hour)); err != nil {
		t.Fatalf("seed scheduled tombstone: %v", err)
	}

	calls := make([]string, 0, 3)
	deletion := &profileMembershipDeletion{
		notifications: lifecycleActorDeletionFake{calls: &calls},
		scheduled:     scheduledposts.NewAccountDeletion(pool, func() time.Time { return now }),
		instagram:     &lifecycleInstagramInactivatorFake{calls: &calls},
		now:           func() time.Time { return now },
	}
	idx := index.NewCraftskyProfile(
		pool,
		lifecycleBackfillerFake{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		deletion,
	)
	if err := idx.Handle(ctx, tap.Event{
		URI:        syntax.ATURI("at://" + owner.String() + "/social.craftsky.actor.profile/self"),
		DID:        owner,
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "delete",
	}); err != nil {
		t.Fatalf("delete profile through indexer: %v", err)
	}

	for table, want := range map[string]int{
		"craftsky_profiles":                     0,
		"scheduled_posts":                       0,
		"scheduled_post_media":                  0,
		"scheduled_post_publication_tombstones": 0,
		"scheduled_post_cleanup_jobs":           1,
	} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s count = %d, want %d", table, got, want)
		}
	}
	var queuedKey string
	if err := pool.QueryRow(ctx, `SELECT object_key FROM scheduled_post_cleanup_jobs`).Scan(&queuedKey); err != nil {
		t.Fatalf("read scheduled cleanup job: %v", err)
	}
	if queuedKey != objectKey {
		t.Fatalf("queued object key = %q, want %q", queuedKey, objectKey)
	}
}

type lifecycleIdentityDeletionFake struct {
	name  string
	calls *[]string
	err   error
}

func (f lifecycleIdentityDeletionFake) HandleIdentityDeleted(context.Context, syntax.DID) error {
	*f.calls = append(*f.calls, f.name)
	return f.err
}

func TestTerminalIdentityDeletionRetriesThroughOrderedIdempotentHandlers(t *testing.T) {
	calls := make([]string, 0, 2)
	wantErr := errors.New("synthetic private purge failure")
	lifecycle := &terminalIdentityDeletion{handlers: []tap.IdentityDeletionHandler{
		lifecycleIdentityDeletionFake{name: "notifications", calls: &calls},
		lifecycleIdentityDeletionFake{name: "instagram", calls: &calls, err: wantErr},
	}}
	err := lifecycle.HandleIdentityDeleted(context.Background(), "did:plc:deleted")
	if !errors.Is(err, wantErr) {
		t.Fatalf("HandleIdentityDeleted error = %v, want %v", err, wantErr)
	}
	if want := []string{"notifications", "instagram"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}
