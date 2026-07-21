package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestInstagramMatchFeedItemIsAnExactActorlessSystemVariant(t *testing.T) {
	t.Parallel()

	store := &fakeNotificationStore{rows: []*api.NotificationRow{{
		ID:   "00000000-0000-0000-0000-000000000321",
		Kind: api.NotificationKindSystem,
		Type: api.NotificationTypeInstagramMatch,
		System: &api.NotificationSystem{
			Count:       99,
			CountCapped: true,
			Destination: api.NotificationDestinationInstagramMigration,
		},
		CreatedAt: time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
		IndexedAt: time.Date(2026, 7, 19, 12, 4, 0, 0, time.UTC),
	}}}
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
	if len(item) != 6 {
		t.Fatalf("system item fields=%v, want exact six-field union variant", mapKeys(item))
	}
	for key, want := range map[string]string{
		"id":        `"00000000-0000-0000-0000-000000000321"`,
		"kind":      `"system"`,
		"type":      `"instagramMatch"`,
		"createdAt": `"2026-07-19T12:00:00Z"`,
		"indexedAt": `"2026-07-19T12:04:00Z"`,
	} {
		if got := string(item[key]); got != want {
			t.Fatalf("%s=%s, want %s", key, got, want)
		}
	}
	var system map[string]any
	if err := json.Unmarshal(item["system"], &system); err != nil {
		t.Fatal(err)
	}
	if len(system) != 3 || system["count"] != float64(99) || system["countCapped"] != true || system["destination"] != "instagramMigration" {
		t.Fatalf("system=%#v", system)
	}
	for _, forbidden := range []string{"actor", "uri", "cid", "rkey", "references", "subjectPost", "reply", "contentAvailable"} {
		if _, ok := item[forbidden]; ok {
			t.Fatalf("actorless system item contains %q: %s", forbidden, recorder.Body.String())
		}
	}
	if store.handleCalls != 0 || len(store.engagementIn) != 0 {
		t.Fatalf("actorless row performed social hydration: handleCalls=%d engagement=%v", store.handleCalls, store.engagementIn)
	}
}

func TestInstagramMatchStoreReadsCheckedUnionAndOrdersByIndexedAt(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyInstagramNotificationMigrations(t, pool)
	ctx := context.Background()

	socialActivity := time.Date(2026, 7, 19, 13, 0, 0, 0, time.UTC)
	socialIndexed := time.Date(2026, 7, 19, 12, 1, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, actor_did, category, subject_key,
			source_uri, source_cid, source_rkey, eligibility_scope,
			recipient_followed_actor, push_enabled_snapshot, state,
			first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000401', 'did:plc:viewer',
			'social', 'did:plc:actor', 'follow', 'social-follow',
			'at://did:plc:actor/app.bsky.graph.follow/social', 'social-cid', 'social',
			'everyone', false, true, 'active', $1, $1, $2, $2
		)
	`, socialActivity, socialIndexed); err != nil {
		t.Fatalf("insert social notification: %v", err)
	}

	systemCreated := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	systemIndexed := time.Date(2026, 7, 19, 12, 4, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, category, subject_key,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
			state, first_activity_at, activity_at, indexed_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_destination, system_group_key, coalesce_until
		) VALUES (
			'00000000-0000-0000-0000-000000000402', 'did:plc:viewer',
			'system', 'instagramMatch', 'instagram-system',
			'everyone', false, true, 'active', $1, $2, $2, $2,
			3, false, 'instagramMigration', 'instagram-group', $1::timestamptz + interval '5 minutes'
		)
	`, systemCreated, systemIndexed); err != nil {
		t.Fatalf("insert system notification: %v", err)
	}

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListNotifications(ctx, "did:plc:viewer", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Kind != api.NotificationKindSystem || first[0].Type != api.NotificationTypeInstagramMatch || first[0].ActorDID != "" || first[0].System == nil {
		t.Fatalf("first page=%+v", first)
	}
	if *first[0].System != (api.NotificationSystem{Count: 3, CountCapped: false, Destination: api.NotificationDestinationInstagramMigration}) {
		t.Fatalf("system=%+v", first[0].System)
	}
	if !first[0].CreatedAt.Equal(systemCreated) || !first[0].IndexedAt.Equal(systemIndexed) || cursor == "" {
		t.Fatalf("system times created=%s indexed=%s cursor=%q", first[0].CreatedAt, first[0].IndexedAt, cursor)
	}

	second, finalCursor, err := store.ListNotifications(ctx, "did:plc:viewer", 1, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Kind != api.NotificationKindSocial || second[0].Type != api.NotificationTypeFollow || second[0].ActorDID != "did:plc:actor" || second[0].System != nil || finalCursor != "" {
		t.Fatalf("second page=%+v cursor=%q", second, finalCursor)
	}
}

func TestInstagramMatchNewnessTracksAdditionsButNotRetractions(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyInstagramNotificationMigrations(t, pool)
	ctx := context.Background()
	store := api.NewPostStore(pool)
	base := time.Date(2026, 7, 19, 14, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, kind, category, subject_key,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
			state, first_activity_at, activity_at, indexed_at,
			initial_push_evaluated_at, system_count, system_count_capped,
			system_destination, system_group_key, coalesce_until
		) VALUES (
			'00000000-0000-0000-0000-000000000421', 'did:plc:viewer',
			'system', 'instagramMatch', 'instagram-newness',
			'everyone', false, true, 'active', $1, $1, $1, $1,
			1, false, 'instagramMigration', 'instagram-newness',
			$1::timestamptz + interval '5 minutes'
		)
	`, base); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkNotificationsSeen(ctx, "did:plc:viewer"); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE notification_events
		SET system_count=2, activity_at=$2, indexed_at=$2
		WHERE id=$1
	`, uuid.MustParse("00000000-0000-0000-0000-000000000421"), base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	count, err := store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("new count after addition=%d, want 1", count)
	}
	if err := store.MarkNotificationsSeen(ctx, "did:plc:viewer"); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `UPDATE notification_events SET system_count=1 WHERE id=$1`, uuid.MustParse("00000000-0000-0000-0000-000000000421")); err != nil {
		t.Fatal(err)
	}
	count, err = store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("count reduction created false newness: %d", count)
	}

	if _, err := pool.Exec(ctx, `UPDATE notification_events SET state='retracted', retracted_at=$2 WHERE id=$1`, uuid.MustParse("00000000-0000-0000-0000-000000000421"), base.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	count, err = store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("retracted system item remained new: %d", count)
	}
}

func applyInstagramNotificationMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000023_instagram_migration.up.sql",
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
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
