package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func TestConsolePreviewShowsCapturedPostMediaAndWithholdsChangedOrUnsafeContent(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	id := uuid.New()
	const did = "did:example:preview-author"
	const uri = "at://" + did + "/social.craftsky.feed.post/preview"
	for _, statement := range []string{
		`INSERT INTO craftsky_posts(uri,did,rkey,cid,text,images,record,created_at,indexed_at) VALUES('` + uri + `','` + did + `','preview','captured-cid','A knitted scarf','[{"cid":"devmedia:scarf","mime":"image/jpeg","alt":"Blue knitted scarf"}]','{}',now(),now())`,
		`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES('` + uri + `','post','captured-cid','clear')`,
	} {
		if _, err := pool.Exec(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot) VALUES($1,$2,'post',$3,'social.craftsky.feed.post','preview',$4,'captured-cid',$3,'{}')`, id, "post:"+uri, did, uri); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	get := func() map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+id.String()+"/preview", nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("preview=%d %s", w.Code, w.Body)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	body := get()
	encoded, _ := json.Marshal(body)
	if body["state"] != "available" || body["mediaAvailable"] != true || !strings.Contains(string(encoded), `"fullsize":"/v1/dev/media/scarf"`) || !strings.Contains(string(encoded), "Blue knitted scarf") {
		t.Fatalf("captured post media missing: %s", encoded)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_posts SET record='{"embed":{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123},"alt":"Weaving demonstration"}}' WHERE uri=$1`, uri); err != nil {
		t.Fatal(err)
	}
	body = get()
	encoded, _ = json.Marshal(body)
	if !strings.Contains(string(encoded), "playlist.m3u8") || !strings.Contains(string(encoded), "Weaving demonstration") {
		t.Fatalf("video preview missing: %s", encoded)
	}

	if _, err := pool.Exec(t.Context(), `INSERT INTO craftsky_project_posts(uri,raw_project,common_craft_type) VALUES($1,'{"common":{"title":"Winter scarf","craftType":"social.craftsky.feed.defs#knitting","materials":[{"text":"Blue wool"}]}}','social.craftsky.feed.defs#knitting')`, uri); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_posts SET images='[]',record='{"embed":{"$type":"app.bsky.embed.external","external":{"uri":"https://example.com/pattern","title":"Pattern notes","description":"Project reference","thumb":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"image/jpeg","size":100}}}}' WHERE uri=$1`, uri); err != nil {
		t.Fatal(err)
	}
	body = get()
	encoded, _ = json.Marshal(body)
	for _, part := range []string{"Winter scarf", "Blue wool", "Pattern notes", "feed_thumbnail"} {
		if !strings.Contains(string(encoded), part) {
			t.Fatalf("missing project/link context %q: %s", part, encoded)
		}
	}

	for _, mutation := range []struct{ sql, state string }{
		{`UPDATE craftsky_posts SET cid='changed-cid' WHERE uri='` + uri + `'`, "unavailable"},
		{`UPDATE craftsky_posts SET cid='captured-cid' WHERE uri='` + uri + `'; UPDATE image_subject_states SET visibility_state='blocked' WHERE subject_uri='` + uri + `'`, "unavailable"},
	} {
		if _, err := pool.Exec(t.Context(), mutation.sql); err != nil {
			t.Fatal(err)
		}
		body = get()
		encoded, _ = json.Marshal(body)
		if body["state"] != mutation.state || body["mediaAvailable"] != false || len(body) != 4 {
			t.Fatalf("unsafe preview: %s", encoded)
		}
	}
	// Incident-linked subjects must expose no ordinary-preview bytes or URLs,
	// even when the public image projection is clear.
	scanID, incidentID := uuid.New(), uuid.New()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`UPDATE image_subject_states SET visibility_state='clear' WHERE subject_uri=$1`, []any{uri}},
		{`INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state) VALUES($1,'preview-restricted','fixture','fixture','fixture','match')`, []any{scanID}},
		{`INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'synthetic','synthetic',now(),now())`, []any{incidentID, scanID}},
		{`INSERT INTO safety_incident_subjects(incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot,linked_at) VALUES($1,$2,$3,'captured-cid','post','image:0',now())`, []any{incidentID, did, uri}},
	} {
		if _, err := pool.Exec(t.Context(), q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	body = get()
	encoded, _ = json.Marshal(body)
	if body["state"] != "restricted" || len(body) != 4 {
		t.Fatalf("restricted content escaped: %s", encoded)
	}
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleHelper)
	helperMux := http.NewServeMux()
	AddRoutes(t.Context(), helperMux, deps)
	denied := httptest.NewRecorder()
	helperMux.ServeHTTP(denied, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+id.String()+"/preview", nil))
	if denied.Code != 403 {
		t.Fatalf("helper preview=%d", denied.Code)
	}

}

func TestConsolePreviewRendersAccountAndEventCardsWithMedia(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	accountID, eventID := uuid.New(), uuid.New()
	const did = "did:example:rich-preview"
	const eventURI = "at://" + did + "/social.craftsky.business.event/fair"
	for _, statement := range []string{
		`INSERT INTO bluesky_profiles(did,display_name,description,avatar_cid,avatar_mime,banner_cid,banner_mime,record_cid) VALUES('` + did + `','Mira Makes','Weaving and knitting','avatar-cid','image/jpeg','banner-cid','image/png','profile-cid')`,
		`INSERT INTO craftsky_profiles(did,crafts,record_cid) VALUES('` + did + `',ARRAY['knitting'],'craft-profile-cid')`,
		`INSERT INTO craftsky_business_events(uri,owner_did,rkey,cid,source_revision,starts_at,ends_at,created_at,raw_record) VALUES('` + eventURI + `','` + did + `','fair','event-cid','rev','2030-01-01','2030-01-02',now(),' {"name":"Textile fair","summary":"Meet local makers","startsAt":"2030-01-01T10:00:00Z","endsAt":"2030-01-02T10:00:00Z","venueName":"Village hall","roles":["vendor"],"createdAt":"2026-10-16T10:00:00Z","image":{"image":{"$type":"blob","ref":{"$link":"bafkreigh2akiscaildc2qj7uuxyd6mfi7wxhgsauy5xnm7ttc7hxq7fhii"},"mimeType":"image/jpeg","size":100},"alt":"Textile stalls"}}')`,
		`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES('` + eventURI + `','event','event-cid','clear')`,
	} {
		if _, err := pool.Exec(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,subject_cid_snapshot,owner_did,safe_snapshot) VALUES($1,$2,'event',$3,'social.craftsky.business.event','fair',$4,'event-cid',$3,'{}'),($5,'account:'||$3,'account',$3,NULL,NULL,NULL,'profile-cid',$3,'{}')`, eventID, "event:"+eventURI, did, eventURI, accountID); err != nil {
		t.Fatal(err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.Config.ModerationAdminEnabled = true
	deps.OperatorAuthentication = consoleTestAuthentication(safetyincident.RoleModerator)
	mux := http.NewServeMux()
	AddRoutes(t.Context(), mux, deps)
	for _, test := range []struct {
		id    uuid.UUID
		parts []string
	}{
		{accountID, []string{`"mediaAvailable":true`, `"banner":"https://cdn.bsky.app/img/banner/`, `"avatar":"https://cdn.bsky.app/img/avatar/`, "Mira Makes", "Weaving and knitting", "knitting"}},
		{eventID, []string{`"mediaAvailable":true`, "Textile fair", "Meet local makers", "Textile stalls", "Village hall", "2030-01-01T10:00:00Z"}},
	} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+test.id.String()+"/preview", nil))
		if w.Code != 200 {
			t.Fatalf("preview=%d %s", w.Code, w.Body)
		}
		for _, part := range test.parts {
			if !strings.Contains(w.Body.String(), part) {
				t.Fatalf("missing %q in %s", part, w.Body)
			}
		}
	}
	if _, err := pool.Exec(t.Context(), `UPDATE bluesky_profiles SET record_cid='changed-profile' WHERE did=$1`, did); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+accountID.String()+"/preview", nil))
	if !strings.Contains(w.Body.String(), `"state":"unavailable"`) || strings.Contains(w.Body.String(), "cdn.bsky.app") {
		t.Fatalf("changed profile leaked: %s", w.Body)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,terminal_at) VALUES($1,'terminal',1,1,'synthetic test',now(),now())`, did); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/v1/admin/moderation/cases/MOD-"+eventID.String()+"/preview", nil))
	if !strings.Contains(w.Body.String(), `"state":"unavailable"`) || strings.Contains(w.Body.String(), "Textile") {
		t.Fatalf("terminal owner leaked: %s", w.Body)
	}

}
