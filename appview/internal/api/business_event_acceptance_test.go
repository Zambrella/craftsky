package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessEventLifecycleAcceptance(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL+`
		CREATE TABLE craftsky_business_record_tombstones (
			uri TEXT PRIMARY KEY,
			owner_did TEXT NOT NULL,
			collection TEXT NOT NULL,
			source_revision TEXT NOT NULL,
			deleted_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	visitor := syntax.DID("did:plc:visitor")
	now := businessEventNow
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed eligible business owner membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed eligible business owner account type: %v", err)
	}

	effects := newBusinessEventEffects()
	factory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return effects, nil
	}
	projector := index.NewCraftskyBusinessEvent()
	store := business.NewStore(pool)
	clock := func() time.Time { return now }

	create := api.PostBusinessEventHandler(factory, clock)
	created := serveBusinessEventAcceptanceMutation(t, create, http.MethodPost, "/v1/events", validBusinessEventBody(true), "", "", "")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	createdResponse := decodeBusinessEventMutationResponse(t, created)
	if createdResponse.DID != owner.String() || createdResponse.CID != businessEventCID1 || createdResponse.Rkey == "" ||
		createdResponse.URI != "at://"+owner.String()+"/social.craftsky.business.event/"+createdResponse.Rkey {
		t.Fatalf("create identity = %+v", createdResponse)
	}
	if _, err := syntax.ParseTID(createdResponse.Rkey); err != nil {
		t.Fatalf("create rkey %q is not a TID: %v", createdResponse.Rkey, err)
	}
	rkey := syntax.RecordKey(createdResponse.Rkey)
	createdRecord := effects.records[rkey]
	if createdRecord["$type"] != "social.craftsky.business.event" || createdRecord["createdAt"] != "2026-09-01T12:34:56Z" {
		t.Fatalf("server-authored create record = %#v", createdRecord)
	}
	projectBusinessEventAcceptance(t, pool, projector, tap.Event{
		Action: "create", URI: syntax.ATURI(createdResponse.URI), DID: owner,
		Collection: "social.craftsky.business.event", Rkey: rkey, CID: syntax.CID(createdResponse.CID),
		Rev: "3mseventrev01", Record: marshalBusinessEventAcceptanceRecord(t, createdRecord),
	})

	direct := api.GetBusinessEventHandler(store, clock)
	projected := serveBusinessEventAcceptanceRead(direct, owner, owner, rkey)
	projectedEvent := decodeBusinessEventAcceptanceEvent(t, projected, http.StatusOK)
	if projectedEvent.URI.String() != createdResponse.URI || projectedEvent.CID.String() != createdResponse.CID ||
		projectedEvent.Name != "Fiber Fair" || projectedEvent.CreatedAt != "2026-09-01T12:34:56Z" {
		t.Fatalf("projected direct event = %+v", projectedEvent)
	}
	var projectedBody map[string]any
	if err := json.Unmarshal(projected.Body.Bytes(), &projectedBody); err != nil {
		t.Fatalf("decode projected event image: %v", err)
	}
	wantProjectedImage := map[string]any{
		"cid":      "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a",
		"mime":     "image/webp",
		"size":     float64(1024),
		"alt":      "Event poster",
		"thumb":    "https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:owner/bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a@webp",
		"fullsize": "https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:owner/bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a@webp",
		"aspectRatio": map[string]any{
			"width":  float64(4),
			"height": float64(3),
		},
	}
	if got := projectedBody["image"]; !jsonObjectsEqual(got, wantProjectedImage) {
		t.Errorf("direct event image = %#v, want exact normalized image %#v", got, wantProjectedImage)
	}

	putsBeforeInjection := len(effects.puts)
	createdAtCreate := strings.TrimSuffix(validBusinessEventBody(false), "}") + `,"createdAt":"2026-09-01T12:34:56Z"}`
	rejectedCreate := serveBusinessEventAcceptanceMutation(t, create, http.MethodPost, "/v1/events", createdAtCreate, "", "", "")
	assertBusinessEventAcceptanceError(t, rejectedCreate, http.StatusUnprocessableEntity, "validation_failed")
	if len(effects.puts) != putsBeforeInjection {
		t.Fatalf("createdAt create reached PDS: puts %d -> %d", putsBeforeInjection, len(effects.puts))
	}

	update := api.PutBusinessEventHandler(factory, clock)
	for _, suppliedCreatedAt := range []string{"2026-09-01T12:34:56Z", "2026-08-01T00:00:00Z"} {
		body := strings.TrimSuffix(validBusinessEventBody(false), "}") + `,"createdAt":"` + suppliedCreatedAt + `"}`
		rejected := serveBusinessEventAcceptanceMutation(t, update, http.MethodPut,
			"/v1/events/"+owner.String()+"/"+rkey.String(), body, businessEventCID1, owner.String(), rkey.String())
		assertBusinessEventAcceptanceError(t, rejected, http.StatusUnprocessableEntity, "validation_failed")
	}
	if len(effects.reads) != 0 || len(effects.puts) != putsBeforeInjection || effects.records[rkey]["createdAt"] != "2026-09-01T12:34:56Z" {
		t.Fatalf("createdAt update changed PDS state: reads=%d puts=%d record=%#v", len(effects.reads), len(effects.puts), effects.records[rkey])
	}

	updated := serveBusinessEventAcceptanceMutation(t, update, http.MethodPut,
		"/v1/events/"+owner.String()+"/"+rkey.String(),
		strings.Replace(validBusinessEventBody(true), "Fiber Fair", "Fiber Fair Updated", 1),
		businessEventCID1, owner.String(), rkey.String())
	if updated.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	updatedResponse := decodeBusinessEventMutationResponse(t, updated)
	if updatedResponse.CID != businessEventCID2 || updatedResponse.CID == createdResponse.CID {
		t.Fatalf("update CID=%q, create CID=%q", updatedResponse.CID, createdResponse.CID)
	}
	if effects.records[rkey]["createdAt"] != createdRecord["createdAt"] {
		t.Fatalf("update createdAt=%v, want preserved %v", effects.records[rkey]["createdAt"], createdRecord["createdAt"])
	}
	projectBusinessEventAcceptance(t, pool, projector, tap.Event{
		Action: "update", URI: syntax.ATURI(updatedResponse.URI), DID: owner,
		Collection: "social.craftsky.business.event", Rkey: rkey, CID: syntax.CID(updatedResponse.CID),
		Rev: "3mseventrev02", Record: marshalBusinessEventAcceptanceRecord(t, effects.records[rkey]),
	})
	projected = serveBusinessEventAcceptanceRead(direct, visitor, owner, rkey)
	projectedEvent = decodeBusinessEventAcceptanceEvent(t, projected, http.StatusOK)
	if projectedEvent.Name != "Fiber Fair Updated" || projectedEvent.CID.String() != businessEventCID2 ||
		projectedEvent.CreatedAt != "2026-09-01T12:34:56Z" {
		t.Fatalf("updated projection = %+v", projectedEvent)
	}

	statusEvents := []struct {
		rkey     syntax.RecordKey
		name     string
		startsAt time.Time
		endsAt   time.Time
		status   string
		wantPast bool
	}{
		{rkey: "3msongoing001", name: "Ongoing", startsAt: now.Add(-time.Hour), endsAt: now.Add(time.Hour), status: "scheduled"},
		{rkey: "3msended00001", name: "Ended", startsAt: now.Add(-2 * time.Hour), endsAt: now.Add(-time.Hour), status: "scheduled", wantPast: true},
		{rkey: "3mscancel0001", name: "Cancelled", startsAt: now.Add(2 * time.Hour), endsAt: now.Add(3 * time.Hour), status: "cancelled"},
		{rkey: "3mspostpone01", name: "Postponed", startsAt: now.Add(4 * time.Hour), endsAt: now.Add(5 * time.Hour), status: "postponed"},
	}
	for _, fixture := range statusEvents {
		record := map[string]any{
			"$type": "social.craftsky.business.event", "name": fixture.name,
			"startsAt": fixture.startsAt.UTC().Format(time.RFC3339), "endsAt": fixture.endsAt.UTC().Format(time.RFC3339),
			"roles": []string{"vendor"}, "status": fixture.status, "createdAt": "2026-09-01T12:34:56Z",
		}
		uri := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.event/" + fixture.rkey.String())
		projectBusinessEventAcceptance(t, pool, projector, tap.Event{
			Action: "create", URI: uri, DID: owner, Collection: "social.craftsky.business.event",
			Rkey: fixture.rkey, CID: syntax.CID(businessEventCID1), Rev: "3msstatusrev1",
			Record: marshalBusinessEventAcceptanceRecord(t, record),
		})
		response := serveBusinessEventAcceptanceRead(direct, visitor, owner, fixture.rkey)
		event := decodeBusinessEventAcceptanceEvent(t, response, http.StatusOK)
		if event.Status.Value != fixture.status || event.Past != fixture.wantPast {
			t.Errorf("%s direct status/past = %q/%v, want %q/%v", fixture.name, event.Status.Value, event.Past, fixture.status, fixture.wantPast)
		}
	}

	upcoming := api.GetProfileBusinessEventsHandler(store, fakeResolver{}, testEventCursorCodec(t), clock)
	upcomingResponse := serveBusinessEventAcceptanceUpcoming(upcoming, visitor, owner)
	if upcomingResponse.Code != http.StatusOK {
		t.Fatalf("upcoming status=%d body=%s", upcomingResponse.Code, upcomingResponse.Body.String())
	}
	var page api.BusinessEventPage
	if err := json.Unmarshal(upcomingResponse.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode upcoming page: %v", err)
	}
	wantUpcoming := map[string]bool{"Ongoing": true, "Fiber Fair Updated": true}
	if len(page.Items) != len(wantUpcoming) {
		t.Fatalf("upcoming events=%+v, want ongoing and scheduled future", page.Items)
	}
	for _, event := range page.Items {
		if !wantUpcoming[event.Name] {
			t.Errorf("upcoming included %q", event.Name)
		}
	}
	var upcomingWire struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(upcomingResponse.Body.Bytes(), &upcomingWire); err != nil {
		t.Fatalf("decode upcoming image wire: %v", err)
	}
	var upcomingImage any
	for _, event := range upcomingWire.Items {
		if event["name"] == "Fiber Fair Updated" {
			upcomingImage = event["image"]
		}
	}
	if !jsonObjectsEqual(upcomingImage, wantProjectedImage) {
		t.Errorf("upcoming event image = %#v, want exact normalized image %#v", upcomingImage, wantProjectedImage)
	}

	remove := api.DeleteBusinessEventHandler(factory)
	deleted := serveBusinessEventAcceptanceMutation(t, remove, http.MethodDelete,
		"/v1/events/"+owner.String()+"/"+rkey.String(), "", businessEventCID2, owner.String(), rkey.String())
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	if _, present := effects.records[rkey]; present {
		t.Fatal("PDS fake retained deleted event")
	}
	projectBusinessEventAcceptance(t, pool, projector, tap.Event{
		Action: "delete", URI: syntax.ATURI(updatedResponse.URI), DID: owner,
		Collection: "social.craftsky.business.event", Rkey: rkey, Rev: "3mseventrev03",
	})
	deletedRead := serveBusinessEventAcceptanceRead(direct, owner, owner, rkey)
	assertBusinessEventAcceptanceError(t, deletedRead, http.StatusNotFound, "event_not_found")
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_business_events WHERE uri=$1`, updatedResponse.URI).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("projected event rows=%d error=%v, want absent", rows, err)
	}
	var tombstoneRevision string
	if err := pool.QueryRow(ctx, `SELECT source_revision FROM craftsky_business_record_tombstones WHERE uri=$1`, updatedResponse.URI).Scan(&tombstoneRevision); err != nil || tombstoneRevision != "3mseventrev03" {
		t.Fatalf("delete tombstone revision=%q error=%v", tombstoneRevision, err)
	}
}

func projectBusinessEventAcceptance(t *testing.T, pool *pgxpool.Pool, projector *index.CraftskyBusinessEvent, event tap.Event) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin event projection: %v", err)
	}
	outcome, err := projector.Project(ctx, tx, event)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		_ = tx.Rollback(ctx)
		t.Fatalf("project %s event: outcome=%+v error=%v", event.Action, outcome, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit event projection: %v", err)
	}
}

func marshalBusinessEventAcceptanceRecord(t *testing.T, record map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal event record: %v", err)
	}
	return raw
}

func serveBusinessEventAcceptanceMutation(
	t *testing.T,
	handler http.Handler,
	method, target, body, ifMatch, did, rkey string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	request.SetPathValue("did", did)
	request.SetPathValue("rkey", rkey)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	request = request.WithContext(ctxkeys.WithRunID(ctx, "business-event-acceptance"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func serveBusinessEventAcceptanceRead(handler http.Handler, caller, owner syntax.DID, rkey syntax.RecordKey) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/events/"+owner.String()+"/"+rkey.String(), nil)
	request.SetPathValue("did", owner.String())
	request.SetPathValue("rkey", rkey.String())
	ctx := middleware.WithDID(request.Context(), caller)
	request = request.WithContext(ctxkeys.WithRunID(ctx, "business-event-acceptance"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func serveBusinessEventAcceptanceUpcoming(handler http.Handler, caller, owner syntax.DID) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/profiles/"+owner.String()+"/events", nil)
	request.SetPathValue("handleOrDid", owner.String())
	request = request.WithContext(middleware.WithDID(request.Context(), caller))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeBusinessEventAcceptanceEvent(t *testing.T, response *httptest.ResponseRecorder, wantStatus int) business.EventView {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("event read status=%d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var event business.EventView
	if err := json.Unmarshal(response.Body.Bytes(), &event); err != nil {
		t.Fatalf("decode event read: %v", err)
	}
	return event
}

func assertBusinessEventAcceptanceError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("error status=%d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error != wantCode {
		t.Fatalf("error body=%s decode=%v, want %q", response.Body.String(), err, wantCode)
	}
}
