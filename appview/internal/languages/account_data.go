package languages

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleIdentityDeleted removes private language preferences during the
// terminal Tap identity-deletion lifecycle. Deleting an already-absent row is
// deliberately successful so the lifecycle remains retry-safe.
func (s *Store) HandleIdentityDeleted(ctx context.Context, did syntax.DID) error {
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM account_language_preferences
		WHERE account_did = $1
	`, did); err != nil {
		return fmt.Errorf("delete language preferences: %w", err)
	}
	return nil
}
