package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

func TestRegisterNotificationDeviceIsIdempotentAndRotatesTokenWithoutEcho(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	handler := api.RegisterNotificationDeviceHandler(api.NewPostStore(pool), nilLogger())
	register := func(token string) api.NotificationDeviceResponse {
		t.Helper()
		req := authedReq(http.MethodPost, "/v1/notifications/devices", `{"platform":"ios","token":"`+token+`"}`, "did:plc:viewer")
		req = req.WithContext(middleware.WithDeviceID(req.Context(), "device-1"))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), token) {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var response api.NotificationDeviceResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	first := register("token-one")
	second := register("token-one")
	rotated := register("token-two")
	if first.AccountSubscriptionID == "" || second != first || rotated != first {
		t.Fatalf("responses=%+v %+v %+v", first, second, rotated)
	}
	var installations, subscriptions int
	var token string
	if err := pool.QueryRow(context.Background(), `SELECT count(*), max(fcm_token) FROM push_installations`).Scan(&installations, &token); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_account_subscriptions`).Scan(&subscriptions); err != nil {
		t.Fatal(err)
	}
	if installations != 1 || subscriptions != 1 || token != "token-two" {
		t.Fatalf("installations=%d subscriptions=%d token=%s", installations, subscriptions, token)
	}
}

func TestRegisterNotificationDeviceRebindsTokenWithoutTransferringAccounts(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	handler := api.RegisterNotificationDeviceHandler(api.NewPostStore(pool), nilLogger())
	register := func(did, deviceID string) api.NotificationDeviceResponse {
		t.Helper()
		req := authedReq(http.MethodPost, "/v1/notifications/devices", `{"platform":"ios","token":"shared-token"}`, did)
		req = req.WithContext(middleware.WithDeviceID(req.Context(), deviceID))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var response api.NotificationDeviceResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	aliceOld := register("did:plc:alice", "old-device")
	bobOld := register("did:plc:bob", "old-device")
	if aliceOld == bobOld {
		t.Fatal("shared installation accounts must have different routing IDs")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key, source_uri, source_cid, source_rkey,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot, state,
			first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
		) VALUES ('00000000-0000-0000-0000-000000000009','did:plc:alice','did:plc:actor','follow','alice','source','cid','r','everyone',false,true,'active',now(),now(),now(),now());
		INSERT INTO push_deliveries (id, notification_id, account_subscription_id, status, next_attempt_at, deadline_at)
		SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000009', id, 'pending', now(), now()+interval '6 hours'
		FROM push_account_subscriptions
	`); err != nil {
		t.Fatal(err)
	}

	aliceNew := register("did:plc:alice", "new-device")
	if aliceNew.AccountSubscriptionID == aliceOld.AccountSubscriptionID {
		t.Fatal("cross-device rebind transferred old routing ID")
	}
	var oldActive, oldSubsActive, oldPending, newSubs int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_installations WHERE device_id='old-device' AND active`).Scan(&oldActive); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE i.device_id='old-device' AND s.active`).Scan(&oldSubsActive); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_deliveries WHERE status IN ('pending','retry','leased')`).Scan(&oldPending); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE i.device_id='new-device' AND s.active`).Scan(&newSubs); err != nil {
		t.Fatal(err)
	}
	if oldActive != 0 || oldSubsActive != 0 || oldPending != 0 || newSubs != 1 {
		t.Fatalf("oldActive=%d oldSubs=%d oldPending=%d newSubs=%d", oldActive, oldSubsActive, oldPending, newSubs)
	}
}

func TestRemoveNotificationSubscriptionOnlyRemovesOwnedAccountOnCurrentDevice(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	store := api.NewPostStore(pool)
	aliceRouting, err := store.RegisterNotificationDevice(context.Background(), "did:plc:alice", "shared-device", "android", "shared-token")
	if err != nil {
		t.Fatal(err)
	}
	bobRouting, err := store.RegisterNotificationDevice(context.Background(), "did:plc:bob", "shared-device", "android", "shared-token")
	if err != nil {
		t.Fatal(err)
	}

	handler := api.RemoveNotificationDeviceHandler(store, nilLogger())
	req := authedReq(http.MethodDelete, "/v1/notifications/devices/"+aliceRouting, "", "did:plc:alice")
	req.SetPathValue("accountSubscriptionId", aliceRouting)
	req = req.WithContext(middleware.WithDeviceID(req.Context(), "shared-device"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var aliceActive, bobActive int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_account_subscriptions WHERE routing_id=$1 AND active`, aliceRouting).Scan(&aliceActive); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_account_subscriptions WHERE routing_id=$1 AND active`, bobRouting).Scan(&bobActive); err != nil {
		t.Fatal(err)
	}
	if aliceActive != 0 || bobActive != 1 {
		t.Fatalf("aliceActive=%d bobActive=%d", aliceActive, bobActive)
	}
}
