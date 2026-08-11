package accountdeletion

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type AuditOutcome string

const AuditOutcomeDeleted AuditOutcome = "deleted"

// AuditSource may contain the operational values needed while deletion is in
// progress. NewDeletionAudit intentionally projects only the minimized audit
// contract from it.
type AuditSource struct {
	JobID                string
	DID                  syntax.DID
	AcceptedAt           time.Time
	Handle               string
	OAuthSessionID       string
	StatusCapabilityHash string
	ExpectedURI          string
	RecordContent        string
}

type DeletionAudit struct {
	JobID      string       `json:"jobId"`
	DID        syntax.DID   `json:"did"`
	AcceptedAt time.Time    `json:"acceptedAt"`
	TerminalAt time.Time    `json:"terminalAt"`
	Outcome    AuditOutcome `json:"outcome"`
	ExpiresAt  time.Time    `json:"expiresAt"`
}

func NewDeletionAudit(source AuditSource, terminalAt time.Time, outcome AuditOutcome) DeletionAudit {
	return DeletionAudit{
		JobID:      source.JobID,
		DID:        source.DID,
		AcceptedAt: source.AcceptedAt,
		TerminalAt: terminalAt,
		Outcome:    outcome,
		ExpiresAt:  terminalAt.Add(30 * 24 * time.Hour),
	}
}

func AuditRetainedAt(audit DeletionAudit, now time.Time) bool {
	return now.Before(audit.ExpiresAt)
}

func AuditBlocksRejoin(DeletionAudit) bool {
	return false
}
