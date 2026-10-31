package chat

import (
	"context"
	"errors"
	"time"

	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

func (service *Service) DeleteMessage(ctx context.Context, info auth.AuthInfo, id ConversationID, message MessageID) error {
	if !validOpaqueID(string(id)) || !validOpaqueID(string(message)) {
		return ErrInvalidIdentifier
	}
	return service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages", "chat.bsky.convo.deleteMessageForSelf"}, func(ctx context.Context, transport *Transport) error {
		connection, err := service.accountInSession(ctx, info)
		if err != nil {
			return err
		}
		if _, err := service.supportedConversation(ctx, transport, info.DID, id); err != nil {
			return err
		}
		var hint string
		if err := service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			var err error
			hint, err = chatdb.New(tx).FindChatMessagePage(ctx, chatdb.FindChatMessagePageParams{OwnerDid: info.DID.String(), Scope: "history:" + string(id), Generation: connection.Generation, Column4: string(message)})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}); err != nil {
			return err
		}
		current, err := currentMessage(ctx, transport, id, message, hint)
		if err != nil {
			return err
		}
		if current.Sender == nil || current.Sender.Did != info.DID.String() || ValidateMessageText(current.Text) != nil {
			return ErrInaccessibleConversation
		}
		var deleted upstream.ConvoDefs_DeletedMessageView
		if err := transport.Procedure(ctx, "chat.bsky.convo.deleteMessageForSelf", upstream.ConvoDeleteMessageForSelf_Input{ConvoId: string(id), MessageId: string(message)}, &deleted); err != nil {
			return err
		}
		before, beforeErr := time.Parse(time.RFC3339Nano, current.SentAt)
		after, afterErr := time.Parse(time.RFC3339Nano, deleted.SentAt)
		if deleted.Id != string(message) || deleted.Rev == "" || len(deleted.Rev) > 512 || deleted.Sender == nil || deleted.Sender.Did != info.DID.String() || beforeErr != nil || afterErr != nil || !before.Equal(after) {
			return ErrProviderResponse
		}
		return nil
	})
}
