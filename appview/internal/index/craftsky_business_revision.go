package index

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

const (
	businessProfileCollection syntax.NSID = "social.craftsky.business.profile"
	businessEventCollection   syntax.NSID = "social.craftsky.business.event"
)

func lockBusinessRecordRevision(ctx context.Context, tx pgx.Tx, table string, event tap.Event) (bool, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, event.URI); err != nil {
		return false, fmt.Errorf("lock business record %s: %w", event.URI, err)
	}
	var rowRevision, tombstoneRevision *string
	rowQuery := fmt.Sprintf(`SELECT source_revision FROM %s WHERE uri=$1 FOR UPDATE`, table)
	var value string
	if err := tx.QueryRow(ctx, rowQuery, event.URI).Scan(&value); err == nil {
		rowRevision = &value
	} else if err != pgx.ErrNoRows {
		return false, fmt.Errorf("read business record revision %s: %w", event.URI, err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT source_revision
		FROM craftsky_business_record_tombstones
		WHERE uri=$1
		FOR UPDATE
	`, event.URI).Scan(&value); err == nil {
		tombstoneRevision = &value
	} else if err != pgx.ErrNoRows {
		return false, fmt.Errorf("read business tombstone revision %s: %w", event.URI, err)
	}
	newest := ""
	if rowRevision != nil {
		newest = *rowRevision
	}
	if tombstoneRevision != nil && *tombstoneRevision > newest {
		newest = *tombstoneRevision
	}
	return string(event.Rev) > newest, nil
}

func writeBusinessTombstone(ctx context.Context, tx pgx.Tx, event tap.Event) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO craftsky_business_record_tombstones
			(uri, owner_did, collection, source_revision)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (uri) DO UPDATE SET
			owner_did = EXCLUDED.owner_did,
			collection = EXCLUDED.collection,
			source_revision = EXCLUDED.source_revision,
			deleted_at = now()
		WHERE craftsky_business_record_tombstones.source_revision < EXCLUDED.source_revision
	`, event.URI, event.DID, event.Collection, event.Rev)
	if err != nil {
		return fmt.Errorf("write business tombstone %s: %w", event.URI, err)
	}
	return nil
}
