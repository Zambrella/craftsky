package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessRoutesDispatchOnlyExactMethodsAndPathsUnderAuth(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	exact := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/v1/profiles/me/account-type"},
		{http.MethodPut, "/v1/profiles/me/business"},
		{http.MethodDelete, "/v1/profiles/me/business"},
		{http.MethodGet, "/v1/profiles/alice.example/events"},
		{http.MethodPost, "/v1/events"},
		{http.MethodGet, "/v1/events"},
		{http.MethodGet, "/v1/events/did:plc:alice/3mzzzzzzzzzzz"},
		{http.MethodPut, "/v1/events/did:plc:alice/3mzzzzzzzzzzz"},
		{http.MethodDelete, "/v1/events/did:plc:alice/3mzzzzzzzzzzz"},
		{http.MethodPost, "/v1/events/did:plc:alice/3mzzzzzzzzzzz/reports"},
	}
	for _, target := range exact {
		t.Run(target.method+" "+target.path, func(t *testing.T) {
			request := httptest.NewRequest(target.method, target.path, nil)
			_, pattern := mux.Handler(request)
			if pattern == "" || pattern == "/" {
				t.Fatalf("route is not registered")
			}
		})
	}

	nearMisses := []struct {
		method string
		path   string
	}{
		{http.MethodPatch, "/v1/profiles/me/account-type"},
		{http.MethodPut, "/v1/profiles/me/account-types"},
		{http.MethodPatch, "/v1/profiles/me/business"},
		{http.MethodGet, "/v1/profiles/alice.example/event"},
		{http.MethodPatch, "/v1/events"},
		{http.MethodPatch, "/v1/events/did:plc:alice/3mzzzzzzzzzzz"},
		{http.MethodPost, "/v1/events/did:plc:alice/3mzzzzzzzzzzz/report"},
	}
	for _, target := range nearMisses {
		t.Run("near miss "+target.method+" "+target.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(target.method, target.path, nil))
			if response.Code != http.StatusNotFound && response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 404 or 405; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestBusinessRoutesEnforceMemberAndCursorContracts(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY, record_cid TEXT NOT NULL);
		INSERT INTO craftsky_profiles(did, record_cid) VALUES ('did:plc:test', 'profile-cid');
		CREATE TABLE craftsky_account_types (
			owner_did TEXT PRIMARY KEY REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
			account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
		);
	`)
	deps := testDeps()
	deps.DB = pool
	deps.BusinessStore = business.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func(method, path, body string, auth, device bool, devDID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if auth {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		if device {
			req.Header.Set("X-Craftsky-Device-Id", "test-device")
		}
		if devDID != "" {
			req.Header.Set("X-Dev-DID", devDID)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	departed := request(http.MethodGet, "/v1/events?cursor=bad", "", true, true, "did:plc:departed")
	if departed.Code != http.StatusNotFound || !strings.Contains(departed.Body.String(), `"error":"profile_not_found"`) {
		t.Fatalf("departed member = %d %s", departed.Code, departed.Body.String())
	}
	assertBusinessRouteErrorEnvelope(t, departed)

	badCursor := request(http.MethodGet, "/v1/events?cursor=bad", "", true, true, "")
	if badCursor.Code != http.StatusBadRequest || !strings.Contains(badCursor.Body.String(), `"error":"invalid_cursor"`) {
		t.Fatalf("bad cursor = %d %s", badCursor.Code, badCursor.Body.String())
	}
	assertBusinessRouteErrorEnvelope(t, badCursor)
}

func TestBusinessRouteRejectsTamperedValidCursor(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY, record_cid TEXT NOT NULL);
		INSERT INTO craftsky_profiles(did, record_cid) VALUES ('did:plc:test', 'profile-cid');
		CREATE TABLE craftsky_account_types (
			owner_did TEXT PRIMARY KEY,
			account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
		);
		CREATE TABLE craftsky_business_events (
			uri TEXT PRIMARY KEY,
			owner_did TEXT NOT NULL,
			rkey TEXT NOT NULL,
			cid TEXT NOT NULL,
			raw_record JSONB NOT NULL,
			source_revision TEXT NOT NULL,
			starts_at TIMESTAMPTZ NOT NULL,
			ends_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			status TEXT,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (owner_did, rkey)
		);
		CREATE TABLE moderation_outputs (
			id TEXT PRIMARY KEY,
			source_did TEXT NOT NULL,
			subject_type TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			subject_uri TEXT,
			value TEXT NOT NULL,
			action TEXT NOT NULL,
			expires_at TIMESTAMPTZ,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	ctx := context.Background()
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 21; index++ {
		startsAt := asOf.Add(time.Duration(index+1) * time.Hour)
		rkey := fmt.Sprintf("3mroutecursor%02d", index)
		uri := "at://did:plc:test/social.craftsky.business.event/" + rkey
		raw := fmt.Sprintf(`{"$type":"social.craftsky.business.event","name":"Route event %d","startsAt":%q,"endsAt":%q,"roles":["vendor"],"createdAt":%q}`,
			index, startsAt.Format(time.RFC3339), startsAt.Add(time.Hour).Format(time.RFC3339), asOf.Format(time.RFC3339))
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_business_events
				(uri,owner_did,rkey,cid,raw_record,source_revision,starts_at,ends_at,created_at)
			VALUES ($1,'did:plc:test',$2,$3,$4::jsonb,$5,$6,$7,$8)
		`, uri, rkey, fmt.Sprintf("bafy-route-%02d", index), raw, fmt.Sprintf("3mrevision%02d", index), startsAt, startsAt.Add(time.Hour), asOf); err != nil {
			t.Fatalf("seed route event %d: %v", index, err)
		}
	}

	codec, err := api.NewEventCursorCodec([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatalf("new cursor codec: %v", err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.BusinessStore = business.NewStore(pool)
	deps.EventCursorCodec = codec
	deps.Now = func() time.Time { return asOf }
	mux := http.NewServeMux()
	AddRoutes(ctx, mux, deps)
	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Craftsky-Device-Id", "test-device")
		req.Header.Set("X-Dev-DID", "did:plc:test")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	first := request("/v1/events")
	if first.Code != http.StatusOK {
		t.Fatalf("first page status = %d, body = %s", first.Code, first.Body.String())
	}
	var page struct {
		Cursor string `json:"cursor"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page); err != nil || page.Cursor == "" {
		t.Fatalf("first page cursor = %q, error %v; body=%s", page.Cursor, err, first.Body.String())
	}
	tampered := page.Cursor[:len(page.Cursor)-1] + "A"
	if strings.HasSuffix(page.Cursor, "A") {
		tampered = page.Cursor[:len(page.Cursor)-1] + "B"
	}
	response := request("/v1/events?cursor=" + url.QueryEscape(tampered))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_cursor"`) {
		t.Fatalf("tampered cursor = %d %s", response.Code, response.Body.String())
	}
	assertBusinessRouteErrorEnvelope(t, response)
}

func assertBusinessRouteErrorEnvelope(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v; body=%s", err, response.Body.String())
	}
	for _, key := range []string{"error", "message", "requestId"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("error body missing %q: %s", key, response.Body.String())
		}
	}
	for _, key := range []string{"request_id", "requestID"} {
		if _, ok := body[key]; ok {
			t.Fatalf("error body contains non-camelCase %q: %s", key, response.Body.String())
		}
	}
}
