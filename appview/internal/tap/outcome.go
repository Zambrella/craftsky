package tap

import (
	"context"
	"encoding/json"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// MaxFrameBytes is the largest Tap WebSocket frame the consumer accepts. Any
// durable quarantine replay payload must obey the same bound so every ACKed
// invalid frame can re-enter the normal decoder without a second size policy.
const MaxFrameBytes = 2 << 20

// OutcomeKind is the closed result vocabulary shared by the Tap boundary,
// durable ingestion, and projection workers.
type OutcomeKind string

const (
	OutcomeApplied          OutcomeKind = "applied"
	OutcomeBlocked          OutcomeKind = "blocked"
	OutcomePermanentInvalid OutcomeKind = "permanent_invalid"
	OutcomeRetryable        OutcomeKind = "retryable"
)

// ReasonCode is safe, bounded operational metadata. Raw errors and record
// payloads must never be used as reason codes or metric labels.
type ReasonCode string

const (
	ReasonNone                      ReasonCode = "none"
	ReasonInvalidEnvelope           ReasonCode = "invalid_envelope"
	ReasonMissingRecord             ReasonCode = "missing_record"
	ReasonInvalidDID                ReasonCode = "invalid_did"
	ReasonInvalidCollection         ReasonCode = "invalid_collection"
	ReasonInvalidRecordKey          ReasonCode = "invalid_record_key"
	ReasonUnsupportedAction         ReasonCode = "unsupported_action"
	ReasonMalformedRecord           ReasonCode = "malformed_record"
	ReasonRecordTooLarge            ReasonCode = "record_too_large"
	ReasonUnsupportedEventType      ReasonCode = "unsupported_event_type"
	ReasonInvalidIdentity           ReasonCode = "invalid_identity"
	ReasonUnsupportedIdentityStatus ReasonCode = "unsupported_identity_status"
	ReasonMissingMember             ReasonCode = "missing_member"
	ReasonMissingSubject            ReasonCode = "missing_subject"
	ReasonOwnerDeparted             ReasonCode = "owner_departed"
	ReasonOwnerTerminal             ReasonCode = "owner_terminal"
	ReasonStaleSource               ReasonCode = "stale_source"
	ReasonSourceOrderUncertain      ReasonCode = "source_order_uncertain"
	ReasonStorageUnavailable        ReasonCode = "storage_unavailable"
	ReasonLeaseLost                 ReasonCode = "lease_lost"
	ReasonProjectionFailure         ReasonCode = "projection_failure"
	ReasonDurableIngestorRequired   ReasonCode = "durable_ingestor_required"
)

// Dependency identifies precise blocked work. Key is a canonical DID or AT
// URI and is stored only in the database, never as a metric label.
type Dependency struct {
	Kind string
	Key  string
}

// Outcome describes whether processing is durably complete for the Tap ACK
// boundary. Applied, Blocked, and PermanentInvalid are acknowledgable only
// because their callers have already committed source/job or quarantine state.
type Outcome struct {
	Kind       OutcomeKind
	Reason     ReasonCode
	Dependency Dependency
}

func Applied() Outcome { return Outcome{Kind: OutcomeApplied, Reason: ReasonNone} }

func Blocked(reason ReasonCode, dependency Dependency) Outcome {
	return Outcome{Kind: OutcomeBlocked, Reason: reason, Dependency: dependency}
}

func PermanentInvalid(reason ReasonCode) Outcome {
	return Outcome{Kind: OutcomePermanentInvalid, Reason: reason}
}

func Retryable(reason ReasonCode) Outcome {
	return Outcome{Kind: OutcomeRetryable, Reason: reason}
}

func (outcome Outcome) Acknowledgable() bool {
	switch outcome.Kind {
	case OutcomeApplied, OutcomeBlocked, OutcomePermanentInvalid:
		return true
	default:
		return false
	}
}

// IdentityEvent is the validated Tap v0.1.10 identity envelope. Deleted is
// the irreversible terminal action; the other documented statuses remain
// durable ordinary identity observations.
type IdentityEvent struct {
	ID       uint64
	DID      syntax.DID
	Handle   string
	IsActive bool
	Status   string
}

// InvalidEvent contains one bounded raw Tap frame for deterministic input
// defects. Persistence keeps the exact bytes for replay while exposing only
// separately bounded diagnostic evidence to operator listing commands.
type InvalidEvent struct {
	ID       uint64
	Type     string
	Reason   ReasonCode
	Envelope json.RawMessage
}

// DurableIngestor owns the ACK contract. A nil error alone is insufficient:
// it must return an acknowledgable outcome only after committing the durable
// source/job, identity lifecycle result, or quarantine row represented by it.
type DurableIngestor interface {
	IngestRecord(context.Context, Event) (Outcome, error)
	IngestIdentity(context.Context, IdentityEvent) (Outcome, error)
	Quarantine(context.Context, InvalidEvent) (Outcome, error)
}
