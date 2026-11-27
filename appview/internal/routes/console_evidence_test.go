package routes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/retention"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func evidenceFixture(t *testing.T) (*Dependencies, uuid.UUID, []byte) {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	scan, incident := uuid.New(), uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'benign-console-image','fixture','fixture','fixture','match')`, scan); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'synthetic','synthetic',now(),now())`, incident, scan); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	deps.SafetyEvidence = safetyincident.NewEvidenceService(pool, safetyincident.NewMemoryEvidenceStore(), time.Now)
	deps.SafetyHolds = safetyincident.NewHoldService(pool)
	return deps, incident, benignEvidencePNG(t)
}
func benignEvidencePNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
func evidenceMux(t *testing.T, deps *Dependencies) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	return mux
}
func evidencePost(mux *http.ServeMux, path, key string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	r := httptest.NewRequest("POST", path, bytes.NewReader(data))
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}
func preserveFixture(t *testing.T, mux *http.ServeMux, incident uuid.UUID, data []byte) string {
	t.Helper()
	w := evidencePost(mux, "/v1/admin/safety/incidents/"+incident.String()+"/evidence", "benign-preserve-fixture", map[string]any{"expectedRevision": 0, "bytes": data, "contentType": "image/png", "reason": "synthetic preservation", "expiresAt": time.Now().Add(24 * time.Hour)})
	if w.Code != 201 {
		t.Fatalf("preserve status=%d", w.Code)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.ID
}

func TestConsoleEvidenceAccessRequiresIncidentAndAuditsBinaryDelivery(t *testing.T) {
	deps, incident, data := evidenceFixture(t)
	mux := evidenceMux(t, deps)
	id := preserveFixture(t, mux, incident, data)
	path := "/v1/admin/safety/evidence/" + id + "/access"
	mismatch := evidencePost(mux, path, "", map[string]any{"incidentReference": uuid.New(), "reason": "wrong incident"})
	if mismatch.Code != 403 {
		t.Fatalf("mismatched incident access status=%d", mismatch.Code)
	}
	accessed := evidencePost(mux, path, "", map[string]any{"incidentReference": incident, "reason": "synthetic verification"})
	if accessed.Code != 200 || !bytes.Equal(accessed.Body.Bytes(), data) {
		t.Fatalf("binary access status=%d length=%d", accessed.Code, accessed.Body.Len())
	}
	for key, want := range map[string]string{"Cache-Control": "no-store", "Content-Type": "application/octet-stream", "X-Content-Type-Options": "nosniff"} {
		if accessed.Header().Get(key) != want {
			t.Fatalf("binary header %s=%q", key, accessed.Header().Get(key))
		}
	}
	audit := httptest.NewRecorder()
	mux.ServeHTTP(audit, httptest.NewRequest("GET", "/v1/admin/safety/evidence/"+id+"/accesses?limit=10", nil))
	if audit.Code != 200 || !strings.Contains(audit.Body.String(), "synthetic verification") || !strings.Contains(audit.Body.String(), `"allowed":false`) || !strings.Contains(audit.Body.String(), "operator:fixture") {
		t.Fatalf("access audit status=%d", audit.Code)
	}
	for _, role := range []safetyincident.Role{safetyincident.RoleModerator, safetyincident.RoleHelper} {
		deps.OperatorAuthentication = consoleTestAuthentication(role)
		denied := evidencePost(evidenceMux(t, deps), path, "", map[string]any{"incidentReference": incident, "reason": "denied"})
		if denied.Code != 403 {
			t.Fatalf("%s binary access status=%d", role, denied.Code)
		}
	}
}

func TestConsoleEvidenceCapabilitiesAndPreservationBounds(t *testing.T) {
	deps, incident, data := evidenceFixture(t)
	mux := evidenceMux(t, deps)
	read := func(mux *http.ServeMux) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/safety/evidence/capabilities", nil))
		return w
	}
	caps := read(mux)
	var capabilities struct {
		PreservationAvailable bool `json:"preservationAvailable"`
		AccessAvailable       bool `json:"accessAvailable"`
		Max                   int  `json:"maxPreservationBytes"`
	}
	if json.Unmarshal(caps.Body.Bytes(), &capabilities) != nil || caps.Code != 200 || !capabilities.PreservationAvailable || !capabilities.AccessAvailable || capabilities.Max != 524288 {
		t.Fatalf("capabilities status=%d max=%d", caps.Code, capabilities.Max)
	}
	for _, input := range []struct {
		data []byte
		mime string
	}{{data, "image/jpeg"}, {make([]byte, 524289), "image/png"}} {
		w := evidencePost(mux, "/v1/admin/safety/incidents/"+incident.String()+"/evidence", uuid.NewString(), map[string]any{"expectedRevision": 0, "bytes": input.data, "contentType": input.mime, "reason": "synthetic validation", "expiresAt": time.Now().Add(time.Hour)})
		if w.Code != 400 {
			t.Fatalf("invalid preservation status=%d", w.Code)
		}
	}
	deps.SafetyEvidence = nil
	caps = read(evidenceMux(t, deps))
	if json.Unmarshal(caps.Body.Bytes(), &capabilities) != nil || caps.Code != 200 || capabilities.PreservationAvailable || capabilities.AccessAvailable {
		t.Fatalf("unconfigured capabilities status=%d", caps.Code)
	}
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	if read(evidenceMux(t, deps)).Code != 403 {
		t.Fatal("moderator read restricted capabilities")
	}
}

func TestConsoleEvidenceHoldReleaseAndRetentionReadback(t *testing.T) {
	deps, incident, data := evidenceFixture(t)
	objects := safetyincident.NewMemoryEvidenceStore()
	deps.SafetyEvidence = safetyincident.NewEvidenceService(deps.DB, objects, time.Now)
	mux := evidenceMux(t, deps)
	id := preserveFixture(t, mux, incident, data)
	held := evidencePost(mux, "/v1/admin/safety/incidents/"+incident.String()+"/holds", "synthetic-evidence-hold", map[string]any{"expectedRevision": 1, "evidenceIds": []string{id}, "basis": "synthetic approved hold", "expiresAt": time.Now().Add(48 * time.Hour)})
	if held.Code != 201 {
		t.Fatalf("create hold=%d", held.Code)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(held.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	metadata := get("/v1/admin/safety/evidence/" + id)
	if metadata.Code != 200 || !strings.Contains(metadata.Body.String(), `"held":true`) {
		t.Fatal("active hold missing from evidence metadata")
	}
	scoped := get("/v1/admin/safety/holds/" + result.ID + "/evidence")
	if scoped.Code != 200 || !strings.Contains(scoped.Body.String(), id) {
		t.Fatalf("hold evidence=%d", scoped.Code)
	}
	worker := retention.NewWorker(deps.DB, objects, func() time.Time { return time.Now().Add(36 * time.Hour) }, nil)
	if deleted, err := worker.RunOnce(t.Context(), 10); err != nil || deleted != 0 {
		t.Fatalf("held evidence retained=%d err=%v", deleted, err)
	}
	released := evidencePost(mux, "/v1/admin/safety/holds/"+result.ID+"/releases", "synthetic-hold-release", map[string]any{"expectedRevision": 0, "reason": "synthetic hold no longer required"})
	if released.Code != 200 {
		t.Fatalf("release=%d", released.Code)
	}
	audit := get("/v1/admin/safety/holds/" + result.ID + "/releases")
	if audit.Code != 200 || !strings.Contains(audit.Body.String(), "operator:fixture") {
		t.Fatalf("release audit=%d", audit.Code)
	}
	if deleted, err := worker.RunOnce(t.Context(), 10); err != nil || deleted != 1 {
		t.Fatalf("expired unheld evidence deletion=%d err=%v", deleted, err)
	}
	metadata = get("/v1/admin/safety/evidence/" + id)
	var state struct {
		DeletedAt *time.Time `json:"deletedAt"`
		Held      bool       `json:"held"`
	}
	if json.Unmarshal(metadata.Body.Bytes(), &state) != nil || metadata.Code != 200 || state.DeletedAt == nil || state.Held {
		t.Fatal("retention state missing")
	}
	unavailable := evidencePost(mux, "/v1/admin/safety/evidence/"+id+"/access", "", map[string]any{"incidentReference": incident, "reason": "expired fixture"})
	if unavailable.Code != 404 {
		t.Fatalf("deleted evidence access=%d", unavailable.Code)
	}
	for _, channel := range []string{"retention-events", "retention-jobs"} {
		w := get("/v1/admin/safety/evidence/" + id + "/" + channel)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("retention read %s=%d", channel, w.Code)
		}
		if strings.Contains(w.Body.String(), "object_key") || strings.Contains(w.Body.String(), "lease_owner") {
			t.Fatal("private retention storage metadata exposed")
		}
	}
}

func TestConsolePreservesEvidenceDirectlyFromUserReportCase(t *testing.T) {
	ctx := t.Context()
	pool := testdb.WithMigratedSchema(t)
	caseID := uuid.New()
	const owner = "did:example:case-evidence-owner"
	const subject = "at://" + owner + "/social.craftsky.feed.post/test"
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(
		id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,
		subject_uri,subject_cid_snapshot,owner_did,safe_snapshot
	) VALUES($1,$2,'post',$3,'social.craftsky.feed.post','test',$2,'test-cid',$3,'{}')`, caseID, subject, owner); err != nil {
		t.Fatal(err)
	}
	objects := safetyincident.NewMemoryEvidenceStore()
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleSafetyAdministrator)
	deps.SafetyEvidence = safetyincident.NewEvidenceService(pool, objects, time.Now)
	deps.SafetyIncidents = safetyincident.NewStore(pool, time.Now)
	admin := evidenceMux(t, deps)
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	moderator := evidenceMux(t, deps)

	path := "/v1/admin/moderation/cases/MOD-" + caseID.String() + "/evidence"
	data := benignEvidencePNG(t)
	expiresAt := time.Now().Add(time.Hour)
	body := map[string]any{"bytes": data, "contentType": "image/png", "reason": "synthetic report evidence", "expiresAt": expiresAt}
	if denied := evidencePost(moderator, path, "manual-evidence-denied", body); denied.Code != 403 {
		t.Fatalf("moderator preserve=%d", denied.Code)
	}
	expired := map[string]any{"bytes": data, "contentType": "image/png", "reason": "expired synthetic retention", "expiresAt": time.Now().Add(-time.Hour)}
	if rejected := evidencePost(admin, path, "manual-evidence-expired", expired); rejected.Code != 400 {
		t.Fatalf("invalid deadline status=%d", rejected.Code)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_incidents WHERE source_case_id=$1`, caseID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid write left incident count=%d err=%v", count, err)
	}

	first := evidencePost(admin, path, "manual-evidence-initial", body)
	if first.Code != 201 {
		t.Fatalf("case evidence preserve=%d %s", first.Code, first.Body)
	}
	replay := evidencePost(admin, path, "manual-evidence-initial", body)
	if replay.Code != 201 || replay.Header().Get("Idempotent-Replayed") != "true" || replay.Body.String() != first.Body.String() {
		t.Fatalf("case evidence replay did not return the committed receipt")
	}
	var receipt struct {
		ID                string `json:"id"`
		IncidentReference string `json:"incidentReference"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &receipt); err != nil || receipt.ID == "" || receipt.IncidentReference == "" || objects.ObjectCount() != 1 {
		t.Fatalf("invalid preservation receipt or object count: %s", first.Body)
	}

	var incidentKind, linkedOwner, linkedURI, linkedCID, caseState string
	var scanResult *uuid.UUID
	var caseRevision int64
	err := pool.QueryRow(ctx, `SELECT i.incident_kind,i.scan_result_id,s.owner_did,s.subject_uri,s.source_cid,c.state,c.revision
		FROM safety_incidents i JOIN safety_incident_subjects s ON s.incident_id=i.id
		JOIN moderation_cases c ON c.id=i.source_case_id WHERE i.id=$1`, receipt.IncidentReference).
		Scan(&incidentKind, &scanResult, &linkedOwner, &linkedURI, &linkedCID, &caseState, &caseRevision)
	if err != nil || incidentKind != "manualEvidence" || scanResult != nil || linkedOwner != owner ||
		linkedURI != subject || linkedCID != "test-cid" || caseState != "open" || caseRevision != 0 {
		t.Fatalf("manual origin or moderation state incorrect: kind=%s case=%s/%d err=%v", incidentKind, caseState, caseRevision, err)
	}
	detail := httptest.NewRecorder()
	admin.ServeHTTP(detail, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+caseID.String(), nil))
	if detail.Code != 200 || !strings.Contains(detail.Body.String(), receipt.IncidentReference) {
		t.Fatalf("linked incident missing from case detail: status=%d %s", detail.Code, detail.Body)
	}
	accessed := evidencePost(admin, "/v1/admin/safety/evidence/"+receipt.ID+"/access", "", map[string]any{
		"incidentReference": receipt.IncidentReference, "reason": "verify synthetic case image",
	})
	if accessed.Code != 200 || !bytes.Equal(accessed.Body.Bytes(), data) {
		t.Fatalf("case image readback status=%d", accessed.Code)
	}

	body["expiresAt"] = expiresAt.Add(time.Hour)
	second := evidencePost(admin, path, "manual-evidence-second", body)
	if second.Code != 201 || !strings.Contains(second.Body.String(), receipt.IncidentReference) || objects.ObjectCount() != 2 {
		t.Fatalf("second image did not reuse case incident: %d %s", second.Code, second.Body)
	}
	worker := retention.NewWorker(pool, objects, func() time.Time { return expiresAt.Add(time.Minute) }, nil)
	if deleted, err := worker.RunOnce(ctx, 10); err != nil || deleted != 1 || objects.ObjectCount() != 1 {
		t.Fatalf("first expiry deleted=%d err=%v", deleted, err)
	}
	if err := pool.QueryRow(ctx, `SELECT owner_did FROM safety_incident_subjects WHERE incident_id=$1`, receipt.IncidentReference).Scan(&linkedOwner); err != nil || linkedOwner != owner {
		t.Fatalf("retained evidence lost its owner link: err=%v", err)
	}
}
