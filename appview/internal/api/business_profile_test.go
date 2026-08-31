package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	businessProfileCID1 = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	businessProfileCID2 = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fa"
)

type businessProfileEffects struct {
	record  map[string]any
	cid     syntax.CID
	reads   []pdseffects.ReadRecordRequest
	puts    []pdseffects.PutRecordRequest
	deletes []pdseffects.DeleteRecordRequest
}

func (effects *businessProfileEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if len(targets) != 0 {
		return nil, errors.New("business profile effects must be owner-only")
	}
	return []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:owner", Generation: ownerGeneration}}, nil
}

func (effects *businessProfileEffects) ReadRecord(
	_ context.Context,
	request pdseffects.ReadRecordRequest,
	out any,
) (syntax.CID, error) {
	effects.reads = append(effects.reads, request)
	if effects.record == nil {
		return "", auth.ErrRecordNotFound
	}
	encoded, _ := json.Marshal(effects.record)
	switch target := out.(type) {
	case *json.RawMessage:
		*target = append((*target)[:0], encoded...)
	case *map[string]any:
		if err := json.Unmarshal(encoded, target); err != nil {
			return "", err
		}
	default:
		return "", errors.New("unexpected business profile read output")
	}
	return effects.cid, nil
}

func (effects *businessProfileEffects) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.puts = append(effects.puts, request)
	if request.ExpectedCID == "*" {
		if effects.record != nil {
			return pdseffects.RecordResult{}, auth.ErrRecordSwapConflict
		}
	} else if effects.record == nil || request.ExpectedCID != effects.cid {
		return pdseffects.RecordResult{}, auth.ErrRecordSwapConflict
	}
	encoded, _ := json.Marshal(request.Record)
	effects.record = nil
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	if err := decoder.Decode(&effects.record); err != nil {
		return pdseffects.RecordResult{}, err
	}
	if effects.cid == "" {
		effects.cid = businessProfileCID1
	} else {
		effects.cid = businessProfileCID2
	}
	return pdseffects.RecordResult{
		URI: "at://did:plc:owner/social.craftsky.business.profile/self",
		CID: effects.cid,
	}, nil
}

func (effects *businessProfileEffects) DeleteRecord(
	_ context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.deletes = append(effects.deletes, request)
	if effects.record == nil || request.ExpectedCID != effects.cid {
		return pdseffects.RecordResult{}, auth.ErrRecordSwapConflict
	}
	effects.record = nil
	effects.cid = ""
	return pdseffects.RecordResult{URI: "at://did:plc:owner/social.craftsky.business.profile/self"}, nil
}

func (*businessProfileEffects) UploadBlob(
	context.Context,
	pdseffects.UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected business profile blob upload")
}

func TestBusinessProfileConditionalCreateReplaceAndDelete(t *testing.T) {
	t.Parallel()
	effects := &businessProfileEffects{}
	var factoryOwners []syntax.DID
	var factorySessions []string
	factory := func(_ context.Context, owner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		factoryOwners = append(factoryOwners, owner)
		factorySessions = append(factorySessions, sessionID)
		return effects, nil
	}
	put := api.PutBusinessProfileHandler(factory)
	remove := api.DeleteBusinessProfileHandler(factory)

	created := serveBusinessProfileRequest(t, put, http.MethodPut, `{}`, "*")
	if created.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	assertBusinessProfileCID(t, created, businessProfileCID1)
	if len(effects.puts) != 1 || effects.puts[0].ExpectedCID != "*" {
		t.Fatalf("conditional create = %+v", effects.puts)
	}
	assertBusinessProfileEffectScope(t, effects.reads[0], effects.puts[0])

	// Simulate an independently authored extension on the authoritative PDS
	// record before the owner's next full replacement.
	effects.record["hoursNote"] = "old hours"
	effects.record["com.example.extension"] = map[string]any{
		"preserve": true,
		"sequence": json.Number("9007199254740993"),
	}
	replaced := serveBusinessProfileRequest(
		t,
		put,
		http.MethodPut,
		`{"tagline":"Owner replacement","businessTypes":["teacher","dyer"]}`,
		businessProfileCID1,
	)
	if replaced.Code != http.StatusOK {
		t.Fatalf("replace status = %d, body = %s", replaced.Code, replaced.Body.String())
	}
	assertBusinessProfileCID(t, replaced, businessProfileCID2)
	if len(effects.puts) != 2 || effects.puts[1].ExpectedCID != businessProfileCID1 {
		t.Fatalf("conditional replacement = %+v", effects.puts)
	}
	replacementRecord, err := json.Marshal(effects.puts[1].Record)
	if err != nil {
		t.Fatalf("marshal replacement record: %v", err)
	}
	if !strings.Contains(string(replacementRecord), `"sequence":9007199254740993`) {
		t.Fatalf("large unknown extension integer lost precision: %s", replacementRecord)
	}
	if effects.record["$type"] != "social.craftsky.business.profile" ||
		effects.record["tagline"] != "Owner replacement" || effects.record["hoursNote"] != nil {
		t.Fatalf("known-field replacement = %#v", effects.record)
	}
	types, ok := effects.record["businessTypes"].([]any)
	if !ok || len(types) != 2 || types[0] != "dyer" || types[1] != "teacher" {
		t.Fatalf("canonical businessTypes = %#v", effects.record["businessTypes"])
	}
	extension, ok := effects.record["com.example.extension"].(map[string]any)
	if !ok || extension["preserve"] != true {
		t.Fatalf("preserved extension = %#v", effects.record["com.example.extension"])
	}

	deleted := serveBusinessProfileRequest(t, remove, http.MethodDelete, "", businessProfileCID2)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	assertBusinessProfileCID(t, deleted, businessProfileCID2)
	if len(effects.deletes) != 1 || effects.deletes[0].ExpectedCID != businessProfileCID2 || effects.record != nil {
		t.Fatalf("conditional delete = %+v, record = %#v", effects.deletes, effects.record)
	}
	if len(factoryOwners) != 3 || len(factorySessions) != 3 {
		t.Fatalf("factory calls = owners %v sessions %v", factoryOwners, factorySessions)
	}
	for index := range factoryOwners {
		if factoryOwners[index] != "did:plc:owner" || factorySessions[index] != "oauth-owner-session" {
			t.Fatalf("factory scope[%d] = owner %q session %q", index, factoryOwners[index], factorySessions[index])
		}
	}
}

func TestBusinessProfileCompleteKnownReplacementsPreserveUnknownExtension(t *testing.T) {
	t.Parallel()
	const productImage = `"image":{"image":{"$type":"blob","ref":{"$link":"` + businessProfileCID1 + `"},"mimeType":"image/png","size":12},"alt":"Product"}`
	tests := []struct {
		name             string
		body             string
		wantTagline      string
		wantProductTitle string
	}{
		{
			name: "detail-only",
			body: `{"businessTypes":["teacher"],"offerings":["classes"],"tagline":"Updated details",` +
				`"hoursNote":"Weekdays","location":{"country":"US","locality":"Portland"},` +
				`"primaryAction":{"type":"shop","destination":"https://example.com/shop"},` +
				`"products":[{"title":"Original product","uri":"https://example.com/original",` + productImage + `}]}`,
			wantTagline:      "Updated details",
			wantProductTitle: "Original product",
		},
		{
			name: "product-only",
			body: `{"businessTypes":["teacher"],"offerings":["classes"],"tagline":"Original details",` +
				`"hoursNote":"Weekdays","location":{"country":"US","locality":"Portland"},` +
				`"primaryAction":{"type":"shop","destination":"https://example.com/shop"},` +
				`"products":[{"title":"Updated product","uri":"https://example.com/updated",` + productImage + `}]}`,
			wantTagline:      "Original details",
			wantProductTitle: "Updated product",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			effects := &businessProfileEffects{
				cid: businessProfileCID1,
				record: map[string]any{
					"$type":         "social.craftsky.business.profile",
					"businessTypes": []any{"teacher"},
					"offerings":     []any{"classes"},
					"tagline":       "Original details",
					"hoursNote":     "Weekdays",
					"serviceArea":   "Remove on replacement",
					"location":      map[string]any{"country": "US", "locality": "Portland"},
					"primaryAction": map[string]any{"type": "shop", "destination": "https://example.com/shop"},
					"products":      []any{map[string]any{"title": "Original product", "uri": "https://example.com/original"}},
					"com.example.extension": map[string]any{
						"nested": map[string]any{"enabled": true, "sequence": json.Number("9007199254740993")},
					},
				},
			}
			handler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			})

			response := serveBusinessProfileRequest(t, handler, http.MethodPut, test.body, businessProfileCID1)
			if response.Code != http.StatusOK {
				t.Fatalf("replace status = %d, body = %s", response.Code, response.Body.String())
			}
			if effects.record["tagline"] != test.wantTagline {
				t.Errorf("tagline = %#v, want %q", effects.record["tagline"], test.wantTagline)
			}
			products, ok := effects.record["products"].([]any)
			if !ok || len(products) != 1 {
				t.Fatalf("products = %#v", effects.record["products"])
			}
			product, ok := products[0].(map[string]any)
			if !ok || product["title"] != test.wantProductTitle {
				t.Errorf("product = %#v, want title %q", products[0], test.wantProductTitle)
			}
			if _, ok := effects.record["serviceArea"]; ok {
				t.Error("omitted known serviceArea survived replacement")
			}
			var submitted map[string]any
			decoder := json.NewDecoder(strings.NewReader(test.body))
			decoder.UseNumber()
			if err := decoder.Decode(&submitted); err != nil {
				t.Fatalf("decode submitted replacement: %v", err)
			}
			for field, want := range submitted {
				gotJSON, err := json.Marshal(effects.record[field])
				if err != nil {
					t.Fatalf("marshal stored %s: %v", field, err)
				}
				wantJSON, err := json.Marshal(want)
				if err != nil {
					t.Fatalf("marshal submitted %s: %v", field, err)
				}
				if !bytes.Equal(gotJSON, wantJSON) {
					t.Errorf("submitted known field %s = %s, want %s", field, gotJSON, wantJSON)
				}
			}
			extension, ok := effects.record["com.example.extension"].(map[string]any)
			if !ok {
				t.Fatalf("unknown extension = %#v", effects.record["com.example.extension"])
			}
			nested, ok := extension["nested"].(map[string]any)
			if !ok || nested["enabled"] != true || nested["sequence"] != json.Number("9007199254740993") {
				t.Fatalf("unknown nested extension = %#v", extension)
			}
			recordJSON, err := json.Marshal(effects.puts[0].Record)
			if err != nil {
				t.Fatalf("marshal PDS replacement: %v", err)
			}
			if !strings.Contains(string(recordJSON), `"sequence":9007199254740993`) {
				t.Fatalf("large unknown extension integer lost precision: %s", recordJSON)
			}
		})
	}
}

func TestBusinessProfilePreconditionConflicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		body    string
		ifMatch string
		record  map[string]any
		cid     syntax.CID
	}{
		{name: "create missing If-Match", method: http.MethodPut, body: `{}`},
		{name: "create malformed If-Match", method: http.MethodPut, body: `{}`, ifMatch: "not-a-cid"},
		{name: "create presence conflict", method: http.MethodPut, body: `{}`, ifMatch: "*", record: map[string]any{"$type": "social.craftsky.business.profile"}, cid: businessProfileCID1},
		{name: "replace missing record", method: http.MethodPut, body: `{}`, ifMatch: businessProfileCID1},
		{name: "replace stale CID", method: http.MethodPut, body: `{}`, ifMatch: businessProfileCID2, record: map[string]any{"$type": "social.craftsky.business.profile"}, cid: businessProfileCID1},
		{name: "delete missing If-Match", method: http.MethodDelete, record: map[string]any{"$type": "social.craftsky.business.profile"}, cid: businessProfileCID1},
		{name: "delete missing record", method: http.MethodDelete, ifMatch: businessProfileCID1},
		{name: "delete stale CID", method: http.MethodDelete, ifMatch: businessProfileCID2, record: map[string]any{"$type": "social.craftsky.business.profile"}, cid: businessProfileCID1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effects := &businessProfileEffects{record: test.record, cid: test.cid}
			factory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			}
			handler := api.PutBusinessProfileHandler(factory)
			if test.method == http.MethodDelete {
				handler = api.DeleteBusinessProfileHandler(factory)
			}
			response := serveBusinessProfileRequest(t, handler, test.method, test.body, test.ifMatch)
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != "pds_record_conflict" || body.RequestID != "business-profile-request" {
				t.Fatalf("conflict envelope = %+v", body)
			}
			if len(effects.puts) != 0 || len(effects.deletes) != 0 {
				t.Fatalf("conflict mutated PDS: puts %d deletes %d", len(effects.puts), len(effects.deletes))
			}
		})
	}
}

func TestBusinessProfilePutUsesStrictCamelCaseValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "snake case", body: `{"business_types":[]}`, code: "unexpected_field"},
		{name: "unknown field", body: `{"ownerDid":"did:plc:other"}`, code: "unexpected_field"},
		{name: "invalid known value", body: `{"businessTypes":["unknown"]}`, code: "validation_failed"},
		{name: "overlong tagline", body: `{"tagline":"` + strings.Repeat("a", 101) + `"}`, code: "validation_failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effects := &businessProfileEffects{}
			handler := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			})
			response := serveBusinessProfileRequest(t, handler, http.MethodPut, test.body, "*")
			if response.Code != http.StatusBadRequest && response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != test.code {
				t.Fatalf("error = %q, want %q; body = %s", body.Error, test.code, response.Body.String())
			}
			if len(effects.reads) != 0 || len(effects.puts) != 0 {
				t.Fatalf("invalid request reached PDS: reads %d puts %d", len(effects.reads), len(effects.puts))
			}
		})
	}
}

func serveBusinessProfileRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	body string,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, "/v1/profiles/me/business", strings.NewReader(body))
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-profile-request")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertBusinessProfileCID(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body struct {
		CID string `json:"cid"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.CID != want {
		t.Fatalf("response CID = %q, want %q; body = %s", body.CID, want, response.Body.String())
	}
}

func assertBusinessProfileEffectScope(
	t *testing.T,
	read pdseffects.ReadRecordRequest,
	put pdseffects.PutRecordRequest,
) {
	t.Helper()
	wantOwner := syntax.DID("did:plc:owner")
	wantCollection := syntax.NSID("social.craftsky.business.profile")
	wantRkey := syntax.RecordKey("self")
	if read.Owner != wantOwner || read.OwnerGeneration != 7 || read.Collection != wantCollection || read.Rkey != wantRkey ||
		len(read.ExpectedOwners) != 1 || read.ExpectedOwners[0] != (ownerlifecycle.ExpectedOwner{Owner: wantOwner, Generation: 7}) {
		t.Fatalf("read scope = %+v", read)
	}
	if put.Owner != wantOwner || put.OwnerGeneration != 7 || put.Collection != wantCollection || put.Rkey != wantRkey ||
		len(put.ExpectedOwners) != 1 || put.ExpectedOwners[0] != (ownerlifecycle.ExpectedOwner{Owner: wantOwner, Generation: 7}) ||
		put.OperationID == "" || put.MutationKey != put.OperationID || !strings.Contains(put.OperationID, "business-profile-request") {
		t.Fatalf("put scope = %+v", put)
	}
}

var _ pdseffects.EffectExecutor = (*businessProfileEffects)(nil)
