package scheduledposts

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const PostCollection = "social.craftsky.feed.post"

var ErrInvalidPublicationFreeze = errors.New("invalid scheduled publication freeze")

type PublicationFreezeRequest struct {
	Owner     syntax.DID
	TID       syntax.TID
	CreatedAt time.Time
	Body      []byte
}

type FrozenPublication struct {
	Rkey        syntax.RecordKey
	CreatedAt   time.Time
	IntendedURI syntax.ATURI
	RecordBytes []byte
	RecordHash  [sha256.Size]byte
}

func FreezePublication(
	existing *FrozenPublication,
	request PublicationFreezeRequest,
) (FrozenPublication, error) {
	if existing != nil {
		frozen := *existing
		frozen.RecordBytes = append([]byte(nil), existing.RecordBytes...)
		return frozen, nil
	}
	if request.Owner == "" || request.TID == "" || request.CreatedAt.IsZero() || len(request.Body) == 0 {
		return FrozenPublication{}, ErrInvalidPublicationFreeze
	}
	rkey, err := syntax.ParseRecordKey(request.TID.String())
	if err != nil {
		return FrozenPublication{}, fmt.Errorf("%w: record key", ErrInvalidPublicationFreeze)
	}
	uri, err := syntax.ParseATURI(fmt.Sprintf("at://%s/%s/%s", request.Owner, PostCollection, rkey))
	if err != nil {
		return FrozenPublication{}, fmt.Errorf("%w: intended uri", ErrInvalidPublicationFreeze)
	}
	body := append([]byte(nil), request.Body...)
	return FrozenPublication{
		Rkey:        rkey,
		CreatedAt:   request.CreatedAt.UTC(),
		IntendedURI: uri,
		RecordBytes: body,
		RecordHash:  sha256.Sum256(body),
	}, nil
}
