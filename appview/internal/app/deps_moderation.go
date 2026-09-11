package app

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
)

type moderationDependencies struct {
	store   *moderation.Store
	service *moderation.Service
	expiry  *moderation.ExpiryWorker
}

func newModerationDependencies(pool *pgxpool.Pool, visibility *api.ModerationStore, observer *observability.Observer, batchSize int) moderationDependencies {
	store := moderation.NewStore(pool)
	writer := notifications.ModerationWriter{}
	return moderationDependencies{
		store: store,
		service: moderation.NewService(
			store,
			visibility,
			time.Now,
			moderation.WithNotificationWriter(writer),
			moderation.WithOperationObserver(observer),
		),
		expiry: moderation.NewExpiryWorker(moderation.NewExpiryProcessor(store, writer, time.Now, observer), batchSize),
	}
}
