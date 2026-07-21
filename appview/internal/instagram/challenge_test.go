package instagram

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestChallengeGenerateUsesCanonicalHighEntropyGrammar(t *testing.T) {
	t.Parallel()

	codec, err := NewChallengeCodec(newDeterministicReader(), bytes.Repeat([]byte{0x5a}, 32))
	if err != nil {
		t.Fatalf("NewChallengeCodec: %v", err)
	}

	grammar := regexp.MustCompile(`^CSKY-[23456789ABCDEFGHJKMNPQRSTVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTVWXYZ]$`)
	seen := make(map[string]struct{}, 10_000)
	for range 10_000 {
		issued, err := codec.Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if display := issued.Display(); !grammar.MatchString(display) {
			t.Fatalf("generated challenge %q does not match canonical grammar", display)
		}
		seen[issued.Display()] = struct{}{}
	}
	if len(seen) != 10_000 {
		t.Fatalf("generated %d unique challenges, want 10000", len(seen))
	}
}

func TestChallengeCanonicalizeAcceptsOnlyASCIIOuterNormalization(t *testing.T) {
	t.Parallel()

	const canonical = "CSKY-2345-6789-ABCD-E"
	for _, input := range []string{
		canonical,
		"csky-2345-6789-abcd-e",
		" \t\r\ncSkY-2345-6789-AbCd-E\n ",
	} {
		got, err := CanonicalizeChallenge(input)
		if err != nil {
			t.Errorf("CanonicalizeChallenge(%q): %v", input, err)
			continue
		}
		if got != canonical {
			t.Errorf("CanonicalizeChallenge(%q) = %q, want %q", input, got, canonical)
		}
	}

	invalid := []string{
		"CSKY-2345-6789-ABCD-E more",
		"send CSKY-2345-6789-ABCD-E",
		"CSKY -2345-6789-ABCD-E",
		"CSKY-2345 6789-ABCD-E",
		"CSKY-2345-6789-ABCD-EF",
		"CSKY-2345-6789-ABCD",
		"CSKY_2345_6789_ABCD_E",
		"CSKY-2345-6789-ABCD-É",
		"CSKY-2345-6789-ABCD-I",
		"CSKY-2345-6789-ABCD-O",
		"CSKY-2345-6789-ABCD-0",
		"CSKY-2345-6789-ABCD-1",
		"\u00a0CSKY-2345-6789-ABCD-E\u00a0",
		"",
	}
	for _, input := range invalid {
		if got, err := CanonicalizeChallenge(input); err == nil {
			t.Errorf("CanonicalizeChallenge(%q) unexpectedly returned %q", input, got)
		}
	}
}

func TestChallengeDigestIsKeyedComparableAndStorageSafe(t *testing.T) {
	t.Parallel()

	const (
		canonical   = "CSKY-2345-6789-ABCD-E"
		didCanary   = "did:plc:synthetic-private-canary"
		emailCanary = "synthetic-private@example.invalid"
		tokenCanary = "synthetic-meta-token-canary"
	)
	keyA := bytes.Repeat([]byte{0xa1}, 32)
	keyB := bytes.Repeat([]byte{0xb2}, 32)
	codecA, err := NewChallengeCodec(strings.NewReader(strings.Repeat("\x01", 128)), keyA)
	if err != nil {
		t.Fatalf("NewChallengeCodec A: %v", err)
	}
	codecB, err := NewChallengeCodec(strings.NewReader(strings.Repeat("\x02", 128)), keyB)
	if err != nil {
		t.Fatalf("NewChallengeCodec B: %v", err)
	}

	digestA, err := codecA.Digest("  csky-2345-6789-abcd-e \n")
	if err != nil {
		t.Fatalf("Digest normalized: %v", err)
	}
	digestAAgain, err := codecA.Digest(canonical)
	if err != nil {
		t.Fatalf("Digest canonical: %v", err)
	}
	digestB, err := codecB.Digest(canonical)
	if err != nil {
		t.Fatalf("Digest other key: %v", err)
	}
	if !digestA.Equal(digestAAgain) {
		t.Fatal("equivalent normalized challenges did not compare equal")
	}
	if digestA.Equal(digestB) {
		t.Fatal("challenge digest compared equal under a different key")
	}

	issued, err := codecA.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	stored := issued.Stored()
	if stored.Digest.Version != ChallengeDigestVersion {
		t.Fatalf("digest version = %d, want %d", stored.Digest.Version, ChallengeDigestVersion)
	}
	if stored.Digest.IsZero() {
		t.Fatal("stored digest is zero")
	}

	diagnostic := fmt.Sprintf("issued=%v stored=%v digest=%v metadata=%s/%s/%s", issued, stored, stored.Digest, didCanary, emailCanary, tokenCanary)
	for _, secret := range []string{issued.Display(), canonical} {
		if strings.Contains(diagnostic, secret) {
			t.Fatalf("diagnostic leaked challenge plaintext %q: %s", secret, diagnostic)
		}
	}
	if strings.Contains(fmt.Sprint(stored), didCanary) ||
		strings.Contains(fmt.Sprint(stored), emailCanary) ||
		strings.Contains(fmt.Sprint(stored), tokenCanary) {
		t.Fatal("stored challenge unexpectedly contains member or provider metadata")
	}
}

func TestChallengeCodecRejectsUnsafeKeysAndEntropyFailure(t *testing.T) {
	t.Parallel()

	if _, err := NewChallengeCodec(bytes.NewReader(nil), bytes.Repeat([]byte{1}, 31)); err == nil {
		t.Fatal("NewChallengeCodec accepted a key shorter than 32 bytes")
	}

	want := errors.New("synthetic entropy failure")
	codec, err := NewChallengeCodec(errorReader{err: want}, bytes.Repeat([]byte{2}, 32))
	if err != nil {
		t.Fatalf("NewChallengeCodec: %v", err)
	}
	if _, err := codec.Generate(); !errors.Is(err, want) {
		t.Fatalf("Generate error = %v, want wrapped %v", err, want)
	}
}

type deterministicReader struct {
	counter uint64
	buffer  []byte
}

func newDeterministicReader() io.Reader {
	return &deterministicReader{}
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	for len(r.buffer) < len(p) {
		var input [8]byte
		binary.BigEndian.PutUint64(input[:], r.counter)
		sum := sha256.Sum256(input[:])
		r.buffer = append(r.buffer, sum[:]...)
		r.counter++
	}
	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
