package instagram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
)

const (
	defaultReconciliationLeaseDuration = time.Minute
	defaultReconciliationMaxAttempts   = 5
	maxReconciliationBatchSize         = 500
)

// InstagramMatchNotificationService is the narrow transactional seam shared
// by suggestion storage and reconciliation. notifications.Service implements
// it directly.
type InstagramMatchNotificationService interface {
	ActivateInstagramMatch(context.Context, pgx.Tx, notifications.InstagramMatchActivation) error
	RetractInstagramMatch(context.Context, pgx.Tx, notifications.InstagramMatchRetraction) error
}

var _ InstagramMatchNotificationService = (*notifications.Service)(nil)

type ReconciliationWorkerOptions struct {
	Pool          *pgxpool.Pool
	Store         *SuggestionStore
	Policy        InstagramSuggestionEligibilityPolicy
	Notifications InstagramMatchNotificationService
	Now           func() time.Time
	NewID         func() uuid.UUID
	LeaseDuration time.Duration
	MaxAttempts   int
}

// ReconciliationWorker claims durable targeted jobs and replays them safely.
// Initial import matching deliberately does not use this worker and therefore
// never creates a system notification.
type ReconciliationWorker struct {
	pool          *pgxpool.Pool
	store         *SuggestionStore
	policy        InstagramSuggestionEligibilityPolicy
	notifications InstagramMatchNotificationService
	now           func() time.Time
	newID         func() uuid.UUID
	leaseDuration time.Duration
	maxAttempts   int
}

func NewReconciliationWorker(options ReconciliationWorkerOptions) (*ReconciliationWorker, error) {
	if options.Pool == nil || options.Store == nil || options.Policy == nil || options.Notifications == nil {
		return nil, errors.New("Instagram reconciliation dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewID == nil {
		options.NewID = uuid.New
	}
	if options.LeaseDuration == 0 {
		options.LeaseDuration = defaultReconciliationLeaseDuration
	}
	if options.MaxAttempts == 0 {
		options.MaxAttempts = defaultReconciliationMaxAttempts
	}
	if options.LeaseDuration <= 0 || options.MaxAttempts < 1 || options.MaxAttempts > defaultReconciliationMaxAttempts {
		return nil, errors.New("invalid Instagram reconciliation limits")
	}
	// The same store is also used by member-facing suggestion actions. Binding
	// the service here guarantees those terminal transitions participate in the
	// notification transaction even when callers used the compatible one-arg
	// store constructor before assembling the worker.
	if options.Store.notifications == nil {
		options.Store.notifications = options.Notifications
	}
	return &ReconciliationWorker{
		pool: options.Pool, store: options.Store, policy: options.Policy,
		notifications: options.Notifications, now: options.Now, newID: options.NewID,
		leaseDuration: options.LeaseDuration, maxAttempts: options.MaxAttempts,
	}, nil
}

type reconciliationJob struct {
	ID         uuid.UUID
	OwnerDID   syntax.DID
	TargetDID  *syntax.DID
	LinkID     *uuid.UUID
	ImportID   *uuid.UUID
	Reason     string
	Attempts   int
	LeaseToken uuid.UUID
}

type reconciliationCandidate struct {
	ImporterDID syntax.DID
	ImportID    uuid.UUID
	Username    string
	TargetDID   syntax.DID
}

// ProcessBatch claims at most limit queued/retryable jobs with SKIP LOCKED,
// processes each independently, and returns the number claimed. A job-level
// error is persisted as retryable/failed and also returned for observability;
// already committed suggestions remain safe for the idempotent replay.
func (w *ReconciliationWorker) ProcessBatch(ctx context.Context, limit int) (int, error) {
	if w == nil || w.pool == nil || w.store == nil || w.policy == nil || w.notifications == nil {
		return 0, errors.New("Instagram reconciliation worker is unavailable")
	}
	if limit < 1 || limit > maxReconciliationBatchSize {
		return 0, errors.New("invalid Instagram reconciliation batch size")
	}
	now := w.now().UTC()
	jobs, err := w.claimJobs(ctx, limit, now)
	if err != nil {
		return 0, err
	}
	var processingErrors []error
	for _, job := range jobs {
		matched, processErr := w.processJob(ctx, job, now)
		if processErr != nil {
			markCtx := context.WithoutCancel(ctx)
			if markErr := w.markJobFailure(markCtx, job, now); markErr != nil {
				processingErrors = append(processingErrors, fmt.Errorf("persist Instagram reconciliation failure: %w", markErr))
			}
			processingErrors = append(processingErrors, fmt.Errorf("process Instagram reconciliation job: %w", processErr))
			continue
		}
		status := "ignored"
		if matched {
			status = "completed"
		}
		if err := w.markJobTerminal(ctx, job, status, now); err != nil {
			processingErrors = append(processingErrors, fmt.Errorf("complete Instagram reconciliation job: %w", err))
		}
	}
	return len(jobs), errors.Join(processingErrors...)
}

func (w *ReconciliationWorker) claimJobs(ctx context.Context, limit int, now time.Time) ([]reconciliationJob, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status=CASE WHEN attempts >= $2 THEN 'failed' ELSE 'retryable' END,
		    next_attempt_at=$1,
		    terminal_at=CASE WHEN attempts >= $2 THEN COALESCE(terminal_at,$1) ELSE NULL END,
		    lease_token=NULL, lease_expires_at=NULL, updated_at=$1
		WHERE status='processing' AND lease_expires_at <= $1
	`, now, w.maxAttempts); err != nil {
		return nil, fmt.Errorf("recover expired Instagram reconciliation leases: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT id, owner_did, target_did, link_id, import_id, reason, attempts
		FROM instagram_reconciliation_jobs
		WHERE status IN ('queued','retryable')
		  AND attempts < $2
		  AND next_attempt_at <= $3
		ORDER BY next_attempt_at, id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit, w.maxAttempts, now)
	if err != nil {
		return nil, fmt.Errorf("claim Instagram reconciliation jobs: %w", err)
	}
	jobs := make([]reconciliationJob, 0, limit)
	for rows.Next() {
		var (
			job      reconciliationJob
			target   sql.NullString
			linkID   uuid.NullUUID
			importID uuid.NullUUID
		)
		if err := rows.Scan(&job.ID, &job.OwnerDID, &target, &linkID, &importID, &job.Reason, &job.Attempts); err != nil {
			rows.Close()
			return nil, err
		}
		if target.Valid {
			parsed := syntax.DID(target.String)
			job.TargetDID = &parsed
		}
		if linkID.Valid {
			parsed := linkID.UUID
			job.LinkID = &parsed
		}
		if importID.Valid {
			parsed := importID.UUID
			job.ImportID = &parsed
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range jobs {
		jobs[index].Attempts++
		jobs[index].LeaseToken = w.newID()
		if jobs[index].LeaseToken == uuid.Nil {
			return nil, errors.New("invalid Instagram reconciliation lease token")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_reconciliation_jobs
			SET status='processing', attempts=$2, lease_token=$3,
			    lease_expires_at=$4, terminal_at=NULL, updated_at=$5
			WHERE id=$1
		`, jobs[index].ID, jobs[index].Attempts, jobs[index].LeaseToken, now.Add(w.leaseDuration), now); err != nil {
			return nil, fmt.Errorf("lease Instagram reconciliation job: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (w *ReconciliationWorker) processJob(ctx context.Context, job reconciliationJob, now time.Time) (bool, error) {
	candidates, err := w.loadCandidates(ctx, job, now)
	if err != nil {
		return false, err
	}
	matched := false
	for _, candidate := range candidates {
		request := SuggestionEligibilityRequest{
			ImporterDID:      candidate.ImporterDID,
			TargetDID:        candidate.TargetDID,
			ImportedUsername: candidate.Username,
		}
		decision, err := w.policy.Evaluate(ctx, EligibilityAtMatch, request)
		if err != nil {
			return matched, err
		}
		if !decision.Eligible {
			continue
		}

		committed, err := w.persistCandidate(ctx, candidate, request, now)
		if err != nil {
			return matched, err
		}
		matched = matched || committed
	}
	return matched, nil
}

func (w *ReconciliationWorker) persistCandidate(ctx context.Context, candidate reconciliationCandidate, request SuggestionEligibilityRequest, now time.Time) (bool, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	decision, err := w.policy.Evaluate(ctx, EligibilityAtPersist, request)
	if err != nil {
		return false, err
	}
	if !decision.Eligible {
		return false, nil
	}
	upsert, err := w.store.upsertPendingSuggestionTx(ctx, tx, UpsertSuggestionParams{
		ID: w.newID(), ImporterDID: candidate.ImporterDID,
		TargetDID: candidate.TargetDID, ImportID: candidate.ImportID,
		Username: candidate.Username, Now: now,
	})
	if errors.Is(err, ErrInstagramResourceNotFound) {
		// The retained handle or link changed after candidate discovery. The
		// exact locked persistence check wins and a later targeted job can
		// reconsider the new state.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if upsert.Created {
		decision, err = w.policy.Evaluate(ctx, EligibilityAtNotificationCreate, request)
		if err != nil {
			return false, err
		}
		if !decision.Eligible {
			return false, nil
		}
		if err := w.notifications.ActivateInstagramMatch(ctx, tx, notifications.InstagramMatchActivation{
			RecipientDID: candidate.ImporterDID,
			SuggestionID: upsert.ID,
			ActivityAt:   now,
		}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return upsert.Supported, nil
}

func (w *ReconciliationWorker) loadCandidates(ctx context.Context, job reconciliationJob, now time.Time) ([]reconciliationCandidate, error) {
	if job.OwnerDID == "" {
		return nil, nil
	}
	query := `
		SELECT DISTINCT i.owner_did, i.id, h.username_normalized, link.owner_did
		FROM instagram_graph_imports i
		JOIN instagram_graph_handles h
		  ON h.import_id=i.id
		JOIN instagram_account_links link
		  ON link.username_normalized=h.username_normalized
		 AND link.state='active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE i.state='active'
		  AND (i.retention_expires_at IS NULL OR i.retention_expires_at>$1)
		  AND (h.retain_until IS NULL OR h.retain_until>$1)`
	args := []any{now}
	switch {
	case job.LinkID != nil:
		query += ` AND link.id=$2 AND link.owner_did=$3`
		args = append(args, *job.LinkID, job.OwnerDID)
	case job.ImportID != nil:
		query += ` AND i.id=$2 AND i.owner_did=$3`
		args = append(args, *job.ImportID, job.OwnerDID)
		if job.TargetDID != nil {
			query += ` AND link.owner_did=$4`
			args = append(args, *job.TargetDID)
		}
	case job.TargetDID != nil:
		query += ` AND i.owner_did=$2 AND link.owner_did=$3`
		args = append(args, job.OwnerDID, *job.TargetDID)
	default:
		return nil, nil
	}
	query += ` ORDER BY i.owner_did, i.id, h.username_normalized, link.owner_did`
	rows, err := w.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load Instagram reconciliation candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]reconciliationCandidate, 0)
	for rows.Next() {
		var candidate reconciliationCandidate
		if err := rows.Scan(&candidate.ImporterDID, &candidate.ImportID, &candidate.Username, &candidate.TargetDID); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (w *ReconciliationWorker) markJobTerminal(ctx context.Context, job reconciliationJob, status string, now time.Time) error {
	if status != "completed" && status != "ignored" {
		return errors.New("invalid Instagram reconciliation terminal state")
	}
	_, err := w.pool.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status=$3, terminal_at=COALESCE(terminal_at,$4),
		    lease_token=NULL, lease_expires_at=NULL, updated_at=$4
		WHERE id=$1 AND status='processing' AND lease_token=$2
	`, job.ID, job.LeaseToken, status, now)
	return err
}

func (w *ReconciliationWorker) markJobFailure(ctx context.Context, job reconciliationJob, now time.Time) error {
	status := "retryable"
	terminalAt := any(nil)
	nextAttemptAt := now.Add(reconciliationRetryDelay(job.Attempts))
	if job.Attempts >= w.maxAttempts {
		status = "failed"
		terminalAt = now
		nextAttemptAt = now
	}
	_, err := w.pool.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status=$3, next_attempt_at=$4, terminal_at=$5,
		    lease_token=NULL, lease_expires_at=NULL, updated_at=$6
		WHERE id=$1 AND status='processing' AND lease_token=$2
	`, job.ID, job.LeaseToken, status, nextAttemptAt, terminalAt, now)
	return err
}

func reconciliationRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Second << min(attempt-1, 8)
	return min(delay, 5*time.Minute)
}

func (w *ReconciliationWorker) String() string {
	if w == nil {
		return "Instagram ReconciliationWorker{unavailable}"
	}
	return "Instagram ReconciliationWorker{database:configured,policy:configured,notifications:configured,clock:configured}"
}
