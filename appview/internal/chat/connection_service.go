package chat

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

func (service *Service) SetConnection(ctx context.Context, info auth.AuthInfo, enabled bool) (Status, error) {
	var result Status
	if !enabled {
		err := service.store.withAccount(ctx, info.DID, func(ctx context.Context, q *chatdb.Queries) error {
			live, err := q.ChatLiveChild(ctx, chatdb.ChatLiveChildParams{AccountDid: info.DID.String(), OauthSessionID: info.SessionID, TokenHash: info.ChildTokenHash[:]})
			if err != nil {
				return err
			}
			if !live {
				return auth.ErrAuthTokenInvalid
			}
			row, err := q.DisconnectChat(ctx, info.DID.String())
			if err != nil {
				return err
			}
			result = Status{Generation: row.Generation, Enabled: false, State: StateDisabled, PushEnabled: row.PushEnabled}
			return nil
		})
		return result, err
	}
	if !service.OperationsAvailable() {
		return Status{}, ErrOperationUnavailable
	}
	err := service.sessions.WithCapability(ctx, info.DID, info.SessionID, auth.OptionalChatScopes(), func(ctx context.Context, session *oauth.ClientSession) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
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
		if !connection.Enabled {
			if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
				row, err := chatdb.New(tx).ConnectChat(ctx, chatdb.ConnectChatParams{OwnerDid: info.DID.String(), ExpectedGeneration: connection.Generation})
				if err != nil {
					return err
				}
				connection = Connection{Enabled: row.Enabled, Generation: row.Generation, PushEnabled: row.PushEnabled}
				return chatdb.New(tx).ResumeChatSync(ctx, info.DID.String())
			}); err != nil {
				return err
			}
		}
		result = Status{Generation: connection.Generation, Enabled: connection.Enabled, PushEnabled: connection.PushEnabled, State: StateReady, Capabilities: EvaluateCapabilities(session.Data.Scopes)}
		return nil
	})
	return result, err
}
