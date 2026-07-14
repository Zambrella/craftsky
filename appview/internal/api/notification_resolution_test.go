package api_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestResolveNotificationIsOwnerOnlyAndFallsBackForRetractedContent(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:actor")
	seedBskyProfile(t, pool, "did:plc:actor", "Actor", "avatar")
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key, source_uri, source_cid, source_rkey,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot, state,
			first_activity_at, activity_at, indexed_at, initial_push_evaluated_at, retracted_at, retraction_reason
		) VALUES ('00000000-0000-0000-0000-000000000001', 'did:plc:viewer', 'did:plc:actor', 'mention',
			'post', 'at://did:plc:actor/social.craftsky.feed.post/deleted', 'cid', 'deleted',
			'everyone', false, true, 'retracted', $1, $1, $1, $1, $1, 'sourceDeleted')
	`, now); err != nil {
		t.Fatal(err)
	}

	store := api.NewPostStore(pool)
	resolution, err := store.ResolveNotification(context.Background(), "did:plc:viewer", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Target.Kind != "actorProfile" || resolution.Target.DID != "did:plc:actor" || resolution.State != "retracted" {
		t.Fatalf("resolution=%+v", resolution)
	}
	if _, err := store.ResolveNotification(context.Background(), "did:plc:other", "00000000-0000-0000-0000-000000000001"); err != api.ErrNotificationNotFound {
		t.Fatalf("cross-owner err=%v, want not found", err)
	}
}

func TestResolveNotificationNeverTargetsModeratedPostsOrActors(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:actor")
	seedBskyProfile(t, pool, "did:plc:actor", "Actor", "avatar")
	hiddenPost := seedPost(t, pool, "did:plc:viewer", "hidden", "secret", time.Now())
	seedModerationOutput(t, pool, "post", "did:plc:viewer", hiddenPost, "hide", time.Now())
	seedModerationOutput(t, pool, "account", "did:plc:actor", "", "takedown", time.Now())

	categories := []api.NotificationType{
		api.NotificationTypeFollow,
		api.NotificationTypeLike,
		api.NotificationTypeRepost,
		api.NotificationTypeReply,
		api.NotificationTypeMention,
		api.NotificationTypeQuote,
	}
	states := []string{"active", "retracted"}
	index := 1
	for _, state := range states {
		for _, category := range categories {
			id := fmt.Sprintf("00000000-0000-0000-0000-%012d", index)
			index++
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO notification_events (
					id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,
					subject_uri,parent_uri,root_uri,quoted_uri,
					eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
					first_activity_at,activity_at,indexed_at,initial_push_evaluated_at
				) VALUES ($1::uuid,'did:plc:viewer','did:plc:actor',$2,$1,$3,'cid','r',$3,$3,$3,$3,
					'everyone',false,true,$4,now(),now(),now(),now())`, id, category, hiddenPost, state); err != nil {
				t.Fatal(err)
			}
			resolution, err := api.NewPostStore(pool).ResolveNotification(context.Background(), "did:plc:viewer", id)
			if err != nil {
				t.Fatalf("%s/%s: %v", state, category, err)
			}
			if resolution.Target.Kind != "notifications" || resolution.Target.URI != "" || resolution.Target.DID != "" {
				t.Fatalf("%s/%s leaked moderated target: %+v", state, category, resolution.Target)
			}
		}
	}
}
