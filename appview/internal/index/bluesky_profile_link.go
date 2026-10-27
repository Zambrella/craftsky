package index

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/profilelinks"
	"social.craftsky/appview/internal/tap"
)

type BlueskyProfileLink struct{}

func NewBlueskyProfileLink() TransactionalIndexer { return &BlueskyProfileLink{} }

func (*BlueskyProfileLink) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	if source.Action == "delete" {
		_, err := tx.Exec(ctx, `DELETE FROM bluesky_profile_links WHERE uri=$1`, source.URI)
		return appliedAfter(err)
	}
	_, err := tx.Exec(ctx, `
  INSERT INTO bluesky_profile_links(uri,did,cid,record) VALUES($1,$2,$3,$4)
  ON CONFLICT(uri) DO UPDATE SET cid=EXCLUDED.cid,record=EXCLUDED.record,indexed_at=now()
  WHERE bluesky_profile_links.cid IS DISTINCT FROM EXCLUDED.cid
 `, source.URI, source.DID, source.CID, source.Record)
	return appliedAfter(err)
}

func NewImageSafetyBlueskyProfileLink(next TransactionalIndexer, key imagesafety.ScanKey) TransactionalIndexer {
	return &profileLinkImageSafety{records: next, safety: &imageSafetySubject{next: profileLinkProjectionComplete{}, key: key, kind: "profile_link", extract: func(raw json.RawMessage) ([]imageRequirement, error) {
		var record profilelinks.Record
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, err
		}
		if record.Icon == nil {
			return nil, nil
		}
		return []imageRequirement{{slot: "icon", blob: record.Icon}}, nil
	}}}
}

// Keep the owner's editable record while its icon is scanning. Public reads
// independently require the matching image subject to be clear, so raw blobs
// are never exposed as serving image URLs before clearance.
type profileLinkImageSafety struct {
	records TransactionalIndexer
	safety  TransactionalIndexer
}

func (p *profileLinkImageSafety) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	outcome, err := p.records.Project(ctx, tx, source)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		return outcome, err
	}
	return p.safety.Project(ctx, tx, source)
}

type profileLinkProjectionComplete struct{}

func (profileLinkProjectionComplete) Project(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
	return tap.Applied(), nil
}
