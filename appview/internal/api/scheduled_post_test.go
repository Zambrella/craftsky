package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/testdb"
	"social.craftsky/appview/internal/testlog"
)

func TestScheduledPostCreateIsOwnerDerivedAndIdempotent(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	handler := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, testlog.Discard())
	operationID := "00000000-0000-4000-8000-000000000601"
	body := `{"operationId":"` + operationID + `","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"private scheduled text","sponsored":true,"langs":["en"]}}`

	first := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", body, "did:plc:alice")
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.Code, first.Body.String())
	}
	firstBody := decodeResponseJSON[map[string]any](t, first)
	if firstBody["id"] == "" || firstBody["status"] != "scheduled" || firstBody["operationId"] != operationID {
		t.Fatalf("first response=%v", firstBody)
	}
	payload := firstBody["payload"].(map[string]any)
	if payload["sponsored"] != true {
		t.Fatalf("first response sponsored = %#v, want true", payload["sponsored"])
	}

	second := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", body, "did:plc:alice")
	if second.Code != http.StatusOK {
		t.Fatalf("idempotent create status=%d body=%s", second.Code, second.Body.String())
	}
	secondBody := decodeResponseJSON[map[string]any](t, second)
	if secondBody["id"] != firstBody["id"] {
		t.Fatalf("idempotent resource id=%v, want %v", secondBody["id"], firstBody["id"])
	}

	conflicting := bytes.ReplaceAll([]byte(body), []byte("private scheduled text"), []byte("changed private text"))
	conflict := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", string(conflicting), "did:plc:alice")
	if conflict.Code != http.StatusConflict {
		t.Fatalf("operation conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
	assertScheduledPostError(t, conflict, "scheduled_operation_conflict")

	bob := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", body, "did:plc:bob")
	if bob.Code != http.StatusCreated {
		t.Fatalf("other-owner create status=%d body=%s", bob.Code, bob.Body.String())
	}
	var owners int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(DISTINCT owner_did) FROM scheduled_posts WHERE operation_id=$1
	`, operationID).Scan(&owners); err != nil {
		t.Fatalf("count operation owners: %v", err)
	}
	if owners != 2 {
		t.Fatalf("operation owner count=%d, want 2", owners)
	}
}

func TestScheduledPostListAndGetAreOwnerScopedOrderedAndShaped(t *testing.T) {
	store, _ := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	logger := testlog.Discard()
	create := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, logger)
	list := ListScheduledPostsHandler(store, logger)
	get := GetScheduledPostHandler(store, logger)

	late := serveScheduledPostRequest(t, create, http.MethodPost, "/v1/scheduled-posts", `{"operationId":"00000000-0000-4000-8000-000000000621","scheduledAt":"2026-08-02T12:10:00Z","payload":{"kind":"standard","text":"later private text","sponsored":false}}`, "did:plc:alice")
	early := serveScheduledPostRequest(t, create, http.MethodPost, "/v1/scheduled-posts", `{"operationId":"00000000-0000-4000-8000-000000000622","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"earlier private text","sponsored":false}}`, "did:plc:alice")
	bob := serveScheduledPostRequest(t, create, http.MethodPost, "/v1/scheduled-posts", `{"operationId":"00000000-0000-4000-8000-000000000623","scheduledAt":"2026-08-02T12:06:00Z","payload":{"kind":"standard","text":"bob private text","sponsored":false}}`, "did:plc:bob")
	for name, response := range map[string]*httptest.ResponseRecorder{"late": late, "early": early, "bob": bob} {
		if response.Code != http.StatusCreated {
			t.Fatalf("%s create status=%d body=%s", name, response.Code, response.Body.String())
		}
	}
	lateBody := decodeResponseJSON[map[string]any](t, late)
	earlyBody := decodeResponseJSON[map[string]any](t, early)
	bobBody := decodeResponseJSON[map[string]any](t, bob)

	listed := serveScheduledPostRequest(t, list, http.MethodGet, "/v1/scheduled-posts", "", "did:plc:alice")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	listBody := decodeResponseJSON[struct {
		Items               []map[string]any `json:"items"`
		Count               int              `json:"count"`
		NeedsAttentionCount int              `json:"needsAttentionCount"`
	}](t, listed)
	if listBody.Count != 2 || listBody.NeedsAttentionCount != 0 || len(listBody.Items) != 2 {
		t.Fatalf("list=%+v", listBody)
	}
	if listBody.Items[0]["id"] != earlyBody["id"] || listBody.Items[1]["id"] != lateBody["id"] {
		t.Fatalf("list order=%v, want early then late", listBody.Items)
	}
	for _, item := range listBody.Items {
		if _, leaked := item["payload"]; leaked {
			t.Fatalf("list leaked full payload: %v", item)
		}
		if item["textPreview"] == "" || item["kind"] != "standard" || item["status"] != "scheduled" {
			t.Fatalf("list item shape=%v", item)
		}
	}

	owned := serveScheduledPostPathRequest(t, get, http.MethodGet, earlyBody["id"].(string), "", "did:plc:alice")
	if owned.Code != http.StatusOK {
		t.Fatalf("get owner status=%d body=%s", owned.Code, owned.Body.String())
	}
	ownedBody := decodeResponseJSON[map[string]any](t, owned)
	payload, ok := ownedBody["payload"].(map[string]any)
	if !ok || payload["text"] != "earlier private text" {
		t.Fatalf("owner detail=%v", ownedBody)
	}

	foreign := serveScheduledPostPathRequest(t, get, http.MethodGet, bobBody["id"].(string), "", "did:plc:alice")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign get status=%d body=%s", foreign.Code, foreign.Body.String())
	}
	assertScheduledPostError(t, foreign, "scheduled_post_not_found")
}

func TestScheduledPostUpdateAndDeleteRespectOwnershipAndPublishingLock(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	logger := testlog.Discard()
	create := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, logger)
	update := UpdateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, logger)
	remove := DeleteScheduledPostHandler(store, func() time.Time { return now }, logger)
	created := serveScheduledPostRequest(t, create, http.MethodPost, "/v1/scheduled-posts", `{"operationId":"00000000-0000-4000-8000-000000000631","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"before edit","sponsored":false}}`, "did:plc:alice")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	createdBody := decodeResponseJSON[map[string]any](t, created)
	id := createdBody["id"].(string)

	foreignUpdate := serveScheduledPostPathRequest(t, update, http.MethodPut, id, `{"scheduledAt":"2026-08-02T12:06:00Z","payload":{"kind":"standard","text":"foreign edit","sponsored":false}}`, "did:plc:bob")
	if foreignUpdate.Code != http.StatusNotFound {
		t.Fatalf("foreign update status=%d body=%s", foreignUpdate.Code, foreignUpdate.Body.String())
	}
	assertScheduledPostError(t, foreignUpdate, "scheduled_post_not_found")

	edited := serveScheduledPostPathRequest(t, update, http.MethodPut, id, `{"scheduledAt":"2026-08-02T12:06:00Z","payload":{"kind":"standard","text":"after edit","sponsored":false}}`, "did:plc:alice")
	if edited.Code != http.StatusOK {
		t.Fatalf("owner update status=%d body=%s", edited.Code, edited.Body.String())
	}
	editedBody := decodeResponseJSON[map[string]any](t, edited)
	if _, leaked := editedBody["payloadVersion"]; leaked {
		t.Fatalf("edit response leaked internal payload version: %s", edited.Body.String())
	}
	if editedBody["scheduledAt"] != "2026-08-02T12:06:00Z" {
		t.Fatalf("edited response=%v", editedBody)
	}

	foreignDelete := serveScheduledPostPathRequest(t, remove, http.MethodDelete, id, "", "did:plc:bob")
	if foreignDelete.Code != http.StatusNoContent {
		t.Fatalf("foreign delete status=%d body=%s", foreignDelete.Code, foreignDelete.Body.String())
	}
	var text string
	if err := pool.QueryRow(t.Context(), `SELECT convert_from(payload_bytes, 'UTF8') FROM scheduled_posts WHERE id=$1`, id).Scan(&text); err != nil {
		t.Fatalf("read after foreign delete: %v", err)
	}
	if text == "" || !bytes.Contains([]byte(text), []byte("after edit")) {
		t.Fatalf("foreign delete changed payload=%q", text)
	}

	if _, err := pool.Exec(t.Context(), `
		UPDATE scheduled_posts
		SET status='publishing', lease_token='00000000-0000-4000-8000-000000000632',
		    lease_expires_at=$2, publication_rkey='3mtest', publication_created_at=$3
		WHERE id=$1
	`, id, now.Add(time.Minute), now); err != nil {
		t.Fatalf("prepare publishing fixture: %v", err)
	}
	lockedUpdate := serveScheduledPostPathRequest(t, update, http.MethodPut, id, `{"scheduledAt":"2026-08-02T12:07:00Z","payload":{"kind":"standard","text":"locked edit","sponsored":false}}`, "did:plc:alice")
	if lockedUpdate.Code != http.StatusConflict {
		t.Fatalf("locked update status=%d body=%s", lockedUpdate.Code, lockedUpdate.Body.String())
	}
	assertScheduledPostError(t, lockedUpdate, "scheduled_post_publishing")
	lockedDelete := serveScheduledPostPathRequest(t, remove, http.MethodDelete, id, "", "did:plc:alice")
	if lockedDelete.Code != http.StatusConflict {
		t.Fatalf("locked delete status=%d body=%s", lockedDelete.Code, lockedDelete.Body.String())
	}
	assertScheduledPostError(t, lockedDelete, "scheduled_post_publishing")

	if _, err := pool.Exec(t.Context(), `
		UPDATE scheduled_posts
		SET status='scheduled', lease_token=NULL, lease_expires_at=NULL,
		    publication_rkey=NULL, publication_created_at=NULL
		WHERE id=$1
	`, id); err != nil {
		t.Fatalf("restore scheduled fixture: %v", err)
	}
	deleted := serveScheduledPostPathRequest(t, remove, http.MethodDelete, id, "", "did:plc:alice")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("owner delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	repeated := serveScheduledPostPathRequest(t, remove, http.MethodDelete, id, "", "did:plc:alice")
	if repeated.Code != http.StatusNoContent {
		t.Fatalf("repeat delete status=%d body=%s", repeated.Code, repeated.Body.String())
	}
}

type recordingManualScheduledPublisher struct {
	params  scheduledposts.UpdateParams
	outcome scheduledposts.ManualPublicationOutcome
	err     error
}

func (publisher *recordingManualScheduledPublisher) PublishManual(
	_ context.Context,
	params scheduledposts.UpdateParams,
) (scheduledposts.ManualPublicationOutcome, error) {
	publisher.params = params
	return publisher.outcome, publisher.err
}

func TestScheduledPostPublicationSubresourceAttemptsTheFullEditImmediately(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	logger := testlog.Discard()
	publisher := &recordingManualScheduledPublisher{
		outcome: scheduledposts.ManualPublicationPublished,
	}
	handler := PublishScheduledPostHandler(
		publisher,
		DefaultMediaLimits(),
		func() time.Time { return now },
		logger,
	)
	id := "00000000-0000-4000-8000-000000000641"
	response := serveScheduledPostPathRequest(t, handler, http.MethodPost, id, `{"payload":{"kind":"standard","text":"post now","sponsored":false}}`, "did:plc:alice")
	if response.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", response.Code, response.Body.String())
	}
	if publisher.params.ID.String() != id ||
		publisher.params.OwnerDID != "did:plc:alice" ||
		!publisher.params.ScheduledAt.Equal(now) ||
		!bytes.Contains(publisher.params.PayloadBytes, []byte("post now")) {
		t.Fatalf("manual publication params=%+v", publisher.params)
	}
}

func newScheduledPostAPITestStore(t *testing.T) (*scheduledposts.Store, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT NOT NULL PRIMARY KEY,
			record_cid TEXT NOT NULL
		);
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			generation BIGINT NOT NULL,
			state TEXT NOT NULL
		);
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO owner_lifecycles(owner_did,generation,state)
		VALUES ('did:plc:alice',1,'active'), ('did:plc:bob',1,'active');
	`)
	for _, path := range []string{
		"000034_scheduled_posts.up.sql",
		"000040_scheduled_media_durability.up.sql",
		"000048_scheduled_post_owner_generation.up.sql",
	} {
		migration, err := testdb.ReadMigration(path)
		if err != nil {
			t.Fatalf("read scheduled-post migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply scheduled-post migration %s: %v", path, err)
		}
	}
	return scheduledposts.NewStore(pool), pool
}
