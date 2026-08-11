package accountdeletion

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestAppServiceCreatesFreshOAuthIntentAndAcceptsOnlyThatOwner(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	randomBytes := make([]byte, 256)
	for index := range randomBytes {
		randomBytes[index] = byte(index)
	}
	random := bytes.NewReader(randomBytes)
	signer, err := NewStatusCapabilitySigner(bytes.Repeat([]byte{3}, 32), func() time.Time { return now }, random)
	if err != nil {
		t.Fatal(err)
	}
	oauth := &databaseOAuthStarter{pool: pool, state: "deletion-state", requestURI: "urn:request:deletion"}
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: store, Signer: signer, OAuth: oauth,
		Now: func() time.Time { return now }, Random: random,
	})
	if err != nil {
		t.Fatal(err)
	}

	intent, err := service.CreateIntent(ctx, CreateIntentParams{Owner: owner, DeviceID: "alice-phone"})
	if err != nil {
		t.Fatal(err)
	}
	if intent.JobID == "" || intent.StatusToken == "" || oauth.identifier != "alice.test" {
		t.Fatalf("intent or identifier unavailable: intent=%v identifier=%q", intent, oauth.identifier)
	}
	request, deletionPurpose, err := service.RequestForState(ctx, oauth.state)
	if err != nil || !deletionPurpose || request.Owner != owner || request.JobID != intent.JobID {
		t.Fatalf("OAuth request metadata = %+v deletion=%v err=%v", request, deletionPurpose, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	completion, err := service.Complete(ctx, request, owner, "deletion-oauth")
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := service.Accept(ctx, AcceptParams{
		JobID: intent.JobID, Owner: owner, DeviceID: "alice-phone",
		StatusCapability: intent.StatusToken, ReauthProof: completion.Proof,
		ConfirmationHandle: "@alice.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != StatusActive || accepted.Phase != PhaseQueued {
		t.Fatalf("accepted = %+v", accepted)
	}
	pending, exists, err := service.PendingLogin(ctx, owner, "alice-phone")
	if err != nil || !exists || pending.JobID != intent.JobID || pending.StatusToken == "" || pending.Handle != "alice.test" {
		t.Fatalf("pending login = %+v exists=%v err=%v", pending, exists, err)
	}
}

func TestAppServiceReplacesExpiredIntentForSameOwner(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	random := bytes.NewReader(bytes.Repeat([]byte{7}, 1024))
	signer, err := NewStatusCapabilitySigner(
		bytes.Repeat([]byte{3}, 32),
		func() time.Time { return now },
		random,
	)
	if err != nil {
		t.Fatal(err)
	}
	oauth := &databaseOAuthStarter{
		pool: pool, state: "expired-state", requestURI: "urn:request:expired",
	}
	service, err := NewAppService(AppServiceOptions{
		Pool:   pool,
		Store:  NewStore(pool, func() time.Time { return now }),
		Signer: signer,
		OAuth:  oauth,
		Now:    func() time.Time { return now },
		Random: random,
	})
	if err != nil {
		t.Fatal(err)
	}

	expired, err := service.CreateIntent(ctx, CreateIntentParams{
		Owner: owner, DeviceID: "alice-phone",
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(11 * time.Minute)
	oauth.state = "replacement-state"
	oauth.requestURI = "urn:request:replacement"
	replacement, err := service.CreateIntent(ctx, CreateIntentParams{
		Owner: owner, DeviceID: "alice-phone",
	})
	if err != nil {
		t.Fatalf("replace expired intent: %v", err)
	}
	if replacement.JobID == expired.JobID {
		t.Fatalf("replacement reused expired job %s", replacement.JobID)
	}

	var oldOperations, oldStatuses, oldAuthRequests, newOperations int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM account_deletion_operations WHERE id=$1),
		(SELECT count(*) FROM account_deletion_status_credentials WHERE job_id=$1),
		(SELECT count(*) FROM oauth_auth_requests WHERE account_deletion_job_id=$1),
		(SELECT count(*) FROM account_deletion_operations WHERE id=$2 AND owner_did=$3)
	`, expired.JobID, replacement.JobID, owner).Scan(
		&oldOperations,
		&oldStatuses,
		&oldAuthRequests,
		&newOperations,
	); err != nil {
		t.Fatal(err)
	}
	if oldOperations != 0 || oldStatuses != 0 || oldAuthRequests != 0 || newOperations != 1 {
		t.Fatalf(
			"expired operation/status/auth and replacement counts = %d/%d/%d/%d",
			oldOperations,
			oldStatuses,
			oldAuthRequests,
			newOperations,
		)
	}
}

func TestAppServiceRejectsAndCleansNonAtomicOAuthRequest(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	random := bytes.NewReader(bytes.Repeat([]byte{9}, 512))
	signer, err := NewStatusCapabilitySigner(
		bytes.Repeat([]byte{3}, 32), func() time.Time { return now }, random,
	)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: NewStore(pool, func() time.Time { return now }), Signer: signer,
		OAuth: &databaseOAuthStarter{
			pool: pool, state: "ordinary-residue", requestURI: "urn:request:ordinary-residue",
			ignoreMetadata: true,
		},
		Now: func() time.Time { return now }, Random: random,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateIntent(ctx, CreateIntentParams{Owner: owner, DeviceID: "alice-phone"}); err == nil {
		t.Fatal("non-atomic ordinary OAuth request unexpectedly created a deletion intent")
	}
	var operations, authRequests int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account_deletion_operations WHERE owner_did=$1`, owner).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests WHERE data->>'request_uri'='urn:request:ordinary-residue'`).Scan(&authRequests); err != nil {
		t.Fatal(err)
	}
	if operations != 0 || authRequests != 0 {
		t.Fatalf("failed atomic OAuth left operations=%d authRequests=%d", operations, authRequests)
	}
}

func TestReplacementReauthenticationRejectsAndCleansNonAtomicOAuthRequest(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000092")
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'alice.test','alice.test',$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at,error_category
		) VALUES($2,$1,'needsAttention','removingCraftskyRecords',$3,'reauthentication')
	`, owner, jobID, now); err != nil {
		t.Fatal(err)
	}
	random := bytes.NewReader(bytes.Repeat([]byte{8}, 512))
	signer, err := NewStatusCapabilitySigner(
		bytes.Repeat([]byte{3}, 32), func() time.Time { return now }, random,
	)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAppService(AppServiceOptions{
		Pool: pool, Store: NewStore(pool, func() time.Time { return now }), Signer: signer,
		OAuth: &databaseOAuthStarter{
			pool: pool, state: "replacement-residue", requestURI: "urn:request:replacement-residue",
			ignoreMetadata: true,
		},
		Now: func() time.Time { return now }, Random: random,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartReauthentication(ctx, jobID, owner); err == nil {
		t.Fatal("non-atomic ordinary OAuth request unexpectedly started replacement reauthentication")
	}
	var operations, authRequests int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests WHERE data->>'request_uri'='urn:request:replacement-residue'`).Scan(&authRequests); err != nil {
		t.Fatal(err)
	}
	if operations != 1 || authRequests != 0 {
		t.Fatalf("failed replacement OAuth left operations=%d authRequests=%d", operations, authRequests)
	}
}

func TestTerminalAuditRemainsReadableWithSignedStatusUntilExpiry(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 15, 5, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000099")
	random := bytes.NewReader(bytes.Repeat([]byte{9}, 64))
	signer, err := NewStatusCapabilitySigner(bytes.Repeat([]byte{4}, 32), func() time.Time { return now }, random)
	if err != nil {
		t.Fatal(err)
	}
	capability, err := signer.Generate(jobID, owner, now.Add(30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_audits(job_id,did,accepted_at,terminal_at,outcome,expires_at)
		VALUES($1,$2,$3,$3,'deleted',$4)
	`, jobID, owner, now, now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	grant, err := store.AuthorizeStatusCapability(ctx, signer, capability.Token, jobID, owner, "alice-phone", StatusRead)
	if err != nil || grant.Owner != owner {
		t.Fatalf("terminal status grant = %+v err=%v", grant, err)
	}
	if _, err := store.AuthorizeStatusCapability(ctx, signer, capability.Token, jobID, owner, "alice-phone", StatusRetry); err == nil {
		t.Fatal("terminal status capability must not authorize Retry")
	}
}

type databaseOAuthStarter struct {
	pool           *pgxpool.Pool
	state          string
	requestURI     string
	identifier     string
	ignoreMetadata bool
}

func (starter *databaseOAuthStarter) StartAuthFlow(ctx context.Context, identifier string) (string, error) {
	starter.identifier = identifier
	data, err := json.Marshal(map[string]string{"request_uri": starter.requestURI})
	if err != nil {
		return "", err
	}
	purpose := "login"
	var owner any
	var jobID any
	if metadata, ok := auth.AuthRequestMetadataFromContext(ctx); ok && !starter.ignoreMetadata {
		purpose = string(metadata.Purpose)
		owner = metadata.Owner
		jobID = metadata.JobID
	}
	if _, err := starter.pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(
			state,data,purpose,account_deletion_owner_did,account_deletion_job_id
		) VALUES($1,$2,$3,$4,$5)
	`, starter.state, data, purpose, owner, jobID); err != nil {
		return "", err
	}
	return "https://auth.invalid/authorize?request_uri=" + starter.requestURI, nil
}
