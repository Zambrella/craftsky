package chat

import (
	"context"
	"time"
)

func (dispatcher *DeliveryDispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	errorDelay := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		processed, err := dispatcher.ProcessOne(ctx)
		if err == nil {
			errorDelay = time.Second
			if processed {
				continue
			}
		}
		if err != nil {
			// ProcessOne owns its private diagnostic. Back off queue/storage errors
			// without duplicating reports or spinning on one unhealthy account.
			timer := time.NewTimer(errorDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			errorDelay = min(30*time.Second, errorDelay*2)
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
