package index_test

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/index"
)

// fakeBackfiller is used by CraftskyProfile tests in Chunk 4 but we also
// verify here that it satisfies the exported interface.
type fakeBackfiller struct {
	calls []syntax.DID
	err   error
}

func (f *fakeBackfiller) Backfill(_ context.Context, did syntax.DID) error {
	f.calls = append(f.calls, did)
	return f.err
}

func TestBlueskyBackfiller_InterfaceShape(t *testing.T) {
	var _ index.BlueskyBackfiller = (*fakeBackfiller)(nil)
}
