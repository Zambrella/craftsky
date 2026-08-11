package accountdeletion

import (
	"context"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// OperationalSource represents values available at the deletion boundary.
// NewOperationalState is the allow-list that prevents unrelated account data
// from being copied into durable deletion state.
type OperationalSource struct {
	JobID                string
	DID                  syntax.DID
	AcceptedAt           time.Time
	Status               Status
	Phase                Phase
	Attempt              int
	OAuthSessionID       string
	StatusCapabilityHash string
	ExpectedURIs         []string
	ReceiptIDs           []string
	Handle               string
	RecordContent        string
	RelationshipData     string
	SettingsData         string
	ImportData           string
	FullURL              string
}

type OperationalState struct {
	JobID                string     `json:"jobId"`
	DID                  syntax.DID `json:"did"`
	AcceptedAt           time.Time  `json:"acceptedAt"`
	Status               Status     `json:"status"`
	Phase                Phase      `json:"phase"`
	Attempt              int        `json:"attempt"`
	OAuthSessionID       string     `json:"oauthSessionId"`
	StatusCapabilityHash string     `json:"statusCapabilityHash"`
	ExpectedURIs         []string   `json:"expectedUris"`
	ReceiptIDs           []string   `json:"receiptIds"`
}

func NewOperationalState(source OperationalSource) OperationalState {
	return OperationalState{
		JobID:                source.JobID,
		DID:                  source.DID,
		AcceptedAt:           source.AcceptedAt,
		Status:               source.Status,
		Phase:                source.Phase,
		Attempt:              source.Attempt,
		OAuthSessionID:       source.OAuthSessionID,
		StatusCapabilityHash: source.StatusCapabilityHash,
		ExpectedURIs:         append([]string(nil), source.ExpectedURIs...),
		ReceiptIDs:           append([]string(nil), source.ReceiptIDs...),
	}
}

func FinalizeOperationalState(state OperationalState, terminalAt time.Time) DeletionAudit {
	return NewDeletionAudit(AuditSource{
		JobID:      state.JobID,
		DID:        state.DID,
		AcceptedAt: state.AcceptedAt,
	}, terminalAt, AuditOutcomeDeleted)
}

type ErrorCategory string

const (
	ErrorCategoryReauthentication  ErrorCategory = "reauthentication"
	ErrorCategoryValidation        ErrorCategory = "validation"
	ErrorCategoryAcceptance        ErrorCategory = "acceptance"
	ErrorCategorySessionRevocation ErrorCategory = "sessionRevocation"
	ErrorCategoryPDS               ErrorCategory = "pds"
	ErrorCategoryPrivateCleanup    ErrorCategory = "privateCleanup"
	ErrorCategoryInstagramCleanup  ErrorCategory = "instagramCleanup"
	ErrorCategoryIndexer           ErrorCategory = "indexerConvergence"
	ErrorCategoryRetry             ErrorCategory = "retry"
	ErrorCategoryTerminal          ErrorCategory = "terminal"
	ErrorCategoryAuditExpiry       ErrorCategory = "auditExpiry"
)

func TelemetryAttributes(phase Phase, outcome AuditOutcome, category ErrorCategory) map[string]string {
	return map[string]string{
		"phase":         string(phase),
		"outcome":       string(outcome),
		"errorCategory": string(category),
	}
}

type DeletionMetricEvent struct {
	Event         string
	Phase         Phase
	Outcome       AuditOutcome
	ErrorCategory ErrorCategory
	Duration      time.Duration
}

type DeletionMetricRecorder interface {
	RecordDeletionMetric(ctx context.Context, event DeletionMetricEvent)
}

type DeletionTelemetry struct {
	logger  *slog.Logger
	metrics DeletionMetricRecorder
}

func NewDeletionTelemetry(logger *slog.Logger, metrics DeletionMetricRecorder) *DeletionTelemetry {
	if logger == nil {
		logger = slog.Default()
	}
	return &DeletionTelemetry{logger: logger, metrics: metrics}
}

func (telemetry *DeletionTelemetry) Accepted(ctx context.Context) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "accepted"})
}

func (telemetry *DeletionTelemetry) Phase(ctx context.Context, phase Phase) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "phase", Phase: phase})
}

func (telemetry *DeletionTelemetry) AutomaticRetry(ctx context.Context, phase Phase, category ErrorCategory) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "automaticRetry", Phase: phase, ErrorCategory: category})
}

func (telemetry *DeletionTelemetry) ManualRetry(ctx context.Context, phase Phase) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "manualRetry", Phase: phase})
}

func (telemetry *DeletionTelemetry) NeedsAttention(ctx context.Context, phase Phase, category ErrorCategory) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "needsAttention", Phase: phase, ErrorCategory: category})
}

func (telemetry *DeletionTelemetry) ConvergenceDelay(ctx context.Context, duration time.Duration) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "convergenceDelay", Phase: PhaseWaitingForIndexerConvergence, Duration: duration})
}

func (telemetry *DeletionTelemetry) TerminalSuccess(ctx context.Context) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "terminalSuccess", Outcome: AuditOutcomeDeleted})
}

func (telemetry *DeletionTelemetry) AuditExpired(ctx context.Context) {
	telemetry.record(ctx, DeletionMetricEvent{Event: "auditExpired", ErrorCategory: ErrorCategoryAuditExpiry})
}

func (telemetry *DeletionTelemetry) record(ctx context.Context, event DeletionMetricEvent) {
	attributes := []slog.Attr{slog.String("event", event.Event)}
	if event.Phase != "" {
		attributes = append(attributes, slog.String("phase", string(event.Phase)))
	}
	if event.Outcome != "" {
		attributes = append(attributes, slog.String("outcome", string(event.Outcome)))
	}
	if event.ErrorCategory != "" {
		attributes = append(attributes, slog.String("error_category", string(event.ErrorCategory)))
	}
	if event.Duration > 0 {
		attributes = append(attributes, slog.Duration("duration", event.Duration))
	}
	telemetry.logger.LogAttrs(ctx, slog.LevelInfo, "account deletion lifecycle", attributes...)
	if telemetry.metrics != nil {
		telemetry.metrics.RecordDeletionMetric(ctx, event)
	}
}
