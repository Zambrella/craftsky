package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuggestionSupportWriter interface {
	UpsertPendingSuggestion(context.Context, UpsertSuggestionParams) (bool, error)
}

type SuggestionMatcher struct {
	pool   *pgxpool.Pool
	store  SuggestionSupportWriter
	policy InstagramSuggestionEligibilityPolicy
	now    func() time.Time
	newID  func() uuid.UUID
}

func NewSuggestionMatcher(pool *pgxpool.Pool, store SuggestionSupportWriter, policy InstagramSuggestionEligibilityPolicy, now func() time.Time) *SuggestionMatcher {
	if now == nil {
		now = time.Now
	}
	return &SuggestionMatcher{pool: pool, store: store, policy: policy, now: now, newID: uuid.New}
}

func (m *SuggestionMatcher) MatchImport(ctx context.Context, owner syntax.DID, importID uuid.UUID) (int, error) {
	if m == nil || m.pool == nil || m.store == nil || m.policy == nil {
		return 0, errors.New("Instagram suggestion matcher unavailable")
	}
	rows, err := m.pool.Query(ctx, `
		SELECT DISTINCT h.username_normalized, link.owner_did
		FROM instagram_graph_imports i
		JOIN instagram_graph_handles h
		  ON h.import_id = i.id AND h.direction = 'following'
		JOIN instagram_account_links link
		  ON link.username_normalized = h.username_normalized
		 AND link.state = 'active'
		 AND link.discoverable
		 AND NOT link.conflict_pending
		WHERE i.id = $1 AND i.owner_did = $2 AND i.state = 'active'
		ORDER BY h.username_normalized, link.owner_did
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
		var username, target string
		if err := rows.Scan(&username, &target); err != nil {
			return 0, err
		}
		candidates = append(candidates, candidate{username: username, target: syntax.DID(target)})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	createdCount := 0
	for _, candidate := range candidates {
		request := SuggestionEligibilityRequest{
			ImporterDID: owner, TargetDID: candidate.target,
			ImportedUsername: candidate.username, Direction: DirectionFollowing,
		}
		decision, err := m.policy.Evaluate(ctx, EligibilityAtMatch, request)
		if err != nil {
			return 0, err
		}
		if !decision.Eligible {
			continue
		}
		decision, err = m.policy.Evaluate(ctx, EligibilityAtPersist, request)
		if err != nil {
			return 0, err
		}
		if !decision.Eligible {
			continue
		}
		created, err := m.store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
			ID: m.newID(), ImporterDID: owner, TargetDID: candidate.target,
			ImportID: importID, Username: candidate.username, Now: m.now().UTC(),
		})
		if err != nil {
			return 0, err
		}
		if created {
			createdCount++
		}
	}
	return createdCount, nil
}

var _ ImportSuggestionMatcher = (*SuggestionMatcher)(nil)
