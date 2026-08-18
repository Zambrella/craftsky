package pdseffects

import (
	"crypto/sha256"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestCanonicalPutBodyFingerprintMatchesExactJSONRequest(t *testing.T) {
	owner := syntax.DID("did:plc:canonical-effect")
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lcanonical")
	recordA := map[string]any{
		"text":   "hello <craft>",
		"nested": map[string]any{"z": float64(2), "a": true},
	}
	recordB := map[string]any{
		"nested": map[string]any{"a": true, "z": float64(2)},
		"text":   "hello <craft>",
	}

	bodyA, fingerprintA, err := canonicalPutBody(owner, collection, rkey, recordA, "")
	if err != nil {
		t.Fatal(err)
	}
	bodyB, fingerprintB, err := canonicalPutBody(owner, collection, rkey, recordB, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(bodyA) != string(bodyB) || fingerprintA != fingerprintB {
		t.Fatalf("canonical bodies differ:\n%s\n%s", bodyA, bodyB)
	}
	wantBody := `{"collection":"social.craftsky.feed.post","record":{"nested":{"a":true,"z":2},"text":"hello \u003ccraft\u003e"},"repo":"did:plc:canonical-effect","rkey":"3lcanonical"}`
	if string(bodyA) != wantBody {
		t.Fatalf("canonical body = %s\nwant = %s", bodyA, wantBody)
	}
	if fingerprintA != sha256.Sum256([]byte(wantBody)) {
		t.Fatal("fingerprint is not SHA-256 of the exact canonical body")
	}
	casBody, casFingerprint, err := canonicalPutBody(
		owner, collection, rkey, recordA, syntax.CID("bafy-prior-record"),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantCASBody := `{"collection":"social.craftsky.feed.post","record":{"nested":{"a":true,"z":2},"text":"hello \u003ccraft\u003e"},"repo":"did:plc:canonical-effect","rkey":"3lcanonical","swapRecord":"bafy-prior-record"}`
	if string(casBody) != wantCASBody || casFingerprint != sha256.Sum256([]byte(wantCASBody)) {
		t.Fatalf("canonical CAS Put body/fingerprint = %s/%x", casBody, casFingerprint)
	}
}

func TestCanonicalDeleteBodyAndDeterministicURI(t *testing.T) {
	owner := syntax.DID("did:plc:canonical-delete")
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3ldelete")
	body, _, uri, err := canonicalDeleteBody(owner, collection, rkey, "bafy-expected-record")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"collection":"social.craftsky.feed.post","repo":"did:plc:canonical-delete","rkey":"3ldelete","swapRecord":"bafy-expected-record"}` {
		t.Fatalf("canonical delete body = %s", body)
	}
	if uri.String() != "at://did:plc:canonical-delete/social.craftsky.feed.post/3ldelete" {
		t.Fatalf("deterministic URI = %q", uri)
	}
	withoutCAS, _, _, err := canonicalDeleteBody(owner, collection, rkey, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(withoutCAS) != `{"collection":"social.craftsky.feed.post","repo":"did:plc:canonical-delete","rkey":"3ldelete"}` {
		t.Fatalf("unconditional canonical delete body = %s", withoutCAS)
	}
}
