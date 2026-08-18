package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

const maxQuarantineEnvelopeBytes = 64 << 10

type QuarantinedEvent struct {
	Fingerprint     [32]byte
	TapEventID      uint64
	EventType       string
	Reason          tap.ReasonCode
	Envelope        []byte
	OccurrenceCount int64
	ReplayState     string
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	ResolvedAt      *time.Time
}

type QuarantineClaimRequest struct {
	Worker        string
	LeaseToken    uuid.UUID
	LeaseDuration time.Duration
	Limit         int
}

type QuarantineClaim struct {
	QuarantinedEvent
	LeaseOwner     string
	LeaseToken     uuid.UUID
	LeaseExpiresAt time.Time
}

type QuarantineReplayHandler func(context.Context, []byte) (tap.Outcome, error)

// ReplayEnvelope reclassifies either a projection-generated source quarantine
// or an original Tap wire envelope. Projection quarantines already have a
// durable source row, so replay only returns the matching current job to the
// normal projector; original envelopes go through Tap's shared decoder and the
// supplied durable ingestor. A stale source quarantine is resolved as a no-op.
func (store *Store) ReplayEnvelope(
	ctx context.Context,
	raw []byte,
	ingestor tap.DurableIngestor,
) (tap.Outcome, error) {
	var sourceEnvelope struct {
		Source *SourceRecord `json:"source"`
	}
	if err := json.Unmarshal(raw, &sourceEnvelope); err == nil && sourceEnvelope.Source != nil {
		source := sourceEnvelope.Source
		if source.URI == "" || source.SourceEventID == 0 || source.SourceFingerprint == ([32]byte{}) {
			return tap.Retryable(tap.ReasonInvalidEnvelope), errors.New("invalid quarantined Tap source envelope")
		}
		now := store.now().UTC().Truncate(time.Microsecond)
		err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
			var currentID int64
			var currentFingerprint []byte
			err := tx.QueryRow(ctx, `
				SELECT source_event_id,source_fingerprint
				FROM tap_source_records
				WHERE uri=$1
				FOR UPDATE
			`, source.URI).Scan(&currentID, &currentFingerprint)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("lock quarantined Tap source: %w", err)
			}
			if currentID != int64(source.SourceEventID) || !bytes.Equal(currentFingerprint, source.SourceFingerprint[:]) {
				return nil
			}
			if _, err := tx.Exec(ctx, `
				UPDATE tap_projection_jobs
				SET state='pending',attempts=0,next_attempt_at=$3,
				    dependency_kind=NULL,dependency_key=NULL,last_reason_code=NULL,
				    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
				    completed_at=NULL,updated_at=$3
				WHERE source_uri=$1 AND source_event_id=$2
				  AND state='permanent_denied'
			`, source.URI, source.SourceEventID, now); err != nil {
				return fmt.Errorf("requeue quarantined Tap source: %w", err)
			}
			return nil
		})
		if err != nil {
			return tap.Retryable(tap.ReasonStorageUnavailable), err
		}
		return tap.Applied(), nil
	}
	return tap.ReplayEnvelope(ctx, raw, ingestor)
}

func (store *Store) Quarantine(ctx context.Context, event tap.InvalidEvent) (tap.Outcome, error) {
	if event.Reason == "" || !validQuarantineReason(event.Reason) {
		return tap.Retryable(tap.ReasonStorageUnavailable), errors.New("invalid quarantine reason")
	}
	fingerprint := quarantineFingerprint(event)
	evidence, err := boundedQuarantineEnvelope(event.Envelope)
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	outcome := tap.PermanentInvalid(event.Reason)
	err = pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		return persistQuarantineTx(ctx, tx, event, fingerprint, evidence, outcome, now)
	})
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return outcome, nil
}

func persistQuarantineTx(
	ctx context.Context,
	tx pgx.Tx,
	event tap.InvalidEvent,
	fingerprint [32]byte,
	evidence []byte,
	outcome tap.Outcome,
	now time.Time,
) error {
	eventType := strings.TrimSpace(event.Type)
	if eventType == "" {
		eventType = "unknown"
	}
	if len(eventType) > 64 {
		eventType = "unsupported"
	}
	if _, err := tx.Exec(ctx, `
			INSERT INTO tap_quarantined_events(
				event_fingerprint,tap_event_id,event_type,reason_code,envelope,
				envelope_bytes,occurrence_count,replay_state,first_seen_at,last_seen_at
			) VALUES($1,$2,$3,$4,$5,$6,1,'quarantined',$7,$7)
			ON CONFLICT(event_fingerprint) DO UPDATE SET
				occurrence_count=tap_quarantined_events.occurrence_count+1,
				last_seen_at=EXCLUDED.last_seen_at,
				replay_state=CASE
					WHEN tap_quarantined_events.replay_state='resolved' THEN 'quarantined'
					ELSE tap_quarantined_events.replay_state
				END,
				resolved_at=CASE
					WHEN tap_quarantined_events.replay_state='resolved' THEN NULL
					ELSE tap_quarantined_events.resolved_at
				END
	`, fingerprint[:], event.ID, eventType, event.Reason, evidence, len(evidence), now); err != nil {
		return fmt.Errorf("persist quarantined Tap event: %w", err)
	}
	return insertReceipt(ctx, tx, fingerprint, event.ID, "quarantine", outcome, "", event.Reason, now)
}

func quarantineSourceTx(ctx context.Context, tx pgx.Tx, source SourceRecord, reason tap.ReasonCode, now time.Time) error {
	envelope, err := json.Marshal(struct {
		ID     uint64          `json:"id"`
		Type   string          `json:"type"`
		Source SourceRecord    `json:"source"`
		Record json.RawMessage `json:"record,omitempty"`
	}{ID: source.SourceEventID, Type: "record", Source: source, Record: source.Record})
	if err != nil {
		return fmt.Errorf("encode quarantined source: %w", err)
	}
	event := tap.InvalidEvent{ID: source.SourceEventID, Type: "record", Reason: reason, Envelope: envelope}
	fingerprint := quarantineFingerprint(event)
	evidence, err := boundedQuarantineEnvelope(envelope)
	if err != nil {
		return err
	}
	return persistQuarantineTx(ctx, tx, event, fingerprint, evidence, tap.PermanentInvalid(reason), now)
}

func quarantineFingerprint(event tap.InvalidEvent) [32]byte {
	hash := sha256.New()
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], event.ID)
	_, _ = hash.Write(id[:])
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(event.Type))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(event.Reason))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(event.Envelope)
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func boundedQuarantineEnvelope(raw []byte) ([]byte, error) {
	if len(raw) <= maxQuarantineEnvelopeBytes && json.Valid(raw) {
		return append([]byte(nil), raw...), nil
	}
	digest := sha256.Sum256(raw)
	evidence, err := json.Marshal(struct {
		Truncated     bool   `json:"truncated"`
		OriginalBytes int    `json:"originalBytes"`
		SHA256        string `json:"sha256"`
	}{Truncated: true, OriginalBytes: len(raw), SHA256: hex.EncodeToString(digest[:])})
	if err != nil {
		return nil, fmt.Errorf("encode bounded quarantine evidence: %w", err)
	}
	return evidence, nil
}

func validQuarantineReason(reason tap.ReasonCode) bool {
	switch reason {
	case tap.ReasonInvalidEnvelope, tap.ReasonMissingRecord, tap.ReasonInvalidDID,
		tap.ReasonInvalidCollection, tap.ReasonInvalidRecordKey,
		tap.ReasonUnsupportedAction, tap.ReasonMalformedRecord,
		tap.ReasonRecordTooLarge,
		tap.ReasonUnsupportedEventType, tap.ReasonInvalidIdentity,
		tap.ReasonUnsupportedIdentityStatus:
		return true
	default:
		return false
	}
}

func (store *Store) ListQuarantine(ctx context.Context, limit int) ([]QuarantinedEvent, error) {
	if limit <= 0 || limit > 1000 {
		return nil, errors.New("quarantine list limit must be between 1 and 1000")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT event_fingerprint,tap_event_id,event_type,reason_code,envelope,
		       occurrence_count,replay_state,first_seen_at,last_seen_at,resolved_at
		FROM tap_quarantined_events
		ORDER BY last_seen_at DESC,event_fingerprint
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list Tap quarantine: %w", err)
	}
	defer rows.Close()
	items := make([]QuarantinedEvent, 0, limit)
	for rows.Next() {
		var item QuarantinedEvent
		var fingerprint []byte
		var eventID int64
		if err := rows.Scan(&fingerprint, &eventID, &item.EventType, &item.Reason,
			&item.Envelope, &item.OccurrenceCount, &item.ReplayState,
			&item.FirstSeenAt, &item.LastSeenAt, &item.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan Tap quarantine: %w", err)
		}
		if len(fingerprint) != sha256.Size || eventID < 0 {
			return nil, errors.New("invalid persisted Tap quarantine row")
		}
		copy(item.Fingerprint[:], fingerprint)
		item.TapEventID = uint64(eventID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap quarantine: %w", err)
	}
	return items, nil
}

func (store *Store) RequestQuarantineReplay(ctx context.Context, fingerprint [32]byte) error {
	if fingerprint == ([32]byte{}) {
		return errors.New("invalid quarantine fingerprint")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_quarantined_events
		SET replay_state='pending',lease_owner=NULL,lease_token=NULL,
		    lease_expires_at=NULL,resolved_at=NULL,last_seen_at=GREATEST(last_seen_at,$2)
		WHERE event_fingerprint=$1 AND replay_state <> 'processing'
	`, fingerprint[:], now)
	if err != nil {
		return fmt.Errorf("request Tap quarantine replay: %w", err)
	}
	if result.RowsAffected() != 1 {
		return errors.New("quarantine row missing or already processing")
	}
	return nil
}

func (store *Store) ClaimQuarantine(ctx context.Context, request QuarantineClaimRequest) ([]QuarantineClaim, error) {
	if strings.TrimSpace(request.Worker) == "" || len(request.Worker) > 256 ||
		request.LeaseToken == uuid.Nil || request.LeaseDuration <= 0 || request.Limit <= 0 {
		return nil, errors.New("invalid quarantine claim")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	expires := now.Add(request.LeaseDuration).Truncate(time.Microsecond)
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT event_fingerprint
			FROM tap_quarantined_events
			WHERE replay_state='pending'
			   OR (replay_state='processing' AND lease_expires_at <= $1)
			ORDER BY last_seen_at,event_fingerprint
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE tap_quarantined_events AS event
		SET replay_state='processing',lease_owner=$3,lease_token=$4,
		    lease_expires_at=$5,last_seen_at=GREATEST(event.last_seen_at,$1)
		FROM candidates
		WHERE event.event_fingerprint=candidates.event_fingerprint
		RETURNING event.event_fingerprint,event.tap_event_id,event.event_type,
		          event.reason_code,event.envelope,event.occurrence_count,
		          event.replay_state,event.first_seen_at,event.last_seen_at,
		          event.resolved_at,event.lease_owner,event.lease_token,
		          event.lease_expires_at
	`, now, request.Limit, strings.TrimSpace(request.Worker), request.LeaseToken, expires)
	if err != nil {
		return nil, fmt.Errorf("claim Tap quarantine: %w", err)
	}
	defer rows.Close()
	claims := make([]QuarantineClaim, 0, request.Limit)
	for rows.Next() {
		var claim QuarantineClaim
		var fingerprint []byte
		var eventID int64
		if err := rows.Scan(&fingerprint, &eventID, &claim.EventType, &claim.Reason,
			&claim.Envelope, &claim.OccurrenceCount, &claim.ReplayState,
			&claim.FirstSeenAt, &claim.LastSeenAt, &claim.ResolvedAt,
			&claim.LeaseOwner, &claim.LeaseToken, &claim.LeaseExpiresAt); err != nil {
			return nil, fmt.Errorf("scan Tap quarantine claim: %w", err)
		}
		if len(fingerprint) != sha256.Size || eventID < 0 {
			return nil, errors.New("invalid persisted Tap quarantine claim")
		}
		copy(claim.Fingerprint[:], fingerprint)
		claim.TapEventID = uint64(eventID)
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap quarantine claims: %w", err)
	}
	return claims, nil
}

func (store *Store) ReplayQuarantine(ctx context.Context, claim QuarantineClaim, handler QuarantineReplayHandler) error {
	if claim.Fingerprint == ([32]byte{}) || claim.LeaseToken == uuid.Nil || handler == nil {
		return ErrProjectionLeaseLost
	}
	outcome, replayErr := handler(ctx, append([]byte(nil), claim.Envelope...))
	if replayErr != nil || !outcome.Acknowledgable() {
		if err := store.rescheduleQuarantine(ctx, claim); err != nil {
			return errors.Join(replayErr, err)
		}
		return replayErr
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_quarantined_events
		SET replay_state='resolved',lease_owner=NULL,lease_token=NULL,
		    lease_expires_at=NULL,resolved_at=$3,last_seen_at=GREATEST(last_seen_at,$3)
		WHERE event_fingerprint=$1 AND replay_state='processing'
		  AND lease_token=$2 AND lease_expires_at>$3
	`, claim.Fingerprint[:], claim.LeaseToken, now)
	if err != nil {
		return fmt.Errorf("complete Tap quarantine replay: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrProjectionLeaseLost
	}
	return nil
}

func (store *Store) rescheduleQuarantine(ctx context.Context, claim QuarantineClaim) error {
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_quarantined_events
		SET replay_state='pending',lease_owner=NULL,lease_token=NULL,
		    lease_expires_at=NULL,last_seen_at=GREATEST(last_seen_at,$3)
		WHERE event_fingerprint=$1 AND replay_state='processing'
		  AND lease_token=$2 AND lease_expires_at>$3
	`, claim.Fingerprint[:], claim.LeaseToken, now)
	if err != nil {
		return fmt.Errorf("reschedule Tap quarantine replay: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrProjectionLeaseLost
	}
	return nil
}
