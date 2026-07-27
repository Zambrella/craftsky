package instagrammeta

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDigestCodecKeysAndSeparatesWebhookIdentifiers(t *testing.T) {
	t.Parallel()

	const (
		messageCanary   = "synthetic-private-message-id"
		challengeCanary = "synthetic-private-challenge"
	)
	canonicalize := func(input string) (string, error) {
		if input != challengeCanary {
			return "", errors.New("invalid challenge")
		}
		return "CANONICAL-SYNTHETIC-CHALLENGE", nil
	}
	codec, err := NewDigestCodec(bytes.Repeat([]byte{0x31}, 32), canonicalize)
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}

	messageDigest, err := codec.MessageID(messageCanary)
	if err != nil {
		t.Fatalf("MessageID: %v", err)
	}
	messageDigestAgain, err := codec.MessageID(messageCanary)
	if err != nil {
		t.Fatalf("MessageID again: %v", err)
	}
	challengeDigest, err := codec.Challenge(challengeCanary)
	if err != nil {
		t.Fatalf("Challenge: %v", err)
	}
	if !messageDigest.Equal(messageDigestAgain) {
		t.Fatal("same message ID did not produce equal keyed digests")
	}
	if messageDigest.Equal(challengeDigest) {
		t.Fatal("message and challenge digest domains collided")
	}
	if messageDigest.Version != DigestVersion || challengeDigest.Version != DigestVersion {
		t.Fatalf("digest versions = %d/%d, want %d", messageDigest.Version, challengeDigest.Version, DigestVersion)
	}

	diagnostic := fmt.Sprintf("message=%v challenge=%+v go=%#v", messageDigest, challengeDigest, challengeDigest)
	for _, canary := range []string{messageCanary, challengeCanary, "CANONICAL-SYNTHETIC-CHALLENGE"} {
		if strings.Contains(diagnostic, canary) {
			t.Fatalf("diagnostic leaked %q: %s", canary, diagnostic)
		}
	}
}

func TestDigestCodecRejectsUnsafeConfigurationAndInputs(t *testing.T) {
	t.Parallel()

	canonicalize := func(input string) (string, error) { return input, nil }
	if _, err := NewDigestCodec(bytes.Repeat([]byte{1}, 31), canonicalize); err == nil {
		t.Fatal("NewDigestCodec accepted a key shorter than 32 bytes")
	}
	if _, err := NewDigestCodec(bytes.Repeat([]byte{1}, 32), nil); err == nil {
		t.Fatal("NewDigestCodec accepted a nil challenge canonicalizer")
	}

	codec, err := NewDigestCodec(bytes.Repeat([]byte{2}, 32), canonicalize)
	if err != nil {
		t.Fatalf("NewDigestCodec: %v", err)
	}
	if _, err := codec.MessageID(""); err == nil {
		t.Fatal("MessageID accepted an empty value")
	}
	if _, err := codec.Challenge(""); err == nil {
		t.Fatal("Challenge accepted an empty value")
	}
}
