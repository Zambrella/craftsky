package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

type publicStatsStub struct {
	result *api.PublicStats
	err    error
}

func (s publicStatsStub) PublicStats(ctx context.Context) (*api.PublicStats, error) {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
		panic("public statistics read must have a bounded deadline")
	}
	return s.result, s.err
}

func TestPublicStatsHandler(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "anonymous aggregate", true: "protected error"}[failed], func(t *testing.T) {
			store := publicStatsStub{result: &api.PublicStats{GeneratedAt: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), Accounts: 8, Posts: 12, ProjectPosts: 7, FinishedProjectPosts: 3}}
			if failed {
				store.err = errors.New("PRIVATE_DATABASE_CANARY")
			}
			request := httptest.NewRequest("GET", "/v1/public/stats", nil)
			request.Header.Set("Authorization", "Bearer PRIVATE_TOKEN_CANARY")
			request.Header.Set("Cookie", "session=PRIVATE_COOKIE_CANARY")
			response := httptest.NewRecorder()
			api.PublicStatsHandler(store).ServeHTTP(response, request)
			wantStatus := 200
			if failed {
				wantStatus = 503
			}
			if response.Code != wantStatus || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
			}
			if strings.Contains(response.Body.String(), "PRIVATE_") {
				t.Fatal("protected value escaped into public response")
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if failed {
				if len(body) != 3 || body["error"] == nil || body["message"] == nil || body["requestId"] == nil {
					t.Fatal("nonstandard error envelope", response.Body.String())
				}
				return
			}
			if len(body) != 6 || string(body["projectsByCraft"]) != "[]" || string(body["projectPosts"]) != "7" || string(body["generatedAt"]) != `"2026-11-06T12:00:00Z"` {
				t.Fatal("unexpected aggregate contract", response.Body.String())
			}
		})
	}
}

func TestPublicStatsStoreCountsVisiblePostSnapshots(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, publicPageDDL)
	store := api.NewPublicStatsStore(pool)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	stats, err := store.PublicStats(ctx)
	if err != nil || stats.Accounts != 0 || stats.Posts != 0 || stats.ProjectPosts != 0 || stats.FinishedProjectPosts != 0 || stats.ProjectsByCraft == nil || len(stats.ProjectsByCraft) != 0 {
		t.Fatalf("empty statistics=%+v err=%v", stats, err)
	}
	if stats.GeneratedAt.IsZero() || time.Since(stats.GeneratedAt) > time.Minute {
		t.Fatalf("missing snapshot timestamp: %v", stats.GeneratedAt)
	}
	const alice = "did:plc:alice"
	const knitting = "social.craftsky.feed.defs#knitting"
	const sewing = "social.craftsky.feed.defs#sewing"
	const extensionCraft = "social.example.crafts#lace"
	const finished = "social.craftsky.feed.defs#finished"
	seedMember(t, pool, alice)
	seedMember(t, pool, "did:plc:labeler")
	project := func(owner, key, craft, status string) string {
		t.Helper()
		uri := seedPost(t, pool, owner, key, "Public project", time.Now())
		seedProjectMaterialization(t, pool, uri, craft, "Project")
		exec(`UPDATE craftsky_project_posts SET common_status=$2 WHERE uri=$1`, uri, status)
		return uri
	}
	// Posts about the same physical project remain distinct snapshots. Additional
	// crafts do not duplicate a post into a second primary-craft bar.
	first := project(alice, "knit1", knitting, finished)
	project(alice, "knit2", knitting, "social.craftsky.feed.defs#wip")
	exec(`UPDATE craftsky_project_posts SET raw_project=jsonb_set(raw_project,'{common,additionalCrafts}',$2::jsonb) WHERE uri=$1`, first, `["`+sewing+`"]`)
	project(alice, "sew", sewing, finished)
	project(alice, "extension", extensionCraft, finished)
	seedPost(t, pool, alice, "ordinary", "Ordinary post", time.Now())
	quote := project(alice, "quote", knitting, finished)
	exec(`UPDATE craftsky_posts SET quote_uri=$2 WHERE uri=$1`, quote, first)
	for _, column := range []string{"reply_root_uri", "reply_parent_uri"} {
		reply := project(alice, column, knitting, finished)
		exec(`UPDATE craftsky_posts SET `+column+`=$2 WHERE uri=$1`, reply, first)
	}
	// Aggregate visibility must match anonymous reads, independent of a viewer.
	for _, state := range []string{"departed", "deletion_pending", "deleting", "terminal"} {
		owner := "did:plc:" + state
		seedMember(t, pool, owner)
		project(owner, "hidden", knitting, finished)
		exec(`UPDATE owner_lifecycles SET state=$2 WHERE owner_did=$1`, owner, state)
	}
	seedMember(t, pool, "did:plc:suspended")
	project("did:plc:suspended", "hidden", knitting, finished)
	exec(`INSERT INTO moderation_account_standings(owner_did,effective_suspended) VALUES('did:plc:suspended',true)`)
	// A record indexed without a CraftSky membership must not contribute.
	project("did:plc:nonmember", "hidden", knitting, finished)
	for _, value := range []string{"hide", "takedown", "warn"} {
		owner := "did:plc:" + value
		seedMember(t, pool, owner)
		project(owner, "hidden", knitting, finished)
		exec(`INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action) VALUES($1,'did:plc:labeler','account',$2,$3,'apply')`, "account-"+value, owner, value)
		uri := project(alice, "post-"+value, knitting, finished)
		exec(`INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,subject_uri,value,action) VALUES($1,'did:plc:labeler','post',$2,$3,$4,'apply')`, "post-"+value, alice, uri, value)
	}
	unclear := project(alice, "unclear", knitting, finished)
	exec(`UPDATE image_subject_states SET visibility_state='blocked' WHERE subject_uri=$1`, unclear)
	missingClearance := project(alice, "missing-clearance", knitting, finished)
	exec(`DELETE FROM image_subject_states WHERE subject_uri=$1`, missingClearance)
	stats, err = store.PublicStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Accounts != 2 || stats.Posts != 6 || stats.ProjectPosts != 4 || stats.FinishedProjectPosts != 3 {
		t.Fatalf("incorrect public totals: %+v", stats)
	}
	wantCrafts := []api.PublicCraftCount{{Craft: knitting, Count: 2}, {Craft: sewing, Count: 1}, {Craft: extensionCraft, Count: 1}}
	if !reflect.DeepEqual(stats.ProjectsByCraft, wantCrafts) {
		t.Fatalf("crafts=%+v want %+v", stats.ProjectsByCraft, wantCrafts)
	}
	// Removing an owner immediately removes their posts from every count.
	exec(`UPDATE owner_lifecycles SET state='departed' WHERE owner_did=$1`, alice)
	stats, err = store.PublicStats(ctx)
	if err != nil || stats.Accounts != 1 || stats.Posts != 0 || stats.ProjectPosts != 0 || stats.FinishedProjectPosts != 0 || len(stats.ProjectsByCraft) != 0 {
		t.Fatalf("departed owner statistics=%+v err=%v", stats, err)
	}
}

func TestPublicStatsStoreMigratedSchema(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:alice','active',1,1,'test',now())`); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:alice")
	uri := seedPost(t, pool, "did:plc:alice", "knit", "Finished scarf", time.Now())
	seedProjectMaterialization(t, pool, uri, "social.craftsky.feed.defs#knitting", "Scarf")
	if _, err := pool.Exec(ctx, `
 UPDATE craftsky_project_posts SET common_status='social.craftsky.feed.defs#finished' WHERE uri=$1;
 `, uri); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri); err != nil {
		t.Fatal(err)
	}
	stats, err := api.NewPublicStatsStore(pool).PublicStats(ctx)
	if err != nil || stats.Accounts != 1 || stats.Posts != 1 || stats.ProjectPosts != 1 || stats.FinishedProjectPosts != 1 || len(stats.ProjectsByCraft) != 1 || stats.ProjectsByCraft[0].Count != 1 {
		t.Fatalf("production-schema statistics=%+v err=%v", stats, err)
	}
}
