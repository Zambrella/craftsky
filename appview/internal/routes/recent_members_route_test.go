package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

// IT-005 / RULE-003 / AC-011: real mux, standard authentication and access.
func TestRecentMembersRouteAuthentication(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	if _, err := pool.Exec(t.Context(), `
 INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:test','viewer'),('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','member');
 INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:test','active',1,1,'test',now()),('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','active',1,1,'test',now());
 INSERT INTO account_onboarding_completions(account_did) VALUES('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa');
 `); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.ProfileStore = api.NewProfileStore(pool)
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	for _, tc := range []struct {
		name         string
		auth, device bool
		status       int
	}{{"missing auth", false, true, 401}, {"missing device", true, false, 400}, {"authenticated", true, true, 200}} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/profiles/recent", nil)
			if tc.auth {
				req.Header.Set("Authorization", "Bearer recent-test")
			}
			if tc.device {
				req.Header.Set("X-Craftsky-Device-Id", "recent-device")
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.status == 200 {
				items, ok := body["items"].([]any)
				if !ok || len(items) != 1 {
					t.Fatalf("static route dispatch: %v", body)
				}
				if rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("viewer-specific response cacheable")
				}
			} else if body["error"] != map[bool]string{true: "unauthorized", false: "missing_device_id"}[tc.device] || body["message"] == nil || body["requestId"] == nil {
				t.Fatalf("error envelope: %v", body)
			}
		})
	}
	policy := mustPolicy("GET", "/v1/profiles/recent")
	if policy.AccessClass != AccessCurrentMember || policy.RateClass != RateClassRead || policy.BodyKind != BodyNoBody {
		t.Fatalf("policy: %+v", policy)
	}
}

// IT-006 / RULE-003, FR-005 / AC-005, AC-011: allowlist after all decorators.
func TestRecentMembersRouteSerializedContract(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	if _, err := pool.Exec(t.Context(), `
 INSERT INTO craftsky_profiles(did,record_cid,crafts) VALUES('did:plc:test','viewer',ARRAY['private-viewer-crafts']),('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','member',ARRAY['knitting']),('did:plc:bbbbbbbbbbbbbbbbbbbbbbbb','member',ARRAY[]::text[]);
 INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:test','active',1,1,'test',now()),('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','active',1,1,'test',now()),('did:plc:bbbbbbbbbbbbbbbbbbbbbbbb','active',1,1,'test',now());
 INSERT INTO account_onboarding_completions(account_did,completed_at) VALUES('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','2026-10-03T00:00:00Z'),('did:plc:bbbbbbbbbbbbbbbbbbbbbbbb','2026-10-02T00:00:00Z');
 INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','alice.test','alice.test',now());
 INSERT INTO bluesky_profiles(did,record_cid,display_name,description,avatar_cid,avatar_mime) VALUES('did:plc:aaaaaaaaaaaaaaaaaaaaaaaa','bsky','Alice','private-description-canary','avatar-cid','image/jpeg'),('did:plc:bbbbbbbbbbbbbbbbbbbbbbbb','bsky',E'\t\n','private-other-description',NULL,NULL);
 `); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.ProfileStore = api.NewProfileStore(pool)
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/profiles/recent", nil)
		req.Header.Set("Authorization", "Bearer private-token-canary")
		req.Header.Set("X-Craftsky-Device-Id", "private-device-canary")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	rec := request()
	if rec.Code != 200 {
		t.Fatalf("success: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 {
		t.Fatalf("page keys: %v", body)
	}
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items: %v", items)
	}
	for i, raw := range items {
		item := raw.(map[string]any)
		wanted := 4
		if i == 0 {
			wanted = 6
		}
		if len(item) != wanted {
			t.Fatalf("item key count %d, want %d: %v", len(item), wanted, item)
		}
		for key := range item {
			if key != "did" && key != "handle" && key != "isCraftskyProfile" && key != "displayName" && key != "avatar" && key != "crafts" {
				t.Fatalf("unexpected summary field %s", key)
			}
		}
		if item["isCraftskyProfile"] != true {
			t.Fatal("missing membership identity")
		}
	}
	first := items[0].(map[string]any)
	firstCrafts, _ := first["crafts"].([]any)
	secondCrafts, _ := items[1].(map[string]any)["crafts"].([]any)
	if len(firstCrafts) != 1 || firstCrafts[0] != "knitting" || len(secondCrafts) != 0 {
		t.Fatalf("discovery crafts: %v / %v", firstCrafts, secondCrafts)
	}

	if first["did"] != "did:plc:aaaaaaaaaaaaaaaaaaaaaaaa" || first["handle"] != "alice.test" || first["displayName"] != "Alice" || first["avatar"] != "https://cdn.bsky.app/img/avatar/plain/did:plc:aaaaaaaaaaaaaaaaaaaaaaaa/avatar-cid@jpeg" {
		t.Fatalf("public presentation: %v", first)
	}
	if items[1].(map[string]any)["handle"] != "handle.invalid" {
		t.Fatal("missing identity cache fallback")
	}
	assertPrivate := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		for _, canary := range []string{"private-token-canary", "private-device-canary", "private-viewer-crafts", "private-description-canary", "2026-10-03"} {
			if strings.Contains(rec.Body.String(), canary) {
				t.Fatalf("private value leaked: %s", canary)
			}
		}
	}
	assertPrivate(rec)
	if _, err := pool.Exec(t.Context(), `DELETE FROM account_onboarding_completions`); err != nil {
		t.Fatal(err)
	}
	empty := request()
	if empty.Code != 200 || strings.TrimSpace(empty.Body.String()) != `{"items":[]}` {
		t.Fatalf("empty: %d %s", empty.Code, empty.Body.String())
	}
	if _, err := pool.Exec(t.Context(), `ALTER TABLE account_onboarding_completions DROP COLUMN completed_at`); err != nil {
		t.Fatal(err)
	}
	failed := request()
	if failed.Code != 500 {
		t.Fatalf("failure: %d %s", failed.Code, failed.Body.String())
	}
	var failure map[string]any
	if err := json.Unmarshal(failed.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure) != 3 || failure["error"] != "internal_error" || failure["message"] != "Unable to load recent members" || failure["requestId"] == nil {
		t.Fatalf("failure envelope: %v", failure)
	}
	assertPrivate(failed)
}
