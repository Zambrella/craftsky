package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestNotificationPreferencesHandlersReturnDefaultsAndPatchSubset(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyNotificationPreferencesTestSchema(t, pool)
	seedMember(t, pool, "did:plc:viewer")
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
	if len(defaults.Preferences) != 9 || defaults.Preferences[notifications.Like].Scope != notifications.Everyone || !defaults.Preferences[notifications.Like].PushEnabled {
		t.Fatalf("defaults=%+v", defaults)
	}
	if got := defaults.Preferences[notifications.InstagramMatch]; got != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: true}) {
		t.Fatalf("instagramMatch default=%+v", got)
	}
	if got := defaults.Preferences[notifications.Moderation]; got != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: true}) {
		t.Fatalf("moderation default=%+v", got)
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
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"moderation":{"pushEnabled":false}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("moderation PATCH status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	get.ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications/preferences", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("reloaded GET status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var reloaded api.NotificationPreferencesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.Preferences[notifications.Moderation] != (notifications.Preference{Scope: notifications.Everyone, PushEnabled: false}) {
		t.Fatalf("reloaded moderation=%+v", reloaded.Preferences[notifications.Moderation])
	}
	if reloaded.Preferences[notifications.Like] != updated.Preferences[notifications.Like] || reloaded.Preferences[notifications.Follow] != updated.Preferences[notifications.Follow] {
		t.Fatalf("moderation patch changed existing preferences: %+v", reloaded)
	}

	recorder = httptest.NewRecorder()
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"unknown":{"pushEnabled":false}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusBadRequest || strings.Contains(recorder.Body.String(), "did:plc:viewer") {
		t.Fatalf("invalid status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestConcurrentFirstNotificationPreferencePatchesMerge(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyNotificationPreferencesTestSchema(t, pool)
	ctx, cancel := context.WithTimeout(ownerlifecycle.WithExpectedGeneration(context.Background(), 1), 5*time.Second)
	defer cancel()
	const did = "did:plc:viewer"
	seedMember(t, pool, did)
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(context.Background())
	// Stop inserts after the first request resolves defaults. Before the
	// fix, both requests reach this lock with a stale copy of the defaults.
	if _, err := blocker.Exec(ctx, `LOCK TABLE notification_preferences IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	store := api.NewPostStore(pool)
	scope, enabled := notifications.PeopleIFollow, false
	results := make(chan error, 2)
	for _, patch := range []notifications.PreferencePatch{{Scope: &scope}, {PushEnabled: &enabled}} {
		go func() {
			_, err := store.PatchNotificationPreferences(ctx, did, map[notifications.Category]notifications.PreferencePatch{notifications.Like: patch})
			results <- err
		}()
	}
	for {
		var waiting int
		err := pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_locks
			WHERE NOT granted AND (
				(relation = 'notification_preferences'::regclass AND mode = 'RowExclusiveLock')
				OR (locktype = 'advisory' AND classid = hashtext('notification_preferences')::oid AND objid = hashtext($1)::oid AND objsubid = 2)
			)
		`, did).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting == 2 {
			break
		}
		runtime.Gosched()
	}
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	preferences, err := store.NotificationPreferences(ctx, did)
	if want := (notifications.Preference{Scope: scope, PushEnabled: enabled}); err != nil || preferences[notifications.Like] != want {
		t.Fatalf("like preference=%+v err=%v, want %+v", preferences[notifications.Like], err, want)
	}
}

func TestInstagramMatchPreferenceAPIRejectsScopePatch(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	applyNotificationPreferencesTestSchema(t, pool)
	seedMember(t, pool, "did:plc:viewer")

	patch := api.PatchNotificationPreferencesHandler(api.NewPostStore(pool), nilLogger())
	recorder := httptest.NewRecorder()
	patch.ServeHTTP(recorder, authedReq(http.MethodPatch, "/v1/notifications/preferences", `{"preferences":{"instagramMatch":{"scope":"peopleIFollow"}}}`, "did:plc:viewer"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("scope PATCH status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func applyNotificationPreferencesTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := testdb.ReadMigration("000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `ALTER TABLE notification_preferences
		DROP CONSTRAINT IF EXISTS notification_preferences_instagram_match_scope_check,
		DROP CONSTRAINT IF EXISTS notification_preferences_category_check,
		ADD CONSTRAINT notification_preferences_category_check CHECK (category IN (
			'like','follow','reply','mention','quote','repost','everythingElse','instagramMatch','moderation'
		)),
		ADD CONSTRAINT notification_preferences_fixed_scope_check CHECK (
			category NOT IN ('instagramMatch','moderation') OR scope='everyone'
		)`); err != nil {
		t.Fatal(err)
	}
}
