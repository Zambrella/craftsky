package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

const (
	businessEventCID1 = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	businessEventCID2 = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fa"
)

var businessEventNow = time.Date(2026, time.September, 1, 12, 34, 56, 789, time.UTC)

type businessEventEffects struct {
	records map[syntax.RecordKey]map[string]any
	cids    map[syntax.RecordKey]syntax.CID
	reads   []pdseffects.ReadRecordRequest
	puts    []pdseffects.PutRecordRequest
	deletes []pdseffects.DeleteRecordRequest
}

func newBusinessEventEffects() *businessEventEffects {
	return &businessEventEffects{
		records: make(map[syntax.RecordKey]map[string]any),
		cids:    make(map[syntax.RecordKey]syntax.CID),
	}
}

func (*businessEventEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if len(targets) != 0 {
		return nil, errors.New("business event effects must be owner-only")
	}
	return []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:owner", Generation: ownerGeneration}}, nil
}

func (effects *businessEventEffects) ReadRecord(
	_ context.Context,
	request pdseffects.ReadRecordRequest,
	out any,
) (syntax.CID, error) {
	effects.reads = append(effects.reads, request)
	record, ok := effects.records[request.Rkey]
	if !ok {
		return "", auth.ErrRecordNotFound
	}
	target, ok := out.(*map[string]any)
	if !ok {
		return "", errors.New("unexpected business event read output")
	}
	encoded, _ := json.Marshal(record)
	if err := json.Unmarshal(encoded, target); err != nil {
		return "", err
	}
	return effects.cids[request.Rkey], nil
}

func (effects *businessEventEffects) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.puts = append(effects.puts, request)
	if request.ExpectedCID != "" && request.ExpectedCID != effects.cids[request.Rkey] {
		return pdseffects.RecordResult{}, auth.ErrRecordSwapConflict
	}
	encoded, _ := json.Marshal(request.Record)
	var record map[string]any
	if err := json.Unmarshal(encoded, &record); err != nil {
		return pdseffects.RecordResult{}, err
	}
	effects.records[request.Rkey] = record
	cid := syntax.CID(businessEventCID1)
	if effects.cids[request.Rkey] != "" {
		cid = businessEventCID2
	}
	effects.cids[request.Rkey] = cid
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.business.event/" + request.Rkey.String())
	return pdseffects.RecordResult{URI: uri, CID: cid}, nil
}

func (effects *businessEventEffects) DeleteRecord(
	_ context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.deletes = append(effects.deletes, request)
	if request.ExpectedCID != effects.cids[request.Rkey] {
		return pdseffects.RecordResult{}, auth.ErrRecordSwapConflict
	}
	delete(effects.records, request.Rkey)
	delete(effects.cids, request.Rkey)
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.business.event/" + request.Rkey.String())
	return pdseffects.RecordResult{URI: uri}, nil
}

func (*businessEventEffects) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected business event blob upload")
}

func TestBusinessEventHTTPPDSCRUD(t *testing.T) {
	t.Parallel()
	effects := newBusinessEventEffects()
	var factoryCalls int
	factory := func(_ context.Context, owner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		factoryCalls++
		if owner != "did:plc:owner" || sessionID != "oauth-owner-session" {
			t.Fatalf("effect factory scope = %q, %q", owner, sessionID)
		}
		return effects, nil
	}
	create := api.PostBusinessEventHandler(factory, func() time.Time { return businessEventNow })
	update := api.PutBusinessEventHandler(factory, func() time.Time { return businessEventNow })
	remove := api.DeleteBusinessEventHandler(factory)

	created := serveBusinessEventRequest(t, create, http.MethodPost, "/v1/events", validBusinessEventBody(true), "", "", "")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	createdResponse := decodeBusinessEventMutationResponse(t, created)
	if createdResponse.DID != "did:plc:owner" || createdResponse.CID != businessEventCID1 || createdResponse.Rkey == "" ||
		createdResponse.URI != "at://did:plc:owner/social.craftsky.business.event/"+createdResponse.Rkey {
		t.Fatalf("create response = %+v", createdResponse)
	}
	if _, err := syntax.ParseTID(createdResponse.Rkey); err != nil {
		t.Fatalf("create rkey %q is not a TID: %v", createdResponse.Rkey, err)
	}
	rkey := syntax.RecordKey(createdResponse.Rkey)
	createdRecord := effects.records[rkey]
	if createdRecord["$type"] != "social.craftsky.business.event" || createdRecord["createdAt"] != "2026-09-01T12:34:56Z" {
		t.Fatalf("created record = %#v", createdRecord)
	}
	image, ok := createdRecord["image"].(map[string]any)
	if !ok || image["alt"] != "Event poster" || image["image"] == nil {
		t.Fatalf("created image = %#v", createdRecord["image"])
	}
	if len(effects.reads) != 0 || len(effects.puts) != 1 || effects.puts[0].ExpectedCID != "" {
		t.Fatalf("create effects: reads=%d puts=%+v", len(effects.reads), effects.puts)
	}
	assertBusinessEventEffectScope(t, effects.puts[0].Owner, effects.puts[0].OwnerGeneration,
		effects.puts[0].ExpectedOwners, effects.puts[0].Collection, effects.puts[0].Rkey)

	updated := serveBusinessEventRequest(t, update, http.MethodPut,
		"/v1/events/did:plc:owner/"+rkey.String(), strings.Replace(validBusinessEventBody(false), "Fiber Fair", "Fiber Fair Updated", 1),
		businessEventCID1, "did:plc:owner", rkey.String())
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updated.Code, updated.Body.String())
	}
	updatedResponse := decodeBusinessEventMutationResponse(t, updated)
	if updatedResponse.CID != businessEventCID2 || updatedResponse.Rkey != rkey.String() {
		t.Fatalf("update response = %+v", updatedResponse)
	}
	if len(effects.reads) != 1 || len(effects.puts) != 2 || effects.puts[1].ExpectedCID != businessEventCID1 {
		t.Fatalf("update effects: reads=%+v puts=%+v", effects.reads, effects.puts)
	}
	if effects.records[rkey]["createdAt"] != "2026-09-01T12:34:56Z" || effects.records[rkey]["name"] != "Fiber Fair Updated" {
		t.Fatalf("updated record = %#v", effects.records[rkey])
	}

	deleted := serveBusinessEventRequest(t, remove, http.MethodDelete,
		"/v1/events/did:plc:owner/"+rkey.String(), "", businessEventCID2, "did:plc:owner", rkey.String())
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	deletedResponse := decodeBusinessEventMutationResponse(t, deleted)
	if deletedResponse.CID != businessEventCID2 || deletedResponse.Rkey != rkey.String() {
		t.Fatalf("delete response = %+v", deletedResponse)
	}
	if len(effects.reads) != 2 || len(effects.deletes) != 1 || effects.deletes[0].ExpectedCID != businessEventCID2 {
		t.Fatalf("delete effects: reads=%+v deletes=%+v", effects.reads, effects.deletes)
	}
	if _, exists := effects.records[rkey]; exists {
		t.Fatal("delete retained PDS record")
	}
	if factoryCalls != 3 {
		t.Fatalf("effect factory calls = %d, want 3", factoryCalls)
	}
}

func TestBusinessEventRejectsInvalidRequestsBeforeMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		body    string
		ifMatch string
		want    string
	}{
		{name: "create createdAt", method: http.MethodPost, body: strings.TrimSuffix(validBusinessEventBody(false), "}") + `,"createdAt":"2026-09-01T12:34:56Z"}`, want: "validation_failed"},
		{name: "update createdAt", method: http.MethodPut, body: strings.TrimSuffix(validBusinessEventBody(false), "}") + `,"createdAt":"2026-08-01T00:00:00Z"}`, ifMatch: businessEventCID1, want: "validation_failed"},
		{name: "snake case", method: http.MethodPost, body: `{"name":"Fair","starts_at":"2026-09-02T10:00:00Z"}`, want: "unexpected_field"},
		{name: "onlineUri", method: http.MethodPost, body: strings.TrimSuffix(validBusinessEventBody(false), "}") + `,"onlineUri":"https://event.example/live"}`, want: "unexpected_field"},
		{name: "invalid catalog", method: http.MethodPost, body: strings.Replace(validBusinessEventBody(false), `"mode":"hybrid"`, `"mode":"virtual"`, 1), want: "validation_failed"},
		{name: "invalid time", method: http.MethodPost, body: strings.Replace(validBusinessEventBody(false), "2026-09-02T18:00:00Z", "2026-09-02T09:00:00Z", 1), want: "validation_failed"},
		{name: "duplicate links", method: http.MethodPost, body: strings.Replace(validBusinessEventBody(false), "https://tickets.example/register", "https://event.example/details", 1), want: "validation_failed"},
		{name: "invalid image", method: http.MethodPost, body: strings.Replace(validBusinessEventBody(true), "image/webp", "image/gif", 1), want: "validation_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effects := newBusinessEventEffects()
			effects.records["3mzzzzzzzzzzz"] = map[string]any{"createdAt": "2026-09-01T12:34:56Z"}
			effects.cids["3mzzzzzzzzzzz"] = businessEventCID1
			handler := api.PostBusinessEventHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			}, func() time.Time { return businessEventNow })
			if test.method == http.MethodPut {
				handler = api.PutBusinessEventHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
					return effects, nil
				}, func() time.Time { return businessEventNow })
			}
			response := serveBusinessEventRequest(t, handler, test.method,
				"/v1/events/did:plc:owner/3mzzzzzzzzzzz", test.body, test.ifMatch, "did:plc:owner", "3mzzzzzzzzzzz")
			assertBusinessEventError(t, response, test.want)
			if len(effects.reads) != 0 || len(effects.puts) != 0 || len(effects.deletes) != 0 {
				t.Fatalf("invalid request reached PDS: reads=%d puts=%d deletes=%d", len(effects.reads), len(effects.puts), len(effects.deletes))
			}
		})
	}
}

func TestBusinessEventRejectsCrossOwnerAndInvalidCIDBeforeExecutor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		did     string
		ifMatch string
		status  int
		error   string
	}{
		{name: "cross-owner update", method: http.MethodPut, did: "did:plc:other", ifMatch: businessEventCID1, status: http.StatusForbidden, error: "forbidden"},
		{name: "cross-owner delete", method: http.MethodDelete, did: "did:plc:other", ifMatch: businessEventCID1, status: http.StatusForbidden, error: "forbidden"},
		{name: "missing update CID", method: http.MethodPut, did: "did:plc:owner", status: http.StatusConflict, error: "pds_record_conflict"},
		{name: "wildcard update CID", method: http.MethodPut, did: "did:plc:owner", ifMatch: "*", status: http.StatusConflict, error: "pds_record_conflict"},
		{name: "malformed delete CID", method: http.MethodDelete, did: "did:plc:owner", ifMatch: "not-a-cid", status: http.StatusConflict, error: "pds_record_conflict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factoryCalls := 0
			factory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				factoryCalls++
				return newBusinessEventEffects(), nil
			}
			handler := api.PutBusinessEventHandler(factory, func() time.Time { return businessEventNow })
			body := validBusinessEventBody(false)
			if test.method == http.MethodDelete {
				handler = api.DeleteBusinessEventHandler(factory)
				body = ""
			}
			response := serveBusinessEventRequest(t, handler, test.method,
				"/v1/events/"+test.did+"/3mzzzzzzzzzzz", body, test.ifMatch, test.did, "3mzzzzzzzzzzz")
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.status, response.Body.String())
			}
			assertBusinessEventError(t, response, test.error)
			if factoryCalls != 0 {
				t.Fatalf("effect executor created %d times", factoryCalls)
			}
		})
	}
}

func TestBusinessEventAuthoritativeCIDConflictsDoNotMutate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		seed   bool
	}{
		{name: "stale update", method: http.MethodPut, seed: true},
		{name: "missing update", method: http.MethodPut},
		{name: "stale delete", method: http.MethodDelete, seed: true},
		{name: "missing delete", method: http.MethodDelete},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effects := newBusinessEventEffects()
			if test.seed {
				effects.records["3mzzzzzzzzzzz"] = map[string]any{
					"$type": "social.craftsky.business.event", "createdAt": "2026-09-01T12:34:56Z",
				}
				effects.cids["3mzzzzzzzzzzz"] = businessEventCID1
			}
			factory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			}
			handler := api.PutBusinessEventHandler(factory, func() time.Time { return businessEventNow })
			body := validBusinessEventBody(false)
			if test.method == http.MethodDelete {
				handler = api.DeleteBusinessEventHandler(factory)
				body = ""
			}
			response := serveBusinessEventRequest(t, handler, test.method,
				"/v1/events/did:plc:owner/3mzzzzzzzzzzz", body, businessEventCID2,
				"did:plc:owner", "3mzzzzzzzzzzz")
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			assertBusinessEventError(t, response, "pds_record_conflict")
			if len(effects.reads) != 1 || len(effects.puts) != 0 || len(effects.deletes) != 0 {
				t.Fatalf("conflict effects: reads=%d puts=%d deletes=%d", len(effects.reads), len(effects.puts), len(effects.deletes))
			}
		})
	}
}

func validBusinessEventBody(withImage bool) string {
	image := ""
	if withImage {
		image = `,"image":{"image":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"image/webp","size":1024},"alt":"Event poster","aspectRatio":{"width":4,"height":3}}`
	}
	return `{"name":"Fiber Fair","startsAt":"2026-09-02T10:00:00Z","endsAt":"2026-09-02T18:00:00Z","roles":["organizer","vendor"],"mode":"hybrid","status":"scheduled","timeZone":"UTC","isAllDay":false,"summary":"Meet our makers","venueName":"Guild Hall","eventUri":"https://event.example/details","registrationUri":"https://tickets.example/register"` + image + `}`
}

func serveBusinessEventRequest(
	t *testing.T,
	handler http.Handler,
	method, target, body, ifMatch, did, rkey string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	if did != "" {
		request.SetPathValue("did", did)
	}
	if rkey != "" {
		request.SetPathValue("rkey", rkey)
	}
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-event-request")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type businessEventMutationResponse struct {
	DID  string `json:"did"`
	Rkey string `json:"rkey"`
	URI  string `json:"uri"`
	CID  string `json:"cid"`
}

func decodeBusinessEventMutationResponse(t *testing.T, response *httptest.ResponseRecorder) businessEventMutationResponse {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for key := range raw {
		if strings.Contains(key, "_") {
			t.Fatalf("response key %q is not camelCase", key)
		}
	}
	var body businessEventMutationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func assertBusinessEventError(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body envelope.Error
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v; body = %s", err, response.Body.String())
	}
	if body.Error != want || body.RequestID != "business-event-request" {
		t.Fatalf("error response = %+v, want error %q", body, want)
	}
}

func assertBusinessEventEffectScope(
	t *testing.T,
	owner syntax.DID,
	generation int64,
	expected []ownerlifecycle.ExpectedOwner,
	collection syntax.NSID,
	rkey syntax.RecordKey,
) {
	t.Helper()
	if owner != "did:plc:owner" || generation != 7 ||
		len(expected) != 1 || expected[0] != (ownerlifecycle.ExpectedOwner{Owner: owner, Generation: 7}) ||
		collection != "social.craftsky.business.event" || rkey == "" {
		t.Fatalf("effect scope = owner %q generation %d expected %+v collection %q rkey %q",
			owner, generation, expected, collection, rkey)
	}
}

var _ pdseffects.EffectExecutor = (*businessEventEffects)(nil)
