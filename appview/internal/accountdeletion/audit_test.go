package accountdeletion

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestDeletionAuditProjectionAndExpiry(t *testing.T) {
	t.Parallel()

	did := syntax.DID("did:plc:audit-canary")
	acceptedAt := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	terminalAt := acceptedAt.Add(2 * time.Hour)
	source := AuditSource{
		JobID:                "job-audit-1",
		DID:                  did,
		AcceptedAt:           acceptedAt,
		Handle:               "forbidden-handle.craftsky.social",
		OAuthSessionID:       "forbidden-oauth-session",
		StatusCapabilityHash: "forbidden-status-hash",
		ExpectedURI:          "at://did:plc:audit-canary/social.craftsky.post/forbidden",
		RecordContent:        "forbidden-record-content",
	}

	audit := NewDeletionAudit(source, terminalAt, AuditOutcomeDeleted)
	if audit.JobID != source.JobID || audit.DID != did || audit.AcceptedAt != acceptedAt || audit.TerminalAt != terminalAt {
		t.Fatalf("audit identity/timestamps = %+v", audit)
	}
	if audit.Outcome != AuditOutcomeDeleted {
		t.Fatalf("audit outcome = %q", audit.Outcome)
	}
	wantExpiry := terminalAt.Add(30 * 24 * time.Hour)
	if !audit.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("audit expiry = %s, want %s", audit.ExpiresAt, wantExpiry)
	}

	encoded, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{source.Handle, source.OAuthSessionID, source.StatusCapabilityHash, source.ExpectedURI, source.RecordContent} {
		if strings.Contains(string(encoded), prohibited) {
			t.Fatalf("audit contains prohibited value %q: %s", prohibited, encoded)
		}
	}

	if !AuditRetainedAt(audit, wantExpiry.Add(-time.Nanosecond)) {
		t.Fatal("audit must exist immediately before expiry")
	}
	if AuditRetainedAt(audit, wantExpiry) || AuditRetainedAt(audit, wantExpiry.Add(time.Nanosecond)) {
		t.Fatal("audit must expire at and after its exact boundary")
	}
	if AuditBlocksRejoin(audit) {
		t.Fatal("a retained deletion audit must never block a fresh membership")
	}
}
