package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestCoordinatedPDSClientBuildsPurposeClientInsideCombinedEffectSession(t *testing.T) {
	owner := syntax.DID("did:plc:combined-purpose-client")
	participant := syntax.DID("did:plc:combined-purpose-participant")
	runner := &recordingEffectSessionRunner{owner: owner, sessionID: "parent"}
	purposeClient := &fenceCheckingPDSClient{inside: &runner.inside}
	coordinated, err := auth.NewCoordinatedPDSClient(
		runner,
		owner,
		"parent",
		func(ctx context.Context, _ *oauth.ClientSession) (auth.PDSClient, error) {
			if ctx == nil || !runner.inside {
				return nil, errors.New("purpose client built outside combined effect session")
			}
			return purposeClient, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: participant, Generation: 7},
		{Owner: owner, Generation: 3},
	}
	err = coordinated.WithActiveEffects(
		context.Background(),
		expected,
		func(ctx context.Context, client auth.PDSClient) error {
			return client.PutRecord(ctx, owner, "social.craftsky.feed.post", "one", map[string]any{"text": "one"})
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || purposeClient.calls != 1 || runner.inside {
		t.Fatalf("combined purpose calls = runner %d client %d inside %t", runner.calls, purposeClient.calls, runner.inside)
	}
}

func TestCoordinatedPDSClientKeepsEveryMethodInsideSessionFence(t *testing.T) {
	owner := syntax.DID("did:plc:coordinated-client")
	runner := &recordingSessionRunner{owner: owner, sessionID: "parent"}
	client := &fenceCheckingPDSClient{inside: &runner.inside}
	coordinated, err := auth.NewCoordinatedPDSClient(
		runner,
		owner,
		"parent",
		func(ctx context.Context, _ *oauth.ClientSession) (auth.PDSClient, error) {
			if ctx == nil || !runner.inside {
				return nil, errors.New("purpose client built outside session fence")
			}
			return client, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var out map[string]any
	if _, err := coordinated.GetRecord(ctx, owner, "social.craftsky.feed.post", "one", &out); err != nil {
		t.Fatal(err)
	}
	if err := coordinated.PutRecord(ctx, owner, "social.craftsky.actor.profile", "self", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := coordinated.CreateRecord(ctx, owner, "social.craftsky.feed.post", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if err := coordinated.DeleteRecord(ctx, owner, "social.craftsky.feed.post", "one"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := coordinated.ListRecords(ctx, owner, "social.craftsky.feed.post", "", 10); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinated.UploadBlob(ctx, "image/png", []byte("image")); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 6 || client.calls != 6 {
		t.Fatalf("coordinated calls = runner %d client %d, want 6/6", runner.calls, client.calls)
	}
}

func TestCoordinatedDeletionPDSClientUsesExactBindingInsideFence(t *testing.T) {
	owner := syntax.DID("did:plc:coordinated-deletion-client")
	authority := auth.DeletionSessionAuthority{
		DeletionCredentialBinding: auth.DeletionCredentialBinding{
			OperationID: uuid.New(), SessionID: "deletion-parent", CredentialGeneration: 7,
		},
		OwnerGeneration: 2, LeaseToken: uuid.New(),
	}
	runner := &recordingDeletionSessionRunner{owner: owner, authority: authority}
	client := &fenceCheckingPDSClient{inside: &runner.inside}
	coordinated, err := auth.NewCoordinatedDeletionPDSClient(
		runner,
		owner,
		authority,
		func(ctx context.Context, _ *oauth.ClientSession) (auth.PDSClient, error) {
			if ctx == nil || !runner.inside {
				return nil, errors.New("deletion purpose client built outside session fence")
			}
			return client, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, _, err := coordinated.ListRecords(ctx, owner, "social.craftsky.feed.post", "", 10); err != nil {
		t.Fatal(err)
	}
	if err := coordinated.DeleteRecord(ctx, owner, "social.craftsky.feed.post", "one"); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 2 || client.calls != 2 {
		t.Fatalf("coordinated deletion calls = runner %d client %d, want 2/2", runner.calls, client.calls)
	}
}

type recordingSessionRunner struct {
	owner     syntax.DID
	sessionID string
	inside    bool
	calls     int
}

type recordingEffectSessionRunner struct {
	owner     syntax.DID
	sessionID string
	inside    bool
	calls     int
}

func (runner *recordingEffectSessionRunner) WithActiveSession(
	context.Context,
	syntax.DID,
	string,
	auth.OAuthSessionOperation,
) error {
	return errors.New("ordinary session entry point must not be used for a combined effect")
}

func (runner *recordingEffectSessionRunner) WithActiveEffectSession(
	ctx context.Context,
	_ []ownerlifecycle.ExpectedOwner,
	owner syntax.DID,
	sessionID string,
	operation auth.OAuthSessionOperation,
) error {
	if owner != runner.owner || sessionID != runner.sessionID || runner.inside {
		return errors.New("invalid combined effect session scope")
	}
	runner.calls++
	runner.inside = true
	defer func() { runner.inside = false }()
	return operation(ctx, &oauth.ClientSession{})
}

type recordingDeletionSessionRunner struct {
	owner     syntax.DID
	authority auth.DeletionSessionAuthority
	inside    bool
	calls     int
}

func (runner *recordingDeletionSessionRunner) WithDeletionSession(
	ctx context.Context,
	owner syntax.DID,
	authority auth.DeletionSessionAuthority,
	operation auth.OAuthSessionOperation,
) error {
	if owner != runner.owner || authority != runner.authority || runner.inside {
		return errors.New("invalid coordinated deletion session scope")
	}
	runner.calls++
	runner.inside = true
	defer func() { runner.inside = false }()
	return operation(ctx, &oauth.ClientSession{})
}

func (runner *recordingSessionRunner) WithActiveSession(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	operation auth.OAuthSessionOperation,
) error {
	if owner != runner.owner || sessionID != runner.sessionID || runner.inside {
		return errors.New("invalid coordinated session scope")
	}
	runner.calls++
	runner.inside = true
	defer func() { runner.inside = false }()
	return operation(ctx, &oauth.ClientSession{})
}

type fenceCheckingPDSClient struct {
	inside *bool
	calls  int
}

func (client *fenceCheckingPDSClient) check() error {
	if client.inside == nil || !*client.inside {
		return errors.New("PDS method escaped session fence")
	}
	client.calls++
	return nil
}

func (client *fenceCheckingPDSClient) GetRecord(
	context.Context, syntax.DID, string, string, any,
) (string, error) {
	return "cid", client.check()
}

func (client *fenceCheckingPDSClient) PutRecord(
	context.Context, syntax.DID, string, string, any,
) error {
	return client.check()
}

func (client *fenceCheckingPDSClient) CreateRecord(
	context.Context, syntax.DID, string, any,
) (syntax.ATURI, syntax.CID, error) {
	err := client.check()
	return "at://did:plc:coordinated-client/social.craftsky.feed.post/one", "cid", err
}

func (client *fenceCheckingPDSClient) DeleteRecord(
	context.Context, syntax.DID, string, string,
) error {
	return client.check()
}

func (client *fenceCheckingPDSClient) UploadBlob(
	context.Context, string, []byte,
) (*auth.UploadedBlob, error) {
	return &auth.UploadedBlob{}, client.check()
}

func (client *fenceCheckingPDSClient) ListRecords(
	context.Context, syntax.DID, string, string, int,
) ([]auth.PDSRecord, string, error) {
	return nil, "", client.check()
}
