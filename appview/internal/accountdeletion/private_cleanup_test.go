package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/testdb"
)

func TestPrivateCleanupReplaysEveryComponentAfterFailure(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	var calls []string
	firstFailure := true
	cleaner, err := NewPrivateCleaner([]PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "databasePrivate", run: func(gotOwner syntax.DID) error {
			if gotOwner != owner {
				t.Fatalf("database component scope = %s", gotOwner)
			}
			calls = append(calls, "databasePrivate")
			return nil
		}},
		fakePrivateCleanupComponent{name: "scheduledPosts", run: func(syntax.DID) error {
			calls = append(calls, "scheduledPosts")
			if firstFailure {
				firstFailure = false
				return errors.New("object cleanup queue unavailable")
			}
			return nil
		}},
		fakePrivateCleanupComponent{name: "instagram", run: func(syntax.DID) error {
			calls = append(calls, "instagram")
			return nil
		}},
	})
	if err != nil {
		t.Fatalf("construct cleaner: %v", err)
	}

	if err := cleaner.Run(context.Background(), owner); err == nil {
		t.Fatal("first cleanup unexpectedly succeeded")
	}
	if got, want := calls, []string{"databasePrivate", "scheduledPosts"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first calls = %v, want %v", got, want)
	}
	if err := cleaner.Run(context.Background(), owner); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
	if got, want := calls, []string{"databasePrivate", "scheduledPosts", "databasePrivate", "scheduledPosts", "instagram"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("retry calls = %v, want %v", got, want)
	}
}

func TestDatabasePrivateCleanupDeletesOnlyOwnerPrivateState(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000902")
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	seedSQL := `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'alice-profile-cid'),($2,'bob-profile-cid');
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
		VALUES
			('at://did:plc:alice/social.craftsky.feed.post/a',$1,'a','alice-post-cid','alice','{}',$3),
			('at://did:plc:bob/social.craftsky.feed.post/b',$2,'b','bob-post-cid','bob','{}',$3);
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'alice-deletion-oauth','{}'),($2,'bob-oauth','{}');
		INSERT INTO account_deletion_operations(
			id,owner_did,state,accepted_at,deletion_oauth_session_id
		) VALUES($4,$1,'active',$3,'alice-deletion-oauth');

		INSERT INTO craftsky_recent_searches(id,viewer_did,search_type,display_label,normalized_payload,normalized_payload_hash)
		VALUES('alice-search',$1,'profile','Alice private search','{}','alice-search-hash'),
		      ('bob-search',$2,'profile','Bob private search','{}','bob-search-hash');
		INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2),($2,$1);
		INSERT INTO account_language_preferences(account_did,primary_language) VALUES($1,'en'),($2,'en');
		INSERT INTO profile_customisations(owner_did,colour,profile_border,profile_background)
		VALUES($1,'blue','none','plain'),($2,'red','none','plain');
		INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at)
		VALUES($1,'standard','at://did:plc:alice/social.craftsky.feed.post/a',gen_random_uuid(),$3,$3),
		      ($2,'standard','at://did:plc:bob/social.craftsky.feed.post/b',gen_random_uuid(),$3,$3);
		INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at)
		VALUES('10000000-0000-4000-8000-000000000001',$1,'Alice folder',$3,$3),
		      ('10000000-0000-4000-8000-000000000002',$2,'Bob folder',$3,$3);
		INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at)
		VALUES($1,'at://did:plc:bob/social.craftsky.feed.post/b','10000000-0000-4000-8000-000000000001',$3),
		      ($2,'at://did:plc:alice/social.craftsky.feed.post/a','10000000-0000-4000-8000-000000000002',$3);
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled)
		VALUES($1,'like','everyone',true),($2,'like','everyone',true);
		INSERT INTO notification_seen_state(account_did,last_seen_revision) VALUES($1,1),($2,1);
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,
			eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
			first_activity_at,activity_at,initial_push_evaluated_at
		) VALUES
			('20000000-0000-4000-8000-000000000001',$1,$2,'like','alice-recipient','at://did:plc:bob/social.craftsky.feed.like/l','cid','l','everyone',false,true,'active',$3,$3,$3),
			('20000000-0000-4000-8000-000000000002',$2,$1,'like','alice-actor','at://did:plc:alice/social.craftsky.feed.like/l','cid','l','everyone',false,true,'active',$3,$3,$3);
		INSERT INTO push_installations(id,device_id,platform,fcm_token)
		VALUES('30000000-0000-4000-8000-000000000001','shared-device','ios','shared-token');
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id)
		VALUES
			('40000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001',$1,'50000000-0000-4000-8000-000000000001'),
			('40000000-0000-4000-8000-000000000002','30000000-0000-4000-8000-000000000001',$2,'50000000-0000-4000-8000-000000000002');
		INSERT INTO moderation_reports(
			id,reporter_did,subject_type,subject_did,submitted_handle_snapshot,reason_type,
			forwarding_status,forwarding_prepared_at
		) VALUES('alice-report',$1,'account',$2,'alice.test','spam','prepared_not_submitted',$3),
		        ('bob-report',$2,'account',$1,'bob.test','spam','prepared_not_submitted',$3);
		INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,value,action)
		VALUES('alice-output',$1,'account',$2,'warn','apply'),
		      ('bob-output',$2,'account',$1,'warn','apply');
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$3),($2,'bob.test','bob.test',$3);
		INSERT INTO bluesky_profiles(did,record_cid) VALUES($1,'alice-bsky-cid'),($2,'bob-bsky-cid');
	`
	seedSQL = strings.NewReplacer(
		"$4", "'"+jobID.String()+"'",
		"$3", "'"+now.Format(time.RFC3339Nano)+"'",
		"$2", "'"+bob.String()+"'",
		"$1", "'"+alice.String()+"'",
	).Replace(seedSQL)
	_, err := pool.Exec(ctx, seedSQL)
	if err != nil {
		t.Fatalf("seed owner-private cleanup fixtures: %v", err)
	}

	component := NewDatabasePrivateCleanup(pool)
	if err := component.Purge(ctx, alice); err != nil {
		t.Fatalf("purge Alice private database state: %v", err)
	}
	if err := component.Purge(ctx, alice); err != nil {
		t.Fatalf("repeat Alice private database purge: %v", err)
	}

	for _, table := range []string{
		"craftsky_recent_searches", "actor_mutes", "account_language_preferences",
		"profile_customisations", "profile_pins", "saved_post_folders", "saved_posts",
		"notification_preferences", "notification_seen_state", "push_account_subscriptions",
	} {
		assertPrivateCleanupCount(t, pool, table, ownerColumnForPrivateCleanupTest(table), alice, 0)
		assertPrivateCleanupCount(t, pool, table, ownerColumnForPrivateCleanupTest(table), bob, 1)
	}
	assertPrivateCleanupCount(t, pool, "notification_events", "recipient_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "notification_events", "recipient_did", bob, 1)
	assertPrivateCleanupCount(t, pool, "moderation_reports", "reporter_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "moderation_reports", "reporter_did", bob, 0)
	assertPrivateCleanupCount(t, pool, "moderation_outputs", "source_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "moderation_outputs", "source_did", bob, 0)
	assertPrivateCleanupCount(t, pool, "push_installations", "id", "30000000-0000-4000-8000-000000000001", 1)

	// Public/indexer-owned projections, shared caches, and the bound deletion
	// authority survive this phase and are handled by their dedicated gates.
	assertPrivateCleanupCount(t, pool, "craftsky_profiles", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "craftsky_posts", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "atproto_identity_cache", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "bluesky_profiles", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "oauth_sessions", "account_did", alice, 1)
}

func ownerColumnForPrivateCleanupTest(table string) string {
	switch table {
	case "craftsky_recent_searches":
		return "viewer_did"
	case "account_language_preferences", "notification_preferences", "notification_seen_state", "push_account_subscriptions":
		return "account_did"
	default:
		return "owner_did"
	}
}

func assertPrivateCleanupCount(t *testing.T, pool *pgxpool.Pool, table, column string, value any, want int) {
	t.Helper()
	var got int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s=$1", table, column)
	if err := pool.QueryRow(context.Background(), query, value).Scan(&got); err != nil {
		t.Fatalf("count %s.%s: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s=%v count=%d want=%d", table, column, value, got, want)
	}
}

type fakePrivateCleanupComponent struct {
	name string
	run  func(syntax.DID) error
}

func (component fakePrivateCleanupComponent) Name() string { return component.name }

func (component fakePrivateCleanupComponent) Purge(_ context.Context, owner syntax.DID) error {
	return component.run(owner)
}
