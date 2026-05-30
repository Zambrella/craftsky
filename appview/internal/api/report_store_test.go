// appview/internal/api/report_store_test.go
package api_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const reportStoreBaseDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,
    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,
    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,
    quote_uri        TEXT,
    quote_cid        TEXT,
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
`

func moderationFlowMigrationDDL(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "000014_moderation_flow.up.sql")
	ddl, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read moderation flow migration: %v", err)
	}
	return string(ddl)
}

func TestReportStore_CreateReport_PersistsPrivatePostAndProfileReports(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, reportStoreBaseDDL+moderationFlowMigrationDDL(t))
	ctx := context.Background()

	for _, did := range []string{"did:plc:alice", "did:plc:bob"} {
		if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, 'seed')`, did); err != nil {
			t.Fatalf("seed profile %s: %v", did, err)
		}
	}
	postURI := "at://did:plc:bob/social.craftsky.feed.post/3lf2abc"
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at, indexed_at)
		VALUES ($1, 'did:plc:bob', '3lf2abc', 'bafy-post-v1', 'reported post', '{}'::jsonb, $2, $2)
	`, postURI, time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	store := api.NewReportStore(pool)
	preparedAt := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	postDetails := "private post report details"
	postRow, err := store.CreateReport(ctx, api.CreateReportInput{
		ReporterDID:             "did:plc:alice",
		SubjectType:             api.ReportSubjectPost,
		SubjectDID:              "did:plc:bob",
		SubjectCollection:       ptrString("social.craftsky.feed.post"),
		SubjectRkey:             ptrString("3lf2abc"),
		SubjectURI:              ptrString(postURI),
		SubjectCIDSnapshot:      ptrString("bafy-post-v1"),
		ReasonType:              "spam",
		Details:                 &postDetails,
		DeviceID:                ptrString("device-1"),
		ForwardingStatus:        "prepared_not_submitted",
		ForwardingSchemaVersion: ptrString("atproto-create-report-v0"),
		ForwardingPreparedAt:    preparedAt,
	})
	if err != nil {
		t.Fatalf("CreateReport post: %v", err)
	}
	if postRow.ID == "" {
		t.Fatal("post report ID is empty")
	}
	assertReportRow(t, postRow, api.ReportRow{
		ReporterDID:             "did:plc:alice",
		SubjectType:             api.ReportSubjectPost,
		SubjectDID:              "did:plc:bob",
		SubjectCollection:       ptrString("social.craftsky.feed.post"),
		SubjectRkey:             ptrString("3lf2abc"),
		SubjectURI:              ptrString(postURI),
		SubjectCIDSnapshot:      ptrString("bafy-post-v1"),
		ReasonType:              "spam",
		Details:                 &postDetails,
		DeviceID:                ptrString("device-1"),
		ForwardingStatus:        "prepared_not_submitted",
		ForwardingSchemaVersion: ptrString("atproto-create-report-v0"),
		ForwardingPreparedAt:    preparedAt,
	})

	profileRow, err := store.CreateReport(ctx, api.CreateReportInput{
		ReporterDID:             "did:plc:alice",
		SubjectType:             api.ReportSubjectAccount,
		SubjectDID:              "did:plc:bob",
		SubmittedHandleSnapshot: ptrString("bob.craftsky.social"),
		ReasonType:              "impersonation",
		DeviceID:                ptrString("device-2"),
		ForwardingStatus:        "prepared_not_submitted",
		ForwardingSchemaVersion: ptrString("atproto-create-report-v0"),
		ForwardingPreparedAt:    preparedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateReport profile: %v", err)
	}
	if profileRow.ID == "" || profileRow.ID == postRow.ID {
		t.Fatalf("profile report ID = %q, post ID = %q", profileRow.ID, postRow.ID)
	}
	assertReportRow(t, profileRow, api.ReportRow{
		ReporterDID:             "did:plc:alice",
		SubjectType:             api.ReportSubjectAccount,
		SubjectDID:              "did:plc:bob",
		SubmittedHandleSnapshot: ptrString("bob.craftsky.social"),
		ReasonType:              "impersonation",
		DeviceID:                ptrString("device-2"),
		ForwardingStatus:        "prepared_not_submitted",
		ForwardingSchemaVersion: ptrString("atproto-create-report-v0"),
		ForwardingPreparedAt:    preparedAt.Add(time.Minute),
	})

	var storedCount int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM moderation_reports`).Scan(&storedCount); err != nil {
		t.Fatalf("count moderation reports: %v", err)
	}
	if storedCount != 2 {
		t.Fatalf("stored reports = %d, want 2", storedCount)
	}

	duplicate, err := store.CreateReport(ctx, api.CreateReportInput{
		ReporterDID:             "did:plc:alice",
		SubjectType:             api.ReportSubjectPost,
		SubjectDID:              "did:plc:bob",
		SubjectCollection:       ptrString("social.craftsky.feed.post"),
		SubjectRkey:             ptrString("3lf2abc"),
		SubjectURI:              ptrString(postURI),
		SubjectCIDSnapshot:      ptrString("bafy-post-v1"),
		ReasonType:              "spam",
		ForwardingStatus:        "prepared_not_submitted",
		ForwardingSchemaVersion: ptrString("atproto-create-report-v0"),
		ForwardingPreparedAt:    preparedAt.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateReport duplicate: %v", err)
	}
	if duplicate.ID == postRow.ID {
		t.Fatal("duplicate report reused the first report ID")
	}
}

func TestPostStore_ResolvePostReportTarget_CanonicalizesIndexedPost(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:bob")
	postURI := seedPost(t, pool, "did:plc:bob", "3lf2abc", "reported post", time.Now())

	store := api.NewPostStore(pool)
	target, err := store.ResolvePostReportTarget(context.Background(), "did:plc:bob", "3lf2abc")
	if err != nil {
		t.Fatalf("ResolvePostReportTarget: %v", err)
	}
	if target.DID != "did:plc:bob" || target.Rkey != "3lf2abc" || target.URI != postURI || target.CIDSnapshot != "bafycid" {
		t.Fatalf("target = %+v", target)
	}
}

func TestProfileStore_ResolveAccountReportTarget_CanonicalizesIndexedProfile(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles (did, record_cid) VALUES ('did:plc:bob', 'seed')`); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	store := api.NewProfileStore(pool)
	target, err := store.ResolveAccountReportTarget(context.Background(), "did:plc:bob")
	if err != nil {
		t.Fatalf("ResolveAccountReportTarget: %v", err)
	}
	if target.DID != "did:plc:bob" || target.SubmittedHandleSnapshot != "" {
		t.Fatalf("target = %+v", target)
	}
}

func assertReportRow(t *testing.T, got *api.ReportRow, want api.ReportRow) {
	t.Helper()
	if got.ReporterDID != want.ReporterDID {
		t.Fatalf("ReporterDID = %q, want %q", got.ReporterDID, want.ReporterDID)
	}
	if got.SubjectType != want.SubjectType {
		t.Fatalf("SubjectType = %q, want %q", got.SubjectType, want.SubjectType)
	}
	if got.SubjectDID != want.SubjectDID {
		t.Fatalf("SubjectDID = %q, want %q", got.SubjectDID, want.SubjectDID)
	}
	assertStringPtr(t, "SubjectCollection", got.SubjectCollection, want.SubjectCollection)
	assertStringPtr(t, "SubjectRkey", got.SubjectRkey, want.SubjectRkey)
	assertStringPtr(t, "SubjectURI", got.SubjectURI, want.SubjectURI)
	assertStringPtr(t, "SubjectCIDSnapshot", got.SubjectCIDSnapshot, want.SubjectCIDSnapshot)
	assertStringPtr(t, "SubmittedHandleSnapshot", got.SubmittedHandleSnapshot, want.SubmittedHandleSnapshot)
	if got.ReasonType != want.ReasonType {
		t.Fatalf("ReasonType = %q, want %q", got.ReasonType, want.ReasonType)
	}
	assertStringPtr(t, "Details", got.Details, want.Details)
	assertStringPtr(t, "DeviceID", got.DeviceID, want.DeviceID)
	if got.ForwardingStatus != want.ForwardingStatus {
		t.Fatalf("ForwardingStatus = %q, want %q", got.ForwardingStatus, want.ForwardingStatus)
	}
	assertStringPtr(t, "ForwardingSchemaVersion", got.ForwardingSchemaVersion, want.ForwardingSchemaVersion)
	if !got.ForwardingPreparedAt.Equal(want.ForwardingPreparedAt) {
		t.Fatalf("ForwardingPreparedAt = %s, want %s", got.ForwardingPreparedAt, want.ForwardingPreparedAt)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func ptrString(value string) *string {
	return &value
}

func assertStringPtr(t *testing.T, field string, got, want *string) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("%s = %v, want %v", field, got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("%s = %q, want %q", field, *got, *want)
	}
}
