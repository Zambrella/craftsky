package instagram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
)

var (
	ErrInstagramSuggestionIneligible    = errors.New("Instagram suggestion ineligible")
	ErrInvalidInstagramSuggestionCursor = errors.New("invalid Instagram suggestion cursor")
)

type InstagramSuggestionReason string

const SuggestionReasonVerifiedInstagramFollow InstagramSuggestionReason = "verifiedInstagramFollow"

type FollowOperationStatus string

const (
	FollowOperationPending          FollowOperationStatus = "pending"
	FollowOperationWriting          FollowOperationStatus = "writing"
	FollowOperationSucceeded        FollowOperationStatus = "succeeded"
	FollowOperationAlreadyFollowing FollowOperationStatus = "alreadyFollowing"
	FollowOperationFailed           FollowOperationStatus = "failed"
)

type Suggestion struct {
	ID          uuid.UUID
	ImporterDID syntax.DID
	TargetDID   syntax.DID
	State       InstagramSuggestionState
	Reason      InstagramSuggestionReason
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Suggestion) String() string { return "Instagram suggestion [REDACTED]" }

type SuggestionEvidence struct {
	Suggestion       Suggestion
	ImportedUsername string
	Direction        ImportDirection
}

func (SuggestionEvidence) String() string { return "Instagram suggestion evidence [REDACTED]" }

type SuggestionCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type UpsertSuggestionParams struct {
	ID          uuid.UUID
	ImporterDID syntax.DID
	TargetDID   syntax.DID
	ImportID    uuid.UUID
	Username    string
	Now         time.Time
}

type FollowOperation struct {
	ID           uuid.UUID
	SuggestionID uuid.UUID
	OwnerDID     syntax.DID
	TargetDID    syntax.DID
	Rkey         syntax.RecordKey
	Status       FollowOperationStatus
	AttemptCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (FollowOperation) String() string { return "Instagram follow operation [REDACTED]" }

type AcceptanceClaim struct {
	Suggestion       Suggestion
	Operation        FollowOperation
	ImportedUsername string
	Direction        ImportDirection
}

func (AcceptanceClaim) String() string { return "Instagram acceptance claim [REDACTED]" }

type SuggestionStore struct {
	pool          *pgxpool.Pool
	notifications InstagramMatchNotificationService
}

// NewSuggestionStore accepts the notification service as an optional argument
// so storage-only tests and deployments migrating the core Instagram tables
// before the notification union can continue to use the store. Production
// construction supplies it, making every terminal transition retract digest
// support in the same transaction.
func NewSuggestionStore(pool *pgxpool.Pool, notificationServices ...InstagramMatchNotificationService) *SuggestionStore {
	var notificationService InstagramMatchNotificationService
	if len(notificationServices) > 0 {
		notificationService = notificationServices[0]
	}
	return &SuggestionStore{pool: pool, notifications: notificationService}
}

func (s *SuggestionStore) UpsertPendingSuggestion(ctx context.Context, params UpsertSuggestionParams) (bool, error) {
	if s == nil || s.pool == nil || params.ID == uuid.Nil || params.ImporterDID == "" || params.TargetDID == "" || params.ImporterDID == params.TargetDID || params.ImportID == uuid.Nil || params.Now.IsZero() {
		return false, errors.New("invalid Instagram suggestion parameters")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	result, err := s.upsertPendingSuggestionTx(ctx, tx, params)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return result.Created, nil
}

type suggestionUpsertResult struct {
	ID        uuid.UUID
	Created   bool
	Supported bool
}

// upsertPendingSuggestionTx is the transactional seam used by targeted
// reconciliation. The caller can attach actorless notification support before
// committing, so a suggestion and its notification can never diverge.
func (s *SuggestionStore) upsertPendingSuggestionTx(ctx context.Context, tx pgx.Tx, params UpsertSuggestionParams) (suggestionUpsertResult, error) {
	if s == nil || s.pool == nil || tx == nil || params.ID == uuid.Nil || params.ImporterDID == "" || params.TargetDID == "" || params.ImporterDID == params.TargetDID || params.ImportID == uuid.Nil || params.Now.IsZero() {
		return suggestionUpsertResult{}, errors.New("invalid Instagram suggestion parameters")
	}
	username, err := NormalizeInstagramUsername(params.Username)
	if err != nil {
		return suggestionUpsertResult{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 3))`, params.ImporterDID); err != nil {
		return suggestionUpsertResult{}, err
	}
	var handleID int64
	if err := tx.QueryRow(ctx, `
		SELECT h.id
		FROM instagram_graph_imports i
		JOIN instagram_graph_handles h ON h.import_id = i.id
		JOIN instagram_account_links link
		  ON link.owner_did = $4
		 AND link.username_normalized = h.username_normalized
		 AND link.state = 'active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE i.id = $1 AND i.owner_did = $2 AND i.state = 'active'
		  AND h.username_normalized = $3 AND h.direction = 'following'
		  AND (i.retention_expires_at IS NULL OR i.retention_expires_at > $5)
		  AND (h.retain_until IS NULL OR h.retain_until > $5)
		LIMIT 1
		FOR UPDATE OF i, h, link
	`, params.ImportID, params.ImporterDID, username, params.TargetDID, params.Now).Scan(&handleID); errors.Is(err, pgx.ErrNoRows) {
		return suggestionUpsertResult{}, ErrInstagramResourceNotFound
	} else if err != nil {
		return suggestionUpsertResult{}, err
	}

	created := false
	var suggestionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO instagram_follow_suggestions (
			id, importer_did, target_did, state, reason, created_at, updated_at
		) VALUES ($1, $2, $3, 'pending', 'verifiedInstagramFollow', $4, $4)
		ON CONFLICT (importer_did, target_did, reason) DO NOTHING
		RETURNING id
	`, params.ID, params.ImporterDID, params.TargetDID, params.Now).Scan(&suggestionID)
	if errors.Is(err, pgx.ErrNoRows) {
		var state InstagramSuggestionState
		if err := tx.QueryRow(ctx, `
			SELECT id, state FROM instagram_follow_suggestions
			WHERE importer_did = $1 AND target_did = $2 AND reason = 'verifiedInstagramFollow'
			FOR UPDATE
		`, params.ImporterDID, params.TargetDID).Scan(&suggestionID, &state); err != nil {
			return suggestionUpsertResult{}, err
		}
		if state.Terminal() {
			return suggestionUpsertResult{ID: suggestionID}, nil
		}
	} else if err != nil {
		return suggestionUpsertResult{}, err
	} else {
		created = true
	}
	if _, err := tx.Exec(ctx, `UPDATE instagram_graph_handles SET matched = true WHERE id = $1`, handleID); err != nil {
		return suggestionUpsertResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_suggestion_sources (suggestion_id, import_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (suggestion_id, import_id) DO NOTHING
	`, suggestionID, params.ImportID, params.Now); err != nil {
		return suggestionUpsertResult{}, err
	}
	return suggestionUpsertResult{ID: suggestionID, Created: created, Supported: true}, nil
}

func (s *SuggestionStore) ListPendingSuggestions(ctx context.Context, owner syntax.DID, limit int, after *SuggestionCursor) ([]SuggestionEvidence, *SuggestionCursor, error) {
	if s == nil || s.pool == nil || owner == "" || limit < 1 {
		return nil, nil, errors.New("invalid Instagram suggestion list parameters")
	}
	if after != nil {
		if after.ID == uuid.Nil || after.CreatedAt.IsZero() {
			return nil, nil, ErrInvalidInstagramSuggestionCursor
		}
		var present bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM instagram_follow_suggestions
				WHERE importer_did = $1 AND id = $2 AND created_at = $3
			)
		`, owner, after.ID, after.CreatedAt).Scan(&present); err != nil {
			return nil, nil, err
		}
		if !present {
			return nil, nil, ErrInvalidInstagramSuggestionCursor
		}
	}
	query := suggestionEvidenceSelect + `
		WHERE s.importer_did = $1 AND s.state = 'pending'
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT $2`
	args := []any{owner, limit + 1}
	if after != nil {
		query = suggestionEvidenceSelect + `
			WHERE s.importer_did = $1 AND s.state = 'pending'
			  AND (s.created_at, s.id) < ($3, $4)
			ORDER BY s.created_at DESC, s.id DESC
			LIMIT $2`
		args = append(args, after.CreatedAt, after.ID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]SuggestionEvidence, 0, limit+1)
	for rows.Next() {
		item, err := scanSuggestionEvidence(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(items) <= limit {
		return items, nil, nil
	}
	items = items[:limit]
	last := items[len(items)-1].Suggestion
	return items, &SuggestionCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
}

func (s *SuggestionStore) DismissSuggestion(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram suggestion store unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var transitionedID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE instagram_follow_suggestions
		SET state = 'dismissed', terminal_at = $3, updated_at = $3
		WHERE id = $1 AND importer_did = $2 AND state = 'pending'
		RETURNING id
	`, id, owner, now).Scan(&transitionedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	if err := s.retractInstagramMatch(ctx, tx, transitionedID, "suggestion_invalidated"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *SuggestionStore) ClaimSuggestionAcceptance(ctx context.Context, owner syntax.DID, id uuid.UUID, rkey string, now time.Time) (AcceptanceClaim, error) {
	if s == nil || s.pool == nil || owner == "" || id == uuid.Nil || now.IsZero() {
		return AcceptanceClaim{}, ErrInstagramResourceNotFound
	}
	parsedRkey, err := syntax.ParseRecordKey(rkey)
	if err != nil {
		return AcceptanceClaim{}, errors.New("invalid follow operation rkey")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AcceptanceClaim{}, err
	}
	defer tx.Rollback(ctx)
	claim, err := scanAcceptanceClaim(tx.QueryRow(ctx, suggestionClaimSelect, id, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return AcceptanceClaim{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return AcceptanceClaim{}, err
	}
	if claim.Suggestion.State == SuggestionDismissed || claim.Suggestion.State == SuggestionInvalidated {
		return AcceptanceClaim{}, ErrInstagramSuggestionIneligible
	}
	if claim.Suggestion.State == SuggestionAccepted || claim.Suggestion.State == SuggestionAlreadyFollowing {
		operation, err := scanFollowOperation(tx.QueryRow(ctx, followOperationBySuggestionQuery, id))
		if err != nil {
			return AcceptanceClaim{}, err
		}
		claim.Operation = operation
		if err := tx.Commit(ctx); err != nil {
			return AcceptanceClaim{}, err
		}
		return claim, nil
	}
	if claim.ImportedUsername == "" || claim.Direction != DirectionFollowing {
		var transitionedID uuid.UUID
		if err := tx.QueryRow(ctx, `
			UPDATE instagram_follow_suggestions
			SET state = 'invalidated', terminal_at = $3, updated_at = $3
			WHERE id = $1 AND importer_did = $2
			RETURNING id
		`, id, owner, now).Scan(&transitionedID); err != nil {
			return AcceptanceClaim{}, err
		}
		if err := s.retractInstagramMatch(ctx, tx, transitionedID, "suggestion_invalidated"); err != nil {
			return AcceptanceClaim{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return AcceptanceClaim{}, err
		}
		return AcceptanceClaim{}, ErrInstagramSuggestionIneligible
	}
	operation, err := scanFollowOperation(tx.QueryRow(ctx, `
		INSERT INTO pds_follow_operations (
			id, suggestion_id, owner_did, target_did, rkey, status,
			attempt_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'writing', 1, $6, $6)
		ON CONFLICT (suggestion_id) DO UPDATE SET
			status = CASE
				WHEN pds_follow_operations.status IN ('succeeded', 'alreadyFollowing') THEN pds_follow_operations.status
				ELSE 'writing'
			END,
			attempt_count = CASE
				WHEN pds_follow_operations.status IN ('succeeded', 'alreadyFollowing') THEN pds_follow_operations.attempt_count
				ELSE pds_follow_operations.attempt_count + 1
			END,
			updated_at = EXCLUDED.updated_at
		RETURNING id, suggestion_id, owner_did, target_did, rkey, status,
		          attempt_count, created_at, updated_at
	`, uuid.New(), id, owner, claim.Suggestion.TargetDID, parsedRkey, now))
	if err != nil {
		return AcceptanceClaim{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state = 'accepting', accepting_since = COALESCE(accepting_since, $3),
		    updated_at = $3
		WHERE id = $1 AND importer_did = $2
	`, id, owner, now); err != nil {
		return AcceptanceClaim{}, err
	}
	claim.Suggestion.State = SuggestionAccepting
	claim.Suggestion.UpdatedAt = now
	claim.Operation = operation
	if err := tx.Commit(ctx); err != nil {
		return AcceptanceClaim{}, err
	}
	return claim, nil
}

func (s *SuggestionStore) CompleteSuggestionAcceptance(ctx context.Context, owner syntax.DID, id uuid.UUID, state InstagramSuggestionState, now time.Time) (Suggestion, error) {
	if state != SuggestionAccepted && state != SuggestionAlreadyFollowing {
		return Suggestion{}, ErrInstagramStateTransition
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Suggestion{}, err
	}
	defer tx.Rollback(ctx)
	operationStatus := FollowOperationSucceeded
	if state == SuggestionAlreadyFollowing {
		operationStatus = FollowOperationAlreadyFollowing
	}
	suggestion, err := scanSuggestion(tx.QueryRow(ctx, `
		UPDATE instagram_follow_suggestions
		SET state = $3, terminal_at = COALESCE(terminal_at, $4), updated_at = $4
		WHERE id = $1 AND importer_did = $2
		  AND state IN ('accepting', $3)
		RETURNING id, importer_did, target_did, state, reason, created_at, updated_at
	`, id, owner, state, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return Suggestion{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return Suggestion{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status = $2, completed_at = COALESCE(completed_at, $3),
		    record_uri = COALESCE(record_uri, 'at://' || owner_did || '/app.bsky.graph.follow/' || rkey),
		    last_error_code = NULL, updated_at = $3
		WHERE suggestion_id = $1
	`, id, operationStatus, now); err != nil {
		return Suggestion{}, err
	}
	if err := s.retractInstagramMatch(ctx, tx, suggestion.ID, "suggestion_invalidated"); err != nil {
		return Suggestion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Suggestion{}, err
	}
	return suggestion, nil
}

func (s *SuggestionStore) ResetSuggestionAcceptance(ctx context.Context, owner syntax.DID, id uuid.UUID, safeErrorCode string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state = 'pending', accepting_since = NULL, updated_at = $3
		WHERE id = $1 AND importer_did = $2 AND state = 'accepting'
	`, id, owner, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status = 'failed', last_error_code = $2, updated_at = $3
		WHERE suggestion_id = $1
	`, id, safeErrorCode, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *SuggestionStore) InvalidateSuggestion(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram suggestion store unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var transitionedID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE instagram_follow_suggestions
		SET state = 'invalidated', terminal_at = $3, updated_at = $3
		WHERE id = $1 AND importer_did = $2 AND state IN ('pending', 'accepting')
		RETURNING id
	`, id, owner, now).Scan(&transitionedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	if err := s.retractInstagramMatch(ctx, tx, transitionedID, "suggestion_invalidated"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *SuggestionStore) retractInstagramMatch(ctx context.Context, tx pgx.Tx, id uuid.UUID, reason string) error {
	if s.notifications == nil {
		return nil
	}
	return s.notifications.RetractInstagramMatch(ctx, tx, notifications.InstagramMatchRetraction{
		SuggestionID: id,
		Reason:       reason,
	})
}

const suggestionEvidenceSelect = `
	SELECT s.id, s.importer_did, s.target_did, s.state, s.reason,
	       s.created_at, s.updated_at,
	       COALESCE(e.username_normalized, ''), COALESCE(e.direction, '')
	FROM instagram_follow_suggestions s
	LEFT JOIN LATERAL (
		SELECT h.username_normalized, h.direction
		FROM instagram_suggestion_sources source
		JOIN instagram_graph_imports i
		  ON i.id = source.import_id
		 AND i.owner_did = s.importer_did
		 AND i.state = 'active'
		JOIN instagram_graph_handles h
		  ON h.import_id = i.id AND h.matched
		JOIN instagram_account_links link
		  ON link.owner_did = s.target_did
		 AND link.username_normalized = h.username_normalized
		WHERE source.suggestion_id = s.id
		ORDER BY source.created_at, h.id
		LIMIT 1
	) e ON true`

const suggestionClaimSelect = suggestionEvidenceSelect + `
	WHERE s.id = $1 AND s.importer_did = $2
	FOR UPDATE OF s`

const followOperationBySuggestionQuery = `
	SELECT id, suggestion_id, owner_did, target_did, rkey, status,
	       attempt_count, created_at, updated_at
	FROM pds_follow_operations
	WHERE suggestion_id = $1`

type suggestionRow interface{ Scan(...any) error }

func scanSuggestion(row suggestionRow) (Suggestion, error) {
	var suggestion Suggestion
	var importer, target string
	if err := row.Scan(
		&suggestion.ID, &importer, &target, &suggestion.State,
		&suggestion.Reason, &suggestion.CreatedAt, &suggestion.UpdatedAt,
	); err != nil {
		return Suggestion{}, err
	}
	if !suggestion.State.Valid() || suggestion.Reason != SuggestionReasonVerifiedInstagramFollow {
		return Suggestion{}, fmt.Errorf("%w: suggestion", ErrInvalidInstagramState)
	}
	suggestion.ImporterDID = syntax.DID(importer)
	suggestion.TargetDID = syntax.DID(target)
	return suggestion, nil
}

func scanSuggestionEvidence(row suggestionRow) (SuggestionEvidence, error) {
	var evidence SuggestionEvidence
	var importer, target string
	if err := row.Scan(
		&evidence.Suggestion.ID, &importer, &target, &evidence.Suggestion.State,
		&evidence.Suggestion.Reason, &evidence.Suggestion.CreatedAt,
		&evidence.Suggestion.UpdatedAt, &evidence.ImportedUsername, &evidence.Direction,
	); err != nil {
		return SuggestionEvidence{}, err
	}
	if !evidence.Suggestion.State.Valid() || evidence.Suggestion.Reason != SuggestionReasonVerifiedInstagramFollow {
		return SuggestionEvidence{}, fmt.Errorf("%w: suggestion evidence", ErrInvalidInstagramState)
	}
	evidence.Suggestion.ImporterDID = syntax.DID(importer)
	evidence.Suggestion.TargetDID = syntax.DID(target)
	return evidence, nil
}

func scanAcceptanceClaim(row suggestionRow) (AcceptanceClaim, error) {
	evidence, err := scanSuggestionEvidence(row)
	if err != nil {
		return AcceptanceClaim{}, err
	}
	return AcceptanceClaim{
		Suggestion: evidence.Suggestion, ImportedUsername: evidence.ImportedUsername,
		Direction: evidence.Direction,
	}, nil
}

func scanFollowOperation(row suggestionRow) (FollowOperation, error) {
	var operation FollowOperation
	var owner, target, rkey string
	if err := row.Scan(
		&operation.ID, &operation.SuggestionID, &owner, &target, &rkey,
		&operation.Status, &operation.AttemptCount, &operation.CreatedAt,
		&operation.UpdatedAt,
	); err != nil {
		return FollowOperation{}, err
	}
	parsed, err := syntax.ParseRecordKey(rkey)
	if err != nil {
		return FollowOperation{}, err
	}
	operation.OwnerDID = syntax.DID(owner)
	operation.TargetDID = syntax.DID(target)
	operation.Rkey = parsed
	return operation, nil
}
