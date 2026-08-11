package accountdeletion

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestStatusCapabilityAuthorizationIsNarrow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	grant := StatusGrant{
		JobID:     "job-alice",
		Owner:     syntax.DID("did:plc:alice"),
		ExpiresAt: now.Add(time.Hour),
	}
	for _, action := range []StatusAction{StatusRead, StatusStartReauthentication, StatusRetry} {
		if err := AuthorizeStatus(grant, now, grant.JobID, grant.Owner, action); err != nil {
			t.Fatalf("matching %s authorization failed: %v", action, err)
		}
	}

	denied := []struct {
		name   string
		grant  StatusGrant
		jobID  string
		owner  syntax.DID
		action StatusAction
		now    time.Time
	}{
		{name: "cross job", grant: grant, jobID: "job-bob", owner: grant.Owner, action: StatusRead, now: now},
		{name: "cross owner", grant: grant, jobID: grant.JobID, owner: syntax.DID("did:plc:bob"), action: StatusRead, now: now},
		{name: "ordinary API", grant: grant, jobID: grant.JobID, owner: grant.Owner, action: StatusOrdinaryAPI, now: now},
		{name: "PDS API", grant: grant, jobID: grant.JobID, owner: grant.Owner, action: StatusPDSAPI, now: now},
		{name: "expired", grant: grant, jobID: grant.JobID, owner: grant.Owner, action: StatusRead, now: grant.ExpiresAt},
		{name: "revoked", grant: StatusGrant{JobID: grant.JobID, Owner: grant.Owner, ExpiresAt: grant.ExpiresAt, Revoked: true}, jobID: grant.JobID, owner: grant.Owner, action: StatusRead, now: now},
	}
	for _, test := range denied {
		t.Run(test.name, func(t *testing.T) {
			if err := AuthorizeStatus(test.grant, test.now, test.jobID, test.owner, test.action); !errors.Is(err, ErrStatusUnauthorized) {
				t.Fatalf("authorization error = %v, want non-leaking ErrStatusUnauthorized", err)
			}
		})
	}

	view := ProjectDeletionStatus(grant.JobID, StatusActive, PhaseRemovingCraftskyRecords, false)
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "oauth") || strings.Contains(string(encoded), string(grant.Owner)) {
		t.Fatalf("status projection disclosed owner or OAuth details: %s", encoded)
	}
}
