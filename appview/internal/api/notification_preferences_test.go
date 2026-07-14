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
	if len(defaults.Preferences) != 7 || defaults.Preferences[notifications.Like].Scope != notifications.Everyone || !defaults.Preferences[notifications.Like].PushEnabled {
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
