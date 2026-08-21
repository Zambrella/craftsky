// appview/internal/api/moderation_store.go
package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

const moderationIdempotencyReceiptTTL = 24 * time.Hour

var (
	ErrModerationIdempotencyKeyMissing   = errors.New("moderation idempotency key is missing")
	ErrModerationIdempotencyKeyInvalid   = errors.New("moderation idempotency key is invalid")
	ErrModerationIdempotencyKeyConflict  = errors.New("moderation idempotency key conflicts with an existing request")
	ErrModerationSourceLifecycleRequired = errors.New("moderation source requires an owner lifecycle")
)

type ModerationSubjectType string
type ModerationValue string
type ModerationAction string
type ModerationSourceAuthority string

const (
	ModerationSubjectPost    ModerationSubjectType = "post"
	ModerationSubjectAccount ModerationSubjectType = "account"

	ModerationValueHide     ModerationValue = "hide"
	ModerationValueTakedown ModerationValue = "takedown"
	ModerationValueWarn     ModerationValue = "warn"

	ModerationActionApply  ModerationAction = "apply"
	ModerationActionNegate ModerationAction = "negate"

	// ModerationSourceTrustedExternal is assigned only after the dedicated
	// moderation credential and configured source allow-list both succeed.
	ModerationSourceTrustedExternal ModerationSourceAuthority = "trusted_external"
)

type ModerationOutputInput struct {
	SourceDID         string
	SourceAuthority   ModerationSourceAuthority
	SubjectType       ModerationSubjectType
	SubjectDID        string
	SubjectCollection *string
	SubjectRkey       *string
	SubjectURI        *string
	Value             ModerationValue
	Action            ModerationAction
	InternalReason    *string
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

type ModerationOutputRow struct {
	ID                string
	SourceDID         string
	SubjectType       ModerationSubjectType
	SubjectDID        string
	SubjectCollection *string
	SubjectRkey       *string
	SubjectURI        *string
	Value             ModerationValue
	Action            ModerationAction
	InternalReason    *string
	ExpiresAt         *time.Time
	CreatedAt         time.Time
	IndexedAt         time.Time
}

type ModerationInsertResult struct {
	OutputID string
	Status   string
	Replayed bool
	Row      *ModerationOutputRow
}

type ModerationSubjectRef struct {
	Type ModerationSubjectType
	DID  string
	URI  *string
}

type ModerationStore struct {
	pool       *pgxpool.Pool
	lifecycles *ownerlifecycle.Store
	now        func() time.Time
}

func NewModerationStore(
	pool *pgxpool.Pool,
	lifecycles *ownerlifecycle.Store,
) (*ModerationStore, error) {
	return NewModerationStoreWithClock(pool, lifecycles, time.Now)
}

func NewModerationStoreWithClock(
	pool *pgxpool.Pool,
	lifecycles *ownerlifecycle.Store,
	now func() time.Time,
) (*ModerationStore, error) {
	if pool == nil || lifecycles == nil {
		return nil, errors.New("moderation store requires database and owner lifecycle stores")
	}
	if now == nil {
		now = time.Now
	}
	return &ModerationStore{pool: pool, lifecycles: lifecycles, now: now}, nil
}

func ValidateModerationIdempotencyKey(key string) error {
	if key == "" {
		return ErrModerationIdempotencyKeyMissing
	}
	if len(key) < 16 || len(key) > 128 {
		return ErrModerationIdempotencyKeyInvalid
	}
	for index := range len(key) {
		if key[index] < 0x20 || key[index] > 0x7e {
			return ErrModerationIdempotencyKeyInvalid
		}
	}
	return nil
}

func (s *ModerationStore) InsertOutput(
	ctx context.Context,
	idempotencyKey string,
	input ModerationOutputInput,
) (*ModerationInsertResult, error) {
	if s == nil || s.pool == nil || s.lifecycles == nil {
		return nil, errors.New("moderation store is unavailable")
	}
	if err := ValidateModerationIdempotencyKey(idempotencyKey); err != nil {
		return nil, err
	}
	sourceDID, err := syntax.ParseDID(input.SourceDID)
	if err != nil {
		return nil, fmt.Errorf("moderation source DID: %w", err)
	}
	subjectDID, err := syntax.ParseDID(input.SubjectDID)
	if err != nil {
		return nil, fmt.Errorf("moderation subject DID: %w", err)
	}
	fingerprint, err := moderationRequestFingerprint(input)
	if err != nil {
		return nil, err
	}
	keyHash := sha256.Sum256([]byte(idempotencyKey))
	now := s.now().UTC().Truncate(time.Microsecond)
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	} else {
		createdAt = createdAt.UTC().Truncate(time.Microsecond)
	}
	outputID := uuid.NewString()
	var inserted *ModerationOutputRow
	result := &ModerationInsertResult{OutputID: outputID, Status: "indexed"}
	err = s.lifecycles.WithNonTerminalOwners(ctx, []syntax.DID{sourceDID, subjectDID}, func(_ context.Context, tx pgx.Tx, existing map[syntax.DID]ownerlifecycle.Lifecycle) error {
		if _, sourceKnown := existing[sourceDID]; !sourceKnown && input.SourceAuthority != ModerationSourceTrustedExternal {
			return ErrModerationSourceLifecycleRequired
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM moderation_idempotency_receipts
			WHERE request_key_hash=$1 AND expires_at <= $2
		`, keyHash[:], now); err != nil {
			return fmt.Errorf("expire moderation idempotency receipt: %w", err)
		}
		insertReceipt, err := tx.Exec(ctx, `
			INSERT INTO moderation_idempotency_receipts(
				request_key_hash,request_fingerprint,output_id,output_status,
				created_at,expires_at
			) VALUES($1,$2,$3,'indexed',$4,$5)
			ON CONFLICT (request_key_hash) DO NOTHING
		`, keyHash[:], fingerprint[:], outputID, now, now.Add(moderationIdempotencyReceiptTTL))
		if err != nil {
			return fmt.Errorf("insert moderation idempotency receipt: %w", err)
		}
		if insertReceipt.RowsAffected() == 0 {
			var storedFingerprint []byte
			if err := tx.QueryRow(ctx, `
				SELECT request_fingerprint,output_id,output_status
				FROM moderation_idempotency_receipts
				WHERE request_key_hash=$1 AND expires_at>$2
			`, keyHash[:], now).Scan(&storedFingerprint, &result.OutputID, &result.Status); err != nil {
				return fmt.Errorf("read moderation idempotency receipt: %w", err)
			}
			if !equalSHA256(storedFingerprint, fingerprint) {
				return ErrModerationIdempotencyKeyConflict
			}
			result.Replayed = true
			return nil
		}

		const q = `
		INSERT INTO moderation_outputs (
			id, source_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, value, action, internal_reason,
			expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING
			id, source_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, value, action, internal_reason,
			expires_at, created_at, indexed_at
	`
		inserted, err = scanModerationOutputRow(tx.QueryRow(ctx, q,
			outputID,
			input.SourceDID,
			string(input.SubjectType),
			input.SubjectDID,
			input.SubjectCollection,
			input.SubjectRkey,
			input.SubjectURI,
			string(input.Value),
			string(input.Action),
			input.InternalReason,
			input.ExpiresAt,
			createdAt,
		))
		if err != nil {
			return fmt.Errorf("moderation output insert: %w", err)
		}
		if qualifiesForModerationRestoration(input) {
			if _, err := tx.Exec(ctx, `
				INSERT INTO moderation_restoration_outbox(
					moderation_output_id,target_did,status,created_at
				) VALUES($1,$2,'pending',$3)
			`, outputID, input.SubjectDID, now); err != nil {
				return fmt.Errorf("moderation restoration intent insert: %w", err)
			}
		}
		result.Row = inserted
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ModerationStore) SweepExpiredIdempotencyReceipts(ctx context.Context, limit int) (int, error) {
	if s == nil || s.pool == nil {
		return 0, errors.New("moderation store is unavailable")
	}
	if limit < 1 || limit > 1000 {
		return 0, errors.New("moderation receipt sweep limit must be between 1 and 1000")
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	result, err := s.pool.Exec(ctx, `
		WITH expired AS (
			SELECT request_key_hash
			FROM moderation_idempotency_receipts
			WHERE expires_at <= $1
			ORDER BY expires_at,request_key_hash
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM moderation_idempotency_receipts AS receipt
		USING expired
		WHERE receipt.request_key_hash=expired.request_key_hash
	`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("sweep moderation idempotency receipts: %w", err)
	}
	return int(result.RowsAffected()), nil
}

func qualifiesForModerationRestoration(input ModerationOutputInput) bool {
	return input.SubjectType == ModerationSubjectAccount &&
		input.Action == ModerationActionNegate &&
		(input.Value == ModerationValueHide || input.Value == ModerationValueTakedown)
}

func moderationRequestFingerprint(input ModerationOutputInput) ([sha256.Size]byte, error) {
	type canonicalRequest struct {
		SourceDID         string                `json:"sourceDid"`
		SubjectType       ModerationSubjectType `json:"subjectType"`
		SubjectDID        string                `json:"subjectDid"`
		SubjectCollection *string               `json:"subjectCollection"`
		SubjectRkey       *string               `json:"subjectRkey"`
		SubjectURI        *string               `json:"subjectUri"`
		Value             ModerationValue       `json:"value"`
		Action            ModerationAction      `json:"action"`
		InternalReason    *string               `json:"internalReason"`
		ExpiresAt         *string               `json:"expiresAt"`
	}
	var expiresAt *string
	if input.ExpiresAt != nil {
		formatted := input.ExpiresAt.UTC().Format(time.RFC3339Nano)
		expiresAt = &formatted
	}
	encoded, err := json.Marshal(canonicalRequest{
		SourceDID: input.SourceDID, SubjectType: input.SubjectType,
		SubjectDID: input.SubjectDID, SubjectCollection: input.SubjectCollection,
		SubjectRkey: input.SubjectRkey, SubjectURI: input.SubjectURI,
		Value: input.Value, Action: input.Action, InternalReason: input.InternalReason,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint moderation request: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func equalSHA256(stored []byte, expected [sha256.Size]byte) bool {
	return bytes.Equal(stored, expected[:])
}

func (s *ModerationStore) ActivePolicyForSubject(ctx context.Context, subject ModerationSubjectRef, now time.Time) (ModerationPolicy, error) {
	q := `
		SELECT
			id, source_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, value, action, internal_reason,
			expires_at, created_at, indexed_at
		FROM moderation_outputs
		WHERE subject_type = $1
		  AND subject_did = $2
		  AND ($3::text IS NULL OR subject_uri = $3)
		  AND NOT appview_owner_is_terminal(source_did)
		  AND NOT appview_owner_is_terminal(subject_did)
		ORDER BY indexed_at ASC, id ASC
	`
	rows, err := s.pool.Query(ctx, q, string(subject.Type), subject.DID, subject.URI)
	if err != nil {
		return ModerationPolicy{}, fmt.Errorf("moderation active policy query: %w", err)
	}
	defer rows.Close()
	outputs := []ModerationOutputRow{}
	for rows.Next() {
		row, err := scanModerationOutputRow(rows)
		if err != nil {
			return ModerationPolicy{}, fmt.Errorf("moderation active policy scan: %w", err)
		}
		outputs = append(outputs, *row)
	}
	if err := rows.Err(); err != nil {
		return ModerationPolicy{}, fmt.Errorf("moderation active policy iter: %w", err)
	}
	return ComputeModerationPolicy(outputs, now), nil
}

func scanModerationOutputRow(scanner pgx.Row) (*ModerationOutputRow, error) {
	out := &ModerationOutputRow{}
	var subjectType, value, action string
	err := scanner.Scan(
		&out.ID,
		&out.SourceDID,
		&subjectType,
		&out.SubjectDID,
		&out.SubjectCollection,
		&out.SubjectRkey,
		&out.SubjectURI,
		&value,
		&action,
		&out.InternalReason,
		&out.ExpiresAt,
		&out.CreatedAt,
		&out.IndexedAt,
	)
	out.SubjectType = ModerationSubjectType(subjectType)
	out.Value = ModerationValue(value)
	out.Action = ModerationAction(action)
	return out, err
}
