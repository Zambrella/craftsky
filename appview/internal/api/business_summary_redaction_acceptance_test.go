package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

type summaryAccountTypeReader struct {
	values map[syntax.DID]business.AccountType
	calls  [][]syntax.DID
}

func TestBlockedBusinessEventListAndDirectReadAreIndistinguishableFromMissing(t *testing.T) {
	pool := testdb.WithSchema(t, businessEventStoreDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:blocked-business")
	visitor := syntax.DID("did:plc:event-visitor")
	rkey := syntax.RecordKey("3msblocked001")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed blocked business profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed blocked business account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks(uri, blocker_did, subject_did)
		VALUES ('at://did:plc:blocked-business/app.bsky.graph.block/visitor', $1, $2)
	`, owner, visitor); err != nil {
		t.Fatalf("seed blocked business relationship: %v", err)
	}
	seedEventFixture(t, pool, eventFixture{
		Owner: owner, Rkey: rkey, Name: "Private through block",
		StartsAt: asOf.Add(time.Hour), EndsAt: asOf.Add(2 * time.Hour),
	})
	store := business.NewStore(pool)

	list := api.GetProfileBusinessEventsHandler(store, nil, testEventCursorCodec(t), func() time.Time { return asOf })
	listRequest := httptest.NewRequest(http.MethodGet, "/v1/profiles/"+owner.String()+"/events", nil)
	listRequest.SetPathValue("handleOrDid", owner.String())
	listRequest = listRequest.WithContext(middleware.WithDID(listRequest.Context(), visitor))
	listResponse := httptest.NewRecorder()
	list.ServeHTTP(listResponse, listRequest)
	var page api.BusinessEventPage
	if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode blocked event list: %v; body=%s", err, listResponse.Body.String())
	}
	if listResponse.Code != http.StatusOK || len(page.Items) != 0 {
		t.Fatalf("blocked event list status/items = %d/%d, want 200/0", listResponse.Code, len(page.Items))
	}

	direct := api.GetBusinessEventHandler(store, func() time.Time { return asOf })
	directRequest := httptest.NewRequest(http.MethodGet, "/v1/events/"+owner.String()+"/"+rkey.String(), nil)
	directRequest.SetPathValue("did", owner.String())
	directRequest.SetPathValue("rkey", rkey.String())
	directRequest = directRequest.WithContext(middleware.WithDID(directRequest.Context(), visitor))
	directResponse := httptest.NewRecorder()
	direct.ServeHTTP(directResponse, directRequest)
	var eventError struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(directResponse.Body.Bytes(), &eventError); err != nil {
		t.Fatalf("decode blocked direct event: %v; body=%s", err, directResponse.Body.String())
	}
	if directResponse.Code != http.StatusNotFound || eventError.Error != "event_not_found" {
		t.Fatalf("blocked direct event status/error = %d/%q, want 404/event_not_found", directResponse.Code, eventError.Error)
	}
}

func (r *summaryAccountTypeReader) ReadAccountTypes(_ context.Context, dids []syntax.DID) (map[syntax.DID]business.AccountType, error) {
	r.calls = append(r.calls, append([]syntax.DID(nil), dids...))
	return r.values, nil
}

func TestBusinessSummaryRedactionAcceptance(t *testing.T) {
	reader := &summaryAccountTypeReader{values: map[syntax.DID]business.AccountType{
		"did:plc:visible": business.AccountTypeBusiness,
		"did:plc:blocked": business.AccountTypeBusiness,
	}}
	hydrator := api.NewIdentityAccountTypeHydrator(reader)
	raw := []byte(`{
		"profile":{"did":"did:plc:visible","handle":"visible.test"},
		"search":{"did":"did:plc:blocked","handle":"blocked.test","blocking":true,"accountType":"business","business":{"tagline":"secret"}},
		"relationship":{"did":"did:plc:blocked-by","handle":"blocked-by.test","blockedBy":true,"accountType":"business","business":{"location":{"locality":"secret"}}},
		"notification":{"available":false,"did":"did:plc:unavailable","handle":"unavailable.test","accountType":"business","business":{"actions":["secret"]}}
	}`)

	hydrated, err := hydrator.HydrateJSON(context.Background(), raw)
	if err != nil {
		t.Fatalf("hydrate summaries: %v", err)
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(hydrated, &body); err != nil {
		t.Fatalf("decode hydrated summaries: %v", err)
	}
	if got := body["profile"]["accountType"]; got != "business" {
		t.Fatalf("visible accountType = %v, want business", got)
	}
	for _, key := range []string{"search", "relationship", "notification"} {
		if _, exists := body[key]["accountType"]; exists {
			t.Errorf("%s retained accountType", key)
		}
		if _, exists := body[key]["business"]; exists {
			t.Errorf("%s retained business", key)
		}
	}
	if len(reader.calls) != 1 || len(reader.calls[0]) != 1 || reader.calls[0][0] != "did:plc:visible" {
		t.Fatalf("account-type batch calls = %v, want one visible DID", reader.calls)
	}
}
