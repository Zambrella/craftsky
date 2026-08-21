package scheduledposts

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var ErrInvalidObjectKey = errors.New("invalid scheduled media object key")

// NewGenerationObjectKey derives a stable, opaque key for one owner lifecycle
// generation and client media ID. Retries of the same request use the same
// key, while a later lifecycle generation can never collide with cleanup from
// an older generation.
func NewGenerationObjectKey(
	ownerDID syntax.DID,
	ownerGeneration int64,
	mediaID uuid.UUID,
) (string, uuid.UUID, error) {
	if ownerDID == "" || ownerGeneration <= 0 || mediaID == uuid.Nil {
		return "", uuid.Nil, ErrInvalidObjectKey
	}
	generationBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(generationBytes, uint64(ownerGeneration))
	identity := append([]byte(ownerDID.String()+"\x00"), generationBytes...)
	identity = append(identity, mediaID[:]...)
	attemptID := uuid.NewSHA1(uuid.NameSpaceOID, identity)
	return fmt.Sprintf("scheduled-media/v2/%d/%s", ownerGeneration, attemptID), attemptID, nil
}
