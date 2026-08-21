package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var (
	ErrProfilePinForbidden      = errors.New("profile pin: forbidden")
	ErrProfilePinTargetNotFound = errors.New("profile pin: target not found")
)

type ProfilePinState struct {
	StandardPostURI *syntax.ATURI `json:"standardPostUri"`
	ProjectPostURI  *syntax.ATURI `json:"projectPostUri"`
}

type ProfileListPin struct {
	Row        *PostRow
	StateToken string
}

type ProfilePinListReader interface {
	ReadProfileListPin(
		context.Context,
		syntax.DID,
		syntax.DID,
		ProfilePinSlot,
		[]string,
	) (*ProfileListPin, error)
}

type ProfilePinOperation string

const (
	ProfilePinOperationPin     ProfilePinOperation = "pin"
	ProfilePinOperationReplace ProfilePinOperation = "replace"
	ProfilePinOperationUnpin   ProfilePinOperation = "unpin"
	ProfilePinOperationNoop    ProfilePinOperation = "noop"
)

type ProfilePinMutationResult struct {
	State     ProfilePinState
	Slot      ProfilePinSlot
	Operation ProfilePinOperation
}

type ProfilePinStoreOptions struct {
	Now      func() time.Time
	NewID    func() uuid.UUID
	Observer *observability.Observer
}

type ProfilePinStore struct {
	pool     *pgxpool.Pool
	now      func() time.Time
	newID    func() uuid.UUID
	observer *observability.Observer
}

func NewProfilePinStore(pool *pgxpool.Pool, options ...ProfilePinStoreOptions) *ProfilePinStore {
	now := time.Now
	newID := uuid.New
	if len(options) > 0 {
		if options[0].Now != nil {
			now = options[0].Now
		}
		if options[0].NewID != nil {
			newID = options[0].NewID
		}
	}
	var observer *observability.Observer
	if len(options) > 0 {
		observer = options[0].Observer
	}
	return &ProfilePinStore{pool: pool, now: now, newID: newID, observer: observer}
}

func (s *ProfilePinStore) Read(ctx context.Context, owner syntax.DID) (ProfilePinState, error) {
	return readProfilePinState(ctx, s.pool, owner)
}

func (s *ProfilePinStore) ReadProfileListPin(
	ctx context.Context,
	owner syntax.DID,
	viewer syntax.DID,
	slot ProfilePinSlot,
	contentLanguages []string,
) (*ProfileListPin, error) {
	if contentLanguages == nil {
		contentLanguages = []string{}
	}
	query := `
		SELECT ` + postSelectColumns + `, pin.state_token::text
		FROM profile_pins pin
		JOIN craftsky_posts p ON p.uri = pin.post_uri
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE pin.owner_did = $1
		  AND pin.slot = $2
		  AND p.did = pin.owner_did
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND (
			($2 = 'standard' AND p.is_project = false)
			OR ($2 = 'project' AND p.is_project = true AND pp.uri IS NOT NULL AND p.quote_uri IS NULL)
		  )
		` + postVisibleModerationPredicate + `
		` + languageVisibilityPredicate("p", "$3", "$4")
	var token string
	row, err := scanPostRowWithExtra(
		s.pool.QueryRow(ctx, query, owner, slot, viewer, contentLanguages),
		&token,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("profile pin list read: %w", err)
	}
	return &ProfileListPin{Row: row, StateToken: token}, nil
}

func (s *ProfilePinStore) Pin(
	ctx context.Context,
	owner syntax.DID,
	targetDID syntax.DID,
	targetRkey syntax.RecordKey,
) (result ProfilePinMutationResult, resultErr error) {
	started := time.Now()
	defer func() {
		s.observeProfilePinMutation("pin", result, resultErr, time.Since(started))
	}()
	if owner != targetDID {
		return ProfilePinMutationResult{}, ErrProfilePinForbidden
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile pin begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, owner, nil); err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile pin authorization: %w", err)
	}

	var (
		targetURI string
		shape     ProfilePinTargetShape
	)
	err = tx.QueryRow(ctx, `
		SELECT p.uri,
		       p.is_project,
		       pp.uri IS NOT NULL,
		       p.reply_root_uri IS NOT NULL,
		       p.reply_parent_uri IS NOT NULL,
		       p.quote_uri IS NOT NULL
		FROM craftsky_posts p
		JOIN craftsky_profiles member ON member.did = p.did
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		WHERE p.did = $1 AND p.rkey = $2
		`+postVisibleModerationPredicate+`
		FOR UPDATE OF p
	`, targetDID, targetRkey).Scan(
		&targetURI,
		&shape.IsProject,
		&shape.HasProjectMaterialization,
		&shape.HasReplyRoot,
		&shape.HasReplyParent,
		&shape.HasQuote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProfilePinMutationResult{}, ErrProfilePinTargetNotFound
	}
	if err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile pin target: %w", err)
	}

	slot, err := ClassifyProfilePinSlot(shape)
	if err != nil {
		return ProfilePinMutationResult{}, err
	}
	if err := lockProfilePinOwner(ctx, tx, owner); err != nil {
		return ProfilePinMutationResult{}, err
	}

	var existingURI string
	err = tx.QueryRow(ctx, `
		SELECT post_uri
		FROM profile_pins
		WHERE owner_did = $1 AND slot = $2
		FOR UPDATE
	`, owner, slot).Scan(&existingURI)
	operation := ProfilePinOperationPin
	switch {
	case err == nil && existingURI == targetURI:
		operation = ProfilePinOperationNoop
	case err == nil:
		operation = ProfilePinOperationReplace
		now := s.now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE profile_pins
			SET post_uri = $3, state_token = $4, updated_at = $5
			WHERE owner_did = $1 AND slot = $2
		`, owner, slot, targetURI, s.newID(), now); err != nil {
			return ProfilePinMutationResult{}, fmt.Errorf("profile pin replace: %w", err)
		}
	case errors.Is(err, pgx.ErrNoRows):
		now := s.now().UTC()
		if _, err := tx.Exec(ctx, `
			INSERT INTO profile_pins (
				owner_did, slot, post_uri, state_token, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $5)
		`, owner, slot, targetURI, s.newID(), now); err != nil {
			return ProfilePinMutationResult{}, fmt.Errorf("profile pin insert: %w", err)
		}
	default:
		return ProfilePinMutationResult{}, fmt.Errorf("profile pin current state: %w", err)
	}

	state, err := readProfilePinState(ctx, tx, owner)
	if err != nil {
		return ProfilePinMutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile pin commit: %w", err)
	}
	return ProfilePinMutationResult{State: state, Slot: slot, Operation: operation}, nil
}

func (s *ProfilePinStore) Unpin(
	ctx context.Context,
	owner syntax.DID,
	targetURI syntax.ATURI,
) (result ProfilePinMutationResult, resultErr error) {
	started := time.Now()
	defer func() {
		s.observeProfilePinMutation("unpin", result, resultErr, time.Since(started))
	}()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile unpin begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, owner, nil); err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile unpin authorization: %w", err)
	}

	if err := lockProfilePinOwner(ctx, tx, owner); err != nil {
		return ProfilePinMutationResult{}, err
	}
	var slot ProfilePinSlot
	err = tx.QueryRow(ctx, `
		DELETE FROM profile_pins
		WHERE owner_did = $1 AND post_uri = $2
		RETURNING slot
	`, owner, targetURI).Scan(&slot)
	operation := ProfilePinOperationUnpin
	if errors.Is(err, pgx.ErrNoRows) {
		operation = ProfilePinOperationNoop
	} else if err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile unpin delete: %w", err)
	}

	state, err := readProfilePinState(ctx, tx, owner)
	if err != nil {
		return ProfilePinMutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProfilePinMutationResult{}, fmt.Errorf("profile unpin commit: %w", err)
	}
	return ProfilePinMutationResult{State: state, Slot: slot, Operation: operation}, nil
}

func (s *ProfilePinStore) observeProfilePinMutation(
	requestedOperation string,
	mutation ProfilePinMutationResult,
	err error,
	duration time.Duration,
) {
	operation := requestedOperation
	slot := string(mutation.Slot)
	result := "success"
	errorClass := "none"
	if err != nil {
		result = "rejected"
		switch {
		case errors.Is(err, ErrProfilePinForbidden):
			errorClass = "forbidden"
		case errors.Is(err, ErrProfilePinTargetNotFound):
			errorClass = "not_found"
		case errors.Is(err, ErrProfilePinNotAllowed):
			errorClass = "policy"
		default:
			result = "error"
			errorClass = "store"
		}
	} else {
		switch mutation.Operation {
		case ProfilePinOperationReplace:
			operation = "replace"
		case ProfilePinOperationNoop:
			result = "noop"
		}
	}
	s.observer.ObserveProfilePin(operation, slot, result, errorClass, duration)
}

func lockProfilePinOwner(ctx context.Context, tx pgx.Tx, owner syntax.DID) error {
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtext('profile_pins'), hashtext($1))
	`, owner); err != nil {
		return fmt.Errorf("profile pin owner lock: %w", err)
	}
	return nil
}

type profilePinQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readProfilePinState(ctx context.Context, q profilePinQuerier, owner syntax.DID) (ProfilePinState, error) {
	rows, err := q.Query(ctx, `
		SELECT slot, post_uri
		FROM profile_pins
		WHERE owner_did = $1
		ORDER BY slot
	`, owner)
	if err != nil {
		return ProfilePinState{}, fmt.Errorf("profile pin read: %w", err)
	}
	defer rows.Close()

	var state ProfilePinState
	for rows.Next() {
		var (
			slot ProfilePinSlot
			uri  string
		)
		if err := rows.Scan(&slot, &uri); err != nil {
			return ProfilePinState{}, fmt.Errorf("profile pin read scan: %w", err)
		}
		parsed := syntax.ATURI(uri)
		switch slot {
		case ProfilePinSlotStandard:
			state.StandardPostURI = &parsed
		case ProfilePinSlotProject:
			state.ProjectPostURI = &parsed
		default:
			return ProfilePinState{}, fmt.Errorf("profile pin read: invalid slot %q", slot)
		}
	}
	if err := rows.Err(); err != nil {
		return ProfilePinState{}, fmt.Errorf("profile pin read rows: %w", err)
	}
	return state, nil
}
