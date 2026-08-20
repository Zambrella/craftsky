package api_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/testdb"
)

// This is the pause-after-ACK barrier: Terminalize has committed the fixed
// tombstone/ledger, but no asynchronous component worker has removed a row.
// Every retained serving or behavioral reference must already be invisible.
func TestTerminalOwnerIsInvisibleAndIneffectiveBeforePhysicalPurge(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyTerminalVisibilityMigrations(t, pool)
	ctx := context.Background()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Now().UTC()
	lifecycle, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	viewer := syntax.DID("did:plc:terminal-viewer")
	terminal := syntax.DID("did:plc:terminal-actor")
	other := syntax.DID("did:plc:terminal-other")
	for _, did := range []syntax.DID{viewer, terminal, other} {
		onboarding, err := lifecycle.EnsureOnboardingOwner(ctx, did)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := lifecycle.Transition(ctx, ownerlifecycle.TransitionRequest{
			Owner: did, ExpectedGeneration: onboarding.Generation,
			To: ownerlifecycle.StateActive, Reason: "profileCreated",
		}); err != nil {
			t.Fatal(err)
		}
	}

	terminalURI := "at://" + terminal.String() + "/social.craftsky.feed.post/terminal-post"
	terminalReplyURI := "at://" + terminal.String() + "/social.craftsky.feed.post/terminal-reply"
	otherURI := "at://" + other.String() + "/social.craftsky.feed.post/other-post"
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid)
		VALUES($1,'viewer-profile'),($2,'terminal-profile'),($3,'other-profile')
	`, viewer, terminal, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles(did,display_name,record_cid)
		VALUES($1,'Viewer','viewer-bsky'),($2,'Terminal Actor','terminal-bsky'),
		      ($3,'Other Actor','other-bsky')
	`, viewer, terminal, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations(
			owner_did,colour,profile_border,profile_background
		) VALUES($1,'orchid','thin','skewdark')
	`, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'viewer.test','viewer.test',now()),
		      ($2,'terminal.test','terminal.test',now()),
		      ($3,'other.test','other.test',now())
	`, viewer, terminal, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(uri,did,rkey,cid,text,tags,record,created_at)
		VALUES($1,$2,'terminal-post','terminal-cid','terminal visible post',ARRAY['terminal'],'{}',now()),
		      ($3,$4,'other-post','other-cid','other visible post',ARRAY['other'],'{}',now())
	`, terminalURI, terminal, otherURI, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts(
			uri,did,rkey,cid,text,reply_root_uri,reply_root_cid,
			reply_parent_uri,reply_parent_cid,tags,record,created_at
		) VALUES($1,$2,'terminal-reply','terminal-reply-cid','terminal reply',
		         $3,'other-cid',$3,'other-cid','{}','{}',now())
	`, terminalReplyURI, terminal, otherURI); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_likes(uri,did,rkey,cid,subject_uri,subject_cid,record,created_at)
		VALUES('at://did:plc:terminal-actor/social.craftsky.feed.like/other',$1,
		       'other','terminal-like-cid',$2,'other-cid','{}',now())
	`, terminal, otherURI); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_reposts(uri,did,rkey,cid,subject_uri,subject_cid,record,created_at)
		VALUES('at://did:plc:terminal-actor/social.craftsky.feed.repost/other',$1,
		       'other','terminal-repost-cid',$2,'other-cid','{}',now())
	`, terminal, otherURI); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows(uri,did,rkey,cid,subject_did,record,created_at)
		VALUES('at://did:plc:terminal-viewer/app.bsky.graph.follow/terminal',
		       $1,'terminal','follow-cid',$2,'{}',now()),
		      ('at://did:plc:terminal-actor/app.bsky.graph.follow/other',
		       $2,'other','terminal-follow-cid',$3,'{}',now())
	`, viewer, terminal, other); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,$2)`, viewer, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks(uri,blocker_did,rkey,cid,subject_did,record,created_at)
		VALUES('at://did:plc:terminal-actor/app.bsky.graph.block/viewer',
		       $2,'viewer','block-cid',$1,'{}',now())
	`, viewer, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(
			id,source_did,subject_type,subject_did,subject_collection,
			subject_rkey,subject_uri,value,action
		) VALUES('terminal-moderation',$1,'post',$2,'social.craftsky.feed.post',
		         'other-post',$3,'hide','apply')
	`, terminal, other, otherURI); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO notification_events(
			id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,
			source_rkey,subject_uri,subject_cid,eligibility_scope,
			recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,
			activity_at,indexed_at,initial_push_evaluated_at
		) VALUES($1,$2,$3,'like',$4,$4,'terminal-cid','terminal-post',$4,
		         'terminal-cid','everyone',true,true,'active',now(),now(),now(),now())
	`, uuid.New(), viewer, terminal, terminalURI); err != nil {
		t.Fatal(err)
	}

	profiles := api.NewProfileStore(pool)
	posts := api.NewPostStore(pool, nil)
	search := api.NewSearchStore(pool, nil)
	follows := api.NewFollowStore(pool)
	customisations := api.NewProfileCustomisationStore(pool)
	relationshipStore := relationships.NewStore(pool)
	moderation, err := api.NewModerationStore(pool, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.Read(ctx, terminal.String(), viewer.String()); err != nil {
		t.Fatalf("pre-terminal profile fixture not visible: %v", err)
	}
	if _, err := posts.ReadOne(ctx, terminal.String(), "terminal-post"); err != nil {
		t.Fatalf("pre-terminal post fixture not visible: %v", err)
	}
	if _, err := posts.ReadOne(ctx, other.String(), "other-post"); !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("pre-terminal moderation fixture did not hide other post: %v", err)
	}
	preTerminalEngagement, err := posts.EngagementSummaries(ctx, viewer.String(), []string{otherURI})
	if err != nil {
		t.Fatal(err)
	}
	if got := preTerminalEngagement[otherURI]; got.LikeCount != 1 || got.RepostCount != 1 || got.ReplyCount != 1 {
		t.Fatalf("pre-terminal engagement fixture = %+v, want one like/repost/reply", got)
	}
	preTerminalOther, err := profiles.Read(ctx, other.String(), viewer.String())
	if err != nil {
		t.Fatal(err)
	}
	if preTerminalOther.FollowerCount == nil || *preTerminalOther.FollowerCount != 1 {
		t.Fatalf("pre-terminal follower count = %v, want 1", preTerminalOther.FollowerCount)
	}
	preTerminalPolicy, err := moderation.ActivePolicyForSubject(ctx, api.ModerationSubjectRef{
		Type: api.ModerationSubjectPost, DID: other.String(), URI: &otherURI,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !preTerminalPolicy.Hidden {
		t.Fatal("pre-terminal moderation fixture did not produce a hidden policy")
	}
	if row, err := follows.FindActiveFollow(ctx, viewer.String(), terminal.String()); err != nil || row == nil {
		t.Fatalf("pre-terminal follow fixture row=%+v err=%v", row, err)
	}
	preTerminalCustomisations, err := customisations.ReadBatch(ctx, []syntax.DID{terminal})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := preTerminalCustomisations[terminal]; !ok {
		t.Fatal("pre-terminal customisation fixture was not hydrated")
	}
	preTerminalQuotes, err := posts.QuoteViewRows(ctx, []api.ResponseStrongRef{{
		URI: terminalURI, CID: "terminal-cid",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := preTerminalQuotes[terminalURI]; got == nil || got.State != "visible" {
		t.Fatalf("pre-terminal quote fixture=%+v, want visible", got)
	}

	terminalState, err := lifecycle.Terminalize(ctx, ownerlifecycle.TerminalizeRequest{
		Owner: terminal, Reason: "identityDeleted",
	})
	if err != nil {
		t.Fatal(err)
	}
	var retainedRows int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM craftsky_profiles WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_posts WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_likes WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_reposts WHERE did=$1)
		     + (SELECT count(*) FROM atproto_follows WHERE did=$1 OR subject_did=$1)
		     + (SELECT count(*) FROM actor_mutes WHERE subject_did=$1)
		     + (SELECT count(*) FROM atproto_blocks WHERE blocker_did=$1)
		     + (SELECT count(*) FROM moderation_outputs WHERE source_did=$1)
		     + (SELECT count(*) FROM notification_events WHERE actor_did=$1)
	`, terminal).Scan(&retainedRows); err != nil {
		t.Fatal(err)
	}
	if retainedRows != 11 {
		t.Fatalf("pause-after-ACK fixture retained rows=%d, want 11", retainedRows)
	}

	if _, err := profiles.Read(ctx, terminal.String(), viewer.String()); !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("terminal profile read error=%v, want not found", err)
	}
	if _, err := posts.ReadOne(ctx, terminal.String(), "terminal-post"); !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("terminal direct post read error=%v, want not found", err)
	}
	timeline, _, err := posts.ListTimeline(ctx, viewer.String(), 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 0 {
		t.Fatalf("timeline exposed terminal rows: %+v", timeline)
	}
	profileResults, _, err := search.SearchProfiles(ctx, viewer.String(), api.ProfileSearchRequest{
		Query: "terminal", Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(profileResults) != 0 {
		t.Fatalf("profile search exposed terminal rows: %+v", profileResults)
	}
	postResults, _, err := search.SearchPosts(ctx, api.PostSearchRequest{
		Query: "terminal", Sort: api.SearchSortChronological, Limit: 20,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(postResults) != 0 {
		t.Fatalf("post search exposed terminal rows: %+v", postResults)
	}
	state, err := relationshipStore.State(ctx, viewer, terminal)
	if err != nil {
		t.Fatal(err)
	}
	if state != (relationships.State{}) {
		t.Fatalf("terminal relationship remained effective: %+v", state)
	}
	notifications, _, err := posts.ListNotifications(ctx, viewer.String(), 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(notifications) != 0 {
		t.Fatalf("notification feed exposed terminal actor: %+v", notifications)
	}
	newCount, err := posts.NotificationNewCount(ctx, viewer.String())
	if err != nil {
		t.Fatal(err)
	}
	if newCount != 0 {
		t.Fatalf("terminal notification new count=%d, want 0", newCount)
	}
	if _, err := posts.ReadOne(ctx, other.String(), "other-post"); err != nil {
		t.Fatalf("terminal moderation source remained effective: %v", err)
	}
	engagement, err := posts.EngagementSummaries(ctx, viewer.String(), []string{otherURI})
	if err != nil {
		t.Fatal(err)
	}
	if got := engagement[otherURI]; got.LikeCount != 0 || got.RepostCount != 0 || got.ReplyCount != 0 {
		t.Fatalf("terminal actor engagement remained effective: %+v", got)
	}
	otherProfile, err := profiles.Read(ctx, other.String(), viewer.String())
	if err != nil {
		t.Fatal(err)
	}
	if otherProfile.FollowerCount == nil || *otherProfile.FollowerCount != 0 {
		t.Fatalf("terminal follower remained effective: %v", otherProfile.FollowerCount)
	}
	policy, err := moderation.ActivePolicyForSubject(ctx, api.ModerationSubjectRef{
		Type: api.ModerationSubjectPost, DID: other.String(), URI: &otherURI,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if policy.Hidden || policy.Warning {
		t.Fatalf("terminal moderation source remained active: %+v", policy)
	}
	if row, err := follows.FindActiveFollow(ctx, viewer.String(), terminal.String()); err != nil || row != nil {
		t.Fatalf("terminal follow lookup row=%+v err=%v, want absent", row, err)
	}
	followedDIDs, err := follows.ListActiveFollowedDIDs(ctx, viewer.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(followedDIDs) != 0 {
		t.Fatalf("terminal followed DIDs remained visible: %v", followedDIDs)
	}
	terminalCustomisations, err := customisations.ReadBatch(ctx, []syntax.DID{terminal})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := terminalCustomisations[terminal]; ok {
		t.Fatalf("terminal customisation remained hydratable: %+v", terminalCustomisations)
	}
	terminalQuotes, err := posts.QuoteViewRows(ctx, []api.ResponseStrongRef{{
		URI: terminalURI, CID: "terminal-cid",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := terminalQuotes[terminalURI]; got == nil || got.State != "unavailable" {
		t.Fatalf("terminal quote existence leaked as %+v, want unavailable", got)
	}
	contextStates, err := posts.RequiredContextStates(ctx, viewer, []syntax.ATURI{syntax.ATURI(terminalURI)})
	if err != nil {
		t.Fatal(err)
	}
	if contextStates[syntax.ATURI(terminalURI)] {
		t.Fatal("terminal saved-post context remained eligible")
	}
	if err := lifecycle.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{
		Owner: terminal, Generation: terminalState.Generation,
	}}, func(context.Context) error {
		return errors.New("terminal effect callback must not execute")
	}); !errors.Is(err, ownerlifecycle.ErrOwnerNotActive) {
		t.Fatalf("terminal effect error=%v, want ErrOwnerNotActive", err)
	}

	purger, err := ownerlifecycle.NewTerminalPurgeProcessor(ownerlifecycle.TerminalPurgeProcessorConfig{
		Store: lifecycle, WorkerID: "terminal-visibility-drain", ComponentLimit: 100,
		RowBatchSize: 100, LeaseDuration: time.Minute, RetryDelay: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := purger.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("drain fixed terminal purge catalogue: processed=%d: %v", processed, err)
	}
	if processed != len(ownerlifecycle.TerminalPurgeCatalogue()) {
		t.Fatalf("processed components=%d, want fixed catalogue size %d", processed, len(ownerlifecycle.TerminalPurgeCatalogue()))
	}
	// Parent roles that own cascading children are deliberately rescheduled
	// rather than deleting an unbounded child fan-out. Advance the deterministic
	// worker clock and drain those bounded dependency rounds before asserting
	// physical convergence.
	for round := 0; round < len(ownerlifecycle.TerminalPurgeCatalogue()); round++ {
		state, err := lifecycle.Get(ctx, terminal)
		if err != nil {
			t.Fatal(err)
		}
		if state.PurgeCompletedAt != nil {
			break
		}
		clock = clock.Add(2 * time.Second)
		if _, err := purger.ProcessBatch(ctx); err != nil {
			t.Fatalf("drain terminal purge dependency round %d: %v", round+1, err)
		}
	}
	var remainingTerminalReferences int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM craftsky_profiles WHERE did=$1)
		     + (SELECT count(*) FROM bluesky_profiles WHERE did=$1)
		     + (SELECT count(*) FROM atproto_identity_cache WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_posts WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_likes WHERE did=$1)
		     + (SELECT count(*) FROM craftsky_reposts WHERE did=$1)
		     + (SELECT count(*) FROM atproto_follows WHERE did=$1 OR subject_did=$1)
		     + (SELECT count(*) FROM actor_mutes WHERE owner_did=$1 OR subject_did=$1)
		     + (SELECT count(*) FROM atproto_blocks WHERE blocker_did=$1 OR subject_did=$1)
		     + (SELECT count(*) FROM moderation_outputs WHERE source_did=$1 OR subject_did=$1)
		     + (SELECT count(*) FROM notification_events WHERE actor_did=$1 OR recipient_did=$1)
	`, terminal).Scan(&remainingTerminalReferences); err != nil {
		t.Fatal(err)
	}
	if remainingTerminalReferences != 0 {
		queries := map[string]string{
			"craftsky_profiles":      `SELECT count(*) FROM craftsky_profiles WHERE did=$1`,
			"bluesky_profiles":       `SELECT count(*) FROM bluesky_profiles WHERE did=$1`,
			"atproto_identity_cache": `SELECT count(*) FROM atproto_identity_cache WHERE did=$1`,
			"craftsky_posts":         `SELECT count(*) FROM craftsky_posts WHERE did=$1`,
			"craftsky_likes":         `SELECT count(*) FROM craftsky_likes WHERE did=$1`,
			"craftsky_reposts":       `SELECT count(*) FROM craftsky_reposts WHERE did=$1`,
			"atproto_follows":        `SELECT count(*) FROM atproto_follows WHERE did=$1 OR subject_did=$1`,
			"actor_mutes":            `SELECT count(*) FROM actor_mutes WHERE owner_did=$1 OR subject_did=$1`,
			"atproto_blocks":         `SELECT count(*) FROM atproto_blocks WHERE blocker_did=$1 OR subject_did=$1`,
			"moderation_outputs":     `SELECT count(*) FROM moderation_outputs WHERE source_did=$1 OR subject_did=$1`,
			"notification_events":    `SELECT count(*) FROM notification_events WHERE actor_did=$1 OR recipient_did=$1`,
		}
		remainingByTable := make(map[string]int)
		for table, query := range queries {
			var count int
			if err := pool.QueryRow(ctx, query, terminal).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count > 0 {
				remainingByTable[table] = count
			}
		}
		t.Fatalf("terminal physical references after bounded drain=%d (%v), want 0", remainingTerminalReferences, remainingByTable)
	}
	completed, err := lifecycle.Get(ctx, terminal)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != ownerlifecycle.StateTerminal || completed.PurgeCompletedAt == nil {
		t.Fatalf("completed terminal tombstone=%+v", completed)
	}
	var retainedLedger int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)::int FROM owner_purge_components
		WHERE owner_did=$1 AND owner_generation=$2
	`, terminal, terminalState.Generation).Scan(&retainedLedger); err != nil {
		t.Fatal(err)
	}
	if retainedLedger != len(ownerlifecycle.TerminalPurgeCatalogue()) {
		t.Fatalf("retained fixed purge ledger=%d, want %d", retainedLedger, len(ownerlifecycle.TerminalPurgeCatalogue()))
	}
}

func applyTerminalVisibilityMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			paths = append(paths, filepath.Join("../../migrations", entry.Name()))
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		switch filepath.Base(path) {
		case "000019_search_foundation.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("gin_trgm_ops"), []byte("public.gin_trgm_ops"))
		case "000024_saved_posts.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("ON DELETE SET NULL (folder_id)"), []byte("ON DELETE NO ACTION"))
		case "000041_account_deletion_safety_tombstones.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("UNIQUE NULLS NOT DISTINCT ("), []byte("UNIQUE ("))
			sql = append(sql, []byte(`
				CREATE UNIQUE INDEX account_deletion_safety_tombstones_null_upload_terminal_visibility_idx
					ON account_deletion_safety_tombstones(operation_id,kind,exact_key)
					WHERE upload_generation IS NULL;
			`)...)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}
