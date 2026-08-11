package accountdeletion

import (
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestBoundOAuthSessionIsWorkerOnly(t *testing.T) {
	t.Parallel()

	binding := OAuthBinding{
		JobID:     "job-alice",
		Owner:     syntax.DID("did:plc:alice"),
		SessionID: "oauth-session-secret",
		Bound:     true,
	}
	sessionID, err := BoundOAuthSessionForWorker(binding, binding.JobID, binding.Owner)
	if err != nil || sessionID != binding.SessionID {
		t.Fatalf("matching worker binding = (%q, %v)", sessionID, err)
	}

	for _, test := range []struct {
		name    string
		binding OAuthBinding
		jobID   string
		owner   syntax.DID
	}{
		{name: "unbound", binding: OAuthBinding{JobID: binding.JobID, Owner: binding.Owner, SessionID: binding.SessionID}, jobID: binding.JobID, owner: binding.Owner},
		{name: "cross job", binding: binding, jobID: "job-bob", owner: binding.Owner},
		{name: "cross owner", binding: binding, jobID: binding.JobID, owner: syntax.DID("did:plc:bob")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BoundOAuthSessionForWorker(test.binding, test.jobID, test.owner); !errors.Is(err, ErrBoundOAuthUnauthorized) {
				t.Fatalf("binding error = %v, want ErrBoundOAuthUnauthorized", err)
			}
		})
	}

	if _, err := BoundOAuthSessionForDevice(binding); !errors.Is(err, ErrBoundOAuthUnauthorized) {
		t.Fatalf("device OAuth disclosure error = %v, want ErrBoundOAuthUnauthorized", err)
	}
}

func TestDeletionOAuthBindingIsIdempotentAndFreshlyReplaceable(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	binding, err := BindDeletionOAuthSession(OAuthBinding{}, "job-alice", owner, "oauth-session-1")
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := BindDeletionOAuthSession(binding, "job-alice", owner, "oauth-session-1")
	if err != nil || duplicate != binding {
		t.Fatalf("duplicate binding = (%+v, %v), want unchanged", duplicate, err)
	}
	if _, err := BindDeletionOAuthSession(binding, "job-alice", owner, "older-unbound-session"); !errors.Is(err, ErrBoundOAuthUnauthorized) {
		t.Fatalf("replacement without fresh proof error = %v", err)
	}

	replacement, err := ReplaceDeletionOAuthSession(binding, "job-alice", owner, "oauth-session-2", true)
	if err != nil || replacement.SessionID != "oauth-session-2" || !replacement.Bound {
		t.Fatalf("fresh replacement = (%+v, %v)", replacement, err)
	}
	if CanActivate(State{Status: StatusNeedsAttention, Phase: PhaseRemovingCraftskyRecords}) {
		t.Fatal("replacement reauthentication must never restore ordinary access")
	}
	if _, err := ReplaceDeletionOAuthSession(binding, "job-alice", owner, "oauth-session-2", false); !errors.Is(err, ErrBoundOAuthUnauthorized) {
		t.Fatalf("unproven replacement error = %v", err)
	}
}
