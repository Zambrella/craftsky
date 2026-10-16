package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/operatorconsole"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func TestConsoleOperatorCreationReplaysAndUsesTrustedActor(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(body string, keys ...string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/admin/operators", strings.NewReader(body))
		key := "operator-create-fixture-01"
		if len(keys) > 0 {
			key = keys[0]
		}
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	first := post(`{"did":"did:example:browser-added","permissionSet":"mod"}`)
	if first.Code != 201 {
		t.Fatalf("allow=%d %s", first.Code, first.Body)
	}
	replay := post(`{"did":"did:example:browser-added","permissionSet":"mod"}`)
	if replay.Code != 201 || replay.Header().Get("Idempotent-Replayed") != "true" || replay.Body.String() != first.Body.String() {
		t.Fatal("operator creation did not safely replay")
	}
	var created struct {
		ID string `json:"operatorId"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/operators/"+created.ID+"/events", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"actorId": "operator:fixture"`) && !strings.Contains(w.Body.String(), `"actorId":"operator:fixture"`) {
		t.Fatalf("audit attribution=%d %s", w.Code, w.Body)
	}
	operations := httptest.NewRecorder()
	mux.ServeHTTP(operations, httptest.NewRequest("GET", "/v1/admin/operations?limit=10", nil))
	var page struct {
		Items []struct {
			ID         string `json:"id"`
			ResourceID string `json:"resourceId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(operations.Body.Bytes(), &page); err != nil || len(page.Items) != 1 {
		t.Fatalf("operations: %s", operations.Body)
	}
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest("GET", "/v1/admin/operations/"+page.Items[0].ID, nil))
	if detail.Code != 200 || page.Items[0].ResourceID != created.ID {
		t.Fatalf("operation detail=%d resource=%s", detail.Code, page.Items[0].ResourceID)
	}
	forged := post(`{"did":"did:example:forged","permissionSet":"admin","actorId":"forged"}`, "operator-forged-fresh-key")
	if forged.Code != 400 {
		t.Fatal("accepted forged actor")
	}
}

func TestConsoleIncidentAssignmentsAreRevisedAndAudited(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := operatorconsole.NewStore(pool, time.Now)
	op, err := store.Allow(t.Context(), syntax.DID("did:example:incident-owner"), "mod")
	if err != nil {
		t.Fatal(err)
	}
	incident, scan := uuid.New(), uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'synthetic','fixture','fixture','fixture','match')`, scan); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'synthetic','synthetic',now(),now())`, incident, scan); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	deps.SafetyIncidents = safetyincident.NewStore(pool, time.Now)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	path := "/v1/admin/safety/incidents/" + incident.String() + "/assignments"
	r := httptest.NewRequest("POST", path, strings.NewReader(`{"expectedRevision":0,"operatorId":"`+op.ID.String()+`","role":"owner"}`))
	r.Header.Set("Idempotency-Key", "assignment-owner-001")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("assignment=%d %s", w.Code, w.Body)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest("GET", "/v1/admin/safety/incidents/"+incident.String(), nil))
	if detail.Code != 200 || !strings.Contains(detail.Body.String(), op.ActorID) {
		t.Fatalf("owner not visible: %d %s", detail.Code, detail.Body)
	}
	r = httptest.NewRequest("POST", path+"/"+result.ID+"/release", strings.NewReader(`{"expectedRevision":1}`))
	r.Header.Set("Idempotency-Key", "assignment-release-001")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("release=%d %s", w.Code, w.Body)
	}
	audit := httptest.NewRecorder()
	mux.ServeHTTP(audit, httptest.NewRequest("GET", "/v1/admin/operators/"+op.ID.String()+"/events", nil))
	if audit.Code != 200 || !strings.Contains(audit.Body.String(), "assignmentGranted") || !strings.Contains(audit.Body.String(), "assignmentReleased") || !strings.Contains(audit.Body.String(), "operator:fixture") {
		t.Fatalf("assignment audit=%d %s", audit.Code, audit.Body)
	}
}

func TestConsoleOperatorRegistryIsAdminOnlyAndCurated(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	op, err := operatorconsole.NewStore(pool, time.Now).Allow(t.Context(), syntax.DID("did:example:registry-fixture"), "helper")
	if err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	for _, role := range []safetyincident.Role{safetyincident.RoleSafetyAdministrator, safetyincident.RoleModerator, safetyincident.RoleHelper} {
		deps.OperatorAuthentication = consoleTestAuthentication(role)
		mux := http.NewServeMux()
		AddRoutes(t.Context(), mux, deps)
		for _, path := range []string{"/v1/admin/operators?limit=1", "/v1/admin/operators/" + op.ID.String(), "/v1/admin/operators/" + op.ID.String() + "/events"} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			want := 403
			if role == safetyincident.RoleSafetyAdministrator {
				want = 200
			}
			if w.Code != want {
				t.Fatalf("%s %s=%d %s", role, path, w.Code, w.Body)
			}
			if want == 200 && (w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "token") || strings.Contains(w.Body.String(), "cookie")) {
				t.Fatalf("unsafe operator metadata: %s", w.Body)
			}
		}
	}
}

func TestConsoleOperatorChangesRequireFreshReviewAndPreserveLastAdmin(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := operatorconsole.NewStore(pool, time.Now)
	admin, err := store.Allow(t.Context(), syntax.DID("did:example:last-admin"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	helper, err := store.Allow(t.Context(), syntax.DID("did:example:change-helper"), "helper")
	if err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(id, action, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/v1/admin/operators/"+id+"/"+action, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	changed := post(helper.ID.String(), "permission-set", "operator-change-001", `{"expectedRevision":0,"permissionSet":"mod"}`)
	if changed.Code != 200 {
		t.Fatalf("set=%d %s", changed.Code, changed.Body)
	}
	stale := post(helper.ID.String(), "disable", "operator-disable-stale", `{"expectedRevision":0}`)
	if stale.Code != 409 {
		t.Fatalf("stale=%d %s", stale.Code, stale.Body)
	}
	disabled := post(helper.ID.String(), "disable", "operator-disable-001", `{"expectedRevision":1}`)
	if disabled.Code != 200 {
		t.Fatalf("disable=%d %s", disabled.Code, disabled.Body)
	}
	if _, err := store.Lookup(t.Context(), helper.DID); err == nil {
		t.Fatal("disabled operator admitted")
	}
	last := post(admin.ID.String(), "disable", "operator-last-disable", `{"expectedRevision":0}`)
	if last.Code != 422 {
		t.Fatalf("last admin=%d %s", last.Code, last.Body)
	}
	demote := post(admin.ID.String(), "permission-set", "operator-last-demote", `{"expectedRevision":0,"permissionSet":"mod"}`)
	if demote.Code != 422 {
		t.Fatalf("last admin demotion=%d %s", demote.Code, demote.Body)
	}
	revoke := post(admin.ID.String(), "session-revocations", "operator-revoke-001", `{"expectedRevision":0}`)
	if revoke.Code != 200 {
		t.Fatalf("revoke=%d %s", revoke.Code, revoke.Body)
	}
}

func TestConsoleConcurrentAdminDeactivationKeepsOneUsableAdmin(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := operatorconsole.NewStore(pool, time.Now)
	first, err := store.Allow(t.Context(), syntax.DID("did:example:concurrent-first"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Allow(t.Context(), syntax.DID("did:example:concurrent-second"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	results := make(chan int, 2)
	for _, op := range []operatorconsole.Operator{first, second} {
		go func() {
			r := httptest.NewRequest("POST", "/v1/admin/operators/"+op.ID.String()+"/disable", strings.NewReader(`{"expectedRevision":0}`))
			r.Header.Set("Idempotency-Key", "concurrent-disable-"+op.ID.String())
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			results <- w.Code
		}()
	}
	codes := map[int]int{}
	codes[<-results]++
	codes[<-results]++
	if codes[200] != 1 || codes[422] != 1 {
		t.Fatalf("concurrent outcomes=%v", codes)
	}
	admitted := 0
	for _, op := range []operatorconsole.Operator{first, second} {
		if _, err := store.Lookup(t.Context(), op.DID); err == nil {
			admitted++
		}
	}
	if admitted != 1 {
		t.Fatalf("usable admins=%d", admitted)
	}
}

func TestConsoleManagementMutationsRejectModeratorAndHelper(t *testing.T) {
	for _, role := range []safetyincident.Role{safetyincident.RoleModerator, safetyincident.RoleHelper} {
		deps := testDeps()
		deps.Config.ModerationAdminEnabled = true
		deps.OperatorAuthentication = consoleTestAuthentication(role)
		mux := http.NewServeMux()
		AddRoutes(t.Context(), mux, deps)
		for _, path := range []string{"/v1/admin/operators", "/v1/admin/operators/64000000-0000-4000-8000-000000000001/permission-set", "/v1/admin/operators/64000000-0000-4000-8000-000000000001/disable", "/v1/admin/operators/64000000-0000-4000-8000-000000000001/session-revocations", "/v1/admin/safety/incidents/64000000-0000-4000-8000-000000000001/assignments", "/v1/admin/safety/incidents/64000000-0000-4000-8000-000000000001/assignments/64000000-0000-4000-8000-000000000002/release"} {
			r := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
			r.Header.Set("Idempotency-Key", "denied-management-fixture")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatalf("%s %s=%d", role, path, w.Code)
			}
		}
	}
}
