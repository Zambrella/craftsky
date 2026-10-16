package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/safetyintake"
	"social.craftsky/appview/internal/testdb"
)

func TestConsoleWorkflowsPaginateWithoutPrivateAuthorityMetadata(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, id := range ids {
		if _, err := pool.Exec(t.Context(), `INSERT INTO safety_workflows(id,workflow_class,state,urgency,responsible_actor,received_at,created_at,updated_at)
		VALUES($1,'authorityRequest','open','immediate','operator:fixture','2030-01-01','2030-01-01','2030-01-01');`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `INSERT INTO safety_authority_requests(workflow_id,requester_reference,verification_state) VALUES($1,'private-reference-canary','pending')`, id); err != nil {
			t.Fatal(err)
		}
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	w := get("/v1/admin/safety/workflows?limit=2")
	if w.Code != 200 {
		t.Fatalf("list status/body = %d/%s", w.Code, w.Body)
	}
	var first struct {
		Items []struct {
			ID string `json:"id"`
		}
		Cursor string `json:"cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Cursor == "" || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "private-reference-canary") {
		t.Fatalf("unsafe or unbounded page: %s", w.Body)
	}
	w = get("/v1/admin/safety/workflows?limit=2&cursor=" + first.Cursor)
	var last struct {
		Items  []struct{ ID string }
		Cursor string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &last); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(last.Items) != 1 || last.Cursor != "" || last.Items[0].ID == first.Items[0].ID || last.Items[0].ID == first.Items[1].ID {
		t.Fatalf("second page: %d/%s", w.Code, w.Body)
	}
	for _, query := range []string{"limit=101", "cursor=invalid", "limit=1&limit=2", "unexpected=x"} {
		if w := get("/v1/admin/safety/workflows?" + query); w.Code != 400 {
			t.Fatalf("%s status=%d", query, w.Code)
		}
	}
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleHelper)
	helper := http.NewServeMux()
	AddRoutes(t.Context(), helper, deps)
	w = httptest.NewRecorder()
	helper.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/workflows", nil))
	if w.Code != 403 {
		t.Fatalf("helper global list=%d", w.Code)
	}
}

func TestConsoleWorkflowCommandsReplayAndRejectStaleReview(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	policy, err := safetyincident.NewDeadlinePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	deps.SafetyWorkflows = safetyincident.NewWorkflowService(pool, policy, time.Now)
	deps.SafetyIntake = safetyintake.NewStore(pool, time.Now)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(path, key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", "console-test-"+key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	body := `{"receivedAt":"2026-10-15T10:00:00Z","preservationCode":"retained"}`
	first := post("/v1/admin/safety/workflows/credible-threats", "threat-create-1", body)
	second := post("/v1/admin/safety/workflows/credible-threats", "threat-create-1", body)
	if first.Code != 201 || second.Code != 201 || first.Body.String() != second.Body.String() || second.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay: %d/%s then %d/%s", first.Code, first.Body, second.Code, second.Body)
	}
	var created struct{ ID string }
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if changed := post("/v1/admin/safety/workflows/credible-threats", "threat-create-1", strings.Replace(body, "retained", "changed", 1)); changed.Code != 409 {
		t.Fatalf("changed replay=%d/%s", changed.Code, changed.Body)
	}
	path := "/v1/admin/safety/workflows/credible-threats/" + created.ID + "/outcome"
	if stale := post(path, "threat-outcome-stale", `{"expectedRevision":99,"outcomeCode":"reviewed"}`); stale.Code != 409 {
		t.Fatalf("stale=%d/%s", stale.Code, stale.Body)
	}
	done := post(path, "threat-outcome-1", `{"expectedRevision":0,"outcomeCode":"reviewed"}`)
	if done.Code != 200 {
		t.Fatalf("outcome=%d/%s", done.Code, done.Body)
	}
	replay := post(path, "threat-outcome-1", `{"expectedRevision":0,"outcomeCode":"reviewed"}`)
	if replay.Code != 200 || replay.Body.String() != done.Body.String() {
		t.Fatalf("outcome replay=%d/%s", replay.Code, replay.Body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/workflows/"+created.ID, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"resolved"`) && !strings.Contains(w.Body.String(), `"state": "resolved"`) {
		t.Fatalf("detail=%d/%s", w.Code, w.Body)
	}
}

func consoleTestAuthentication(role safetyincident.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permissions := map[string]bool{}
			for _, permission := range safetyincident.OperatorPermissions() {
				permissions[string(permission)] = role.AllowsPermission(permission)
			}
			next.ServeHTTP(w, r.WithContext(ctxkeys.WithModerator(r.Context(), ctxkeys.Moderator{ActorID: "operator:fixture", SourceSystem: "operator-console", Role: string(role), Permissions: permissions})))
		})
	}
}

func TestConsoleScanMetadataExcludesCapabilitiesAndLeaseSecrets(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	id := uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'blob-canary','fixture','policy','corpus','error')`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_jobs(id,scan_result_id,state,lease_owner,lease_token,lease_expires_at) VALUES($1,$2,'leased','private-lease-owner-canary',$3,now()+interval '1 minute')`, uuid.New(), id, uuid.New()); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	for _, path := range []string{"/v1/admin/safety/scans", "/v1/admin/safety/scans/" + id.String()} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"scannerId": "fixture"`) && !strings.Contains(w.Body.String(), `"scannerId":"fixture"`) {
			t.Fatalf("read: %d/%s", w.Code, w.Body)
		}
		for _, blocked := range []string{"blob-canary", "leaseToken", "leaseOwner", "private-lease-owner-canary", "objectKey"} {
			if strings.Contains(w.Body.String(), blocked) {
				t.Fatalf("protected metadata %s exposed", blocked)
			}
		}
	}
}

func TestConsoleReadersReturnBoundedEmptyCollectionsAndMissingDetails(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	for _, resource := range []string{"incidents", "external-intakes", "evidence", "holds"} {
		path := "/v1/admin/safety/" + resource
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
			t.Fatalf("%s: %d/%s", resource, w.Code, w.Body)
		}
		if resource != "incidents" {
			w = httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", path+"/"+uuid.NewString(), nil))
			if w.Code != 404 {
				t.Fatalf("missing %s=%d/%s", resource, w.Code, w.Body)
			}
		}
	}
}

func TestConsoleConfirmsProfileDetectionAsAccountCase(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	result := uuid.New()
	now := time.Now().UTC()
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'profile-blob','fixture','policy','corpus','match')`, result); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO profile_image_candidates(profile_did,slot,source_cid,candidate_blob_cid,scan_result_id,declared_mime) VALUES('did:plc:profile','avatar','profile-cid','profile-blob',$1,'image/jpeg')`, result); err != nil {
		t.Fatal(err)
	}
	incidents := safetyincident.NewStore(pool, time.Now)
	for range 2 {
		if err := pgx.BeginFunc(t.Context(), pool, func(tx pgx.Tx) error {
			return incidents.RecordMatchTx(t.Context(), tx, imagesafety.MatchDetection{ResultID: result, ProviderReference: "fixture", IntegrityMetadataReference: "integrity", DetectedAt: now})
		}); err != nil {
			t.Fatalf("record detection: %v", err)
		}
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	deps.SafetyIncidents = incidents
	deps.ImageSafetyHealth = imagesafety.NewWorkerStore(pool, time.Now)
	deps.SafetyWork = []api.SafetyWorkReader{api.NewConsoleWorkReader(pool, time.Now)}
	deps.ModerationCases = moderation.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/incidents", nil))
	var page struct {
		Items []struct {
			ID       string
			Revision int64
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(page.Items) != 1 {
		t.Fatalf("detection list=%d/%s", w.Code, w.Body)
	}
	id := page.Items[0].ID
	r := httptest.NewRequest("POST", "/v1/admin/safety/incidents/"+id+"/confirmations", strings.NewReader(`{"expectedRevision":0,"subjectUri":"at://did:plc:profile/app.bsky.actor.profile/self"}`))
	r.Header.Set("Idempotency-Key", "profile-confirmation-1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	var confirmed struct {
		CaseReference string `json:"caseReference"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &confirmed); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || confirmed.CaseReference == "" {
		t.Fatalf("confirmation=%d/%s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/"+confirmed.CaseReference, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"subjectType":"account"`) {
		t.Fatalf("profile case=%d/%s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/status", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), id) || !strings.Contains(w.Body.String(), "caseCreated") {
		t.Fatalf("active work disappeared=%d/%s", w.Code, w.Body)
	}

}

func TestConsoleRetryAndHoldReleaseRequireCurrentReviewAndTrustedActor(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	scan, incident, evidence, hold := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'retry-blob','fixture','policy','corpus','error')`, []any{scan}},
		{`INSERT INTO image_scan_jobs(id,scan_result_id,state,attempts,safe_error_category) VALUES($1,$2,'dead_letter',5,'scanner_unavailable')`, []any{uuid.New(), scan}},
		{`INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'provider','integrity',now(),now())`, []any{incident, scan}},
		{`INSERT INTO safety_evidence(id,incident_id,object_key,integrity_sha256,content_type,byte_size,preservation_reason,created_by,retention_expires_at,created_at) VALUES($1,$2,'private-object-key-canary',decode(repeat('00',32),'hex'),'image/png',8,'retained','operator:fixture',now()+interval '1 day',now())`, []any{evidence, incident}},
		{`INSERT INTO safety_legal_holds(id,incident_id,basis,approved_by,created_by,expires_at,created_at) VALUES($1,$2,'review','operator:fixture','operator:fixture',now()+interval '1 day',now())`, []any{hold, incident}},
		{`INSERT INTO safety_legal_hold_evidence(hold_id,evidence_id,created_at) VALUES($1,$2,now())`, []any{hold, evidence}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(t.Context(), statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(path, key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	retryPath := "/v1/admin/safety/scans/" + scan.String() + "/retries"
	body := `{"expectedRevision":0,"expectedVersion":1}`
	retried := post(retryPath, "scan-retry-command-1", body)
	if retried.Code != 200 {
		t.Fatalf("retry=%d/%s", retried.Code, retried.Body)
	}
	replay := post(retryPath, "scan-retry-command-1", body)
	if replay.Code != 200 || replay.Body.String() != retried.Body.String() {
		t.Fatalf("retry replay=%d/%s", replay.Code, replay.Body)
	}
	releasePath := "/v1/admin/safety/holds/" + hold.String() + "/releases"
	released := post(releasePath, "hold-release-command-1", `{"expectedRevision":0,"reason":"review completed"}`)
	if released.Code != 200 {
		t.Fatalf("release=%d/%s", released.Code, released.Body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/evidence/"+evidence.String(), nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"held":false`) && !strings.Contains(w.Body.String(), `"held": false`) || strings.Contains(w.Body.String(), "private-object-key-canary") {
		t.Fatalf("retention metadata=%d/%s", w.Code, w.Body)
	}
	if stale := post(releasePath, "hold-release-command-2", `{"expectedRevision":0,"reason":"review completed"}`); stale.Code != 409 {
		t.Fatalf("stale release=%d/%s", stale.Code, stale.Body)
	}
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mod := http.NewServeMux()
	AddRoutes(t.Context(), mod, deps)
	w = httptest.NewRecorder()
	r := httptest.NewRequest("POST", releasePath, strings.NewReader(`{}`))
	mod.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("mod release=%d", w.Code)
	}
}

func TestConsoleEvidencePreservationNeverReturnsPrivateObjectKey(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	scan, incident := uuid.New(), uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'evidence-blob','fixture','policy','corpus','match')`, scan); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'provider','integrity',now(),now())`, incident, scan); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	deps.SafetyEvidence = safetyincident.NewEvidenceService(pool, safetyincident.NewMemoryEvidenceStore(), time.Now)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	fixtureBytes := benignEvidencePNG(t)
	encoded, _ := json.Marshal(map[string]any{"expectedRevision": 0, "bytes": fixtureBytes, "contentType": "image/png", "reason": "review", "expiresAt": "2031-01-01T00:00:00Z"})
	body := string(encoded)
	var prior string
	for range 2 {
		r := httptest.NewRequest("POST", "/v1/admin/safety/incidents/"+incident.String()+"/evidence", strings.NewReader(body))
		r.Header.Set("Idempotency-Key", "preserve-evidence-key-1")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 201 || strings.Contains(w.Body.String(), "objectKey") || strings.Contains(w.Body.String(), "bytes") || strings.Contains(w.Body.String(), base64.StdEncoding.EncodeToString(fixtureBytes)) {
			t.Fatalf("unsafe preservation=%d/%s", w.Code, w.Body)
		}
		if prior != "" && prior != w.Body.String() {
			t.Fatalf("preservation replay changed")
		}
		prior = w.Body.String()
	}
}

func TestConsoleOperatorCaseHasTrustedProvenanceAndSafePreview(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/admin/moderation/cases", strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	body := `{"subjectType":"account","subjectDid":"did:plc:caseowner","reason":"operator review"}`
	created := post("operator-created-case-1", body)
	if created.Code != 201 {
		t.Fatalf("create=%d/%s", created.Code, created.Body)
	}
	var result struct {
		CaseReference string `json:"caseReference"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "/preview", "/provenance"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/"+result.CaseReference+suffix, nil))
		if w.Code != 200 {
			t.Fatalf("%s=%d/%s", suffix, w.Code, w.Body)
		}
		if suffix == "" && !strings.Contains(w.Body.String(), `"origin":"operatorCreated"`) {
			t.Fatalf("origin=%s", w.Body)
		}
		if suffix == "/provenance" && !strings.Contains(w.Body.String(), "operator:fixture") {
			t.Fatalf("actor=%s", w.Body)
		}
	}
	forged := post("operator-created-case-2", strings.Replace(body, `"reason":`, `"actorId":"forged","reason":`, 1))
	if forged.Code != 400 {
		t.Fatalf("forged actor=%d/%s", forged.Code, forged.Body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/accounts/did:plc:caseowner/moderation/history", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), result.CaseReference) {
		t.Fatalf("account history=%d/%s", w.Code, w.Body)
	}
}

func TestConsoleConcurrentCreatesHaveOneDurableOutcome(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	policy, err := safetyincident.NewDeadlinePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	deps.SafetyWorkflows = safetyincident.NewWorkflowService(pool, policy, time.Now)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	responses := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		go func() {
			r := httptest.NewRequest("POST", "/v1/admin/safety/workflows/credible-threats", strings.NewReader(`{"receivedAt":"2026-10-15T10:00:00Z","preservationCode":"retained"}`))
			r.Header.Set("Idempotency-Key", "concurrent-threat-create-1")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			responses <- w
		}()
	}
	prior := ""
	for range 8 {
		w := <-responses
		if w.Code != 201 || prior != "" && prior != w.Body.String() {
			t.Fatalf("concurrent outcome=%d/%s", w.Code, w.Body)
		}
		prior = w.Body.String()
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/workflows", nil))
	var page struct{ Items []json.RawMessage }
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("created %d workflows", len(page.Items))
	}
}

func TestConsoleExternalCaseRetainsIntakeAndTrustedAttribution(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(path, key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	intake := post("/v1/admin/safety/external-intakes", "external-intake-create-1", `{"providerMessageReference":"protected-message-canary","receivedAt":"2026-10-15T10:00:00Z","canonicalSubject":"at://did:plc:external/app.bsky.actor.profile/self","kind":"complaint","urgency":"priority","attachmentStatus":"none"}`)
	if intake.Code != 201 {
		t.Fatalf("intake=%d/%s", intake.Code, intake.Body)
	}
	var accepted struct{ ID string }
	if err := json.Unmarshal(intake.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	body := `{"subjectType":"account","subjectDid":"did:plc:external","reason":"external complaint review","intakeId":"` + accepted.ID + `"}`
	created := post("/v1/admin/moderation/cases", "external-case-create-1", body)
	if created.Code != 201 {
		t.Fatalf("external case=%d/%s", created.Code, created.Body)
	}
	var result struct {
		CaseReference string `json:"caseReference"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/"+result.CaseReference+"/provenance", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), accepted.ID) || !strings.Contains(w.Body.String(), "operator:fixture") || strings.Contains(w.Body.String(), "protected-message-canary") {
		t.Fatalf("provenance=%d/%s", w.Code, w.Body)
	}
	forged := post("/v1/admin/moderation/cases", "external-case-forged-1", strings.Replace(body, "did:plc:external", "did:plc:unrelated", 1))
	if forged.Code != 422 {
		t.Fatalf("unrelated intake subject=%d/%s", forged.Code, forged.Body)
	}
	for _, body := range []string{`{"expectedRevision":0,"expectedRevision":1}`, `{"expectedRevision":0,"actorId":"forged"}`, `{"expectedRevision":0} {}`} {
		w := post("/v1/admin/safety/external-intakes/"+accepted.ID+"/correspondence", uuid.NewString(), body)
		if w.Code != 400 {
			t.Fatalf("malformed command=%d/%s", w.Code, w.Body)
		}
	}
}

func TestConsoleEligibilityReviewsRejectStaleChangesAndReplay(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	post := func(key, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/admin/eligibility/reviews", strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	body := `{"expectedRevision":0,"accountDid":"did:plc:eligibility","state":"restricted","evidenceKind":"self_disclosure","evidenceReference":"protected-evidence-canary","reason":"reviewed disclosure","appealGuidance":"Contact support"}`
	first := post("eligibility-review-create-1", body)
	if first.Code != 200 {
		t.Fatalf("review=%d/%s", first.Code, first.Body)
	}
	replay := post("eligibility-review-create-1", body)
	if replay.Code != 200 || replay.Body.String() != first.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("review replay=%d/%s", replay.Code, replay.Body)
	}
	stale := post("eligibility-review-stale-1", `{"expectedRevision":0,"accountDid":"did:plc:eligibility","state":"eligible","reason":"appeal upheld"}`)
	if stale.Code != 409 {
		t.Fatalf("stale eligibility=%d/%s", stale.Code, stale.Body)
	}
	restored := post("eligibility-review-restore-1", `{"expectedRevision":1,"accountDid":"did:plc:eligibility","state":"eligible","reason":"appeal upheld"}`)
	if restored.Code != 200 {
		t.Fatalf("restore eligibility=%d/%s", restored.Code, restored.Body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/accounts/did:plc:eligibility/eligibility", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"revision":2`) {
		t.Fatalf("eligibility read=%d/%s", w.Code, w.Body)
	}
}

func TestConsoleIntimateImageRejectsUnconfiguredTargetWithoutCommitting(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	policy, _ := safetyincident.NewDeadlinePolicy(nil)
	deps.SafetyWorkflows = safetyincident.NewWorkflowService(pool, policy, time.Now)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	r := httptest.NewRequest("POST", "/v1/admin/safety/workflows/intimate-images", strings.NewReader(`{"receivedAt":"2026-10-15T08:00:00Z","standingCode":"synthetic-standing","declarationsCode":"synthetic-declarations"}`))
	r.Header.Set("Idempotency-Key", "unconfigured-target-fixture")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "deadline_target_not_configured") {
		t.Fatalf("unconfigured target=%d %s", w.Code, w.Body)
	}
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest("GET", "/v1/admin/safety/workflows?limit=10", nil))
	if list.Code != 200 || strings.Contains(list.Body.String(), "intimateImage") {
		t.Fatalf("rejected command committed: %s", list.Body)
	}
}
