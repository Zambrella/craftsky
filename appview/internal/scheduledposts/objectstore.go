package scheduledposts

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidObjectKey = errors.New("invalid scheduled media object key")

func NewObjectKey(mediaID uuid.UUID) (string, error) {
	if mediaID == uuid.Nil {
		return "", ErrInvalidObjectKey
	}
	return fmt.Sprintf("scheduled-media/%s", mediaID), nil
}
