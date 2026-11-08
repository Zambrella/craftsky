package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/testdb"
)

type publicAccountTypeStub struct{}

func (publicAccountTypeStub) ReadAccountType(context.Context, syntax.DID) (business.AccountType, error) {
	return business.AccountTypeRegular, nil
}

type publicPageStub struct {
	calls int
	err   error
}

func (s *publicPageStub) PublicProfile(context.Context, syntax.DID) (*api.PublicProfile, error) {
	s.calls++
	return &api.PublicProfile{PublicAuthor: api.PublicAuthor{DID: "did:plc:alice", Handle: "alice.test"}}, s.err
}
func (s *publicPageStub) PublicPost(context.Context, syntax.DID, syntax.RecordKey) (*api.PublicPost, error) {
	s.calls++
	return &api.PublicPost{Text: "public text"}, s.err
}
func (s *publicPageStub) PublicPosts(context.Context, syntax.DID, bool, int, string) (*api.PublicPostPage, error) {
	s.calls++
	return &api.PublicPostPage{Items: []api.PublicPost{}}, s.err
}
func TestPublicPagesHandler(t *testing.T) {
	for _, tc := range []struct {
		name, resource, path, did, key string
		err                            error
		status, calls                  int
	}{
		{name: "anonymous profile", resource: "profile", did: "did:plc:alice", status: 200, calls: 1},
		{name: "anonymous post", resource: "post", did: "did:plc:alice", key: "post1", status: 200, calls: 1},
		{name: "missing", resource: "profile", did: "did:plc:alice", err: api.ErrProfileNotFound, status: 404, calls: 1},
		{name: "store failure", resource: "post", did: "did:plc:alice", key: "post1", err: errors.New("PRIVATE_CANARY"), status: 503, calls: 1},
		{name: "invalid did", resource: "post", did: "did:!", key: "post1", status: 400},
		{name: "invalid key", resource: "post", did: "did:plc:alice", key: "bad/key", status: 400},
		{name: "invalid cursor", resource: "posts", did: "did:plc:alice", path: "?cursor=broken", status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &publicPageStub{err: tc.err}
			r := httptest.NewRequest("GET", "/"+tc.path, nil)
			r.SetPathValue("handleOrDid", tc.did)
			r.SetPathValue("did", tc.did)
			r.SetPathValue("rkey", tc.key)
			r.Header.Set("Authorization", "Bearer PRIVATE_CANARY")
			r.Header.Set("Cookie", "session=PRIVATE_CANARY")
			w := httptest.NewRecorder()
			api.PublicPagesHandler(store, fakeResolver{handleFor: "alice.test"}, tc.resource).ServeHTTP(w, r)
			if w.Code != tc.status || store.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, store.calls, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "PRIVATE_CANARY") {
				t.Fatal("unsafe response", w.Body.String())
			}
		})
	}
}

const publicPageDDL = postStoreDDL + `
CREATE OR REPLACE FUNCTION appview_owner_is_active(owner TEXT) RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$ SELECT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=owner AND state='active') $$;
ALTER TABLE bluesky_profiles ADD COLUMN IF NOT EXISTS description TEXT, ADD COLUMN IF NOT EXISTS pronouns TEXT, ADD COLUMN IF NOT EXISTS banner_cid TEXT, ADD COLUMN IF NOT EXISTS banner_mime TEXT, ADD COLUMN IF NOT EXISTS record_cid TEXT NOT NULL DEFAULT 'profile-cid', ADD COLUMN IF NOT EXISTS links JSONB NOT NULL DEFAULT '[]';
CREATE TABLE profile_customisations(owner_did TEXT PRIMARY KEY,colour TEXT,profile_background TEXT);
CREATE TABLE bluesky_profile_links(uri TEXT PRIMARY KEY,did TEXT,cid TEXT,record JSONB);
CREATE TABLE moderation_account_standings(owner_did TEXT PRIMARY KEY,effective_suspended BOOLEAN);
`

func TestPublicPageStoreVisibilityAndPagination(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, publicPageDDL)
	if err := testdb.ApplyMigrations(ctx, pool, "000118_profile_guest_visibility.up.sql"); err != nil {
		t.Fatal(err)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "")
	exec(`UPDATE bluesky_profiles SET avatar_cid=NULL,avatar_mime=NULL WHERE did='did:plc:alice'`)
	exec(`UPDATE bluesky_profiles SET guest_visibility_hidden=false,guest_visibility_revision='3m234567abc22',guest_visibility_record_cid=record_cid,guest_visibility_evidence='source' WHERE did='did:plc:alice'`)

	seedMember(t, pool, "did:plc:labeler")
	store := api.NewPublicPageStore(pool, api.NewPostStore(pool), fakeResolver{handleFor: "alice.test"}, publicAccountTypeStub{})
	for i := 0; i < 4; i++ {
		seedPost(t, pool, "did:plc:alice", fmt.Sprintf("post%d", i), fmt.Sprintf("Text %d", i), time.Now().Add(time.Duration(i)*time.Second))
	}
	profile, err := store.PublicProfile(ctx, "did:plc:alice")
	if err != nil || profile.DisplayName == nil || *profile.DisplayName != "Alice" {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	if profile.CreatedAt.IsZero() || profile.PostsLast7Days != 4 {
		t.Fatalf("public summary missing join date or recent posts: %+v", profile)
	}
	const avatarCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
	// Profile indexing populates serving columns only after per-slot clearance.
	// It does not write the record-level image_subject_states used for posts.
	if profile.Avatar != nil || profile.Banner != nil {
		t.Fatal("profile without cleared serving images exposed images")
	}
	exec(`UPDATE bluesky_profiles SET avatar_cid=$1,avatar_mime='image/jpeg' WHERE did='did:plc:alice'`, avatarCID)
	profile, err = store.PublicProfile(ctx, "did:plc:alice")
	if err != nil || profile.Avatar == nil || profile.Banner != nil {
		t.Fatal("cleared avatar missing or uncleared banner exposed", err)
	}
	// A later profile revision can keep the last cleared avatar while a new
	// candidate is scanned. The serving image is independent of record CID.
	exec(`UPDATE bluesky_profiles SET record_cid='new-profile-revision',banner_cid=$1,banner_mime='image/png' WHERE did='did:plc:alice'`, avatarCID)
	profile, err = store.PublicProfile(ctx, "did:plc:alice")
	if err != nil || profile.Avatar == nil || profile.Banner == nil {
		t.Fatal("last cleared profile images missing after record update", err)
	}
	// Rescans and removal clear serving columns independently.
	exec(`UPDATE bluesky_profiles SET avatar_cid=NULL,avatar_mime=NULL WHERE did='did:plc:alice'`)
	profile, err = store.PublicProfile(ctx, "did:plc:alice")
	if err != nil || profile.Avatar != nil || profile.Banner == nil {
		t.Fatal("withdrawn avatar exposed or independent banner hidden", err)
	}
	exec(`INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action) VALUES ('hide-account','did:plc:labeler','account','did:plc:alice','hide','apply')`)
	if _, err := store.PublicProfile(ctx, "did:plc:alice"); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatal("hidden profile visible", err)
	}
	exec(`DELETE FROM moderation_outputs WHERE id='hide-account'`)

	first, err := store.PublicPosts(ctx, "did:plc:alice", false, 2, "")
	if err != nil || len(first.Items) != 2 || first.Cursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := store.PublicPosts(ctx, "did:plc:alice", false, 2, first.Cursor)
	if err != nil || len(second.Items) != 2 || second.Cursor != "" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	seen := map[syntax.RecordKey]bool{}
	for _, p := range append(first.Items, second.Items...) {
		if seen[p.Rkey] {
			t.Fatal("duplicate page item")
		}
		seen[p.Rkey] = true
	}
	post, err := store.PublicPost(ctx, "did:plc:alice", "post0")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(post)
	for _, key := range []string{"viewer", "recordRevision", "facets", "moderation", "quoteView", "editableLinks"} {
		if strings.Contains(string(raw), key) {
			t.Fatal("private contract expansion", string(raw))
		}
	}
	exec(`UPDATE image_subject_states SET visibility_state='blocked' WHERE subject_uri=$1`, "at://did:plc:alice/social.craftsky.feed.post/post0")
	if _, err := store.PublicPost(ctx, "did:plc:alice", "post0"); !errors.Is(err, api.ErrPostNotFound) {
		t.Fatal("unsafe post visible", err)
	}
	exec(`INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,subject_uri,value,action) VALUES ('warning','did:plc:labeler','post','did:plc:alice','at://did:plc:alice/social.craftsky.feed.post/post1','warn','apply')`)
	page, err := store.PublicPosts(ctx, "did:plc:alice", false, 20, "")
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("filter before limit: %+v %v", page, err)
	}
	exec(`INSERT INTO moderation_account_standings VALUES ('did:plc:alice',true)`)
	if _, err := store.PublicProfile(ctx, "did:plc:alice"); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatal("suspended profile visible", err)
	}
	if _, err := store.PublicPost(ctx, "did:plc:alice", "post2"); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatal("suspended author's post visible", err)
	}
	exec(`DELETE FROM moderation_account_standings`)
	exec(`UPDATE owner_lifecycles SET state='terminal',terminal_at=now() WHERE owner_did='did:plc:alice'`)
	if _, err := store.PublicProfile(ctx, "did:plc:alice"); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatal("terminal profile visible", err)
	}
	exec(`UPDATE owner_lifecycles SET state='active',terminal_at=NULL WHERE owner_did='did:plc:alice'`)
	exec(`DELETE FROM craftsky_profiles WHERE did='did:plc:alice'`)
	if _, err := store.PublicPost(ctx, "did:plc:alice", "post2"); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatal("nonmember post visible", err)
	}
}

// Execute the new queries against the exact production schema as well as the
// focused visibility fixture, to catch drift in table/column/function contracts.
func TestPublicPageStoreMigratedSchema(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES ('did:plc:alice','active',1,1,'test',now())`)
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "")
	exec(`UPDATE bluesky_profiles SET guest_visibility_hidden=false,guest_visibility_revision='3m234567abc22',guest_visibility_record_cid=record_cid,guest_visibility_evidence='source' WHERE did='did:plc:alice'`)
	uri := seedPost(t, pool, "did:plc:alice", "first", "Public project", time.Now())
	exec(`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri)
	store := api.NewPublicPageStore(pool, api.NewPostStore(pool), fakeResolver{handleFor: "alice.test"}, business.NewStore(pool))
	if _, err := store.PublicProfile(ctx, "did:plc:alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublicPost(ctx, "did:plc:alice", "first"); err != nil {
		t.Fatal(err)
	}
	posts, err := store.PublicPosts(ctx, "did:plc:alice", false, 20, "")
	if err != nil || len(posts.Items) != 1 {
		t.Fatalf("posts=%+v err=%v", posts, err)
	}
}

// IT-008 / RULE-001, RULE-002, RULE-004 / AC-006, AC-010, AC-017.
func TestPublicPagesEnforceGuestVisibilityWithoutRepair(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES ('did:plc:alice','active',1,1,'test',now())`); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "PRIVATE_AUTHOR_CANARY", "")
	uri := seedPost(t, pool, "did:plc:alice", "guest", "PRIVATE_POST_CANARY", time.Now())
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri); err != nil {
		t.Fatal(err)
	}
	store := api.NewPublicPageStore(pool, api.NewPostStore(pool), fakeResolver{handleFor: "alice.test"}, business.NewStore(pool))
	public, hidden := false, true
	for _, tc := range []struct {
		name   string
		hidden *bool
		hold   bool
		status int
	}{
		{"unknown", nil, false, 404}, {"public", &public, false, 200}, {"opted out", &hidden, false, 403}, {"pending write", &public, true, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hidden != nil {
				if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hidden=$1,guest_visibility_revision='3m234567abc22',guest_visibility_record_cid='profile',guest_visibility_evidence='source',guest_visibility_hold_key=NULL WHERE did='did:plc:alice'`, *tc.hidden); err != nil {
					t.Fatal(err)
				}
			}
			if tc.hold {
				if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hold_key='018f4d5c-7a61-7d40-a1a2-777777777777' WHERE did='did:plc:alice'`); err != nil {
					t.Fatal(err)
				}
			}
			for _, resource := range []string{"profile", "post", "posts", "projects"} {
				request := httptest.NewRequest("GET", "/v1/public/test", nil)
				request.SetPathValue("handleOrDid", "did:plc:alice")
				request.SetPathValue("did", "did:plc:alice")
				request.SetPathValue("rkey", "guest")
				request.Header.Set("Authorization", "Bearer spoofed-owner")
				request.Header.Set("Cookie", "owner=spoofed")
				response := httptest.NewRecorder()
				api.PublicPagesHandler(store, fakeResolver{}, resource).ServeHTTP(response, request)
				if response.Code != tc.status {
					t.Fatalf("%s status=%d want=%d body=%s", resource, response.Code, tc.status, response.Body.String())
				}
				if tc.status != 200 && (strings.Contains(response.Body.String(), "PRIVATE_") || strings.Contains(response.Body.String(), "spoofed")) {
					t.Fatal("guest response exposed protected data")
				}
			}
			var repairs int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_repository_jobs WHERE did='did:plc:alice'`).Scan(&repairs); err != nil || repairs != 0 {
				t.Fatalf("guest reads caused repair: %d %v", repairs, err)
			}
		})
	}
}

// UT-009 / NFR-002 / AC-012: bounds are validated before dependency calls.
func TestPublicPageRequestBounds(t *testing.T) {
	for _, query := range []string{"?limit=21", "?limit=0", "?limit=-1", "?limit=abc", "?cursor=" + strings.Repeat("a", 2049), "?viewerDid=did:plc:owner", "?sort=popular", "?limit=1&limit=2"} {
		t.Run(query[:min(len(query), 45)], func(t *testing.T) {
			store := &publicPageStub{}
			r := httptest.NewRequest("GET", "/"+query, nil)
			r.SetPathValue("handleOrDid", "did:plc:alice")
			w := httptest.NewRecorder()
			api.PublicPagesHandler(store, fakeResolver{}, "posts").ServeHTTP(w, r)
			if w.Code != 400 || store.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, store.calls, w.Body.String())
			}
		})
	}
}
