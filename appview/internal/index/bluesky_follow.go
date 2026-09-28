package index

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
)

const blueskyFollowNSID syntax.NSID = "app.bsky.graph.follow"

// BlueskyFollow projects normalized per-URI facts and logical follow aggregates.
type BlueskyFollow struct {
	lifecycle notifications.Lifecycle
}

func NewBlueskyFollow(_ *pgxpool.Pool, lifecycles ...notifications.Lifecycle) *BlueskyFollow {
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &BlueskyFollow{lifecycle: lifecycle}
}

type blueskyFollowRecord struct {
	Subject   syntax.DID `json:"subject"`
	CreatedAt string     `json:"createdAt"`
}
