package eligibility

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type State string

const (
	StateEligible   State = "eligible"
	StateRestricted State = "restricted"
)

type EvidenceKind string

const (
	EvidenceSelfDisclosure     EvidenceKind = "self_disclosure"
	EvidenceGuardianDisclosure EvidenceKind = "guardian_disclosure"
	EvidenceVerifiedAuthority  EvidenceKind = "verified_authority"
	EvidenceVerifiedAccount    EvidenceKind = "verified_account_record"
)

var ErrInvalidReview = errors.New("invalid age eligibility review")

type Status struct {
	State          State  `json:"state"`
	Appealable     bool   `json:"appealable"`
	AppealGuidance string `json:"appealGuidance,omitempty"`
}

type Review struct {
	AccountDID        syntax.DID
	State             State
	EvidenceKind      EvidenceKind
	EvidenceReference string
	ReviewerID        string
	Reason            string
	AppealGuidance    string
	ReviewedAt        time.Time
}

func (review Review) Validate() error {
	if review.AccountDID == "" || (review.State != StateEligible && review.State != StateRestricted) ||
		strings.TrimSpace(review.ReviewerID) == "" || strings.TrimSpace(review.Reason) == "" {
		return ErrInvalidReview
	}
	if review.State == StateRestricted {
		if !review.EvidenceKind.Allowed() || strings.TrimSpace(review.EvidenceReference) == "" || strings.TrimSpace(review.AppealGuidance) == "" {
			return ErrInvalidReview
		}
	}
	return nil
}

func (kind EvidenceKind) Allowed() bool {
	switch kind {
	case EvidenceSelfDisclosure, EvidenceGuardianDisclosure, EvidenceVerifiedAuthority, EvidenceVerifiedAccount:
		return true
	default:
		return false
	}
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (store *Store) Status(ctx context.Context, did syntax.DID) (Status, error) {
	var status Status
	var guidance *string
	err := store.pool.QueryRow(ctx, `
		SELECT state,appeal_guidance FROM account_age_eligibility WHERE account_did=$1
	`, did).Scan(&status.State, &guidance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Status{State: StateEligible}, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("read age eligibility: %w", err)
	}
	status.Appealable = status.State == StateRestricted
	if guidance != nil {
		status.AppealGuidance = *guidance
	}
	return status, nil
}

func (store *Store) Restricted(ctx context.Context, did syntax.DID) (bool, error) {
	status, err := store.Status(ctx, did)
	return status.State == StateRestricted, err
}

func (store *Store) Review(ctx context.Context, review Review) (Status, error) {
	if err := review.Validate(); err != nil {
		return Status{}, err
	}
	if review.ReviewedAt.IsZero() {
		review.ReviewedAt = time.Now().UTC()
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("begin age eligibility review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fromState := StateEligible
	revision := int64(1)
	var current State
	var currentRevision int64
	err = tx.QueryRow(ctx, `SELECT state,revision FROM account_age_eligibility WHERE account_did=$1 FOR UPDATE`, review.AccountDID).Scan(&current, &currentRevision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Status{}, fmt.Errorf("lock age eligibility: %w", err)
	}
	if err == nil {
		fromState = current
		revision = currentRevision + 1
	}

	var evidenceKind, evidenceReference, appealGuidance any
	if review.State == StateRestricted {
		evidenceKind, evidenceReference, appealGuidance = review.EvidenceKind, strings.TrimSpace(review.EvidenceReference), strings.TrimSpace(review.AppealGuidance)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_age_eligibility(
			account_did,state,evidence_kind,evidence_reference,reviewer_id,review_reason,
			appeal_guidance,revision,reviewed_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		ON CONFLICT(account_did) DO UPDATE SET
			state=excluded.state,evidence_kind=excluded.evidence_kind,
			evidence_reference=excluded.evidence_reference,reviewer_id=excluded.reviewer_id,
			review_reason=excluded.review_reason,appeal_guidance=excluded.appeal_guidance,
			revision=excluded.revision,reviewed_at=excluded.reviewed_at,updated_at=excluded.updated_at
	`, review.AccountDID, review.State, evidenceKind, evidenceReference, strings.TrimSpace(review.ReviewerID), strings.TrimSpace(review.Reason), appealGuidance, revision, review.ReviewedAt); err != nil {
		return Status{}, fmt.Errorf("save age eligibility: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_age_eligibility_events(
			id,account_did,revision,from_state,to_state,evidence_kind,evidence_reference,reviewer_id,reason,created_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, uuid.New(), review.AccountDID, revision, fromState, review.State, evidenceKind, evidenceReference, strings.TrimSpace(review.ReviewerID), strings.TrimSpace(review.Reason), review.ReviewedAt); err != nil {
		return Status{}, fmt.Errorf("append age eligibility event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Status{}, fmt.Errorf("commit age eligibility review: %w", err)
	}
	return Status{State: review.State, Appealable: review.State == StateRestricted, AppealGuidance: strings.TrimSpace(review.AppealGuidance)}, nil
}
