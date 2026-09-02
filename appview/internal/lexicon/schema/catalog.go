package schema

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atdata"
	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
)

//go:embed business/*.json
var businessSchemas embed.FS

var businessCatalog = func() *indigolexicon.BaseCatalog {
	catalog := indigolexicon.NewBaseCatalog()
	if err := catalog.LoadEmbedFS(businessSchemas); err != nil {
		panic(fmt.Sprintf("load embedded business lexicons: %v", err))
	}
	return catalog
}()

func ValidateBusinessRecord(raw json.RawMessage, nsid string) error {
	decoded, err := atdata.UnmarshalJSON(raw)
	if err != nil {
		return fmt.Errorf("decode business record: %w", err)
	}
	record := decoded
	if record == nil {
		return fmt.Errorf("decode business record: expected object")
	}
	if _, ok := record["$type"]; !ok {
		record["$type"] = nsid
	}
	return indigolexicon.ValidateRecord(businessCatalog, record, nsid, 0)
}
