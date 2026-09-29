package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/imagesafety"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

type imageRequirement struct {
	slot string
	blob *util.LexBlob
}

type imageSafetySubject struct {
	next    TransactionalIndexer
	key     imagesafety.ScanKey
	kind    string
	extract func(json.RawMessage) ([]imageRequirement, error)
}

func NewImageSafetyCraftskyBusinessEvent(next TransactionalIndexer, key imagesafety.ScanKey) TransactionalIndexer {
	return &imageSafetySubject{next: next, key: key, kind: "business_event", extract: extractBusinessEventImages}
}

func NewImageSafetyCraftskyBusinessProfile(next TransactionalIndexer, key imagesafety.ScanKey) TransactionalIndexer {
	return &imageSafetySubject{next: next, key: key, kind: "business_profile", extract: extractBusinessProfileImages}
}

func (projector *imageSafetySubject) Project(ctx context.Context, tx pgx.Tx, event tap.Event) (tap.Outcome, error) {
	if tx == nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("image safety projection requires a transaction")
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
	requirements, err := projector.extract(event.Record)
	if err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM image_subject_requirements WHERE subject_uri=$1`, event.URI); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("delete prior image requirements %s: %w", event.URI, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)
		VALUES($1,$2,$3,'blocked')
		ON CONFLICT(subject_uri) DO UPDATE SET
			subject_kind=EXCLUDED.subject_kind,
			source_cid=EXCLUDED.source_cid,
			visibility_state='blocked',
			updated_at=now()
	`, event.URI, projector.kind, event.CID); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("block image subject %s: %w", event.URI, err)
	}

	states := make([]imagesafety.State, 0, len(requirements))
	for _, requirement := range requirements {
		if requirement.blob == nil {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		blobCID := syntax.CID(requirement.blob.Ref.String())
		if blobCID == "" || requirement.blob.MimeType == "" || requirement.blob.Size < 0 {
			return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
		}
		resultID, state, err := reconcileImageRequirement(
			ctx, tx, projector.key, event, requirement.slot, blobCID,
			requirement.blob.MimeType, requirement.blob.Size,
		)
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO image_subject_requirements(
				subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id
			) VALUES($1,$2,$3,$4,$5,$6)
		`, event.URI, event.CID, projector.kind, requirement.slot, blobCID, resultID); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("insert image requirement %s slot %s: %w", event.URI, requirement.slot, err)
		}
		states = append(states, state)
	}
	if !imagesafety.ParentDisplayEligible(states) {
		return tap.Blocked(tap.ReasonImageScanPending, tap.Dependency{Kind: "image_subject_uri", Key: event.URI.String()}), nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE image_subject_states SET visibility_state='clear',updated_at=now()
		WHERE subject_uri=$1 AND source_cid=$2
	`, event.URI, event.CID); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("clear image subject %s: %w", event.URI, err)
	}
	return projector.next.Project(ctx, tx, event)
}

func extractBusinessEventImages(raw json.RawMessage) ([]imageRequirement, error) {
	var record craftskylex.BusinessEvent
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	if record.Image == nil {
		return nil, nil
	}
	if record.Image.Image == nil {
		return nil, fmt.Errorf("event image blob is missing")
	}
	return []imageRequirement{{slot: "image", blob: record.Image.Image}}, nil
}

func extractBusinessProfileImages(raw json.RawMessage) ([]imageRequirement, error) {
	var record craftskylex.BusinessProfile
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	requirements := make([]imageRequirement, 0, len(record.Products))
	for position, product := range record.Products {
		if product == nil || product.Image == nil {
			continue
		}
		if product.Image.Image == nil {
			return nil, fmt.Errorf("product image blob is missing")
		}
		requirements = append(requirements, imageRequirement{
			slot: fmt.Sprintf("products.%d.image", position),
			blob: product.Image.Image,
		})
	}
	return requirements, nil
}
