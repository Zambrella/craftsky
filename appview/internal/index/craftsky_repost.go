// appview/internal/index/craftsky_repost.go
package index

import (
	"encoding/json"
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/notifications"
)

type CraftskyRepost struct {
	logger    *slog.Logger
	lifecycle notifications.Lifecycle
}

func NewCraftskyRepost(_ *pgxpool.Pool, logger *slog.Logger, lifecycles ...notifications.Lifecycle) *CraftskyRepost {
	if logger == nil {
		logger = slog.Default()
	}
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &CraftskyRepost{logger: logger, lifecycle: lifecycle}
}

const craftskyRepostNSID syntax.NSID = "social.craftsky.feed.repost"

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
