package api

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// RecentMemberRow contains only the presentation needed by discovery.
// Join metadata remains private and is used exclusively for SQL ordering.
type RecentMemberRow struct {
	Crafts      []string
	DID         syntax.DID
	Handle      syntax.Handle
	DisplayName *string
	AvatarCID   *string
	AvatarMime  *string
}

func (s *ProfileStore) ListRecentMembers(ctx context.Context, viewer syntax.DID) ([]RecentMemberRow, error) {
	rows, err := s.pool.Query(ctx, `
 SELECT cp.did, COALESCE(NULLIF(ic.handle,''),'handle.invalid'), NULLIF(btrim(bp.display_name),''), bp.avatar_cid, bp.avatar_mime, cp.crafts
 FROM craftsky_profiles cp
 JOIN account_onboarding_completions completion ON completion.account_did=cp.did
 LEFT JOIN atproto_identity_cache ic ON ic.did=cp.did
 LEFT JOIN bluesky_profiles bp ON bp.did=cp.did
 WHERE appview_owner_is_active(cp.did)
 AND cp.did <> $1
 AND NOT EXISTS (
   SELECT 1 FROM actor_mutes m
   WHERE m.owner_did=$1 AND m.subject_did=cp.did
     AND NOT appview_owner_is_terminal(m.owner_did)
     AND NOT appview_owner_is_terminal(m.subject_did)
 )
 AND NOT EXISTS (
   SELECT 1 FROM pds_set_aggregates b
   WHERE b.kind='block'
     AND ((b.actor_did=$1 AND b.subject_did=cp.did) OR (b.actor_did=cp.did AND b.subject_did=$1))
     AND NOT appview_owner_is_terminal(b.actor_did)
     AND NOT appview_owner_is_terminal(b.subject_did)
 )
 `+profileVisibleModerationPredicate+`
 ORDER BY completion.completed_at DESC, cp.did COLLATE "C" ASC
 LIMIT 5`, viewer)
	if err != nil {
		return nil, fmt.Errorf("list recent members: %w", err)
	}
	defer rows.Close()
	result := make([]RecentMemberRow, 0, 5)
	for rows.Next() {
		var row RecentMemberRow
		if err := rows.Scan(&row.DID, &row.Handle, &row.DisplayName, &row.AvatarCID, &row.AvatarMime, &row.Crafts); err != nil {
			return nil, fmt.Errorf("scan recent member: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recent members: %w", err)
	}
	return result, nil
}
