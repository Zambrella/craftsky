package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

type fakeProfileCustomisationBatchReader struct {
	values map[syntax.DID]api.ProfileCustomisation
	calls  int
	dids   []syntax.DID
}

func (f *fakeProfileCustomisationBatchReader) ReadBatch(
	_ context.Context,
	dids []syntax.DID,
) (map[syntax.DID]api.ProfileCustomisation, error) {
	f.calls++
	f.dids = append([]syntax.DID(nil), dids...)
	return f.values, nil
}

func TestProfileCustomisationResponseBuildersIncludeCompleteDefaults(t *testing.T) {
	t.Parallel()

	profile := api.BuildProfileResponse(&api.ProfileRow{
		DID:               "did:plc:alice",
		Crafts:            []string{},
		IsCraftskyProfile: true,
	}, "alice.example", true)
	account := api.BuildProfileAccountSummary(&api.ProfileAccountRow{
		DID:               "did:plc:alice",
		IsCraftskyProfile: true,
	}, "alice.example")
	post := api.BuildPostResponse(&api.PostRow{
		DID:   "did:plc:alice",
		URI:   "at://did:plc:alice/social.craftsky.feed.post/1",
		Rkey:  "1",
		CID:   "cid",
		Text:  "text",
		Tags:  []string{},
		Langs: []string{},
	}, "alice.example")

	for name, value := range map[string]any{
		"profile": profile,
		"account": account,
		"author":  post.Author,
	} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal %s: %v", name, err)
		}
		assertCustomisationMap(t, body["customisation"], "cobalt", "medium", "none")
	}
}

func TestIdentityCustomisationHydratorDecoratesNestedMembersInOneBatch(t *testing.T) {
	ctx := context.Background()
	reader := &fakeProfileCustomisationBatchReader{values: map[syntax.DID]api.ProfileCustomisation{
		"did:plc:alice": {Colour: "teal", Border: "thick", Background: "x2"},
		"did:plc:bob":   api.DefaultProfileCustomisation,
	}}
	hydrator := api.NewIdentityCustomisationHydrator(reader)
	raw := []byte(`{
		"profile":{"did":"did:plc:alice","handle":"alice.example","avatar":"alice.png"},
		"items":[
			{"author":{"did":"did:plc:bob","handle":"bob.example","avatar":null}},
			{"author":{"did":"did:plc:alice","handle":"alice.example","avatar":"alice.png"}},
			{"author":{"did":"did:plc:outside","handle":"outside.example","avatar":"outside.png"}},
			{"availability":"blocked"},
			{"actor":{"available":false,"did":"did:plc:bob","handle":""}}
		],
		"actor":null
	}`)

	hydrated, err := hydrator.HydrateJSON(ctx, raw)
	if err != nil {
		t.Fatalf("hydrate JSON: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(hydrated, &body); err != nil {
		t.Fatalf("decode hydrated JSON: %v", err)
	}
	profile := body["profile"].(map[string]any)
	assertCustomisationMap(t, profile["customisation"], "teal", "thick", "x2")
	items := body["items"].([]any)
	bob := items[0].(map[string]any)["author"].(map[string]any)
	assertCustomisationMap(t, bob["customisation"], "cobalt", "medium", "none")
	aliceAgain := items[1].(map[string]any)["author"].(map[string]any)
	assertCustomisationMap(t, aliceAgain["customisation"], "teal", "thick", "x2")
	outside := items[2].(map[string]any)["author"].(map[string]any)
	if _, ok := outside["customisation"]; ok {
		t.Fatalf("non-member identity gained customisation: %v", outside)
	}
	if _, ok := items[3].(map[string]any)["customisation"]; ok {
		t.Fatal("stripped placeholder gained customisation")
	}
	unavailableActor := items[4].(map[string]any)["actor"].(map[string]any)
	if _, ok := unavailableActor["customisation"]; ok {
		t.Fatalf("unavailable actor gained customisation: %v", unavailableActor)
	}
}

func TestProfileCustomisationStoreReadBatchDeduplicatesAndDefaultsMembers(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read profile customisation migration: %v", err)
	}
	pool := testdb.WithSchema(t, profileCustomisationStoreTestDDL+string(migration))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES ('did:plc:alice', 'orchid', 'thin', 'skewdark')
	`); err != nil {
		t.Fatalf("seed Alice customisation: %v", err)
	}

	got, err := api.NewProfileCustomisationStore(pool).ReadBatch(ctx, []syntax.DID{
		"did:plc:alice",
		"did:plc:bob",
		"did:plc:alice",
		"did:plc:outside",
	})
	if err != nil {
		t.Fatalf("read batch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("batch size = %d, want two current members: %v", len(got), got)
	}
	if got["did:plc:alice"] != (api.ProfileCustomisation{Colour: "orchid", Border: "thin", Background: "skewdark"}) {
		t.Fatalf("Alice batch value = %+v", got["did:plc:alice"])
	}
	if got["did:plc:bob"] != api.DefaultProfileCustomisation {
		t.Fatalf("Bob batch value = %+v, want defaults", got["did:plc:bob"])
	}
}

func TestIdentityCustomisationHydratorWrapsSuccessfulJSONHandlers(t *testing.T) {
	t.Parallel()

	reader := &fakeProfileCustomisationBatchReader{values: map[syntax.DID]api.ProfileCustomisation{
		"did:plc:alice": {Colour: "rose", Border: "thin", Background: "dotcrossdark"},
	}}
	hydrator := api.NewIdentityCustomisationHydrator(reader)
	handler := hydrator.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"author":{"did":"did:plc:alice","handle":"alice.example"}}`))
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/example", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if reader.calls != 1 || len(reader.dids) != 1 || reader.dids[0] != "did:plc:alice" {
		t.Fatalf("batch calls/dids = %d/%v", reader.calls, reader.dids)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	author := body["author"].(map[string]any)
	assertCustomisationMap(t, author["customisation"], "rose", "thin", "dotcrossdark")
}

func assertCustomisationMap(t *testing.T, raw any, colour, border, background string) {
	t.Helper()
	value, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("customisation = %#v, want object", raw)
	}
	if value["colour"] != colour || value["profileBorder"] != border || value["profileBackground"] != background {
		t.Fatalf("customisation = %v, want %s/%s/%s", value, colour, border, background)
	}
}
