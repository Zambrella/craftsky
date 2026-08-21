package ingestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const ingestionProjectionFixtureDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT PRIMARY KEY,
    record_cid TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    cid TEXT NOT NULL
);
CREATE TABLE craftsky_likes (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    subject_uri TEXT NOT NULL,
    cid TEXT NOT NULL
);
`

func TestHistoricalSourcesConvergeWhenDependenciesArriveLater(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	actor := syntax.DID("did:plc:alice")
	postURI := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/post-one")

	postOutcome, err := store.IngestRecord(ctx, tap.Event{
		ID: 1, URI: postURI, DID: actor, Collection: "social.craftsky.feed.post",
		Rkey: "post-one", Rev: "3aaaaaaaaaaa2", CID: "bafy-post", Action: "create",
		Record: json.RawMessage(`{"text":"post","createdAt":"2026-08-14T12:00:00Z"}`),
	})
	if err != nil {
		t.Fatalf("ingest post before membership: %v", err)
	}
	assertBlockedOutcome(t, postOutcome, tap.ReasonMissingMember, "member_did", actor.String())
	assertSourceAndJob(t, store, postURI, "create", "blocked", "member_did", actor.String())

	profileURI := syntax.ATURI("at://did:plc:alice/social.craftsky.actor.profile/self")
	profileOutcome, err := store.IngestRecord(ctx, tap.Event{
		ID: 2, URI: profileURI, DID: actor, Collection: "social.craftsky.actor.profile",
		Rkey: "self", Rev: "3aaaaaaaaaaa3", CID: "bafy-profile", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	})
	if err != nil || profileOutcome.Kind != tap.OutcomeApplied {
		t.Fatalf("ingest membership profile outcome=%+v err=%v", profileOutcome, err)
	}
	profileClaim := claimOneProjection(t, store, "worker-profile")
	if profileClaim.SourceURI != profileURI {
		t.Fatalf("first claim source=%s, want profile %s", profileClaim.SourceURI, profileURI)
	}
	if err := store.Project(ctx, profileClaim, func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		_, err := tx.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,$2)`, source.DID, source.CID)
		return tap.Applied(), err
	}); err != nil {
		t.Fatalf("project profile: %v", err)
	}

	postClaim := claimOneProjection(t, store, "worker-post")
	if postClaim.SourceURI != postURI {
		t.Fatalf("woken claim source=%s, want post %s", postClaim.SourceURI, postURI)
	}
	if err := store.Project(ctx, postClaim, func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		_, err := tx.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,cid) VALUES($1,$2,$3)`, source.URI, source.DID, source.CID)
		return tap.Applied(), err
	}); err != nil {
		t.Fatalf("project post: %v", err)
	}

	likeURI := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/like-one")
	likeOutcome, err := store.IngestRecord(ctx, tap.Event{
		ID: 3, URI: likeURI, DID: actor, Collection: "social.craftsky.feed.like",
		Rkey: "like-one", Rev: "3aaaaaaaaaaa4", CID: "bafy-like", Action: "create",
		Record: json.RawMessage(`{"subject":{"uri":"at://did:plc:bob/social.craftsky.feed.post/subject","cid":"bafy-subject"},"createdAt":"2026-08-14T12:01:00Z"}`),
	})
	if err != nil {
		t.Fatalf("ingest interaction before subject: %v", err)
	}
	subjectURI := "at://did:plc:bob/social.craftsky.feed.post/subject"
	if likeOutcome.Kind != tap.OutcomeApplied {
		t.Fatalf("durable interaction receipt outcome=%+v, want applied", likeOutcome)
	}
	likeClaim := claimOneProjection(t, store, "worker-like-missing-subject")
	if likeClaim.SourceURI != likeURI {
		t.Fatalf("interaction claim source=%s, want %s", likeClaim.SourceURI, likeURI)
	}
	if err := store.Project(ctx, likeClaim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: subjectURI}), nil
	}); err != nil {
		t.Fatalf("block interaction on missing subject: %v", err)
	}
	assertSourceAndJob(t, store, likeURI, "create", "blocked", "subject_uri", subjectURI)

	// A post from another repository wakes the precise subject dependency.
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,cid) VALUES($1,$2,$3)`, subjectURI, "did:plc:bob", "bafy-subject"); err != nil {
		t.Fatalf("insert subject projection: %v", err)
	}
	if err := store.WakeDependency(ctx, tap.Dependency{Kind: "subject_uri", Key: subjectURI}); err != nil {
		t.Fatalf("wake subject dependency: %v", err)
	}
	likeClaim = claimOneProjection(t, store, "worker-like")
	if likeClaim.SourceURI != likeURI {
		t.Fatalf("interaction claim source=%s, want %s", likeClaim.SourceURI, likeURI)
	}
	if err := store.Project(ctx, likeClaim, func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		_, err := tx.Exec(ctx, `INSERT INTO craftsky_likes(uri,did,subject_uri,cid) VALUES($1,$2,$3,$4)`, source.URI, source.DID, subjectURI, source.CID)
		return tap.Applied(), err
	}); err != nil {
		t.Fatalf("project interaction: %v", err)
	}

	var likes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_likes WHERE uri=$1`, likeURI).Scan(&likes); err != nil || likes != 1 {
		t.Fatalf("projected likes=%d err=%v", likes, err)
	}
}

func TestConflictingSameTapIDCommitsUncertaintyAndRepositoryReconciliation(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	actor := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')`, actor); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/post-one")
	first := tap.Event{
		ID: 10, URI: uri, DID: actor, Collection: "social.craftsky.feed.post",
		Rkey: "post-one", Rev: "3aaaaaaaaaaa5", CID: "bafy-first", Action: "create",
		Record: json.RawMessage(`{"text":"first","createdAt":"2026-08-14T12:00:00Z"}`),
	}
	if _, err := store.IngestRecord(ctx, first); err != nil {
		t.Fatalf("ingest first source: %v", err)
	}
	conflict := first
	conflict.Record = json.RawMessage(`{"text":"conflict","createdAt":"2026-08-14T12:00:00Z"}`)
	outcome, err := store.IngestRecord(ctx, conflict)
	if err != nil {
		t.Fatalf("ingest ordering conflict: %v", err)
	}
	assertBlockedOutcome(t, outcome, tap.ReasonSourceOrderUncertain, "repository_did", actor.String())

	source, err := store.Source(ctx, uri)
	if err != nil {
		t.Fatalf("read uncertain source: %v", err)
	}
	if source.OrderingStatus != "uncertain" || source.CID != "bafy-first" {
		t.Fatalf("uncertain source=%+v; conflict must not guess a winner", source)
	}
	job, err := store.ProjectionJob(ctx, uri)
	if err != nil {
		t.Fatalf("read blocked projection: %v", err)
	}
	if job.State != "blocked" || job.Dependency.Kind != "repository_did" || job.Dependency.Key != actor.String() {
		t.Fatalf("projection job=%+v", job)
	}
	var repositories int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM tap_repository_jobs
		WHERE did=$1 AND job_kind='pds_reconcile' AND state='pending'
	`, actor).Scan(&repositories); err != nil || repositories != 1 {
		t.Fatalf("repository reconciliation jobs=%d err=%v", repositories, err)
	}
}

func TestConflictingSameTapIDDoesNotAckWithoutItsProjectionJob(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	actor := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')`, actor); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/missing-job")
	first := tap.Event{
		ID: 11, URI: uri, DID: actor, Collection: "social.craftsky.feed.post",
		Rkey: "missing-job", Rev: "3aaaaaaaaaaa6", CID: "bafy-first", Action: "create",
		Record: json.RawMessage(`{"text":"first","createdAt":"2026-08-14T12:00:00Z"}`),
	}
	if _, err := store.IngestRecord(ctx, first); err != nil {
		t.Fatalf("ingest first source: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tap_projection_jobs WHERE source_uri=$1`, uri); err != nil {
		t.Fatalf("remove projection job fixture: %v", err)
	}
	conflict := first
	conflict.CID = "bafy-conflict"
	conflict.Record = json.RawMessage(`{"text":"conflict","createdAt":"2026-08-14T12:00:00Z"}`)
	outcome, err := store.IngestRecord(ctx, conflict)
	if err == nil || outcome.Kind != tap.OutcomeRetryable {
		t.Fatalf("conflict without job outcome=%+v err=%v, want retryable", outcome, err)
	}
	source, readErr := store.Source(ctx, uri)
	if readErr != nil || source.OrderingStatus != "authoritative" {
		t.Fatalf("source after rejected conflict=%+v err=%v", source, readErr)
	}
	assertSourceReceiptCount(t, pool, uri, 1)
}

func TestPermanentProjectionDefectIsQuarantinedInTheJobTransaction(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	uri := syntax.ATURI("at://did:plc:invalid/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 50, URI: uri, DID: "did:plc:invalid", Collection: "social.craftsky.actor.profile",
		Rkey: "self", Rev: "3aaaaaaaaaaa7", CID: "bafy-invalid", Action: "create",
		Record: json.RawMessage(`{"crafts":"not-an-array"}`),
	}); err != nil {
		t.Fatalf("ingest syntactically valid source: %v", err)
	}
	claim := claimOneProjection(t, store, "worker-invalid")
	if err := store.Project(ctx, claim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("classify permanent projection defect: %v", err)
	}
	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "permanent_denied" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	var quarantined int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM tap_quarantined_events
		WHERE tap_event_id=50 AND reason_code='malformed_record'
	`).Scan(&quarantined); err != nil || quarantined != 1 {
		t.Fatalf("quarantine rows=%d err=%v", quarantined, err)
	}
}

func TestMaximumSizedSourcePermanentProjectionQuarantineCommits(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	const sourceLimit = 1 << 20
	prefix := []byte(`{"crafts":"not-an-array","padding":"`)
	suffix := []byte(`"}`)
	record := make([]byte, 0, sourceLimit)
	record = append(record, prefix...)
	record = append(record, bytes.Repeat([]byte("x"), sourceLimit-len(prefix)-len(suffix))...)
	record = append(record, suffix...)
	if len(record) != sourceLimit || !json.Valid(record) {
		t.Fatalf("maximum source record bytes=%d valid=%t", len(record), json.Valid(record))
	}

	uri := syntax.ATURI("at://did:plc:max-quarantine/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 53, URI: uri, DID: "did:plc:max-quarantine", Collection: "social.craftsky.actor.profile",
		Rkey: "self", Rev: "3aaaaaaaaaaab", CID: "bafy-max-quarantine", Action: "create",
		Record: record,
	}); err != nil {
		t.Fatalf("ingest maximum syntactically valid source: %v", err)
	}
	claim := claimOneProjection(t, store, "worker-max-quarantine")
	if err := store.Project(ctx, claim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("commit maximum permanent projection quarantine: %v", err)
	}

	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "permanent_denied" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	var diagnosticEnvelope, replayEnvelope []byte
	if err := pool.QueryRow(ctx, `
		SELECT envelope,replay_envelope
		FROM tap_quarantined_events
		WHERE tap_event_id=53 AND reason_code='malformed_record'
	`).Scan(&diagnosticEnvelope, &replayEnvelope); err != nil {
		t.Fatalf("read maximum source quarantine: %v", err)
	}
	if len(diagnosticEnvelope) > 64<<10 {
		t.Fatalf("diagnostic envelope bytes=%d, want at most 65536", len(diagnosticEnvelope))
	}
	if len(replayEnvelope) == 0 || len(replayEnvelope) > tap.MaxFrameBytes {
		t.Fatalf("replay envelope bytes=%d, want 1..%d", len(replayEnvelope), tap.MaxFrameBytes)
	}
	var replay struct {
		Source struct {
			URI               string   `json:"URI"`
			SourceEventID     uint64   `json:"SourceEventID"`
			SourceFingerprint [32]byte `json:"SourceFingerprint"`
		} `json:"source"`
	}
	if err := json.Unmarshal(replayEnvelope, &replay); err != nil {
		t.Fatalf("decode maximum source replay envelope: %v", err)
	}
	if replay.Source.URI != uri.String() || replay.Source.SourceEventID != 53 ||
		replay.Source.SourceFingerprint == ([32]byte{}) {
		t.Fatalf("unexpected maximum source replay envelope identity")
	}
	var replayFields map[string]json.RawMessage
	if err := json.Unmarshal(replayEnvelope, &replayFields); err != nil {
		t.Fatalf("decode maximum source replay fields: %v", err)
	}
	if _, present := replayFields["record"]; present {
		t.Fatal("maximum source replay envelope retained durable source record")
	}
}

func TestJSONBExpandedSourcePermanentProjectionQuarantineCommits(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()

	// PostgreSQL JSONB normalizes exponent notation into a full decimal. This
	// tiny valid wire record expands beyond Tap's 2 MiB frame limit after the
	// durable source row is read back, so projection replay must never embed it.
	var record strings.Builder
	record.WriteString(`{"crafts":"not-an-array"`)
	for index := range 17 {
		record.WriteString(`,"n`)
		record.WriteString(string(rune('a' + index)))
		record.WriteString(`":1e131071`)
	}
	record.WriteByte('}')
	rawRecord := []byte(record.String())
	if len(rawRecord) >= 1<<20 || !json.Valid(rawRecord) {
		t.Fatalf("expanding source record bytes=%d valid=%t", len(rawRecord), json.Valid(rawRecord))
	}

	uri := syntax.ATURI("at://did:plc:jsonb-expanded-quarantine/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 54, URI: uri, DID: "did:plc:jsonb-expanded-quarantine", Collection: "social.craftsky.actor.profile",
		Rkey: "self", Rev: "3aaaaaaaaaaac", CID: "bafy-jsonb-expanded-quarantine", Action: "create",
		Record: rawRecord,
	}); err != nil {
		t.Fatalf("ingest compact source: %v", err)
	}
	claim := claimOneProjection(t, store, "worker-jsonb-expanded-quarantine")
	if err := store.Project(ctx, claim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("commit JSONB-expanded permanent projection quarantine: %v", err)
	}

	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "permanent_denied" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	var replayEnvelope []byte
	if err := pool.QueryRow(ctx, `
		SELECT replay_envelope
		FROM tap_quarantined_events
		WHERE tap_event_id=54 AND reason_code='malformed_record'
	`).Scan(&replayEnvelope); err != nil {
		t.Fatalf("read JSONB-expanded source quarantine: %v", err)
	}
	var replay map[string]json.RawMessage
	if err := json.Unmarshal(replayEnvelope, &replay); err != nil {
		t.Fatalf("decode JSONB-expanded source replay envelope: %v", err)
	}
	if _, present := replay["record"]; present {
		t.Fatal("projection replay envelope retained JSONB source record")
	}
}

func TestEscapedNULSourceReachesPermanentProjectionClassification(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	rawRecord := json.RawMessage(`{"crafts":"not-an-array","unknown":"\u0000"}`)
	if !json.Valid(rawRecord) {
		t.Fatal("escaped-NUL source fixture is not valid JSON")
	}

	uri := syntax.ATURI("at://did:plc:escaped-nul-quarantine/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 55, URI: uri, DID: "did:plc:escaped-nul-quarantine", Collection: "social.craftsky.actor.profile",
		Rkey: "self", Rev: "3aaaaaaaaaaad", CID: "bafy-escaped-nul-quarantine", Action: "create",
		Record: rawRecord,
	}); err != nil {
		t.Fatalf("ingest syntactically valid escaped-NUL source: %v", err)
	}
	claim := claimOneProjection(t, store, "worker-escaped-nul-quarantine")
	if err := store.Project(ctx, claim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("classify escaped-NUL projection defect: %v", err)
	}
	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "permanent_denied" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}

func TestMalformedInteractionIsPersistedBeforeProjectionClassification(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	actor := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')`, actor); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/malformed")
	outcome, err := store.IngestRecord(ctx, tap.Event{
		ID: 51, URI: uri, DID: actor, Collection: "social.craftsky.feed.like",
		Rkey: "malformed", Rev: "3aaaaaaaaaaae", CID: "bafy-invalid", Action: "create",
		Record: json.RawMessage(`{"createdAt":"2026-08-14T12:00:00Z"}`),
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("ingest syntactically valid interaction outcome=%+v err=%v", outcome, err)
	}
	assertSourceAndJob(t, store, uri, "create", "pending", "", "")

	claim := claimOneProjection(t, store, "worker-invalid-interaction")
	if err := store.Project(ctx, claim, func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}); err != nil {
		t.Fatalf("classify malformed interaction: %v", err)
	}
	var quarantined int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM tap_quarantined_events
		WHERE tap_event_id=51 AND reason_code='malformed_record'
	`).Scan(&quarantined); err != nil || quarantined != 1 {
		t.Fatalf("quarantine rows=%d err=%v", quarantined, err)
	}
}

func TestSourceLimitUsesWireBytesRatherThanExpandedJSONBText(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	owner := syntax.DID("did:plc:json-size")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')`, owner); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	// PostgreSQL JSONB renders a space after every comma. The durable limit is
	// the original Tap record bytes, not that expanded storage rendering.
	record := append([]byte(`{"items":[`), bytes.Repeat([]byte("0,"), 500_000)...)
	record[len(record)-1] = ']'
	record = append(record, '}')
	if len(record) > 1<<20 || !json.Valid(record) {
		t.Fatalf("invalid bounded test record bytes=%d", len(record))
	}
	outcome, err := store.IngestRecord(ctx, tap.Event{
		ID: 52, URI: "at://did:plc:json-size/social.craftsky.feed.post/large-json",
		DID: owner, Collection: "social.craftsky.feed.post", Rkey: "large-json",
		Rev: "3aaaaaaaaaaaf", CID: "bafy-large-json", Action: "create", Record: record,
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("ingest wire-bounded expanded JSON outcome=%+v err=%v", outcome, err)
	}
}

func applyTapDurabilityMigration(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) {
	t.Helper()
	for _, path := range []string{
		"../../migrations/000045_tap_ingestion_durability.up.sql",
		"../../migrations/000051_tap_quarantine_replay_payload.up.sql",
		"../../migrations/000058_tap_projection_generation_column.up.sql",
	} {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read Tap durability migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply Tap durability migration %s: %v", path, err)
		}
	}
}

func claimOneProjection(t *testing.T, store *ingestion.Store, worker string) ingestion.ProjectionClaim {
	t.Helper()
	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: worker, LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil {
		t.Fatalf("claim projection: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims=%+v, want one", claims)
	}
	return claims[0]
}

func assertBlockedOutcome(t *testing.T, outcome tap.Outcome, reason tap.ReasonCode, kind, key string) {
	t.Helper()
	if outcome.Kind != tap.OutcomeBlocked || outcome.Reason != reason ||
		outcome.Dependency.Kind != kind || outcome.Dependency.Key != key {
		t.Fatalf("outcome=%+v, want blocked %s %s/%s", outcome, reason, kind, key)
	}
}

func assertSourceAndJob(t *testing.T, store *ingestion.Store, uri syntax.ATURI, action, state, dependencyKind, dependencyKey string) {
	t.Helper()
	source, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatalf("read source %s: %v", uri, err)
	}
	if source.Action != action {
		t.Fatalf("source action=%s, want %s", source.Action, action)
	}
	job, err := store.ProjectionJob(context.Background(), uri)
	if err != nil {
		t.Fatalf("read projection job %s: %v", uri, err)
	}
	if job.State != state || job.Dependency.Kind != dependencyKind || job.Dependency.Key != dependencyKey {
		t.Fatalf("job=%+v", job)
	}
}
