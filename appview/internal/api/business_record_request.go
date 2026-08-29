package api

import (
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	goCID "github.com/ipfs/go-cid"

	"social.craftsky/appview/internal/api/envelope"
)

var ErrPDSRecordConflict = errors.New("pds record conflict")

func ParseBusinessIfMatch(request *http.Request) (syntax.CID, error) {
	raw := request.Header.Get("If-Match")
	if raw == "*" {
		return syntax.CID(raw), nil
	}
	parsed, err := goCID.Decode(raw)
	if err != nil || parsed.String() != raw {
		return "", ErrPDSRecordConflict
	}
	return syntax.CID(raw), nil
}

func RequireBusinessRecordPrecondition(expected, current syntax.CID, exists bool) error {
	if expected == "*" {
		if exists {
			return ErrPDSRecordConflict
		}
		return nil
	}
	if !exists {
		return ErrPDSRecordConflict
	}
	return RequireCurrentCID(expected, current)
}

func RequireCurrentCID(expected, current syntax.CID) error {
	if expected == "" || current == "" || expected != current {
		return ErrPDSRecordConflict
	}
	return nil
}

func WritePDSRecordConflict(w http.ResponseWriter, requestID string) {
	envelope.WriteError(w, http.StatusConflict, "pds_record_conflict", "PDS record changed; refresh and try again", requestID, nil)
}
