package instagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrSuggestionIneligible = errors.New("instagram suggestion is no longer eligible")

type SuggestionFollowRequest struct {
	OperationKey     uuid.UUID
	RequestID        string
	SessionID        string
	SuggestionID     uuid.UUID
	Owner            syntax.DID
	Target           syntax.DID
	OwnerGeneration  int64
	TargetGeneration int64
	Rkey             syntax.RecordKey
	CreatedAt        time.Time
}

type SuggestionFollowResult struct {
	Outcome   SuggestionState
	RecordURI syntax.ATURI
	RecordCID string
}

type SuggestionCommandState string

const (
	SuggestionCommandAccepted  SuggestionCommandState = "accepted"
	SuggestionCommandRejected  SuggestionCommandState = "rejected"
	SuggestionCommandAmbiguous SuggestionCommandState = "ambiguous"
)

type SuggestionCommandResult struct {
	State             SuggestionCommandState
	HTTPStatus        int
	ResponseBody      json.RawMessage
	ResponseHeaders   json.RawMessage
	RetryAfterSeconds int
}

type SuggestionCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type SuggestionFollowExecutor interface {
	FollowSuggestion(context.Context, SuggestionFollowRequest) (SuggestionCommandResult, error)
}

// SuggestionService is the sole Instagram-specific path to a public follow.
// It receives a narrow ordinary-effect executor rather than an OAuth session
// selector, PDS factory, or raw client.
type SuggestionService struct {
	store      *PrivateSuggestionStore
	lifecycles *ownerlifecycle.Store
	policy     InstagramSuggestionEligibilityPolicy
	follow     SuggestionFollowExecutor
}

func NewSuggestionService(
	store *PrivateSuggestionStore,
	lifecycles *ownerlifecycle.Store,
	policy InstagramSuggestionEligibilityPolicy,
	follow SuggestionFollowExecutor,
) (*SuggestionService, error) {
	if store == nil || lifecycles == nil || policy == nil || follow == nil {
		return nil, errors.New("instagram suggestion service dependencies are required")
	}
	return &SuggestionService{
		store: store, lifecycles: lifecycles, policy: policy, follow: follow,
	}, nil
}

func (service *SuggestionService) ListPending(
	ctx context.Context,
	importer syntax.DID,
	limit int,
	cursor *SuggestionCursor,
) ([]PrivateSuggestion, *SuggestionCursor, error) {
	return service.store.ListPending(ctx, importer, limit, cursor)
}

func (service *SuggestionService) Dismiss(
	ctx context.Context,
	importer syntax.DID,
	suggestionID uuid.UUID,
) (bool, error) {
	return service.store.Dismiss(ctx, importer, suggestionID)
}

func (service *SuggestionService) Accept(
	ctx context.Context,
	importer syntax.DID,
	suggestionID uuid.UUID,
	sessionID string,
	operationKey uuid.UUID,
	requestID string,
) (SuggestionCommandResult, error) {
	if service == nil || importer == "" || suggestionID == uuid.Nil || strings.TrimSpace(sessionID) == "" ||
		operationKey == uuid.Nil || strings.TrimSpace(requestID) == "" {
		return SuggestionCommandResult{}, errors.New("invalid Instagram suggestion acceptance")
	}
	suggestion, err := service.store.GetOwned(ctx, importer, suggestionID)
	if err != nil {
		return SuggestionCommandResult{}, err
	}
	expected := []ownerlifecycle.ExpectedOwner{
		{Owner: suggestion.ImporterDID, Generation: suggestion.ImporterGeneration},
		{Owner: suggestion.TargetDID, Generation: suggestion.TargetGeneration},
	}
	accepted := suggestion
	if !suggestion.State.Terminal() {
		err = service.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
			if err := service.lifecycles.WithActiveEffectTransaction(effectCtx, func(tx pgx.Tx) error {
				var loadErr error
				accepted, loadErr = service.store.getOwned(effectCtx, tx, importer, suggestionID, true)
				return loadErr
			}); err != nil {
				return err
			}
			if accepted.State.Terminal() {
				return nil
			}
			if err := ValidateSuggestionGenerations(
				accepted.ImporterGeneration,
				accepted.TargetGeneration,
				suggestion.ImporterGeneration,
				suggestion.TargetGeneration,
			); err != nil {
				return err
			}

			decision, err := service.policy.Evaluate(effectCtx, EligibilityAtAccept, SuggestionEligibilityRequest{
				ImporterDID:      accepted.ImporterDID,
				TargetDID:        accepted.TargetDID,
				ImportedUsername: accepted.ImportedUsername,
			})
			if err != nil {
				return err
			}
			if !decision.Eligible && decision.Reason != EligibilityAlreadyFollowing {
				if invalidateErr := service.lifecycles.WithActiveEffectTransaction(effectCtx, func(tx pgx.Tx) error {
					var updateErr error
					accepted, updateErr = service.store.invalidate(effectCtx, tx, importer, suggestionID)
					return updateErr
				}); invalidateErr != nil {
					return invalidateErr
				}
				return ErrSuggestionIneligible
			}

			if err := service.lifecycles.WithActiveEffectTransaction(effectCtx, func(tx pgx.Tx) error {
				var reserveErr error
				accepted, reserveErr = service.store.reserveAcceptance(effectCtx, tx, importer, suggestionID)
				return reserveErr
			}); err != nil {
				return err
			}
			if accepted.State.Terminal() {
				return nil
			}
			return nil
		})
		if err != nil {
			return SuggestionCommandResult{}, err
		}
	}

	rkey := syntax.RecordKey("3l" + strings.ReplaceAll(suggestionID.String(), "-", ""))
	if _, err := syntax.ParseRecordKey(rkey.String()); err != nil {
		return SuggestionCommandResult{}, fmt.Errorf("derive Instagram suggestion follow key: %w", err)
	}
	result, err := service.follow.FollowSuggestion(ctx, SuggestionFollowRequest{
		OperationKey: operationKey, RequestID: requestID, SessionID: sessionID, SuggestionID: suggestionID,
		Owner: accepted.ImporterDID, Target: accepted.TargetDID,
		OwnerGeneration: accepted.ImporterGeneration, TargetGeneration: accepted.TargetGeneration,
		Rkey: rkey, CreatedAt: accepted.CreatedAt,
	})
	if err != nil || result.State != SuggestionCommandAccepted || accepted.State.Terminal() {
		return result, err
	}
	var response struct {
		State SuggestionState `json:"state"`
	}
	if jsonErr := json.Unmarshal(result.ResponseBody, &response); jsonErr != nil {
		return SuggestionCommandResult{}, jsonErr
	}
	followResult := SuggestionFollowResult{Outcome: response.State}
	if response.State == SuggestionFollowed {
		followResult.RecordURI = syntax.ATURI("at://" + accepted.ImporterDID.String() + "/app.bsky.graph.follow/" + rkey.String())
	}
	err = service.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
		return service.lifecycles.WithActiveEffectTransaction(effectCtx, func(tx pgx.Tx) error {
			current, loadErr := service.store.getOwned(effectCtx, tx, importer, suggestionID, true)
			if loadErr != nil || current.State.Terminal() {
				return loadErr
			}
			_, completeErr := service.store.completeAcceptance(effectCtx, tx, importer, suggestionID, followResult)
			return completeErr
		})
	})
	return result, err
}

func (store *PrivateSuggestionStore) GetOwned(
	ctx context.Context,
	importer syntax.DID,
	suggestionID uuid.UUID,
) (PrivateSuggestion, error) {
	if store == nil || store.pool == nil || importer == "" || suggestionID == uuid.Nil {
		return PrivateSuggestion{}, errors.New("invalid private Instagram suggestion lookup")
	}
	return store.getOwned(ctx, store.pool, importer, suggestionID, false)
}

func (store *PrivateSuggestionStore) ListPending(
	ctx context.Context,
	importer syntax.DID,
	limit int,
	cursor *SuggestionCursor,
) ([]PrivateSuggestion, *SuggestionCursor, error) {
	if store == nil || store.pool == nil || importer == "" {
		return nil, nil, errors.New("invalid private Instagram suggestion list")
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 50 {
		return nil, nil, errors.New("invalid private Instagram suggestion list limit")
	}
	var cursorAt any
	var cursorID any
	if cursor != nil {
		if cursor.CreatedAt.IsZero() || cursor.ID == uuid.Nil {
			return nil, nil, errors.New("invalid private Instagram suggestion cursor")
		}
		cursorAt = cursor.CreatedAt.UTC()
		cursorID = cursor.ID
	}
	rows, err := store.pool.Query(ctx, `
		SELECT `+privateSuggestionQualifiedColumns+`,link.username_normalized
		FROM instagram_private_suggestions AS suggestion
		JOIN instagram_account_links AS link ON link.id=suggestion.evidence_link_id
		JOIN owner_lifecycles AS importer_lifecycle
		  ON importer_lifecycle.owner_did=suggestion.importer_did
		 AND importer_lifecycle.state='active'
		 AND importer_lifecycle.generation=suggestion.importer_generation
		JOIN owner_lifecycles AS target_lifecycle
		  ON target_lifecycle.owner_did=suggestion.target_did
		 AND target_lifecycle.state='active'
		 AND target_lifecycle.generation=suggestion.target_generation
		WHERE suggestion.importer_did=$1
		  AND suggestion.state='pending'
		  AND ($2::timestamptz IS NULL OR (suggestion.created_at,suggestion.id) < ($2,$3::uuid))
		ORDER BY suggestion.created_at DESC,suggestion.id DESC
		LIMIT $4
	`, importer, cursorAt, cursorID, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]PrivateSuggestion, 0, limit+1)
	for rows.Next() {
		item, err := scanPrivateSuggestionWithUsername(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *SuggestionCursor
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = &SuggestionCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return items, next, nil
}

type suggestionQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (store *PrivateSuggestionStore) getOwned(
	ctx context.Context,
	queryer suggestionQueryer,
	importer syntax.DID,
	suggestionID uuid.UUID,
	forUpdate bool,
) (PrivateSuggestion, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE OF suggestion"
	}
	row := queryer.QueryRow(ctx, `
		SELECT `+privateSuggestionQualifiedColumns+`,link.username_normalized
		FROM instagram_private_suggestions AS suggestion
		JOIN instagram_account_links AS link ON link.id=suggestion.evidence_link_id
		WHERE suggestion.id=$1 AND suggestion.importer_did=$2`+lock,
		suggestionID,
		importer,
	)
	suggestion, err := scanPrivateSuggestionWithUsername(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return PrivateSuggestion{}, ErrInstagramResourceNotFound
	}
	return suggestion, err
}

const privateSuggestionQualifiedColumns = `
	suggestion.id,suggestion.importer_did,suggestion.target_did,
	suggestion.importer_generation,suggestion.target_generation,
	suggestion.evidence_link_id,suggestion.state,suggestion.accepting_since,
	suggestion.terminal_at,suggestion.result_record_uri,
	suggestion.result_record_cid,suggestion.created_at,suggestion.updated_at`

func (store *PrivateSuggestionStore) reserveAcceptance(
	ctx context.Context,
	tx pgx.Tx,
	importer syntax.DID,
	suggestionID uuid.UUID,
) (PrivateSuggestion, error) {
	now := store.now().UTC()
	suggestion, err := scanPrivateSuggestion(tx.QueryRow(ctx, `
		UPDATE instagram_private_suggestions AS suggestion
		SET state='accepting',accepting_since=COALESCE(accepting_since,$3),updated_at=$3
		WHERE suggestion.id=$1 AND suggestion.importer_did=$2
		  AND suggestion.state IN ('pending','accepting')
		  AND EXISTS (
			SELECT 1
			FROM instagram_account_links AS link
			WHERE link.id=suggestion.evidence_link_id
			  AND link.owner_did=suggestion.target_did
			  AND link.state='active'
			  AND link.discoverable
			  AND NOT link.conflict_pending
		  )
		RETURNING `+privateSuggestionColumns,
		suggestionID, importer, now,
	))
	if err == nil {
		return suggestion, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PrivateSuggestion{}, err
	}
	return store.getOwned(ctx, tx, importer, suggestionID, true)
}

func (store *PrivateSuggestionStore) invalidate(
	ctx context.Context,
	tx pgx.Tx,
	importer syntax.DID,
	suggestionID uuid.UUID,
) (PrivateSuggestion, error) {
	now := store.now().UTC()
	suggestion, err := scanPrivateSuggestion(tx.QueryRow(ctx, `
		UPDATE instagram_private_suggestions
		SET state='invalidated',accepting_since=NULL,terminal_at=$3,
		    result_record_uri=NULL,result_record_cid=NULL,updated_at=$3
		WHERE id=$1 AND importer_did=$2 AND state IN ('pending','accepting')
		RETURNING `+privateSuggestionColumns,
		suggestionID, importer, now,
	))
	if err == nil {
		return suggestion, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PrivateSuggestion{}, err
	}
	return store.getOwned(ctx, tx, importer, suggestionID, true)
}

func (store *PrivateSuggestionStore) completeAcceptance(
	ctx context.Context,
	tx pgx.Tx,
	importer syntax.DID,
	suggestionID uuid.UUID,
	result SuggestionFollowResult,
) (PrivateSuggestion, error) {
	if result.Outcome != SuggestionFollowed && result.Outcome != SuggestionAlreadyFollowing {
		return PrivateSuggestion{}, ErrInstagramStateTransition
	}
	if result.Outcome == SuggestionFollowed && result.RecordURI == "" {
		return PrivateSuggestion{}, errors.New("followed Instagram suggestion requires record URI")
	}
	now := store.now().UTC()
	var recordURI any
	var recordCID any
	if result.RecordURI != "" {
		recordURI = result.RecordURI.String()
	}
	if result.RecordCID != "" {
		recordCID = result.RecordCID
	}
	suggestion, err := scanPrivateSuggestion(tx.QueryRow(ctx, `
		UPDATE instagram_private_suggestions
		SET state=$3,accepting_since=NULL,terminal_at=$4,
		    result_record_uri=$5,result_record_cid=$6,updated_at=$4
		WHERE id=$1 AND importer_did=$2 AND state='accepting'
		RETURNING `+privateSuggestionColumns,
		suggestionID, importer, result.Outcome, now, recordURI, recordCID,
	))
	if err == nil {
		return suggestion, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return PrivateSuggestion{}, err
	}
	existing, err := store.getOwned(ctx, tx, importer, suggestionID, true)
	if err != nil {
		return PrivateSuggestion{}, err
	}
	if existing.State != result.Outcome {
		return PrivateSuggestion{}, ErrInstagramStateTransition
	}
	return existing, nil
}

func scanPrivateSuggestionWithUsername(row privateSuggestionScanner) (PrivateSuggestion, error) {
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
		&suggestion.ImportedUsername,
	)
	if resultURI != nil {
		parsed := syntax.ATURI(*resultURI)
		suggestion.ResultRecordURI = &parsed
	}
	return suggestion, err
}
