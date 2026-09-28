package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestPostStoreEngagementUsesLogicalSetAggregates(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	postURI := "at://did:plc:author/social.craftsky.feed.post/3aaaaaaaaaaa2"
	for _, did := range []string{"did:plc:author", "did:plc:bob", "did:plc:carol"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
			) VALUES($1,'active',1,1,'test',$2,$2,$2)
		`, did, now); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, did); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES($1,'did:plc:author','3aaaaaaaaaaa2','bafy-post','post','{}'::jsonb,$2)
	`, postURI, now); err != nil {
		t.Fatal(err)
	}
	seedInteractionAggregate(t, pool, "like", "did:plc:bob", postURI, 2, now)
	seedInteractionAggregate(t, pool, "like", "did:plc:carol", postURI, 1, now.Add(time.Minute))
	seedInteractionAggregate(t, pool, "repost", "did:plc:bob", postURI, 2, now)
	store := api.NewPostStore(pool)
	counts, err := store.CountActiveLikes(ctx, []string{postURI})
	if err != nil {
		t.Fatal(err)
	}
	if counts[postURI] != 2 {
		t.Fatalf("logical like count = %d, want 2", counts[postURI])
	}
	summaries, err := store.EngagementSummaries(ctx, "did:plc:bob", []string{}, []string{postURI})
	if err != nil {
		t.Fatal(err)
	}
	summary := summaries[postURI]
	if summary.LikeCount != 2 || !summary.ViewerHasLiked || summary.RepostCount != 1 || !summary.ViewerHasReposted {
		t.Fatalf("engagement summary = %+v", summary)
	}
}

func seedInteractionAggregate(t *testing.T, pool *pgxpool.Pool, kind, actor, subject string, sourceCount int, activityAt time.Time) {
	t.Helper()
	ctx := context.Background()
	var representative string
	for index := 0; index < sourceCount; index++ {
		uri := fmt.Sprintf("at://%s/social.craftsky.feed.%s/3aaaaaaaaaaa%d", actor, kind, index+3)
		if index == 0 {
			representative = uri
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO tap_source_records(
				uri,did,collection,rkey,source_event_id,source_fingerprint,revision,cid,
				action,record,record_bytes,live,ordering_status,projection_disposition,
				structural_validation_status,semantic_validation_status
			) VALUES($1,$2,$3,$4,$5,decode(repeat('00',32),'hex'),$4,
				'bafy-interaction','create','{}'::json,2,false,'authoritative','eligible','valid','valid')
		`, uri, actor, "social.craftsky.feed."+kind, fmt.Sprintf("3aaaaaaaaaaa%d", index+3), index+1); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO pds_set_sources(
				source_uri,kind,actor_did,scope_key,subject_uri,subject_cid,activity_at,eligible
			) VALUES($1,$2,$3,$4,$4,'bafy-post',$5,true)
		`, uri, kind, actor, subject, activityAt.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_set_aggregates(
			kind,actor_did,scope_key,subject_uri,eligible_source_count,representative_source_uri,activated_at
		) VALUES($1,$2,$3,$3,$4,$5,$6)
	`, kind, actor, subject, sourceCount, representative, activityAt); err != nil {
		t.Fatal(err)
	}
}
