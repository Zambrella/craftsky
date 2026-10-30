package ownerlifecycle

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/postfence"
)

func (store *Store) WithPostPublicationFence(ctx context.Context, uri syntax.ATURI, callback func(context.Context) error) error {
	state, ok := ctx.Value(fenceConnectionContextKey{}).(*fenceConnectionState)
	if !ok || state == nil || state.conn == nil || state.discard || callback == nil {
		return ErrFenceRequired
	}
	uncertain := true
	defer func() {
		if uncertain {
			state.discard = true
		}
	}()
	discard, err := postfence.WithConnection(ctx, state.conn.Conn(), uri, store.fencer.acquireTimeout, callback)
	uncertain = false
	if discard {
		state.discard = true
	}
	return err
}
