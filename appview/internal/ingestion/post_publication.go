package ingestion

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/postpublication"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

func preparePostPublicationTx(ctx context.Context, tx pgx.Tx, event tap.Event, validation sourcevalidation.Result, orderingStatus, disposition string) (*postpublication.Facts, error) {
	if event.Collection != "social.craftsky.feed.post" || event.Action == "delete" || orderingStatus != "authoritative" || disposition == "denied_terminal" || validation.StructuralStatus != sourcevalidation.Valid || validation.SemanticStatus != sourcevalidation.Valid {
		return nil, nil
	}
	return postpublication.Prepare(ctx, tx, postpublication.Source{
		URI: event.URI, DID: event.DID, Record: event.Record,
		Live: event.Live, Action: event.Action,
	})
}

func restrictPostPublicationTx(ctx context.Context, tx pgx.Tx, event tap.Event, validation sourcevalidation.Result, orderingStatus, disposition string, now time.Time) error {
	if event.Collection != "social.craftsky.feed.post" || orderingStatus != "authoritative" || disposition == "denied_terminal" {
		return nil
	}
	if event.Action != "delete" && (validation.StructuralStatus != sourcevalidation.Valid || validation.SemanticStatus != sourcevalidation.Valid) {
		return nil
	}
	raw := event.Record
	if event.Action == "delete" {
		raw = nil
	}
	_, err := postpublication.RestrictCurrent(ctx, tx, event.URI, raw, now)
	return err
}
