package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestProtocolRecordPreservesIntegerPrecision(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"uri":"at://did:plc:example/app.bsky.actor.profile/self","cid":"nonempty-cid","value":{"$type":"app.bsky.actor.profile","futureCount":9007199254740993}}`))}, nil
	}))
	var value map[string]any
	_, err := client.GetRecord(context.Background(), "did:plc:example", "app.bsky.actor.profile", "self", &value)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(value)
	t.Logf("round-tripped record: %s", b)
	if !strings.Contains(string(b), "9007199254740993") {
		t.Fatal("record integer changed during read")
	}
}
