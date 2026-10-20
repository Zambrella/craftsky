package chat

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

var ErrDisabled = errors.New("messaging is disconnected")
var ErrRestricted = errors.New("messaging is restricted")

type SessionRunner interface {
	WithCapability(context.Context, syntax.DID, string, []string, auth.OAuthSessionOperation) error
}
type TransportFactory func(*oauth.ClientSession) (*Transport, error)
type ServiceOptions struct {
	Store      *Store
	Sessions   SessionRunner
	Transport  TransportFactory
	Operations *OperationLedger
	Now        func() time.Time
}
type Service struct {
	store      *Store
	sessions   SessionRunner
	transport  TransportFactory
	operations *OperationLedger
	now        func() time.Time
}

func (service *Service) OperationsAvailable() bool {
	return service != nil && service.operations != nil
}

func NewService(options ServiceOptions) (*Service, error) {
	if options.Store == nil || options.Sessions == nil || options.Transport == nil {
		return nil, errors.New("messaging service dependencies unavailable")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Service{store: options.Store, sessions: options.Sessions, transport: options.Transport, operations: options.Operations, now: options.Now}, nil
}

func (service *Service) withRPC(ctx context.Context, info auth.AuthInfo, methods []syntax.NSID, operation func(context.Context, *Transport) error) error {
	required := rpcScopes(append([]syntax.NSID{"chat.bsky.actor.getStatus"}, methods...)...)
	return service.withRequired(ctx, info, required, operation)
}

func (service *Service) withRequired(ctx context.Context, info auth.AuthInfo, required []string, operation func(context.Context, *Transport) error) error {
	return service.sessions.WithCapability(ctx, info.DID, info.SessionID, required, func(ctx context.Context, session *oauth.ClientSession) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		if !connection.Enabled {
			return ErrDisabled
		}
		transport, err := service.transport(session)
		if err != nil {
			return err
		}
		var status struct {
			ChatDisabled *bool `json:"chatDisabled"`
		}
		if err := transport.Query(ctx, "chat.bsky.actor.getStatus", nil, &status); err != nil {
			return err
		}
		if status.ChatDisabled == nil {
			return ErrProviderResponse
		}
		if *status.ChatDisabled {
			return ErrRestricted
		}
		return operation(ctx, transport)
	})
}

type ConnectionState string

const (
	StateDisabled            ConnectionState = "disabled"
	StateAuthorizationNeeded ConnectionState = "authorizationNeeded"
	StateRestricted          ConnectionState = "restricted"
	StateUnavailable         ConnectionState = "unavailable"
	StateReady               ConnectionState = "ready"
)

type Status struct {
	SyncState     string          `json:"syncState,omitempty"`
	LastSuccessAt *time.Time      `json:"lastSuccessAt,omitempty"`
	RecoveryCode  string          `json:"recoveryCode,omitempty"`
	Generation    int64           `json:"generation"`
	Enabled       bool            `json:"enabled"`
	State         ConnectionState `json:"state"`
	PushEnabled   bool            `json:"pushEnabled"`
	Capabilities  Capabilities    `json:"capabilities"`
}

// accountInSession reuses the coordinator's already-held owner/session fence.
// It never acquires another owner lock or holds a SQL transaction across I/O.
func (service *Service) accountInSession(ctx context.Context, info auth.AuthInfo) (Connection, error) {
	owner := info.DID
	if info.ChildTokenHash == ([32]byte{}) {
		return Connection{}, auth.ErrAuthTokenInvalid
	}
	var result Connection
	err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		queries := chatdb.New(tx)
		live, err := queries.ChatLiveChild(ctx, chatdb.ChatLiveChildParams{AccountDid: owner.String(), OauthSessionID: info.SessionID, TokenHash: info.ChildTokenHash[:]})
		if err != nil {
			return err
		}
		if !live {
			return auth.ErrAuthTokenInvalid
		}
		if err := queries.EnsureChatAccount(ctx, owner.String()); err != nil {
			return err
		}
		row, err := queries.ChatConnection(ctx, owner.String())
		result = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
		return err
	})
	return result, err
}

func (service *Service) Status(ctx context.Context, info auth.AuthInfo) (Status, error) {
	connection, err := service.store.ConnectionForChild(ctx, info)
	if errors.Is(err, auth.ErrAuthTokenInvalid) {
		return Status{State: StateAuthorizationNeeded}, nil
	}
	if err != nil {
		return Status{}, err
	}
	result := Status{Generation: connection.Generation, Enabled: connection.Enabled, PushEnabled: connection.PushEnabled, State: StateDisabled}
	if !connection.Enabled {
		return result, nil
	}
	if !service.OperationsAvailable() {
		result.State = StateUnavailable
		return result, nil
	}
	err = service.sessions.WithCapability(ctx, info.DID, info.SessionID, coreChatScopes(), func(ctx context.Context, session *oauth.ClientSession) error {
		current, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		if !current.Enabled || current.Generation != connection.Generation {
			return ErrDisabled
		}
		transport, err := service.transport(session)
		if err != nil {
			return err
		}
		var status struct {
			ChatDisabled *bool `json:"chatDisabled"`
		}
		result.SyncState = "syncing"
		if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			snapshot, err := chatdb.New(tx).ChatSyncSnapshot(ctx, info.DID.String())
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			result.SyncState = snapshot.SafeState
			if snapshot.LastSuccessAt.Valid {
				at := snapshot.LastSuccessAt.Time.UTC()
				result.LastSuccessAt = &at
			}
			return nil
		}); err != nil {
			return err
		}
		queryErr := transport.Query(ctx, "chat.bsky.actor.getStatus", nil, &status)
		result.Capabilities = EvaluateCapabilities(session.Data.Scopes)
		if queryErr != nil {
			return queryErr
		}
		if status.ChatDisabled == nil {
			return ErrProviderResponse
		}
		if *status.ChatDisabled {
			result.State = StateRestricted
		} else {
			result.State = StateReady
		}
		return nil
	})
	switch {
	case errors.Is(err, auth.ErrAuthTokenInvalid), errors.Is(err, auth.ErrPDSPermissionsRequired), errors.Is(err, auth.ErrPDSSessionExpired), errors.Is(err, auth.ErrOAuthSessionNotFound):
		result.State = StateAuthorizationNeeded
	case errors.Is(err, ErrDisabled):
		result.State = StateDisabled
		result.Enabled = false
	case err != nil:
		result.State = StateUnavailable
	}
	if result.State != StateReady {
		result.RecoveryCode = string(result.State)
	} else if result.SyncState != "ready" && result.SyncState != "syncing" {
		result.RecoveryCode = result.SyncState
	}
	return result, nil
}
