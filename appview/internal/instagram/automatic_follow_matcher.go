package instagram

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AutomaticFollowMatcher struct {
	pool   *pgxpool.Pool
	store  *AutomaticFollowStore
	policy InstagramSuggestionEligibilityPolicy
	now    func() time.Time
	newID  func() uuid.UUID
}

func NewAutomaticFollowMatcher(
	pool *pgxpool.Pool,
	store *AutomaticFollowStore,
	policy InstagramSuggestionEligibilityPolicy,
	now func() time.Time,
) *AutomaticFollowMatcher {
	if now == nil {
		now = time.Now
	}
	return &AutomaticFollowMatcher{
		pool: pool, store: store, policy: policy, now: now, newID: uuid.New,
	}
}

func (m *AutomaticFollowMatcher) MatchImport(
	ctx context.Context,
	owner syntax.DID,
	importID uuid.UUID,
) (int, error) {
	if m == nil || m.pool == nil || m.store == nil || m.policy == nil ||
		owner == "" || importID == uuid.Nil {
		return 0, errors.New("Instagram automatic-follow matcher unavailable")
	}
	rows, err := m.pool.Query(ctx, `
		SELECT DISTINCT handle.username_normalized, link.owner_did
		FROM instagram_graph_imports import
		JOIN instagram_graph_handles handle ON handle.import_id=import.id
		JOIN instagram_account_links link
		  ON link.username_normalized=handle.username_normalized
		 AND link.state='active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE import.id=$1
		  AND import.owner_did=$2
		  AND import.state='active'
		ORDER BY handle.username_normalized, link.owner_did
	`, importID, owner)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type candidate struct {
		username string
		target   syntax.DID
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.username, &item.target); err != nil {
			return 0, err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	created := 0
	for _, candidate := range candidates {
		request := SuggestionEligibilityRequest{
			ImporterDID: owner, TargetDID: candidate.target,
			ImportedUsername: candidate.username,
		}
		eligible := true
		for _, stage := range []EligibilityStage{EligibilityAtMatch, EligibilityAtPersist} {
			decision, err := m.policy.Evaluate(ctx, stage, request)
			if err != nil {
				return 0, err
			}
			if !decision.Eligible {
				eligible = false
				break
			}
		}
		if !eligible {
			continue
		}
		id := m.newID()
		result, err := m.store.ReconcileCandidate(ctx, ReconcileAutomaticFollowParams{
			ID:          id,
			ImporterDID: owner,
			TargetDID:   candidate.target,
			ImportID:    importID,
			Username:    candidate.username,
			Rkey:        syntax.RecordKey("3l" + strings.ReplaceAll(id.String(), "-", "")),
			Now:         m.now().UTC(),
		})
		if err != nil {
			return 0, err
		}
		if result.Created {
			created++
		}
	}
	return created, nil
}

var _ ImportMatcher = (*AutomaticFollowMatcher)(nil)
