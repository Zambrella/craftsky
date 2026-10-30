package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

// IT-016: optional commit evidence must retain the existing session fence.
func TestCoordinatedPDSClientKeepsCommitEvidenceInsideSessionFence(t *testing.T) {
	owner := syntax.DID("did:plc:coordinated-client")
	runner := &recordingSessionRunner{owner: owner, sessionID: "parent"}
	client := &fenceCheckingCommitClient{fenceCheckingPDSClient: &fenceCheckingPDSClient{inside: &runner.inside}}
	coordinated, err := auth.NewCoordinatedPDSClient(runner, owner, "parent", func(context.Context, *oauth.ClientSession) (auth.PDSClient, error) {
		if !runner.inside {
			return nil, errors.New("client built outside fence")
		}
		return client, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	headClient, ok := any(coordinated).(auth.RepositoryHeadMetadataClient)
	if !ok {
		t.Fatal("coordinated client does not forward repository-head evidence")
	}
	writeClient, ok := any(coordinated).(auth.RepositoryWriteMetadataClient)
	if !ok {
		t.Fatal("coordinated client does not forward write-commit evidence")
	}
	head, err := headClient.LatestCommitMetadata(context.Background(), owner)
	if err != nil || head.CID != "head" || head.Revision != "3aaaaaaaaaaaa" {
		t.Fatalf("head=%+v %v", head, err)
	}
	receipt, err := writeClient.ApplyWritesWithCommit(context.Background(), owner, head.CID, []auth.RepositoryWrite{{Action: "update", Collection: "social.craftsky.feed.post", RKey: "one", Record: map[string]any{"text": "after"}}})
	commit := receipt.Commit
	if err != nil || commit.CID != "commit" || commit.Revision != "3aaaaaaaaaaab" || runner.calls != 2 || client.calls != 2 || runner.inside {
		t.Fatalf("commit=%+v %v calls=%d/%d inside=%v", commit, err, runner.calls, client.calls, runner.inside)
	}
}

type fenceCheckingCommitClient struct{ *fenceCheckingPDSClient }

func (client *fenceCheckingCommitClient) LatestCommitMetadata(context.Context, syntax.DID) (auth.RepositoryCommit, error) {
	return auth.RepositoryCommit{CID: "head", Revision: "3aaaaaaaaaaaa"}, client.check()
}
func (client *fenceCheckingCommitClient) ApplyWritesWithCommit(context.Context, syntax.DID, syntax.CID, []auth.RepositoryWrite) (auth.RepositoryWriteReceipt, error) {
	return auth.RepositoryWriteReceipt{Commit: auth.RepositoryCommit{CID: "commit", Revision: "3aaaaaaaaaaab"}}, client.check()
}
