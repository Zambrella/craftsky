package pdscommands

import (
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestFingerprintGoldenVectors(t *testing.T) {
	request, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            "did:plc:golden",
		OperationKind:    "create_post",
		Intent:           json.RawMessage(`{"text":"hello","sponsored":false,"count":7}`),
		Conditions:       json.RawMessage(`{"ownerGeneration":3}`),
		Blobs: []BlobReference{{
			CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", MIMEType: "image/jpeg", Size: 1024,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	const wantRequest = "e7a4d65ff0e7e4476c9f074607aba99175bb5139508710551253e3e236893900"
	if got := hex.EncodeToString(request[:]); got != wantRequest {
		t.Fatalf("request fingerprint = %s, want %s", got, wantRequest)
	}

	dispatch, err := DispatchFingerprint(DispatchFingerprintInput{
		AlgorithmVersion:   1,
		RequestFingerprint: request,
		RepositoryHead:     "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fa",
		Generated:          json.RawMessage(`{"createdAt":"2026-09-23T18:00:00Z","rkey":"3aaaaaaaaaaa2"}`),
		Steps: []DispatchStep{
			{Action: "create", URI: "at://did:plc:golden/social.craftsky.feed.post/3aaaaaaaaaaa2", Body: json.RawMessage(`{"createdAt":"2026-09-23T18:00:00Z","sponsored":false,"text":"hello"}`)},
			{Action: "delete", URI: "at://did:plc:golden/social.craftsky.feed.like/3aaaaaaaaaaa3", ExpectedCID: "bafy-old"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	const wantDispatch = "266444052d63a204687b2dd3598a6a6c3787631728170da55b06e6f0541ff732"
	if got := hex.EncodeToString(dispatch[:]); got != wantDispatch {
		t.Fatalf("dispatch fingerprint = %s, want %s", got, wantDispatch)
	}
}
