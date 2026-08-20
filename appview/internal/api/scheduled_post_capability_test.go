package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/scheduledposts"
)

// TestScheduledPostHandlersAcceptNarrowCapabilities is a compile-time
// characterization. Each fake deliberately implements only the repository
// operations consumed by the handler it is passed to.
func TestScheduledPostHandlersAcceptNarrowCapabilities(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		_ = api.CreateScheduledPostHandler(scheduledPostCreator{}, api.MediaLimits{}, nil, nil)
	})
	t.Run("list", func(t *testing.T) {
		_ = api.ListScheduledPostsHandler(scheduledPostLister{}, nil)
	})
	t.Run("get", func(t *testing.T) {
		_ = api.GetScheduledPostHandler(scheduledPostReader{}, nil)
	})
	t.Run("update", func(t *testing.T) {
		_ = api.UpdateScheduledPostHandler(scheduledPostUpdater{}, api.MediaLimits{}, nil, nil)
	})
	t.Run("delete", func(t *testing.T) {
		_ = api.DeleteScheduledPostHandler(scheduledPostDeleter{}, nil, nil)
	})
}

type scheduledPostCreator struct{}

func (scheduledPostCreator) Create(context.Context, scheduledposts.CreateParams) (scheduledposts.ScheduledPost, error) {
	return scheduledposts.ScheduledPost{}, nil
}

type scheduledPostLister struct{}

func (scheduledPostLister) List(context.Context, syntax.DID) ([]scheduledposts.Resource, error) {
	return nil, nil
}

type scheduledPostReader struct{}

func (scheduledPostReader) Get(context.Context, syntax.DID, uuid.UUID) (scheduledposts.Resource, error) {
	return scheduledposts.Resource{}, nil
}

type scheduledPostUpdater struct {
	scheduledPostReader
}

func (scheduledPostUpdater) Update(context.Context, scheduledposts.UpdateParams) (scheduledposts.UpdateResult, error) {
	return scheduledposts.UpdateResult{}, nil
}

type scheduledPostDeleter struct{}

func (scheduledPostDeleter) Delete(context.Context, syntax.DID, uuid.UUID, time.Time) error {
	return nil
}
