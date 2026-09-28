package pdscommands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestRequestAndDispatchFingerprintsAreCanonicalAndScoped(t *testing.T) {
	key := uuid.MustParse("018f4d5c-7a61-7d40-a1a2-0123456789ab")
	leftScope := ScopedCommandKey{Owner: "did:plc:alice", OperationKind: "like", OperationKey: key}
	rightScope := ScopedCommandKey{Owner: "did:plc:bob", OperationKind: "like", OperationKey: key}
	otherKind := ScopedCommandKey{Owner: "did:plc:alice", OperationKind: "repost", OperationKey: key}
	if leftScope == rightScope || leftScope == otherKind {
		t.Fatal("same UUID must remain independent across owner and operation-kind scopes")
	}

	requestA := RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            syntax.DID("did:plc:alice"),
		OperationKind:    "like",
		Intent:           json.RawMessage(`{"subject":{"uri":"at://did:plc:post/social.craftsky.feed.post/3aaaaaaaaaaa2","cid":"bafy"},"enabled":true}`),
		Conditions:       json.RawMessage(`{"generation":7,"expected":null}`),
	}
	requestB := requestA
	requestB.Intent = json.RawMessage(`{ "enabled": true, "subject": { "cid": "bafy", "uri": "at://did:plc:post/social.craftsky.feed.post/3aaaaaaaaaaa2" } }`)
	requestB.Conditions = json.RawMessage(`{"expected":null,"generation":7}`)
	fingerprintA, err := RequestFingerprint(requestA)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintB, err := RequestFingerprint(requestB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fingerprintA[:], fingerprintB[:]) {
		t.Fatalf("equivalent request JSON produced different fingerprints: %x != %x", fingerprintA, fingerprintB)
	}
	changed := requestA
	changed.OperationKind = "repost"
	changedFingerprint, err := RequestFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprintA == changedFingerprint {
		t.Fatal("changed immutable request intent produced the same fingerprint")
	}

	plan := DispatchFingerprintInput{
		AlgorithmVersion:   1,
		RequestFingerprint: fingerprintA,
		RepositoryHead:     "bafy-head",
		Generated:          json.RawMessage(`{"rkey":"3aaaaaaaaaaa2","createdAt":"2026-09-23T17:00:00Z"}`),
		Steps: []DispatchStep{
			{Action: "create", URI: "at://did:plc:alice/social.craftsky.feed.like/3aaaaaaaaaaa2", Body: json.RawMessage(`{"subject":"post"}`)},
		},
	}
	dispatchA, err := DispatchFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.RepositoryHead = "bafy-other-head"
	dispatchB, err := DispatchFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if dispatchA == dispatchB {
		t.Fatal("different exact dispatch plans produced the same fingerprint")
	}
}
