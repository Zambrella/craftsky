package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

const postInteractionIdentityDDL = `
CREATE TABLE atproto_identity_cache (
	did TEXT PRIMARY KEY,
	handle TEXT NOT NULL,
	handle_lower TEXT NOT NULL UNIQUE,
	resolved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

const postInteractionPolicyLifecycleDDL = `
CREATE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
	SELECT COALESCE((SELECT state = 'terminal' FROM owner_lifecycles WHERE owner_did = candidate_did), false)
$$;
CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
	SELECT COALESCE((SELECT state = 'active' FROM owner_lifecycles WHERE owner_did = candidate_did), false)
$$;
`

// IT-001: FR-003, RULE-001, RULE-003; AC-003, AC-010.
func TestPostStore_ListPostInteractionAccounts_Likes(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:alice")
	actors := []syntax.DID{"did:plc:dana", "did:plc:carol", "did:plc:bob", "did:plc:deleted"}

	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	for _, actor := range actors {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), actor.String(), "")
		handle := actor.String()[len("did:plc:"):] + ".test"
		if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
			t.Fatalf("seed identity %s: %v", actor, err)
		}
	}

	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	rootURI := seedPost(t, pool, owner.String(), "root", "root", now)
	commentURI := seedReplyPost(t, pool, owner.String(), "comment", "comment", rootURI, rootURI, now.Add(time.Minute))
	replyURI := seedReplyPost(t, pool, owner.String(), "reply", "reply", rootURI, commentURI, now.Add(2*time.Minute))
	targets := []struct {
		name string
		rkey syntax.RecordKey
		uri  string
	}{
		{name: "root", rkey: syntax.RecordKey("root"), uri: rootURI},
		{name: "comment", rkey: syntax.RecordKey("comment"), uri: commentURI},
		{name: "nested reply", rkey: syntax.RecordKey("reply"), uri: replyURI},
	}

	for targetIndex, target := range targets {
		for actorIndex, actor := range actors[:3] {
			rkey := fmt.Sprintf("t%d-%d", targetIndex, actorIndex)
			seedInteraction(t, pool, "like", actor.String(), rkey, target.uri, false)
		}
		seedInteraction(t, pool, "like", actors[3].String(), fmt.Sprintf("deleted-%d", targetIndex), target.uri, true)
	}

	store := api.NewPostStore(pool)
	for _, tc := range targets {
		t.Run(tc.name, func(t *testing.T) {
			target, err := store.ResolveInteractionTarget(ctx, viewer, owner, tc.rkey, api.PostInteractionLikes)
			if err != nil {
				t.Fatalf("resolve target: %v", err)
			}
			first, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, 2, "")
			if err != nil {
				t.Fatalf("first page: %v", err)
			}
			if first.TotalCount != 3 {
				t.Fatalf("totalCount = %d, want 3", first.TotalCount)
			}
			if first.Cursor == nil || *first.Cursor == "" {
				t.Fatal("first cursor is absent")
			}
			assertAccountDIDs(t, first.Items, []syntax.DID{actors[0], actors[1]})

			last, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, 2, *first.Cursor)
			if err != nil {
				t.Fatalf("last page: %v", err)
			}
			if last.TotalCount != 3 {
				t.Fatalf("last totalCount = %d, want 3", last.TotalCount)
			}
			assertAccountDIDs(t, last.Items, []syntax.DID{actors[2]})
			if last.Cursor != nil {
				t.Fatalf("last cursor = %q, want omitted", *last.Cursor)
			}
		})
	}
}

// IT-002, UT-003: BR-002, FR-004, FR-010, RULE-001, RULE-002, RULE-003;
// AC-003, AC-004, AC-010, AC-011.
func TestPostStore_ListPostInteractionAccounts_Reposts(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:alice")
	quoteOnly := syntax.DID("did:plc:quote-only")
	repostOnly := syntax.DID("did:plc:repost-only")
	dual := syntax.DID("did:plc:dual")
	deleted := syntax.DID("did:plc:deleted")

	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	for _, actor := range []syntax.DID{quoteOnly, repostOnly, dual, deleted} {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), actor.String(), "")
		handle := actor.String()[len("did:plc:"):] + ".test"
		if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
			t.Fatalf("seed identity %s: %v", actor, err)
		}
	}

	base := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	targetURI := seedPost(t, pool, owner.String(), "root", "root", base)
	seedQuotePost(t, pool, quoteOnly.String(), "quote-only-1", "quote only", targetURI, "bafyroot", base.Add(time.Minute))
	seedQuotePost(t, pool, quoteOnly.String(), "quote-only-2", "another quote", targetURI, "bafyroot", base.Add(2*time.Minute))
	seedQuotePost(t, pool, dual.String(), "dual-quote", "dual quote", targetURI, "bafyroot", base.Add(3*time.Minute))
	repostOnlyURI := seedInteraction(t, pool, "repost", repostOnly.String(), "repost-only", targetURI, false)
	dualURI := seedInteraction(t, pool, "repost", dual.String(), "dual-repost", targetURI, false)
	seedInteraction(t, pool, "repost", deleted.String(), "deleted-repost", targetURI, true)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_reposts SET created_at = $1 WHERE uri = $2`, base.Add(4*time.Minute), repostOnlyURI); err != nil {
		t.Fatalf("set repost-only order: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_reposts SET created_at = $1 WHERE uri = $2`, base.Add(5*time.Minute), dualURI); err != nil {
		t.Fatalf("set dual order: %v", err)
	}

	store := api.NewPostStore(pool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionReposts)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	page, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionReposts, 10, "")
	if err != nil {
		t.Fatalf("list reposts: %v", err)
	}
	if page.TotalCount != 2 {
		t.Fatalf("totalCount = %d, want 2", page.TotalCount)
	}
	assertAccountDIDs(t, page.Items, []syntax.DID{dual, repostOnly})
	if page.Cursor != nil {
		t.Fatalf("cursor = %q, want omitted", *page.Cursor)
	}

	quoteTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve quote target: %v", err)
	}
	quotes, quoteCursor, err := store.ListQuotePosts(ctx, viewer, quoteTarget, []string{}, 10, "")
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if quoteCursor != "" {
		t.Fatalf("quote cursor = %q, want omitted", quoteCursor)
	}
	assertPostRowURIs(t, quotes, []string{
		"at://did:plc:dual/social.craftsky.feed.post/dual-quote",
		"at://did:plc:quote-only/social.craftsky.feed.post/quote-only-2",
		"at://did:plc:quote-only/social.craftsky.feed.post/quote-only-1",
	})

	for _, tc := range []struct {
		viewer             syntax.DID
		wantViewerReposted bool
	}{
		{viewer: dual, wantViewerReposted: true},
		{viewer: quoteOnly, wantViewerReposted: false},
	} {
		summaries, summaryErr := store.EngagementSummaries(ctx, tc.viewer.String(), []string{}, []string{targetURI})
		if summaryErr != nil {
			t.Fatalf("engagement summary for %s: %v", tc.viewer, summaryErr)
		}
		summary := summaries[targetURI]
		if summary.RepostCount != 2 || summary.QuoteCount != 3 || summary.ViewerHasReposted != tc.wantViewerReposted {
			t.Fatalf("engagement summary for %s = %+v, want reposts 2 quotes 3 viewerHasReposted %v", tc.viewer, summary, tc.wantViewerReposted)
		}
	}
}

// IT-003: BR-002, FR-006, RULE-001, RULE-002, RULE-003;
// AC-004, AC-010, AC-011.
func TestPostStore_ListQuotePosts(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:alice")
	dana := syntax.DID("did:plc:dana")
	carol := syntax.DID("did:plc:carol")
	hiddenAuthor := syntax.DID("did:plc:hidden")
	deletedAuthor := syntax.DID("did:plc:deleted")

	for _, actor := range []syntax.DID{viewer, owner, dana, carol, hiddenAuthor, deletedAuthor} {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), actor.String()+" display", "")
	}

	base := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	targetURI := seedPost(t, pool, owner.String(), "root", "root", base)
	seedReplyPost(t, pool, owner.String(), "reply-target", "reply target", targetURI, targetURI, base.Add(30*time.Second))
	danaOlder := seedQuotePost(t, pool, dana.String(), "dana-older", "first Dana quote", targetURI, "bafyroot", base.Add(time.Minute))
	carolQuote := seedQuotePost(t, pool, carol.String(), "carol", "Carol quote", targetURI, "bafyroot", base.Add(2*time.Minute))
	danaNewest := seedQuotePost(t, pool, dana.String(), "dana-newest", "second Dana quote", targetURI, "bafyroot", base.Add(3*time.Minute))
	hidden := seedQuotePost(t, pool, hiddenAuthor.String(), "hidden", "hidden quote", targetURI, "bafyroot", base.Add(4*time.Minute))
	deleted := seedQuotePost(t, pool, deletedAuthor.String(), "deleted", "deleted quote", targetURI, "bafyroot", base.Add(5*time.Minute))
	seedModerationOutput(t, pool, "post", hiddenAuthor.String(), hidden, "hide", base.Add(6*time.Minute))
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri = $1`, deleted); err != nil {
		t.Fatalf("delete quote post: %v", err)
	}

	store := api.NewPostStore(pool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	first, firstCursor, err := store.ListQuotePosts(ctx, viewer, target, []string{}, 2, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	assertPostRowURIs(t, first, []string{danaNewest, carolQuote})
	if firstCursor == "" {
		t.Fatal("first cursor is absent")
	}

	newest := first[0]
	if newest.DID != dana.String() || newest.Rkey != "dana-newest" || newest.Text != "second Dana quote" {
		t.Fatalf("newest quote row = %+v, want existing post row fields", newest)
	}
	if newest.QuoteURI == nil || *newest.QuoteURI != targetURI || newest.QuoteCID == nil || *newest.QuoteCID != "bafyroot" {
		t.Fatalf("newest quote ref = (%v, %v), want (%s, bafyroot)", newest.QuoteURI, newest.QuoteCID, targetURI)
	}
	if newest.ReplyRootURI != nil || newest.ReplyParentURI != nil || newest.AuthorDisplayName == nil || *newest.AuthorDisplayName != dana.String()+" display" {
		t.Fatalf("newest quote one-level row shape = %+v", newest)
	}

	last, lastCursor, err := store.ListQuotePosts(ctx, viewer, target, []string{}, 2, firstCursor)
	if err != nil {
		t.Fatalf("last page: %v", err)
	}
	assertPostRowURIs(t, last, []string{danaOlder})
	if lastCursor != "" {
		t.Fatalf("last cursor = %q, want omitted", lastCursor)
	}

	if _, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("reply-target"), api.PostInteractionQuotes); err != api.ErrPostNotFound {
		t.Fatalf("resolve reply target error = %v, want %v", err, api.ErrPostNotFound)
	}
}

// UT-002, UT-008: FR-009, NFR-002; AC-010, AC-016.
func TestPostInteractionCursorValidation(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:alice")
	actorA := syntax.DID("did:plc:actor-a")
	actorB := syntax.DID("did:plc:actor-b")
	for _, actor := range []syntax.DID{viewer, owner, actorA, actorB} {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), actor.String(), "")
		if actor != viewer && actor != owner {
			handle := actor.String()[len("did:plc:"):] + ".test"
			if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
				t.Fatalf("seed identity %s: %v", actor, err)
			}
		}
	}

	createdAt := time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC)
	firstTargetURI := seedPost(t, pool, owner.String(), "first", "first", createdAt)
	seedPost(t, pool, owner.String(), "second", "second", createdAt)
	for _, actor := range []syntax.DID{actorA, actorB} {
		likeURI := seedInteraction(t, pool, "like", actor.String(), "like", firstTargetURI, false)
		repostURI := seedInteraction(t, pool, "repost", actor.String(), "repost", firstTargetURI, false)
		if _, err := pool.Exec(ctx, `UPDATE craftsky_likes SET created_at = $1 WHERE uri = $2`, createdAt, likeURI); err != nil {
			t.Fatalf("set like order: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_reposts SET created_at = $1 WHERE uri = $2`, createdAt, repostURI); err != nil {
			t.Fatalf("set repost order: %v", err)
		}
		seedQuotePost(t, pool, actor.String(), "quote", "quote", firstTargetURI, "bafyroot", createdAt)
	}

	store := api.NewPostStore(pool)
	likesTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("first"), api.PostInteractionLikes)
	if err != nil {
		t.Fatalf("resolve likes target: %v", err)
	}
	repostsTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("first"), api.PostInteractionReposts)
	if err != nil {
		t.Fatalf("resolve reposts target: %v", err)
	}
	quotesTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("first"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve quotes target: %v", err)
	}
	otherTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("second"), api.PostInteractionLikes)
	if err != nil {
		t.Fatalf("resolve other target: %v", err)
	}

	likesFirst, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, "")
	if err != nil || likesFirst.Cursor == nil {
		t.Fatalf("likes first page cursor = %v, err = %v", likesFirst.Cursor, err)
	}
	quotesFirst, quotesCursor, err := store.ListQuotePosts(ctx, viewer, quotesTarget, []string{}, 1, "")
	if err != nil || len(quotesFirst) != 1 || quotesCursor == "" {
		t.Fatalf("quotes first page = %d, cursor = %q, err = %v", len(quotesFirst), quotesCursor, err)
	}

	t.Run("cursor payload binds exact kind and target", func(t *testing.T) {
		payload, err := envelope.DecodeCursor(*likesFirst.Cursor)
		if err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["kind"] != string(api.PostInteractionLikes) || payload["target"] != likesTarget.URI.String() {
			t.Fatalf("cursor binding = kind %v target %v, want %s %s", payload["kind"], payload["target"], api.PostInteractionLikes, likesTarget.URI)
		}
	})

	t.Run("valid cursors decode and continue", func(t *testing.T) {
		likesLast, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, *likesFirst.Cursor)
		if err != nil || len(likesLast.Items) != 1 {
			t.Fatalf("likes continuation items = %d, err = %v", len(likesLast.Items), err)
		}
		quotesLast, _, err := store.ListQuotePosts(ctx, viewer, quotesTarget, []string{}, 1, quotesCursor)
		if err != nil || len(quotesLast) != 1 {
			t.Fatalf("quotes continuation items = %d, err = %v", len(quotesLast), err)
		}
	})

	badTimestamp, err := envelope.EncodeCursor(map[string]any{
		"kind":      string(api.PostInteractionLikes),
		"target":    likesTarget.URI.String(),
		"createdAt": "yesterday",
		"uri":       "at://did:plc:actor-a/social.craftsky.feed.like/like",
	})
	if err != nil {
		t.Fatalf("encode bad timestamp cursor: %v", err)
	}
	missingURI, err := envelope.EncodeCursor(map[string]any{
		"kind":      string(api.PostInteractionLikes),
		"target":    likesTarget.URI.String(),
		"createdAt": createdAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("encode missing URI cursor: %v", err)
	}

	for _, tc := range []struct {
		name string
		list func(string) error
		cur  string
	}{
		{
			name: "malformed likes",
			cur:  "not-base64",
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, cursor)
				return err
			},
		},
		{
			name: "bad timestamp",
			cur:  badTimestamp,
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, cursor)
				return err
			},
		},
		{
			name: "missing URI",
			cur:  missingURI,
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, cursor)
				return err
			},
		},
		{
			name: "malformed reposts",
			cur:  "not-base64",
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, repostsTarget, api.PostInteractionReposts, 1, cursor)
				return err
			},
		},
		{
			name: "malformed quotes",
			cur:  "not-base64",
			list: func(cursor string) error {
				_, _, err := store.ListQuotePosts(ctx, viewer, quotesTarget, []string{}, 1, cursor)
				return err
			},
		},
		{
			name: "likes cursor on reposts",
			cur:  *likesFirst.Cursor,
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, repostsTarget, api.PostInteractionReposts, 1, cursor)
				return err
			},
		},
		{
			name: "likes cursor on quotes",
			cur:  *likesFirst.Cursor,
			list: func(cursor string) error {
				_, _, err := store.ListQuotePosts(ctx, viewer, quotesTarget, []string{}, 1, cursor)
				return err
			},
		},
		{
			name: "quotes cursor on likes",
			cur:  quotesCursor,
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, likesTarget, api.PostInteractionLikes, 1, cursor)
				return err
			},
		},
		{
			name: "likes cursor on another target",
			cur:  *likesFirst.Cursor,
			list: func(cursor string) error {
				_, err := store.ListPostInteractionAccounts(ctx, viewer, otherTarget, api.PostInteractionLikes, 1, cursor)
				return err
			},
		},
		{
			name: "quotes cursor on another target",
			cur:  quotesCursor,
			list: func(cursor string) error {
				otherQuotesTarget := *otherTarget
				_, _, err := store.ListQuotePosts(ctx, viewer, &otherQuotesTarget, []string{}, 1, cursor)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.list(tc.cur); !errors.Is(err, envelope.ErrInvalidCursor) {
				t.Fatalf("error = %v, want invalid cursor", err)
			}
		})
	}
}

// IT-010: FR-003, FR-004, FR-006, NFR-002; AC-010.
func TestPostStore_PostInteractionPaginationUsesCreatedAtAndURIDesc(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:alice")
	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	createdAt := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	targetURI := seedPost(t, pool, owner.String(), "root", "root", createdAt.Add(-time.Hour))

	actors := []syntax.DID{
		"did:plc:actor-a",
		"did:plc:actor-b",
		"did:plc:actor-c",
		"did:plc:actor-d",
		"did:plc:actor-e",
	}
	quoteURIs := make([]string, 0, len(actors))
	for i, actor := range actors {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), actor.String(), "")
		handle := actor.String()[len("did:plc:"):] + ".test"
		if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
			t.Fatalf("seed identity %s: %v", actor, err)
		}
		likeURI := seedInteraction(t, pool, "like", actor.String(), "like", targetURI, false)
		repostURI := seedInteraction(t, pool, "repost", actor.String(), "repost", targetURI, false)
		interactionTime := createdAt
		if i == 0 {
			interactionTime = createdAt.Add(time.Minute)
		} else if i == len(actors)-1 {
			interactionTime = createdAt.Add(-time.Minute)
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_likes SET created_at = $1 WHERE uri = $2`, interactionTime, likeURI); err != nil {
			t.Fatalf("set like order: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_reposts SET created_at = $1 WHERE uri = $2`, interactionTime, repostURI); err != nil {
			t.Fatalf("set repost order: %v", err)
		}
		quoteURIs = append(quoteURIs, seedQuotePost(t, pool, actor.String(), "quote", "quote", targetURI, "bafyroot", interactionTime))
	}

	store := api.NewPostStore(pool)
	likesTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionLikes)
	if err != nil {
		t.Fatalf("resolve likes target: %v", err)
	}
	repostsTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionReposts)
	if err != nil {
		t.Fatalf("resolve reposts target: %v", err)
	}
	quotesTarget, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve quotes target: %v", err)
	}

	wantDIDs := []syntax.DID{actors[0], actors[3], actors[2], actors[1], actors[4]}
	wantQuoteURIs := []string{quoteURIs[0], quoteURIs[3], quoteURIs[2], quoteURIs[1], quoteURIs[4]}

	for _, tc := range []struct {
		name       string
		kind       api.PostInteractionKind
		target     *api.PostInteractionTarget
		wantDIDs   []syntax.DID
		wantQuotes []string
	}{
		{name: "likes", kind: api.PostInteractionLikes, target: likesTarget, wantDIDs: wantDIDs},
		{name: "reposts", kind: api.PostInteractionReposts, target: repostsTarget, wantDIDs: wantDIDs},
		{name: "quotes", kind: api.PostInteractionQuotes, target: quotesTarget, wantQuotes: wantQuoteURIs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.kind == api.PostInteractionQuotes {
				assertQuotePagination(t, ctx, store, tc.target, tc.wantQuotes)
				return
			}
			assertAccountPagination(t, ctx, store, tc.target, tc.kind, tc.wantDIDs)
		})
	}
}

func TestPostStore_IR005EmptyContinuationRetainsAuthoritativeTotal(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:owner")
	newer := syntax.DID("did:plc:newer")
	older := syntax.DID("did:plc:older")
	for _, did := range []syntax.DID{viewer, owner, newer, older} {
		seedMember(t, pool, did.String())
	}
	for _, actor := range []syntax.DID{newer, older} {
		seedBskyProfile(t, pool, actor.String(), actor.String(), "")
		handle := actor.String()[len("did:plc:"):] + ".test"
		if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
			t.Fatalf("seed identity %s: %v", actor, err)
		}
	}
	targetURI := seedPost(t, pool, owner.String(), "root", "root", time.Now().UTC())
	newerURI := seedInteraction(t, pool, "like", newer.String(), "newer", targetURI, false)
	olderURI := seedInteraction(t, pool, "like", older.String(), "older", targetURI, false)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_likes SET created_at = CASE uri WHEN $1 THEN $3::timestamptz ELSE $4::timestamptz END WHERE uri IN ($1, $2)`,
		newerURI, olderURI, time.Now().UTC(), time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("set interaction order: %v", err)
	}

	store := api.NewPostStore(pool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionLikes)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	first, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, 1, "")
	if err != nil || first.Cursor == nil || first.TotalCount != 2 {
		t.Fatalf("first page = %+v, error %v", first, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_likes SET deleted_at = now() WHERE uri = $1`, olderURI); err != nil {
		t.Fatalf("delete continuation interaction: %v", err)
	}
	second, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, 1, *first.Cursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 0 || second.Cursor != nil || second.TotalCount != 1 {
		t.Fatalf("empty continuation = %+v, want no items/cursor and total 1", second)
	}
}

// IT-004, IT-012: FR-005, FR-010; AC-012, AC-017.
func TestPostStore_PostInteractionAccountPolicyAndCountAlignment(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL+postInteractionPolicyLifecycleDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:owner")
	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	targetURI := seedPost(t, pool, owner.String(), "root", "root", time.Now().UTC())

	actors := map[string]syntax.DID{
		"current":             "did:plc:current",
		"former":              "did:plc:former",
		"terminal":            "did:plc:terminal",
		"muted":               "did:plc:muted",
		"warned":              "did:plc:warned",
		"viewer blocks actor": "did:plc:viewer-blocks",
		"actor blocks viewer": "did:plc:blocks-viewer",
		"actor blocks author": "did:plc:blocks-author",
		"author blocks actor": "did:plc:author-blocks",
		"hidden":              "did:plc:hidden",
		"takedown":            "did:plc:takedown",
	}
	for name, actor := range actors {
		seedMember(t, pool, actor.String())
		seedBskyProfile(t, pool, actor.String(), name, "")
		handle := actor.String()[len("did:plc:"):] + ".test"
		if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, handle); err != nil {
			t.Fatalf("seed identity %s: %v", name, err)
		}
		seedInteraction(t, pool, "like", actor.String(), "policy-like", targetURI, false)
		seedInteraction(t, pool, "repost", actor.String(), "policy-repost", targetURI, false)
	}
	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state = 'departed' WHERE owner_did = $1`, actors["former"]); err != nil {
		t.Fatalf("mark former: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state = 'terminal', terminal_at = now() WHERE owner_did = $1`, actors["terminal"]); err != nil {
		t.Fatalf("mark terminal: %v", err)
	}
	seedMute(t, pool, viewer, actors["muted"])
	seedBlock(t, pool, viewer, actors["viewer blocks actor"], "viewer-blocks")
	seedBlock(t, pool, actors["actor blocks viewer"], viewer, "blocks-viewer")
	seedBlock(t, pool, actors["actor blocks author"], owner, "blocks-author")
	seedBlock(t, pool, owner, actors["author blocks actor"], "author-blocks")
	base := time.Now().UTC()
	seedModerationOutput(t, pool, "account", actors["hidden"].String(), "", "hide", base)
	seedModerationOutput(t, pool, "account", actors["takedown"].String(), "", "takedown", base.Add(time.Second))
	seedModerationOutput(t, pool, "account", actors["warned"].String(), "", "warn", base.Add(2*time.Second))

	store := api.NewPostStore(pool)
	want := []syntax.DID{actors["warned"], actors["muted"], actors["current"]}
	for _, kind := range []api.PostInteractionKind{api.PostInteractionLikes, api.PostInteractionReposts} {
		target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), kind)
		if err != nil {
			t.Fatalf("resolve %s: %v", kind, err)
		}
		page, err := store.ListPostInteractionAccounts(ctx, viewer, target, kind, 20, "")
		if err != nil {
			t.Fatalf("list %s: %v", kind, err)
		}
		if page.TotalCount != len(want) {
			t.Fatalf("%s totalCount = %d, want %d", kind, page.TotalCount, len(want))
		}
		got := make([]syntax.DID, 0, len(page.Items))
		for _, item := range page.Items {
			got = append(got, item.DID)
			if item.DID == actors["muted"] && !item.Muted {
				t.Fatal("muted eligible actor lost relationship state")
			}
		}
		if !sameDIDSet(got, want) {
			t.Fatalf("%s actors = %v, want set %v", kind, got, want)
		}
	}

	summaries, err := store.EngagementSummaries(ctx, viewer.String(), []string{}, []string{targetURI})
	if err != nil {
		t.Fatalf("engagement summaries: %v", err)
	}
	if got := summaries[targetURI]; got.LikeCount != len(want) || got.RepostCount != len(want) {
		t.Fatalf("engagement counts = likes %d reposts %d, want %d each", got.LikeCount, got.RepostCount, len(want))
	}
}

// IT-004, IT-012: FR-010; AC-017.
func TestPostStore_QuotePolicyAndCountAlignment(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionPolicyLifecycleDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:owner")
	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	targetURI := seedPost(t, pool, owner.String(), "root", "root", time.Now().UTC())

	authors := map[string]syntax.DID{
		"visible":             "did:plc:quote-visible",
		"muted":               "did:plc:quote-muted",
		"warned":              "did:plc:quote-warned",
		"viewer blocked":      "did:plc:quote-viewer-blocked",
		"blocks viewer":       "did:plc:quote-blocks-viewer",
		"target blocked":      "did:plc:quote-target-blocked",
		"blocked by target":   "did:plc:quote-blocked-by-target",
		"hidden account":      "did:plc:quote-hidden-account",
		"takedown account":    "did:plc:quote-takedown-account",
		"hidden post":         "did:plc:quote-hidden-post",
		"takedown post":       "did:plc:quote-takedown-post",
		"language ineligible": "did:plc:quote-french",
		"terminal":            "did:plc:quote-terminal",
	}
	quotes := make(map[string]string, len(authors))
	base := time.Now().UTC().Add(time.Minute)
	i := 0
	for name, author := range authors {
		seedMember(t, pool, author.String())
		seedBskyProfile(t, pool, author.String(), name, "")
		quotes[name] = seedQuotePost(t, pool, author.String(), "quote", name, targetURI, "bafyroot", base.Add(time.Duration(i)*time.Second))
		i++
		langs := []string{"en"}
		if name == "language ineligible" {
			langs = []string{"fr"}
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET langs = $2 WHERE uri = $1`, quotes[name], langs); err != nil {
			t.Fatalf("seed %s languages: %v", name, err)
		}
	}
	seedMute(t, pool, viewer, authors["muted"])
	seedBlock(t, pool, viewer, authors["viewer blocked"], "quote-viewer-blocked")
	seedBlock(t, pool, authors["blocks viewer"], viewer, "quote-blocks-viewer")
	seedBlock(t, pool, authors["target blocked"], owner, "quote-target-blocked")
	seedBlock(t, pool, owner, authors["blocked by target"], "quote-blocked-by-target")
	seedModerationOutput(t, pool, "account", authors["hidden account"].String(), "", "hide", base.Add(time.Minute))
	seedModerationOutput(t, pool, "account", authors["takedown account"].String(), "", "takedown", base.Add(2*time.Minute))
	seedModerationOutput(t, pool, "post", authors["hidden post"].String(), quotes["hidden post"], "hide", base.Add(3*time.Minute))
	seedModerationOutput(t, pool, "post", authors["takedown post"].String(), quotes["takedown post"], "takedown", base.Add(4*time.Minute))
	seedModerationOutput(t, pool, "account", authors["warned"].String(), "", "warn", base.Add(5*time.Minute))
	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET state = 'terminal', terminal_at = now() WHERE owner_did = $1`, authors["terminal"]); err != nil {
		t.Fatalf("mark quote author terminal: %v", err)
	}

	store := api.NewPostStore(pool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve quotes target: %v", err)
	}
	rows, cursor, err := store.ListQuotePosts(ctx, viewer, target, []string{"en"}, 20, "")
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want exhausted", cursor)
	}
	want := []string{quotes["visible"], quotes["muted"], quotes["warned"]}
	if got := postRowURIs(rows); !sameStringSet(got, want) {
		t.Fatalf("quote rows = %v, want set %v", got, want)
	}

	summaries, err := store.EngagementSummaries(ctx, viewer.String(), []string{"en"}, []string{targetURI})
	if err != nil {
		t.Fatalf("engagement summaries: %v", err)
	}
	if got := summaries[targetURI].QuoteCount; got != len(want) {
		t.Fatalf("quoteCount = %d, want %d", got, len(want))
	}
}

// IT-011: FR-009; AC-016.
func TestPostStore_ResolveInteractionTargetProtectsUnavailableTargetsAndReplyKinds(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:owner")
	commentAuthor := syntax.DID("did:plc:comment-author")
	nestedAuthor := syntax.DID("did:plc:nested-author")
	seedMember(t, pool, viewer.String())
	seedMember(t, pool, owner.String())
	seedMember(t, pool, commentAuthor.String())
	seedMember(t, pool, nestedAuthor.String())
	rootURI := seedPost(t, pool, owner.String(), "root", "root", time.Now().UTC())
	commentURI := seedReplyPost(t, pool, commentAuthor.String(), "reply", "reply", rootURI, rootURI, time.Now().UTC())
	seedReplyPost(t, pool, nestedAuthor.String(), "nested", "nested", rootURI, commentURI, time.Now().UTC())
	store := api.NewPostStore(pool)
	for _, kind := range []api.PostInteractionKind{api.PostInteractionLikes, api.PostInteractionReposts, api.PostInteractionQuotes} {
		if _, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("missing"), kind); !errors.Is(err, api.ErrPostNotFound) {
			t.Fatalf("%s missing target error = %v, want post not found", kind, err)
		}
	}

	if _, err := store.ResolveInteractionTarget(ctx, viewer, commentAuthor, syntax.RecordKey("reply"), api.PostInteractionLikes); err != nil {
		t.Fatalf("likes reply target: %v", err)
	}
	for _, kind := range []api.PostInteractionKind{api.PostInteractionReposts, api.PostInteractionQuotes} {
		if _, err := store.ResolveInteractionTarget(ctx, viewer, commentAuthor, syntax.RecordKey("reply"), kind); !errors.Is(err, api.ErrPostNotFound) {
			t.Fatalf("%s reply target error = %v, want post not found", kind, err)
		}
	}
	seedBlock(t, pool, commentAuthor, owner, "comment-blocks-parent")
	if _, err := store.ResolveInteractionTarget(ctx, viewer, commentAuthor, syntax.RecordKey("reply"), api.PostInteractionLikes); !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("likes protected reply target error = %v, want post not found", err)
	}
	if _, err := store.ResolveInteractionTarget(ctx, viewer, nestedAuthor, syntax.RecordKey("nested"), api.PostInteractionLikes); !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("likes reply under protected ancestor error = %v, want post not found", err)
	}

	seedBlock(t, pool, viewer, owner, "viewer-blocks-target")
	for _, kind := range []api.PostInteractionKind{api.PostInteractionLikes, api.PostInteractionReposts, api.PostInteractionQuotes} {
		if _, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), kind); !errors.Is(err, api.ErrPostNotFound) {
			t.Fatalf("%s blocked target error = %v, want post not found", kind, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atproto_blocks WHERE rkey = 'viewer-blocks-target'`); err != nil {
		t.Fatalf("remove target block: %v", err)
	}
	seedModerationOutput(t, pool, "post", owner.String(), rootURI, "hide", time.Now().UTC())
	for _, kind := range []api.PostInteractionKind{api.PostInteractionLikes, api.PostInteractionReposts, api.PostInteractionQuotes} {
		if _, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), kind); !errors.Is(err, api.ErrPostNotFound) {
			t.Fatalf("%s hidden target error = %v, want post not found", kind, err)
		}
	}
}

// IT-013: NFR-001; AC-019.
func TestPostStore_PostInteractionHydrationIsBounded(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	owner := syntax.DID("did:plc:owner")
	authors := []syntax.DID{"did:plc:author-a", "did:plc:author-b"}
	actors := []syntax.DID{
		"did:plc:actor-a", "did:plc:actor-b", "did:plc:actor-c",
		"did:plc:actor-d", "did:plc:actor-e", "did:plc:actor-f",
	}
	for _, did := range append(append([]syntax.DID{viewer, owner}, authors...), actors...) {
		seedMember(t, pool, did.String())
		seedBskyProfile(t, pool, did.String(), did.String(), "")
		if did != viewer {
			handle := did.String()[len("did:plc:"):] + ".test"
			if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, did, handle); err != nil {
				t.Fatalf("seed identity %s: %v", did, err)
			}
		}
	}

	base := time.Date(2026, 9, 6, 15, 0, 0, 0, time.UTC)
	targetURI := seedPost(t, pool, owner.String(), "root", "root", base)
	for i, actor := range actors {
		seedInteraction(t, pool, "like", actor.String(), "bounded", targetURI, false)
		seedQuotePost(t, pool, authors[i%len(authors)].String(), fmt.Sprintf("quote-%d", i), fmt.Sprintf("quote %d", i), targetURI, "bafyroot", base.Add(time.Duration(i+1)*time.Minute))
	}

	queries := &countingQueryTracer{}
	countedPool := newCountedPool(t, pool, queries)
	store := api.NewPostStore(countedPool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("root"), api.PostInteractionLikes)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}

	queries.reset()
	if _, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, 1, ""); err != nil {
		t.Fatalf("one account: %v", err)
	}
	oneAccountQueries := queries.count()
	queries.reset()
	accounts, err := store.ListPostInteractionAccounts(ctx, viewer, target, api.PostInteractionLikes, len(actors), "")
	if err != nil {
		t.Fatalf("full account page: %v", err)
	}
	if len(accounts.Items) != len(actors) || queries.count() != oneAccountQueries {
		t.Fatalf("full account hydration = %d items/%d queries, want %d items/%d queries", len(accounts.Items), queries.count(), len(actors), oneAccountQueries)
	}

	quoteTarget := *target
	resolver := &countingPostInteractionResolver{}
	quoteCtx := middleware.WithDID(ctx, viewer)
	queries.reset()
	if _, _, err := store.ListQuotePostResponses(quoteCtx, viewer, &quoteTarget, []string{}, 1, "", resolver); err != nil {
		t.Fatalf("one quote: %v", err)
	}
	oneQuoteQueries := queries.count()
	queries.reset()
	resolver.reset()
	quotes, cursor, err := store.ListQuotePostResponses(quoteCtx, viewer, &quoteTarget, []string{}, len(actors), "", resolver)
	if err != nil {
		t.Fatalf("full quote page: %v", err)
	}
	if cursor != "" || len(quotes) != len(actors) || queries.count() != oneQuoteQueries {
		t.Fatalf("full quote hydration = %d items/%d queries/cursor %q, want %d items/%d queries/exhausted", len(quotes), queries.count(), cursor, len(actors), oneQuoteQueries)
	}
	for _, quote := range quotes {
		if quote.Author.Handle == "" || quote.QuoteView == nil || quote.QuoteView.Post == nil || quote.QuoteView.Post.Author.Handle == "" {
			t.Fatalf("quote was not fully hydrated: %+v", quote)
		}
	}
	wantResolved := map[syntax.DID]int{}
	if !mapsEqual(resolver.calls, wantResolved) {
		t.Fatalf("resolver calls = %v, want bounded cache hydration without external calls", resolver.calls)
	}
}

// REG-005: quote-list hydration retains revealable moderation placeholders and
// stops after the quoted post's compact preview, even when that post itself
// quotes an unavailable record.
func TestPostStore_ListQuotePostResponses_UsesOnePreviewLevel(t *testing.T) {
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:viewer")
	targetAuthor := syntax.DID("did:plc:target")
	visibleAuthor := syntax.DID("did:plc:visible")
	mutedAuthor := syntax.DID("did:plc:muted")
	for _, did := range []syntax.DID{viewer, targetAuthor, visibleAuthor, mutedAuthor} {
		seedMember(t, pool, did.String())
		seedBskyProfile(t, pool, did.String(), did.String(), "")
		if did != viewer {
			handle := did.String()[len("did:plc:"):] + ".test"
			if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, did, handle); err != nil {
				t.Fatalf("seed identity %s: %v", did, err)
			}
		}
	}

	base := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	missingNestedURI := "at://did:plc:missing/social.craftsky.feed.post/unavailable"
	targetURI := seedQuotePost(t, pool, targetAuthor.String(), "target-quote", "target quotes unavailable", missingNestedURI, "bafymissing", base)
	visibleURI := seedQuotePost(t, pool, visibleAuthor.String(), "visible-outer", "visible outer", targetURI, "bafycid-target-quote", base.Add(time.Minute))
	mutedURI := seedQuotePost(t, pool, mutedAuthor.String(), "muted-outer", "muted outer sentinel", targetURI, "bafycid-target-quote", base.Add(2*time.Minute))
	seedMute(t, pool, viewer, mutedAuthor)

	store := api.NewPostStore(pool)
	target, err := store.ResolveInteractionTarget(ctx, viewer, targetAuthor, syntax.RecordKey("target-quote"), api.PostInteractionQuotes)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	items, cursor, err := store.ListQuotePostResponses(ctx, viewer, target, []string{}, 10, "", &countingPostInteractionResolver{})
	if err != nil {
		t.Fatalf("list hydrated quotes: %v", err)
	}
	if cursor != "" || len(items) != 2 {
		t.Fatalf("items/cursor = %d/%q, want 2/exhausted", len(items), cursor)
	}

	byURI := make(map[string]*api.PostResponse, len(items))
	for _, item := range items {
		byURI[item.URI] = item
	}
	muted := byURI[mutedURI]
	if muted == nil || muted.Availability != "muted" || muted.Relationship == nil || !muted.Relationship.Revealable {
		t.Fatalf("muted quote response = %+v, want revealable muted placeholder", muted)
	}
	mutedJSON, err := json.Marshal(muted)
	if err != nil {
		t.Fatalf("marshal muted quote: %v", err)
	}
	if string(mutedJSON) != `{"uri":"`+mutedURI+`","availability":"muted","relationship":{"state":"muted","revealable":true}}` {
		t.Fatalf("muted quote wire = %s", mutedJSON)
	}

	visible := byURI[visibleURI]
	if visible == nil || visible.QuoteView == nil || visible.QuoteView.State != "visible" || visible.QuoteView.Post == nil || visible.QuoteView.Post.URI != targetURI {
		t.Fatalf("visible quote response = %+v, want one target preview", visible)
	}
	visibleJSON, err := json.Marshal(visible)
	if err != nil {
		t.Fatalf("marshal visible quote: %v", err)
	}
	var wire struct {
		QuoteView struct {
			Post map[string]any `json:"post"`
		} `json:"quoteView"`
	}
	if err := json.Unmarshal(visibleJSON, &wire); err != nil {
		t.Fatalf("decode visible quote: %v", err)
	}
	if _, ok := wire.QuoteView.Post["quote"]; ok {
		t.Fatalf("nested quote ref leaked into compact preview: %s", visibleJSON)
	}
	if _, ok := wire.QuoteView.Post["quoteView"]; ok {
		t.Fatalf("recursive quote view leaked into compact preview: %s", visibleJSON)
	}
	if bytes.Contains(visibleJSON, []byte(missingNestedURI)) {
		t.Fatalf("unavailable nested target leaked into one-level response: %s", visibleJSON)
	}
}

type countingQueryTracer struct {
	queries int
}

func (tracer *countingQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	tracer.queries++
	return ctx
}

func (*countingQueryTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (tracer *countingQueryTracer) reset() { tracer.queries = 0 }

func (tracer *countingQueryTracer) count() int { return tracer.queries }

func newCountedPool(t *testing.T, source *pgxpool.Pool, tracer pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	cfg := source.Config()
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create counted pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type countingPostInteractionResolver struct {
	calls map[syntax.DID]int
}

func (resolver *countingPostInteractionResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	if resolver.calls == nil {
		resolver.calls = make(map[syntax.DID]int)
	}
	resolver.calls[did]++
	return syntax.Handle(did.String()[len("did:plc:"):] + ".test"), nil
}

func (*countingPostInteractionResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "", errors.New("unexpected ResolveDID call")
}

func (resolver *countingPostInteractionResolver) reset() {
	resolver.calls = make(map[syntax.DID]int)
}

func mapsEqual[K comparable, V comparable](left, right map[K]V) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func seedMute(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, viewer, actor syntax.DID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO actor_mutes (owner_did, subject_did) VALUES ($1, $2)`, viewer, actor); err != nil {
		t.Fatalf("seed mute: %v", err)
	}
}

func seedBlock(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, blocker, subject syntax.DID, rkey string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://' || $1 || '/app.bsky.graph.block/' || $3, $1, $3, 'cid', $2, '{}', now())
	`, blocker, subject, rkey); err != nil {
		t.Fatalf("seed block: %v", err)
	}
}

func sameDIDSet(got, want []syntax.DID) bool {
	if len(got) != len(want) {
		return false
	}
	gotSet := make(map[syntax.DID]struct{}, len(got))
	for _, did := range got {
		gotSet[did] = struct{}{}
	}
	for _, did := range want {
		if _, ok := gotSet[did]; !ok {
			return false
		}
	}
	return true
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	gotSet := make(map[string]struct{}, len(got))
	for _, value := range got {
		gotSet[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := gotSet[value]; !ok {
			return false
		}
	}
	return true
}

func assertAccountPagination(t *testing.T, ctx context.Context, store *api.PostStore, target *api.PostInteractionTarget, kind api.PostInteractionKind, want []syntax.DID) {
	t.Helper()
	exact, err := store.ListPostInteractionAccounts(ctx, syntax.DID("did:plc:viewer"), target, kind, len(want), "")
	if err != nil {
		t.Fatalf("exact-size page: %v", err)
	}
	assertAccountDIDs(t, exact.Items, want)
	if exact.Cursor != nil {
		t.Fatalf("exact-size cursor = %q, want omitted", *exact.Cursor)
	}

	var got []syntax.DID
	cursor := ""
	for {
		page, err := store.ListPostInteractionAccounts(ctx, syntax.DID("did:plc:viewer"), target, kind, 2, cursor)
		if err != nil {
			t.Fatalf("page %d: %v", len(got)/2+1, err)
		}
		if len(page.Items) > 2 {
			t.Fatalf("page has %d items, want at most 2", len(page.Items))
		}
		for _, item := range page.Items {
			got = append(got, item.DID)
		}
		if page.Cursor == nil {
			break
		}
		cursor = *page.Cursor
	}
	if !slices.Equal(got, want) {
		t.Fatalf("traversal = %v, want %v", got, want)
	}
}

func assertQuotePagination(t *testing.T, ctx context.Context, store *api.PostStore, target *api.PostInteractionTarget, want []string) {
	t.Helper()
	exact, cursor, err := store.ListQuotePosts(ctx, syntax.DID("did:plc:viewer"), target, []string{}, len(want), "")
	if err != nil {
		t.Fatalf("exact-size page: %v", err)
	}
	assertPostRowURIs(t, exact, want)
	if cursor != "" {
		t.Fatalf("exact-size cursor = %q, want omitted", cursor)
	}

	var got []string
	cursor = ""
	for {
		page, next, err := store.ListQuotePosts(ctx, syntax.DID("did:plc:viewer"), target, []string{}, 2, cursor)
		if err != nil {
			t.Fatalf("page %d: %v", len(got)/2+1, err)
		}
		if len(page) > 2 {
			t.Fatalf("page has %d items, want at most 2", len(page))
		}
		for _, item := range page {
			got = append(got, item.URI)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if !slices.Equal(got, want) {
		t.Fatalf("traversal = %v, want %v", got, want)
	}
}

func assertAccountDIDs(t *testing.T, items []api.ProfileAccountSummary, want []syntax.DID) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("item count = %d, want %d", len(items), len(want))
	}
	for i := range want {
		if items[i].DID != want[i] {
			t.Fatalf("item %d DID = %s, want %s", i, items[i].DID, want[i])
		}
	}
}
