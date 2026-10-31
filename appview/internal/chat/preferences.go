package chat

import (
	"context"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

type PushPreference struct {
	PushEnabled bool `json:"pushEnabled"`
}

func (service *Service) PushPreference(ctx context.Context, info auth.AuthInfo) (PushPreference, error) {
	connection, err := service.store.ConnectionForChild(ctx, info)
	return PushPreference{PushEnabled: connection.PushEnabled}, err
}
func (service *Service) SetPushPreference(ctx context.Context, info auth.AuthInfo, enabled bool) (PushPreference, error) {
	err := service.store.withAccount(ctx, info.DID, func(ctx context.Context, q *chatdb.Queries) error {
		live, err := q.ChatLiveChild(ctx, chatdb.ChatLiveChildParams{AccountDid: info.DID.String(), OauthSessionID: info.SessionID, TokenHash: info.ChildTokenHash[:]})
		if err != nil {
			return err
		}
		if !live {
			return auth.ErrAuthTokenInvalid
		}
		return q.SetChatPushPreference(ctx, chatdb.SetChatPushPreferenceParams{OwnerDid: info.DID.String(), PushEnabled: enabled})
	})
	return PushPreference{PushEnabled: enabled}, err
}
