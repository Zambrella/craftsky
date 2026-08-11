package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestStatusCapabilityIsSignedHashBackedAndOwnerJobRestricted(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000011")
	signer, err := NewStatusCapabilitySigner(
		[]byte("0123456789abcdef0123456789abcdef"),
		func() time.Time { return now },
		bytes.NewReader(bytes.Repeat([]byte{0x42}, 64)),
	)
	if err != nil {
		t.Fatal(err)
	}
	capability, err := signer.Generate(jobID, owner, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if capability.Token == "" || bytes.Contains([]byte(capability.Token), []byte("status-token")) {
		t.Fatalf("generated status capability is not opaque: %q", capability.Token)
	}
	if !bytes.Equal(capability.Hash, HashSecret(capability.Token)) {
		t.Fatal("status capability hash does not match the returned token")
	}

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, accountDeletionStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at)
		VALUES($1,$2,'active','queued',$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_status_credentials(
			token_hash,job_id,owner_did,device_id,expires_at
		) VALUES($1,$2,$3,'alice-phone',$4)
	`, capability.Hash, jobID, owner, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	for _, action := range []StatusAction{StatusRead, StatusStartReauthentication, StatusRetry} {
		grant, err := store.AuthorizeStatusCapability(ctx, signer, capability.Token, jobID, owner, "alice-phone", action)
		if err != nil || grant.JobID != jobID.String() || grant.Owner != owner {
			t.Fatalf("authorize %s = (%+v, %v)", action, grant, err)
		}
	}

	for _, test := range []struct {
		name   string
		jobID  uuid.UUID
		owner  syntax.DID
		action StatusAction
		token  string
	}{
		{name: "cross job", jobID: uuid.MustParse("10000000-0000-0000-0000-000000000012"), owner: owner, action: StatusRead, token: capability.Token},
		{name: "cross owner", jobID: jobID, owner: syntax.DID("did:plc:bob"), action: StatusRead, token: capability.Token},
		{name: "ordinary API", jobID: jobID, owner: owner, action: StatusOrdinaryAPI, token: capability.Token},
		{name: "PDS API", jobID: jobID, owner: owner, action: StatusPDSAPI, token: capability.Token},
		{name: "tampered", jobID: jobID, owner: owner, action: StatusRead, token: capability.Token + "x"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := store.AuthorizeStatusCapability(ctx, signer, test.token, test.jobID, test.owner, "alice-phone", test.action); !errors.Is(err, ErrStatusUnauthorized) {
				t.Fatalf("authorization error = %v, want ErrStatusUnauthorized", err)
			}
		})
	}

	if _, err := pool.Exec(ctx, `UPDATE account_deletion_status_credentials SET revoked_at=$2 WHERE job_id=$1`, jobID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AuthorizeStatusCapability(ctx, signer, capability.Token, jobID, owner, "alice-phone", StatusRead); !errors.Is(err, ErrStatusUnauthorized) {
		t.Fatalf("revoked authorization error = %v", err)
	}

	expiredSigner, err := NewStatusCapabilitySigner(
		[]byte("0123456789abcdef0123456789abcdef"),
		func() time.Time { return now.Add(time.Hour) },
		bytes.NewReader(bytes.Repeat([]byte{0x24}, 64)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expiredSigner.Verify(capability.Token); !errors.Is(err, ErrStatusUnauthorized) {
		t.Fatalf("expiry-boundary verification error = %v", err)
	}
}
