package instagram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxOperatorBatch = 500

var (
	ErrInvalidOperatorBatch        = errors.New("Instagram operator batch must be between 1 and 500")
	ErrInvalidOperatorRequest      = errors.New("invalid Instagram operator request")
	ErrOperatorResourceNotFound    = errors.New("Instagram operator resource not found")
	ErrOperatorConflictResolved    = errors.New("Instagram conflict already has a different resolution")
	ErrOperatorJobNotRetryable     = errors.New("Instagram job is not safely retryable")
	ErrOperatorConfigurationUnsafe = errors.New("Instagram operator key is unavailable")
)

type OperatorConflictResolution string

const (
	ResolutionKeepExisting   OperatorConflictResolution = "keepExisting"
	ResolutionRevokeExisting OperatorConflictResolution = "revokeExisting"
)

func (r OperatorConflictResolution) Valid() bool {
	return r == ResolutionKeepExisting || r == ResolutionRevokeExisting
}

func (r OperatorConflictResolution) State() InstagramConflictState {
	if r == ResolutionKeepExisting {
		return ConflictResolvedKeepExisting
	}
	if r == ResolutionRevokeExisting {
		return ConflictResolvedRevokeExisting
	}
	return ""
}

type OperatorConflict struct {
	ID        uuid.UUID
	State     InstagramConflictState
	OpenedAt  time.Time
	ExpiresAt time.Time
}

func (OperatorConflict) String() string     { return "Instagram operator conflict {opaque}" }
func (v OperatorConflict) GoString() string { return v.String() }
func (v OperatorConflict) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, v.String())
}

type OperatorConflictResult struct {
	ID      uuid.UUID
	State   InstagramConflictState
	Changed bool
}

func (OperatorConflictResult) String() string { return "Instagram operator conflict result {opaque}" }

type OperatorLinkResult struct {
	ID      uuid.UUID
	State   InstagramLinkState
	Changed bool
}

func (OperatorLinkResult) String() string { return "Instagram operator link result {opaque}" }

type OperatorJobKind string

const (
	OperatorJobWebhook        OperatorJobKind = "webhook"
	OperatorJobReconciliation OperatorJobKind = "reconciliation"
)

func (k OperatorJobKind) Valid() bool {
	return k == OperatorJobWebhook || k == OperatorJobReconciliation
}

type OperatorJob struct {
	ID            uuid.UUID
	Kind          OperatorJobKind
	Status        string
	Attempts      int
	NextAttemptAt time.Time
	TerminalAt    *time.Time
	CreatedAt     time.Time
}

func (OperatorJob) String() string     { return "Instagram operator job {opaque}" }
func (v OperatorJob) GoString() string { return v.String() }
func (v OperatorJob) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, v.String())
}

type OperatorJobResult struct {
	ID      uuid.UUID
	Kind    OperatorJobKind
	Status  string
	Changed bool
}

func (OperatorJobResult) String() string { return "Instagram operator job result {opaque}" }

type OperatorService struct {
	pool *pgxpool.Pool
	key  []byte
	now  func() time.Time
}

func NewOperatorService(pool *pgxpool.Pool, key []byte, now func() time.Time) (*OperatorService, error) {
	if pool == nil || len(key) < 32 {
		return nil, ErrOperatorConfigurationUnsafe
	}
	if now == nil {
		now = time.Now
	}
	return &OperatorService{pool: pool, key: append([]byte(nil), key...), now: now}, nil
}

func (s *OperatorService) ListOpenConflicts(ctx context.Context, limit int, after uuid.UUID) ([]OperatorConflict, uuid.UUID, error) {
	if err := s.validateBatch(limit); err != nil {
		return nil, uuid.Nil, err
	}
	var cursorTime time.Time
	if after != uuid.Nil {
		if err := s.pool.QueryRow(ctx, `SELECT opened_at FROM instagram_link_conflicts WHERE id=$1`, after).Scan(&cursorTime); errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, ErrOperatorResourceNotFound
		} else if err != nil {
			return nil, uuid.Nil, fmt.Errorf("read Instagram conflict cursor: %w", err)
		}
	}
	query := `
		SELECT id,state,opened_at,expires_at
		FROM instagram_link_conflicts
		WHERE state='open'
		ORDER BY opened_at,id
		LIMIT $1`
	args := []any{limit + 1}
	if after != uuid.Nil {
		query = `
			SELECT id,state,opened_at,expires_at
			FROM instagram_link_conflicts
			WHERE state='open' AND (opened_at,id)>($2,$3)
			ORDER BY opened_at,id
			LIMIT $1`
		args = append(args, cursorTime, after)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("list Instagram conflicts: %w", err)
	}
	defer rows.Close()
	items := make([]OperatorConflict, 0, limit+1)
	for rows.Next() {
		var item OperatorConflict
		if err := rows.Scan(&item.ID, &item.State, &item.OpenedAt, &item.ExpiresAt); err != nil {
			return nil, uuid.Nil, fmt.Errorf("scan Instagram conflict: %w", err)
		}
		if item.State != ConflictOpen {
			return nil, uuid.Nil, ErrInvalidInstagramState
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, uuid.Nil, fmt.Errorf("iterate Instagram conflicts: %w", err)
	}
	if len(items) <= limit {
		return items, uuid.Nil, nil
	}
	items = items[:limit]
	return items, items[len(items)-1].ID, nil
}

func (s *OperatorService) ResolveConflict(ctx context.Context, id uuid.UUID, resolution OperatorConflictResolution) (OperatorConflictResult, error) {
	if s == nil || s.pool == nil || id == uuid.Nil || !resolution.Valid() {
		return OperatorConflictResult{}, ErrInvalidOperatorRequest
	}
	now := s.now().UTC()
	result := OperatorConflictResult{ID: id, State: resolution.State()}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var state InstagramConflictState
		var existingLinkID *uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT state,existing_link_id
			FROM instagram_link_conflicts WHERE id=$1 FOR UPDATE
		`, id).Scan(&state, &existingLinkID); errors.Is(err, pgx.ErrNoRows) {
			return ErrOperatorResourceNotFound
		} else if err != nil {
			return fmt.Errorf("read Instagram conflict: %w", err)
		}
		if !state.Valid() {
			return ErrInvalidInstagramState
		}
		if state == resolution.State() {
			return nil
		}
		if state != ConflictOpen {
			return ErrOperatorConflictResolved
		}
		owner := ""
		if existingLinkID != nil {
			if err := tx.QueryRow(ctx, `SELECT owner_did FROM instagram_account_links WHERE id=$1`, *existingLinkID).Scan(&owner); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("read Instagram conflict owner: %w", err)
			}
		}
		if resolution == ResolutionRevokeExisting && existingLinkID != nil {
			if _, _, err := s.revokeLinkTx(ctx, tx, *existingLinkID, now, true); err != nil && !errors.Is(err, ErrOperatorResourceNotFound) {
				return err
			}
		}
		noteDigest := s.resolutionDigest(resolution)
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_link_conflicts
			SET state=$2,existing_link_id=NULL,claimant_attempt_id=NULL,
			    claimant_link_id=NULL,igsid_digest_version=NULL,igsid_digest=NULL,
			    resolution_note_digest=$3,resolved_at=$4,updated_at=$4
			WHERE id=$1 AND state='open'
		`, id, resolution.State(), noteDigest, now); err != nil {
			return fmt.Errorf("resolve Instagram conflict: %w", err)
		}
		if err := refreshConflictFlags(ctx, tx, now); err != nil {
			return err
		}
		if err := insertOperatorAudit(ctx, tx, owner, "operator_conflict_resolved", "conflict", id.String(), string(resolution), now); err != nil {
			return err
		}
		result.Changed = true
		return nil
	})
	return result, err
}

func (s *OperatorService) RevokeLink(ctx context.Context, id uuid.UUID) (OperatorLinkResult, error) {
	if s == nil || s.pool == nil || id == uuid.Nil {
		return OperatorLinkResult{}, ErrInvalidOperatorRequest
	}
	now := s.now().UTC()
	result := OperatorLinkResult{ID: id, State: LinkRevoked}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		_, changed, err := s.revokeLinkTx(ctx, tx, id, now, true)
		result.Changed = changed
		return err
	})
	return result, err
}

func (s *OperatorService) revokeLinkTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time, audit bool) (string, bool, error) {
	var owner string
	var state InstagramLinkState
	if err := tx.QueryRow(ctx, `SELECT owner_did,state FROM instagram_account_links WHERE id=$1 FOR UPDATE`, id).Scan(&owner, &state); errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrOperatorResourceNotFound
	} else if err != nil {
		return "", false, fmt.Errorf("read operator Instagram link: %w", err)
	}
	if !state.Valid() {
		return "", false, ErrInvalidInstagramState
	}
	if state.Terminal() {
		return owner, false, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links
		SET state='revoked',igsid=NULL,username=NULL,username_normalized=NULL,
		    discoverable=false,conflict_pending=false,membership_inactive_at=NULL,
		    revoked_at=$2,raw_identity_purge_at=$2::timestamptz+interval '90 days',updated_at=$2
		WHERE id=$1
	`, id, now); err != nil {
		return "", false, fmt.Errorf("operator revoke Instagram link: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_identity_claims
		SET state='revoked',released_at=COALESCE(released_at,$2),
		    anonymize_at=$2::timestamptz+interval '90 days',updated_at=$2
		WHERE link_id=$1 AND state IN ('active','disputed')
	`, id, now); err != nil {
		return "", false, fmt.Errorf("operator release Instagram claim: %w", err)
	}
	suggestionIDs, err := retentionUUIDs(ctx, tx, `
		UPDATE instagram_follow_suggestions
		SET state='invalidated',accepting_since=NULL,
		    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
		WHERE target_did=$1 AND state IN ('pending','accepting')
		RETURNING id
	`, owner, now)
	if err != nil {
		return "", false, fmt.Errorf("operator invalidate Instagram suggestions: %w", err)
	}
	if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "operatorLinkRevoked", now); err != nil {
		return "", false, err
	}
	if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "link_revoked", now); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status='ignored',lease_token=NULL,lease_expires_at=NULL,
		    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
		WHERE link_id=$1 AND status IN ('queued','processing','retryable')
	`, id, now); err != nil {
		return "", false, fmt.Errorf("operator cancel Instagram jobs: %w", err)
	}
	if audit {
		if err := insertOperatorAudit(ctx, tx, owner, "operator_link_revoked", "link", id.String(), "revoked", now); err != nil {
			return "", false, err
		}
	}
	return owner, true, nil
}

func (s *OperatorService) ListJobs(ctx context.Context, kind OperatorJobKind, limit int, after uuid.UUID) ([]OperatorJob, uuid.UUID, error) {
	if !kind.Valid() {
		return nil, uuid.Nil, ErrInvalidOperatorRequest
	}
	if err := s.validateBatch(limit); err != nil {
		return nil, uuid.Nil, err
	}
	var cursorTime time.Time
	if after != uuid.Nil {
		query := `SELECT created_at FROM instagram_webhook_work WHERE id=$1`
		if kind == OperatorJobReconciliation {
			query = `SELECT created_at FROM instagram_reconciliation_jobs WHERE id=$1`
		}
		if err := s.pool.QueryRow(ctx, query, after).Scan(&cursorTime); errors.Is(err, pgx.ErrNoRows) {
			return nil, uuid.Nil, ErrOperatorResourceNotFound
		} else if err != nil {
			return nil, uuid.Nil, fmt.Errorf("read Instagram job cursor: %w", err)
		}
	}
	query := `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_webhook_work ORDER BY created_at,id LIMIT $1`
	args := []any{limit + 1}
	if kind == OperatorJobReconciliation {
		query = `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_reconciliation_jobs ORDER BY created_at,id LIMIT $1`
	}
	if after != uuid.Nil {
		if kind == OperatorJobWebhook {
			query = `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_webhook_work WHERE (created_at,id)>($2,$3) ORDER BY created_at,id LIMIT $1`
		} else {
			query = `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_reconciliation_jobs WHERE (created_at,id)>($2,$3) ORDER BY created_at,id LIMIT $1`
		}
		args = append(args, cursorTime, after)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("list Instagram jobs: %w", err)
	}
	defer rows.Close()
	items := make([]OperatorJob, 0, limit+1)
	for rows.Next() {
		item := OperatorJob{Kind: kind}
		if err := rows.Scan(&item.ID, &item.Status, &item.Attempts, &item.NextAttemptAt, &item.TerminalAt, &item.CreatedAt); err != nil {
			return nil, uuid.Nil, fmt.Errorf("scan Instagram job: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, uuid.Nil, fmt.Errorf("iterate Instagram jobs: %w", err)
	}
	if len(items) <= limit {
		return items, uuid.Nil, nil
	}
	items = items[:limit]
	return items, items[len(items)-1].ID, nil
}

func (s *OperatorService) InspectJob(ctx context.Context, kind OperatorJobKind, id uuid.UUID) (OperatorJob, error) {
	if s == nil || s.pool == nil || !kind.Valid() || id == uuid.Nil {
		return OperatorJob{}, ErrInvalidOperatorRequest
	}
	query := `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_webhook_work WHERE id=$1`
	if kind == OperatorJobReconciliation {
		query = `SELECT id,status,attempts,next_attempt_at,terminal_at,created_at FROM instagram_reconciliation_jobs WHERE id=$1`
	}
	job := OperatorJob{Kind: kind}
	if err := s.pool.QueryRow(ctx, query, id).Scan(&job.ID, &job.Status, &job.Attempts, &job.NextAttemptAt, &job.TerminalAt, &job.CreatedAt); errors.Is(err, pgx.ErrNoRows) {
		return OperatorJob{}, ErrOperatorResourceNotFound
	} else if err != nil {
		return OperatorJob{}, fmt.Errorf("inspect Instagram job: %w", err)
	}
	return job, nil
}

func (s *OperatorService) RetryJob(ctx context.Context, kind OperatorJobKind, id uuid.UUID) (OperatorJobResult, error) {
	if s == nil || s.pool == nil || !kind.Valid() || id == uuid.Nil {
		return OperatorJobResult{}, ErrInvalidOperatorRequest
	}
	now := s.now().UTC()
	result := OperatorJobResult{ID: id, Kind: kind, Status: "queued"}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		query := `SELECT status,NULL::text FROM instagram_webhook_work WHERE id=$1 FOR UPDATE`
		if kind == OperatorJobReconciliation {
			query = `SELECT status,owner_did FROM instagram_reconciliation_jobs WHERE id=$1 FOR UPDATE`
		}
		var status string
		var owner *string
		if err := tx.QueryRow(ctx, query, id).Scan(&status, &owner); errors.Is(err, pgx.ErrNoRows) {
			return ErrOperatorResourceNotFound
		} else if err != nil {
			return fmt.Errorf("read retryable Instagram job: %w", err)
		}
		if status == "queued" {
			return nil
		}
		allowed := status == "retryable"
		if kind == OperatorJobReconciliation {
			allowed = status == "retryable" || status == "failed"
		}
		if !allowed {
			return ErrOperatorJobNotRetryable
		}
		if kind == OperatorJobWebhook {
			if _, err := tx.Exec(ctx, `
				UPDATE instagram_webhook_work
				SET status='queued',attempts=0,next_attempt_at=$2,
				    lease_token=NULL,lease_expires_at=NULL,terminal_at=NULL,
				    terminal_reason=NULL,updated_at=$2
				WHERE id=$1 AND status='retryable'
			`, id, now); err != nil {
				return fmt.Errorf("retry Instagram webhook job: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE instagram_reconciliation_jobs
				SET status='queued',attempts=0,next_attempt_at=$2,
				    lease_token=NULL,lease_expires_at=NULL,terminal_at=NULL,updated_at=$2
				WHERE id=$1 AND status IN ('retryable','failed')
			`, id, now); err != nil {
				return fmt.Errorf("retry Instagram reconciliation job: %w", err)
			}
		}
		ownerValue := ""
		if owner != nil {
			ownerValue = *owner
		}
		if err := insertOperatorAudit(ctx, tx, ownerValue, "operator_job_retried", "job", id.String(), string(kind), now); err != nil {
			return err
		}
		result.Changed = true
		return nil
	})
	return result, err
}

func (s *OperatorService) validateBatch(limit int) error {
	if s == nil || s.pool == nil {
		return ErrInvalidOperatorRequest
	}
	if limit < 1 || limit > MaxOperatorBatch {
		return ErrInvalidOperatorBatch
	}
	return nil
}

func (s *OperatorService) resolutionDigest(resolution OperatorConflictResolution) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte("instagram-operator-conflict-resolution:v1:" + string(resolution)))
	return mac.Sum(nil)
}

func insertOperatorAudit(ctx context.Context, tx pgx.Tx, owner, action, subjectKind, subjectID, outcome string, now time.Time) error {
	var ownerValue any
	if owner != "" {
		ownerValue = owner
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome,created_at)
		VALUES($1,$2,$3,$4,$5,$6)
	`, ownerValue, action, subjectKind, subjectID, outcome, now); err != nil {
		return fmt.Errorf("write Instagram operator audit: %w", err)
	}
	return nil
}

func (OperatorService) String() string {
	return "Instagram OperatorService{database:configured,key:[REDACTED],clock:configured}"
}

func (s OperatorService) GoString() string { return s.String() }
func (s OperatorService) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, s.String())
}
