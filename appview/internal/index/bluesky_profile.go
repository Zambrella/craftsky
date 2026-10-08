// appview/internal/index/bluesky_profile.go
package index

import (
	"context"
	"encoding/json"
	"fmt"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/tap"
)

// BlueskyProfile indexes app.bsky.actor.profile events into the
// bluesky_profiles table for Craftsky and non-Craftsky accounts.
// Required invariant: idempotent on (DID, CID).
type BlueskyProfile struct {
	pool         *pgxpool.Pool
	projectionDB transactionalDatabase
	imageScanKey *imagesafety.ScanKey
}

var _ Indexer = (*BlueskyProfile)(nil)

// NewBlueskyProfile builds an indexer backed by the given pool.
func NewBlueskyProfile(pool *pgxpool.Pool) *BlueskyProfile {
	return &BlueskyProfile{pool: pool}
}

func NewImageSafetyBlueskyProfile(pool *pgxpool.Pool, key imagesafety.ScanKey) *BlueskyProfile {
	return &BlueskyProfile{pool: pool, imageScanKey: &key}
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
	Pronouns    *string         `json:"pronouns,omitempty"`
	Avatar      *blueskyBlobRef `json:"avatar,omitempty"`
	Banner      *blueskyBlobRef `json:"banner,omitempty"`
}

func (b *BlueskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyProfileNSID {
		return nil
	}
	if b.imageScanKey != nil {
		return b.handleWithImageSafety(ctx, ev)
	}

	switch ev.Action {
	case "create", "update":
		var rec blueskyProfileRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		const q = `
			INSERT INTO bluesky_profiles
				(did, display_name, description, pronouns,
				 avatar_cid, avatar_mime, banner_cid, banner_mime, record_cid)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (did) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description  = EXCLUDED.description,
				pronouns     = EXCLUDED.pronouns,
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
		if _, err := b.database().Exec(ctx, q,
			ev.DID, rec.DisplayName, rec.Description, rec.Pronouns,
			avatarCID, avatarMime, bannerCID, bannerMime, ev.CID); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.database().Exec(ctx,
			`DELETE FROM bluesky_profiles WHERE did = $1`, ev.DID); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (b *BlueskyProfile) handleWithImageSafety(ctx context.Context, ev tap.Event) error {
	if b.imageScanKey.ScannerID == "" || b.imageScanKey.PolicyVersion == "" || b.imageScanKey.CorpusVersion == "" {
		return fmt.Errorf("image safety scan identity is incomplete")
	}
	tx, err := b.database().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin profile image reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	switch ev.Action {
	case "create", "update":
		var record bsky.ActorProfile
		if err := json.Unmarshal(ev.Record, &record); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bluesky_profiles(did,display_name,description,pronouns,record_cid)
			VALUES($1,$2,$3,$4,$5)
			ON CONFLICT(did) DO UPDATE SET
				display_name=EXCLUDED.display_name,
				description=EXCLUDED.description,
				pronouns=EXCLUDED.pronouns,
				record_cid=EXCLUDED.record_cid,
				indexed_at=now()
			WHERE bluesky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
		`, ev.DID, record.DisplayName, record.Description, record.Pronouns, ev.CID); err != nil {
			return fmt.Errorf("upsert safe profile fields %s: %w", ev.URI, err)
		}
		if err := b.reconcileProfileImage(ctx, tx, ev, "avatar", record.Avatar); err != nil {
			return err
		}
		if err := b.reconcileProfileImage(ctx, tx, ev, "banner", record.Banner); err != nil {
			return err
		}
	case "delete":
		if _, err := tx.Exec(ctx, `DELETE FROM profile_image_candidates WHERE profile_did=$1`, ev.DID); err != nil {
			return fmt.Errorf("delete profile image candidates %s: %w", ev.DID, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM bluesky_profiles WHERE did=$1`, ev.DID); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit profile image reconciliation: %w", err)
	}
	return nil
}

func (b *BlueskyProfile) reconcileProfileImage(
	ctx context.Context,
	tx pgx.Tx,
	event tap.Event,
	slot string,
	blob *util.LexBlob,
) error {
	if blob == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM profile_image_candidates WHERE profile_did=$1 AND slot=$2`, event.DID, slot); err != nil {
			return fmt.Errorf("delete %s candidate for %s: %w", slot, event.DID, err)
		}
		query := `UPDATE bluesky_profiles SET avatar_cid=NULL,avatar_mime=NULL WHERE did=$1`
		if slot == "banner" {
			query = `UPDATE bluesky_profiles SET banner_cid=NULL,banner_mime=NULL WHERE did=$1`
		}
		if _, err := tx.Exec(ctx, query, event.DID); err != nil {
			return fmt.Errorf("clear %s serving image for %s: %w", slot, event.DID, err)
		}
		return nil
	}

	image := &profileImage{CID: syntax.CID(blob.Ref.String()), MIMEType: blob.MimeType}
	if image.CID == "" || image.MIMEType == "" || blob.Size < 0 {
		return fmt.Errorf("invalid %s image on %s", slot, event.URI)
	}
	resultID, state, err := reconcileImageRequirement(
		ctx, tx, *b.imageScanKey, event, slot, image.CID, image.MIMEType, blob.Size,
	)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profile_image_candidates(
			profile_did,slot,source_cid,candidate_blob_cid,declared_mime,scan_result_id
		) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(profile_did,slot) DO UPDATE SET
			source_cid=EXCLUDED.source_cid,
			candidate_blob_cid=EXCLUDED.candidate_blob_cid,
			declared_mime=EXCLUDED.declared_mime,
			scan_result_id=EXCLUDED.scan_result_id,
			updated_at=now()
	`, event.DID, slot, event.CID, image.CID, image.MIMEType, resultID); err != nil {
		return fmt.Errorf("upsert %s candidate for %s: %w", slot, event.DID, err)
	}

	var previousCID, previousMIME *string
	columns := "avatar_cid,avatar_mime"
	if slot == "banner" {
		columns = "banner_cid,banner_mime"
	}
	if err := tx.QueryRow(ctx, `SELECT `+columns+` FROM bluesky_profiles WHERE did=$1`, event.DID).Scan(&previousCID, &previousMIME); err != nil {
		return fmt.Errorf("read prior %s for %s: %w", slot, event.DID, err)
	}
	var previous *profileImage
	if previousCID != nil && previousMIME != nil {
		previous = &profileImage{CID: syntax.CID(*previousCID), MIMEType: *previousMIME}
	}
	selected := selectProfileImage(image, state, previous)
	var selectedCID, selectedMIME any
	if selected != nil {
		selectedCID = selected.CID
		selectedMIME = selected.MIMEType
	}
	query := `UPDATE bluesky_profiles SET avatar_cid=$2,avatar_mime=$3 WHERE did=$1`
	if slot == "banner" {
		query = `UPDATE bluesky_profiles SET banner_cid=$2,banner_mime=$3 WHERE did=$1`
	}
	if _, err := tx.Exec(ctx, query, event.DID, selectedCID, selectedMIME); err != nil {
		return fmt.Errorf("select %s serving image for %s: %w", slot, event.DID, err)
	}
	return nil
}

func (b *BlueskyProfile) database() transactionalDatabase {
	if b.projectionDB != nil {
		return b.projectionDB
	}
	return b.pool
}
