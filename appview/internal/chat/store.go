package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrConnectionChanged = errors.New("messaging connection changed")

type Connection struct {
	Enabled     bool
	Generation  int64
	PushEnabled bool
}

// Store holds only private operational metadata, never message/report bodies.
type Store struct {
	pool   *pgxpool.Pool
	owners *ownerlifecycle.Store
}

func NewStore(pool *pgxpool.Pool, owners *ownerlifecycle.Store) (*Store, error) {
	if pool == nil || owners == nil {
		return nil, errors.New("messaging store dependencies unavailable")
	}
	return &Store{pool: pool, owners: owners}, nil
}

// withAccount is an outer owner-fence boundary, not a nested session helper.
// Callback/session consumers already inside a fence must instead use the
// existing owner's WithAuthTransaction with transaction-scoped queries.
func (store *Store) withAccount(ctx context.Context, owner syntax.DID, operation func(context.Context, *chatdb.Queries) error) error {
	if store == nil || owner == "" {
		return ownerlifecycle.ErrInvalidOwner
	}
	return store.owners.WithExistingAuth(ctx, owner, func(ctx context.Context, authority ownerlifecycle.Lifecycle) error {
		if authority.State != ownerlifecycle.StateActive {
			return ownerlifecycle.ErrOwnerNotActive
		}
		return store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			queries := chatdb.New(tx)
			if err := queries.EnsureChatAccount(ctx, owner.String()); err != nil {
				return fmt.Errorf("ensure messaging account: %w", err)
			}
			return operation(ctx, queries)
		})
	})
}

func (store *Store) Connection(ctx context.Context, owner syntax.DID) (Connection, error) {
	var result Connection
	err := store.withAccount(ctx, owner, func(ctx context.Context, queries *chatdb.Queries) error {
		row, err := queries.ChatConnection(ctx, owner.String())
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}

func (store *Store) ConnectionForChild(ctx context.Context, info auth.AuthInfo) (Connection, error) {
	if info.ChildTokenHash == ([32]byte{}) {
		return Connection{}, auth.ErrAuthTokenInvalid
	}
	var result Connection
	err := store.withAccount(ctx, info.DID, func(ctx context.Context, q *chatdb.Queries) error {
		live, err := q.ChatLiveChild(ctx, chatdb.ChatLiveChildParams{AccountDid: info.DID.String(), OauthSessionID: info.SessionID, TokenHash: info.ChildTokenHash[:]})
		if err != nil {
			return err
		}
		if !live {
			return auth.ErrAuthTokenInvalid
		}
		row, err := q.ChatConnection(ctx, info.DID.String())
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}

// Connect is a local metadata transition. Its caller must establish effective
// consent before admission; expectedGeneration prevents late callbacks from
// undoing a disconnect. Capability/session validation stays in the service.
func (store *Store) Connect(ctx context.Context, owner syntax.DID, expectedGeneration int64, push *bool) (Connection, error) {
	var result Connection
	err := store.withAccount(ctx, owner, func(ctx context.Context, queries *chatdb.Queries) error {
		choice := pgtype.Bool{}
		if push != nil {
			choice = pgtype.Bool{Bool: *push, Valid: true}
		}
		row, err := queries.ConnectChat(ctx, chatdb.ConnectChatParams{OwnerDid: owner.String(), ExpectedGeneration: expectedGeneration, PushEnabled: choice})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConnectionChanged
		}
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}

func (store *Store) Disconnect(ctx context.Context, owner syntax.DID) (Connection, error) {
	var result Connection
	err := store.withAccount(ctx, owner, func(ctx context.Context, queries *chatdb.Queries) error {
		row, err := queries.DisconnectChat(ctx, owner.String())
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}
