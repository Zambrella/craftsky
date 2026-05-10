// appview/internal/index/craftsky_interaction.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

type craftskyInteractionRecord struct {
	CreatedAt  string
	SubjectURI string
	SubjectCID string
}

func handleCraftskyInteractionUpsert(
	ctx context.Context,
	pool *pgxpool.Pool,
	ev tap.Event,
	table string,
	decode func(json.RawMessage) (craftskyInteractionRecord, error),
) error {
	isMember, err := isCraftskyMember(ctx, pool, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		return nil
	}

	rec, err := decode(ev.Record)
	if err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	if rec.SubjectURI == "" || rec.SubjectCID == "" {
		return nil
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", ev.URI, err)
	}
	defer tx.Rollback(ctx)

	var subjectExists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_posts WHERE uri = $1)`, rec.SubjectURI).
		Scan(&subjectExists); err != nil {
		return fmt.Errorf("subject check %s: %w", rec.SubjectURI, err)
	}
	if !subjectExists {
		return nil
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET deleted_at = now(), indexed_at = now()
		WHERE did = $1 AND subject_uri = $2 AND deleted_at IS NULL AND uri <> $3
	`, table), ev.DID, rec.SubjectURI, ev.URI); err != nil {
		return fmt.Errorf("soft-delete duplicate %s: %w", ev.URI, err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(uri, did, rkey, cid, subject_uri, subject_cid, record, created_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL)
		ON CONFLICT (uri) DO UPDATE SET
			cid         = EXCLUDED.cid,
			subject_uri = EXCLUDED.subject_uri,
			subject_cid = EXCLUDED.subject_cid,
			record      = EXCLUDED.record,
			created_at  = EXCLUDED.created_at,
			indexed_at  = now(),
			deleted_at  = NULL
	`, table),
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.SubjectURI, rec.SubjectCID,
		ev.Record, createdAt,
	); err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", ev.URI, err)
	}
	return nil
}

func handleCraftskyInteractionDelete(ctx context.Context, pool *pgxpool.Pool, ev tap.Event, table string) error {
	if _, err := pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET deleted_at = now(), indexed_at = now()
		WHERE uri = $1 AND deleted_at IS NULL
	`, table), ev.URI); err != nil {
		return fmt.Errorf("soft-delete %s: %w", ev.URI, err)
	}
	return nil
}

func isCraftskyMember(ctx context.Context, pool *pgxpool.Pool, did syntax.DID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}
