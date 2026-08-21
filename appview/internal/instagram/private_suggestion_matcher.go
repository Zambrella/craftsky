package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PrivateSuggestionMatcher performs exact private matching and has only a
// narrow persistence capability. It cannot select an OAuth session or call a
// PDS writer.
type PrivateSuggestionMatcher struct {
	pool   *pgxpool.Pool
	store  *PrivateSuggestionStore
	policy InstagramSuggestionEligibilityPolicy
	now    func() time.Time
	newID  func() uuid.UUID
}

func NewPrivateSuggestionMatcher(
	pool *pgxpool.Pool,
	store *PrivateSuggestionStore,
	policy InstagramSuggestionEligibilityPolicy,
	now func() time.Time,
) *PrivateSuggestionMatcher {
	if now == nil {
		now = time.Now
	}
	return &PrivateSuggestionMatcher{
		pool: pool, store: store, policy: policy, now: now, newID: uuid.New,
	}
}

func (matcher *PrivateSuggestionMatcher) MatchImport(
	ctx context.Context,
	owner syntax.DID,
	importID uuid.UUID,
) (int, error) {
	if matcher == nil || matcher.pool == nil || matcher.store == nil || matcher.policy == nil ||
		owner == "" || importID == uuid.Nil {
		return 0, errors.New("private Instagram suggestion matcher unavailable")
	}
	rows, err := matcher.pool.Query(ctx, `
		SELECT DISTINCT handle.username_normalized,link.owner_did
		FROM instagram_graph_imports AS import
		JOIN instagram_graph_handles AS handle ON handle.import_id=import.id
		JOIN instagram_account_links AS link
		  ON link.username_normalized=handle.username_normalized
		 AND link.state='active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE import.id=$1
		  AND import.owner_did=$2
		  AND import.state='active'
		ORDER BY handle.username_normalized,link.owner_did
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
			decision, err := matcher.policy.Evaluate(ctx, stage, request)
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
		result, err := matcher.store.ReconcileCandidate(ctx, ReconcilePrivateSuggestionParams{
			ID:          matcher.newID(),
			ImporterDID: owner,
			TargetDID:   candidate.target,
			ImportID:    importID,
			Username:    candidate.username,
			Now:         matcher.now().UTC(),
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

var _ ImportMatcher = (*PrivateSuggestionMatcher)(nil)
