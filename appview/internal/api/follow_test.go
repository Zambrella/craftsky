// appview/internal/api/follow_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

type fakeFollowGraphStore struct {
	active     *api.FollowRow
	findErr    error
	upsertErr  error
	deleteErr  error
	lastDelete string
	lastUpsert *api.FollowRow
}

func (f *fakeFollowGraphStore) FindActiveFollow(_ context.Context, _ string, _ string) (*api.FollowRow, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.active, nil
}

func (f *fakeFollowGraphStore) UpsertActive(_ context.Context, row api.FollowRow, _ json.RawMessage) error {
	f.lastUpsert = &row
	return f.upsertErr
}

func (f *fakeFollowGraphStore) DeleteActiveByURI(_ context.Context, uri string) error {
	f.lastDelete = uri
	return f.deleteErr
}

type fakeFollowPDS struct {
	createURI        syntax.ATURI
	createCID        syntax.CID
	createErr        error
	createRepo       syntax.DID
	createCollection string
	createRecord     map[string]any
	deleteErr        error
	deleteRepo       syntax.DID
	deleteCollection string
	deleteRkey       string
}

func (f *fakeFollowPDS) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", errors.New("not implemented")
}
func (f *fakeFollowPDS) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return errors.New("not implemented")
}
func (f *fakeFollowPDS) CreateRecord(_ context.Context, repo syntax.DID, collection string, record any) (syntax.ATURI, syntax.CID, error) {
	f.createRepo = repo
	f.createCollection = collection
	f.createRecord, _ = record.(map[string]any)
	if f.createErr != nil {
		return "", "", f.createErr
	}
	return f.createURI, f.createCID, nil
}
func (f *fakeFollowPDS) DeleteRecord(_ context.Context, repo syntax.DID, collection string, rkey string) error {
	f.deleteRepo = repo
	f.deleteCollection = collection
	f.deleteRkey = rkey
	return f.deleteErr
}
func (f *fakeFollowPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}

type fakeFollowProfileStore struct {
	row            *api.ProfileRow
	err            error
	lastProfileDID string
	lastViewerDID  string
}

func (f *fakeFollowProfileStore) Read(_ context.Context, profileDID string, viewerDID string) (*api.ProfileRow, error) {
	f.lastProfileDID = profileDID
	f.lastViewerDID = viewerDID
	return f.row, f.err
}

func TestFollowProfileHandler_WritesFollowRecordAndReturnsProfile(t *testing.T) {
	t.Parallel()

	graph := &fakeFollowGraphStore{}
	pds := &fakeFollowPDS{createURI: "at://did:plc:alice/app.bsky.graph.follow/f1", createCID: "bafyfollow1"}
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: true}}
	resolver := fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"}
	var gotFactorySID string

	h := api.FollowProfileHandler(
		graph,
		profiles,
		resolver,
		func(_ context.Context, _ syntax.DID, sid string) (auth.PDSClient, error) {
			gotFactorySID = sid
			return pds, nil
		},
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createRepo != "did:plc:alice" {
		t.Fatalf("CreateRecord repo = %q, want did:plc:alice", pds.createRepo)
	}
	if pds.createCollection != "app.bsky.graph.follow" {
		t.Fatalf("CreateRecord collection = %q, want app.bsky.graph.follow", pds.createCollection)
	}
	if subj, _ := pds.createRecord["subject"].(string); subj != "did:plc:bob" {
		t.Fatalf("CreateRecord subject = %q, want did:plc:bob", subj)
	}
	if graph.lastUpsert != nil {
		t.Fatal("did not expect local follow graph upsert; Tap round trip should converge state")
	}
	if gotFactorySID != "sess-alice" {
		t.Fatalf("newPDS sid = %q, want sess-alice", gotFactorySID)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["accessToken"]; ok {
		t.Fatal("response leaked accessToken")
	}
	if _, ok := body["refreshToken"]; ok {
		t.Fatal("response leaked refreshToken")
	}
	if following, ok := body["viewerIsFollowing"].(bool); !ok || !following {
		t.Fatalf("viewerIsFollowing = %v (ok=%v), want true", body["viewerIsFollowing"], ok)
	}
}

func TestUnfollowProfileHandler_DeletesActiveRecordAndReturnsProfile(t *testing.T) {
	t.Parallel()

	graph := &fakeFollowGraphStore{active: &api.FollowRow{URI: "at://did:plc:alice/app.bsky.graph.follow/f1", DID: "did:plc:alice", Rkey: "f1", SubjectDID: "did:plc:bob", CreatedAt: time.Now()}}
	pds := &fakeFollowPDS{}
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: true}}
	resolver := fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"}

	h := api.UnfollowProfileHandler(
		graph,
		profiles,
		resolver,
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteRepo != "did:plc:alice" {
		t.Fatalf("DeleteRecord repo = %q, want did:plc:alice", pds.deleteRepo)
	}
	if pds.deleteCollection != "app.bsky.graph.follow" {
		t.Fatalf("DeleteRecord collection = %q, want app.bsky.graph.follow", pds.deleteCollection)
	}
	if pds.deleteRkey != "f1" {
		t.Fatalf("DeleteRecord rkey = %q, want f1", pds.deleteRkey)
	}
	if graph.lastDelete != "" {
		t.Fatalf("did not expect local follow graph delete; got %q", graph.lastDelete)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if following, ok := body["viewerIsFollowing"].(bool); !ok || following {
		t.Fatalf("viewerIsFollowing = %v (ok=%v), want false", body["viewerIsFollowing"], ok)
	}
}

func TestFollowProfileHandler_InvalidIdentifier(t *testing.T) {
	t.Parallel()

	h := api.FollowProfileHandler(
		&fakeFollowGraphStore{},
		&fakeFollowProfileStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return &fakeFollowPDS{}, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@not-valid/follows", nil)
	req.SetPathValue("handleOrDid", "NOT VALID")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env["error"] != "invalid_identifier" {
		t.Fatalf("error code = %v, want invalid_identifier", env["error"])
	}
}

func TestFollowProfileHandler_SelfRejected(t *testing.T) {
	t.Parallel()

	h := api.FollowProfileHandler(
		&fakeFollowGraphStore{},
		&fakeFollowProfileStore{},
		fakeResolver{didFor: "did:plc:alice"},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return &fakeFollowPDS{}, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@alice.example/follows", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env["error"] != "self_follow_not_allowed" {
		t.Fatalf("error code = %v, want self_follow_not_allowed", env["error"])
	}
}

func TestUnfollowProfileHandler_NoActiveIsIdempotent(t *testing.T) {
	t.Parallel()

	graph := &fakeFollowGraphStore{active: nil}
	pds := &fakeFollowPDS{}
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: true}}
	resolver := fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"}

	h := api.UnfollowProfileHandler(
		graph,
		profiles,
		resolver,
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteRkey != "" {
		t.Fatalf("expected no DeleteRecord call, got rkey=%q", pds.deleteRkey)
	}
}

func TestFollowProfileHandler_AlreadyFollowingIsIdempotent(t *testing.T) {
	t.Parallel()

	graph := &fakeFollowGraphStore{active: &api.FollowRow{URI: "at://did:plc:alice/app.bsky.graph.follow/f1", DID: "did:plc:alice", Rkey: "f1", SubjectDID: "did:plc:bob", CreatedAt: time.Now()}}
	pds := &fakeFollowPDS{}
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: true}}
	resolver := fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"}

	h := api.FollowProfileHandler(
		graph,
		profiles,
		resolver,
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if pds.createCollection != "" {
		t.Fatalf("expected no CreateRecord call, got collection=%q", pds.createCollection)
	}
}

func TestFollowProfileHandler_AlreadyFollowingResponseDoesNotDoubleCount(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	seedCraftskyProfilesForFollowHandler(t, pool, ctx)
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:alice/app.bsky.graph.follow/f1', 'did:plc:alice', 'f1', 'cid-follow', 'did:plc:bob', '{"subject":"did:plc:bob"}', now())
	`); err != nil {
		t.Fatalf("seed follow: %v", err)
	}

	graph := api.NewFollowStore(pool)
	profiles := api.NewProfileStore(pool)
	pds := &fakeFollowPDS{}
	h := api.FollowProfileHandler(
		graph,
		profiles,
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	req = req.WithContext(middleware.WithOAuthSessionID(middleware.WithDID(req.Context(), "did:plc:alice"), "sess-alice"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCollection != "" {
		t.Fatalf("expected no CreateRecord call, got collection=%q", pds.createCollection)
	}
	var body api.ProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.ViewerIsFollowing {
		t.Fatalf("viewerIsFollowing = false, want true")
	}
	if body.FollowerCount == nil || *body.FollowerCount != 1 {
		t.Fatalf("followerCount = %v, want 1", body.FollowerCount)
	}
}

func TestUnfollowProfileHandler_ActiveResponseSubtractsBeforeTapDelete(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	seedCraftskyProfilesForFollowHandler(t, pool, ctx)
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:alice/app.bsky.graph.follow/f1', 'did:plc:alice', 'f1', 'cid-follow', 'did:plc:bob', '{"subject":"did:plc:bob"}', now())
	`); err != nil {
		t.Fatalf("seed follow: %v", err)
	}

	graph := api.NewFollowStore(pool)
	profiles := api.NewProfileStore(pool)
	pds := &fakeFollowPDS{}
	h := api.UnfollowProfileHandler(
		graph,
		profiles,
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.craftsky.social"},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@bob.craftsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "bob.craftsky.social")
	req = req.WithContext(middleware.WithOAuthSessionID(middleware.WithDID(req.Context(), "did:plc:alice"), "sess-alice"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteRkey != "f1" {
		t.Fatalf("DeleteRecord rkey = %q, want f1", pds.deleteRkey)
	}
	var body api.ProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ViewerIsFollowing {
		t.Fatalf("viewerIsFollowing = true, want false")
	}
	if body.FollowerCount == nil || *body.FollowerCount != 0 {
		t.Fatalf("followerCount = %v, want 0", body.FollowerCount)
	}
}

func seedCraftskyProfilesForFollowHandler(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	for _, did := range []string{"did:plc:alice", "did:plc:bob"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
			did,
		); err != nil {
			t.Fatalf("seed craftsky profile %s: %v", did, err)
		}
	}
}

func TestFollowProfileHandler_AllowsNonCraftskyTarget(t *testing.T) {
	t.Parallel()

	graph := &fakeFollowGraphStore{}
	pds := &fakeFollowPDS{createURI: "at://did:plc:alice/app.bsky.graph.follow/f2", createCID: "bafyfollow2"}
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:carol", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: false}}
	resolver := fakeResolver{didFor: "did:plc:carol", handleFor: "carol.bsky.social"}

	h := api.FollowProfileHandler(
		graph,
		profiles,
		resolver,
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/profiles/@carol.bsky.social/follows", nil)
	req.SetPathValue("handleOrDid", "carol.bsky.social")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if subj, _ := pds.createRecord["subject"].(string); subj != "did:plc:carol" {
		t.Fatalf("subject=%q want did:plc:carol", subj)
	}
}

func TestUnfollowProfileHandler_InvalidIdentifier(t *testing.T) {
	t.Parallel()

	h := api.UnfollowProfileHandler(
		&fakeFollowGraphStore{},
		&fakeFollowProfileStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return &fakeFollowPDS{}, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@not-valid/follows", nil)
	req.SetPathValue("handleOrDid", "NOT VALID")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env["error"] != "invalid_identifier" {
		t.Fatalf("error code = %v, want invalid_identifier", env["error"])
	}
}

func TestUnfollowProfileHandler_SelfRejected(t *testing.T) {
	t.Parallel()

	h := api.UnfollowProfileHandler(
		&fakeFollowGraphStore{},
		&fakeFollowProfileStore{},
		fakeResolver{didFor: "did:plc:alice"},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return &fakeFollowPDS{}, nil },
		nilLogger(),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/profiles/@alice.example/follows", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "sess-alice")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env["error"] != "self_follow_not_allowed" {
		t.Fatalf("error code = %v, want self_follow_not_allowed", env["error"])
	}
}
