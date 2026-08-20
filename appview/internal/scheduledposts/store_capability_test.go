package scheduledposts

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestManualPublicationServiceAcceptsNarrowStore(t *testing.T) {
	_, _ = NewManualPublicationService(manualPublicationStoreFake{}, nil)
}

type manualPublicationStoreFake struct{}

func (manualPublicationStoreFake) PrepareManualPublication(context.Context, UpdateParams) (WorkItem, error) {
	return WorkItem{}, nil
}

func (manualPublicationStoreFake) Get(context.Context, syntax.DID, uuid.UUID) (Resource, error) {
	return Resource{}, nil
}

func TestPublicationProcessorAcceptsNarrowStore(t *testing.T) {
	_ = PublicationProcessorOptions{Store: publicationProcessorStoreFake{}}
}

type publicationProcessorStoreFake struct{}

func (publicationProcessorStoreFake) publicationSnapshot(context.Context, PublishingClaim) (publicationSnapshot, error) {
	return publicationSnapshot{}, nil
}

func (publicationProcessorStoreFake) SaveFrozenRecord(context.Context, FrozenRecordParams) error {
	return nil
}

func (publicationProcessorStoreFake) AcquirePublishingEffect(context.Context, PublishingClaim) (*PublishingEffectGuard, error) {
	return nil, nil
}

func (publicationProcessorStoreFake) FinalizePublication(context.Context, FinalizePublicationParams) (FinalizePublicationResult, error) {
	return FinalizePublicationResult{}, nil
}

func (publicationProcessorStoreFake) failPublication(context.Context, PublishingClaim, FailureDecision, time.Time) (Status, error) {
	return StatusScheduled, nil
}
