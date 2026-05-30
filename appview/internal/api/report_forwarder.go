// appview/internal/api/report_forwarder.go
package api

import (
	"context"
	"time"
)

const reportForwardingSchemaVersion = "atproto-create-report-v0"

// ReportForwarder is the seam for future PDS/Ozone report forwarding. The MVP
// implementation prepares future payload inputs in memory but stops before any
// network submission.
type ReportForwarder interface {
	Prepare(ctx context.Context, input ReportForwardingInput) (ForwardingMetadata, error)
}

// ReportSubjectSnapshot carries canonical subject identity into the forwarding
// seam. It may include private/audit snapshots and must not be serialized into
// persisted forwarding metadata.
type ReportSubjectSnapshot struct {
	Type           ReportSubjectType
	DID            string
	Collection     *string
	Rkey           *string
	URI            *string
	CIDSnapshot    *string
	HandleSnapshot *string
}

// ReportForwardingInput contains all data needed to construct a future
// atproto/Ozone report payload. It is intentionally input-only for this MVP.
type ReportForwardingInput struct {
	ReportID    string
	ReporterDID string
	Subject     ReportSubjectSnapshot
	ReasonType  string
	Details     *string
}

// ForwardingMetadata is the only forwarding data intended for persistence in
// the MVP. It deliberately excludes the full prepared future payload, reporter,
// reason, details, and subject identity.
type ForwardingMetadata struct {
	Status        string    `json:"status"`
	SchemaVersion *string   `json:"schemaVersion,omitempty"`
	PreparedAt    time.Time `json:"preparedAt"`
}

// PlaceholderReportForwarder prepares report-forwarding metadata without
// submitting to PDS/Ozone.
type PlaceholderReportForwarder struct {
	now func() time.Time
}

func NewPlaceholderReportForwarder(now func() time.Time) *PlaceholderReportForwarder {
	if now == nil {
		now = time.Now
	}
	return &PlaceholderReportForwarder{now: now}
}

func (f *PlaceholderReportForwarder) Prepare(context.Context, ReportForwardingInput) (ForwardingMetadata, error) {
	schemaVersion := reportForwardingSchemaVersion
	return ForwardingMetadata{
		Status:        "prepared_not_submitted",
		SchemaVersion: &schemaVersion,
		PreparedAt:    f.now().UTC(),
	}, nil
}
