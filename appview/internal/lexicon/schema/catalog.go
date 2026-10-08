package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/bluesky-social/indigo/atproto/atdata"
	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
)

//go:embed actor/*.json business/*.json
var craftskySchemas embed.FS

var craftskyCatalog = func() *indigolexicon.BaseCatalog {
	catalog := indigolexicon.NewBaseCatalog()
	if err := catalog.LoadEmbedFS(craftskySchemas); err != nil {
		panic(fmt.Sprintf("load embedded Craftsky lexicons: %v", err))
	}
	return catalog
}()

func ValidateBusinessRecord(raw json.RawMessage, nsid string) error {
	return validateRecord(raw, nsid, "business")
}

func ValidateCraftskyRecord(raw json.RawMessage, nsid string) error {
	return validateRecord(raw, nsid, "Craftsky")
}

func validateRecord(raw json.RawMessage, nsid, kind string) error {
	decoded, err := atdata.UnmarshalJSON(raw)
	if err != nil {
		return &ValidationError{Cause: fmt.Errorf("decode %s record: %w", kind, err)}
	}
	record := decoded
	if record == nil {
		return &ValidationError{Cause: fmt.Errorf("decode %s record: expected object", kind)}
	}
	if _, ok := record["$type"]; !ok {
		record["$type"] = nsid
	}
	if err := indigolexicon.ValidateRecord(craftskyCatalog, record, nsid, 0); err != nil {
		failure := &ValidationError{Cause: err}
		if match := oversizedBlob.FindStringSubmatch(err.Error()); match != nil {
			failure.BlobBytes, _ = strconv.ParseInt(match[1], 10, 64)
		}
		return failure
	}
	return nil
}

// ValidationError retains the original validator cause while exposing only
// reviewed rule metadata. Dependency prose and record values stay private.
type ValidationError struct {
	Cause        error
	MissingField string
	BlobBytes    int64
}

var oversizedBlob = regexp.MustCompile(`^blob size too large: ([0-9]{1,18})$`)

func (e *ValidationError) Unwrap() error { return e.Cause }
func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.DiagnosticMessage()
}
func (e *ValidationError) DiagnosticMessage() string {
	switch e.MissingField {
	case "text", "sponsored", "createdAt":
		return "published record missing required field: " + e.MissingField
	}
	if e.BlobBytes > 0 {
		return fmt.Sprintf("published record blob exceeds lexicon size limit (%d bytes)", e.BlobBytes)
	}
	return "published record does not satisfy lexicon"
}
