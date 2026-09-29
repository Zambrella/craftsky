package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/imagesafety"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

type imageSafetyCraftskyPost struct {
	next TransactionalIndexer
	key  imagesafety.ScanKey
}

func NewImageSafetyCraftskyPost(next TransactionalIndexer, key imagesafety.ScanKey) TransactionalIndexer {
	return &imageSafetyCraftskyPost{next: next, key: key}
}

func (projector *imageSafetyCraftskyPost) Project(
	ctx context.Context,
	tx pgx.Tx,
	event tap.Event,
) (tap.Outcome, error) {
	if tx == nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("image safety post projection requires a transaction")
	}
	if event.Action == "delete" {
		if _, err := tx.Exec(ctx, `DELETE FROM image_subject_states WHERE subject_uri=$1`, event.URI); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("delete image subject state %s: %w", event.URI, err)
		}
		return projector.next.Project(ctx, tx, event)
	}
	if projector.key.ScannerID == "" || projector.key.PolicyVersion == "" || projector.key.CorpusVersion == "" {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("image safety scan identity is incomplete")
	}

	var record craftskylex.FeedPost
	if err := json.Unmarshal(event.Record, &record); err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}

	if _, err := tx.Exec(ctx, `DELETE FROM image_subject_requirements WHERE subject_uri=$1`, event.URI); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("delete prior image requirements %s: %w", event.URI, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
		VALUES($1,'post',$2,'blocked')
		ON CONFLICT(subject_uri) DO UPDATE SET
			subject_kind=EXCLUDED.subject_kind,
			source_cid=EXCLUDED.source_cid,
			visibility_state='blocked',
			updated_at=now()
	`, event.URI, event.CID); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("block image subject %s: %w", event.URI, err)
	}

	requirements := make([]imageRequirement, 0, len(record.Images)+1)
	for position, image := range record.Images {
		if image == nil || image.Image == nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		requirements = append(requirements, imageRequirement{slot: fmt.Sprintf("images.%d", position), blob: image.Image})
	}
	if record.Embed != nil && record.Embed.EmbedExternal != nil {
		if record.Embed.EmbedExternal.External == nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		if record.Embed.EmbedExternal.External.Thumb != nil {
			requirements = append(requirements, imageRequirement{slot: "embed.external.thumb", blob: record.Embed.EmbedExternal.External.Thumb})
		}
	}

	states := make([]imagesafety.State, 0, len(requirements))
	for _, requirement := range requirements {
		blobCID := syntax.CID(requirement.blob.Ref.String())
		if blobCID == "" || requirement.blob.MimeType == "" || requirement.blob.Size < 0 {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		resultID, state, err := reconcileImageRequirement(ctx, tx, projector.key, event, requirement.slot, blobCID, requirement.blob.MimeType, requirement.blob.Size)
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO image_subject_requirements(
				subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id
			) VALUES($1,$2,'post',$3,$4,$5)
		`, event.URI, event.CID, requirement.slot, blobCID, resultID); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("insert image requirement %s slot %s: %w", event.URI, requirement.slot, err)
		}
		states = append(states, state)
	}

	if !imagesafety.ParentDisplayEligible(states) {
		return tap.Blocked(tap.ReasonImageScanPending, tap.Dependency{
			Kind: "image_subject_uri",
			Key:  event.URI.String(),
		}), nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE image_subject_states
		SET visibility_state='clear', updated_at=now()
		WHERE subject_uri=$1 AND source_cid=$2
	`, event.URI, event.CID); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("clear image subject %s: %w", event.URI, err)
	}
	return projector.next.Project(ctx, tx, event)
}

func reconcileImageRequirement(
	ctx context.Context,
	tx pgx.Tx,
	baseKey imagesafety.ScanKey,
	event tap.Event,
	slot string,
	blobCID syntax.CID,
	mimeType string,
	size int64,
) (uuid.UUID, imagesafety.State, error) {
	resultID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state
		) VALUES($1,$2,$3,$4,$5,'pending')
		ON CONFLICT(blob_cid,scanner_id,policy_version,corpus_version) DO NOTHING
	`, resultID, blobCID, baseKey.ScannerID, baseKey.PolicyVersion, baseKey.CorpusVersion); err != nil {
		return uuid.Nil, "", fmt.Errorf("ensure scan result for %s: %w", blobCID, err)
	}

	var state imagesafety.State
	if err := tx.QueryRow(ctx, `
		SELECT id,state
		FROM image_scan_results
		WHERE blob_cid=$1 AND scanner_id=$2 AND policy_version=$3 AND corpus_version=$4
	`, blobCID, baseKey.ScannerID, baseKey.PolicyVersion, baseKey.CorpusVersion).Scan(&resultID, &state); err != nil {
		return uuid.Nil, "", fmt.Errorf("read scan result for %s: %w", blobCID, err)
	}
	if !state.Valid() {
		return uuid.Nil, "", fmt.Errorf("invalid scan state for %s", blobCID)
	}
	if !state.Terminal() {
		if _, err := tx.Exec(ctx, `
			INSERT INTO image_scan_jobs(id,scan_result_id)
			VALUES($1,$2)
			ON CONFLICT(scan_result_id) DO NOTHING
		`, uuid.New(), resultID); err != nil {
			return uuid.Nil, "", fmt.Errorf("ensure scan job for %s: %w", blobCID, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO image_blob_sources(
			blob_cid,source_did,source_uri,source_cid,declared_mime,declared_size
		) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(blob_cid,source_uri,source_cid) DO UPDATE SET
			source_did=EXCLUDED.source_did,
			declared_mime=EXCLUDED.declared_mime,
			declared_size=EXCLUDED.declared_size,
			observed_at=now()
	`, blobCID, event.DID, event.URI, event.CID, mimeType, size); err != nil {
		return uuid.Nil, "", fmt.Errorf("record image source %s slot %s: %w", event.URI, slot, err)
	}
	return resultID, state, nil
}
