package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"testing"
)

type protocolRawRecord map[string]any

func (r protocolRawRecord) MarshalCBOR(w io.Writer) error {
	b, e := atdata.MarshalCBOR(map[string]any(r))
	if e != nil {
		return e
	}
	_, e = w.Write(b)
	return e
}

type protocolSnapshotRT struct{ body []byte }

func (rt protocolSnapshotRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(rt.body)), Request: r}, nil
}
func TestProtocolSnapshotPreservesExtensions(t *testing.T) {
	did := syntax.DID("did:plc:repositorysnapshot")
	key := repositorySnapshotTestKey(t, 1)
	for _, unknownEmbed := range []bool{false, true} {
		rec := protocolRawRecord{"$type": "social.craftsky.feed.post", "text": "hello", "sponsored": false, "createdAt": "2026-10-12T10:00:00Z", "extension": "preserve-me"}
		if unknownEmbed {
			rec["embed"] = map[string]any{"$type": "social.craftsky.feed.post#futureEmbed", "url": "https://example.com"}
		}
		fixture := buildRepositorySnapshotCARWithRecords(t, did, key, "3aaaaaaaaaaa2", []repositorySnapshotFixtureRecord{{path: "social.craftsky.feed.post/3aaaaaaaaaaa2", record: rec}})
		directory := staticRepositorySnapshotDirectory(t, did, "https://pds.example.com", key, nil)
		fetcher, err := newRepositorySnapshotFetcher(directory, &http.Client{Transport: protocolSnapshotRT{fixture.data}})
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := fetcher.Fetch(context.Background(), did, repositorySnapshotRegistry{"social.craftsky.feed.post"})

		if err != nil {
			t.Fatal(err)
		}
		_, records, err := snapshot.Collection(did, "social.craftsky.feed.post")
		if err != nil || len(records) != 1 {
			t.Fatalf("records: %v %v", records, err)
		}
		var got map[string]any
		_ = json.Unmarshal(records[0].Record, &got)
		t.Logf("snapshot record: %s", records[0].Record)
		if got["extension"] != "preserve-me" {
			t.Fatal("snapshot lost extension field")
		}
		if unknownEmbed && got["embed"] == nil {
			t.Fatal("snapshot lost open union embed")
		}
	}
}
