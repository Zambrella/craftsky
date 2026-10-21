package chat

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

type NotificationTarget struct {
	Type           string         `json:"type"`
	ConversationID ConversationID `json:"conversationId"`
}

func (NotificationTarget) String() string   { return "ChatNotificationTarget([REDACTED])" }
func (NotificationTarget) GoString() string { return "ChatNotificationTarget([REDACTED])" }
func (service *Service) NotificationOpen(ctx context.Context, info auth.AuthInfo, reference uuid.UUID) (NotificationTarget, error) {
	var result NotificationTarget
	if reference == uuid.Nil {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo"}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		var target chatdb.ResolveChatNotificationOpenRow
		err = service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			var err error
			target, err = chatdb.New(tx).ResolveChatNotificationOpen(ctx, chatdb.ResolveChatNotificationOpenParams{OwnerDid: info.DID.String(), OpenReference: dbUUID(reference)})
			return err
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInaccessibleConversation
		}
		if err != nil {
			return err
		}
		if target.MessagingGeneration != connection.Generation {
			return ErrInaccessibleConversation
		}
		id, err := ParseConversationID(target.ConversationID)
		if err != nil {
			return ErrInaccessibleConversation
		}
		if _, err := service.supportedConversation(ctx, transport, info.DID, id); err != nil {
			return err
		}
		result = NotificationTarget{Type: "conversation", ConversationID: id}
		return nil
	})
	return result, err
}
