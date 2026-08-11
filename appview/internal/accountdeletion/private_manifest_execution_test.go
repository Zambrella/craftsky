package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/testdb"
)

func TestPrivateManifestExecutesProductionCleanupWithOwnerAndSharedControls(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000912")

	seedPrivateManifestExecutionFixture(t, pool, alice, bob, jobID, now)

	limiter, err := instagram.NewPostgresRateLimiter(pool, bytes.Repeat([]byte{0x4d}, 32), func() time.Time { return now })
	if err != nil {
		t.Fatalf("construct Instagram limiter: %v", err)
	}
	for _, owner := range []syntax.DID{alice, bob} {
		for _, scope := range []instagram.RateLimitScope{
			instagram.RateLimitChallengeDID,
			instagram.RateLimitConfirmationDID,
			instagram.RateLimitImportDID,
		} {
			if _, err := limiter.AllowIdentifier(ctx, scope, []byte(owner), time.Hour, 10); err != nil {
				t.Fatalf("seed %s rate bucket: %v", owner, err)
			}
		}
	}
	instagramPrivate := instagram.NewPrivateDataService(pool, limiter, func() time.Time { return now })
	instagramCleanup, err := NewNamedPrivateCleanup(
		"instagramPrivate",
		func(ctx context.Context, _ uuid.UUID, owner syntax.DID) error {
			return PurgeInstagramForAccountDeletion(ctx, instagramPrivate, owner)
		},
	)
	if err != nil {
		t.Fatalf("construct Instagram cleanup: %v", err)
	}
	scheduledStore := scheduledposts.NewStore(pool)
	scheduledCleanup := scheduledposts.NewAccountDeletion(pool, func() time.Time { return now })
	cleaner, err := NewPrivateCleaner(NewStore(pool, func() time.Time { return now }), []PrivateCleanupComponent{
		NewDatabasePrivateCleanup(pool),
		instagramCleanup,
		scheduledCleanup,
	})
	if err != nil {
		t.Fatalf("construct production private cleaner: %v", err)
	}

	if err := cleaner.Run(ctx, jobID, alice); err != nil {
		t.Fatalf("run Alice private cleanup: %v", err)
	}
	if err := cleaner.Run(ctx, jobID, alice); err != nil {
		t.Fatalf("replay Alice private cleanup: %v", err)
	}

	for _, component := range cleaner.ComponentNames() {
		assertPrivateCleanupCount(t, pool, "account_deletion_cleanup_steps", "component", component, 1)
	}
	for _, fixture := range []struct {
		table  string
		column string
	}{
		{"craftsky_recent_searches", "viewer_did"},
		{"actor_mutes", "owner_did"},
		{"account_language_preferences", "account_did"},
		{"saved_post_folders", "owner_did"},
		{"saved_posts", "owner_did"},
		{"profile_customisations", "owner_did"},
		{"profile_pins", "owner_did"},
		{"notification_preferences", "account_did"},
		{"notification_seen_state", "account_did"},
		{"push_account_subscriptions", "account_did"},
		{"instagram_verification_attempts", "owner_did"},
		{"instagram_account_links", "owner_did"},
		{"instagram_identity_claims", "owner_did"},
		{"instagram_graph_imports", "owner_did"},
	} {
		assertPrivateCleanupCount(t, pool, fixture.table, fixture.column, alice, 0)
		assertPrivateCleanupCount(t, pool, fixture.table, fixture.column, bob, 1)
	}
	assertPrivateCleanupCount(t, pool, "scheduled_post_media", "owner_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "scheduled_post_media", "owner_did", bob, 1)
	assertPrivateCleanupCount(t, pool, "instagram_graph_handles", "username_normalized", "alice.shared", 0)
	assertPrivateCleanupCount(t, pool, "instagram_graph_handles", "username_normalized", "bob.shared", 1)
	assertPrivateCleanupCount(t, pool, "push_installations", "device_id", "shared-device", 1)
	assertPrivateCleanupCount(t, pool, "push_installations", "device_id", "alice-only-device", 0)

	// Public/indexer-owned projections, shared identity caches, Bob's controls,
	// and the deletion-bound OAuth session must survive the private phase.
	for _, fixture := range []struct {
		table  string
		column string
		value  any
	}{
		{"craftsky_profiles", "did", alice},
		{"craftsky_posts", "did", alice},
		{"atproto_identity_cache", "did", alice},
		{"bluesky_profiles", "did", alice},
		{"oauth_sessions", "session_id", "alice-deletion-oauth"},
	} {
		assertPrivateCleanupCount(t, pool, fixture.table, fixture.column, fixture.value, 1)
	}

	objects := &manifestMemoryObjectStore{objects: map[string][]byte{
		"scheduled-media/alice": []byte("alice-private"),
		"scheduled-media/bob":   []byte("bob-private"),
	}}
	processor, err := scheduledposts.NewCleanupProcessor(scheduledposts.CleanupProcessorOptions{
		Store: scheduledStore, Objects: objects, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("construct scheduled object cleanup: %v", err)
	}
	processed, err := processor.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("delete queued scheduled objects: %v", err)
	}
	if processed != 1 {
		t.Fatalf("scheduled cleanup processed=%d, want 1", processed)
	}
	if objects.has("scheduled-media/alice") {
		t.Fatal("Alice scheduled object survived cleanup")
	}
	if !objects.has("scheduled-media/bob") {
		t.Fatal("Bob scheduled object was deleted")
	}
	assertPrivateCleanupCount(t, pool, "scheduled_post_cleanup_jobs", "object_key", "scheduled-media/alice", 0)
}

func seedPrivateManifestExecutionFixture(
	t *testing.T,
	pool *pgxpool.Pool,
	alice, bob syntax.DID,
	jobID uuid.UUID,
	now time.Time,
) {
	t.Helper()
	// Kept as one fixture so a single acceptance run exercises direct owner
	// rows, indirect/cascading rows, orphan-only resources, and retain controls.
	seedSQL := `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'alice-profile'),($2,'bob-profile');
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES
		 ('at://did:plc:alice/social.craftsky.feed.post/a',$1,'a','alice-post','alice','{}',$4);
		INSERT INTO oauth_sessions(account_did,session_id,data) VALUES
		 ($1,'alice-deletion-oauth','{}'),($1,'alice-ordinary-oauth','{}'),($2,'bob-oauth','{}');
		INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id) VALUES
		 (decode(repeat('11',32),'hex'),$1,'alice-ordinary-oauth'),
		 (decode(repeat('22',32),'hex'),$2,'bob-oauth');
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at,deletion_oauth_session_id)
		 VALUES($3,$1,'active','removingPrivateData',$4,'alice-deletion-oauth');
		INSERT INTO craftsky_recent_searches(id,viewer_did,search_type,display_label,normalized_payload,normalized_payload_hash) VALUES
		 ('alice-search',$1,'profile','Alice','{}','alice-hash'),('bob-search',$2,'profile','Bob','{}','bob-hash');
		INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2),($2,$1);
		INSERT INTO account_language_preferences(account_did,primary_language) VALUES($1,'en'),($2,'en');
		INSERT INTO profile_customisations(owner_did,colour,profile_border,profile_background) VALUES
		 ($1,'blue','none','plain'),($2,'red','none','plain');
		INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at) VALUES
		 ($1,'standard','at://did:plc:alice/social.craftsky.feed.post/a',gen_random_uuid(),$4,$4),
		 ($2,'standard','at://did:plc:alice/social.craftsky.feed.post/a',gen_random_uuid(),$4,$4);
		INSERT INTO saved_post_folders(id,owner_did,name,created_at,updated_at) VALUES
		 ('10000000-0000-4000-8000-000000000011',$1,'Alice',$4,$4),
		 ('10000000-0000-4000-8000-000000000012',$2,'Bob',$4,$4);
		INSERT INTO saved_posts(owner_did,post_uri,folder_id,saved_at) VALUES
		 ($1,'at://did:plc:alice/social.craftsky.feed.post/a','10000000-0000-4000-8000-000000000011',$4),
		 ($2,'at://did:plc:alice/social.craftsky.feed.post/a','10000000-0000-4000-8000-000000000012',$4);
		INSERT INTO notification_preferences(account_did,category,scope,push_enabled) VALUES
		 ($1,'like','everyone',true),($2,'like','everyone',true);
		INSERT INTO notification_seen_state(account_did,last_seen_revision) VALUES($1,1),($2,1);
		INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES
		 ('30000000-0000-4000-8000-000000000011','shared-device','ios','shared-token'),
		 ('30000000-0000-4000-8000-000000000012','alice-only-device','ios','alice-token');
		INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES
		 ('40000000-0000-4000-8000-000000000011','30000000-0000-4000-8000-000000000011',$1,'50000000-0000-4000-8000-000000000011'),
		 ('40000000-0000-4000-8000-000000000012','30000000-0000-4000-8000-000000000011',$2,'50000000-0000-4000-8000-000000000012'),
		 ('40000000-0000-4000-8000-000000000013','30000000-0000-4000-8000-000000000012',$1,'50000000-0000-4000-8000-000000000013');
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES
		 ($1,'alice.test','alice.test',$4),($2,'bob.test','bob.test',$4);
		INSERT INTO bluesky_profiles(did,record_cid) VALUES($1,'alice-bsky'),($2,'bob-bsky');

		INSERT INTO scheduled_post_media(id,owner_did,object_key,state,mime_type,size_bytes,sha256,blob_cid,unclaimed_expires_at) VALUES
		 ('60000000-0000-4000-8000-000000000011',$1,'scheduled-media/alice','ready','image/jpeg',4,decode(repeat('33',32),'hex'),'bafk-alice',$4 + interval '1 day'),
		 ('60000000-0000-4000-8000-000000000012',$2,'scheduled-media/bob','ready','image/jpeg',4,decode(repeat('44',32),'hex'),'bafk-bob',$4 + interval '1 day');

		INSERT INTO instagram_verification_attempts(id,owner_did,state,candidate_igsid,expires_at,processing_started_at,created_at,updated_at) VALUES
		 ('70000000-0000-4000-8000-000000000011',$1,'processing','alice-igsid',$4 + interval '1 hour',$4,$4,$4),
		 ('70000000-0000-4000-8000-000000000012',$2,'processing','bob-igsid',$4 + interval '1 hour',$4,$4,$4);
		INSERT INTO instagram_account_links(id,owner_did,state,igsid,igsid_digest_version,igsid_digest,username,username_normalized,discoverable,conflict_pending,verified_at,created_at,updated_at) VALUES
		 ('71000000-0000-4000-8000-000000000011',$1,'active','alice-igsid',1,decode(repeat('55',32),'hex'),'alice.ig','alice.ig',true,false,$4,$4,$4),
		 ('71000000-0000-4000-8000-000000000012',$2,'active','bob-igsid',1,decode(repeat('66',32),'hex'),'bob.ig','bob.ig',true,false,$4,$4,$4);
		INSERT INTO instagram_identity_claims(id,link_id,owner_did,state,igsid_digest_version,igsid_digest,claimed_at,created_at,updated_at) VALUES
		 ('72000000-0000-4000-8000-000000000011','71000000-0000-4000-8000-000000000011',$1,'active',1,decode(repeat('55',32),'hex'),$4,$4,$4),
		 ('72000000-0000-4000-8000-000000000012','71000000-0000-4000-8000-000000000012',$2,'active',1,decode(repeat('66',32),'hex'),$4,$4,$4);
		INSERT INTO instagram_graph_imports(id,owner_did,state,source_type,following_count,created_at,updated_at) VALUES
		 ('73000000-0000-4000-8000-000000000011',$1,'active','manual',1,$4,$4),
		 ('73000000-0000-4000-8000-000000000012',$2,'active','manual',1,$4,$4);
		INSERT INTO instagram_graph_handles(import_id,username_normalized,matched,created_at) VALUES
		 ('73000000-0000-4000-8000-000000000011','alice.shared',false,$4),
		 ('73000000-0000-4000-8000-000000000012','bob.shared',false,$4);
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome,created_at) VALUES
		 ($1,'fixture','owner',$1,'created',$4),($2,'fixture','owner',$2,'created',$4);
	`
	seedSQL = strings.NewReplacer(
		"$4", "'"+now.Format(time.RFC3339Nano)+"'::timestamptz",
		"$3", "'"+jobID.String()+"'",
		"$2", "'"+bob.String()+"'",
		"$1", "'"+alice.String()+"'",
	).Replace(seedSQL)
	_, err := pool.Exec(context.Background(), seedSQL)
	if err != nil {
		t.Fatalf("seed executable private manifest fixture: %v", err)
	}
}

type manifestMemoryObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (store *manifestMemoryObjectStore) Put(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.objects[key] = data
	return nil
}

func (store *manifestMemoryObjectStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	data, ok := store.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (store *manifestMemoryObjectStore) Delete(_ context.Context, key string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.objects, key)
	return nil
}

func (store *manifestMemoryObjectStore) has(key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.objects[key]
	return ok
}
