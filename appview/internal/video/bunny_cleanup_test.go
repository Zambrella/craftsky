package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func cleanupFixture(t *testing.T) (*Bunny, *BunnyCleanup, string, func(string), func(string)) {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:bunny-cleanup-owner")
	_, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
 VALUES($1,'active',1,1,'test',now())`, owner)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewBunny(pool, "123", "stream-key-canary", "test.b-cdn.net", func(context.Context, syntax.DID, string) (BunnyBlobEffects, error) { return &bunnyEffects{}, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	_, err = pool.Exec(ctx, `INSERT INTO bunny_video_jobs(id,owner_did,owner_generation,library_id,video_id,state,blob_cid,blob_size)
 VALUES($1,$2,1,'123','aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa','ready','bafyvideo',100)`, id, owner)
	if err != nil {
		t.Fatal(err)
	}
	add := func(key string) {
		t.Helper()
		_, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
  VALUES($1,$2,$3,'postcid','video','{"embed":{"video":{"ref":{"$link":"bafyvideo"}}}}',now())`, "at://"+owner.String()+"/social.craftsky.feed.post/"+key, owner, key)
		if err != nil {
			t.Fatal(err)
		}
	}
	remove := func(key string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE rkey=$1`, key); err != nil {
			t.Fatal(err)
		}
	}
	return b, NewBunnyCleanup(b, "purge-key-canary", nil), id, add, remove
}

func cleanupStep(t *testing.T, w *BunnyCleanup) int {
	t.Helper()
	n, err := w.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func cleanupState(t *testing.T, b *Bunny, id string) string {
	t.Helper()
	var state string
	if err := b.pool.QueryRow(context.Background(), `SELECT state FROM bunny_video_jobs WHERE id=$1`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestBunnyCleanupLastReferenceAndPurge(t *testing.T) {
	b, w, id, add, remove := cleanupFixture(t)
	var calls []string
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Host)
		switch r.URL.Host {
		case "video.bunnycdn.com":
			if r.Method != "DELETE" || r.Header.Get("AccessKey") != "stream-key-canary" {
				t.Fatal("wrong origin credential or operation")
			}
		case "api.bunny.net":
			if r.Header.Get("AccessKey") != "purge-key-canary" || r.URL.Path != "/purge" || r.URL.Query().Get("async") != "false" || r.URL.Query().Get("url") != "https://test.b-cdn.net/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa/*" {
				t.Fatal("wrong purge scope or credential")
			}
		default:
			t.Fatal("unexpected external destination")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	add("first")
	add("second")
	remove("first")
	cleanupStep(t, w)
	if len(calls) != 0 || cleanupState(t, b, id) != "ready" {
		t.Fatal("shared video was removed")
	}
	remove("second")
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "deleting" || len(calls) != 0 {
		t.Fatal("deletion intent not durable before external call")
	}
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "purging" {
		t.Fatal("purge obligation was lost")
	}
	// Restart between origin deletion and purge.
	cleanupStep(t, NewBunnyCleanup(b, "purge-key-canary", nil))
	if cleanupState(t, b, id) != "deleted" || fmt.Sprint(calls) != "[DELETE video.bunnycdn.com POST api.bunny.net]" {
		t.Fatal(calls)
	}
	if cleanupStep(t, w) != 0 {
		t.Fatal("completed cleanup repeated")
	}
	limits, err := b.Get(context.Background(), "did:plc:bunny-cleanup-owner", "")
	if err != nil || *limits.RemainingDailyVideos != 19 {
		t.Fatal("deleting a video bypassed allocation quota")
	}
}

func TestBunnyCleanupRetriesDeleteAndPurgeIndependently(t *testing.T) {
	b, w, id, add, remove := cleanupFixture(t)
	deletes, purges := 0, 0
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		status := http.StatusServiceUnavailable
		if r.Method == "DELETE" {
			deletes++
			if deletes > 1 {
				status = 404
			}
		} else {
			purges++
			if purges > 1 {
				status = 200
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	due := func() {
		t.Helper()
		_, err := b.pool.Exec(context.Background(), `UPDATE bunny_video_jobs SET cleanup_next_attempt_at=now() WHERE id=$1`, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	add("retry")
	remove("retry")
	cleanupStep(t, w)
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "deleting" || cleanupStep(t, w) != 0 {
		t.Fatal("delete failure not backed off")
	}
	due()
	cleanupStep(t, NewBunnyCleanup(b, "purge-key-canary", nil))
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "purging" || cleanupStep(t, w) != 0 {
		t.Fatal("purge failure not retained")
	}
	due()
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "deleted" || deletes != 2 || purges != 2 {
		t.Fatalf("deletes=%d purges=%d", deletes, purges)
	}
}

func TestBunnyCleanupReferenceRollbackAndConcurrentReplacement(t *testing.T) {
	b, w, id, add, remove := cleanupFixture(t)
	b.client.Transport = bunnyRoundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("referenced video must not be deleted")
		return nil, nil
	})
	add("existing")
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM craftsky_posts WHERE rkey='existing'`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if cleanupStep(t, w) != 0 {
		t.Fatal("rolled-back deletion queued cleanup")
	}
	remove("existing")
	tx, err = b.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
 VALUES('at://did:plc:bunny-cleanup-owner/social.craftsky.feed.post/replacement','did:plc:bunny-cleanup-owner','replacement','cid','video','{"embed":{"video":{"ref":{"$link":"bafyvideo"}}}}',now())`)
	if err != nil {
		t.Fatal(err)
	}
	if cleanupStep(t, w) != 0 {
		t.Fatal("worker raced an uncommitted reference")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if cleanupStep(t, w) != 0 || cleanupState(t, b, id) != "ready" {
		t.Fatal("new reference failed to cancel cleanup")
	}
}

func TestBunnyCleanupVideoRemovedByEditAndMissingPurgeKey(t *testing.T) {
	b, w, id, add, _ := cleanupFixture(t)
	add("edited")
	if _, err := b.pool.Exec(context.Background(), `UPDATE craftsky_posts SET record='{}' WHERE rkey='edited'`); err != nil {
		t.Fatal(err)
	}
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method != "DELETE" {
			t.Fatal("purge must not be sent without its credential")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	w.purgeKey = ""
	cleanupStep(t, w)
	cleanupStep(t, w)
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "purging" {
		t.Fatal("missing purge key lost cleanup authority")
	}
	var attempts int
	if err := b.pool.QueryRow(context.Background(), `SELECT cleanup_attempts FROM bunny_video_jobs WHERE id=$1`, id).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatal("missing key did not schedule retry", err)
	}
}

func TestBunnyCleanupProtectsInFlightPublication(t *testing.T) {
	b, w, id, add, remove := cleanupFixture(t)
	ctx := context.Background()
	add("original")
	remove("original")
	if _, err := b.pool.Exec(ctx, `UPDATE bunny_video_jobs SET lease_until=now()+interval '2 minutes' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if cleanupStep(t, w) != 0 {
		t.Fatal("fresh publication reservation was ignored")
	}
	command := uuid.NewString()
	_, err := b.pool.Exec(ctx, `INSERT INTO pds_commands(id,owner_did,owner_generation,operation_kind,operation_key,request_fingerprint_version,request_fingerprint,immutable_request,active_plan_version)
 VALUES($1,'did:plc:bunny-cleanup-owner',1,'post.create',$1,1,decode(repeat('00',32),'hex'),'{}',1)`, command)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.pool.Exec(ctx, `INSERT INTO pds_command_steps(command_id,plan_version,ordinal,action,repo_did,collection,rkey,record,selected_uri)
 VALUES($1,1,0,'create','did:plc:bunny-cleanup-owner','social.craftsky.feed.post','pending','{"embed":{"video":{"ref":{"$link":"bafyvideo"}}}}','at://did:plc:bunny-cleanup-owner/social.craftsky.feed.post/pending')`, command)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.pool.Exec(ctx, `UPDATE bunny_video_jobs SET lease_until='-infinity' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "ready" {
		t.Fatal("unsettled PDS publication was not retained")
	}
	_, err = b.pool.Exec(ctx, `UPDATE pds_commands SET state='rejected',terminal_http_status=400,terminal_at=now(),replay_expires_at=now()+interval '1 day' WHERE id=$1`, command)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.pool.Exec(ctx, `UPDATE bunny_video_jobs SET cleanup_next_attempt_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	cleanupStep(t, w)
	if cleanupState(t, b, id) != "deleting" {
		t.Fatal("settled unreferenced video did not resume cleanup")
	}
}
