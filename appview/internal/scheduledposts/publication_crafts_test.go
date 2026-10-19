package scheduledposts

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestScheduledPostCraftsPublication(t *testing.T) {
	payload, err := DecodePayload([]byte(`{"kind":"standard","text":"hello","sponsored":false,"crafts":[{"craftType":"social.craftsky.feed.defs#sewing"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	payload, err = DecodePayload(encoded)
	if err != nil {
		t.Fatal(err)
	}
	record, err := publicationRecord(payload, nil, time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"crafts":[{"craftType":"social.craftsky.feed.defs#sewing"}]`)) || record["project"] != nil {
		t.Fatalf("publication changed reserved metadata: %s", raw)
	}
}
