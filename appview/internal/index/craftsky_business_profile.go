package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

type CraftskyBusinessProfile struct{}

var _ TransactionalIndexer = (*CraftskyBusinessProfile)(nil)

func NewCraftskyBusinessProfile() *CraftskyBusinessProfile {
	return &CraftskyBusinessProfile{}
}

func (*CraftskyBusinessProfile) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if event.Collection != businessProfileCollection {
		return tap.Applied(), nil
	}
	newer, err := lockBusinessRecordRevision(ctx, tx, "craftsky_business_profiles", event)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if !newer {
		return tap.Applied(), nil
	}
	switch event.Action {
	case "create", "update":
		var record craftskylex.BusinessProfile
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO craftsky_business_profiles
				(owner_did, uri, cid, raw_record, source_revision)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (owner_did) DO UPDATE SET
				uri = EXCLUDED.uri,
				cid = EXCLUDED.cid,
				raw_record = EXCLUDED.raw_record,
				source_revision = EXCLUDED.source_revision,
				indexed_at = now()
		`, event.DID, event.URI, event.CID, event.Record, event.Rev); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("upsert business profile %s: %w", event.URI, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_business_record_tombstones WHERE uri=$1`, event.URI); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("clear business profile tombstone %s: %w", event.URI, err)
		}
	case "delete":
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_business_profiles WHERE uri=$1`, event.URI); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("delete business profile %s: %w", event.URI, err)
		}
		if err := writeBusinessTombstone(ctx, tx, event); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
	default:
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	return tap.Applied(), nil
}
