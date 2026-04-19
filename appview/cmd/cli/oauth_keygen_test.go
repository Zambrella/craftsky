package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
)

func TestOAuthKeygenRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := runOAuthKeygen(&buf); err != nil {
		t.Fatalf("runOAuthKeygen: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatal("runOAuthKeygen produced empty output")
	}
	priv, err := atcrypto.ParsePrivateMultibase(out)
	if err != nil {
		t.Fatalf("output did not parse as multibase private key: %v", err)
	}
	if _, ok := priv.(*atcrypto.PrivateKeyP256); !ok {
		t.Fatalf("expected P-256 private key, got %T", priv)
	}
}
