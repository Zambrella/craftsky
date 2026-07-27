package instagrammeta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignatureAcceptsOnlyExactSHA256HeaderOverExactBytes(t *testing.T) {
	t.Parallel()

	secret := []byte("synthetic-app-secret-only-for-tests")
	body := []byte("{\"object\":\"instagram\",\"synthetic\":true}\n")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	digest := hex.EncodeToString(mac.Sum(nil))
	valid := "sha256=" + digest

	if err := VerifySignature(secret, body, valid); err != nil {
		t.Fatalf("VerifySignature(valid): %v", err)
	}

	mutated := append([]byte(nil), body...)
	mutated[len(mutated)-1] = ' '
	for name, input := range map[string]struct {
		secret []byte
		body   []byte
		header string
	}{
		"mutated body":     {secret: secret, body: mutated, header: valid},
		"wrong secret":     {secret: []byte("different-synthetic-secret"), body: body, header: valid},
		"missing header":   {secret: secret, body: body},
		"wrong algorithm":  {secret: secret, body: body, header: "sha1=" + digest},
		"case variant":     {secret: secret, body: body, header: "SHA256=" + digest},
		"short digest":     {secret: secret, body: body, header: "sha256=" + digest[:62]},
		"non-hex digest":   {secret: secret, body: body, header: "sha256=" + digest[:63] + "z"},
		"extra whitespace": {secret: secret, body: body, header: " " + valid},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := VerifySignature(input.secret, input.body, input.header); err == nil {
				t.Fatal("VerifySignature unexpectedly accepted invalid input")
			}
		})
	}
}

func TestVerifySignatureValuesRequiresExactlyOneHeaderValue(t *testing.T) {
	t.Parallel()

	secret := []byte("synthetic-app-secret-only-for-tests")
	body := []byte(`{"object":"instagram"}`)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	valid := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := VerifySignatureValues(secret, body, []string{valid}); err != nil {
		t.Fatalf("VerifySignatureValues(valid): %v", err)
	}
	for _, values := range [][]string{nil, {}, {valid, valid}} {
		if err := VerifySignatureValues(secret, body, values); err == nil {
			t.Fatalf("VerifySignatureValues(%d values) unexpectedly succeeded", len(values))
		}
	}
}
