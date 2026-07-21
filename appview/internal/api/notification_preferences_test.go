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
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationPreferencesHandlersReturnDefaultsAndPatchSubset(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	store := api.NewPostStore(pool)

	get := api.GetNotificationPreferencesHandler(store, nilLogger())
	recorder := httptest.NewRecorder()
	get.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications/preferences", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var defaults api.NotificationPreferencesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &defaults); err != nil {
		t.Fatal(err)
	}
	if len(defaults.Preferences) != 8 || defaults.Preferences[notifications.Like].Scope != notifications.Everyone || !defaults.Preferences[notifications.Like].PushEnabled || defaults.Preferences[notifications.InstagramMatch] != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: true}) {
		t.Fatalf("defaults=%+v", defaults)
	}

	patch := api.PatchNotificationPreferencesHandler(store, nilLogger())
	recorder = httptest.NewRecorder()
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"like":{"scope":"peopleIFollow","pushEnabled":false}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("PATCH status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var updated api.NotificationPreferencesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Preferences[notifications.Like] != (notifications.Preference{Scope: notifications.PeopleIFollow, PushEnabled: false}) || updated.Preferences[notifications.Follow] != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: true}) {
		t.Fatalf("updated=%+v", updated)
	}

	recorder = httptest.NewRecorder()
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"unknown":{"pushEnabled":false}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusBadRequest || strings.Contains(recorder.Body.String(), "did:plc:viewer") {
		t.Fatalf("invalid status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestInstagramMatchPreferenceAPIAllowsPushOnlyAndPersistsFixedScope(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
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

	store := api.NewPostStore(pool)
	get := api.GetNotificationPreferencesHandler(store, nilLogger())
	recorder := httptest.NewRecorder()
	get.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications/preferences", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var defaults api.NotificationPreferencesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &defaults); err != nil {
		t.Fatal(err)
	}
	if len(defaults.Preferences) != 8 || defaults.Preferences[notifications.InstagramMatch] != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: true}) {
		t.Fatalf("instagramMatch defaults=%+v", defaults.Preferences)
	}

	patch := api.PatchNotificationPreferencesHandler(store, nilLogger())
	recorder = httptest.NewRecorder()
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"instagramMatch":{"pushEnabled":false}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("push PATCH status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var updated api.NotificationPreferencesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Preferences[notifications.InstagramMatch] != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: false}) {
		t.Fatalf("updated instagramMatch=%+v", updated.Preferences[notifications.InstagramMatch])
	}

	for _, scope := range []string{"everyone", "peopleIFollow"} {
		recorder = httptest.NewRecorder()
		body := `{"preferences":{"instagramMatch":{"scope":"` + scope + `","pushEnabled":true}}}`
		patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", body, "did:plc:viewer"))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("scope %s PATCH status=%d body=%s", scope, recorder.Code, recorder.Body.String())
		}
	}

	var scope notifications.Scope
	var pushEnabled bool
	if err := pool.QueryRow(context.Background(), `
		SELECT scope, push_enabled
		FROM notification_preferences
		WHERE account_did = 'did:plc:viewer' AND category = 'instagramMatch'
	`).Scan(&scope, &pushEnabled); err != nil {
		t.Fatal(err)
	}
	if scope != notifications.Everyone || pushEnabled {
		t.Fatalf("persisted instagramMatch scope=%q push=%t", scope, pushEnabled)
	}

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_preferences (account_did, category, scope, push_enabled)
		VALUES ('did:plc:direct-write', 'instagramMatch', 'peopleIFollow', true)
	`); err == nil {
		t.Fatal("database accepted actor scope for instagramMatch")
	}
}
