// appview/internal/api/post_relationship_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/relationships"
)

func (s *PostStore) RequireCurrentMember(ctx context.Context, did syntax.DID) error {
	return relationships.RequireCurrentMember(ctx, relationships.NewStore(s.pool), did)
}

func (s *PostStore) AuthorizeDirectedInteraction(
	ctx context.Context,
	actor syntax.DID,
	subject syntax.DID,
	operation relationships.Operation,
) error {
	started := time.Now()
	metricOperation := directedAuthorizationMetricOperation(operation)
	relationshipStore := relationships.NewStore(s.pool)
	if err := relationships.RequireCurrentMember(ctx, relationshipStore, subject); err != nil {
		if s.observer != nil {
			errorClass := "store"
			if errors.Is(err, relationships.ErrProfileNotFound) {
				errorClass = "membership"
			}
			s.observer.ObserveRelationshipOutcome(metricOperation, "membership", "error", errorClass, time.Since(started))
		}
		return err
	}
	state, err := relationshipStore.State(ctx, actor, subject)
	if err != nil {
		if s.observer != nil {
			s.observer.ObserveRelationshipOutcome(metricOperation, "store", "error", "store", time.Since(started))
		}
		return err
	}
	if !relationships.Authorize(operation, state, false).Allowed {
		if s.observer != nil {
			s.observer.ObserveRelationshipOutcome(metricOperation, "policy", "denied", "policy", time.Since(started))
		}
		return ErrInteractionBlocked
	}
	return nil
}

func directedAuthorizationMetricOperation(operation relationships.Operation) string {
	switch operation {
	case relationships.OperationFollowCreate:
		return "authorization_follow"
	case relationships.OperationLikeCreate:
		return "authorization_like"
	case relationships.OperationRepostCreate:
		return "authorization_repost"
	case relationships.OperationReplyCreate:
		return "authorization_reply"
	case relationships.OperationQuoteCreate:
		return "authorization_quote"
	case relationships.OperationMentionCreate:
		return "authorization_mention"
	default:
		return "authorization"
	}
}

func (s *PostStore) RelationshipState(ctx context.Context, viewer, subject syntax.DID) (relationships.State, error) {
	states, err := s.RelationshipStates(ctx, viewer, []syntax.DID{subject})
	if err != nil {
		return relationships.State{}, err
	}
	state, ok := states[subject]
	if !ok {
		return relationships.State{}, relationships.ErrProfileNotFound
	}
	return state, nil
}

func (s *PostStore) RelationshipStates(ctx context.Context, viewer syntax.DID, subjects []syntax.DID) (map[syntax.DID]relationships.State, error) {
	out := make(map[syntax.DID]relationships.State)
	if len(subjects) == 0 {
		return out, nil
	}
	values := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		values = append(values, subject.String())
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cp.did,
			EXISTS (SELECT 1 FROM actor_mutes mute
			        WHERE mute.owner_did = $1 AND mute.subject_did = cp.did
			          AND NOT appview_owner_is_terminal(mute.owner_did)
			          AND NOT appview_owner_is_terminal(mute.subject_did)),
			EXISTS (SELECT 1 FROM atproto_blocks block
			        WHERE block.blocker_did = $1 AND block.subject_did = cp.did
			          AND NOT appview_owner_is_terminal(block.blocker_did)
			          AND NOT appview_owner_is_terminal(block.subject_did)),
			EXISTS (SELECT 1 FROM atproto_blocks block
			        WHERE block.blocker_did = cp.did AND block.subject_did = $1
			          AND NOT appview_owner_is_terminal(block.blocker_did)
			          AND NOT appview_owner_is_terminal(block.subject_did))
		FROM craftsky_profiles cp
		WHERE cp.did = ANY($2)
		  AND NOT appview_owner_is_terminal(cp.did)
	`, viewer, values)
	if err != nil {
		return nil, fmt.Errorf("batch post relationship state: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var did syntax.DID
		var state relationships.State
		if err := rows.Scan(&did, &state.Muted, &state.Blocking, &state.BlockedBy); err != nil {
			return nil, fmt.Errorf("scan post relationship state: %w", err)
		}
		out[did] = state
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post relationship state: %w", err)
	}
	return out, nil
}

func (s *PostStore) BlockedPairs(ctx context.Context, pairs []RelationshipPair) (map[RelationshipPair]bool, error) {
	out := make(map[RelationshipPair]bool, len(pairs))
	if len(pairs) == 0 {
		return out, nil
	}
	first := make([]string, 0, len(pairs))
	second := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		first = append(first, pair.First.String())
		second = append(second, pair.Second.String())
	}
	rows, err := s.pool.Query(ctx, `
		SELECT pair.first_did, pair.second_did,
			EXISTS (
				SELECT 1 FROM atproto_blocks block
				WHERE ((block.blocker_did = pair.first_did AND block.subject_did = pair.second_did)
				   OR (block.blocker_did = pair.second_did AND block.subject_did = pair.first_did))
				  AND NOT appview_owner_is_terminal(block.blocker_did)
				  AND NOT appview_owner_is_terminal(block.subject_did)
			)
		FROM unnest($1::text[], $2::text[]) AS pair(first_did, second_did)
	`, first, second)
	if err != nil {
		return nil, fmt.Errorf("batch reference block pairs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pair RelationshipPair
		var blocked bool
		if err := rows.Scan(&pair.First, &pair.Second, &blocked); err != nil {
			return nil, fmt.Errorf("scan reference block pair: %w", err)
		}
		out[pair] = blocked
	}
	return out, rows.Err()
}
