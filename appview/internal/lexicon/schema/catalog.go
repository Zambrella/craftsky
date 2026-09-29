package schema

import (
	"embed"
	"encoding/json"
	"fmt"

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
		return fmt.Errorf("decode %s record: %w", kind, err)
	}
	record := decoded
	if record == nil {
		return fmt.Errorf("decode %s record: expected object", kind)
	}
	if _, ok := record["$type"]; !ok {
		record["$type"] = nsid
	}
	return indigolexicon.ValidateRecord(craftskyCatalog, record, nsid, 0)
}
