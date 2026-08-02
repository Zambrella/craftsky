package scheduledposts

import (
	"context"
	"errors"
	"fmt"
)

type ManualPublicationOutcome string

const (
	ManualPublicationPublished   ManualPublicationOutcome = "published"
	ManualPublicationReconciling ManualPublicationOutcome = "reconciling"
)

type ManualPublisher interface {
	PublishManual(context.Context, UpdateParams) (ManualPublicationOutcome, error)
}

type ManualPublicationService struct {
	store     *Store
	processor *PublicationProcessor
}

func NewManualPublicationService(
	store *Store,
	processor *PublicationProcessor,
) (*ManualPublicationService, error) {
	if store == nil || processor == nil {
		return nil, errors.New("manual scheduled publication dependencies are required")
	}
	return &ManualPublicationService{store: store, processor: processor}, nil
}

func (service *ManualPublicationService) PublishManual(
	ctx context.Context,
	params UpdateParams,
) (ManualPublicationOutcome, error) {
	item, err := service.store.PrepareManualPublication(ctx, params)
	if err != nil {
		return "", err
	}
	if err := service.processor.Process(ctx, item); err != nil {
		if errors.Is(err, ErrPublicationAmbiguous) {
			return ManualPublicationReconciling, nil
		}
		return "", fmt.Errorf("manual scheduled publication: %w", err)
	}
	resource, err := service.store.Get(ctx, params.OwnerDID, params.ID)
	if errors.Is(err, ErrScheduleNotFound) {
		return ManualPublicationPublished, nil
	}
	if err != nil {
		return "", err
	}
	if resource.Status == StatusNeedsAttention {
		return "", ErrManualPublicationFailed
	}
	if resource.Status == StatusPublishing {
		return ManualPublicationReconciling, nil
	}
	return "", fmt.Errorf("unexpected manual publication status %s", resource.Status)
}
