package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestIT007ScheduledPrepublicationPDSOmitsPrivateTargetAcrossSinks(t *testing.T) {
	for _, dependencyFailure := range []bool{true, false} {
		t.Run(map[bool]string{true: "dependency failure", false: "expected missing record"}[dependencyFailure], func(t *testing.T) {
			pool := testdb.WithMigratedSchema(t)
			store := NewStore(pool)
			now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
			if _, err := pool.Exec(context.Background(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:alice','active',1,1,'test',$1,$1,$1)`, now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid,created_at) VALUES('did:plc:alice','bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',$1)`, now); err != nil {
				t.Fatal(err)
			}
			payload, _ := EncodePayload(Payload{Kind: PostKindStandard, Text: "PRIVATE_SCHEDULE_TEXT"})
			created, err := store.Create(context.Background(), CreateParams{ID: uuid.New(), OwnerDID: "did:plc:alice", OperationID: uuid.New(), RequestHash: [32]byte{1}, ScheduledAt: now, PayloadBytes: payload, PayloadHash: sha256.Sum256(payload), PayloadVersion: 1})
			if err != nil {
				t.Fatal(err)
			}
			lifecycles, err := ownerlifecycle.NewStore(pool, newScheduledTestOwnerFencer(t, pool), func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
			pds := &recordingScheduledPDS{}
			if dependencyFailure {
				pds.getErrOnce = &pgconn.PgError{Code: "08006", Message: "PRIVATE_STORAGE_PROSE"}
			}
			factory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil })
			client, err := factory(context.Background(), created.OwnerDID, "PRIVATE_SESSION")
			if err != nil {
				t.Fatal(err)
			}
			processor, err := NewPublicationProcessor(PublicationProcessorOptions{Store: store, Sessions: stubPublicationSessionSelector{wantOwner: created.OwnerDID, sessionID: "PRIVATE_SESSION"}, NewCommands: journaledRecordingGuardedFactory(t, pool, lifecycles, client, func() time.Time { return now }), Objects: newMemoryPrivateObjectStore(), Observer: observer, Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			worker, err := NewWorker(WorkerOptions{Store: store, Processor: processor, Observer: observer, Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if err != nil || processed != 1 {
				t.Fatalf("publication result: %d %v", processed, err)
			}
			var rkey string
			if err := pool.QueryRow(context.Background(), `SELECT selected_rkey FROM pds_commands WHERE owner_did=$1`, created.OwnerDID).Scan(&rkey); err != nil {
				t.Fatal(err)
			}
			observer.Flush(time.Second)
			exported, _ := json.Marshal(transport.Events())
			all := local.String() + string(exported)
			for _, forbidden := range []string{rkey, "PRIVATE_SCHEDULE_TEXT", "PRIVATE_STORAGE_PROSE", "PRIVATE_SESSION"} {
				if forbidden != "" && strings.Contains(all, forbidden) {
					t.Errorf("private value reached diagnostics: %q", forbidden)
				}
			}
			if dependencyFailure {
				for _, positive := range []string{"08006", "pgconn.PgError", "did:plc:alice", created.ID.String()} {
					if !strings.Contains(all, positive) {
						t.Errorf("missing useful field %s", positive)
					}
				}
			}
		})
	}
}

type faultingPublicationObjects struct {
	PrivateObjectStore
	stage string
	cause error
}

func (store faultingPublicationObjects) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if store.stage == "open" {
		return nil, store.cause
	}
	reader, err := store.PrivateObjectStore.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	return faultingPublicationReader{ReadCloser: reader, stage: store.stage, cause: store.cause}, nil
}

type faultingPublicationReader struct {
	io.ReadCloser
	stage string
	cause error
}

func (reader faultingPublicationReader) Read(bytes []byte) (int, error) {
	if reader.stage == "read" {
		return 0, reader.cause
	}
	return reader.ReadCloser.Read(bytes)
}
func (reader faultingPublicationReader) Close() error {
	err := reader.ReadCloser.Close()
	if reader.stage == "close" {
		return reader.cause
	}
	return err
}

func TestIT007ScheduledMediaIOCausesSurviveAcrossSinks(t *testing.T) {
	for _, stage := range []string{"open", "read", "close"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newPublicationRecoveryFixture(t, 1, false)
			causes := map[string]error{
				"open":  &smithyhttp.ResponseError{Response: &smithyhttp.Response{Response: &http.Response{StatusCode: 503, Header: http.Header{"Secret": []string{"PRIVATE_HEADER"}}}}, Err: os.ErrNotExist},
				"read":  &net.OpError{Op: "read", Net: "tcp", Err: io.ErrUnexpectedEOF},
				"close": &os.PathError{Op: "close", Path: "PRIVATE_OBJECT_KEY", Err: io.ErrClosedPipe},
			}
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
			processor, err := NewPublicationProcessor(PublicationProcessorOptions{Store: fixture.store, Sessions: stubPublicationSessionSelector{wantOwner: fixture.owner, sessionID: "PRIVATE_SESSION"}, NewCommands: recordingGuardedFactory(fixture.pds, nil), Objects: faultingPublicationObjects{PrivateObjectStore: fixture.objects, stage: stage, cause: causes[stage]}, Observer: observer, Now: func() time.Time { return fixture.now }})
			if err != nil {
				t.Fatal(err)
			}
			worker, err := NewWorker(WorkerOptions{Store: fixture.store, Processor: processor, Observer: observer, Now: func() time.Time { return fixture.now }})
			if err != nil {
				t.Fatal(err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if err != nil || processed != 1 {
				t.Fatalf("worker result changed: %d %v", processed, err)
			}
			observer.Flush(time.Second)
			exported, _ := json.Marshal(transport.Events())
			wantType := map[string]string{"open": "*http.ResponseError", "read": "*net.OpError", "close": "*fs.PathError"}[stage]
			for _, sink := range []string{local.String(), string(exported)} {
				for _, positive := range []string{wantType, "media_upload", fixture.owner.String(), fixture.id.String()} {
					if !strings.Contains(sink, positive) {
						t.Errorf("missing selected field %s", positive)
					}
				}
				if stage == "open" && !strings.Contains(sink, "503") {
					t.Error("missing storage HTTP status")
				}
				for _, private := range []string{"PRIVATE_HEADER", "PRIVATE_OBJECT_KEY", "PRIVATE_SESSION", string(fixture.bodies[0]), fixture.media[0].ObjectKey} {
					if strings.Contains(sink, private) {
						t.Errorf("private value leaked: %q", private)
					}
				}
			}
			var status, code string
			if err := fixture.store.pool.QueryRow(context.Background(), `SELECT status,last_error_code FROM scheduled_posts WHERE id=$1`, fixture.id).Scan(&status, &code); err != nil {
				t.Fatal(err)
			}
			wantCode := "media_invalid"
			if stage == "open" {
				wantCode = "object_unavailable"
			}
			wantStatus := string(StatusNeedsAttention)
			if stage == "open" {
				wantStatus = string(StatusRetrying)
			}
			if status != wantStatus || code != wantCode {
				t.Fatalf("policy changed: %s %s", status, code)
			}
		})
	}
}
