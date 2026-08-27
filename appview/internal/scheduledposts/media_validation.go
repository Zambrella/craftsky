package scheduledposts

import (
	"fmt"

	"github.com/google/uuid"
)

const MaxExternalThumbnailBytes int64 = 1_000_000

type scheduledMediaRole uint8

const (
	scheduledMediaImage scheduledMediaRole = iota
	scheduledMediaExternalThumbnail
)

type scheduledMediaReference struct {
	id   uuid.UUID
	role scheduledMediaRole
}

func decodeScheduledMediaReferences(payloadBytes []byte) ([]scheduledMediaReference, error) {
	payload, err := DecodePayload(payloadBytes)
	if err != nil {
		return nil, err
	}
	references := make([]scheduledMediaReference, 0, len(payload.Media)+1)
	for _, media := range payload.Media {
		id, err := uuid.Parse(media.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled image reference: %w", err)
		}
		references = append(references, scheduledMediaReference{id: id, role: scheduledMediaImage})
	}
	if payload.External != nil && payload.External.ThumbMediaID != "" {
		id, err := uuid.Parse(payload.External.ThumbMediaID)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled external reference: %w", err)
		}
		references = append(references, scheduledMediaReference{id: id, role: scheduledMediaExternalThumbnail})
	}
	return references, nil
}

func validateScheduledMediaReference(reference scheduledMediaReference, mimeType string, sizeBytes int64) error {
	if reference.role != scheduledMediaExternalThumbnail {
		return nil
	}
	if sizeBytes < 1 || sizeBytes > MaxExternalThumbnailBytes {
		return ErrScheduledMediaInvalid
	}
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		return nil
	default:
		return ErrScheduledMediaInvalid
	}
}

func referencesMatchIDs(references []scheduledMediaReference, ids []uuid.UUID) bool {
	if len(references) != len(ids) {
		return false
	}
	for index := range references {
		if references[index].id != ids[index] {
			return false
		}
	}
	return true
}
