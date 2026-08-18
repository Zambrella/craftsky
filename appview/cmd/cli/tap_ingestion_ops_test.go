package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
)

type fakeTapIngestionOperations struct {
	projection    []ingestion.ProjectionBacklogItem
	repositories  []ingestion.RepositoryJob
	quarantine    []ingestion.QuarantinedEvent
	replayed      [32]byte
	reconcileDID  syntax.DID
	reconcileKind ingestion.RepositoryJobKind
}

func (fake *fakeTapIngestionOperations) ListProjectionBacklog(context.Context, int) ([]ingestion.ProjectionBacklogItem, error) {
	return fake.projection, nil
}

func (fake *fakeTapIngestionOperations) ListRepositoryBacklog(context.Context, int) ([]ingestion.RepositoryJob, error) {
	return fake.repositories, nil
}

func (fake *fakeTapIngestionOperations) ListQuarantine(context.Context, int) ([]ingestion.QuarantinedEvent, error) {
	return fake.quarantine, nil
}

func (fake *fakeTapIngestionOperations) RequestQuarantineReplay(_ context.Context, fingerprint [32]byte) error {
	fake.replayed = fingerprint
	return nil
}

func (fake *fakeTapIngestionOperations) EnqueueRepositoryJob(_ context.Context, did syntax.DID, kind ingestion.RepositoryJobKind) error {
	fake.reconcileDID = did
	fake.reconcileKind = kind
	return nil
}

func TestWriteTapIngestionBacklogUsesBoundedOperatorFields(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	fake := &fakeTapIngestionOperations{projection: []ingestion.ProjectionBacklogItem{{
		SourceURI:  "at://did:plc:a/social.craftsky.feed.post/one",
		Collection: "social.craftsky.feed.post", State: "blocked",
		Dependency: tap.Dependency{Kind: "member_did", Key: "did:plc:a"},
		Attempts:   2, NextAttempt: now, LastReason: tap.ReasonMissingMember, UpdatedAt: now,
	}}}
	var out bytes.Buffer
	if err := writeTapIngestionBacklog(context.Background(), fake, 100, &out); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	for _, want := range []string{`"sourceUri"`, `"dependencyKind"`, `"member_did"`, `"repository": []`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("backlog output missing %s: %s", want, out.String())
		}
	}
}

func TestTapQuarantineReplayRequiresExactFingerprint(t *testing.T) {
	fake := &fakeTapIngestionOperations{}
	if err := requestTapQuarantineReplay(context.Background(), fake, "not-a-fingerprint"); err == nil {
		t.Fatal("invalid quarantine fingerprint succeeded")
	}
	want := [32]byte{1, 2, 3}
	if err := requestTapQuarantineReplay(context.Background(), fake, hex.EncodeToString(want[:])); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	if fake.replayed != want {
		t.Fatalf("replayed fingerprint=%x want=%x", fake.replayed, want)
	}
}

func TestWriteTapQuarantineIncludesBoundedDiagnosticEvidence(t *testing.T) {
	fake := &fakeTapIngestionOperations{quarantine: []ingestion.QuarantinedEvent{{
		TapEventID: 7, EventType: "record", Reason: tap.ReasonInvalidCollection,
		Envelope: []byte(`{"id":7,"type":"record"}`), OccurrenceCount: 1,
		ReplayState: "quarantined", FirstSeenAt: time.Unix(1, 0).UTC(), LastSeenAt: time.Unix(1, 0).UTC(),
	}}}
	var out bytes.Buffer
	if err := writeTapQuarantine(context.Background(), fake, 100, &out); err != nil {
		t.Fatalf("write quarantine: %v", err)
	}
	for _, want := range []string{`"reason": "invalid_collection"`, `"evidence": {`, `"id": 7`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("quarantine output missing %s: %s", want, out.String())
		}
	}
}

func TestEnqueueTapPDSReconciliationParsesDID(t *testing.T) {
	fake := &fakeTapIngestionOperations{}
	if err := enqueueTapPDSReconciliation(context.Background(), fake, "not-a-did"); err == nil {
		t.Fatal("invalid reconciliation DID succeeded")
	}
	if err := enqueueTapPDSReconciliation(context.Background(), fake, "did:plc:alice"); err != nil {
		t.Fatalf("enqueue reconciliation: %v", err)
	}
	if fake.reconcileDID != "did:plc:alice" || fake.reconcileKind != ingestion.RepositoryJobPDSReconcile {
		t.Fatalf("reconciliation=%s/%s", fake.reconcileDID, fake.reconcileKind)
	}
}
