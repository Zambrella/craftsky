// Package postrecord owns current-record editing values without API or storage dependencies.
package postrecord

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/recordjson"
)

type EditTimeSource string

const (
	EditConfirmed EditTimeSource = "confirmed"
	EditReported  EditTimeSource = "reported"
	EditObserved  EditTimeSource = "observed"
)

// EditState describes only the current version. A nil EditedAt means that no
// content edit is known; creation time or discovery is not evidence of editing.
type EditState struct {
	CID           syntax.CID
	Revision      syntax.TID
	ContentDigest [32]byte
	EditedAt      *time.Time
	TimeSource    EditTimeSource
}

func ObserveEdit(previous *EditState, record json.RawMessage, cid syntax.CID, revision syntax.TID, observedAt time.Time) (EditState, error) {
	if previous != nil && previous.Revision >= revision {
		return *previous, nil
	}
	fields, err := recordjson.Parse(record)
	if err != nil {
		return EditState{}, err
	}
	declared, _ := fields["editedAt"].(string)
	delete(fields, "editedAt")
	canonical, err := json.Marshal(fields)
	if err != nil {
		return EditState{}, err
	}
	state := EditState{CID: cid, Revision: revision, ContentDigest: sha256.Sum256(canonical)}
	if previous != nil && previous.ContentDigest == state.ContentDigest {
		state.EditedAt = previous.EditedAt
		state.TimeSource = previous.TimeSource
		return state, nil
	}
	if timestamp, err := time.Parse(time.RFC3339Nano, declared); err == nil && !timestamp.After(observedAt) {
		state.EditedAt = &timestamp
		state.TimeSource = EditReported
	}
	if previous != nil && previous.ContentDigest != state.ContentDigest && state.EditedAt == nil {
		timestamp := observedAt.UTC()
		state.EditedAt = &timestamp
		state.TimeSource = EditObserved
	}
	return state, nil
}
