package ownerlifecycle

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Lifecycle struct {
	Owner            syntax.DID
	State            State
	Generation       int64
	AuthEpoch        int64
	TransitionReason string
	TransitionedAt   time.Time
	TerminalAt       *time.Time
	PurgeCompletedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TransitionRequest struct {
	Owner              syntax.DID
	ExpectedGeneration int64
	To                 State
	Reason             string
}

// TransitionParticipant runs after the lifecycle row is locked and updated,
// but before cleanup/outbox rows and commit. It may update later row-lock
// classes using tx; returning an error rolls the whole transition back.
type TransitionParticipant func(
	ctx context.Context,
	tx pgx.Tx,
	before Lifecycle,
	after Lifecycle,
) error

type ExpectedOwner struct {
	Owner      syntax.DID
	Generation int64
	// AllowMissing is a fence-only expectation for a target DID that was not
	// known to the AppView when an effect scope was resolved. Generation must
	// be zero. The DID is still canonically locked and must remain absent when
	// the remote-effect boundary is crossed.
	AllowMissing bool
}

type PurgeComponent struct {
	Component string
	DIDRole   string
}

type TerminalizeRequest struct {
	Owner      syntax.DID
	Reason     string
	Components []PurgeComponent
}

// TerminalParticipant composes auth/session invalidation with the terminal
// tombstone commit. Before is nil when the terminal identity event is the
// first AppView knowledge of the DID.
type TerminalParticipant func(
	ctx context.Context,
	tx pgx.Tx,
	before *Lifecycle,
	terminal Lifecycle,
) error

type PurgeState string

const (
	PurgePending  PurgeState = "pending"
	PurgeRunning  PurgeState = "running"
	PurgeComplete PurgeState = "complete"
)

type PurgeClaimRequest struct {
	Worker        string
	LeaseToken    uuid.UUID
	LeaseDuration time.Duration
	Limit         int
}

type PurgeClaim struct {
	Owner           syntax.DID
	OwnerGeneration int64
	Component       string
	DIDRole         string
	State           PurgeState
	Attempts        int
	LeaseOwner      string
	LeaseToken      uuid.UUID
	LeaseExpiresAt  time.Time
}

type EffectKind string

const (
	EffectPDSRecord    EffectKind = "pds_record"
	EffectObjectPut    EffectKind = "object_put"
	EffectObjectDelete EffectKind = "object_delete"
)

func (kind EffectKind) valid() bool {
	switch kind {
	case EffectPDSRecord, EffectObjectPut, EffectObjectDelete:
		return true
	default:
		return false
	}
}

type EffectAction string

const (
	EffectActionPutRecord    EffectAction = "put_record"
	EffectActionDeleteRecord EffectAction = "delete_record"
	EffectActionUploadBlob   EffectAction = "upload_blob"
	EffectActionPutObject    EffectAction = "put_object"
	EffectActionDeleteObject EffectAction = "delete_object"
)

func (action EffectAction) validFor(kind EffectKind) bool {
	switch kind {
	case EffectPDSRecord:
		return action == EffectActionPutRecord || action == EffectActionDeleteRecord
	case EffectObjectPut:
		return action == EffectActionPutObject || action == EffectActionUploadBlob
	case EffectObjectDelete:
		return action == EffectActionDeleteObject
	default:
		return false
	}
}

func defaultEffectAction(kind EffectKind) EffectAction {
	switch kind {
	case EffectPDSRecord:
		return EffectActionPutRecord
	case EffectObjectPut:
		return EffectActionPutObject
	case EffectObjectDelete:
		return EffectActionDeleteObject
	default:
		return ""
	}
}

type EffectOutcome string

const (
	OutcomePrepared               EffectOutcome = "prepared"
	OutcomeDispatched             EffectOutcome = "dispatched"
	OutcomeAccepted               EffectOutcome = "accepted"
	OutcomeRejected               EffectOutcome = "rejected"
	OutcomeAbandonedPreTransition EffectOutcome = "abandoned_pre_transition"
	OutcomeUnknownPreTransition   EffectOutcome = "outcome_unknown_pre_transition"
	OutcomeReconciledAccepted     EffectOutcome = "reconciled_accepted"
	OutcomeReconciledNotAccepted  EffectOutcome = "reconciled_not_accepted"
	OutcomeReconciliationMismatch EffectOutcome = "reconciliation_mismatch"
)

type ProjectionDisposition string

const (
	ProjectionPending         ProjectionDisposition = "pending"
	ProjectionHiddenNonActive ProjectionDisposition = "hidden_non_active"
	ProjectionEligibleCurrent ProjectionDisposition = "eligible_current"
	ProjectionDeniedTerminal  ProjectionDisposition = "denied_terminal"
	ProjectionNotApplicable   ProjectionDisposition = "not_applicable"
)

type NewEffectAttempt struct {
	OperationID        string
	Owner              syntax.DID
	OwnerGeneration    int64
	Kind               EffectKind
	Action             EffectAction
	MutationKey        string
	DeterministicKey   string
	RequestFingerprint [32]byte
	ExpectedCID        string
	RemoteDeadline     time.Time
}

type EffectAttempt struct {
	OperationID           string
	Owner                 syntax.DID
	OwnerGeneration       int64
	Kind                  EffectKind
	Action                EffectAction
	MutationKey           string
	DeterministicKey      string
	RequestFingerprint    [32]byte
	ExpectedCID           string
	ResultCID             string
	Outcome               EffectOutcome
	ProjectionDisposition ProjectionDisposition
	RepeatForbidden       bool
	RemoteDeadline        time.Time
	DispatchedAt          *time.Time
	CompletedAt           *time.Time
	ReconciledAt          *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ReconcileEffectRequest struct {
	OperationID string
	Owner       syntax.DID
	Outcome     EffectOutcome
	ResultCID   string
}
