package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/pdscommands"
)

// PostEditResult carries acceptance independently of asynchronously indexed
// content. Never persist a synthetic post in a replayable edit response: it
// would invent engagement and could replay content hidden after acceptance.
type PostEditResult struct {
	Changed             bool       `json:"changed"`
	AcceptedCID         syntax.CID `json:"acceptedCid"`
	EditedAt            *time.Time `json:"editedAt,omitempty"`
	RecordRevision      syntax.TID `json:"recordRevision,omitempty"`
	ObservationCID      syntax.CID `json:"observationCid,omitempty"`
	ObservationRevision syntax.TID `json:"observationRevision,omitempty"`
}

func acceptedPostEditResult(record pdscommands.AuthoritativeRecord, expected syntax.CID) (pdscommands.TerminalResult, error) {
	result := PostEditResult{Changed: record.CID != expected, AcceptedCID: record.CID,
		RecordRevision: record.Revision, ObservationCID: record.ObservationCID,
		ObservationRevision: record.ObservationRevision}
	var metadata struct {
		EditedAt *time.Time `json:"editedAt"`
	}
	if err := json.Unmarshal(record.Record, &metadata); err != nil {
		return pdscommands.TerminalResult{}, err
	}
	result.EditedAt = postEditTimeUTC(metadata.EditedAt)
	body, err := json.Marshal(result)
	return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body}, err
}
