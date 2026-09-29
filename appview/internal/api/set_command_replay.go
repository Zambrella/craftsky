package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/pdscommands"
)

type setCommandReplayLookup interface {
	LookupCommand(context.Context, syntax.DID, string, uuid.UUID) (*pdscommands.ReplayCommand, error)
}

// existingSetCommand checks immutable route identity before using stored inputs.
// Mutable target reads and policy checks belong to new commands only.
func existingSetCommand(
	writer http.ResponseWriter, request *http.Request, commands SetCommandExecutor,
	owner syntax.DID, generation int64, kind string, key uuid.UUID,
	requestTarget string, runID string,
) (*pdscommands.ReplayCommand, bool) {
	lookup, ok := commands.(setCommandReplayLookup)
	if !ok {
		return nil, false
	}
	replay, err := lookup.LookupCommand(request.Context(), owner, kind, key)
	if err != nil {
		WriteCommandError(writer, runID, err)
		return nil, true
	}
	if replay == nil {
		return nil, false
	}
	var intent struct {
		SubjectURI    string `json:"subjectUri"`
		RequestTarget string `json:"requestTarget"`
	}
	if json.Unmarshal(replay.Intent, &intent) != nil || replay.OwnerGeneration != generation ||
		(intent.SubjectURI != requestTarget && intent.RequestTarget != requestTarget) {
		WriteCommandError(writer, runID, pdscommands.ErrIdempotencyConflict)
		return nil, true
	}
	if replay.Result.State == pdscommands.CommandAccepted || replay.Result.State == pdscommands.CommandRejected {
		WriteCommandResponse(writer, CommandResultFromStored(replay.Result))
		return nil, true
	}
	return replay, false
}

func commandPostURI(writer http.ResponseWriter, request *http.Request, runID string) (string, syntax.DID, bool) {
	did, err := syntax.ParseDID(request.PathValue("did"))
	if err != nil {
		envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
		return "", "", false
	}
	rkey, err := syntax.ParseRecordKey(request.PathValue("rkey"))
	if err != nil {
		envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "rkey path segment is not a valid record key", runID, nil)
		return "", "", false
	}
	return "at://" + did.String() + "/" + craftskyPostNSID + "/" + rkey.String(), did, true
}

func replayPostTarget(raw json.RawMessage) (*PostTargetRef, bool) {
	var intent struct {
		URI string `json:"subjectUri"`
		CID string `json:"subjectCid"`
	}
	if json.Unmarshal(raw, &intent) != nil || intent.URI == "" || intent.CID == "" {
		return nil, false
	}
	return &PostTargetRef{URI: intent.URI, CID: intent.CID}, true
}
