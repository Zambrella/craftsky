// appview/internal/index/craftsky_repost.go
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

type CraftskyRepost struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	lifecycle notifications.Lifecycle
}

var _ Indexer = (*CraftskyRepost)(nil)

func NewCraftskyRepost(pool *pgxpool.Pool, logger *slog.Logger, lifecycles ...notifications.Lifecycle) *CraftskyRepost {
	if logger == nil {
		logger = slog.Default()
	}
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &CraftskyRepost{pool: pool, logger: logger, lifecycle: lifecycle}
}

const craftskyRepostNSID syntax.NSID = "social.craftsky.feed.repost"

func (c *CraftskyRepost) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyRepostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return handleCraftskyInteractionUpsert(ctx, c.pool, ev, "craftsky_reposts", notifications.Repost, c.lifecycle, decodeCraftskyRepost)
	case "delete":
		return handleCraftskyInteractionDelete(ctx, c.pool, ev, "craftsky_reposts", c.lifecycle)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func decodeCraftskyRepost(raw json.RawMessage) (craftskyInteractionRecord, error) {
	var rec craftskylex.FeedRepost
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
