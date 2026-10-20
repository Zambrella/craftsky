package chat

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
)

func (service *Service) Receipt(ctx context.Context, info auth.AuthInfo, id uuid.UUID) (OperationReceipt, error) {
	var result OperationReceipt
	if id == uuid.Nil {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{}, func(ctx context.Context, _ *Transport) error {
		return service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			q := chatdb.New(tx)
			operationID := pgtype.UUID{Bytes: id, Valid: true}
			if _, err := q.ExpirePreparedChatOperation(ctx, chatdb.ExpirePreparedChatOperationParams{OwnerDid: info.DID.String(), OperationID: operationID}); err != nil {
				return err
			}
			row, err := q.GetChatOperation(ctx, chatdb.GetChatOperationParams{OwnerDid: info.DID.String(), OperationID: operationID})
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInaccessibleConversation
			}
			if err != nil {
				return err
			}
			result = receiptFor(row)
			return nil
		})
	})
	return result, err
}
