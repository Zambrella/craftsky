package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/postrecord"
)

type publicationPreparationStore interface {
	publicationSnapshot(context.Context, PublishingClaim) (publicationSnapshot, error)
	SaveFrozenRecord(context.Context, FrozenRecordParams) error
}

type publicationEffectStore interface {
	AcquirePublishingEffect(context.Context, PublishingClaim) (*PublishingEffectGuard, error)
}

type publicationStateStore interface {
	FinalizePublication(context.Context, FinalizePublicationParams) (FinalizePublicationResult, error)
	failPublication(context.Context, PublishingClaim, FailureDecision, time.Time) (Status, error)
}

type publicationProcessorStore interface {
	publicationPreparationStore
	publicationEffectStore
	publicationStateStore
}

type PublicationProcessorOptions struct {
	Store         publicationProcessorStore
	Sessions      PublicationSessionSelector
	NewEffects    pdseffects.GuardedExecutorFactory
	Objects       PrivateObjectStore
	Now           func() time.Time
	Validate      func(context.Context, syntax.DID, Payload) error
	MaxMediaBytes int64
	Observer      OperationalObserver
}

type PublicationProcessor struct {
	store         publicationProcessorStore
	sessions      PublicationSessionSelector
	newEffects    pdseffects.GuardedExecutorFactory
	objects       PrivateObjectStore
	now           func() time.Time
	validate      func(context.Context, syntax.DID, Payload) error
	maxMediaBytes int64
	observer      OperationalObserver
}

func NewPublicationProcessor(options PublicationProcessorOptions) (*PublicationProcessor, error) {
	if options.Store == nil || options.Sessions == nil || options.NewEffects == nil || options.Objects == nil {
		return nil, errors.New("scheduled publication processor dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Validate == nil {
		options.Validate = func(context.Context, syntax.DID, Payload) error { return nil }
	}
	if options.MaxMediaBytes == 0 {
		options.MaxMediaBytes = 2_000_000
	}
	if options.MaxMediaBytes < 1 {
		return nil, errors.New("scheduled publication media limit is invalid")
	}
	return &PublicationProcessor{store: options.Store, sessions: options.Sessions,
		newEffects: options.NewEffects, objects: options.Objects, now: options.Now,
		validate: options.Validate, maxMediaBytes: options.MaxMediaBytes,
		observer: options.Observer}, nil
}

func (p *PublicationProcessor) Process(ctx context.Context, item WorkItem) (processErr error) {
	started := time.Now()
	attempt := 0
	var startLatency time.Duration
	defer func() {
		if p.observer != nil && attempt > 0 {
			p.observer.ObserveScheduledPublication(
				attempt,
				startLatency,
				time.Since(started),
			)
		}
	}()
	claim := workItemClaim(item)
	if claim.OwnerGeneration <= 0 {
		return errors.New("scheduled publication generation is unavailable")
	}

	snapshot, err := p.store.publicationSnapshot(ctx, claim)
	if errors.Is(err, ErrWorkerLeaseLost) || errors.Is(err, ErrScheduleNotFound) {
		if p.observer != nil {
			p.observer.ObserveScheduledOperation(
				"stale_worker", "stale", "stale_worker", time.Since(started),
			)
		}
		return nil
	}
	if err != nil {
		return err
	}
	attempt = snapshot.AttemptCount
	startLatency = started.UTC().Sub(snapshot.ScheduledAt)
	if startLatency < 0 {
		startLatency = 0
	}
	payload, err := DecodePayload(snapshot.Payload)
	if err != nil {
		return p.recordFailure(ctx, claim, ErrPolicyInvalid, item.Manual)
	}
	mediaReferences, err := decodeScheduledMediaReferences(snapshot.Payload)
	if err != nil || len(snapshot.Media) != len(mediaReferences) {
		return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
	}
	for index, media := range snapshot.Media {
		if media.ID != mediaReferences[index].id {
			return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
		}
		if media.SizeBytes < 1 || media.SizeBytes > p.maxMediaBytes {
			return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
		}
		if err := validateScheduledMediaReference(
			mediaReferences[index], media.MIMEType, media.SizeBytes,
		); err != nil {
			return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
		}
	}
	if err := p.validate(ctx, claim.OwnerDID, payload); err != nil {
		return p.recordFailure(ctx, claim, ErrPolicyInvalid, item.Manual)
	}

	recordBytes := snapshot.FrozenRecord
	if len(recordBytes) == 0 {
		blobs, err := predictedPublicationBlobs(snapshot.Media)
		if err != nil {
			return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
		}
		record, err := publicationRecord(payload, blobs, claim.CreatedAt)
		if err != nil {
			return p.recordFailure(ctx, claim, ErrPolicyInvalid, item.Manual)
		}
		body, err := json.Marshal(record)
		if err != nil {
			return p.recordFailure(ctx, claim, ErrPolicyInvalid, item.Manual)
		}
		tid, err := syntax.ParseTID(claim.Rkey.String())
		if err != nil {
			return p.recordFailure(ctx, claim, ErrRecordConflict, item.Manual)
		}
		frozen, err := FreezePublication(nil, PublicationFreezeRequest{
			Owner: claim.OwnerDID, TID: tid, CreatedAt: claim.CreatedAt, Body: body,
		})
		if err != nil || frozen.Rkey != claim.Rkey {
			return p.recordFailure(ctx, claim, ErrRecordConflict, item.Manual)
		}
		recordBytes = frozen.RecordBytes
		if err := p.store.SaveFrozenRecord(ctx, FrozenRecordParams{
			ID: claim.ID, OwnerDID: claim.OwnerDID, OwnerGeneration: claim.OwnerGeneration,
			LeaseToken:     claim.LeaseToken,
			PayloadVersion: claim.PayloadVersion, RecordBytes: recordBytes,
			RecordHash: frozen.RecordHash, Now: p.now().UTC(),
		}); err != nil {
			return err
		}
	}

	sessionID, err := SelectPublicationSession(ctx, p.sessions, claim.OwnerDID)
	if err != nil {
		return p.recordFailure(ctx, claim, err, item.Manual)
	}
	coordinator, err := p.newEffects(ctx, claim.OwnerDID, sessionID)
	if err != nil || coordinator == nil {
		return p.recordFailure(ctx, claim, ErrAuthUnavailable, item.Manual)
	}
	expectedOwners := []ownerlifecycle.ExpectedOwner{{
		Owner: claim.OwnerDID, Generation: claim.OwnerGeneration,
	}}
	finalized := false
	effectErr := coordinator.WithGuardedEffects(
		ctx,
		expectedOwners,
		func(effectCtx context.Context, effects pdseffects.EffectExecutor) (effectErr error) {
			guard, err := p.store.AcquirePublishingEffect(effectCtx, claim)
			if err != nil {
				return err
			}
			effectCtx = guard.bind(effectCtx)
			defer func() {
				effectErr = errors.Join(effectErr, guard.Release(effectCtx))
			}()
			if err := p.uploadPrivateMedia(
				effectCtx,
				effects,
				claim,
				expectedOwners,
				snapshot.Media,
			); err != nil {
				return p.handleEffectFailure(effectCtx, claim, scheduledBlobEffect, err, item.Manual)
			}
			var record map[string]any
			if err := json.Unmarshal(recordBytes, &record); err != nil {
				return p.recordFailure(effectCtx, claim, ErrRecordConflict, item.Manual)
			}
			effectID := scheduledRecordEffectIdentity(claim)
			result, err := effects.PutRecord(effectCtx, pdseffects.PutRecordRequest{
				OperationID:     effectID,
				MutationKey:     effectID,
				Owner:           claim.OwnerDID,
				OwnerGeneration: claim.OwnerGeneration,
				ExpectedOwners:  expectedOwners,
				Collection:      syntax.NSID(PostCollection),
				Rkey:            claim.Rkey,
				Record:          record,
			})
			if err != nil {
				return p.handleEffectFailure(
					effectCtx, claim, scheduledRecordEffect, err, item.Manual,
				)
			}
			_, err = p.store.FinalizePublication(effectCtx, FinalizePublicationParams{
				Claim: claim, PublicationURI: result.URI, PublicationCID: result.CID,
				PublishedAt: p.now().UTC(),
			})
			if err == nil {
				finalized = true
			}
			return err
		},
	)
	if errors.Is(effectErr, ownerlifecycle.ErrGenerationChanged) ||
		errors.Is(effectErr, ownerlifecycle.ErrOwnerNotActive) ||
		errors.Is(effectErr, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(effectErr, ErrWorkerLeaseLost) || errors.Is(effectErr, ErrScheduleNotFound) {
		if p.observer != nil {
			p.observer.ObserveScheduledOperation(
				"stale_worker", "stale", "stale_worker", time.Since(started),
			)
		}
		return nil
	}
	if item.Manual && errors.Is(effectErr, pdseffects.ErrOutcomeAmbiguous) {
		return errors.Join(ErrPublicationAmbiguous, effectErr)
	}
	if finalized && effectErr != nil {
		return fmt.Errorf("release finalized scheduled publication effect: %w", effectErr)
	}
	if effectErr == nil && p.observer != nil {
		p.observer.ObserveScheduledOperation(
			"publish", "success", "none", time.Since(started),
		)
	}
	return effectErr
}

func (p *PublicationProcessor) recordFailure(
	ctx context.Context,
	claim PublishingClaim,
	cause error,
	manual bool,
) error {
	decision := ClassifyPublicationFailure(cause)
	if manual {
		decision.Disposition = FailureNeedsAttention
	}
	started := time.Now()
	status, err := p.store.failPublication(ctx, claim, decision, p.now().UTC())
	if err != nil {
		return err
	}
	if p.observer != nil {
		operation := "retry"
		if status == StatusNeedsAttention {
			operation = "needs_attention"
		}
		p.observer.ObserveScheduledOperation(
			operation,
			"failure",
			decision.SafeCode,
			time.Since(started),
		)
	}
	return nil
}

func (p *PublicationProcessor) uploadPrivateMedia(
	ctx context.Context,
	effects pdseffects.EffectExecutor,
	claim PublishingClaim,
	expectedOwners []ownerlifecycle.ExpectedOwner,
	media []publicationMedia,
) error {
	for ordinal, item := range media {
		body, err := p.objects.Open(ctx, item.ObjectKey)
		if err != nil {
			return ErrObjectUnavailable
		}
		bytesValue, readErr := io.ReadAll(io.LimitReader(body, item.SizeBytes+1))
		closeErr := body.Close()
		if readErr != nil || closeErr != nil || int64(len(bytesValue)) != item.SizeBytes || sha256.Sum256(bytesValue) != item.SHA256 {
			return ErrMediaInvalid
		}
		effectID := scheduledBlobEffectIdentity(claim, ordinal)
		uploaded, err := effects.UploadBlob(ctx, pdseffects.UploadBlobRequest{
			OperationID:     effectID,
			MutationKey:     effectID,
			Owner:           claim.OwnerDID,
			OwnerGeneration: claim.OwnerGeneration,
			ExpectedOwners:  expectedOwners,
			MIME:            item.MIMEType,
			Bytes:           bytesValue,
		})
		if err != nil {
			return err
		}
		if uploaded == nil || len(uploaded.Raw) == 0 {
			return errors.Join(ErrMediaInvalid, errors.New("durable blob effect returned no result"))
		}
		expected := publicationBlob(item)
		expectedJSON, expectedErr := json.Marshal(expected)
		actualJSON, actualErr := json.Marshal(uploaded.Raw)
		if expectedErr != nil || actualErr != nil ||
			uploaded.CID != item.BlobCID.String() ||
			uploaded.MIME != item.MIMEType ||
			uploaded.Size != item.SizeBytes ||
			!bytes.Equal(actualJSON, expectedJSON) {
			return ErrMediaInvalid
		}
	}
	return nil
}

type scheduledEffectKind uint8

const (
	scheduledRecordEffect scheduledEffectKind = iota + 1
	scheduledBlobEffect
)

func scheduledEffectIdentityBase(claim PublishingClaim) string {
	return fmt.Sprintf(
		"scheduled-post/%s/g%d/v%d",
		claim.ID,
		claim.OwnerGeneration,
		claim.PayloadVersion,
	)
}

func scheduledRecordEffectIdentity(claim PublishingClaim) string {
	return scheduledEffectIdentityBase(claim) + "/record"
}

func scheduledBlobEffectIdentity(claim PublishingClaim, ordinal int) string {
	return fmt.Sprintf("%s/blob/%d", scheduledEffectIdentityBase(claim), ordinal)
}

func classifyScheduledEffectError(kind scheduledEffectKind, err error) error {
	if errors.Is(err, pdseffects.ErrEffectConflict) ||
		errors.Is(err, pdseffects.ErrEffectRejected) {
		if kind == scheduledBlobEffect {
			return errors.Join(ErrMediaInvalid, err)
		}
		return errors.Join(ErrRecordConflict, err)
	}
	return errors.Join(ErrPDSUnavailable, err)
}

func (p *PublicationProcessor) handleEffectFailure(
	ctx context.Context,
	claim PublishingClaim,
	kind scheduledEffectKind,
	err error,
	manual bool,
) error {
	if errors.Is(err, ownerlifecycle.ErrGenerationChanged) ||
		errors.Is(err, ownerlifecycle.ErrOwnerNotActive) ||
		errors.Is(err, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(err, ErrWorkerLeaseLost) ||
		errors.Is(err, ErrScheduleNotFound) {
		return err
	}
	if manual && errors.Is(err, pdseffects.ErrOutcomeAmbiguous) {
		return errors.Join(ErrPublicationAmbiguous, err)
	}
	return p.recordFailure(ctx, claim, classifyScheduledEffectError(kind, err), manual)
}

func predictedPublicationBlobs(media []publicationMedia) ([]map[string]any, error) {
	blobs := make([]map[string]any, 0, len(media))
	for _, item := range media {
		if item.BlobCID == "" || item.MIMEType == "" || item.SizeBytes < 1 {
			return nil, ErrMediaInvalid
		}
		blobs = append(blobs, publicationBlob(item))
	}
	return blobs, nil
}

func publicationBlob(item publicationMedia) map[string]any {
	return map[string]any{
		"$type":    "blob",
		"ref":      map[string]any{"$link": item.BlobCID.String()},
		"mimeType": item.MIMEType,
		"size":     item.SizeBytes,
	}
}

func publicationRecord(payload Payload, blobs []map[string]any, createdAt time.Time) (map[string]any, error) {
	expectedBlobs := len(payload.Media)
	if payload.External != nil && payload.External.ThumbMediaID != "" {
		expectedBlobs++
	}
	if len(blobs) != expectedBlobs {
		return nil, ErrMediaInvalid
	}
	record := map[string]any{"$type": PostCollection, "text": payload.Text, "createdAt": createdAt.UTC().Format(time.RFC3339)}
	if len(payload.Facets) > 0 {
		var facets any
		if err := json.Unmarshal(payload.Facets, &facets); err != nil {
			return nil, err
		}
		record["facets"] = facets
	}
	if len(payload.Langs) > 0 {
		record["langs"] = payload.Langs
	}
	if len(payload.Project) > 0 {
		var project any
		if err := json.Unmarshal(payload.Project, &project); err != nil {
			return nil, err
		}
		record["project"] = project
	}
	if len(blobs) > 0 {
		images := make([]map[string]any, 0, len(payload.Media))
		for index, blob := range blobs[:len(payload.Media)] {
			image := map[string]any{"image": blob}
			if payload.Media[index].Alt != "" {
				image["alt"] = payload.Media[index].Alt
			}
			if payload.Media[index].Width > 0 && payload.Media[index].Height > 0 {
				image["aspectRatio"] = map[string]any{"width": payload.Media[index].Width, "height": payload.Media[index].Height}
			}
			images = append(images, image)
		}
		if len(images) > 0 {
			record["images"] = images
		}
	}
	if payload.External != nil {
		var thumb map[string]any
		if payload.External.ThumbMediaID != "" {
			thumb = blobs[len(payload.Media)]
		}
		embed, err := postrecord.ExternalEmbed(
			payload.External.URI,
			payload.External.Title,
			payload.External.Description,
			thumb,
		)
		if err != nil {
			return nil, err
		}
		record["embed"] = embed
	}
	return record, nil
}
