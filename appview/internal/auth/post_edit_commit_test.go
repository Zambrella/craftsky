package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// IT-016 / IT-005: edit reconciliation needs repository revision evidence,
// rather than ordering by a CID or a request's local completion time.
func TestIndigoPDSClientPreservesRepositoryHeadRevision(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonPDSResponse(http.StatusOK, `{"cid":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq","rev":"3aaaaaaaaaaaa"}`), nil
	}))
	metadata, ok := any(client).(RepositoryHeadMetadataClient)
	if !ok {
		t.Fatal("PDS adapter does not expose repository revision evidence")
	}
	head, err := metadata.LatestCommitMetadata(context.Background(), "did:plc:xyz")
	if err != nil || head.CID != "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq" || head.Revision != "3aaaaaaaaaaaa" {
		t.Fatalf("repository head = %+v, %v", head, err)
	}
}

// IT-016 / IT-005: retain the write response's commit, not a later repo head.
func TestIndigoPDSClientPreservesReturnedWriteCommit(t *testing.T) {
	calls := 0
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/xrpc/com.atproto.repo.applyWrites" {
			t.Fatalf("unexpected request %s", r.URL.Path)
		}
		var body struct {
			SwapCommit string
			Writes     []struct {
				Type       string `json:"$type"`
				SwapRecord string
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.SwapCommit != "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq" || len(body.Writes) != 1 || body.Writes[0].Type != "com.atproto.repo.applyWrites#update" || body.Writes[0].SwapRecord != "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq" {
			t.Fatalf("conditional update=%+v", body)
		}
		return jsonPDSResponse(http.StatusOK, `{"commit":{"cid":"bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m","rev":"3aaaaaaaaaaab"},"results":[]}`), nil
	}))
	metadata, ok := any(client).(RepositoryWriteMetadataClient)
	if !ok {
		t.Fatal("PDS adapter discards returned write commit evidence")
	}
	receipt, err := metadata.ApplyWritesWithCommit(context.Background(), "did:plc:xyz", "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", []RepositoryWrite{{Action: "update", Collection: "social.craftsky.feed.post", RKey: "3existingpost", ExpectedCID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Record: map[string]any{"text": "after"}}})
	commit := receipt.Commit
	if err != nil || commit.CID != "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m" || commit.Revision != "3aaaaaaaaaaab" || calls != 1 {
		t.Fatalf("returned commit=%+v, %v requests=%d", commit, err, calls)
	}
}

// IT-005 / IT-016: a later external update cannot erase the identity returned
// by the successful conditional write itself.
func TestIndigoPDSClientPreservesReturnedRecordIdentity(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonPDSResponse(http.StatusOK, `{"commit":{"cid":"bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m","rev":"3aaaaaaaaaaab"},"results":[{"$type":"com.atproto.repo.applyWrites#updateResult","uri":"at://did:plc:xyz/social.craftsky.feed.post/3existingpost","cid":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq","validationStatus":"valid"}]}`), nil
	}))
	receipt, err := client.ApplyWritesWithCommit(context.Background(), "did:plc:xyz", "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", []RepositoryWrite{{Action: "update", Collection: "social.craftsky.feed.post", RKey: "3existingpost", Record: map[string]any{"text": "after"}}})
	if err != nil || len(receipt.Records) != 1 || receipt.Records[0].URI != "at://did:plc:xyz/social.craftsky.feed.post/3existingpost" || receipt.Records[0].CID != "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq" {
		t.Fatalf("write receipt lost record result: %+v, %v", receipt, err)
	}
}
