package ops_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ops"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/testdb"
)

func TestSnapshotUsesMigratedSchemaAndReadOnlyConnection(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	var now time.Time
	if err := pool.QueryRow(ctx, `SELECT now()`).Scan(&now); err != nil {
		t.Fatal(err)
	}
	for index, item := range []struct {
		state    string
		attempts int
		due      time.Time
	}{{"pending", 6, now.Add(-2 * time.Minute)}, {"pending", 1, now.Add(time.Hour)}, {"complete", 9, now.Add(-time.Hour)}} {
		if _, err := pool.Exec(ctx, `INSERT INTO tap_repository_jobs(id,did,job_kind,state,attempts,next_attempt_at,created_at,updated_at,last_successful_at)
			VALUES($1,$2,'pds_reconcile',$3,$4,$5,$6,$6,CASE WHEN $3='complete' THEN $6::timestamptz END)`, uuid.New(), "did:plc:private-canary-"+string(rune('a'+index)), item.state, item.attempts, item.due, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES('did:plc:alice','active',1,1,'test',$1,$1,$1)`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid,created_at) VALUES('did:plc:alice','cid',$1)`, now); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"kind":"standard","text":"private-payload-canary","langs":["en"],"media":[]}`)
	for _, scheduledAt := range []time.Time{now.Add(-2 * time.Minute), now.Add(time.Hour)} {
		if _, err := scheduledposts.NewStore(pool).Create(ctx, scheduledposts.CreateParams{
			ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(), RequestHash: sha256.Sum256([]byte(scheduledAt.String())),
			ScheduledAt: scheduledAt, PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), PayloadVersion: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot,created_at,updated_at)
		VALUES($1,'account:private-case-canary','account','did:plc:private-case-canary','did:plc:private-case-canary','{"reason":"private-report-canary"}',$2,$2)`, uuid.New(), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	config := pool.Config()
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	readOnly, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	snapshot, err := ops.ReadSnapshot(ctx, readOnly, 5)
	if err != nil {
		t.Fatal(err)
	}
	queues := map[string]ops.Queue{}
	for _, queue := range snapshot.Queues {
		queues[queue.Name] = queue
	}
	if len(queues) != 13 {
		t.Fatalf("queues=%d, want all 13 operational summaries", len(queues))
	}
	repository := queues["tapRepository"]
	if repository.Pending != 2 || repository.Due != 1 || repository.Overdue != 1 || repository.RepeatedAttempts != 1 || repository.MaxAttempts != 6 || repository.States["pending"] != 2 {
		t.Fatalf("repository=%+v", repository)
	}
	if repository.OldestDueAt == nil || !repository.OldestDueAt.Equal(now.Add(-2*time.Minute)) {
		t.Fatalf("oldest due=%v", repository.OldestDueAt)
	}
	scheduled := queues["scheduledPublication"]
	if scheduled.Pending != 2 || scheduled.Due != 1 || scheduled.Overdue != 1 {
		t.Fatalf("scheduled=%+v", scheduled)
	}
	if queues["moderationCases"].Pending != 1 || queues["moderationCases"].Due != 0 {
		t.Fatalf("manual moderation=%+v", queues["moderationCases"])
	}
	if queues["tapProjection"].OldestPendingAt != nil || queues["tapProjection"].Pending != 0 {
		t.Fatalf("empty queue=%+v", queues["tapProjection"])
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{"private-canary", "private-payload-canary", "private-case-canary", "private-report-canary", "did:plc:"} {
		if strings.Contains(string(encoded), canary) {
			t.Fatalf("snapshot exposed %q", canary)
		}
	}
}

func TestSnapshotRejectsInvalidInputs(t *testing.T) {
	if _, err := ops.ReadSnapshot(context.Background(), nil, 5); err == nil {
		t.Fatal("nil database accepted")
	}
}
