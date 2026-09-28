package api

import (
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

// An explicit set removal must recognize exactly the records that can become
// valid source facts through Tap, including records written by other clients.
func validAuthoritativeSetRecord(record pdscommands.AuthoritativeRecord, collection syntax.NSID) bool {
	parsed, err := syntax.ParseATURI(record.URI.String())
	if err != nil || parsed.Collection() != collection || record.CID == "" {
		return false
	}
	result := sourcevalidation.Validate(tap.Event{
		URI: record.URI, DID: parsed.Authority().DID(), Collection: collection,
		Rkey: parsed.RecordKey(), Action: "create", CID: record.CID, Record: record.Record,
	})
	return result.StructuralStatus == sourcevalidation.Valid && result.SemanticStatus == sourcevalidation.Valid
}
