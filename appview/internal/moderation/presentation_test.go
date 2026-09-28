package moderation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPresentSubjectSnapshotUsesAnExplicitOwnerSafeAllowlist(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"post",
		"did":"did:plc:owner",
		"collection":"social.craftsky.feed.post",
		"rkey":"3snapshot",
		"uri":"at://did:plc:owner/social.craftsky.feed.post/3snapshot",
		"cid":"bafy-original",
		"submittedHandle":"owner.example",
		"reportText":"private-report-sentinel",
		"evidence":"private-evidence-sentinel",
		"moderatorId":"private-moderator-sentinel",
		"deviceId":"private-device-sentinel",
		"credential":"private-credential-sentinel"
	}`)

	snapshot, err := presentSubjectSnapshot(raw, SubjectSnapshot{})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"cid":"bafy-original"`) {
		t.Fatalf("snapshot lost safe subject data: %s", wire)
	}
	for _, secret := range []string{"private-report-sentinel", "private-evidence-sentinel", "private-moderator-sentinel", "private-device-sentinel", "private-credential-sentinel"} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("snapshot leaked %q: %s", secret, wire)
		}
	}
}

func TestPresentSubjectSnapshotFallsBackToCanonicalRetainedSubject(t *testing.T) {
	fallback := SubjectSnapshot{
		Type:       SubjectPost,
		DID:        "did:plc:owner",
		Collection: "social.craftsky.feed.post",
		Rkey:       "3deleted",
		URI:        "at://did:plc:owner/social.craftsky.feed.post/3deleted",
		CID:        "bafy-before-delete",
	}

	snapshot, err := presentSubjectSnapshot(json.RawMessage(`{"type":"post"}`), fallback)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot != fallback {
		t.Fatalf("snapshot = %+v, want retained fallback %+v", snapshot, fallback)
	}
}
