package api_test

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/relationships"
)

// TestPostHandlersAcceptNarrowCapabilities is primarily a compile-time
// characterization: each fake intentionally implements only the operations
// consumed by the handler it is passed to.
func TestPostHandlersAcceptNarrowCapabilities(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		_ = api.CreatePostHandler(createPostCapability{}, nil, nil, api.MediaLimits{}, nil)
	})
	t.Run("read", func(t *testing.T) {
		_ = api.GetPostHandler(readPostCapability{}, nil, nil)
	})
	t.Run("conversation", func(t *testing.T) {
		_ = api.ListCommentRepliesHandler(conversationCapability{}, nil, nil)
		_ = api.GetPostCommentsHandler(conversationCapability{}, nil, nil)
	})
	t.Run("interactions", func(t *testing.T) {
		_ = api.LikePostHandler(likePostCapability{}, nil, nil)
		_ = api.UnlikePostHandler(unlikePostCapability{}, nil, nil)
		_ = api.RepostPostHandler(repostPostCapability{}, nil, nil)
		_ = api.UnrepostPostHandler(unrepostPostCapability{}, nil, nil)
	})
	t.Run("author feeds", func(t *testing.T) {
		_ = api.ListPostsByAuthorHandler(authorPostsCapability{}, nil, nil, nil)
		_ = api.ListProjectsByAuthorHandler(authorProjectsCapability{}, nil, nil, nil)
		_ = api.ListCommentsByAuthorHandler(authorCommentsCapability{}, nil, nil)
	})
}

type quoteHydrationCapability struct{}

func (quoteHydrationCapability) QuoteViewRows(context.Context, []api.ResponseStrongRef) (map[string]*api.QuoteViewRow, error) {
	return nil, nil
}

func (quoteHydrationCapability) RelationshipStates(context.Context, syntax.DID, []syntax.DID) (map[syntax.DID]relationships.State, error) {
	return nil, nil
}

func (quoteHydrationCapability) BlockedPairs(context.Context, []api.RelationshipPair) (map[api.RelationshipPair]bool, error) {
	return nil, nil
}

type createPostCapability struct {
	quoteHydrationCapability
}

func (createPostCapability) AuthorizeDirectedInteraction(context.Context, syntax.DID, syntax.DID, relationships.Operation) error {
	return nil
}

func (createPostCapability) ResolveShareTarget(context.Context, string, string) (*api.ShareTargetRef, error) {
	return nil, nil
}

func (createPostCapability) ReadAuthor(context.Context, string) (*api.PostAuthorRow, error) {
	return nil, nil
}

type readPostCapability struct {
	quoteHydrationCapability
}

func (readPostCapability) ReadOne(context.Context, string, string) (*api.PostRow, error) {
	return nil, nil
}

func (readPostCapability) RelationshipState(context.Context, syntax.DID, syntax.DID) (relationships.State, error) {
	return relationships.State{}, nil
}

func (readPostCapability) EngagementSummaries(context.Context, string, []string) (map[string]api.EngagementSummary, error) {
	return nil, nil
}

type conversationCapability struct {
	quoteHydrationCapability
}

func (conversationCapability) ReadOne(context.Context, string, string) (*api.PostRow, error) {
	return nil, nil
}

func (conversationCapability) ReadPostByURI(context.Context, string) (*api.PostRow, error) {
	return nil, nil
}

func (conversationCapability) ListRootComments(context.Context, string, string, string, int, string) ([]*api.PostRow, string, error) {
	return nil, "", nil
}

func (conversationCapability) ListCommentBranchReplies(context.Context, string, string, int, string) ([]*api.PostRow, string, error) {
	return nil, "", nil
}

func (conversationCapability) ListCommentBranchRepliesAround(context.Context, string, string, string, int) ([]*api.PostRow, string, error) {
	return nil, "", nil
}

func (conversationCapability) RelationshipState(context.Context, syntax.DID, syntax.DID) (relationships.State, error) {
	return relationships.State{}, nil
}

func (conversationCapability) EngagementSummaries(context.Context, string, []string) (map[string]api.EngagementSummary, error) {
	return nil, nil
}

type likePostCapability struct{}

func (likePostCapability) AuthorizeDirectedInteraction(context.Context, syntax.DID, syntax.DID, relationships.Operation) error {
	return nil
}

func (likePostCapability) ResolvePostTarget(context.Context, string, string) (*api.PostTargetRef, error) {
	return nil, nil
}

func (likePostCapability) FindActiveLike(context.Context, string, string) (*api.InteractionRow, error) {
	return nil, nil
}

type unlikePostCapability struct{}

func (unlikePostCapability) ResolvePostTarget(context.Context, string, string) (*api.PostTargetRef, error) {
	return nil, nil
}

func (unlikePostCapability) FindActiveLike(context.Context, string, string) (*api.InteractionRow, error) {
	return nil, nil
}

type repostPostCapability struct{}

func (repostPostCapability) AuthorizeDirectedInteraction(context.Context, syntax.DID, syntax.DID, relationships.Operation) error {
	return nil
}

func (repostPostCapability) ResolveShareTarget(context.Context, string, string) (*api.ShareTargetRef, error) {
	return nil, nil
}

func (repostPostCapability) FindActiveRepost(context.Context, string, string) (*api.InteractionRow, error) {
	return nil, nil
}

type unrepostPostCapability struct{}

func (unrepostPostCapability) ResolvePostTarget(context.Context, string, string) (*api.PostTargetRef, error) {
	return nil, nil
}

func (unrepostPostCapability) FindActiveRepost(context.Context, string, string) (*api.InteractionRow, error) {
	return nil, nil
}

type authorFeedHydrationCapability struct {
	quoteHydrationCapability
}

func (authorFeedHydrationCapability) RelationshipState(context.Context, syntax.DID, syntax.DID) (relationships.State, error) {
	return relationships.State{}, nil
}

func (authorFeedHydrationCapability) EngagementSummaries(context.Context, string, []string) (map[string]api.EngagementSummary, error) {
	return nil, nil
}

type authorPostsCapability struct {
	authorFeedHydrationCapability
}

func (authorPostsCapability) ListByAuthor(context.Context, string, int, string) ([]*api.PostRow, string, error) {
	return nil, "", nil
}

type authorProjectsCapability struct {
	authorFeedHydrationCapability
}

func (authorProjectsCapability) ListProjectsByAuthor(context.Context, string, int, string) ([]*api.PostRow, string, error) {
	return nil, "", nil
}

type authorCommentsCapability struct {
	authorFeedHydrationCapability
}

func (authorCommentsCapability) ListCommentsByAuthor(context.Context, string, int, string) ([]*api.PostRow, string, error) {
	return nil, "", nil
}
