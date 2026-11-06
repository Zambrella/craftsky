package api_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

func TestExtendedNetworkMembershipPaginationAndAttribution(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	for _, did := range []string{"viewer", "alice", "bob", "carol", "dana", "outsider"} {
		seedMember(t, pool, "did:plc:"+did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:bob", "bob")
	seedFollow(t, pool, "did:plc:alice", "did:plc:carol", "carol")
	seedFollow(t, pool, "did:plc:bob", "did:plc:carol", "carol")
	seedFollow(t, pool, "did:plc:alice", "did:plc:dana", "dana")
	seedFollow(t, pool, "did:plc:alice", "did:plc:viewer", "viewer")
	seedFollow(t, pool, "did:plc:alice", "did:plc:bob", "bob")
	base := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"viewer", "alice", "bob", "outsider"} {
		seedPost(t, pool, "did:plc:"+did, "excluded", "excluded", base.Add(time.Hour))
	}
	firstURI := seedPost(t, pool, "did:plc:dana", "one", "first", base)
	secondURI := seedPost(t, pool, "did:plc:carol", "two", "second", base)
	seedReplyPost(t, pool, "did:plc:carol", "reply", "reply", firstURI, firstURI, base.Add(time.Hour))
	seedInteraction(t, pool, "repost", "did:plc:carol", "repost", firstURI, false)
	imported := seedPost(t, pool, "did:plc:carol", "import", "import", base.Add(time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET external_import_source='instagram' WHERE uri=$1`, imported); err != nil {
		t.Fatal(err)
	}
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "avatar-cid")
	store := api.NewPostStore(pool)
	first, cursor, err := store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := timelineURIs(first); !slices.Equal(got, []string{firstURI}) || cursor == "" {
		t.Fatalf("first=%v cursor=%q", got, cursor)
	}
	if _, _, err := store.ListTimeline(ctx, "did:plc:viewer", 1, cursor); !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("extended cursor accepted by Following: %v", err)
	}
	second, next, err := store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 1, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if got := timelineURIs(second); !slices.Equal(got, []string{secondURI}) || next != "" {
		t.Fatalf("second=%v next=%q", got, next)
	}
	if second[0].FollowedBy == nil || second[0].FollowedBy.DID != "did:plc:alice" || *second[0].FollowedBy.DisplayName != "Alice" {
		t.Fatalf("attribution=%+v", second[0].FollowedBy)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:carol", "now-following")
	afterFollow, _, err := store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 1, cursor)
	if err != nil || len(afterFollow) != 0 {
		t.Fatalf("existing cursor after follow=%v err=%v", timelineURIs(afterFollow), err)
	}
	refreshed, _, err := store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 20, "")
	if err != nil || !slices.Equal(timelineURIs(refreshed), []string{firstURI}) {
		t.Fatalf("after follow=%v err=%v", timelineURIs(refreshed), err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM pds_set_aggregates WHERE kind='follow' AND actor_did='did:plc:alice' AND subject_did='did:plc:dana'`); err != nil {
		t.Fatal(err)
	}
	refreshed, _, err = store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 20, "")
	if err != nil || len(refreshed) != 0 {
		t.Fatalf("after path removal=%v err=%v", timelineURIs(refreshed), err)
	}
}

func TestExtendedNetworkVisibilityBeforePagination(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:viewer")
	base := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	names := []string{"allowed", "mutedauthor", "blockedauthor", "reverseblockedauthor", "hiddenpost", "hiddenauthor", "wronglanguage", "mutedbridge", "blockedbridge", "reverseblockedbridge", "hiddenbridge", "nonmemberbridge", "blockededge", "terminalauthor", "terminalbridge", "imagescan"}
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION appview_owner_is_terminal(candidate_did TEXT) RETURNS BOOLEAN LANGUAGE SQL IMMUTABLE AS $$ SELECT strpos(candidate_did, 'terminal') > 0 $$;
    CREATE OR REPLACE FUNCTION appview_image_subject_is_clear(candidate_uri TEXT, candidate_cid TEXT) RETURNS BOOLEAN LANGUAGE SQL IMMUTABLE AS $$ SELECT strpos(candidate_uri, 'imagescan') = 0 $$;`); err != nil {
		t.Fatal(err)
	}
	var want string
	for i, name := range names {
		bridge, author := "did:plc:bridge"+name, "did:plc:author"+name
		if name != "nonmemberbridge" {
			seedMember(t, pool, bridge)
		}
		seedMember(t, pool, author)
		seedFollow(t, pool, "did:plc:viewer", bridge, "bridge")
		seedFollow(t, pool, bridge, author, "author")
		uri := seedPost(t, pool, author, "post", name, base.Add(time.Duration(i)*time.Minute))
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET langs=ARRAY['en']::text[] WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
		switch name {
		case "allowed":
			want = uri
		case "mutedauthor", "mutedbridge":
			target := author
			if name == "mutedbridge" {
				target = bridge
			}
			if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes(owner_did,subject_did) VALUES('did:plc:viewer',$1)`, target); err != nil {
				t.Fatal(err)
			}
		case "blockedauthor", "blockedbridge":
			target := author
			if name == "blockedbridge" {
				target = bridge
			}
			seedBlockAggregate(t, pool, "did:plc:viewer", target, base)
		case "reverseblockedauthor", "reverseblockedbridge":
			target := author
			if name == "reverseblockedbridge" {
				target = bridge
			}
			seedBlockAggregate(t, pool, target, "did:plc:viewer", base)
		case "hiddenpost":
			seedModerationOutput(t, pool, "post", author, uri, "hide", base)
		case "hiddenauthor":
			seedModerationOutput(t, pool, "account", author, "", "takedown", base)
		case "hiddenbridge":
			seedModerationOutput(t, pool, "account", bridge, "", "hide", base)
		case "wronglanguage":
			if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET langs=ARRAY['fr']::text[] WHERE uri=$1`, uri); err != nil {
				t.Fatal(err)
			}
		case "blockededge":
			seedBlockAggregate(t, pool, author, bridge, base)
		}
	}
	items, next, err := api.NewPostStore(pool).ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", []string{"en"}, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(timelineURIs(items), []string{want}) || next != "" {
		t.Fatalf("items=%v next=%q", timelineURIs(items), next)
	}
}

func TestExtendedNetworkCursorIsFeedScoped(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	store := api.NewPostStore(pool)
	for _, cursor := range []string{"invalid", mustNetworkCursor(t, map[string]any{"activityAt": "2026-11-06T12:00:00Z", "itemKey": "post:foo"}), mustNetworkCursor(t, map[string]any{"feed": "extended-network", "activityAt": "bad", "itemKey": "post:foo"}), mustNetworkCursor(t, map[string]any{"feed": "extended-network", "activityAt": "2026-11-06T12:00:00Z", "itemKey": "repost:foo"})} {
		_, _, err := store.ListExtendedNetworkWithLanguages(context.Background(), "did:plc:viewer", nil, 20, cursor)
		if !errors.Is(err, envelope.ErrInvalidCursor) {
			t.Fatalf("cursor %q error=%v", cursor, err)
		}
	}
}

func mustNetworkCursor(t *testing.T, payload map[string]any) string {
	t.Helper()
	cursor, err := envelope.EncodeCursor(payload)
	if err != nil {
		t.Fatal(err)
	}
	return cursor
}

func TestExtendedNetworkUsesSurvivingVisibleConnection(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	for _, name := range []string{"viewer", "alice", "bob", "carol"} {
		seedMember(t, pool, "did:plc:"+name)
	}
	for _, name := range []string{"alice", "bob"} {
		seedFollow(t, pool, "did:plc:viewer", "did:plc:"+name, name)
		seedFollow(t, pool, "did:plc:"+name, "did:plc:carol", "carol")
	}
	uri := seedPost(t, pool, "did:plc:carol", "post", "Hello", time.Now().UTC())
	if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes(owner_did,subject_did) VALUES('did:plc:viewer','did:plc:alice')`); err != nil {
		t.Fatal(err)
	}
	store := api.NewPostStore(pool)
	items, _, err := store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(timelineURIs(items), []string{uri}) || items[0].FollowedBy.DID != "did:plc:bob" {
		t.Fatalf("items=%+v", items)
	}
	seedBlockAggregate(t, pool, "did:plc:bob", "did:plc:viewer", time.Now().UTC())
	items, _, err = store.ListExtendedNetworkWithLanguages(ctx, "did:plc:viewer", nil, 20, "")
	if err != nil || len(items) != 0 {
		t.Fatalf("all paths hidden items=%+v err=%v", items, err)
	}
}
