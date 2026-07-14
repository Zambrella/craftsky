// appview/internal/index/craftsky_like.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
)

type CraftskyLike struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	lifecycle notifications.Lifecycle
}

var _ Indexer = (*CraftskyLike)(nil)

func NewCraftskyLike(pool *pgxpool.Pool, logger *slog.Logger, lifecycles ...notifications.Lifecycle) *CraftskyLike {
	if logger == nil {
		logger = slog.Default()
	}
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &CraftskyLike{pool: pool, logger: logger, lifecycle: lifecycle}
}

const craftskyLikeNSID syntax.NSID = "social.craftsky.feed.like"

func (c *CraftskyLike) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyLikeNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return handleCraftskyInteractionUpsert(ctx, c.pool, ev, "craftsky_likes", notifications.Like, c.lifecycle, decodeCraftskyLike)
	case "delete":
		return handleCraftskyInteractionDelete(ctx, c.pool, ev, "craftsky_likes", c.lifecycle)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func decodeCraftskyLike(raw json.RawMessage) (craftskyInteractionRecord, error) {
	var rec craftskylex.FeedLike
	if err := json.Unmarshal(raw, &rec); err != nil {
		return craftskyInteractionRecord{}, err
	}
	if rec.Subject == nil {
		return craftskyInteractionRecord{CreatedAt: rec.CreatedAt}, nil
	}
	return craftskyInteractionRecord{
		CreatedAt:  rec.CreatedAt,
		SubjectURI: rec.Subject.Uri,
		SubjectCID: rec.Subject.Cid,
	}, nil
}
