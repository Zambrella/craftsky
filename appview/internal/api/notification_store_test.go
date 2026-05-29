// appview/internal/api/notification_store_test.go
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

const notificationStoreDDL = timelineStoreDDL

func notificationURIs(rows []*api.NotificationRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.URI)
	}
	return out
}

func notificationRowByURI(rows []*api.NotificationRow, uri string) *api.NotificationRow {
	for _, row := range rows {
		if row.URI == uri {
			return row
		}
	}
	return nil
}

func TestNotificationStore_ListNotifications_DerivesFollowNotificationsScopedToViewer(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyalice")
	seedBskyProfile(t, pool, "did:plc:bob", "Bob", "bafybob")

	aliceFollow := seedFollow(t, pool, "did:plc:alice", "did:plc:viewer", "follow-viewer")
	bobFollow := seedFollow(t, pool, "did:plc:bob", "did:plc:carol", "follow-carol")

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}

	got := notificationURIs(rows)
	if !slices.Contains(got, aliceFollow) {
		t.Fatalf("notification URIs = %v, want containing Alice follow %s", got, aliceFollow)
	}
	if slices.Contains(got, bobFollow) {
		t.Fatalf("notification URIs = %v, must not contain Bob follow directed elsewhere %s", got, bobFollow)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1 scoped follow notification; rows=%+v", len(rows), rows)
	}
	row := rows[0]
	if row.Type != api.NotificationTypeFollow {
		t.Fatalf("row.Type = %q, want %q", row.Type, api.NotificationTypeFollow)
	}
	if row.ActorDID != "did:plc:alice" {
		t.Fatalf("row.ActorDID = %q, want did:plc:alice", row.ActorDID)
	}
	if row.ActorDisplayName == nil || *row.ActorDisplayName != "Alice" {
		t.Fatalf("row.ActorDisplayName = %v, want Alice", row.ActorDisplayName)
	}
	if row.SubjectPost != nil {
		t.Fatalf("row.SubjectPost = %+v, want nil for follow notification", row.SubjectPost)
	}
}

func TestNotificationStore_ListNotifications_DerivesActiveLikeNotificationsForViewerPosts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyalice")

	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)
	otherPost := seedPost(t, pool, "did:plc:carol", "carol-root", "carol post", base)
	aliceLike := seedInteraction(t, pool, "like", "did:plc:alice", "like-viewer", viewerPost, false)
	bobLike := seedInteraction(t, pool, "like", "did:plc:bob", "like-carol", otherPost, false)

	store := api.NewPostStore(pool)
	rows, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	got := notificationURIs(rows)
	if slices.Contains(got, bobLike) {
		t.Fatalf("notification URIs = %v, must not contain like for another author's post %s", got, bobLike)
	}
	row := notificationRowByURI(rows, aliceLike)
	if row == nil {
		t.Fatalf("notification URIs = %v, want Alice like %s", got, aliceLike)
	}
	if row.Type != api.NotificationTypeLike {
		t.Fatalf("row.Type = %q, want %q", row.Type, api.NotificationTypeLike)
	}
	if row.ActorDID != "did:plc:alice" {
		t.Fatalf("row.ActorDID = %q, want did:plc:alice", row.ActorDID)
	}
	if row.SubjectPost == nil {
		t.Fatal("row.SubjectPost = nil, want liked viewer post")
	}
	if row.SubjectPost.URI != viewerPost || row.SubjectPost.DID != "did:plc:viewer" {
		t.Fatalf("row.SubjectPost = %+v, want viewer post %s", row.SubjectPost, viewerPost)
	}
}

func TestNotificationStore_ListNotifications_DerivesActiveRepostNotificationsForViewerPosts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 12, 30, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyalice")

	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)
	otherPost := seedPost(t, pool, "did:plc:carol", "carol-root", "carol post", base)
	aliceRepost := seedInteraction(t, pool, "repost", "did:plc:alice", "repost-viewer", viewerPost, false)
	bobRepost := seedInteraction(t, pool, "repost", "did:plc:bob", "repost-carol", otherPost, false)

	store := api.NewPostStore(pool)
	rows, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	got := notificationURIs(rows)
	if slices.Contains(got, bobRepost) {
		t.Fatalf("notification URIs = %v, must not contain repost for another author's post %s", got, bobRepost)
	}
	row := notificationRowByURI(rows, aliceRepost)
	if row == nil {
		t.Fatalf("notification URIs = %v, want Alice repost %s", got, aliceRepost)
	}
	if row.Type != api.NotificationTypeRepost {
		t.Fatalf("row.Type = %q, want %q", row.Type, api.NotificationTypeRepost)
	}
	if row.ActorDID != "did:plc:alice" {
		t.Fatalf("row.ActorDID = %q, want did:plc:alice", row.ActorDID)
	}
	if row.SubjectPost == nil || row.SubjectPost.URI != viewerPost || row.SubjectPost.DID != "did:plc:viewer" {
		t.Fatalf("row.SubjectPost = %+v, want viewer post %s", row.SubjectPost, viewerPost)
	}
}

func TestNotificationStore_ListNotifications_DerivesDirectReplyNotificationsWithFocus(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 13, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyalice")

	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)
	carolPost := seedPost(t, pool, "did:plc:carol", "carol-root", "carol post", base)
	aliceReply := seedReplyPost(t, pool, "did:plc:alice", "reply-viewer", "alice reply", viewerPost, viewerPost, base.Add(time.Minute))
	bobReply := seedReplyPost(t, pool, "did:plc:bob", "reply-carol", "bob reply", carolPost, carolPost, base.Add(2*time.Minute))
	deeperReply := seedReplyPost(t, pool, "did:plc:alice", "reply-deeper", "deeper reply", viewerPost, aliceReply, base.Add(3*time.Minute))

	store := api.NewPostStore(pool)
	rows, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	got := notificationURIs(rows)
	for _, excluded := range []string{bobReply, deeperReply} {
		if slices.Contains(got, excluded) {
			t.Fatalf("notification URIs = %v, must not contain out-of-scope reply %s", got, excluded)
		}
	}
	row := notificationRowByURI(rows, aliceReply)
	if row == nil {
		t.Fatalf("notification URIs = %v, want Alice direct reply %s", got, aliceReply)
	}
	if row.Type != api.NotificationTypeReply {
		t.Fatalf("row.Type = %q, want %q", row.Type, api.NotificationTypeReply)
	}
	if row.ActorDID != "did:plc:alice" {
		t.Fatalf("row.ActorDID = %q, want did:plc:alice", row.ActorDID)
	}
	if row.SubjectPost == nil || row.SubjectPost.URI != viewerPost {
		t.Fatalf("row.SubjectPost = %+v, want parent post %s", row.SubjectPost, viewerPost)
	}
	if row.Reply == nil {
		t.Fatal("row.Reply = nil, want reply focus identity")
	}
	if row.Reply.URI != aliceReply || row.Reply.Rkey != "reply-viewer" || row.Reply.CID == "" {
		t.Fatalf("row.Reply = %+v, want identity for %s", row.Reply, aliceReply)
	}
}

func TestNotificationStore_ListNotifications_ExcludesSelfGeneratedNotifications(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 14, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)
	seedFollow(t, pool, "did:plc:viewer", "did:plc:viewer", "follow-self")
	selfLike := seedInteraction(t, pool, "like", "did:plc:viewer", "like-self", viewerPost, false)
	selfRepost := seedInteraction(t, pool, "repost", "did:plc:viewer", "repost-self", viewerPost, false)
	selfReply := seedReplyPost(t, pool, "did:plc:viewer", "reply-self", "self reply", viewerPost, viewerPost, base.Add(time.Minute))

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
	got := notificationURIs(rows)
	for _, excluded := range []string{selfLike, selfRepost, selfReply} {
		if slices.Contains(got, excluded) {
			t.Fatalf("notification URIs = %v, must not contain self-generated row %s", got, excluded)
		}
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %+v, want no self-generated notifications", rows)
	}
}

func TestNotificationStore_ListNotifications_ExcludesDeletedLikesAndReposts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 14, 30, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)
	activeLike := seedInteraction(t, pool, "like", "did:plc:alice", "like-active", viewerPost, false)
	deletedLike := seedInteraction(t, pool, "like", "did:plc:bob", "like-deleted", viewerPost, true)
	activeRepost := seedInteraction(t, pool, "repost", "did:plc:alice", "repost-active", viewerPost, false)
	deletedRepost := seedInteraction(t, pool, "repost", "did:plc:bob", "repost-deleted", viewerPost, true)

	store := api.NewPostStore(pool)
	rows, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	got := notificationURIs(rows)
	for _, want := range []string{activeLike, activeRepost} {
		if !slices.Contains(got, want) {
			t.Fatalf("notification URIs = %v, want active interaction %s", got, want)
		}
	}
	for _, excluded := range []string{deletedLike, deletedRepost} {
		if slices.Contains(got, excluded) {
			t.Fatalf("notification URIs = %v, must not contain deleted interaction %s", got, excluded)
		}
	}
}

func TestNotificationStore_ListNotifications_OrdersMixedTypesByIndexedAtThenURIDesc(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 15, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)

	oldestFollow := seedFollow(t, pool, "did:plc:bob", "did:plc:viewer", "a-oldest")
	tiedLowLike := seedInteraction(t, pool, "like", "did:plc:alice", "a-tied", viewerPost, false)
	tiedHighRepost := seedInteraction(t, pool, "repost", "did:plc:carol", "z-tied", viewerPost, false)
	newestReply := seedReplyPost(t, pool, "did:plc:alice", "z-newest", "newest reply", viewerPost, viewerPost, base.Add(10*time.Minute))
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_likes SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(5*time.Minute), tiedLowLike); err != nil {
		t.Fatalf("move like timestamp: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_reposts SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(5*time.Minute), tiedHighRepost); err != nil {
		t.Fatalf("move repost timestamp: %v", err)
	}

	// seedFollow uses a fixed timestamp newer than base but older than newestReply;
	// make it the oldest event for this test to prove indexed_at dominates type.
	if _, err := pool.Exec(context.Background(), `UPDATE atproto_follows SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(-time.Minute), oldestFollow); err != nil {
		t.Fatalf("move follow timestamp: %v", err)
	}

	store := api.NewPostStore(pool)
	rows, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}

	got := notificationURIs(rows)
	want := []string{newestReply, tiedHighRepost, tiedLowLike, oldestFollow}
	if !slices.Equal(got, want) {
		t.Fatalf("notification URIs = %v, want %v", got, want)
	}
}

func TestNotificationStore_ListNotifications_PaginatesMixedTypesWithoutDuplicatesOrSkips(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	base := time.Date(2026, 5, 28, 16, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	viewerPost := seedPost(t, pool, "did:plc:viewer", "viewer-root", "viewer post", base)

	row1 := seedReplyPost(t, pool, "did:plc:alice", "row-1", "reply", viewerPost, viewerPost, base.Add(5*time.Minute))
	row2 := seedFollow(t, pool, "did:plc:bob", "did:plc:viewer", "row-2")
	row3 := seedInteraction(t, pool, "repost", "did:plc:carol", "row-3", viewerPost, false)
	row4 := seedInteraction(t, pool, "like", "did:plc:alice", "row-4", viewerPost, false)
	row5 := seedFollow(t, pool, "did:plc:carol", "did:plc:viewer", "row-5")
	if _, err := pool.Exec(context.Background(), `UPDATE atproto_follows SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(4*time.Minute), row2); err != nil {
		t.Fatalf("move row2: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_reposts SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(3*time.Minute), row3); err != nil {
		t.Fatalf("move row3: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_likes SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(2*time.Minute), row4); err != nil {
		t.Fatalf("move row4: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE atproto_follows SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(time.Minute), row5); err != nil {
		t.Fatalf("move row5: %v", err)
	}

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListNotifications first: %v", err)
	}
	if cursor == "" {
		t.Fatal("first cursor = empty, want next page cursor")
	}
	second, nextCursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 2, cursor)
	if err != nil {
		t.Fatalf("ListNotifications second: %v", err)
	}
	if nextCursor == "" {
		t.Fatal("second cursor = empty, want final page cursor")
	}
	third, finalCursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 2, nextCursor)
	if err != nil {
		t.Fatalf("ListNotifications third: %v", err)
	}
	if finalCursor != "" {
		t.Fatalf("final cursor = %q, want empty", finalCursor)
	}

	combined := append(notificationURIs(first), notificationURIs(second)...)
	combined = append(combined, notificationURIs(third)...)
	want := []string{row1, row2, row3, row4, row5}
	if !slices.Equal(combined, want) {
		t.Fatalf("combined pages = %v, want %v", combined, want)
	}
}

func TestNotificationStore_ListNotifications_OmitsCursorWhenExactFullFinalPage(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	row1 := seedFollow(t, pool, "did:plc:alice", "did:plc:viewer", "row-1")
	row2 := seedFollow(t, pool, "did:plc:bob", "did:plc:viewer", "row-2")

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListNotifications(context.Background(), "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if got, want := notificationURIs(rows), []string{row2, row1}; !slices.Equal(got, want) {
		t.Fatalf("notification URIs = %v, want %v", got, want)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty when final page exactly fills requested limit", cursor)
	}
}

func TestNotificationStore_ListNotifications_InvalidCursorReturnsInvalidCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, notificationStoreDDL)
	store := api.NewPostStore(pool)

	_, _, err := store.ListNotifications(context.Background(), "did:plc:viewer", 20, "not-a-cursor")
	if !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("err = %v, want ErrInvalidCursor", err)
	}
}
