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
	"social.craftsky/appview/internal/tap"
)

type CraftskyLike struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

var _ Indexer = (*CraftskyLike)(nil)

func NewCraftskyLike(pool *pgxpool.Pool, logger *slog.Logger) *CraftskyLike {
	if logger == nil {
		logger = slog.Default()
	}
	return &CraftskyLike{pool: pool, logger: logger}
}

const craftskyLikeNSID syntax.NSID = "social.craftsky.feed.like"

func (c *CraftskyLike) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyLikeNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return handleCraftskyInteractionUpsert(ctx, c.pool, ev, "craftsky_likes", decodeCraftskyLike)
	case "delete":
		return handleCraftskyInteractionDelete(ctx, c.pool, ev, "craftsky_likes")
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
