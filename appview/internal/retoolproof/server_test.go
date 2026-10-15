package retoolproof

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/testdb"
)

func TestRealModerationRoundTrip(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	fixture, err := Seed(t.Context(), pool, "test-alpha-credential", "test-beta-credential")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(pool, fixture, "https://proof.example")
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, token, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/v1/admin/moderation/cases", "", "", ""); w.Code != 401 {
		t.Fatalf("anonymous queue: %d", w.Code)
	}
	identity := call("GET", "/v1/admin/proof/identity", "test-alpha-credential", "", "")
	if identity.Code != 200 || !strings.Contains(identity.Body.String(), "proof-alpha") {
		t.Fatalf("identity: %d %s", identity.Code, identity.Body)
	}
	queue := call("GET", "/v1/admin/moderation/cases", "test-alpha-credential", "", "")
	if queue.Code != 200 || !strings.Contains(queue.Body.String(), fixture.CaseReference) {
		t.Fatalf("queue: %d %s", queue.Code, queue.Body)
	}
	path := "/v1/admin/moderation/cases/" + fixture.CaseReference
	body := `{"expectedRevision":0,"disposition":"violation","reason":"spam","evidenceNotes":"Synthetic fixture review","consequences":["visibilityHide","strike"]}`
	decision := call("POST", path+"/decisions", "test-alpha-credential", "proof-decision-key-0001", body)
	if decision.Code != 200 {
		t.Fatalf("decision: %d %s", decision.Code, decision.Body)
	}
	replay := call("POST", path+"/decisions", "test-alpha-credential", "proof-decision-key-0001", body)
	if replay.Code != 200 || !strings.Contains(replay.Body.String(), `"replayed":true`) {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body)
	}
	stale := call("POST", path+"/effect-changes", "test-beta-credential", "proof-stale-key-000001", `{"expectedRevision":0,"negateEffects":["visibilityHide","strike"],"rationale":"Synthetic reversal"}`)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "case_revision_conflict") {
		t.Fatalf("stale: %d %s", stale.Code, stale.Body)
	}
	undo := call("POST", path+"/effect-changes", "test-beta-credential", "proof-reversal-key-0001", `{"expectedRevision":1,"negateEffects":["visibilityHide","strike"],"rationale":"Synthetic reversal"}`)
	if undo.Code != 200 {
		t.Fatalf("undo: %d %s", undo.Code, undo.Body)
	}
	var detail moderation.AdminDetail
	w := call("GET", path, "test-alpha-credential", "", "")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &detail) != nil {
		t.Fatalf("detail: %d %s", w.Code, w.Body)
	}
	if detail.Revision != 2 || detail.Standing.ActiveStrikeCount != 0 || len(detail.Events) != 2 || detail.Events[0].ActorID != "proof-alpha" || detail.Events[1].ActorID != "proof-beta" {
		t.Fatalf("incorrect final state: %+v", detail)
	}
	for _, effect := range detail.Effects {
		if effect.Active {
			t.Fatalf("effect still active: %+v", effect)
		}
	}
	var applied, negated int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE action='apply'), count(*) FILTER (WHERE action='negate') FROM moderation_outputs`).Scan(&applied, &negated); err != nil {
		t.Fatal(err)
	}
	if applied != 1 || negated != 1 {
		t.Fatalf("visibility output: apply=%d negate=%d", applied, negated)
	}
}

func TestProofAdapterPreservesCommandGuardsAndEvidenceAudit(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	fixture, err := Seed(t.Context(), pool, "test-alpha-credential", "test-beta-credential")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(pool, fixture, "https://proof.example")
	if err != nil {
		t.Fatal(err)
	}
	call := func(path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call("/v1/admin/proof/decisions", "test-alpha-credential", `{"expectedRevision":0,"idempotencyKey":"short"}`); w.Code != 400 {
		t.Fatalf("invalid key accepted: %d", w.Code)
	}
	if w := call("/v1/admin/proof/decisions", "test-alpha-credential", `{"expectedRevision":0,"idempotencyKey":"proof-adapter-decision","actorId":"spoofed"}`); w.Code != 400 {
		t.Fatalf("spoofed identity accepted: %d", w.Code)
	}
	if w := call("/v1/admin/proof/decisions", "test-alpha-credential", `{"expectedRevision":0,"idempotencyKey":"proof-adapter-decision"}`); w.Code != 200 {
		t.Fatalf("adapter decision: %d %s", w.Code, w.Body)
	}
	if w := call("/v1/admin/proof/effect-changes", "test-beta-credential", `{"expectedRevision":1,"idempotencyKey":"proof-adapter-reversal"}`); w.Code != 200 {
		t.Fatalf("adapter reversal: %d %s", w.Code, w.Body)
	}
	detail, err := moderation.NewStore(pool).AdminDetail(t.Context(), fixture.CaseReference)
	if err != nil || detail.Revision != 2 || detail.Standing.ActiveStrikeCount != 0 || detail.Events[1].ActorID != "proof-beta" {
		t.Fatalf("adapter result: %+v %v", detail, err)
	}
	evidencePath := "/v1/admin/safety/evidence/" + fixture.EvidenceID.String() + "/access"
	allowed := call(evidencePath, "test-alpha-credential", `{"reason":"Benign transport test"}`)
	if allowed.Code != 200 || allowed.Body.Len() != 68 || allowed.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(allowed.Body.String(), "\x89PNG") {
		t.Fatalf("binary transport: status=%d bytes=%d", allowed.Code, allowed.Body.Len())
	}
	denied := call(evidencePath, "test-beta-credential", `{"reason":"Denied transport test"}`)
	if denied.Code != 403 || strings.Contains(denied.Body.String(), "\x89PNG") {
		t.Fatalf("evidence denial: %d", denied.Code)
	}
	var total, permitted int
	if err := pool.QueryRow(t.Context(), `SELECT count(*),count(*) FILTER (WHERE allowed) FROM safety_evidence_accesses`).Scan(&total, &permitted); err != nil {
		t.Fatal(err)
	}
	if total != 2 || permitted != 1 {
		t.Fatalf("audit: total=%d allowed=%d", total, permitted)
	}
}
