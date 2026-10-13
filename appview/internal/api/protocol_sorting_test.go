package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestProtocolSearchClampsFutureDatesAcrossPages(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:alice")
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	newPost := seedPost(t, pool, "did:plc:alice", "new", "New", now)
	futurePost := seedPost(t, pool, "did:plc:alice", "future", "Future", now.Add(-time.Hour))
	newProject := seedSearchProject(t, pool, "did:plc:alice", "new-project", "New project", "social.craftsky.feed.defs#knitting", "New", now)
	futureProject := seedSearchProject(t, pool, "did:plc:alice", "future-project", "Future project", "social.craftsky.feed.defs#knitting", "Future", now.Add(-time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at=$1 WHERE uri IN ($2,$3)`, now.AddDate(10, 0, 0), futurePost, futureProject); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)

	for _, uri := range []string{newPost, futurePost, newProject, futureProject} {
		seedPostTags(t, pool, uri, []string{"protocol"})
	}
	first, cursor, err := store.SearchHashtagPosts(ctx, "protocol", api.SearchSortChronological, 2, "", now)
	if err != nil || !slices.Equal(searchURIs(first), []string{newProject, newPost}) || cursor == "" {
		t.Fatalf("first chronological page=%v cursor=%q error=%v", searchURIs(first), cursor, err)
	}
	second, next, err := store.SearchHashtagPosts(ctx, "protocol", api.SearchSortChronological, 2, cursor, now)
	if err != nil || !slices.Equal(searchURIs(second), []string{futureProject, futurePost}) || next != "" {
		t.Fatalf("second chronological page=%v cursor=%q error=%v", searchURIs(second), next, err)
	}
	projectRequest := api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 1, Filters: map[string][]string{}}
	first, cursor, err = store.SearchProjectsWithLanguages(ctx, "", nil, projectRequest, now)
	if err != nil || !slices.Equal(searchURIs(first), []string{newProject}) || cursor == "" {
		t.Fatalf("first project page=%v error=%v", searchURIs(first), err)
	}
	projectRequest.Cursor = cursor
	second, next, err = store.SearchProjectsWithLanguages(ctx, "", nil, projectRequest, now)
	if err != nil || !slices.Equal(searchURIs(second), []string{futureProject}) || next != "" {
		t.Fatalf("second project page=%v error=%v", searchURIs(second), err)
	}
}

func TestProtocolConversationClampsFutureDatesAcrossPages(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:alice")
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "Root", now.Add(-3*time.Hour))
	oldComment := seedReplyPost(t, pool, "did:plc:alice", "old-comment", "Old", root, root, now.Add(-2*time.Hour))
	newComment := seedReplyPost(t, pool, "did:plc:alice", "new-comment", "New", root, root, now)
	oldReply := seedReplyPost(t, pool, "did:plc:alice", "old-reply", "Old reply", root, oldComment, now.Add(-time.Hour))
	newReply := seedReplyPost(t, pool, "did:plc:alice", "new-reply", "New reply", root, oldComment, now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at=$1 WHERE uri IN ($2,$3)`, now.AddDate(10, 0, 0), oldComment, oldReply); err != nil {
		t.Fatal(err)
	}
	store := api.NewPostStore(pool, nil)
	comments, cursor, err := store.ListRootComments(ctx, root, "", "oldest", 1, "")
	if err != nil || len(comments) != 1 || comments[0].URI != oldComment || cursor == "" {
		t.Fatalf("first comments=%+v error=%v", comments, err)
	}
	comments, _, err = store.ListRootComments(ctx, root, "", "oldest", 1, cursor)
	if err != nil || len(comments) != 1 || comments[0].URI != newComment {
		t.Fatalf("second comments=%+v error=%v", comments, err)
	}
	replies, cursor, err := store.ListCommentBranchReplies(ctx, oldComment, root, 1, "")
	if err != nil || len(replies) != 1 || replies[0].URI != oldReply || cursor == "" {
		t.Fatalf("first replies=%+v error=%v", replies, err)
	}
	replies, _, err = store.ListCommentBranchReplies(ctx, oldComment, root, 1, cursor)
	if err != nil || len(replies) != 1 || replies[0].URI != newReply {
		t.Fatalf("second replies=%+v error=%v", replies, err)
	}
	focused, cursor, err := store.ListCommentBranchRepliesAround(ctx, oldComment, root, oldReply, 1)
	if err != nil || len(focused) != 1 || focused[0].URI != oldReply || cursor == "" {
		t.Fatalf("focused replies=%+v cursor=%q error=%v", focused, cursor, err)
	}
}
