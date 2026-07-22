package followwrite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

func TestServiceSharesOrdinaryCreateAndDeterministicInstagramPut(t *testing.T) {
	t.Parallel()
	client := &recordingPDS{}
	service := NewService(func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) { return client, nil })
	owner := syntax.DID("did:plc:synthetic-follow-owner")
	target := syntax.DID("did:plc:synthetic-follow-target")
	createdAt := time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)

	if err := service.Write(context.Background(), owner, target, "ordinary-session", nil, createdAt); err != nil {
		t.Fatal(err)
	}
	rkey := syntax.RecordKey("3ksyntheticfollow")
	if err := service.Write(context.Background(), owner, target, "instagram-session", &rkey, createdAt); err != nil {
		t.Fatal(err)
	}
	if client.creates != 1 || client.puts != 1 || client.putRkey != rkey.String() {
		t.Fatalf("creates=%d puts=%d putRkey=%s", client.creates, client.puts, client.putRkey)
	}
	for _, record := range []map[string]any{client.createRecord, client.putRecord} {
		if record["$type"] != Collection || record["subject"] != target.String() || record["createdAt"] != createdAt.Format(time.RFC3339) {
			t.Fatalf("follow record=%v", record)
		}
	}
}

func TestServicePreservesPDSFailuresAndRejectsSelfFollow(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("synthetic PDS failure")
	service := NewService(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return nil, sentinel })
	owner := syntax.DID("did:plc:synthetic-follow-owner")
	target := syntax.DID("did:plc:synthetic-follow-target")
	createdAt := time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)
	if err := service.Write(context.Background(), owner, target, "session", nil, createdAt); !errors.Is(err, sentinel) {
		t.Fatalf("write error=%v", err)
	}
	if err := service.Write(context.Background(), owner, owner, "session", nil, createdAt); !errors.Is(err, ErrSelfFollow) {
		t.Fatalf("self follow error=%v", err)
	}
}

type recordingPDS struct {
	creates      int
	puts         int
	putRkey      string
	createRecord map[string]any
	putRecord    map[string]any
}

func (*recordingPDS) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", errors.New("not implemented")
}
func (p *recordingPDS) PutRecord(_ context.Context, _ syntax.DID, _ string, rkey string, record any) error {
	p.puts++
	p.putRkey = rkey
	p.putRecord, _ = record.(map[string]any)
	return nil
}
func (p *recordingPDS) CreateRecord(_ context.Context, _ syntax.DID, _ string, record any) (syntax.ATURI, syntax.CID, error) {
	p.creates++
	p.createRecord, _ = record.(map[string]any)
	return "", "", nil
}
func (*recordingPDS) DeleteRecord(context.Context, syntax.DID, string, string) error { return nil }
func (*recordingPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}
