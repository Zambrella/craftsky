// appview/internal/api/moderation_store.go
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModerationSubjectType string
type ModerationValue string
type ModerationAction string

const (
	ModerationSubjectPost    ModerationSubjectType = "post"
	ModerationSubjectAccount ModerationSubjectType = "account"

	ModerationValueHide     ModerationValue = "hide"
	ModerationValueTakedown ModerationValue = "takedown"
	ModerationValueWarn     ModerationValue = "warn"

	ModerationActionApply  ModerationAction = "apply"
	ModerationActionNegate ModerationAction = "negate"
)

type ModerationOutputInput struct {
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

type ModerationSubjectRef struct {
	Type ModerationSubjectType
	DID  string
	URI  *string
}

type ModerationStore struct {
	pool *pgxpool.Pool
}

func NewModerationStore(pool *pgxpool.Pool) *ModerationStore {
	return &ModerationStore{pool: pool}
}

func (s *ModerationStore) InsertOutput(ctx context.Context, input ModerationOutputInput) (*ModerationOutputRow, error) {
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
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
	row, err := scanModerationOutputRow(s.pool.QueryRow(ctx, q,
		uuid.NewString(),
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
		return nil, fmt.Errorf("moderation output insert: %w", err)
	}
	return row, nil
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
