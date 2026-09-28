package api

import (
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/pdscommands"
)

func TestAuthoritativeSetValidationMatchesIngestionAcrossCollections(t *testing.T) {
	for _, test := range []struct {
		collection syntax.NSID
		body       string
	}{
		{"app.bsky.graph.follow", `{"subject":"did:plc:target","createdAt":"2026-09-24T12:00:00Z"}`},
		{"app.bsky.graph.block", `{"subject":"did:plc:target","createdAt":"2026-09-24T12:00:00Z"}`},
		{"social.craftsky.feed.like", `{"subject":{"uri":"at://did:plc:target/social.craftsky.feed.post/3aaaaaaaaaaa1","cid":"bafy-subject"},"createdAt":"2026-09-24T12:00:00Z"}`},
		{"social.craftsky.feed.repost", `{"subject":{"uri":"at://did:plc:target/social.craftsky.feed.post/3aaaaaaaaaaa1","cid":"bafy-subject"},"createdAt":"2026-09-24T12:00:00Z"}`},
	} {
		t.Run(test.collection.String(), func(t *testing.T) {
			record := pdscommands.AuthoritativeRecord{
				URI: syntax.ATURI("at://did:plc:actor/" + test.collection.String() + "/3aaaaaaaaaaa2"),
				CID: "bafy-source", Record: json.RawMessage(test.body),
			}
			if !validAuthoritativeSetRecord(record, test.collection) {
				t.Fatal("valid external record without $type was ignored")
			}
			record.URI = syntax.ATURI("at://did:plc:actor/" + test.collection.String() + "/invalid-key")
			if validAuthoritativeSetRecord(record, test.collection) {
				t.Fatal("invalid source key was accepted")
			}
		})
	}
}
