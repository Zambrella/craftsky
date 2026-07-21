package instagrammeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestIntegrationTypesRedactConfigurationAndIdentityValues(t *testing.T) {
	t.Parallel()

	const (
		tokenCanary   = "synthetic-private-token-canary"
		accountCanary = "synthetic-private-account-canary"
	)
	keyCanary := bytes.Repeat([]byte("synthetic-private-hmac-key-canary"), 2)
	codec, err := NewDigestCodec(keyCanary, func(input string) (string, error) { return input, nil })
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	reducer, err := NewPayloadReducer(accountCanary, codec)
	if err != nil {
		t.Fatalf("NewPayloadReducer: %v", err)
	}
	config := HTTPClientConfig{
		HTTPClient:        http.DefaultClient,
		APIVersion:        "v99.0",
		AccessToken:       tokenCanary,
		OfficialAccountID: accountCanary,
	}
	client, err := NewHTTPClient(config)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}

	diagnostic := fmt.Sprintf(
		"codec=%v/%+v/%#v reducer=%v/%+v/%#v config=%v/%+v/%#v client=%v/%+v/%#v",
		codec, codec, codec,
		reducer, reducer, reducer,
		config, config, config,
		client, client, client,
	)
	for _, private := range []string{tokenCanary, accountCanary, string(keyCanary)} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("diagnostic leaked %q: %s", private, diagnostic)
		}
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Marshal config: %v", err)
	}
	for _, private := range []string{tokenCanary, accountCanary} {
		if bytes.Contains(encoded, []byte(private)) {
			t.Fatalf("JSON serialization leaked %q: %s", private, encoded)
		}
	}
}
