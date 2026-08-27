package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/followergrowth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

const followerGrowthRouteLifecycleDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT PRIMARY KEY,
    record_cid TEXT NOT NULL
);
INSERT INTO craftsky_profiles(did,record_cid) VALUES
    ('did:plc:alice','alice-cid'),
    ('did:plc:bob','bob-cid');
CREATE TABLE atproto_follows (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(did,rkey),
    UNIQUE(did,subject_did)
);
CREATE TABLE owner_lifecycles (
    owner_did TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    generation BIGINT NOT NULL,
    auth_epoch BIGINT NOT NULL,
    transition_reason TEXT NOT NULL,
    transitioned_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    purge_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
INSERT INTO owner_lifecycles(
    owner_did,state,generation,auth_epoch,transition_reason,
    transitioned_at,created_at,updated_at
) VALUES
    ('did:plc:alice','active',1,1,'test',now(),now(),now()),
    ('did:plc:bob','active',1,1,'test',now(),now(),now());
`

func TestFollowerGrowthRoutePolicyAndHandler(t *testing.T) {
	policy := mustPolicy("GET", "/v1/profiles/me/follower-growth")
	if policy.AccessClass != AccessCurrentMember || policy.RateClass != RateClassRead || policy.BodyKind != BodyNoBody {
		t.Fatalf("route policy = %+v, want current-member read with no body", policy)
	}

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	availableFrom := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	latest := followergrowth.Snapshot{
		Date:          time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC),
		FollowerCount: 42,
		CapturedAt:    time.Date(2026, time.August, 24, 0, 0, 2, 0, time.UTC),
	}
	reader := &recordingGrowthReader{history: followergrowth.History{
		AvailableFrom: &availableFrom,
		Latest:        &latest,
		Snapshots:     []followergrowth.Snapshot{latest},
	}}
	handler := api.GetFollowerGrowthHandler(reader, func() time.Time { return now })

	t.Run("supported owner request succeeds", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/follower-growth?period=7d", nil)
		request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:alice")))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
		}
		if reader.owner != "did:plc:alice" {
			t.Fatalf("reader owner = %q, want authenticated Alice", reader.owner)
		}
		if !reader.dateRange.Start.Equal(time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("range start = %s, want 2026-08-19", reader.dateRange.Start)
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body["period"] != "7d" || body["latestFollowerCount"] != float64(42) {
			t.Fatalf("response = %v, want Alice's 7d persisted history", body)
		}
	})

	for _, target := range []string{
		"/v1/profiles/me/follower-growth",
		"/v1/profiles/me/follower-growth?period=bad",
		"/v1/profiles/me/follower-growth?period=7d&period=30d",
		"/v1/profiles/me/follower-growth?period=7d&did=did:plc:bob",
		"/v1/profiles/me/follower-growth?period=7d&handle=bob.example",
		"/v1/profiles/me/follower-growth?period=7d&timezone=Europe/London",
		"/v1/profiles/me/follower-growth?period=7d&cursor=opaque",
		"/v1/profiles/me/follower-growth?period=7d&range=custom",
	} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:alice")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			assertGrowthError(t, response, http.StatusBadRequest, "invalid_period")
		})
	}

	t.Run("missing authenticated owner fails closed", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/profiles/me/follower-growth?period=30d", nil))
		assertGrowthError(t, response, http.StatusInternalServerError, "internal_error")
	})

	t.Run("store failure uses canonical error", func(t *testing.T) {
		failing := api.GetFollowerGrowthHandler(
			&recordingGrowthReader{err: errors.New("private database details")},
			func() time.Time { return now },
		)
		request := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/follower-growth?period=30d", nil)
		request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:alice")))
		response := httptest.NewRecorder()
		failing.ServeHTTP(response, request)
		assertGrowthError(t, response, http.StatusInternalServerError, "internal_error")
		if response.Body.String() == "private database details" {
			t.Fatal("response exposed store error")
		}
	})
}

func TestFollowerGrowthProductionRouteEnforcesCurrentOwnerBoundary(t *testing.T) {
	pool := testdb.WithSchema(t, followerGrowthRouteLifecycleDDL)
	latest := followergrowth.Snapshot{
		Date:          time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC),
		FollowerCount: 42,
		CapturedAt:    time.Date(2026, time.August, 24, 0, 0, 2, 0, time.UTC),
	}
	reader := &ownerGrowthReader{histories: map[syntax.DID]followergrowth.History{
		"did:plc:alice": {AvailableFrom: &latest.Date, Latest: &latest, Snapshots: []followergrowth.Snapshot{latest}},
		"did:plc:bob":   {},
	}}
	deps := testDeps()
	deps.DB = pool
	deps.AuthService = &auth.MockAuthService{DefaultDID: "did:plc:alice"}
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	deps.FollowerGrowth = reader
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func(target string, authenticated, device bool, devDID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if authenticated {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		if device {
			req.Header.Set("X-Craftsky-Device-Id", "growth-test-device")
		}
		if devDID != "" {
			req.Header.Set("X-Dev-DID", devDID)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}

	if got := request("/v1/profiles/me/follower-growth?period=7d", false, true, ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401; body=%s", got.Code, got.Body.String())
	}
	if got := request("/v1/profiles/me/follower-growth?period=7d", true, false, ""); got.Code != http.StatusBadRequest {
		t.Fatalf("missing-device status = %d, want 400; body=%s", got.Code, got.Body.String())
	}
	if got := request("/v1/profiles/me/follower-growth?period=7d", true, true, "did:plc:departed"); got.Code != http.StatusNotFound {
		t.Fatalf("departed-member status = %d, want 404; body=%s", got.Code, got.Body.String())
	}
	if got := request("/v1/profiles/did:plc:bob/follower-growth?period=7d", true, true, ""); got.Code != http.StatusNotFound {
		t.Fatalf("arbitrary-profile status = %d, want 404; body=%s", got.Code, got.Body.String())
	}

	for _, period := range []string{"7d", "30d", "1y"} {
		got := request("/v1/profiles/me/follower-growth?period="+period, true, true, "")
		if got.Code != http.StatusOK {
			t.Fatalf("Alice %s status = %d, want 200; body=%s", period, got.Code, got.Body.String())
		}
	}
	bob := request("/v1/profiles/me/follower-growth?period=30d", true, true, "did:plc:bob")
	if bob.Code != http.StatusOK {
		t.Fatalf("Bob no-history status = %d, want 200; body=%s", bob.Code, bob.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(bob.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode Bob response: %v", err)
	}
	for _, key := range []string{"availableFrom", "latestSnapshotDate", "latestCapturedAt", "latestFollowerCount", "netChange"} {
		if body[key] != nil {
			t.Fatalf("Bob no-history %s = %v, want null", key, body[key])
		}
	}
	if len(reader.owners) != 4 || reader.owners[len(reader.owners)-1] != "did:plc:bob" {
		t.Fatalf("reader owners = %v, want three Alice requests then Bob", reader.owners)
	}
}

func TestFollowerGrowthProductionRouteUsesPersistedHistoryWithoutLiveOverlay(t *testing.T) {
	pool := testdb.WithSchema(t, followerGrowthRouteLifecycleDDL)
	migration, err := os.ReadFile("../../migrations/000060_follower_growth_snapshots.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply follower-growth migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO follower_growth_snapshots(
			profile_did,snapshot_date,follower_count,captured_at
		) VALUES('did:plc:alice','2026-08-24',8,'2026-08-24T00:00:02Z');
		INSERT INTO atproto_follows(uri,did,rkey,cid,subject_did,record,created_at)
		VALUES('at://did:plc:bob/app.bsky.graph.follow/alice','did:plc:bob','alice','live-cid','did:plc:alice','{}',now());
	`); err != nil {
		t.Fatalf("seed persisted and live counts: %v", err)
	}
	deps := testDeps()
	deps.DB = pool
	deps.AuthService = &auth.MockAuthService{DefaultDID: "did:plc:alice"}
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	deps.FollowerGrowth = followergrowth.NewStore(pool)
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/follower-growth?period=7d", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Craftsky-Device-Id", "growth-live-overlay-test")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["latestFollowerCount"] != float64(8) || body["latestSnapshotDate"] != "2026-08-24" {
		t.Fatalf("response latest metadata = %v/%v, want persisted 8 on 2026-08-24", body["latestFollowerCount"], body["latestSnapshotDate"])
	}
	var liveCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT follower_count FROM craftsky_profile_follower_counts WHERE profile_did='did:plc:alice'
	`).Scan(&liveCount); err != nil {
		t.Fatal(err)
	}
	if liveCount != 1 {
		t.Fatalf("live count = %d, want fixture to differ from persisted count 8", liveCount)
	}
}

type recordingGrowthReader struct {
	history   followergrowth.History
	err       error
	owner     syntax.DID
	dateRange followergrowth.DateRange
}

type ownerGrowthReader struct {
	histories map[syntax.DID]followergrowth.History
	owners    []syntax.DID
}

func (r *ownerGrowthReader) Read(
	_ context.Context,
	owner syntax.DID,
	_ followergrowth.DateRange,
) (followergrowth.History, error) {
	r.owners = append(r.owners, owner)
	return r.histories[owner], nil
}

func (r *recordingGrowthReader) Read(
	_ context.Context,
	owner syntax.DID,
	dateRange followergrowth.DateRange,
) (followergrowth.History, error) {
	r.owner = owner
	r.dateRange = dateRange
	return r.history, r.err
}

func assertGrowthError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body %s", response.Code, status, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] != code || body["message"] == nil {
		t.Fatalf("error response = %v, want code %q and message", body, code)
	}
	if _, ok := body["requestId"]; !ok {
		t.Fatalf("error response missing requestId: %v", body)
	}
}
