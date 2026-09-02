package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

type CraftskyBusinessEvent struct{}

var _ TransactionalIndexer = (*CraftskyBusinessEvent)(nil)

func NewCraftskyBusinessEvent() *CraftskyBusinessEvent {
	return &CraftskyBusinessEvent{}
}

func (*CraftskyBusinessEvent) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Collection != businessEventCollection {
		return tap.Applied(), nil
	}
	newer, err := lockBusinessRecordRevision(ctx, tx, "craftsky_business_events", event)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if !newer {
		return tap.Applied(), nil
	}
	switch event.Action {
	case "create", "update":
		var record craftskylex.BusinessEvent
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		startsAt, startErr := time.Parse(time.RFC3339Nano, record.StartsAt)
		endsAt, endErr := time.Parse(time.RFC3339Nano, record.EndsAt)
		createdAt, createdErr := time.Parse(time.RFC3339Nano, record.CreatedAt)
		if startErr != nil || endErr != nil || createdErr != nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO craftsky_business_events
				(uri, owner_did, rkey, cid, raw_record, source_revision,
				 starts_at, ends_at, created_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (uri) DO UPDATE SET
				owner_did = EXCLUDED.owner_did,
				rkey = EXCLUDED.rkey,
				cid = EXCLUDED.cid,
				raw_record = EXCLUDED.raw_record,
				source_revision = EXCLUDED.source_revision,
				starts_at = EXCLUDED.starts_at,
				ends_at = EXCLUDED.ends_at,
				created_at = EXCLUDED.created_at,
				status = EXCLUDED.status,
				indexed_at = now()
		`, event.URI, event.DID, event.Rkey, event.CID, event.Record, event.Rev,
			startsAt, endsAt, createdAt, record.Status); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("upsert business event %s: %w", event.URI, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_business_record_tombstones WHERE uri=$1`, event.URI); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("clear business event tombstone %s: %w", event.URI, err)
		}
	case "delete":
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_business_events WHERE uri=$1`, event.URI); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("delete business event %s: %w", event.URI, err)
		}
		if err := writeBusinessTombstone(ctx, tx, event); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
	default:
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	return tap.Applied(), nil
}
