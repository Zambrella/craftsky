package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const maxEffectReconciliationPage = 1000

// UnresolvedPDSAttempts returns one newest unresolved Put representative per
// URI. Paging by URI prevents duplicate/A-to-B-to-A attempts from consuming
// separate remote reads or being reconciled against each other's content.
func (store *Store) UnresolvedPDSAttempts(
	ctx context.Context,
	owner syntax.DID,
	limit int,
) ([]EffectAttempt, bool, error) {
	if store == nil || owner == "" || limit <= 0 || limit > maxEffectReconciliationPage {
		return nil, false, errors.New("invalid unresolved PDS effect query")
	}
	rows, err := store.pool.Query(ctx, effectAttemptSelect+`
		WHERE operation_id IN (
			SELECT DISTINCT ON (deterministic_key) operation_id
			FROM owner_effect_attempts
			WHERE owner_did=$1
			  AND effect_kind='pds_record'
			  AND effect_action='put_record'
			  AND remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
			ORDER BY deterministic_key,mutation_sequence DESC
		)
		ORDER BY deterministic_key
		LIMIT $2
	`, owner, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list unresolved PDS effects: %w", err)
	}
	defer rows.Close()
	attempts := make([]EffectAttempt, 0, limit)
	for rows.Next() {
		if len(attempts) == limit {
			return attempts, true, nil
		}
		attempt, err := scanEffectAttempt(rows)
		if err != nil {
			return nil, false, err
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate unresolved PDS effects: %w", err)
	}
	return attempts, false, nil
}
