// Package ingestion persists Tap source state before asynchronous projection.
package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
)

var (
	ErrProjectionLeaseLost = errors.New("tap projection lease lost")
)

const maxDurableRecordBytes = 1 << 20

type Store struct {
	pool           *pgxpool.Pool
	now            func() time.Time
	lifecycleAware bool
}

func NewStore(pool *pgxpool.Pool, now func() time.Time) (*Store, error) {
	if pool == nil {
		return nil, errors.New("ingestion store requires a database pool")
	}
	if now == nil {
		now = time.Now
	}
	return &Store{pool: pool, now: now}, nil
}

type SourceRecord struct {
	URI                   syntax.ATURI
	DID                   syntax.DID
	Collection            syntax.NSID
	Rkey                  syntax.RecordKey
	SourceEventID         uint64
	SourceFingerprint     [32]byte
	Revision              string
	CID                   syntax.CID
	Action                string
	Record                json.RawMessage
	RecordBytes           int
	Live                  bool
	OrderingStatus        string
	ProjectionDisposition string
	OwnerGeneration       *int64
	EffectOperationID     string
	ProjectionVersion     int
	ObservedAt            time.Time
	UpdatedAt             time.Time
}

type ProjectionJob struct {
	ID             int64
	SourceURI      syntax.ATURI
	ProjectionKind string
	SourceEventID  uint64
	State          string
	Dependency     tap.Dependency
	Attempts       int
	LeaseOwner     string
	LeaseToken     uuid.UUID
	LeaseExpiresAt time.Time
}

type ProjectionClaimRequest struct {
	Worker        string
	LeaseToken    uuid.UUID
	LeaseDuration time.Duration
	Limit         int
}

type ProjectionClaim struct {
	ProjectionJob
}

type Projector func(context.Context, pgx.Tx, SourceRecord) (tap.Outcome, error)

func (store *Store) IngestRecord(ctx context.Context, event tap.Event) (tap.Outcome, error) {
	if err := validateRecordEvent(event); err != nil {
		return tap.Retryable(tap.ReasonMalformedRecord), err
	}
	fingerprint, err := recordFingerprint(event)
	if err != nil {
		return tap.Retryable(tap.ReasonMalformedRecord), err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	var outcome tap.Outcome
	err = pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var ingestErr error
		outcome, ingestErr = store.ingestRecordTx(ctx, tx, event, fingerprint, now, nil)
		return ingestErr
	})
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return outcome, nil
}

type sourceAuthority struct {
	Lifecycle         ownerlifecycle.Lifecycle
	Authoritative     bool
	LockedOperationID string
}

func (store *Store) ingestRecordTx(
	ctx context.Context,
	tx pgx.Tx,
	event tap.Event,
	fingerprint [32]byte,
	now time.Time,
	authority *sourceAuthority,
) (tap.Outcome, error) {
	var currentID int64
	var currentFingerprint []byte
	readErr := tx.QueryRow(ctx, `
		SELECT source_event_id,source_fingerprint
		FROM tap_source_records
		WHERE uri=$1
		FOR UPDATE
	`, event.URI).Scan(&currentID, &currentFingerprint)
	switch {
	case readErr == nil && currentID > int64(event.ID):
		outcome := tap.Applied()
		return outcome, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, tap.ReasonStaleSource, now)
	case readErr == nil && currentID == int64(event.ID) && bytes.Equal(currentFingerprint, fingerprint[:]):
		outcome, err := readCurrentJobOutcome(ctx, tx, event.URI)
		if err != nil {
			return tap.Outcome{}, err
		}
		return outcome, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, outcome.Reason, now)
	case readErr == nil && currentID == int64(event.ID):
		return markSourceConflictTx(ctx, tx, event, fingerprint, now)
	case readErr != nil && !errors.Is(readErr, pgx.ErrNoRows):
		return tap.Outcome{}, fmt.Errorf("read current Tap source: %w", readErr)
	}

	outcome, err := store.initialProjectionOutcome(ctx, tx, event)
	if err != nil {
		return tap.Outcome{}, err
	}
	disposition := "eligible"
	orderingStatus := "authoritative"
	var ownerGeneration any
	var effectOperationID any
	if authority != nil {
		ownerGeneration = authority.Lifecycle.Generation
		switch authority.Lifecycle.State {
		case "terminal":
			disposition = "denied_terminal"
			outcome = tap.PermanentInvalid(tap.ReasonOwnerTerminal)
		case "departed", "deletion_pending", "deleting":
			if event.Action != "delete" {
				disposition = "blocked_departed"
				outcome = tap.Blocked(tap.ReasonOwnerDeparted, tap.Dependency{Kind: "member_did", Key: event.DID.String()})
			}
		}
		if event.Action != "delete" {
			recordContentFingerprint, err := pdseffects.RecordContentFingerprint(
				event.DID, event.Collection, event.Rkey, event.Record,
			)
			if err != nil {
				return tap.Outcome{}, err
			}
			resolution, err := ownerlifecycle.ResolvePDSRecordSourceTx(
				ctx,
				tx,
				authority.Lifecycle,
				ownerlifecycle.PDSRecordSourceObservation{
					Owner: event.DID, URI: event.URI, CID: event.CID,
					RecordFingerprint: recordContentFingerprint,
					LockedOperationID: authority.LockedOperationID,
					Authoritative:     authority.Authoritative,
				},
				now,
			)
			if err != nil {
				return tap.Outcome{}, err
			}
			switch resolution.Match {
			case ownerlifecycle.EffectSourceAmbiguous:
				orderingStatus = "uncertain"
				disposition = "pending"
				outcome = tap.Blocked(
					tap.ReasonSourceOrderUncertain,
					tap.Dependency{Kind: "repository_did", Key: event.DID.String()},
				)
			case ownerlifecycle.EffectSourceMatched:
				effectOperationID = resolution.Attempt.OperationID
				ownerGeneration = resolution.Attempt.OwnerGeneration
				switch resolution.Attempt.ProjectionDisposition {
				case ownerlifecycle.ProjectionEligibleCurrent:
					disposition = "eligible"
				case ownerlifecycle.ProjectionHiddenNonActive:
					if resolution.NeedsAuthoritative {
						orderingStatus = "uncertain"
						disposition = "pending"
						outcome = tap.Blocked(
							tap.ReasonSourceOrderUncertain,
							tap.Dependency{Kind: "repository_did", Key: event.DID.String()},
						)
					} else {
						disposition = "blocked_departed"
						outcome = tap.Blocked(
							tap.ReasonOwnerDeparted,
							tap.Dependency{Kind: "member_did", Key: event.DID.String()},
						)
					}
				case ownerlifecycle.ProjectionDeniedTerminal:
					disposition = "denied_terminal"
					outcome = tap.PermanentInvalid(tap.ReasonOwnerTerminal)
				case ownerlifecycle.ProjectionNotApplicable:
					disposition = "not_accepted"
					outcome = tap.PermanentInvalid(tap.ReasonStaleSource)
				default:
					orderingStatus = "uncertain"
					disposition = "pending"
					outcome = tap.Blocked(
						tap.ReasonSourceOrderUncertain,
						tap.Dependency{Kind: "repository_did", Key: event.DID.String()},
					)
				}
			}
		}
	}
	if outcome.Kind == tap.OutcomeBlocked && disposition == "eligible" {
		disposition = "pending"
	}
	record := any(nil)
	cid := any(nil)
	recordBytes := 0
	if event.Action != "delete" {
		record = event.Record
		cid = event.CID
		recordBytes = len(event.Record)
	} else if event.CID != "" {
		cid = event.CID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,
			projection_disposition,owner_generation,effect_operation_id,observed_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17)
		ON CONFLICT(uri) DO UPDATE SET
			did=EXCLUDED.did,collection=EXCLUDED.collection,rkey=EXCLUDED.rkey,
			source_event_id=EXCLUDED.source_event_id,
			source_fingerprint=EXCLUDED.source_fingerprint,revision=EXCLUDED.revision,
			cid=EXCLUDED.cid,action=EXCLUDED.action,record=EXCLUDED.record,
			record_bytes=EXCLUDED.record_bytes,
			live=EXCLUDED.live,ordering_status=EXCLUDED.ordering_status,
			projection_disposition=EXCLUDED.projection_disposition,
			owner_generation=EXCLUDED.owner_generation,
			effect_operation_id=EXCLUDED.effect_operation_id,
			observed_at=EXCLUDED.observed_at,updated_at=EXCLUDED.updated_at
	`, event.URI, event.DID, event.Collection, event.Rkey, event.ID, fingerprint[:],
		event.Rev, cid, event.Action, record, recordBytes, event.Live, orderingStatus,
		disposition, ownerGeneration, effectOperationID, now); err != nil {
		return tap.Outcome{}, fmt.Errorf("upsert Tap source: %w", err)
	}
	state := "pending"
	var dependencyKind, dependencyKey any
	if outcome.Kind == tap.OutcomeBlocked {
		state = "blocked"
		dependencyKind = outcome.Dependency.Kind
		dependencyKey = outcome.Dependency.Key
	} else if outcome.Kind == tap.OutcomePermanentInvalid {
		state = "permanent_denied"
	}
	var completedAt any
	if state == "permanent_denied" {
		completedAt = now
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,
			dependency_kind,dependency_key,next_attempt_at,last_reason_code,
			completed_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$7,$7)
		ON CONFLICT(source_uri,projection_kind) DO UPDATE SET
			source_event_id=EXCLUDED.source_event_id,state=EXCLUDED.state,
			dependency_kind=EXCLUDED.dependency_kind,
			dependency_key=EXCLUDED.dependency_key,attempts=0,
			next_attempt_at=EXCLUDED.next_attempt_at,
			lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			last_reason_code=EXCLUDED.last_reason_code,
			completed_at=EXCLUDED.completed_at,updated_at=EXCLUDED.updated_at
	`, event.URI, projectionKind(event.Collection), event.ID, state,
		dependencyKind, dependencyKey, now, nullableString(string(outcome.Reason)), completedAt); err != nil {
		return tap.Outcome{}, fmt.Errorf("upsert Tap projection job: %w", err)
	}
	if orderingStatus == "uncertain" {
		if err := enqueueRepositoryJob(ctx, tx, event.DID, string(RepositoryJobPDSReconcile), now); err != nil {
			return tap.Outcome{}, err
		}
	}
	return outcome, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, outcome.Reason, now)
}

func (store *Store) prepareEffectSourcesForRejoinTx(
	ctx context.Context,
	tx pgx.Tx,
	lifecycle ownerlifecycle.Lifecycle,
	now time.Time,
) error {
	if lifecycle.Owner == "" || lifecycle.State != ownerlifecycle.StateActive || lifecycle.Generation <= 0 {
		return errors.New("invalid effect-source rejoin authority")
	}
	rows, err := tx.Query(ctx, `
		UPDATE tap_source_records AS source
		SET ordering_status='uncertain',projection_disposition='pending',updated_at=$2
		FROM owner_effect_attempts AS attempt
		WHERE source.did=$1
		  AND source.effect_operation_id=attempt.operation_id
		  AND source.owner_generation=attempt.owner_generation
		  AND source.action<>'delete'
		  AND attempt.effect_kind='pds_record'
		  AND attempt.effect_action='put_record'
		  AND attempt.projection_disposition='hidden_non_active'
		RETURNING source.uri,source.source_event_id
	`, lifecycle.Owner, now)
	if err != nil {
		return fmt.Errorf("prepare effect sources for rejoin: %w", err)
	}
	defer rows.Close()
	type sourceVersion struct {
		uri     syntax.ATURI
		eventID int64
	}
	versions := make([]sourceVersion, 0)
	for rows.Next() {
		var version sourceVersion
		if err := rows.Scan(&version.uri, &version.eventID); err != nil {
			return fmt.Errorf("scan effect source rejoin: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate effect source rejoin: %w", err)
	}
	for _, version := range versions {
		result, err := tx.Exec(ctx, `
			UPDATE tap_projection_jobs
			SET state='blocked',dependency_kind='repository_did',dependency_key=$3,
			    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			    last_reason_code=$4,completed_at=NULL,updated_at=$5
			WHERE source_uri=$1 AND source_event_id=$2
		`, version.uri, version.eventID, lifecycle.Owner, tap.ReasonSourceOrderUncertain, now)
		if err != nil {
			return fmt.Errorf("block effect source pending rejoin read: %w", err)
		}
		if result.RowsAffected() != 1 {
			return ErrProjectionLeaseLost
		}
	}
	if len(versions) > 0 {
		if err := enqueueRepositoryJob(
			ctx, tx, lifecycle.Owner, string(RepositoryJobPDSReconcile), now,
		); err != nil {
			return err
		}
	}
	return nil
}

func markSourceConflictTx(ctx context.Context, tx pgx.Tx, event tap.Event, fingerprint [32]byte, now time.Time) (tap.Outcome, error) {
	outcome := tap.Blocked(
		tap.ReasonSourceOrderUncertain,
		tap.Dependency{Kind: "repository_did", Key: event.DID.String()},
	)
	sourceResult, err := tx.Exec(ctx, `
		UPDATE tap_source_records
		SET ordering_status='uncertain',projection_disposition='pending',updated_at=$2
		WHERE uri=$1
	`, event.URI, now)
	if err != nil {
		return tap.Outcome{}, fmt.Errorf("mark uncertain Tap source: %w", err)
	}
	if sourceResult.RowsAffected() != 1 {
		return tap.Outcome{}, errors.New("mark uncertain Tap source: source row missing")
	}
	jobResult, err := tx.Exec(ctx, `
		UPDATE tap_projection_jobs
		SET state='blocked',dependency_kind='repository_did',dependency_key=$2,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=$3,completed_at=NULL,updated_at=$4
		WHERE source_uri=$1
	`, event.URI, event.DID, tap.ReasonSourceOrderUncertain, now)
	if err != nil {
		return tap.Outcome{}, fmt.Errorf("block uncertain Tap projection: %w", err)
	}
	if jobResult.RowsAffected() != 1 {
		return tap.Outcome{}, errors.New("block uncertain Tap projection: projection job missing")
	}
	if err := enqueueRepositoryJob(ctx, tx, event.DID, string(RepositoryJobPDSReconcile), now); err != nil {
		return tap.Outcome{}, err
	}
	return outcome, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, outcome.Reason, now)
}

func validateRecordEvent(event tap.Event) error {
	if event.ID == 0 || event.URI == "" || event.DID == "" || event.Collection == "" ||
		event.Rkey == "" || strings.TrimSpace(event.Rev) == "" {
		return errors.New("invalid Tap record envelope")
	}
	switch event.Action {
	case "create", "update":
		if event.CID == "" || len(event.Record) == 0 || len(event.Record) > maxDurableRecordBytes || !json.Valid(event.Record) {
			return errors.New("invalid Tap record body")
		}
	case "delete":
	default:
		return errors.New("unsupported Tap record action")
	}
	return nil
}

func recordFingerprint(event tap.Event) ([32]byte, error) {
	canonical := struct {
		ID       uint64          `json:"id"`
		URI      syntax.ATURI    `json:"uri"`
		CID      syntax.CID      `json:"cid,omitempty"`
		Revision string          `json:"revision"`
		Action   string          `json:"action"`
		Record   json.RawMessage `json:"record,omitempty"`
	}{event.ID, event.URI, event.CID, event.Rev, event.Action, event.Record}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return [32]byte{}, fmt.Errorf("fingerprint Tap record: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func (store *Store) initialProjectionOutcome(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	switch event.Collection {
	case "social.craftsky.actor.profile":
		return tap.Applied(), nil
	case "social.craftsky.feed.post", "social.craftsky.feed.like", "social.craftsky.feed.repost",
		"app.bsky.actor.profile", "app.bsky.graph.follow", "app.bsky.graph.block":
	default:
		return tap.Outcome{}, fmt.Errorf("unsupported Tap collection %s", event.Collection)
	}
	if event.Action == "delete" {
		return tap.Applied(), nil
	}
	var member bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)`, event.DID).Scan(&member); err != nil {
		return tap.Outcome{}, fmt.Errorf("check projection membership: %w", err)
	}
	if !member {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: event.DID.String()}), nil
	}
	// Semantic record validation and subject-dependency discovery belong to the
	// transactional projector. The ACK boundary retains every syntactically
	// valid source first, so a malformed supported record can be quarantined and
	// an interaction whose subject arrives later can become explicitly blocked
	// without losing the original event.
	return tap.Applied(), nil
}

func projectionKind(collection syntax.NSID) string {
	return strings.ReplaceAll(collection.String(), ".", "_")
}

func insertReceipt(ctx context.Context, tx pgx.Tx, fingerprint [32]byte, eventID uint64, eventType string, outcome tap.Outcome, sourceURI syntax.ATURI, reason tap.ReasonCode, now time.Time) error {
	_, err := insertReceiptIfNew(ctx, tx, fingerprint, eventID, eventType, outcome, sourceURI, reason, now)
	return err
}

func insertReceiptIfNew(ctx context.Context, tx pgx.Tx, fingerprint [32]byte, eventID uint64, eventType string, outcome tap.Outcome, sourceURI syntax.ATURI, reason tap.ReasonCode, now time.Time) (bool, error) {
	if reason == "" {
		reason = tap.ReasonNone
	}
	result, err := tx.Exec(ctx, `
		INSERT INTO tap_ingestion_receipts(
			event_fingerprint,tap_event_id,event_type,outcome,source_uri,reason_code,received_at
		) VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(event_fingerprint) DO NOTHING
	`, fingerprint[:], eventID, eventType, outcome.Kind, nullableString(sourceURI.String()), reason, now)
	if err != nil {
		return false, fmt.Errorf("insert Tap ingestion receipt: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func enqueueIdentityRefreshTx(ctx context.Context, tx pgx.Tx, did syntax.DID, eventID uint64, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_identity_refresh_state(
			did,next_attempt_at,attempt_count,last_result,updated_at,tap_event_id
		) VALUES($1,$2,0,'pending',$2,$3)
		ON CONFLICT(did) DO UPDATE SET
			next_attempt_at=EXCLUDED.next_attempt_at,
			attempt_count=0,
			last_result='pending',
			updated_at=EXCLUDED.updated_at,
			tap_event_id=EXCLUDED.tap_event_id
		WHERE atproto_identity_refresh_state.tap_event_id IS NULL
		   OR atproto_identity_refresh_state.tap_event_id < EXCLUDED.tap_event_id
	`, did, now, eventID); err != nil {
		return fmt.Errorf("enqueue Tap identity refresh %s: %w", did, err)
	}
	return nil
}

func readCurrentJobOutcome(ctx context.Context, tx pgx.Tx, uri syntax.ATURI) (tap.Outcome, error) {
	var state string
	var kind, key *string
	var reason *tap.ReasonCode
	if err := tx.QueryRow(ctx, `
		SELECT state,dependency_kind,dependency_key,last_reason_code
		FROM tap_projection_jobs WHERE source_uri=$1
	`, uri).Scan(&state, &kind, &key, &reason); err != nil {
		return tap.Outcome{}, fmt.Errorf("read current projection job: %w", err)
	}
	if state == "blocked" && kind != nil && key != nil {
		blockedReason := tap.ReasonMissingMember
		if reason != nil {
			blockedReason = *reason
		} else {
			switch *kind {
			case "subject_uri":
				blockedReason = tap.ReasonMissingSubject
			case "repository_did":
				blockedReason = tap.ReasonSourceOrderUncertain
			}
		}
		return tap.Blocked(blockedReason, tap.Dependency{Kind: *kind, Key: *key}), nil
	}
	if state == "permanent_denied" {
		deniedReason := tap.ReasonOwnerTerminal
		if reason != nil {
			deniedReason = *reason
		}
		return tap.PermanentInvalid(deniedReason), nil
	}
	return tap.Applied(), nil
}

func (store *Store) ClaimProjectionJobs(ctx context.Context, request ProjectionClaimRequest) ([]ProjectionClaim, error) {
	if strings.TrimSpace(request.Worker) == "" || len(request.Worker) > 256 || request.LeaseToken == uuid.Nil ||
		request.LeaseDuration <= 0 || request.Limit <= 0 {
		return nil, errors.New("invalid Tap projection claim")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	expires := now.Add(request.LeaseDuration).Truncate(time.Microsecond)
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM tap_projection_jobs
			WHERE next_attempt_at <= $1
			  AND (state='pending' OR (state='processing' AND lease_expires_at <= $1))
			ORDER BY next_attempt_at,id
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE tap_projection_jobs AS job
		SET state='processing',attempts=job.attempts+1,
		    lease_owner=$3,lease_token=$4,lease_expires_at=$5,updated_at=$1
		FROM candidates
		WHERE job.id=candidates.id
		RETURNING job.id,job.source_uri,job.projection_kind,job.source_event_id,
		          job.state,job.attempts,job.lease_owner,job.lease_token,job.lease_expires_at
	`, now, request.Limit, strings.TrimSpace(request.Worker), request.LeaseToken, expires)
	if err != nil {
		return nil, fmt.Errorf("claim Tap projection jobs: %w", err)
	}
	defer rows.Close()
	claims := make([]ProjectionClaim, 0, request.Limit)
	for rows.Next() {
		var claim ProjectionClaim
		if err := rows.Scan(&claim.ID, &claim.SourceURI, &claim.ProjectionKind, &claim.SourceEventID,
			&claim.State, &claim.Attempts, &claim.LeaseOwner, &claim.LeaseToken, &claim.LeaseExpiresAt); err != nil {
			return nil, fmt.Errorf("scan Tap projection claim: %w", err)
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap projection claims: %w", err)
	}
	return claims, nil
}

func (store *Store) Project(ctx context.Context, claim ProjectionClaim, projector Projector) error {
	return store.project(ctx, claim, projector, time.Second)
}

func (store *Store) project(ctx context.Context, claim ProjectionClaim, projector Projector, retryDelay time.Duration) error {
	if claim.ID <= 0 || claim.LeaseToken == uuid.Nil || projector == nil {
		return ErrProjectionLeaseLost
	}
	if retryDelay <= 0 {
		return errors.New("tap projection retry delay must be positive")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var fencedActor syntax.DID
		if store.lifecycleAware {
			fencedActor = claim.SourceURI.Authority().DID()
			if fencedActor == "" {
				return errors.New("tap projection source URI has no DID authority")
			}
			// The actor fence must precede the source/job row locks below.
			// Profile lifecycle transitions take the exclusive owner fence
			// before updating this same source row; reversing that order would
			// create a row-lock/advisory-lock deadlock.
			if _, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, []syntax.DID{fencedActor}); err != nil {
				return fmt.Errorf("lock Tap projection actor lifecycle: %w", err)
			}
		}
		source, err := sourceTx(ctx, tx, claim.SourceURI)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && source.SourceEventID != claim.SourceEventID) {
			return ErrProjectionLeaseLost
		}
		if err != nil {
			return fmt.Errorf("lock Tap projection source: %w", err)
		}
		if store.lifecycleAware && source.DID != fencedActor {
			return errors.New("tap projection source DID does not match URI authority")
		}
		var state string
		var token uuid.UUID
		var expires time.Time
		var sourceEventID int64
		if err := tx.QueryRow(ctx, `
			SELECT state,lease_token,lease_expires_at,source_event_id
			FROM tap_projection_jobs WHERE id=$1 FOR UPDATE
		`, claim.ID).Scan(&state, &token, &expires, &sourceEventID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrProjectionLeaseLost
			}
			return fmt.Errorf("lock Tap projection job: %w", err)
		}
		if state != "processing" || token != claim.LeaseToken || !expires.After(now) || sourceEventID != int64(claim.SourceEventID) {
			return ErrProjectionLeaseLost
		}
		outcome, eligibleErr := store.projectionEligibility(ctx, tx, source, now)
		if eligibleErr != nil {
			return eligibleErr
		}
		if outcome.Kind == "" {
			outcome, err = projector(ctx, tx, source)
		} else {
			err = nil
		}
		if err != nil || outcome.Kind == tap.OutcomeRetryable {
			if err != nil {
				return err
			}
			return errors.New("projector returned retryable outcome")
		}
		switch outcome.Kind {
		case tap.OutcomeApplied:
			result, err := tx.Exec(ctx, `
				UPDATE tap_projection_jobs
				SET state='complete',lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
				    dependency_kind=NULL,dependency_key=NULL,last_reason_code=NULL,
				    completed_at=$3,updated_at=$3
				WHERE id=$1 AND lease_token=$2 AND source_event_id=$4
			`, claim.ID, claim.LeaseToken, now, claim.SourceEventID)
			if err != nil {
				return fmt.Errorf("complete Tap projection: %w", err)
			}
			if result.RowsAffected() != 1 {
				return ErrProjectionLeaseLost
			}
			if source.Action != "delete" {
				switch source.Collection {
				case "social.craftsky.actor.profile":
					if err := wakeDependencyTx(ctx, tx, tap.Dependency{Kind: "member_did", Key: source.DID.String()}, now); err != nil {
						return err
					}
				case "social.craftsky.feed.post":
					if err := wakeDependencyTx(ctx, tx, tap.Dependency{Kind: "subject_uri", Key: source.URI.String()}, now); err != nil {
						return err
					}
				}
			}
		case tap.OutcomeBlocked:
			if outcome.Dependency.Kind == "" || outcome.Dependency.Key == "" {
				return errors.New("blocked projection omitted dependency")
			}
			result, err := tx.Exec(ctx, `
				UPDATE tap_projection_jobs
				SET state='blocked',dependency_kind=$3,dependency_key=$4,
				    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
				    last_reason_code=$5,updated_at=$6
				WHERE id=$1 AND lease_token=$2
			`, claim.ID, claim.LeaseToken, outcome.Dependency.Kind, outcome.Dependency.Key, outcome.Reason, now)
			if err != nil {
				return fmt.Errorf("block Tap projection: %w", err)
			}
			if result.RowsAffected() != 1 {
				return ErrProjectionLeaseLost
			}
		case tap.OutcomePermanentInvalid:
			if validQuarantineReason(outcome.Reason) {
				if err := quarantineSourceTx(ctx, tx, source, outcome.Reason, now); err != nil {
					return err
				}
			}
			result, err := tx.Exec(ctx, `
				UPDATE tap_projection_jobs
				SET state='permanent_denied',dependency_kind=NULL,dependency_key=NULL,
				    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
				    last_reason_code=$3,completed_at=$4,updated_at=$4
				WHERE id=$1 AND lease_token=$2
			`, claim.ID, claim.LeaseToken, outcome.Reason, now)
			if err != nil {
				return fmt.Errorf("deny Tap projection: %w", err)
			}
			if result.RowsAffected() != 1 {
				return ErrProjectionLeaseLost
			}
		default:
			return errors.New("projector returned invalid outcome")
		}
		return nil
	})
	if err == nil || errors.Is(err, ErrProjectionLeaseLost) {
		return err
	}
	rescheduleErr := store.rescheduleProjection(ctx, claim, now.Add(retryDelay), tap.ReasonProjectionFailure)
	return errors.Join(err, rescheduleErr)
}

func (store *Store) rescheduleProjection(ctx context.Context, claim ProjectionClaim, next time.Time, reason tap.ReasonCode) error {
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_projection_jobs
		SET state='pending',dependency_kind=NULL,dependency_key=NULL,
		    next_attempt_at=$3,lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=$4,updated_at=$5
		WHERE id=$1 AND state='processing' AND lease_token=$2 AND lease_expires_at>$5
	`, claim.ID, claim.LeaseToken, next, reason, now)
	if err != nil {
		return fmt.Errorf("reschedule Tap projection: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrProjectionLeaseLost
	}
	return nil
}

func (store *Store) projectionEligibility(ctx context.Context, tx pgx.Tx, source SourceRecord, now time.Time) (tap.Outcome, error) {
	if !store.lifecycleAware {
		return tap.Outcome{}, nil
	}
	var state string
	var generation int64
	err := tx.QueryRow(ctx, `
		SELECT state,generation FROM owner_lifecycles WHERE owner_did=$1
	`, source.DID).Scan(&state, &generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: source.DID.String()}), nil
	}
	if err != nil {
		return tap.Outcome{}, fmt.Errorf("read owner projection lifecycle: %w", err)
	}
	if state == "terminal" || source.ProjectionDisposition == "denied_terminal" {
		return tap.PermanentInvalid(tap.ReasonOwnerTerminal), nil
	}
	if source.EffectOperationID != "" && source.Action != "delete" {
		recordContentFingerprint, err := pdseffects.RecordContentFingerprint(
			source.DID, source.Collection, source.Rkey, source.Record,
		)
		if err != nil {
			return tap.Outcome{}, err
		}
		attempt, err := ownerlifecycle.LockedPDSRecordEffectTx(
			ctx,
			tx,
			source.EffectOperationID,
			ownerlifecycle.PDSRecordSourceObservation{
				Owner: source.DID, URI: source.URI, CID: source.CID,
				RecordFingerprint: recordContentFingerprint,
			},
		)
		if errors.Is(err, ownerlifecycle.ErrEffectSourceMismatch) {
			if err := enqueueRepositoryJob(
				ctx, tx, source.DID, string(RepositoryJobPDSReconcile), now,
			); err != nil {
				return tap.Outcome{}, err
			}
			return tap.Blocked(
				tap.ReasonSourceOrderUncertain,
				tap.Dependency{Kind: "repository_did", Key: source.DID.String()},
			), nil
		}
		if err != nil {
			return tap.Outcome{}, fmt.Errorf("lock source PDS effect attempt: %w", err)
		}
		sourceDisposition := "pending"
		switch attempt.ProjectionDisposition {
		case ownerlifecycle.ProjectionDeniedTerminal:
			sourceDisposition = "denied_terminal"
		case ownerlifecycle.ProjectionNotApplicable:
			sourceDisposition = "not_accepted"
		case ownerlifecycle.ProjectionHiddenNonActive:
			sourceDisposition = "blocked_departed"
		case ownerlifecycle.ProjectionEligibleCurrent:
			sourceDisposition = "eligible"
		}
		if source.ProjectionDisposition != sourceDisposition {
			if _, err := tx.Exec(ctx, `
				UPDATE tap_source_records
				SET projection_disposition=$2,updated_at=$3
				WHERE uri=$1 AND effect_operation_id=$4
			`, source.URI, sourceDisposition, now, source.EffectOperationID); err != nil {
				return tap.Outcome{}, fmt.Errorf("synchronize source effect disposition: %w", err)
			}
		}
		switch attempt.ProjectionDisposition {
		case ownerlifecycle.ProjectionDeniedTerminal:
			return tap.PermanentInvalid(tap.ReasonOwnerTerminal), nil
		case ownerlifecycle.ProjectionNotApplicable:
			return tap.PermanentInvalid(tap.ReasonStaleSource), nil
		case ownerlifecycle.ProjectionHiddenNonActive, ownerlifecycle.ProjectionPending:
			if state == "active" {
				if err := enqueueRepositoryJob(
					ctx, tx, source.DID, string(RepositoryJobPDSReconcile), now,
				); err != nil {
					return tap.Outcome{}, err
				}
				return tap.Blocked(
					tap.ReasonSourceOrderUncertain,
					tap.Dependency{Kind: "repository_did", Key: source.DID.String()},
				), nil
			}
			return tap.Blocked(
				tap.ReasonOwnerDeparted,
				tap.Dependency{Kind: "member_did", Key: source.DID.String()},
			), nil
		case ownerlifecycle.ProjectionEligibleCurrent:
			if state != "active" {
				return tap.Blocked(
					tap.ReasonOwnerDeparted,
					tap.Dependency{Kind: "member_did", Key: source.DID.String()},
				), nil
			}
			return tap.Outcome{}, nil
		default:
			return tap.Outcome{}, ownerlifecycle.ErrEffectSourceAmbiguous
		}
	}
	if source.ProjectionDisposition == "not_accepted" {
		return tap.PermanentInvalid(tap.ReasonStaleSource), nil
	}
	if source.Action == "delete" {
		return tap.Outcome{}, nil
	}
	if state != "active" {
		return tap.Blocked(tap.ReasonOwnerDeparted, tap.Dependency{Kind: "member_did", Key: source.DID.String()}), nil
	}
	if source.OwnerGeneration == nil || *source.OwnerGeneration != generation || source.OrderingStatus == "uncertain" {
		if err := enqueueRepositoryJob(ctx, tx, source.DID, string(RepositoryJobPDSReconcile), now); err != nil {
			return tap.Outcome{}, err
		}
		return tap.Blocked(tap.ReasonSourceOrderUncertain, tap.Dependency{Kind: "repository_did", Key: source.DID.String()}), nil
	}
	return tap.Outcome{}, nil
}

func (store *Store) WakeDependency(ctx context.Context, dependency tap.Dependency) error {
	if dependency.Kind == "" || dependency.Key == "" {
		return errors.New("invalid Tap projection dependency")
	}
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		return wakeDependencyTx(ctx, tx, dependency, store.now().UTC().Truncate(time.Microsecond))
	})
}

func wakeDependencyTx(ctx context.Context, tx pgx.Tx, dependency tap.Dependency, now time.Time) error {
	_, err := tx.Exec(ctx, `
		UPDATE tap_projection_jobs
		SET state='pending',dependency_kind=NULL,dependency_key=NULL,
		    next_attempt_at=$3,last_reason_code=NULL,updated_at=$3
		WHERE state='blocked' AND dependency_kind=$1 AND dependency_key=$2
	`, dependency.Kind, dependency.Key, now)
	if err != nil {
		return fmt.Errorf("wake Tap projection dependency: %w", err)
	}
	return nil
}

func (store *Store) Source(ctx context.Context, uri syntax.ATURI) (SourceRecord, error) {
	return sourceRow(store.pool.QueryRow(ctx, sourceSelect+` WHERE uri=$1`, uri))
}

func sourceTx(ctx context.Context, tx pgx.Tx, uri syntax.ATURI) (SourceRecord, error) {
	return sourceRow(tx.QueryRow(ctx, sourceSelect+` WHERE uri=$1 FOR UPDATE`, uri))
}

const sourceSelect = `
	SELECT uri,did,collection,rkey,source_event_id,source_fingerprint,
	       revision,cid,action,record,record_bytes,live,ordering_status,projection_disposition,
	       owner_generation,effect_operation_id,projection_version,observed_at,updated_at
	FROM tap_source_records`

type rowScanner interface{ Scan(...any) error }

func sourceRow(row rowScanner) (SourceRecord, error) {
	var source SourceRecord
	var fingerprint []byte
	var cid, effectOperationID *string
	var record []byte
	var sourceEventID int64
	if err := row.Scan(&source.URI, &source.DID, &source.Collection, &source.Rkey,
		&sourceEventID, &fingerprint, &source.Revision, &cid, &source.Action, &record,
		&source.RecordBytes, &source.Live, &source.OrderingStatus, &source.ProjectionDisposition,
		&source.OwnerGeneration, &effectOperationID, &source.ProjectionVersion,
		&source.ObservedAt, &source.UpdatedAt); err != nil {
		return SourceRecord{}, err
	}
	if sourceEventID < 0 || len(fingerprint) != sha256.Size || source.RecordBytes < 0 || source.RecordBytes > maxDurableRecordBytes {
		return SourceRecord{}, errors.New("invalid persisted Tap source")
	}
	source.SourceEventID = uint64(sourceEventID)
	copy(source.SourceFingerprint[:], fingerprint)
	if cid != nil {
		source.CID = syntax.CID(*cid)
	}
	if effectOperationID != nil {
		source.EffectOperationID = *effectOperationID
	}
	if record != nil {
		source.Record = append(json.RawMessage(nil), record...)
	}
	return source, nil
}

func (store *Store) ProjectionJob(ctx context.Context, uri syntax.ATURI) (ProjectionJob, error) {
	var job ProjectionJob
	var kind, key, owner, token *string
	var expires *time.Time
	var sourceEventID int64
	if err := store.pool.QueryRow(ctx, `
		SELECT id,source_uri,projection_kind,source_event_id,state,
		       dependency_kind,dependency_key,attempts,lease_owner,
		       lease_token::text,lease_expires_at
		FROM tap_projection_jobs WHERE source_uri=$1
	`, uri).Scan(&job.ID, &job.SourceURI, &job.ProjectionKind, &sourceEventID, &job.State,
		&kind, &key, &job.Attempts, &owner, &token, &expires); err != nil {
		return ProjectionJob{}, err
	}
	job.SourceEventID = uint64(sourceEventID)
	if kind != nil {
		job.Dependency.Kind = *kind
	}
	if key != nil {
		job.Dependency.Key = *key
	}
	if owner != nil {
		job.LeaseOwner = *owner
	}
	if token != nil {
		job.LeaseToken, _ = uuid.Parse(*token)
	}
	if expires != nil {
		job.LeaseExpiresAt = *expires
	}
	return job, nil
}

func enqueueRepositoryJob(ctx context.Context, tx pgx.Tx, did syntax.DID, kind string, now time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO tap_repository_jobs(id,did,job_kind,state,next_attempt_at,created_at,updated_at)
		VALUES($1,$2,$3,'pending',$4,$4,$4)
		ON CONFLICT(did,job_kind) DO UPDATE SET
			state='pending',next_attempt_at=EXCLUDED.next_attempt_at,
			lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			last_reason_code=NULL,last_successful_at=NULL,updated_at=EXCLUDED.updated_at
	`, uuid.New(), did, kind, now)
	if err != nil {
		return fmt.Errorf("enqueue Tap repository job: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
