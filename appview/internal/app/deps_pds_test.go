package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/pdseffects"
)

type deleteEffectStub struct {
	pdseffects.EffectExecutor
	err error
}

func (stub deleteEffectStub) DeleteRecord(
	context.Context,
	pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, stub.err
}

type repositoryJobEnqueuerStub struct {
	did   syntax.DID
	kind  ingestion.RepositoryJobKind
	calls int
	err   error
}

func (stub *repositoryJobEnqueuerStub) EnqueueRepositoryJob(
	_ context.Context,
	did syntax.DID,
	kind ingestion.RepositoryJobKind,
) error {
	stub.did = did
	stub.kind = kind
	stub.calls++
	return stub.err
}

func TestDeleteReconcilingExecutor(t *testing.T) {
	owner := syntax.DID("did:plc:delete-reconciliation")
	request := pdseffects.DeleteRecordRequest{
		Owner: owner, Collection: "social.craftsky.feed.like", Rkey: "like",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("queues authoritative repository reconciliation after success", func(t *testing.T) {
		jobs := &repositoryJobEnqueuerStub{}
		executor := &deleteReconcilingExecutor{
			EffectExecutor: deleteEffectStub{}, repositoryJobs: jobs, logger: logger,
		}

		if _, err := executor.DeleteRecord(t.Context(), request); err != nil {
			t.Fatal(err)
		}
		if jobs.calls != 1 || jobs.did != owner || jobs.kind != ingestion.RepositoryJobPDSReconcile {
			t.Fatalf("queued job = (%d, %q, %q), want one PDS reconciliation for %q", jobs.calls, jobs.did, jobs.kind, owner)
		}
	})

	t.Run("does not queue when the remote delete fails", func(t *testing.T) {
		jobs := &repositoryJobEnqueuerStub{}
		executor := &deleteReconcilingExecutor{
			EffectExecutor: deleteEffectStub{err: errors.New("delete failed")}, repositoryJobs: jobs, logger: logger,
		}

		if _, err := executor.DeleteRecord(t.Context(), request); err == nil {
			t.Fatal("expected delete error")
		}
		if jobs.calls != 0 {
			t.Fatalf("queue calls = %d, want 0", jobs.calls)
		}
	})

	t.Run("preserves remote success when queueing fails", func(t *testing.T) {
		jobs := &repositoryJobEnqueuerStub{err: errors.New("queue failed")}
		executor := &deleteReconcilingExecutor{
			EffectExecutor: deleteEffectStub{}, repositoryJobs: jobs, logger: logger,
		}

		if _, err := executor.DeleteRecord(t.Context(), request); err != nil {
			t.Fatalf("DeleteRecord() error = %v, want remote success preserved", err)
		}
		if jobs.calls != 1 {
			t.Fatalf("queue calls = %d, want 1", jobs.calls)
		}
	})
}
