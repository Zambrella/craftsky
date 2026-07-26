package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestNotificationNewnessAccountWideAcrossDevicesAndIsolatedByAccount(t *testing.T) {
	pool := testdb.WithSchema(t, routeModerationDDL)
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
		CREATE TABLE actor_mutes (
			owner_did TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			PRIMARY KEY (owner_did, subject_did)
		);
		CREATE TABLE atproto_blocks (
			uri TEXT PRIMARY KEY,
			blocker_did TEXT NOT NULL,
			subject_did TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create relationship tables: %v", err)
	}

	insert := func(id, recipient string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_events (
				id, recipient_did, actor_did, category, subject_key,
				source_uri, source_cid, source_rkey,
				eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
				state, first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
			) VALUES (
				$1::uuid, $2, 'did:plc:actor', 'follow', $1::uuid::text,
				'at://did:plc:actor/app.bsky.graph.follow/' || $1::uuid::text, 'cid', $1::uuid::text,
				'everyone', false, true, 'active', now(), now(), now(), now()
			)
		`, id, recipient); err != nil {
			t.Fatalf("insert notification: %v", err)
		}
	}
	insert("20000000-0000-0000-0000-000000000001", "did:plc:alice")
	insert("20000000-0000-0000-0000-000000000002", "did:plc:alice")
	insert("20000000-0000-0000-0000-000000000003", "did:plc:bob")

	deps := testDeps()
	deps.DB = pool
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)

	request := func(method, path, did, device string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Dev-DID", did)
		req.Header.Set("X-Craftsky-Device-Id", device)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)
		return recorder
	}
	count := func(did, device string) int64 {
		t.Helper()
		recorder := request(http.MethodGet, "/v1/notifications/new-count", did, device)
		if recorder.Code != http.StatusOK {
			t.Fatalf("count %s/%s status=%d body=%s", did, device, recorder.Code, recorder.Body.String())
		}
		var body struct {
			NewCount int64 `json:"newCount"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.NewCount
	}

	if got := count("did:plc:alice", "device-a"); got != 2 {
		t.Fatalf("Alice device A count=%d, want 2", got)
	}
	if got := count("did:plc:alice", "device-b"); got != 2 {
		t.Fatalf("Alice device B prefetch count=%d, want 2", got)
	}
	seen := request(http.MethodPost, "/v1/notifications/seen", "did:plc:alice", "device-a")
	if seen.Code != http.StatusNoContent || seen.Body.Len() != 0 {
		t.Fatalf("seen status=%d body=%q", seen.Code, seen.Body.String())
	}
	if got := count("did:plc:alice", "device-b"); got != 0 {
		t.Fatalf("Alice device B count after device A acknowledgement=%d, want 0", got)
	}
	if got := count("did:plc:bob", "device-a"); got != 1 {
		t.Fatalf("Bob shared-device count=%d, want 1", got)
	}
}
