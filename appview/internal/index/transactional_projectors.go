package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

var (
	_ TransactionalIndexer = (*CraftskyProfile)(nil)
	_ TransactionalIndexer = (*CraftskyPost)(nil)
	_ TransactionalIndexer = (*CraftskyLike)(nil)
	_ TransactionalIndexer = (*CraftskyRepost)(nil)
	_ TransactionalIndexer = (*BlueskyProfile)(nil)
	_ TransactionalIndexer = (*BlueskyFollow)(nil)
	_ TransactionalIndexer = (*BlueskyBlock)(nil)
)

type noopBlueskyBackfiller struct{}

func (noopBlueskyBackfiller) Backfill(context.Context, syntax.DID) error { return nil }

func (indexer *CraftskyProfile) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	clone := *indexer
	clone.projectionDB = tx
	// Durable tap_add_repo work was committed at source ingestion. Never make
	// a remote PDS/Tap call while the projection transaction holds locks.
	clone.backfiller = noopBlueskyBackfiller{}
	return appliedAfter(clone.Handle(ctx, event))
}

func (indexer *CraftskyPost) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	clone := *indexer
	clone.projectionDB = tx
	return appliedAfter(clone.Handle(ctx, event))
}

func (indexer *CraftskyLike) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	return projectInteraction(ctx, tx, event, decodeCraftskyLike, func() error {
		clone := *indexer
		clone.projectionDB = tx
		return clone.Handle(ctx, event)
	})
}

func (indexer *CraftskyRepost) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	return projectInteraction(ctx, tx, event, decodeCraftskyRepost, func() error {
		clone := *indexer
		clone.projectionDB = tx
		return clone.Handle(ctx, event)
	})
}

func (indexer *BlueskyProfile) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	clone := *indexer
	clone.projectionDB = tx
	return appliedAfter(clone.Handle(ctx, event))
}

func (indexer *BlueskyFollow) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	clone := *indexer
	clone.projectionDB = tx
	return appliedAfter(clone.Handle(ctx, event))
}

func (indexer *BlueskyBlock) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	clone := *indexer
	clone.projectionDB = tx
	return appliedAfter(clone.Handle(ctx, event))
}

func projectInteraction(
	ctx context.Context,
	tx pgx.Tx,
	event tap.Event,
	decode func(json.RawMessage) (craftskyInteractionRecord, error),
	project func() error,
) (tap.Outcome, error) {
	if event.Action == "delete" {
		return appliedAfter(project())
	}
	memberOutcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
	if err != nil || !ready {
		return memberOutcome, err
	}
	record, err := decode(event.Record)
	if err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	var subjectExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM craftsky_posts
			WHERE uri=$1 AND NOT appview_owner_is_terminal(did)
		)
	`, record.SubjectURI).Scan(&subjectExists); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("check interaction subject: %w", err)
	}
	if !subjectExists {
		return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: record.SubjectURI}), nil
	}
	return appliedAfter(project())
}

func projectionMemberReady(ctx context.Context, tx pgx.Tx, did syntax.DID) (tap.Outcome, bool, error) {
	var member, terminal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1),
		       appview_owner_is_terminal($1)
	`, did).Scan(&member, &terminal); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), false, fmt.Errorf("check projection membership: %w", err)
	}
	if terminal {
		return tap.PermanentInvalid(tap.ReasonOwnerTerminal), false, nil
	}
	if !member {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: did.String()}), false, nil
	}
	return tap.Outcome{}, true, nil
}

func appliedAfter(err error) (tap.Outcome, error) {
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	return tap.Applied(), nil
}
