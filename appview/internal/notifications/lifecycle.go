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
}

type Retraction struct {
	SourceURI syntax.ATURI
	Reason    string
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

func (NoopLifecycle) Activate(context.Context, pgx.Tx, Activation) error { return nil }
func (NoopLifecycle) Retract(context.Context, pgx.Tx, Retraction) error  { return nil }

type NoopActorDeletion struct{}

func (NoopActorDeletion) HardDeleteByActor(context.Context, pgx.Tx, syntax.DID) error { return nil }
