package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const businessEventModerationDDL = `
CREATE TABLE craftsky_account_types (
    owner_did TEXT PRIMARY KEY,
    account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
);
CREATE TABLE craftsky_business_events (
    uri TEXT PRIMARY KEY,
    owner_did TEXT NOT NULL,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    raw_record JSONB NOT NULL,
    source_revision TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    status TEXT,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_did, rkey)
);
CREATE TABLE atproto_blocks (
    uri TEXT PRIMARY KEY,
    blocker_did TEXT NOT NULL,
    subject_did TEXT NOT NULL
);
`

func businessEventModerationMigrationDDL(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "000063_business_event_moderation.up.sql")
	ddl, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read business event moderation migration: %v", err)
	}
	return string(ddl)
}

func TestBusinessEventReportPersistsExactRecordSnapshot(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t)+businessEventModerationDDL+businessEventModerationMigrationDDL(t))
	ctx := context.Background()
	reporter := syntax.DID("did:plc:event-reporter")
	owner := syntax.DID("did:plc:event-owner")
	rkey := syntax.RecordKey("3msreportabcd")
	uri := syntax.ATURI("at://did:plc:event-owner/social.craftsky.business.event/3msreportabcd")
	cid := syntax.CID("bafyreieventsnapshot")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	_, lifecycles := newModerationStore(t, pool, func() time.Time { return asOf })
	for _, did := range []syntax.DID{reporter, owner} {
		if _, err := lifecycles.EnsureOnboardingOwner(ctx, did); err != nil {
			t.Fatalf("seed owner lifecycle %s: %v", did, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES ($1)`, did); err != nil {
			t.Fatalf("seed member %s: %v", did, err)
		}
		if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
			Owner: did, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
		}); err != nil {
			t.Fatalf("activate member lifecycle %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed owner type: %v", err)
	}
	seedModerationBusinessEvent(t, pool, owner, rkey, cid, asOf.Add(time.Hour), asOf.Add(2*time.Hour))

	reports := api.NewReportStore(pool)
	forwarder := api.NewPlaceholderReportForwarder(func() time.Time { return asOf })
	handler := api.ReportBusinessEventHandler(business.NewStore(pool), reports, forwarder, nilLogger(), func() time.Time { return asOf })
	req := httptest.NewRequest(http.MethodPost, "/v1/events/"+owner.String()+"/"+rkey.String()+"/reports", strings.NewReader(`{"reasonType":"spam"}`))
	req.SetPathValue("did", owner.String())
	req.SetPathValue("rkey", rkey.String())
	reqCtx := middleware.WithDID(req.Context(), reporter)
	reqCtx = middleware.WithDeviceID(reqCtx, "event-report-device")
	reqCtx = ownerlifecycle.WithExpectedGeneration(reqCtx, 2)
	req = req.WithContext(reqCtx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("report status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var stored struct {
		Type, DID, Collection, Rkey, URI, CID string
	}
	if err := pool.QueryRow(ctx, `
		SELECT subject_type, subject_did, subject_collection, subject_rkey, subject_uri, subject_cid_snapshot
		FROM moderation_reports
	`).Scan(&stored.Type, &stored.DID, &stored.Collection, &stored.Rkey, &stored.URI, &stored.CID); err != nil {
		t.Fatalf("read event report snapshot: %v", err)
	}
	if stored.Type != "event" || stored.DID != owner.String() ||
		stored.Collection != "social.craftsky.business.event" || stored.Rkey != rkey.String() ||
		stored.URI != uri.String() || stored.CID != cid.String() {
		t.Fatalf("event report snapshot = %+v", stored)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_reports(
			id, reporter_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, reason_type, forwarding_status, forwarding_prepared_at
		) VALUES ('invalid-event-report', $1, 'event', $2, 'social.craftsky.business.event', $3, $4, 'spam', 'prepared_not_submitted', $5)
	`, reporter, owner, rkey, uri, asOf); err == nil {
		t.Fatal("event report without CID snapshot passed the 000063 constraint")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(
			id, source_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, value, action
		) VALUES ('invalid-event-output', 'did:plc:labeler', 'event', $1, 'social.craftsky.feed.post', $2, $3, 'hide', 'apply')
	`, owner, rkey, uri); err == nil {
		t.Fatal("event moderation with the wrong collection passed the 000063 constraint")
	}
}

func TestBusinessEventModerationHideAndNegateAreURIScoped(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t)+businessEventModerationDDL+businessEventModerationMigrationDDL(t))
	ctx := context.Background()
	owner := syntax.DID("did:plc:event-owner")
	visitor := syntax.DID("did:plc:event-visitor")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	moderation, lifecycles := newModerationStore(t, pool, func() time.Time { return asOf })
	for _, did := range []syntax.DID{owner, visitor} {
		if _, err := lifecycles.EnsureOnboardingOwner(ctx, did); err != nil {
			t.Fatalf("seed owner lifecycle %s: %v", did, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES ($1)`, did); err != nil {
			t.Fatalf("seed member %s: %v", did, err)
		}
		if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
			Owner: did, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
		}); err != nil {
			t.Fatalf("activate member lifecycle %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed owner type: %v", err)
	}
	firstRkey := syntax.RecordKey("3msmoderateaa")
	secondRkey := syntax.RecordKey("3msmoderateab")
	firstURI := seedModerationBusinessEvent(t, pool, owner, firstRkey, "bafyreieventone", asOf.Add(time.Hour), asOf.Add(2*time.Hour))
	secondURI := seedModerationBusinessEvent(t, pool, owner, secondRkey, "bafyreieventtwo", asOf.Add(3*time.Hour), asOf.Add(4*time.Hour))

	decodeEvent := func(rkey syntax.RecordKey, action string) api.ModerationOutputInput {
		t.Helper()
		input, err := api.DecodeSyntheticModerationRequest(strings.NewReader(`{
			"subject":{"type":"event","did":"`+owner.String()+`","rkey":"`+rkey.String()+`"},
			"value":"hide","action":"`+action+`"
		}`), api.ModerationRequestConfig{DefaultSourceDID: "did:plc:labeler", TrustedSourceDIDs: []string{"did:plc:labeler"}})
		if err != nil {
			t.Fatalf("decode event moderation %s: %v", action, err)
		}
		return input
	}
	if _, err := moderation.InsertOutput(ctx, "event-moderation-apply-0001", decodeEvent(firstRkey, "apply")); err != nil {
		t.Fatalf("apply first event hide: %v", err)
	}
	if _, err := moderation.InsertOutput(ctx, "event-moderation-apply-0002", decodeEvent(secondRkey, "apply")); err != nil {
		t.Fatalf("apply second event hide: %v", err)
	}

	store := business.NewStore(pool)
	if _, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: firstRkey, AsOf: asOf}); !errors.Is(err, business.ErrEventNotFound) {
		t.Fatalf("visitor read hidden first event error = %v", err)
	}
	ownerView, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: owner, OwnerDID: owner, Rkey: firstRkey, AsOf: asOf})
	if err != nil || !equalStrings(ownerView.PublicSuppressionReasons, []string{"record-moderated"}) {
		t.Fatalf("hidden owner diagnostics = (%+v, %v)", ownerView, err)
	}

	if _, err := moderation.InsertOutput(ctx, "event-moderation-negate-0001", decodeEvent(firstRkey, "negate")); err != nil {
		t.Fatalf("negate first event hide: %v", err)
	}
	restored, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: firstRkey, AsOf: asOf})
	if err != nil || restored.URI != firstURI || len(restored.PublicSuppressionReasons) != 0 {
		t.Fatalf("restored first event = (%+v, %v)", restored, err)
	}
	if _, err := store.ReadEvent(ctx, business.EventReadInput{CallerDID: visitor, OwnerDID: owner, Rkey: secondRkey, AsOf: asOf}); !errors.Is(err, business.ErrEventNotFound) {
		t.Fatalf("second event %s restored by different URI negate: %v", secondURI, err)
	}
}

func seedModerationBusinessEvent(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, rkey syntax.RecordKey, cid syntax.CID, startsAt, endsAt time.Time) syntax.ATURI {
	t.Helper()
	uri := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.event/" + rkey.String())
	record, err := json.Marshal(map[string]any{
		"$type": "social.craftsky.business.event", "name": "Moderated event " + rkey.String(),
		"startsAt": startsAt.UTC().Format(time.RFC3339), "endsAt": endsAt.UTC().Format(time.RFC3339),
		"roles": []string{"vendor"}, "createdAt": "2026-08-01T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_business_events
			(uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, '3msrevision001', $6, $7, '2026-08-01T09:00:00Z')
	`, uri, owner, rkey, cid, record, startsAt, endsAt); err != nil {
		t.Fatalf("seed moderated event: %v", err)
	}
	return uri
}
