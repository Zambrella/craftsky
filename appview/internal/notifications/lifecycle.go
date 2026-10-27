package notifications

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

type Activation struct {
	RecipientDID syntax.DID
	ActorDID     syntax.DID
	Category     Category
	SubjectKey   string
	SourceURI    syntax.ATURI
	SourceCID    syntax.CID
	SourceRkey   syntax.RecordKey
	SubjectURI   syntax.ATURI
	SubjectCID   syntax.CID
	ParentURI    syntax.ATURI
	ParentCID    syntax.CID
	RootURI      syntax.ATURI
	RootCID      syntax.CID
	QuotedURI    syntax.ATURI
	QuotedCID    syntax.CID
	ActivityAt   time.Time
	publication  bool
}

type Retraction struct {
	SourceURI    syntax.ATURI
	RecipientDID syntax.DID
	ActorDID     syntax.DID
	Category     Category
	SubjectKey   string
	Reason       string
}

// Lifecycle participates in a source indexer's transaction. Implementations
// must not perform provider or other network work from these methods.
type Lifecycle interface {
	Activate(context.Context, pgx.Tx, Activation) error
	Retract(context.Context, pgx.Tx, Retraction) error
}

type ActorDeletion interface {
	HardDeleteByActor(context.Context, pgx.Tx, syntax.DID) error
}

type NoopLifecycle struct{}

// ActivatePublication applies publication-specific duplicate rules while
// retaining the existing lifecycle abstraction for indexers.
func ActivatePublication(ctx context.Context, lifecycle Lifecycle, tx pgx.Tx, activation Activation) error {
	activation.publication = true
	return lifecycle.Activate(ctx, tx, activation)
}

// RestrictPublication retracts withdrawn reasons without reactivating existing
// events or changing their activity/read state. Retraction cancels only pending
// delivery work; completed provider sends cannot be recalled.
func RestrictPublication(ctx context.Context, lifecycle Lifecycle, tx pgx.Tx, uri syntax.ATURI, allowed map[syntax.DID]bool) error {
	rows, err := tx.Query(ctx, `SELECT recipient_did,actor_did,category,subject_key FROM notification_events
		WHERE source_uri=$1 AND state='active' AND category IN ('reply','mention','quote')
		ORDER BY recipient_did,category`, uri)
	if err != nil {
		return err
	}
	type existingPublication struct {
		Recipient  syntax.DID
		Actor      syntax.DID
		Category   Category
		SubjectKey string
	}
	events, err := pgx.CollectRows(rows, pgx.RowToStructByPos[existingPublication])
	if err != nil {
		return err
	}
	for _, event := range events {
		if !allowed[event.Recipient] {
			if err := lifecycle.Retract(ctx, tx, Retraction{SourceURI: uri, RecipientDID: event.Recipient, ActorDID: event.Actor, Category: event.Category, SubjectKey: event.SubjectKey, Reason: "publicationReasonRemoved"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (NoopLifecycle) Activate(context.Context, pgx.Tx, Activation) error { return nil }
func (NoopLifecycle) Retract(context.Context, pgx.Tx, Retraction) error  { return nil }

type NoopActorDeletion struct{}

func (NoopActorDeletion) HardDeleteByActor(context.Context, pgx.Tx, syntax.DID) error { return nil }
