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
	"social.craftsky/appview/internal/ownerlifecycle"
)

type PrivateSuggestion struct {
	ID                 uuid.UUID
	ImporterDID        syntax.DID
	TargetDID          syntax.DID
	ImporterGeneration int64
	TargetGeneration   int64
	EvidenceLinkID     uuid.UUID
	State              SuggestionState
	AcceptingSince     *time.Time
	TerminalAt         *time.Time
	ResultRecordURI    *syntax.ATURI
	ResultRecordCID    *string
	ImportedUsername   string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (PrivateSuggestion) String() string {
	return "Instagram private suggestion [REDACTED]"
}

func (suggestion PrivateSuggestion) GoString() string { return suggestion.String() }

type ReconcilePrivateSuggestionParams struct {
	ID          uuid.UUID
	ImporterDID syntax.DID
	TargetDID   syntax.DID
	ImportID    uuid.UUID
	Username    string
	Now         time.Time
}

type ReconcilePrivateSuggestionResult struct {
	Suggestion PrivateSuggestion
	Created    bool
}

// PrivateSuggestionStore deliberately has no OAuth, PDS, or session-selection
// dependency. Its notification capability only writes caller-private AppView
// state in the same transaction as a newly created suggestion.
type PrivateSuggestionStore struct {
	pool          *pgxpool.Pool
	lifecycles    *ownerlifecycle.Store
	notifications *notifications.Service
	now           func() time.Time
}

func NewPrivateSuggestionStore(
	pool *pgxpool.Pool,
	lifecycles *ownerlifecycle.Store,
	notificationService *notifications.Service,
	now func() time.Time,
) (*PrivateSuggestionStore, error) {
	if pool == nil || lifecycles == nil || notificationService == nil {
		return nil, errors.New("private Instagram suggestion store requires database, lifecycle, and notification services")
	}
	if now == nil {
		now = time.Now
	}
	return &PrivateSuggestionStore{
		pool: pool, lifecycles: lifecycles, notifications: notificationService, now: now,
	}, nil
}

func (store *PrivateSuggestionStore) ReconcileCandidate(
	ctx context.Context,
	params ReconcilePrivateSuggestionParams,
) (ReconcilePrivateSuggestionResult, error) {
	if store == nil || store.pool == nil || store.lifecycles == nil ||
		params.ID == uuid.Nil || params.ImporterDID == "" || params.TargetDID == "" ||
		params.ImporterDID == params.TargetDID || params.ImportID == uuid.Nil || params.Now.IsZero() {
		return ReconcilePrivateSuggestionResult{}, errors.New("invalid private Instagram suggestion reconciliation")
	}
	username, err := NormalizeInstagramUsername(params.Username)
	if err != nil {
		return ReconcilePrivateSuggestionResult{}, err
	}
	params.Now = params.Now.UTC()

	var result ReconcilePrivateSuggestionResult
	err = store.lifecycles.WithNonTerminalOwners(
		ctx,
		[]syntax.DID{params.ImporterDID, params.TargetDID},
		func(fenceCtx context.Context, tx pgx.Tx, current map[syntax.DID]ownerlifecycle.Lifecycle) error {
			importer, importerExists := current[params.ImporterDID]
			target, targetExists := current[params.TargetDID]
			if !importerExists || !targetExists ||
				importer.State != ownerlifecycle.StateActive || target.State != ownerlifecycle.StateActive {
				return ownerlifecycle.ErrOwnerNotActive
			}

			var (
				handleID int64
				linkID   uuid.UUID
			)
			err := tx.QueryRow(fenceCtx, `
				SELECT handle.id,link.id
				FROM instagram_graph_imports AS import
				JOIN instagram_graph_handles AS handle
				  ON handle.import_id=import.id
				JOIN instagram_account_links AS link
				  ON link.owner_did=$4
				 AND link.username_normalized=handle.username_normalized
				 AND link.state='active'
				 AND link.discoverable
				 AND NOT link.conflict_pending
				WHERE import.id=$1
				  AND import.owner_did=$2
				  AND import.state='active'
				  AND handle.username_normalized=$3
				LIMIT 1
				FOR UPDATE OF import,handle,link
			`, params.ImportID, params.ImporterDID, username, params.TargetDID).Scan(&handleID, &linkID)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInstagramResourceNotFound
			}
			if err != nil {
				return fmt.Errorf("load private Instagram suggestion evidence: %w", err)
			}

			row := tx.QueryRow(fenceCtx, `
				INSERT INTO instagram_private_suggestions(
					id,importer_did,target_did,importer_generation,target_generation,
					evidence_link_id,state,reason,created_at,updated_at
				) VALUES (
					$1,$2,$3,$4,$5,$6,'pending','verifiedInstagramFollow',$7,$7
				)
				ON CONFLICT (
					importer_did,target_did,importer_generation,target_generation,
					evidence_link_id,reason
				) DO NOTHING
				RETURNING `+privateSuggestionColumns,
				params.ID,
				params.ImporterDID,
				params.TargetDID,
				importer.Generation,
				target.Generation,
				linkID,
				params.Now,
			)
			result.Suggestion, err = scanPrivateSuggestion(row)
			result.Created = err == nil
			if errors.Is(err, pgx.ErrNoRows) {
				result.Suggestion, err = scanPrivateSuggestion(tx.QueryRow(fenceCtx, `
					SELECT `+privateSuggestionColumns+`
					FROM instagram_private_suggestions
					WHERE importer_did=$1
					  AND target_did=$2
					  AND importer_generation=$3
					  AND target_generation=$4
					  AND evidence_link_id=$5
					  AND reason='verifiedInstagramFollow'
					FOR UPDATE
				`, params.ImporterDID, params.TargetDID, importer.Generation, target.Generation, linkID))
			}
			if err != nil {
				return fmt.Errorf("persist private Instagram suggestion: %w", err)
			}
			if result.Created {
				if err := store.notifications.ActivateInstagramMatch(
					fenceCtx,
					tx,
					notifications.InstagramMatchActivation{
						RecipientDID: params.ImporterDID,
						ActorDID:     params.TargetDID,
						SuggestionID: result.Suggestion.ID,
						ActivityAt:   params.Now,
					},
				); err != nil {
					return fmt.Errorf("activate private Instagram suggestion notification: %w", err)
				}
			}
			if _, err := tx.Exec(fenceCtx, `
				INSERT INTO instagram_private_suggestion_sources(suggestion_id,import_id,created_at)
				VALUES ($1,$2,$3)
				ON CONFLICT (suggestion_id,import_id) DO NOTHING
			`, result.Suggestion.ID, params.ImportID, params.Now); err != nil {
				return fmt.Errorf("persist private Instagram suggestion source: %w", err)
			}
			if _, err := tx.Exec(fenceCtx, `
				UPDATE instagram_graph_handles SET matched=true WHERE id=$1
			`, handleID); err != nil {
				return fmt.Errorf("mark private Instagram suggestion evidence matched: %w", err)
			}
			return nil
		},
	)
	return result, err
}

// Dismiss is deliberately non-disclosing: absent and foreign suggestion IDs
// are successful no-ops. Only the importing owner can move a pending row to
// the monotonic dismissed state.
func (store *PrivateSuggestionStore) Dismiss(
	ctx context.Context,
	importer syntax.DID,
	suggestionID uuid.UUID,
) (bool, error) {
	if store == nil || store.pool == nil || store.lifecycles == nil || importer == "" || suggestionID == uuid.Nil {
		return false, errors.New("invalid private Instagram suggestion dismissal")
	}
	var target syntax.DID
	err := store.pool.QueryRow(ctx, `
		SELECT target_did
		FROM instagram_private_suggestions
		WHERE id=$1 AND importer_did=$2
	`, suggestionID, importer).Scan(&target)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	changed := false
	err = store.lifecycles.WithNonTerminalOwners(
		ctx,
		[]syntax.DID{importer, target},
		func(fenceCtx context.Context, tx pgx.Tx, current map[syntax.DID]ownerlifecycle.Lifecycle) error {
			importerLifecycle, importerExists := current[importer]
			targetLifecycle, targetExists := current[target]
			if !importerExists || !targetExists ||
				importerLifecycle.State != ownerlifecycle.StateActive ||
				targetLifecycle.State != ownerlifecycle.StateActive {
				return ownerlifecycle.ErrOwnerNotActive
			}
			var state SuggestionState
			var importerGeneration, targetGeneration int64
			err := tx.QueryRow(fenceCtx, `
				SELECT state,importer_generation,target_generation
				FROM instagram_private_suggestions
				WHERE id=$1 AND importer_did=$2 AND target_did=$3
				FOR UPDATE
			`, suggestionID, importer, target).Scan(&state, &importerGeneration, &targetGeneration)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := ValidateSuggestionGenerations(
				importerGeneration,
				targetGeneration,
				importerLifecycle.Generation,
				targetLifecycle.Generation,
			); err != nil {
				return err
			}
			if state != SuggestionPending {
				return nil
			}
			now := store.now().UTC()
			tag, err := tx.Exec(fenceCtx, `
				UPDATE instagram_private_suggestions
				SET state='dismissed',accepting_since=NULL,terminal_at=$3,updated_at=$3
				WHERE id=$1 AND importer_did=$2 AND state='pending'
			`, suggestionID, importer, now)
			if err != nil {
				return err
			}
			changed = tag.RowsAffected() == 1
			return nil
		},
	)
	return changed, err
}

const privateSuggestionColumns = `
	id,importer_did,target_did,importer_generation,target_generation,
	evidence_link_id,state,accepting_since,terminal_at,result_record_uri,
	result_record_cid,created_at,updated_at`

type privateSuggestionScanner interface {
	Scan(dest ...any) error
}

func scanPrivateSuggestion(row privateSuggestionScanner) (PrivateSuggestion, error) {
	var (
		suggestion PrivateSuggestion
		resultURI  *string
	)
	err := row.Scan(
		&suggestion.ID,
		&suggestion.ImporterDID,
		&suggestion.TargetDID,
		&suggestion.ImporterGeneration,
		&suggestion.TargetGeneration,
		&suggestion.EvidenceLinkID,
		&suggestion.State,
		&suggestion.AcceptingSince,
		&suggestion.TerminalAt,
		&resultURI,
		&suggestion.ResultRecordCID,
		&suggestion.CreatedAt,
		&suggestion.UpdatedAt,
	)
	if resultURI != nil {
		parsed := syntax.ATURI(*resultURI)
		suggestion.ResultRecordURI = &parsed
	}
	return suggestion, err
}
