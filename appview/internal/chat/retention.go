package chat

import (
	"context"
	"social.craftsky/appview/internal/chatdb"
	"time"
)

// Each statement has its own fixed 100-row budget. Lifetime operation,
// message-disposition and request-generation ledgers are never TTL-purged.
func (service *Service) Maintain(ctx context.Context) error {
	tx, err := service.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := chatdb.New(tx)
	for _, operation := range []func(context.Context) error{q.ExpireChatDeliveries, q.CompleteEmptyChatNotifications, q.PruneChatDeliveryDetails, q.PruneChatNotificationRouting, q.PruneExpiredChatViewActivity, q.PruneExpiredChatContinuations} {
		if err := operation(ctx); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Retention needs no usable provider session or operation secret. Paused or
// disabled messaging must not retain expired private routing indefinitely.
func (service *Service) RunMaintenance(ctx context.Context, observer SyncFailureObserver) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		operationCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := service.Maintain(operationCtx)
		cancel()
		if err != nil && ctx.Err() == nil && observer != nil {
			observer.ObservePrivateFailure(ctx, err, "", "", "chat.retention", "cleanup", "retry", 1)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
