package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/instagram"
)

func TestInstagramOperatorCLIRequiresExplicitOpaqueIdentifiersAndBounds(t *testing.T) {
	backend := &fakeInstagramCLIBackend{}
	for _, args := range [][]string{
		{"conflicts", "resolve", "--resolution", "keep-existing"},
		{"links", "revoke"},
		{"jobs", "inspect", "--kind", "reconciliation"},
		{"jobs", "retry", "--kind", "reconciliation"},
		{"conflicts", "list", "--limit", "501"},
		{"jobs", "list", "--kind", "webhook", "--limit", "501"},
	} {
		cmd := newInstagramCmd(fakeInstagramLoader(backend))
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("instagram %v succeeded", args)
		}
	}
	if backend.calls != 0 {
		t.Fatalf("backend calls=%d want=0 for invalid commands", backend.calls)
	}
}

func TestInstagramOperatorCLIEmitsBoundedRedactedConflictAndJobOutput(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	conflictID := uuid.MustParse("61000000-0000-0000-0000-000000000001")
	jobID := uuid.MustParse("61000000-0000-0000-0000-000000000002")
	backend := &fakeInstagramCLIBackend{
		conflicts: []instagram.OperatorConflict{{ID: conflictID, State: instagram.ConflictOpen, OpenedAt: now, ExpiresAt: now.AddDate(1, 0, 0)}},
		jobs:      []instagram.OperatorJob{{ID: jobID, Kind: instagram.OperatorJobReconciliation, Status: "failed", Attempts: 5, NextAttemptAt: now, CreatedAt: now}},
	}

	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"conflicts", "list", "--limit", "1"}, conflictID.String()},
		{[]string{"jobs", "list", "--kind", "reconciliation", "--limit", "1"}, jobID.String()},
		{[]string{"jobs", "inspect", "--kind", "reconciliation", "--job-id", jobID.String()}, jobID.String()},
	} {
		var output bytes.Buffer
		cmd := newInstagramCmd(fakeInstagramLoader(backend))
		cmd.SetArgs(test.args)
		cmd.SetOut(&output)
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("instagram %v: %v", test.args, err)
		}
		got := output.String()
		if !strings.Contains(got, test.want) {
			t.Fatalf("instagram %v output=%q want opaque ID", test.args, got)
		}
		for _, forbidden := range []string{"did:plc:private", "synthetic.private", "igsid-private", "challenge-private", "digest-private"} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("instagram %v output leaked %q: %q", test.args, forbidden, got)
			}
		}
		for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
			if len(line) > 240 {
				t.Fatalf("instagram %v emitted unbounded line of %d bytes", test.args, len(line))
			}
		}
	}
}

func TestInstagramOperatorCLIWiresExplicitAuditedMutationsAndIdempotentResults(t *testing.T) {
	conflictID := uuid.MustParse("62000000-0000-0000-0000-000000000001")
	linkID := uuid.MustParse("62000000-0000-0000-0000-000000000002")
	jobID := uuid.MustParse("62000000-0000-0000-0000-000000000003")
	backend := &fakeInstagramCLIBackend{}

	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"conflicts", "resolve", "--conflict-id", conflictID.String(), "--resolution", "revoke-existing"}, "changed=true"},
		{[]string{"links", "revoke", "--link-id", linkID.String()}, "changed=true"},
		{[]string{"jobs", "retry", "--kind", "reconciliation", "--job-id", jobID.String()}, "changed=true"},
	} {
		var output bytes.Buffer
		cmd := newInstagramCmd(fakeInstagramLoader(backend))
		cmd.SetArgs(test.args)
		cmd.SetOut(&output)
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("instagram %v: %v", test.args, err)
		}
		if !strings.Contains(output.String(), test.want) {
			t.Fatalf("instagram %v output=%q want %q", test.args, output.String(), test.want)
		}
	}
	if backend.resolution != instagram.ResolutionRevokeExisting || backend.conflictID != conflictID {
		t.Fatalf("conflict resolution=%s id=%s", backend.resolution, backend.conflictID)
	}
	if backend.linkID != linkID || backend.jobID != jobID || backend.jobKind != instagram.OperatorJobReconciliation {
		t.Fatalf("mutation wiring link=%s job=%s kind=%s", backend.linkID, backend.jobID, backend.jobKind)
	}
}

type fakeInstagramCLIBackend struct {
	calls      int
	conflicts  []instagram.OperatorConflict
	jobs       []instagram.OperatorJob
	conflictID uuid.UUID
	resolution instagram.OperatorConflictResolution
	linkID     uuid.UUID
	jobID      uuid.UUID
	jobKind    instagram.OperatorJobKind
}

func (f *fakeInstagramCLIBackend) ListOpenConflicts(context.Context, int, uuid.UUID) ([]instagram.OperatorConflict, uuid.UUID, error) {
	f.calls++
	return f.conflicts, uuid.Nil, nil
}

func (f *fakeInstagramCLIBackend) ResolveConflict(_ context.Context, id uuid.UUID, resolution instagram.OperatorConflictResolution) (instagram.OperatorConflictResult, error) {
	f.calls++
	f.conflictID, f.resolution = id, resolution
	return instagram.OperatorConflictResult{ID: id, State: resolution.State(), Changed: true}, nil
}

func (f *fakeInstagramCLIBackend) RevokeLink(_ context.Context, id uuid.UUID) (instagram.OperatorLinkResult, error) {
	f.calls++
	f.linkID = id
	return instagram.OperatorLinkResult{ID: id, State: instagram.LinkRevoked, Changed: true}, nil
}

func (f *fakeInstagramCLIBackend) ListJobs(_ context.Context, kind instagram.OperatorJobKind, _ int, _ uuid.UUID) ([]instagram.OperatorJob, uuid.UUID, error) {
	f.calls++
	f.jobKind = kind
	return f.jobs, uuid.Nil, nil
}

func (f *fakeInstagramCLIBackend) InspectJob(_ context.Context, kind instagram.OperatorJobKind, id uuid.UUID) (instagram.OperatorJob, error) {
	f.calls++
	f.jobKind, f.jobID = kind, id
	for _, job := range f.jobs {
		if job.ID == id {
			return job, nil
		}
	}
	return instagram.OperatorJob{}, errors.New("synthetic missing job")
}

func (f *fakeInstagramCLIBackend) RetryJob(_ context.Context, kind instagram.OperatorJobKind, id uuid.UUID) (instagram.OperatorJobResult, error) {
	f.calls++
	f.jobKind, f.jobID = kind, id
	return instagram.OperatorJobResult{ID: id, Kind: kind, Status: "queued", Changed: true}, nil
}

func fakeInstagramLoader(backend instagramCLIBackend) instagramCLIBackendLoader {
	return func(context.Context) (instagramCLIBackend, func(), error) {
		return backend, func() {}, nil
	}
}
