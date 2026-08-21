package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool   *pgxpool.Pool
	fencer *Fencer
	now    func() time.Time
}

func NewStore(pool *pgxpool.Pool, fencer *Fencer, now func() time.Time) (*Store, error) {
	if pool == nil || fencer == nil {
		return nil, errors.New("owner lifecycle store requires a database pool and fencer")
	}
	if now == nil {
		now = time.Now
	}
	return &Store{pool: pool, fencer: fencer, now: now}, nil
}

func (store *Store) Get(ctx context.Context, owner syntax.DID) (Lifecycle, error) {
	if owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	return scanLifecycle(store.queryRow(ctx, lifecycleSelect+` WHERE owner_did=$1`, owner))
}

// IsTerminal applies the same authoritative predicate used by serving,
// projection, and effect SQL. An unknown DID is not terminal; code that needs
// positive lifecycle authority must use Get or an active/non-terminal fence.
func (store *Store) IsTerminal(ctx context.Context, owner syntax.DID) (bool, error) {
	if owner == "" {
		return false, ErrInvalidOwner
	}
	var terminal bool
	if err := store.queryRow(ctx,
		`SELECT appview_owner_is_terminal($1)`, owner,
	).Scan(&terminal); err != nil {
		return false, fmt.Errorf("read terminal owner predicate: %w", err)
	}
	return terminal, nil
}

// EnsureOnboardingOwner creates the explicit first-login lifecycle while the
// exclusive owner fence is held. A concurrent terminal tombstone wins: the
// existing terminal row is re-read and returned, never overwritten.
func (store *Store) EnsureOnboardingOwner(ctx context.Context, owner syntax.DID) (Lifecycle, error) {
	if owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	var lifecycle Lifecycle
	err := store.fencer.WithExclusive(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			now := store.now().UTC()
			if _, err := tx.Exec(fenceCtx, `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES($1,'departed',1,1,'onboarding',$2,$2,$2)
				ON CONFLICT (owner_did) DO NOTHING
			`, owner, now); err != nil {
				return fmt.Errorf("ensure onboarding owner: %w", err)
			}
			var err error
			lifecycle, err = scanLifecycle(tx.QueryRow(fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, owner))
			return err
		})
	})
	return lifecycle, err
}

// Transition performs one monotonic lifecycle transition under the exclusive
// owner fence. Terminal transitions must use Terminalize so the tombstone and
// fixed purge ledger are committed atomically.
func (store *Store) Transition(ctx context.Context, request TransitionRequest) (Lifecycle, error) {
	return store.TransitionWith(ctx, request, nil)
}

// TransitionWith composes later row-lock classes into the same short
// lifecycle transaction. The participant must follow the package lock order;
// it runs after the lifecycle row and before effect cleanup rows.
func (store *Store) TransitionWith(
	ctx context.Context,
	request TransitionRequest,
	participant TransitionParticipant,
) (Lifecycle, error) {
	if request.Owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	if request.ExpectedGeneration <= 0 || !validReason(request.Reason) || request.To == StateTerminal {
		return Lifecycle{}, fmt.Errorf("%w: invalid transition request", ErrInvalidTransition)
	}
	var updated Lifecycle
	err := store.fencer.WithExclusive(ctx, []syntax.DID{request.Owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			current, err := scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, request.Owner,
			))
			if err != nil {
				return err
			}
			if current.State == StateTerminal {
				return ErrTerminalOwner
			}
			if current.Generation != request.ExpectedGeneration {
				return ErrGenerationChanged
			}
			if err := ValidateTransition(current.State, request.To); err != nil {
				return err
			}
			now := store.now().UTC()
			authEpoch := current.AuthEpoch
			if transitionAdvancesAuthEpoch(current.State, request.To) {
				authEpoch++
			}
			updated, err = scanLifecycle(tx.QueryRow(fenceCtx, `
				UPDATE owner_lifecycles
				SET state=$2,generation=generation+1,auth_epoch=$3,
				    transition_reason=$4,transitioned_at=$5,terminal_at=NULL,
				    purge_completed_at=NULL,updated_at=$5
				WHERE owner_did=$1 AND generation=$6
				RETURNING owner_did,state,generation,auth_epoch,transition_reason,
				          transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
			`, request.Owner, request.To, authEpoch, strings.TrimSpace(request.Reason), now, request.ExpectedGeneration))
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrGenerationChanged
			}
			if err != nil {
				return fmt.Errorf("update owner lifecycle: %w", err)
			}
			if participant != nil {
				if err := participant(fenceCtx, tx, current, updated); err != nil {
					return err
				}
			}
			if current.State == StateActive && request.To != StateActive {
				if err := closeOwnerEffectsTx(fenceCtx, tx, request.Owner, current.Generation, false, now); err != nil {
					return err
				}
			}
			return nil
		})
	})
	return updated, err
}

// AdvanceAuthEpoch invalidates all credentials tied to the current epoch at
// one owner-scoped linearization point without changing lifecycle generation.
func (store *Store) AdvanceAuthEpoch(
	ctx context.Context,
	owner syntax.DID,
	expectedGeneration int64,
	reason string,
) (Lifecycle, error) {
	if owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	if expectedGeneration <= 0 || !validReason(reason) {
		return Lifecycle{}, fmt.Errorf("%w: invalid auth epoch request", ErrInvalidTransition)
	}
	var updated Lifecycle
	err := store.fencer.WithExclusive(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			current, err := scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, owner,
			))
			if err != nil {
				return err
			}
			if current.State == StateTerminal {
				return ErrTerminalOwner
			}
			if current.Generation != expectedGeneration {
				return ErrGenerationChanged
			}
			now := store.now().UTC()
			updated, err = scanLifecycle(tx.QueryRow(fenceCtx, `
				UPDATE owner_lifecycles
				SET auth_epoch=auth_epoch+1,transition_reason=$3,
				    transitioned_at=$4,updated_at=$4
				WHERE owner_did=$1 AND generation=$2
				RETURNING owner_did,state,generation,auth_epoch,transition_reason,
				          transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
			`, owner, expectedGeneration, strings.TrimSpace(reason), now))
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrGenerationChanged
			}
			return err
		})
	})
	return updated, err
}

type activeEffectsContextKey struct{}

type authTransitionContextKey struct{}

// WithOnboardingAuth owns the complete first-login security boundary. It
// creates the explicit departed authority when necessary, rejects a terminal
// tombstone, and keeps the exclusive owner fence held while callback performs
// bounded OAuth work and short local transactions. The callback must use
// WithAuthTransaction for database work so a MaxConns=1 pool cannot
// self-starve while its fence connection is checked out.
func (store *Store) WithOnboardingAuth(
	ctx context.Context,
	owner syntax.DID,
	callback func(context.Context, Lifecycle) error,
) error {
	if owner == "" {
		return ErrInvalidOwner
	}
	if callback == nil {
		return errors.New("invalid onboarding auth callback")
	}
	return store.fencer.WithExclusive(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		var lifecycle Lifecycle
		if err := store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			now := store.now().UTC()
			if _, err := tx.Exec(fenceCtx, `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES($1,'departed',1,1,'onboarding',$2,$2,$2)
				ON CONFLICT (owner_did) DO NOTHING
			`, owner, now); err != nil {
				return fmt.Errorf("ensure onboarding auth owner: %w", err)
			}
			var err error
			lifecycle, err = scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, owner,
			))
			if err != nil {
				return err
			}
			if lifecycle.State == StateTerminal {
				return ErrTerminalOwner
			}
			return nil
		}); err != nil {
			return err
		}
		authCtx := context.WithValue(fenceCtx, authTransitionContextKey{}, lifecycle)
		return callback(authCtx, lifecycle)
	})
}

// WithExistingAuth is the callback, redemption, confirmation, and logout-all
// counterpart to WithOnboardingAuth. It never creates authority: the owner row
// must already exist and remain non-terminal for the full exclusive scope.
func (store *Store) WithExistingAuth(
	ctx context.Context,
	owner syntax.DID,
	callback func(context.Context, Lifecycle) error,
) error {
	if owner == "" {
		return ErrInvalidOwner
	}
	if callback == nil {
		return errors.New("invalid existing auth callback")
	}
	return store.fencer.WithExclusive(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		var lifecycle Lifecycle
		if err := store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			var err error
			lifecycle, err = scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, owner,
			))
			if err != nil {
				return err
			}
			if lifecycle.State == StateTerminal {
				return ErrTerminalOwner
			}
			return nil
		}); err != nil {
			return err
		}
		authCtx := context.WithValue(fenceCtx, authTransitionContextKey{}, lifecycle)
		return callback(authCtx, lifecycle)
	})
}

// WithActiveSessionAuth holds a shared owner fence and one exclusive OAuth
// parent-session fence on the same dedicated connection. Owner transitions can
// therefore never cross the operation, while independent parents for the same
// active owner may proceed concurrently. The callback may use
// WithAuthTransaction for short persistence work; it must not open a pool
// transaction while the dedicated fence connection is held.
func (store *Store) WithActiveSessionAuth(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	callback func(context.Context, Lifecycle) error,
) error {
	return store.withSessionAuth(ctx, owner, sessionID, StateActive, ErrOwnerNotActive, callback)
}

// WithDeletionSessionAuth is the privileged counterpart to
// WithActiveSessionAuth. It admits only a deleting owner and otherwise uses
// the identical shared-owner/exclusive-parent advisory-lock order.
func (store *Store) WithDeletionSessionAuth(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	callback func(context.Context, Lifecycle) error,
) error {
	return store.withSessionAuth(ctx, owner, sessionID, StateDeleting, ErrOwnerNotDeleting, callback)
}

func (store *Store) withSessionAuth(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	expectedState State,
	stateError error,
	callback func(context.Context, Lifecycle) error,
) error {
	if owner == "" || sessionID == "" {
		return ErrInvalidOwner
	}
	if callback == nil {
		return errors.New("invalid session auth callback")
	}
	return store.fencer.WithShared(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		conn, ok := fencedConnection(fenceCtx)
		if !ok {
			return ErrFenceRequired
		}
		lifecycle, err := scanLifecycle(conn.QueryRow(
			fenceCtx, lifecycleSelect+` WHERE owner_did=$1`, owner,
		))
		if err != nil {
			return err
		}
		if lifecycle.State != expectedState {
			if lifecycle.State == StateTerminal {
				return ErrTerminalOwner
			}
			return stateError
		}
		authCtx := context.WithValue(fenceCtx, authTransitionContextKey{}, lifecycle)
		return store.fencer.withParentSession(authCtx, owner, sessionID, func(parentCtx context.Context) error {
			return callback(parentCtx, lifecycle)
		})
	})
}

// WithAuthTransaction runs one short transaction on the connection already
// holding the exclusive owner fence. Remote I/O must stay outside callback.
func (store *Store) WithAuthTransaction(ctx context.Context, callback func(pgx.Tx) error) error {
	if callback == nil {
		return errors.New("invalid auth transaction")
	}
	if _, ok := ctx.Value(authTransitionContextKey{}).(Lifecycle); !ok {
		return ErrFenceRequired
	}
	return store.beginFenced(ctx, callback)
}

// WithOnboardingEffect admits only the exact departed lifecycle already held
// by WithOnboardingAuth or WithExistingAuth. It does not acquire another
// advisory lock. This narrow
// scope exists solely for the callback-time creation of the deterministic
// social.craftsky.actor.profile/self record before ordinary membership and
// session activation; it must not be used as a general non-active effect
// bypass.
func (store *Store) WithOnboardingEffect(
	ctx context.Context,
	owner syntax.DID,
	expectedGeneration int64,
	callback func(context.Context) error,
) error {
	if owner == "" || callback == nil {
		return errors.New("invalid onboarding effect request")
	}
	authority, ok := ctx.Value(authTransitionContextKey{}).(Lifecycle)
	if !ok {
		return ErrFenceRequired
	}
	if ctx.Value(activeEffectsContextKey{}) != nil {
		return ErrFenceReentry
	}
	if authority.Owner != owner {
		return ErrFenceRequired
	}
	if authority.Generation != expectedGeneration {
		return ErrGenerationChanged
	}
	if authority.State != StateDeparted {
		if authority.State == StateTerminal {
			return ErrTerminalOwner
		}
		return ErrOwnerNotOnboarding
	}
	effectCtx := context.WithValue(
		ctx,
		activeEffectsContextKey{},
		activeEffectScope{owner: expectedGeneration},
	)
	return callback(effectCtx)
}

type activeEffectScope map[syntax.DID]int64

func normalizeExpectedOwners(expected []ExpectedOwner) (activeEffectScope, []syntax.DID, error) {
	if len(expected) == 0 {
		return nil, nil, errors.New("invalid active effect request")
	}
	expectedByOwner := make(activeEffectScope, len(expected))
	owners := make([]syntax.DID, 0, len(expected))
	for _, item := range expected {
		if item.Owner == "" {
			return nil, nil, ErrInvalidOwner
		}
		if item.AllowMissing {
			if item.Generation != 0 {
				return nil, nil, ErrGenerationChanged
			}
		} else if item.Generation <= 0 {
			return nil, nil, ErrGenerationChanged
		}
		if generation, exists := expectedByOwner[item.Owner]; exists && generation != item.Generation {
			return nil, nil, ErrGenerationChanged
		}
		expectedByOwner[item.Owner] = item.Generation
		owners = append(owners, item.Owner)
	}
	return expectedByOwner, owners, nil
}

func (store *Store) validateActiveOwners(
	ctx context.Context,
	expectedByOwner activeEffectScope,
	owners []syntax.DID,
) (map[syntax.DID]Lifecycle, error) {
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	authorities := make(map[syntax.DID]Lifecycle, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := store.Get(ctx, owner)
		if expectedByOwner[owner] == 0 {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				continue
			case err != nil:
				return nil, err
			case lifecycle.State == StateTerminal:
				return nil, ErrTerminalOwner
			case lifecycle.State != StateActive:
				return nil, ErrOwnerNotActive
			default:
				return nil, ErrGenerationChanged
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOwnerNotActive
		}
		if err != nil {
			return nil, err
		}
		if lifecycle.Generation != expectedByOwner[owner] {
			return nil, ErrGenerationChanged
		}
		if lifecycle.State != StateActive {
			return nil, ErrOwnerNotActive
		}
		authorities[owner] = lifecycle
	}
	return authorities, nil
}

// NonTerminalParticipant performs a short local transaction while shared
// lifecycle fences prevent any listed owner from terminalizing. Existing
// non-terminal rows are supplied by DID; an absent row is deliberately omitted
// so the caller must apply its explicit missing-owner policy.
type NonTerminalParticipant func(
	context.Context,
	pgx.Tx,
	map[syntax.DID]Lifecycle,
) error

// WithNonTerminalOwners locks existing lifecycle rows in canonical DID order,
// rejects every terminal tombstone, and runs participant on the same database
// connection that holds the shared advisory fences. It is for local atomic
// writes such as trusted moderation intake; it must never contain remote I/O.
func (store *Store) WithNonTerminalOwners(
	ctx context.Context,
	owners []syntax.DID,
	participant NonTerminalParticipant,
) error {
	if participant == nil {
		return errors.New("invalid non-terminal owner transaction")
	}
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return err
	}
	return store.fencer.WithShared(ctx, canonical, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			existing := make(map[syntax.DID]Lifecycle, len(canonical))
			for _, owner := range canonical {
				lifecycle, err := scanLifecycle(tx.QueryRow(
					fenceCtx,
					lifecycleSelect+` WHERE owner_did=$1 FOR SHARE`,
					owner,
				))
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				if lifecycle.State == StateTerminal {
					return ErrTerminalOwner
				}
				existing[owner] = lifecycle
			}
			return participant(fenceCtx, tx, existing)
		})
	})
}

// WithActiveEffects acquires all owner fences in canonical DID order, then
// re-reads every lifecycle and expected generation before allowing the effect
// callback to cross a remote boundary. The shared fence remains held through
// the entire callback and its local completion CAS.
func (store *Store) WithActiveEffects(
	ctx context.Context,
	expected []ExpectedOwner,
	callback func(context.Context) error,
) error {
	if callback == nil {
		return errors.New("invalid active effect request")
	}
	expectedByOwner, owners, err := normalizeExpectedOwners(expected)
	if err != nil {
		return err
	}
	return store.fencer.WithShared(ctx, owners, func(fenceCtx context.Context) error {
		if _, err := store.validateActiveOwners(fenceCtx, expectedByOwner, owners); err != nil {
			return err
		}
		effectCtx := context.WithValue(fenceCtx, activeEffectsContextKey{}, expectedByOwner)
		return callback(effectCtx)
	})
}

// WithActiveEffectSessionAuth owns the complete ordinary authenticated-effect
// lock order: every participant owner fence in canonical DID order, then the
// operation owner's parent-session fence, all on one dedicated connection.
// The callback context admits both durable effect transactions and OAuth
// session persistence transactions without recursively acquiring an owner
// fence.
func (store *Store) WithActiveEffectSessionAuth(
	ctx context.Context,
	expected []ExpectedOwner,
	sessionOwner syntax.DID,
	sessionID string,
	callback func(context.Context, Lifecycle) error,
) error {
	if callback == nil || sessionOwner == "" || sessionID == "" {
		return errors.New("invalid active effect session request")
	}
	expectedByOwner, owners, err := normalizeExpectedOwners(expected)
	if err != nil {
		return err
	}
	if generation, exists := expectedByOwner[sessionOwner]; !exists || generation <= 0 {
		return ErrFenceRequired
	}
	return store.fencer.WithShared(ctx, owners, func(fenceCtx context.Context) error {
		authorities, err := store.validateActiveOwners(fenceCtx, expectedByOwner, owners)
		if err != nil {
			return err
		}
		authority := authorities[sessionOwner]
		effectCtx := context.WithValue(fenceCtx, activeEffectsContextKey{}, expectedByOwner)
		effectCtx = context.WithValue(effectCtx, authTransitionContextKey{}, authority)
		return store.fencer.withParentSession(
			effectCtx,
			sessionOwner,
			sessionID,
			func(parentCtx context.Context) error {
				return callback(parentCtx, authority)
			},
		)
	})
}

// WithActiveEffectTransaction runs a short local transaction on the same
// dedicated connection that holds the active-effect advisory fences. It lets
// a coordinator persist pre-call and completion state without acquiring a
// second pool connection (which would deadlock when MaxConns is one). Remote
// I/O must happen outside the callback while the surrounding
// WithActiveEffects scope remains active.
func (store *Store) WithActiveEffectTransaction(
	ctx context.Context,
	callback func(pgx.Tx) error,
) error {
	if callback == nil {
		return errors.New("invalid active effect transaction")
	}
	if _, ok := ctx.Value(activeEffectsContextKey{}).(activeEffectScope); !ok {
		return ErrFenceRequired
	}
	return store.beginFenced(ctx, callback)
}

const lifecycleSelect = `
	SELECT owner_did,state,generation,auth_epoch,transition_reason,
	       transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
	FROM owner_lifecycles`

type lifecycleRowScanner interface {
	Scan(dest ...any) error
}

func scanLifecycle(row lifecycleRowScanner) (Lifecycle, error) {
	var lifecycle Lifecycle
	err := row.Scan(
		&lifecycle.Owner,
		&lifecycle.State,
		&lifecycle.Generation,
		&lifecycle.AuthEpoch,
		&lifecycle.TransitionReason,
		&lifecycle.TransitionedAt,
		&lifecycle.TerminalAt,
		&lifecycle.PurgeCompletedAt,
		&lifecycle.CreatedAt,
		&lifecycle.UpdatedAt,
	)
	return lifecycle, err
}

func validReason(reason string) bool {
	trimmed := strings.TrimSpace(reason)
	return trimmed != "" && len([]rune(trimmed)) <= 256
}

func (store *Store) beginFenced(ctx context.Context, callback func(pgx.Tx) error) error {
	conn, ok := fencedConnection(ctx)
	if !ok {
		return ErrFenceRequired
	}
	return pgx.BeginFunc(ctx, conn.Conn(), callback)
}

func (store *Store) queryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	if conn, ok := fencedConnection(ctx); ok {
		return conn.QueryRow(ctx, sql, arguments...)
	}
	return store.pool.QueryRow(ctx, sql, arguments...)
}

func (store *Store) exec(ctx context.Context, sql string, arguments ...any) error {
	if conn, ok := fencedConnection(ctx); ok {
		_, err := conn.Exec(ctx, sql, arguments...)
		return err
	}
	_, err := store.pool.Exec(ctx, sql, arguments...)
	return err
}

// WithEffectTransaction runs a short transaction on the same dedicated
// connection that holds the active owner fence. It must not contain network
// I/O; use the surrounding WithActiveEffects callback for the remote call.
func (store *Store) WithEffectTransaction(
	ctx context.Context,
	callback func(pgx.Tx) error,
) error {
	if _, ok := ctx.Value(activeEffectsContextKey{}).(activeEffectScope); !ok || callback == nil {
		return ErrFenceRequired
	}
	return store.beginFenced(ctx, callback)
}
