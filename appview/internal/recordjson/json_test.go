package recordjson

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecordIntegerLimitsAndRoundTrip(t *testing.T) {
	raw := []byte(`{"$type":"app.bsky.actor.profile","min":-9223372036854775808,"max":9223372036854775807,"extension":{"counter":9007199254740993}}`)
	record, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if record["min"] != int64(-9223372036854775808) || record["max"] != int64(9223372036854775807) {
		t.Fatalf("signed integers changed: %+v", record)
	}
	var value map[string]any
	if err := Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	roundTrip, err := json.Marshal(value)
	if err != nil || !strings.Contains(string(roundTrip), "9007199254740993") {
		t.Fatalf("round trip=%s error=%v", roundTrip, err)
	}
}

func TestRecordRejectsNonIntegerAndNonRecordData(t *testing.T) {
	for _, raw := range []string{`{"value":1.5}`, `{"value":9223372036854775808}`, `null`, `[]`, `{} {}`, `{"$link":"invalid"}`} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("invalid record accepted: %s", raw)
		}
	}
}
