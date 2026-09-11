package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationNewCountUsesAccountMarkerAndListVisibility(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
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
	seedMember(t, pool, "did:plc:viewer")

	insert := func(id, actor, state string) int64 {
		t.Helper()
		var revision int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO notification_events (
				id, recipient_did, actor_did, category, subject_key,
				source_uri, source_cid, source_rkey,
				eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
				state, first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
			) VALUES (
				$1::uuid, 'did:plc:viewer', $2, 'follow', $1::uuid::text,
				'at://' || $2 || '/app.bsky.graph.follow/' || $1::uuid::text, 'cid', $1::uuid::text,
				'everyone', false, true, $3, now(), now(), now(), now()
			)
			RETURNING newness_revision
		`, id, actor, state).Scan(&revision); err != nil {
			t.Fatalf("insert notification: %v", err)
		}
		return revision
	}

	oldRevision := insert("00000000-0000-0000-0000-000000000001", "did:plc:visible-old", "active")
	insert("00000000-0000-0000-0000-000000000002", "did:plc:retracted", "retracted")
	insert("00000000-0000-0000-0000-000000000003", "did:plc:hidden", "active")
	insert("00000000-0000-0000-0000-000000000004", "did:plc:visible-new", "active")
	seedModerationOutput(t, pool, "account", "did:plc:hidden", "", "hide", time.Now().UTC())

	handler := api.NotificationNewCountHandler(api.NewPostStore(pool), nilLogger())
	requestCount := func() int64 {
		t.Helper()
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications/new-count", "", "did:plc:viewer"))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var body struct {
			NewCount int64 `json:"newCount"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return body.NewCount
	}

	if count := requestCount(); count != 2 {
		t.Fatalf("first-use count=%d, want 2", count)
	}
	listed, _, err := api.NewPostStore(pool).ListNotifications(ctx, "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed=%d, want 2 to match new count visibility", len(listed))
	}
	var markerRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_seen_state`).Scan(&markerRows); err != nil {
		t.Fatal(err)
	}
	if markerRows != 0 {
		t.Fatalf("GET created %d acknowledgement rows, want 0", markerRows)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_seen_state (account_did, last_seen_revision)
		VALUES ('did:plc:viewer', $1)
	`, oldRevision); err != nil {
		t.Fatal(err)
	}
	if count := requestCount(); count != 1 {
		t.Fatalf("count after marker=%d, want 1", count)
	}
}

func TestActorlessModerationNotificationIsListedCountedAndHasNoActorDTO(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
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
	if _, err := pool.Exec(ctx, `
		ALTER TABLE notification_events
			DROP CONSTRAINT notification_events_category_check,
			ALTER COLUMN actor_did DROP NOT NULL,
			ALTER COLUMN source_uri DROP NOT NULL,
			ALTER COLUMN source_cid DROP NOT NULL,
			ALTER COLUMN source_rkey DROP NOT NULL,
			ADD COLUMN moderation_case_reference TEXT,
			ADD CONSTRAINT notification_events_category_check CHECK (category IN (
				'like','follow','reply','mention','quote','repost','everythingElse','moderation'
			));
		CREATE OR REPLACE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL IMMUTABLE
		AS $$ SELECT candidate_did IS NULL $$;
		INSERT INTO notification_events(
			id,recipient_did,category,subject_key,moderation_case_reference,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
			first_activity_at,activity_at,indexed_at,initial_push_evaluated_at
		) VALUES(
			'00000000-0000-4000-8000-000000000001','did:plc:viewer','moderation','event-1',
			'MOD-550e8400-e29b-41d4-a716-446655440000','everyone',false,true,'active',
			now(),now(),now(),now()
		)
	`); err != nil {
		t.Fatal(err)
	}

	store := api.NewPostStore(pool)
	count, err := store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil || count != 1 {
		t.Fatalf("new count=%d err=%v", count, err)
	}
	rows, _, err := store.ListNotifications(ctx, "did:plc:viewer", 20, "")
	if err != nil || len(rows) != 1 {
		t.Fatalf("listed=%d err=%v", len(rows), err)
	}
	if rows[0].ActorDID != "" || rows[0].CaseReference != "MOD-550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("moderation row=%+v", rows[0])
	}

	recorder := httptest.NewRecorder()
	handler := api.ListNotificationsHandler(store, nil, nilLogger())
	handler.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Items []struct {
			Type          string          `json:"type"`
			Actor         json.RawMessage `json:"actor"`
			CaseReference string          `json:"caseReference"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Type != "moderation" || len(body.Items[0].Actor) != 0 ||
		body.Items[0].CaseReference != "MOD-550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("moderation DTO=%s", recorder.Body.String())
	}
}

func TestNotificationMarkSeenUsesStatementSnapshot(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
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
	seedMember(t, pool, "did:plc:viewer")

	insert := func(id string) int64 {
		t.Helper()
		var revision int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO notification_events (
				id, recipient_did, actor_did, category, subject_key,
				source_uri, source_cid, source_rkey,
				eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
				state, first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
			) VALUES (
				$1::uuid, 'did:plc:viewer', 'did:plc:actor', 'follow', $1::uuid::text,
				'at://did:plc:actor/app.bsky.graph.follow/' || $1::uuid::text, 'cid', $1::uuid::text,
				'everyone', false, true, 'active', now(), now(), now(), now()
			)
			RETURNING newness_revision
		`, id).Scan(&revision); err != nil {
			t.Fatalf("insert notification: %v", err)
		}
		return revision
	}

	firstRevision := insert("10000000-0000-0000-0000-000000000001")
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_seen_state (account_did, last_seen_revision)
		VALUES ('did:plc:viewer', 0)
	`); err != nil {
		t.Fatal(err)
	}

	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx, `
		UPDATE notification_seen_state
		SET updated_at = updated_at
		WHERE account_did = 'did:plc:viewer'
	`); err != nil {
		t.Fatal(err)
	}
	var blockerXID string
	if err := blocker.QueryRow(ctx, `SELECT txid_current()::text`).Scan(&blockerXID); err != nil {
		t.Fatal(err)
	}

	store := api.NewPostStore(pool)
	result := make(chan error, 1)
	go func() {
		result <- store.MarkNotificationsSeen(ctx, "did:plc:viewer")
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'transactionid'
				  AND transactionid = $1::xid
				  AND NOT granted
			)
		`, blockerXID).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("mark-seen did not reach the blocked acknowledgement upsert")
		}
		time.Sleep(10 * time.Millisecond)
	}

	secondRevision := insert("10000000-0000-0000-0000-000000000002")
	if secondRevision <= firstRevision {
		t.Fatalf("second revision=%d, want greater than %d", secondRevision, firstRevision)
	}
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatalf("mark seen: %v", err)
	}

	var marker int64
	if err := pool.QueryRow(ctx, `
		SELECT last_seen_revision
		FROM notification_seen_state
		WHERE account_did = 'did:plc:viewer'
	`).Scan(&marker); err != nil {
		t.Fatal(err)
	}
	if marker != firstRevision {
		t.Fatalf("marker=%d, want captured revision %d", marker, firstRevision)
	}
	count, err := store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("new count=%d, want concurrent notification to remain new", count)
	}

	recorder := httptest.NewRecorder()
	api.MarkNotificationsSeenHandler(store, nilLogger()).ServeHTTP(
		recorder,
		authedReq(http.MethodPost, "/v1/notifications/seen", "", "did:plc:viewer"),
	)
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("mark-seen response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	count, err = store.NotificationNewCount(ctx, "did:plc:viewer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("new count after second acknowledgement=%d, want 0", count)
	}
}
