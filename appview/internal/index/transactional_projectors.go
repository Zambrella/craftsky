package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
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

// NewTransactionalCraftskyProfile builds the profile projector used by the
// durable ingestion pipeline. Repository tracking and profile backfill are
// already committed as repository jobs during source ingestion, so this
// projector must never perform remote work while its transaction holds locks.
func NewTransactionalCraftskyProfile(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	actorDeletion notifications.ActorDeletion,
) *CraftskyProfile {
	return NewCraftskyProfile(pool, noopBlueskyBackfiller{}, logger, actorDeletion)
}

func (indexer *CraftskyProfile) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
	clone := *indexer
	clone.projectionDB = tx
	// Durable tap_add_repo work was committed at source ingestion. Never make
	// a remote PDS/Tap call while the projection transaction holds locks.
	clone.backfiller = noopBlueskyBackfiller{}
	return appliedAfter(clone.Handle(ctx, event))
}

func (indexer *CraftskyPost) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
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

func (indexer *CraftskyLike) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
	changes, err := projectSetSourceTx(ctx, tx, source, source.UpdatedAt)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	return projectInteraction(ctx, tx, event, decodeCraftskyLike, func() error {
		return applyInteractionSetChanges(ctx, tx, indexer.lifecycle, event, changes, notifications.Like, decodeCraftskyLike)
	})
}

func applyInteractionSetChanges(
	ctx context.Context,
	tx pgx.Tx,
	lifecycle notifications.Lifecycle,
	event tap.Event,
	changes []SetAggregateChange,
	category notifications.Category,
	decode func(json.RawMessage) (craftskyInteractionRecord, error),
) error {
	for _, change := range changes {
		var activation notifications.Activation
		var retraction notifications.Retraction
		switch change.Transition {
		case SetActivated:
			record, err := decode(event.Record)
			if err != nil {
				return fmt.Errorf("decode activated %s %s: %w", category, event.URI, err)
			}
			activityAt, err := time.Parse(time.RFC3339Nano, record.CreatedAt)
			if err != nil {
				return fmt.Errorf("parse activated %s timestamp %s: %w", category, event.URI, err)
			}
			var recipientDID syntax.DID
			var subjectCID syntax.CID
			var rootURI syntax.ATURI
			var rootCID syntax.CID
			if err := tx.QueryRow(ctx, `
				SELECT did,cid,COALESCE(reply_root_uri,uri),COALESCE(reply_root_cid,cid)
				FROM craftsky_posts WHERE uri=$1 AND NOT appview_owner_is_terminal(did)
			`, change.Scope.Key).Scan(&recipientDID, &subjectCID, &rootURI, &rootCID); err != nil {
				return fmt.Errorf("read activated like subject %s: %w", change.Scope.Key, err)
			}
			activation = notifications.Activation{
				RecipientDID: recipientDID,
				ActorDID:     change.Scope.Actor,
				Category:     category,
				SubjectKey:   change.Scope.Key,
				SourceURI:    event.URI,
				SourceCID:    event.CID,
				SourceRkey:   event.Rkey,
				SubjectURI:   syntax.ATURI(change.Scope.Key),
				SubjectCID:   subjectCID,
				RootURI:      rootURI,
				RootCID:      rootCID,
				ActivityAt:   activityAt,
			}
		case SetDeactivated:
			var recipientDID syntax.DID
			err := tx.QueryRow(ctx, `
				SELECT recipient_did FROM notification_events
				WHERE actor_did=$1 AND category=$2 AND subject_key=$3
				LIMIT 1
			`, change.Scope.Actor, category, change.Scope.Key).Scan(&recipientDID)
			if err == pgx.ErrNoRows {
				continue
			}
			if err != nil {
				return fmt.Errorf("read deactivated %s notification %s: %w", category, change.Scope.Key, err)
			}
			retraction = notifications.Retraction{
				RecipientDID: recipientDID,
				ActorDID:     change.Scope.Actor,
				Category:     category,
				SubjectKey:   change.Scope.Key,
				Reason:       "setDeactivated",
			}
		}
		if err := notifications.ApplySetTransition(
			ctx, tx, lifecycle, notifications.SetTransition(change.Transition), activation, retraction,
		); err != nil {
			return fmt.Errorf("apply %s notification transition for %s: %w", category, change.Scope.Key, err)
		}
	}
	return nil
}

func (indexer *CraftskyRepost) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
	changes, err := projectSetSourceTx(ctx, tx, source, source.UpdatedAt)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	return projectInteraction(ctx, tx, event, decodeCraftskyRepost, func() error {
		return applyInteractionSetChanges(ctx, tx, indexer.lifecycle, event, changes, notifications.Repost, decodeCraftskyRepost)
	})
}

func (indexer *BlueskyProfile) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
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

func (indexer *BlueskyFollow) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
	changes, err := projectSetSourceTx(ctx, tx, source, source.UpdatedAt)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	if err := applyFollowSetChanges(ctx, tx, indexer.lifecycle, event, changes); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	return tap.Applied(), nil
}

func applyFollowSetChanges(
	ctx context.Context,
	tx pgx.Tx,
	lifecycle notifications.Lifecycle,
	event tap.Event,
	changes []SetAggregateChange,
) error {
	for _, change := range changes {
		var activation notifications.Activation
		var retraction notifications.Retraction
		switch change.Transition {
		case SetActivated:
			recipient, err := syntax.ParseDID(change.Scope.Key)
			if err != nil {
				return fmt.Errorf("parse activated follow recipient %s: %w", change.Scope.Key, err)
			}
			if recipient == change.Scope.Actor {
				continue
			}
			var recipientIsMember bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM craftsky_profiles
					WHERE did=$1 AND NOT appview_owner_is_terminal(did)
				)
			`, recipient).Scan(&recipientIsMember); err != nil {
				return fmt.Errorf("check logical follow recipient membership: %w", err)
			}
			if !recipientIsMember {
				continue
			}
			var record blueskyFollowRecord
			if err := json.Unmarshal(event.Record, &record); err != nil {
				return fmt.Errorf("decode activated follow %s: %w", event.URI, err)
			}
			activityAt, err := time.Parse(time.RFC3339Nano, record.CreatedAt)
			if err != nil {
				return fmt.Errorf("parse activated follow timestamp %s: %w", event.URI, err)
			}
			activation = notifications.Activation{
				RecipientDID: recipient,
				ActorDID:     change.Scope.Actor,
				Category:     notifications.Follow,
				SubjectKey:   change.Scope.Key,
				SourceURI:    event.URI,
				SourceCID:    event.CID,
				SourceRkey:   event.Rkey,
				ActivityAt:   activityAt,
			}
		case SetDeactivated:
			var recipient syntax.DID
			err := tx.QueryRow(ctx, `
				SELECT recipient_did FROM notification_events
				WHERE actor_did=$1 AND category=$2 AND subject_key=$3
				LIMIT 1
			`, change.Scope.Actor, notifications.Follow, change.Scope.Key).Scan(&recipient)
			if err == pgx.ErrNoRows {
				continue
			}
			if err != nil {
				return fmt.Errorf("read deactivated follow notification %s: %w", change.Scope.Key, err)
			}
			retraction = notifications.Retraction{
				RecipientDID: recipient,
				ActorDID:     change.Scope.Actor,
				Category:     notifications.Follow,
				SubjectKey:   change.Scope.Key,
				Reason:       "setDeactivated",
			}
		}
		if err := notifications.ApplySetTransition(
			ctx, tx, lifecycle, notifications.SetTransition(change.Transition), activation, retraction,
		); err != nil {
			return fmt.Errorf("apply follow notification transition for %s: %w", change.Scope.Key, err)
		}
	}
	return nil
}

func (indexer *BlueskyBlock) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	event := eventFromSource(source)
	changes, err := projectSetSourceTx(ctx, tx, source, source.UpdatedAt)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if event.Action != "delete" {
		outcome, ready, err := projectionMemberReady(ctx, tx, event.DID)
		if err != nil || !ready {
			return outcome, err
		}
	}
	clone := *indexer
	clone.projectionDB = tx
	clone.skipDeliveryCancellation = true
	for _, change := range changes {
		if change.Transition != SetActivated {
			continue
		}
		subject, err := syntax.ParseDID(change.Scope.Key)
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if err := clone.cancelPendingDeliveries(ctx, tx, change.Scope.Actor, subject); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
	}
	return tap.Applied(), nil
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

func eventFromSource(source ingestion.SourceRecord) tap.Event {
	return tap.Event{
		URI: source.URI, CID: source.CID, DID: source.DID,
		Collection: source.Collection, Rkey: source.Rkey,
		Action: source.Action, Record: source.Record, Live: source.Live,
		ID: source.SourceEventID, Rev: source.Revision,
	}
}
