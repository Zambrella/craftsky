// appview/internal/index/bluesky_profile.go
package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// BlueskyProfile indexes app.bsky.actor.profile events into the
// bluesky_profiles table for Craftsky and non-Craftsky accounts.
// Required invariant: idempotent on (DID, CID).
type BlueskyProfile struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*BlueskyProfile)(nil)

// NewBlueskyProfile builds an indexer backed by the given pool.
func NewBlueskyProfile(pool *pgxpool.Pool) *BlueskyProfile {
	return &BlueskyProfile{pool: pool}
}

const blueskyProfileNSID syntax.NSID = "app.bsky.actor.profile"

// blueskyBlobRef is the atproto blob-reference shape carried inside an
// app.bsky.actor.profile record. We only need the CID link and MIME type.
type blueskyBlobRef struct {
	Ref struct {
		Link string `json:"$link"`
	} `json:"ref"`
	MimeType string `json:"mimeType"`
}

type blueskyProfileRecord struct {
	DisplayName *string         `json:"displayName,omitempty"`
	Description *string         `json:"description,omitempty"`
	Avatar      *blueskyBlobRef `json:"avatar,omitempty"`
	Banner      *blueskyBlobRef `json:"banner,omitempty"`
}

func (b *BlueskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyProfileNSID {
		return nil
	}

	switch ev.Action {
	case "create", "update":
		var rec blueskyProfileRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		const q = `
			INSERT INTO bluesky_profiles
				(did, display_name, description,
				 avatar_cid, avatar_mime, banner_cid, banner_mime, record_cid)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (did) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description  = EXCLUDED.description,
				avatar_cid   = EXCLUDED.avatar_cid,
				avatar_mime  = EXCLUDED.avatar_mime,
				banner_cid   = EXCLUDED.banner_cid,
				banner_mime  = EXCLUDED.banner_mime,
				record_cid   = EXCLUDED.record_cid,
				indexed_at   = now()
			WHERE bluesky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
		`
		var (
			avatarCID, avatarMime *string
			bannerCID, bannerMime *string
		)
		if rec.Avatar != nil && rec.Avatar.Ref.Link != "" {
			avatarCID = &rec.Avatar.Ref.Link
			avatarMime = &rec.Avatar.MimeType
		}
		if rec.Banner != nil && rec.Banner.Ref.Link != "" {
			bannerCID = &rec.Banner.Ref.Link
			bannerMime = &rec.Banner.MimeType
		}
		if _, err := b.pool.Exec(ctx, q,
			ev.DID, rec.DisplayName, rec.Description,
			avatarCID, avatarMime, bannerCID, bannerMime, ev.CID); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.pool.Exec(ctx,
			`DELETE FROM bluesky_profiles WHERE did = $1`, ev.DID); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (b *BlueskyProfile) isMember(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	err := b.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}
