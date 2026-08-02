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

	"social.craftsky/appview/internal/auth"
)

type PublicationProcessorOptions struct {
	Store         *Store
	Sessions      PublicationSessionSelector
	NewPDS        auth.PDSClientFactory
	Objects       PrivateObjectStore
	Now           func() time.Time
	Validate      func(context.Context, syntax.DID, Payload) error
	MaxMediaBytes int64
	Observer      OperationalObserver
}

type PublicationProcessor struct {
	store         *Store
	sessions      PublicationSessionSelector
	newPDS        auth.PDSClientFactory
	objects       PrivateObjectStore
	now           func() time.Time
	validate      func(context.Context, syntax.DID, Payload) error
	maxMediaBytes int64
	observer      OperationalObserver
}

func NewPublicationProcessor(options PublicationProcessorOptions) (*PublicationProcessor, error) {
	if options.Store == nil || options.Sessions == nil || options.NewPDS == nil || options.Objects == nil {
		return nil, errors.New("scheduled publication processor dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Validate == nil {
		options.Validate = func(context.Context, syntax.DID, Payload) error { return nil }
	}
	if options.MaxMediaBytes == 0 {
		options.MaxMediaBytes = 15 * 1024 * 1024
	}
	if options.MaxMediaBytes < 1 {
		return nil, errors.New("scheduled publication media limit is invalid")
	}
	return &PublicationProcessor{store: options.Store, sessions: options.Sessions,
		newPDS: options.NewPDS, objects: options.Objects, now: options.Now,
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
	guard, err := p.store.AcquirePublishingEffect(ctx, claim)
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
	defer func() {
		if releaseErr := guard.Release(context.Background()); processErr == nil && releaseErr != nil {
			processErr = releaseErr
		}
	}()

	snapshot, err := p.store.publicationSnapshot(ctx, claim)
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
	if len(snapshot.Media) != len(payload.Media) {
		return p.recordFailure(ctx, claim, ErrMediaInvalid, item.Manual)
	}
	for _, media := range snapshot.Media {
		if media.SizeBytes < 1 || media.SizeBytes > p.maxMediaBytes {
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
			ID: claim.ID, OwnerDID: claim.OwnerDID, LeaseToken: claim.LeaseToken,
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
	pds, err := p.newPDS(ctx, claim.OwnerDID, sessionID)
	if err != nil {
		return p.recordFailure(ctx, claim, ErrAuthUnavailable, item.Manual)
	}
	if err := p.uploadPrivateMedia(ctx, pds, snapshot.Media); err != nil {
		return p.recordFailure(ctx, claim, err, item.Manual)
	}
	var record map[string]any
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return p.recordFailure(ctx, claim, ErrRecordConflict, item.Manual)
	}

	cid, found, err := matchingPDSRecord(ctx, pds, claim, recordBytes)
	if err != nil {
		return p.recordFailure(ctx, claim, err, item.Manual)
	}
	if !found {
		if err := pds.PutRecord(ctx, claim.OwnerDID, PostCollection, claim.Rkey.String(), record); err != nil {
			if item.Manual {
				return ErrPublicationAmbiguous
			}
			return p.recordFailure(ctx, claim, ErrPDSUnavailable, false)
		}
		cid, found, err = matchingPDSRecord(ctx, pds, claim, recordBytes)
		if err != nil {
			if item.Manual {
				return ErrPublicationAmbiguous
			}
			return p.recordFailure(ctx, claim, err, false)
		}
		if !found {
			if item.Manual {
				return ErrPublicationAmbiguous
			}
			return p.recordFailure(ctx, claim, ErrPDSUnavailable, false)
		}
	}
	uri := syntax.ATURI(fmt.Sprintf("at://%s/%s/%s", claim.OwnerDID, PostCollection, claim.Rkey))
	_, err = p.store.FinalizePublication(ctx, FinalizePublicationParams{
		Claim: claim, PublicationURI: uri, PublicationCID: syntax.CID(cid), PublishedAt: p.now().UTC(),
	})
	if err == nil && p.observer != nil {
		p.observer.ObserveScheduledOperation(
			"publish", "success", "none", time.Since(started),
		)
	}
	return err
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

func (p *PublicationProcessor) uploadPrivateMedia(ctx context.Context, pds auth.PDSClient, media []publicationMedia) error {
	for _, item := range media {
		body, err := p.objects.Open(ctx, item.ObjectKey)
		if err != nil {
			return ErrObjectUnavailable
		}
		bytesValue, readErr := io.ReadAll(io.LimitReader(body, item.SizeBytes+1))
		closeErr := body.Close()
		if readErr != nil || closeErr != nil || int64(len(bytesValue)) != item.SizeBytes || sha256.Sum256(bytesValue) != item.SHA256 {
			return ErrMediaInvalid
		}
		uploaded, err := pds.UploadBlob(ctx, item.MIMEType, bytesValue)
		if err != nil || uploaded == nil || len(uploaded.Raw) == 0 {
			return ErrPDSUnavailable
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
	if len(blobs) != len(payload.Media) {
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
		images := make([]map[string]any, 0, len(blobs))
		for index, blob := range blobs {
			image := map[string]any{"image": blob}
			if payload.Media[index].Alt != "" {
				image["alt"] = payload.Media[index].Alt
			}
			if payload.Media[index].Width > 0 && payload.Media[index].Height > 0 {
				image["aspectRatio"] = map[string]any{"width": payload.Media[index].Width, "height": payload.Media[index].Height}
			}
			images = append(images, image)
		}
		record["images"] = images
	}
	return record, nil
}

func matchingPDSRecord(ctx context.Context, pds auth.PDSClient, claim PublishingClaim, expected []byte) (string, bool, error) {
	var existing map[string]any
	cid, err := pds.GetRecord(ctx, claim.OwnerDID, PostCollection, claim.Rkey.String(), &existing)
	if errors.Is(err, auth.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, ErrPDSUnavailable
	}
	actual, err := json.Marshal(existing)
	if err != nil {
		return "", false, ErrRecordConflict
	}
	if !bytes.Equal(actual, expected) {
		return "", false, ErrRecordConflict
	}
	return cid, true, nil
}
