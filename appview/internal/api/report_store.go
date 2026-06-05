// appview/internal/api/report_store.go
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportSubjectType identifies the kind of subject a private report targets.
type ReportSubjectType string

const (
	// ReportSubjectPost is a report against an indexed Craftsky post.
	ReportSubjectPost ReportSubjectType = "post"
	// ReportSubjectAccount is a report against a profile/account DID.
	ReportSubjectAccount ReportSubjectType = "account"
)

// CreateReportInput is the private AppView report row shape accepted by
// ReportStore. Request validation and subject resolution happen before this
// layer; the store persists canonical subject snapshots and safe forwarding
// metadata only.
type CreateReportInput struct {
	ReporterDID             string
	SubjectType             ReportSubjectType
	SubjectDID              string
	SubjectCollection       *string
	SubjectRkey             *string
	SubjectURI              *string
	SubjectCIDSnapshot      *string
	SubmittedHandleSnapshot *string
	ReasonType              string
	Details                 *string
	DeviceID                *string
	ForwardingStatus        string
	ForwardingSchemaVersion *string
	ForwardingPreparedAt    time.Time
}

// ReportRow is a persisted private report row. It intentionally contains no
// prepared future forwarding payload and is not returned directly from public
// report endpoints.
type ReportRow struct {
	ID                      string
	ReporterDID             string
	SubjectType             ReportSubjectType
	SubjectDID              string
	SubjectCollection       *string
	SubjectRkey             *string
	SubjectURI              *string
	SubjectCIDSnapshot      *string
	SubmittedHandleSnapshot *string
	ReasonType              string
	Details                 *string
	DeviceID                *string
	ForwardingStatus        string
	ForwardingSchemaVersion *string
	ForwardingPreparedAt    time.Time
	CreatedAt               time.Time
}

// ReportStore persists AppView-private moderation report intake rows.
type ReportStore struct {
	pool *pgxpool.Pool
}

func NewReportStore(pool *pgxpool.Pool) *ReportStore {
	return &ReportStore{pool: pool}
}

// CreateReport inserts a private report row. Duplicate reports are allowed by
// design; there is deliberately no uniqueness constraint over reporter/subject.
func (s *ReportStore) CreateReport(ctx context.Context, input CreateReportInput) (*ReportRow, error) {
	const q = `
		INSERT INTO moderation_reports (
			id,
			reporter_did,
			subject_type,
			subject_did,
			subject_collection,
			subject_rkey,
			subject_uri,
			subject_cid_snapshot,
			submitted_handle_snapshot,
			reason_type,
			details,
			device_id,
			forwarding_status,
			forwarding_schema_version,
			forwarding_prepared_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING
			id,
			reporter_did,
			subject_type,
			subject_did,
			subject_collection,
			subject_rkey,
			subject_uri,
			subject_cid_snapshot,
			submitted_handle_snapshot,
			reason_type,
			details,
			device_id,
			forwarding_status,
			forwarding_schema_version,
			forwarding_prepared_at,
			created_at
	`
	out := &ReportRow{}
	var subjectType string
	err := s.pool.QueryRow(ctx, q,
		uuid.NewString(),
		input.ReporterDID,
		string(input.SubjectType),
		input.SubjectDID,
		input.SubjectCollection,
		input.SubjectRkey,
		input.SubjectURI,
		input.SubjectCIDSnapshot,
		input.SubmittedHandleSnapshot,
		input.ReasonType,
		input.Details,
		input.DeviceID,
		input.ForwardingStatus,
		input.ForwardingSchemaVersion,
		input.ForwardingPreparedAt,
	).Scan(
		&out.ID,
		&out.ReporterDID,
		&subjectType,
		&out.SubjectDID,
		&out.SubjectCollection,
		&out.SubjectRkey,
		&out.SubjectURI,
		&out.SubjectCIDSnapshot,
		&out.SubmittedHandleSnapshot,
		&out.ReasonType,
		&out.Details,
		&out.DeviceID,
		&out.ForwardingStatus,
		&out.ForwardingSchemaVersion,
		&out.ForwardingPreparedAt,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("report create: %w", err)
	}
	out.SubjectType = ReportSubjectType(subjectType)
	return out, nil
}
