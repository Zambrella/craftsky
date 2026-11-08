package api

import (
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// IR-004 / IT-004, REG-004 / FR-003, NFR-001 / AC-005, AC-013.
func TestPublicPostPreservesImportedDisclosure(t *testing.T) {
	source := "instagram"
	row := &PostRow{URI: "at://did:plc:alice/social.craftsky.feed.post/imported", CID: "bafy-imported", DID: "did:plc:alice", Rkey: "imported", ExternalImportSource: &source}
	member := BuildPostResponse(row, syntax.Handle("alice.test"))
	if member.ExternalImport == nil {
		t.Fatal("fixture lacks member provenance")
	}
	store := &PublicPageStore{posts: &PostStore{}}
	public := store.publicPost(row, PublicAuthor{DID: "did:plc:alice", Handle: "alice.test"})
	wire, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(wire, &body); err != nil {
		t.Fatal(err)
	}
	var disclosure ExternalImportResponse
	if err := json.Unmarshal(body["externalImport"], &disclosure); err != nil {
		t.Fatalf("guest DTO lost public disclosure: %s (%v)", wire, err)
	}
	if disclosure != *member.ExternalImport {
		t.Fatalf("guest/member provenance differs: %+v %+v", disclosure, member.ExternalImport)
	}
}
