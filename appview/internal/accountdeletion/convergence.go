package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConvergenceVerifier struct {
	pool *pgxpool.Pool
}

func NewConvergenceVerifier(pool *pgxpool.Pool) *ConvergenceVerifier {
	return &ConvergenceVerifier{pool: pool}
}

func (verifier *ConvergenceVerifier) IsConverged(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (bool, error) {
	if verifier == nil || verifier.pool == nil || jobID == uuid.Nil || owner == "" {
		return false, errors.New("invalid account deletion convergence scope")
	}
	var converged bool
	err := verifier.pool.QueryRow(ctx, `
		WITH scoped_expected AS (
			SELECT expected.uri,expected.collection,expected.delete_requested_at
			FROM account_deletion_expected_records expected
			JOIN account_deletion_operations operation ON operation.id=expected.job_id
			WHERE expected.job_id=$1 AND operation.owner_did=$2
		), blockers AS (
			SELECT expected.uri
			FROM scoped_expected expected
			LEFT JOIN account_deletion_index_receipts receipt
			  ON receipt.job_id=$1 AND receipt.uri=expected.uri
			WHERE expected.delete_requested_at IS NULL OR receipt.uri IS NULL
			UNION ALL
			SELECT post.uri FROM craftsky_posts post
			JOIN scoped_expected expected ON expected.uri=post.uri
			UNION ALL
			SELECT project.uri FROM craftsky_project_posts project
			JOIN scoped_expected expected ON expected.uri=project.uri
			UNION ALL
			SELECT mention.post_uri FROM craftsky_post_mentions mention
			JOIN scoped_expected expected ON expected.uri=mention.post_uri
			UNION ALL
			SELECT interaction.uri FROM craftsky_likes interaction
			JOIN scoped_expected expected ON expected.uri=interaction.uri
			UNION ALL
			SELECT interaction.uri FROM craftsky_reposts interaction
			JOIN scoped_expected expected ON expected.uri=interaction.uri
			UNION ALL
			SELECT profile.did FROM craftsky_profiles profile
			WHERE profile.did=$2 AND EXISTS(
				SELECT 1 FROM scoped_expected
				WHERE collection='social.craftsky.actor.profile'
			)
			UNION ALL
			SELECT event.id::text
			FROM notification_events event
			WHERE event.state='active' AND EXISTS(
				SELECT 1 FROM scoped_expected expected
				WHERE expected.uri IN (
					event.source_uri,event.subject_uri,event.parent_uri,
					event.root_uri,event.quoted_uri
				)
			)
		)
		SELECT NOT EXISTS(SELECT 1 FROM blockers)
	`, jobID, owner).Scan(&converged)
	if err != nil {
		return false, fmt.Errorf("verify account deletion convergence: %w", err)
	}
	return converged, nil
}
