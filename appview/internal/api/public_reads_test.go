package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/testdb"
)

func publicReadFixture(t *testing.T) (*pgxpool.Pool, *api.PublicPageStore) {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	for _, owner := range []string{"public", "hidden", "unknown"} {
		did := "did:plc:" + owner
		if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES ($1,'active',1,1,'test',now())`, did); err != nil {
			t.Fatal(err)
		}
		seedMember(t, pool, did)
		seedSearchIdentity(t, pool, did, owner+".test", owner+" AUTHOR_CANARY", owner+" BIO_CANARY")
		if owner != "unknown" {
			if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hidden=$2,guest_visibility_revision='3m234567abc22',guest_visibility_record_cid=record_cid,guest_visibility_evidence='source' WHERE did=$1`, did, owner == "hidden"); err != nil {
				t.Fatal(err)
			}
		}
	}
	resolver := fakeResolver{handlesByDID: map[string]syntax.Handle{"did:plc:public": "public.test", "did:plc:hidden": "hidden.test", "did:plc:unknown": "unknown.test"}}
	return pool, api.NewPublicPageStore(pool, api.NewPostStore(pool), resolver, business.NewStore(pool))
}

// IT-002 / FR-001, FR-010 / AC-001, AC-010, AC-012.
func TestPublicProjectsFiltersBeforePagination(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 12; i++ {
		owner := []string{"public", "hidden", "unknown"}[i%3]
		uri := seedSearchProject(t, pool, "did:plc:"+owner, fmt.Sprint("p", i), owner+" PROJECT_CANARY", api.ProjectCraftTypeKnitting, "Guest project", now.Add(time.Duration(i)*time.Minute))
		if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
	}
	req := api.ProjectListRequest{Limit: 2, Sort: api.SearchSortChronological, Filters: map[string][]string{"craftType": {"knitting"}}}
	first, err := store.PublicProjects(ctx, req)
	if err != nil || len(first.Items) != 2 || first.Cursor == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	req.Cursor = first.Cursor
	second, err := store.PublicProjects(ctx, req)
	if err != nil || len(second.Items) != 2 || second.Cursor != "" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	seen := map[syntax.RecordKey]bool{}
	for _, item := range append(first.Items, second.Items...) {
		if item.Author.DID != "did:plc:public" || seen[item.Rkey] {
			t.Fatalf("ineligible/duplicate item=%+v", item)
		}
		seen[item.Rkey] = true
	}
	raw, _ := json.Marshal(first)
	for _, canary := range []string{"hidden", "unknown", "viewerIsFollowing", "recordRevision", "viewerLiked", "muted"} {
		if containsJSON(raw, canary) {
			t.Fatalf("public serialization leaked %s: %s", canary, raw)
		}
	}
	req.Cursor = ""
	req.Filters["craftType"] = []string{"crochet"}
	empty, err := store.PublicProjects(ctx, req)
	if err != nil || len(empty.Items) != 0 {
		t.Fatalf("primary craft filter=%+v %v", empty, err)
	}
}

func containsJSON(raw []byte, text string) bool {
	for i := 0; i+len(text) <= len(raw); i++ {
		if string(raw[i:i+len(text)]) == text {
			return true
		}
	}
	return false
}

// IT-003 / FR-002, FR-010 / AC-003, AC-010, AC-012.
func TestPublicSearchUsesOnlyEligibleCandidates(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	for _, owner := range []string{"public", "hidden", "unknown"} {
		for i := 0; i < 3; i++ {
			uri := seedSearchProject(t, pool, "did:plc:"+owner, fmt.Sprint("project", i), owner+" knitting #needle", api.ProjectCraftTypeKnitting, "knitting", now.Add(time.Duration(i)*time.Minute))
			if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET tags=ARRAY['needle'],record=jsonb_set(record,'{project}','{"common":{"tags":["needle"]}}'::jsonb) WHERE uri=$1`, uri); err != nil {
				t.Fatal(err)
			}
			post := seedPost(t, pool, "did:plc:"+owner, fmt.Sprint("post", i), owner+" knitting", now.Add(time.Duration(i)*time.Minute))
			if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=ANY($1::text[])`, []string{uri, post}); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, category := range []string{"profiles", "posts", "projects", "hashtags"} {
		t.Run(category, func(t *testing.T) {
			query := "knitting"
			if category == "profiles" {
				query = "AUTHOR_CANARY"
			}
			if category == "hashtags" {
				query = "needle"
			}
			result, err := store.PublicSearch(ctx, category, query, 2, "")
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(result)
			for _, canary := range []string{"hidden", "unknown", "viewerIsFollowing", "muted", "blocking", "recordRevision"} {
				if containsJSON(raw, canary) {
					t.Fatalf("search leaked %q: %s", canary, raw)
				}
			}
			switch category {
			case "posts", "projects":
				page := result.(*api.PublicPostPage)
				if len(page.Items) != 2 || page.Cursor == "" {
					t.Fatalf("eligible page=%s", raw)
				}
				next, err := store.PublicSearch(ctx, category, query, 2, page.Cursor)
				if err != nil || len(next.(*api.PublicPostPage).Items) != 1 {
					t.Fatalf("next=%+v %v", next, err)
				}
			case "profiles":
				if len(result.(*api.PublicProfilePage).Items) != 1 {
					t.Fatalf("profiles=%s", raw)
				}
			case "hashtags":
				if !containsJSON(raw, `"postCount":3`) {
					t.Fatalf("unfiltered hashtag aggregation: %s", raw)
				}
			}
		})
	}
	tagPage, err := store.PublicTagPosts(ctx, "needle", 2, "")
	if err != nil || len(tagPage.Items) != 2 || tagPage.Cursor == "" {
		t.Fatalf("tag page=%+v %v", tagPage, err)
	}
	for _, item := range tagPage.Items {
		if item.Author.DID != "did:plc:public" {
			t.Fatalf("ineligible tagged item: %+v", item)
		}
	}

}

// UT-009 / NFR-002 / AC-012: cursors cannot cross audience, category or filters.
func TestPublicCursorBindsResourceAndFilters(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		uri := seedSearchProject(t, pool, "did:plc:public", fmt.Sprint("p", i), "knitting", api.ProjectCraftTypeKnitting, "knitting", time.Now().Add(-time.Duration(i+1)*time.Minute))
		if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
	}
	req := api.ProjectListRequest{Limit: 1, Sort: api.SearchSortChronological, Filters: map[string][]string{"craftType": {"knitting"}}}
	page, err := store.PublicProjects(ctx, req)
	if err != nil || page.Cursor == "" {
		t.Fatalf("page=%+v %v", page, err)
	}
	req.Cursor = page.Cursor
	req.Filters["craftType"] = []string{"crochet"}
	if _, err := store.PublicProjects(ctx, req); err == nil {
		t.Fatal("cursor accepted for different craft filter")
	}
	if _, err := store.PublicSearch(ctx, "projects", "knitting", 1, page.Cursor); err == nil {
		t.Fatal("feed cursor accepted for submitted search")
	}
	memberRows, memberCursor, err := api.NewSearchStore(pool, nil).SearchProjects(ctx, api.ProjectSearchRequest{Limit: 1, Sort: api.SearchSortChronological}, time.Now())
	if err != nil || len(memberRows) != 1 || memberCursor == "" {
		t.Fatalf("member fixture=%d cursor=%q %v", len(memberRows), memberCursor, err)
	}
	req.Filters["craftType"] = []string{"knitting"}
	req.Cursor = memberCursor
	if _, err := store.PublicProjects(ctx, req); err == nil {
		t.Fatal("member cursor accepted by guest feed")
	}
}

// IT-004 / FR-003, FR-010 / AC-004, AC-005, AC-010.
func TestPublicThreadAndEmbedsRespectIndependentVisibility(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	now := time.Now().Add(-time.Hour)
	root := seedPost(t, pool, "did:plc:public", "root", "Public root", now)
	hiddenRoot := seedPost(t, pool, "did:plc:hidden", "root", "hidden ROOT_CANARY", now)
	quote := seedPost(t, pool, "did:plc:public", "quote", "Public commentary", now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$2,quote_cid='hidden-cid' WHERE uri=$1`, quote, hiddenRoot); err != nil {
		t.Fatal(err)
	}
	for i, owner := range []string{"hidden", "unknown", "public"} {
		seedReplyPost(t, pool, "did:plc:"+owner, "comment", owner+" COMMENT_CANARY", root, root, now.Add(time.Duration(i+1)*time.Minute))
	}
	independent := seedReplyPost(t, pool, "did:plc:public", "independent", "Public standalone reply", hiddenRoot, hiddenRoot, now.Add(time.Minute))
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	post, err := store.PublicPost(ctx, "did:plc:public", "quote")
	if err != nil {
		t.Fatal(err)
	}
	if post.QuoteView == nil || post.QuoteView.State != "unavailable" || post.QuoteView.Post != nil {
		t.Fatalf("excluded quote=%+v", post.QuoteView)
	}
	thread, err := store.PublicThread(ctx, "did:plc:public", "root", 20, "")
	if err != nil || len(thread.Replies.Items) != 1 {
		t.Fatalf("thread=%+v %v", thread, err)
	}
	direct, err := store.PublicThread(ctx, "did:plc:public", "independent", 20, "")
	if err != nil || direct.Post.URI != syntax.ATURI(independent) || len(direct.Ancestors) != 1 || direct.Ancestors[0].State != "unavailable" {
		t.Fatalf("independent reply=%+v %v", direct, err)
	}
	comments, err := store.PublicProfilePosts(ctx, "did:plc:public", "comments", 20, "")
	if err != nil || len(comments.Items) != 2 {
		t.Fatalf("comments=%+v %v", comments, err)
	}
	for _, result := range []any{post, thread, direct, comments} {
		raw, _ := json.Marshal(result)
		for _, canary := range []string{"hidden", "unknown", "ROOT_CANARY", "viewerHas", "recordRevision"} {
			if containsJSON(raw, canary) {
				t.Fatalf("nested disclosure %s: %s", canary, raw)
			}
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$2 WHERE uri=$1`, quote, root); err != nil {
		t.Fatal(err)
	}
	visible, err := store.PublicPost(ctx, "did:plc:public", "quote")
	if err != nil || visible.QuoteView == nil || visible.QuoteView.State != "visible" || visible.QuoteView.Post.Text != "Public root" {
		t.Fatalf("visible quote=%+v %v", visible, err)
	}
}

// IT-004 / FR-003 / AC-004, AC-006: public profile facts exclude personalized
// counts and owner analytics, while preserving the shared profile presentation.
func TestPublicProfileIncludesPublicFacts(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO profile_customisations(owner_did,colour,profile_background) VALUES ('did:plc:public','cobalt','none')`); err != nil {
		t.Fatal(err)
	}
	seedPost(t, pool, "did:plc:public", "ordinary", "Public post", time.Now())
	seedSearchProject(t, pool, "did:plc:public", "project", "Public project", api.ProjectCraftTypeKnitting, "Public project", time.Now())
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	profile, err := store.PublicProfile(ctx, "did:plc:public")
	if err != nil {
		t.Fatal(err)
	}
	if profile.PostCount != 1 || profile.ProjectCount != 1 || profile.Customisation == nil {
		t.Fatalf("missing public profile facts: %+v", profile)
	}
	raw, _ := json.Marshal(profile)
	for _, private := range []string{"mutualFollowerCount", "postsLast7Days", "viewer", "editableLinks", "muted", "blocking"} {
		if containsJSON(raw, private) {
			t.Fatalf("private profile field %s: %s", private, raw)
		}
	}
}

// IT-004 / FR-003, FR-010 / AC-004, AC-006, AC-010.
func TestPublicInteractionIdentitiesAndRepostsExcludeHiddenOwners(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	now := time.Now().Add(-time.Hour)
	root := seedPost(t, pool, "did:plc:public", "root", "Public root", now)
	for _, owner := range []string{"public", "hidden", "unknown"} {
		for _, kind := range []string{"like", "repost"} {
			seedSearchSet(t, pool, kind, "did:plc:"+owner, root, false)
		}
		quote := seedPost(t, pool, "did:plc:"+owner, "quote", owner+" QUOTE_CANARY", now)
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$2,quote_cid='root-cid' WHERE uri=$1`, quote, root); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts`); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"likes", "reposts", "quotes"} {
		result, err := store.PublicInteractions(ctx, "did:plc:public", "root", api.PostInteractionKind(kind), 20, "")
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(result)
		for _, canary := range []string{"hidden", "unknown", "viewer", "muted", "blocking"} {
			if containsJSON(raw, canary) {
				t.Fatalf("identity leak %s: %s", canary, raw)
			}
		}
		if kind == "quotes" {
			if len(result.(*api.PublicPostPage).Items) != 1 {
				t.Fatalf("quotes=%s", raw)
			}
		} else {
			if len(result.(*api.PublicProfilePage).Items) != 1 {
				t.Fatalf("identities=%s", raw)
			}
		}
	}
}

// IT-004, IT-011 / FR-003, RULE-003 / AC-004, AC-010, AC-016.
func TestPublicBusinessEventsUseGuestPolicy(t *testing.T) {
	pool, store := publicReadFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	owner := syntax.DID("did:plc:public")
	seedBusinessTestLicense(t, pool, owner)
	for i := 0; i < 3; i++ {
		key := fmt.Sprint("e", i)
		uri := "at://" + owner.String() + "/social.craftsky.business.event/" + key
		starts, ends := now.Add(time.Duration(i+1)*time.Hour), now.Add(time.Duration(i+2)*time.Hour)
		raw, _ := json.Marshal(map[string]any{"$type": "social.craftsky.business.event", "name": "Public event", "startsAt": starts.Format(time.RFC3339Nano), "endsAt": ends.Format(time.RFC3339Nano), "roles": []string{"vendor"}, "createdAt": now.Format(time.RFC3339Nano)})
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_business_events(uri,owner_did,rkey,cid,raw_record,source_revision,starts_at,ends_at,created_at) VALUES ($1,$2,$3,'event-cid',$4,'3m234567abc22',$5,$6,$7)`, uri, owner, key, raw, starts, ends, now); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES ($1,'event','event-cid','clear')`, uri); err != nil {
			t.Fatal(err)
		}
	}
	event, err := store.PublicEvent(ctx, owner, "e0")
	if err != nil || event.Name != "Public event" {
		t.Fatalf("event=%+v %v", event, err)
	}
	page, err := store.PublicEvents(ctx, owner, 2, "")
	if err != nil || len(page.Items) != 2 || page.Cursor == "" {
		t.Fatalf("events=%+v %v", page, err)
	}
	next, err := store.PublicEvents(ctx, owner, 2, page.Cursor)
	if err != nil || len(next.Items) != 1 || next.Cursor != "" {
		t.Fatalf("next=%+v %v", next, err)
	}
	raw, _ := json.Marshal(event)
	if containsJSON(raw, "ExclusionReasons") || containsJSON(raw, "SuppressionReasons") {
		t.Fatalf("management metadata leaked: %s", raw)
	}
	if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hidden=true WHERE did=$1`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublicEvent(ctx, owner, "e0"); !errors.Is(err, api.ErrPublicAuthenticationRequired) {
		t.Fatalf("opted-out event accessible: %v", err)
	}
}

// IT-012 / RULE-004: constructing/reconstructing the public read composition and
// browsing legacy content cannot populate visibility or create ingestion jobs.
func TestPublicReadCompositionNeverRepairsUnknownProfiles(t *testing.T) {
	pool, _ := publicReadFixture(t)
	ctx := context.Background()
	seedSearchProject(t, pool, "did:plc:unknown", "legacy", "Legacy unknown project", api.ProjectCraftTypeKnitting, "Legacy", time.Now())
	snapshot := func() string {
		t.Helper()
		var value string
		if err := pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(p) ORDER BY did)::text FROM bluesky_profiles p`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	for restart := 0; restart < 3; restart++ {
		store := api.NewPublicPageStore(pool, api.NewPostStore(pool), fakeResolver{}, business.NewStore(pool))
		for request := 0; request < 3; request++ {
			if _, err := store.PublicProfile(ctx, "did:plc:unknown"); !errors.Is(err, api.ErrProfileNotFound) {
				t.Fatalf("unknown profile: %v", err)
			}
			if _, err := store.PublicSearch(ctx, "posts", "Legacy", 20, ""); err != nil {
				t.Fatal(err)
			}
			if _, err := store.PublicProjects(ctx, api.ProjectListRequest{Limit: 20, Sort: api.SearchSortChronological}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if after := snapshot(); after != before {
		t.Fatal("public read composition changed legacy profiles")
	}
	for _, table := range []string{"tap_repository_jobs", "tap_projection_jobs"} {
		var jobs int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&jobs); err != nil || jobs != 0 {
			t.Fatalf("%s jobs=%d err=%v", table, jobs, err)
		}
	}
}
