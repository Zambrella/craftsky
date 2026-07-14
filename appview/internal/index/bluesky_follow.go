// appview/internal/index/bluesky_follow.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
)

const blueskyFollowNSID syntax.NSID = "app.bsky.graph.follow"

// BlueskyFollow indexes app.bsky.graph.follow events into atproto_follows.
// Required invariant: idempotent on (URI, CID).
type BlueskyFollow struct {
	pool      *pgxpool.Pool
	lifecycle notifications.Lifecycle
}

var _ Indexer = (*BlueskyFollow)(nil)

func NewBlueskyFollow(pool *pgxpool.Pool, lifecycles ...notifications.Lifecycle) *BlueskyFollow {
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return &BlueskyFollow{pool: pool, lifecycle: lifecycle}
}

type blueskyFollowRecord struct {
	Subject   syntax.DID `json:"subject"`
	CreatedAt string     `json:"createdAt"`
}

func (b *BlueskyFollow) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyFollowNSID {
		return nil
	}

	switch ev.Action {
	case "create", "update":
		var rec blueskyFollowRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		if rec.Subject == "" {
			return fmt.Errorf("follow record missing subject on %s", ev.URI)
		}
		createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
		if err != nil {
			return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
		}

		if err := b.upsertActive(ctx, ev, rec.Subject, createdAt); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		tx, err := b.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin delete %s: %w", ev.URI, err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx,
			`DELETE FROM atproto_follows WHERE uri = $1`, ev.URI); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		if err := b.lifecycle.Retract(ctx, tx, notifications.Retraction{SourceURI: ev.URI, Reason: "sourceDeleted"}); err != nil {
			return fmt.Errorf("retract notification for %s: %w", ev.URI, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (b *BlueskyFollow) upsertActive(ctx context.Context, ev tap.Event, subject syntax.DID, createdAt time.Time) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM atproto_follows
		WHERE did = $1 AND subject_did = $2 AND uri <> $3
	`, ev.DID, subject, ev.URI); err != nil {
		return fmt.Errorf("collapse duplicate pair: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM atproto_follows
		WHERE did = $1 AND rkey = $2 AND uri <> $3
	`, ev.DID, ev.Rkey, ev.URI); err != nil {
		return fmt.Errorf("collapse duplicate rkey: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO atproto_follows
			(uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (uri) DO UPDATE SET
			did = EXCLUDED.did,
			rkey = EXCLUDED.rkey,
			cid = EXCLUDED.cid,
			subject_did = EXCLUDED.subject_did,
			record = EXCLUDED.record,
			created_at = EXCLUDED.created_at,
			indexed_at = now()
	`, ev.URI, ev.DID, ev.Rkey, ev.CID, subject, ev.Record, createdAt); err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	if subject != ev.DID {
		var recipientIsMember bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)`, subject).Scan(&recipientIsMember); err != nil {
			return fmt.Errorf("check notification recipient membership: %w", err)
		}
		if recipientIsMember {
			if err := b.lifecycle.Activate(ctx, tx, notifications.Activation{
				RecipientDID: subject,
				ActorDID:     ev.DID,
				Category:     notifications.Follow,
				SubjectKey:   subject.String(),
				SourceURI:    ev.URI,
				SourceCID:    ev.CID,
				SourceRkey:   ev.Rkey,
				ActivityAt:   createdAt,
			}); err != nil {
				return fmt.Errorf("activate notification: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
