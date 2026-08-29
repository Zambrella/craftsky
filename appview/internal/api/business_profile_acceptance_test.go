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
	"social.craftsky/appview/internal/testdb"
)

type countingBusinessProfileReader struct {
	store            *business.Store
	accountTypeReads int
	profileReads     int
}

func (reader *countingBusinessProfileReader) ReadAccountType(ctx context.Context, did syntax.DID) (business.AccountType, error) {
	reader.accountTypeReads++
	return reader.store.ReadAccountType(ctx, did)
}

func (reader *countingBusinessProfileReader) ReadEligibleProfile(ctx context.Context, did syntax.DID) (*business.ProfileView, error) {
	reader.profileReads++
	return reader.store.ReadEligibleProfile(ctx, did)
}

func TestBusinessDeclarationPresentation(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			record_cid TEXT NOT NULL
		);
		CREATE TABLE craftsky_account_types (
			owner_did TEXT PRIMARY KEY,
			account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
		);
		CREATE TABLE craftsky_business_profiles (
			owner_did TEXT PRIMARY KEY,
			uri TEXT NOT NULL UNIQUE,
			cid TEXT NOT NULL,
			raw_record JSONB NOT NULL,
			source_revision TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	ctx := context.Background()
	did := syntax.DID("did:plc:business")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, did); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, did); err != nil {
		t.Fatalf("seed account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES ($1, 'at://did:plc:business/social.craftsky.business.profile/self', 'business-cid', $2, '3mprofile00001')
	`, did, `{
		"$type":"social.craftsky.business.profile",
		"tagline":"Small-batch colour, made slowly.",
		"hoursNote":"Studio visits by appointment only.",
		"serviceArea":"Ships throughout the UK; local pickup in Leeds.",
		"location":{"country":"gb","locality":"Leeds","region":"West Yorkshire","street":"Not public"},
		"primaryAction":{"type":"shop","destination":"https://shop.example/products?from=craftsky"},
		"products":[
			{"title":"First skein","uri":"https://shop.example/first","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":101},"alt":"Rust-coloured yarn"},"price":{"amount":"12.5","currency":"GBP"}},
			{"title":"Second skein","uri":"https://shop.example/second","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/png","size":102}}},
			{"title":"Third skein","uri":"https://shop.example/third","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/webp","size":103},"alt":"Blue yarn"}},
			{"title":"Fourth skein","uri":"https://shop.example/fourth","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":104}},"price":{"amount":"20","currency":"GBP"}}
		]
	}`); err != nil {
		t.Fatalf("seed eligible business profile: %v", err)
	}

	profileStore := &fakeStore{row: &api.ProfileRow{
		DID:               did.String(),
		Crafts:            []string{},
		CreatedAt:         time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC),
		IsCraftskyProfile: true,
	}}
	businessProfiles := &countingBusinessProfileReader{store: business.NewStore(pool)}
	handler := api.GetProfileHandler(
		profileStore,
		businessProfiles,
		fakeResolver{handleFor: "business.example"},
		nilLogger(),
	)

	response := serveBusinessProfileRead(handler, did)
	if response.Code != http.StatusOK {
		t.Fatalf("eligible profile status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode eligible profile: %v", err)
	}
	if body["accountType"] != "business" {
		t.Fatalf("accountType = %v, want business", body["accountType"])
	}
	if _, exists := body["events"]; exists {
		t.Fatalf("REG-003: main profile response embedded events: %s", response.Body.String())
	}
	if businessProfiles.accountTypeReads != 1 || businessProfiles.profileReads != 1 {
		t.Fatalf("eligible business reads = account type %d, profile %d, want 1 each", businessProfiles.accountTypeReads, businessProfiles.profileReads)
	}
	presentation, ok := body["business"].(map[string]any)
	if !ok {
		t.Fatalf("business = %#v, want object", body["business"])
	}
	for field, want := range map[string]string{
		"tagline":     "Small-batch colour, made slowly.",
		"hoursNote":   "Studio visits by appointment only.",
		"serviceArea": "Ships throughout the UK; local pickup in Leeds.",
	} {
		if presentation[field] != want {
			t.Errorf("business.%s = %v, want %q", field, presentation[field], want)
		}
	}
	location, ok := presentation["location"].(map[string]any)
	if !ok || location["country"] != "GB" || location["locality"] != "Leeds" || len(location) != 2 {
		t.Errorf("business.location = %#v, want canonical country/locality only", presentation["location"])
	}
	action, ok := presentation["primaryAction"].(map[string]any)
	if !ok || action["type"] != "shop" || action["destination"] != "https://shop.example/products?from=craftsky" {
		t.Errorf("business.primaryAction = %#v", presentation["primaryAction"])
	}
	products, ok := presentation["products"].([]any)
	if !ok || len(products) != 4 {
		t.Fatalf("business.products = %#v, want four cards", presentation["products"])
	}
	for index, wantTitle := range []string{"First skein", "Second skein", "Third skein", "Fourth skein"} {
		product, ok := products[index].(map[string]any)
		if !ok || product["title"] != wantTitle || product["uri"] != "https://shop.example/"+[]string{"first", "second", "third", "fourth"}[index] {
			t.Errorf("business.products[%d] = %#v", index, products[index])
		}
		if _, ok := product["image"].(map[string]any); !ok {
			t.Errorf("business.products[%d].image = %#v, want object", index, product["image"])
		}
	}
	first := products[0].(map[string]any)
	if first["image"].(map[string]any)["alt"] != "Rust-coloured yarn" {
		t.Errorf("first product alt = %#v", first["image"])
	}
	price, ok := first["price"].(map[string]any)
	if !ok || price["amount"] != "12.5" || price["currency"] != "GBP" || len(price) != 2 {
		t.Errorf("first product price = %#v, want seller-authored amount/currency only", first["price"])
	}
	assertNoBusinessClaimsOrEvents(t, body)

	profileStore.row.Blocking = true
	blocked := serveBusinessProfileRead(handler, did)
	var blockedBody map[string]any
	if err := json.Unmarshal(blocked.Body.Bytes(), &blockedBody); err != nil {
		t.Fatalf("decode blocked profile: %v", err)
	}
	if _, exists := blockedBody["accountType"]; exists {
		t.Errorf("blocked profile exposed accountType: %s", blocked.Body.String())
	}
	if _, exists := blockedBody["business"]; exists {
		t.Errorf("blocked profile exposed business: %s", blocked.Body.String())
	}
	if businessProfiles.accountTypeReads != 1 || businessProfiles.profileReads != 1 {
		t.Errorf("blocked profile performed business reads: account type %d, profile %d", businessProfiles.accountTypeReads, businessProfiles.profileReads)
	}
}

func serveBusinessProfileRead(handler http.Handler, did syntax.DID) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/profiles/@"+did.String(), nil)
	request.SetPathValue("handleOrDid", did.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertNoBusinessClaimsOrEvents(t *testing.T, body map[string]any) {
	t.Helper()
	for _, field := range []string{
		"events", "event", "disclaimer", "inventory", "availability", "synchronization",
		"shipping", "tax", "checkout", "accuracy",
	} {
		if jsonContainsKey(body, field) {
			t.Errorf("profile response exposed forbidden %q field: %#v", field, body)
		}
	}
}

func jsonContainsKey(value any, target string) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == target || jsonContainsKey(child, target) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if jsonContainsKey(child, target) {
				return true
			}
		}
	}
	return false
}
