package ingestion_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

func TestBusinessRecordsIngestBeforeMembershipWithoutDependencyRetry(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:business-before-membership")

	events := []tap.Event{
		{
			ID: 201, URI: "at://did:plc:business-before-membership/social.craftsky.business.profile/self",
			DID: owner, Collection: "social.craftsky.business.profile", Rkey: "self",
			Rev: "3aaaaaaaaaaac", CID: "bafy-business-profile", Action: "create",
			Record: json.RawMessage(`{"$type":"social.craftsky.business.profile","tagline":"Before membership"}`),
		},
		{
			ID: 202, URI: "at://did:plc:business-before-membership/social.craftsky.business.event/3meventrecord",
			DID: owner, Collection: "social.craftsky.business.event", Rkey: "3meventrecord",
			Rev: "3aaaaaaaaaaad", CID: "bafy-business-event", Action: "create",
			Record: json.RawMessage(`{"$type":"social.craftsky.business.event","name":"Market","startsAt":"2026-09-10T10:00:00Z","endsAt":"2026-09-10T12:00:00Z","roles":["vendor"],"createdAt":"2026-09-01T00:00:00Z"}`),
		},
	}

	for _, event := range events {
		outcome, err := service.IngestRecord(context.Background(), event)
		if err != nil || outcome.Kind != tap.OutcomeApplied || outcome.Dependency.Kind != "" {
			t.Fatalf("ingest %s outcome=%+v error=%v", event.Collection, outcome, err)
		}
		job, err := store.ProjectionJob(context.Background(), event.URI)
		if err != nil || job.State != "pending" {
			t.Fatalf("projection job for %s = %+v, error %v", event.Collection, job, err)
		}
	}
	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: "business-before-membership", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: len(events),
	})
	if err != nil || len(claims) != len(events) {
		t.Fatalf("projection claims = %d, error %v", len(claims), err)
	}
	projected := make(map[syntax.NSID]bool, len(events))
	for _, claim := range claims {
		if err := store.Project(context.Background(), claim, func(_ context.Context, _ pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
			projected[source.Collection] = true
			return tap.Applied(), nil
		}); err != nil {
			t.Fatalf("project pre-membership business source: %v", err)
		}
	}
	for _, event := range events {
		job, err := store.ProjectionJob(context.Background(), event.URI)
		if err != nil || job.State != "complete" || !projected[event.Collection] {
			t.Fatalf("projected %s job=%+v seen=%t error=%v", event.Collection, job, projected[event.Collection], err)
		}
	}

	lifecycle, err := lifecycles.Get(context.Background(), owner)
	if err != nil || lifecycle.State != ownerlifecycle.StateDeparted {
		t.Fatalf("pre-membership lifecycle = %+v, error %v", lifecycle, err)
	}
	var memberships int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_profiles WHERE did=$1`, owner).Scan(&memberships); err != nil || memberships != 0 {
		t.Fatalf("pre-membership rows = %d, error %v", memberships, err)
	}
}
