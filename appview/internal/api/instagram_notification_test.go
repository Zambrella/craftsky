package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestInstagramMatchFeedItemIsActorfulAndSourceLess(t *testing.T) {
	t.Parallel()

	displayName := "Synthetic Friend"
	store := &fakeNotificationStore{
		handles: map[string]syntax.Handle{
			"did:plc:synthetic-match-actor": syntax.Handle("synthetic.test"),
		},
		rows: []*api.NotificationRow{{
			ID:                     "00000000-0000-0000-0000-000000000321",
			Type:                   api.NotificationTypeInstagramMatch,
			ActorDID:               "did:plc:synthetic-match-actor",
			ActorDisplayName:       &displayName,
			ActorViewerIsFollowing: true,
			CreatedAt:              time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			IndexedAt:              time.Date(2026, 7, 27, 12, 0, 1, 0, time.UTC),
		}},
	}
	recorder := httptest.NewRecorder()
	api.ListNotificationsHandler(store, fakeResolver{}, nilLogger()).ServeHTTP(
		recorder,
		authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items=%d", len(response.Items))
	}
	item := response.Items[0]
	if _, present := item["actor"]; !present {
		t.Fatalf("actorful item has no actor: %s", recorder.Body.String())
	}
	for _, forbidden := range []string{
		"system", "uri", "cid", "rkey", "references", "subjectPost", "reply",
	} {
		if _, present := item[forbidden]; present {
			t.Fatalf("source-less item contains %q: %s", forbidden, recorder.Body.String())
		}
	}
	if store.handleCalls != 1 {
		t.Fatalf("handle hydration calls=%d", store.handleCalls)
	}
}

func TestInstagramMatchStoreReadsActorfulMigratedRow(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyInstagramNotificationMigrations(t, pool)
	ctx := context.Background()
	createdAt := time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC)
	indexedAt := createdAt.Add(time.Second)
	seedFollow(
		t,
		pool,
		"did:plc:viewer",
		"did:plc:synthetic-match-actor",
		"3kinstagrammatch",
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			indexed_at, initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000402',
			'did:plc:viewer',
			'did:plc:synthetic-match-actor',
			'instagramMatch',
			'00000000-0000-0000-0000-000000000403',
			'everyone',true,true,'active',$1,$1,$2,$2
		)
	`, createdAt, indexedAt); err != nil {
		t.Fatal(err)
	}
	rows, cursor, err := api.NewPostStore(pool).ListNotifications(
		ctx,
		"did:plc:viewer",
		20,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || cursor != "" {
		t.Fatalf("rows=%+v cursor=%q", rows, cursor)
	}
	row := rows[0]
	if row.Type != api.NotificationTypeInstagramMatch ||
		row.ActorDID != "did:plc:synthetic-match-actor" ||
		!row.ActorViewerIsFollowing ||
		row.URI != "" || row.CID != "" || row.Rkey != "" {
		t.Fatalf("row=%+v", row)
	}
}

func applyInstagramNotificationMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
		"../../migrations/000031_instagram_automatic_follow_storage_names.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
}
