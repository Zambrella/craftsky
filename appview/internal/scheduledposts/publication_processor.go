package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
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

type GuardedCommandOperation func(
	context.Context,
	pdseffects.EffectExecutor,
	pdscommands.AlreadyFencedCommandExecutor,
) error

type GuardedCommandCoordinator interface {
	WithGuardedCommands(
		context.Context,
		[]ownerlifecycle.ExpectedOwner,
		GuardedCommandOperation,
	) error
}

type GuardedCommandCoordinatorFactory func(
	context.Context,
	syntax.DID,
	string,
) (GuardedCommandCoordinator, error)

type PublicationProcessorOptions struct {
	CheckPlusAccess func(context.Context, syntax.DID) (bool, error)
	WithPlusAccess  func(context.Context, syntax.DID, func(context.Context) error) error
	Store           publicationProcessorStore
	Sessions        PublicationSessionSelector
	NewCommands     GuardedCommandCoordinatorFactory
	Objects         PrivateObjectStore
	Now             func() time.Time
	Validate        func(context.Context, syntax.DID, Payload) error
	MaxMediaBytes   int64
	Observer        OperationalObserver
}

type PublicationProcessor struct {
	checkPlusAccess func(context.Context, syntax.DID) (bool, error)
	withPlusAccess  func(context.Context, syntax.DID, func(context.Context) error) error
	store           publicationProcessorStore
	sessions        PublicationSessionSelector
	newCommands     GuardedCommandCoordinatorFactory
	objects         PrivateObjectStore
	now             func() time.Time
	validate        func(context.Context, syntax.DID, Payload) error
	maxMediaBytes   int64
	observer        OperationalObserver
}

func NewPublicationProcessor(options PublicationProcessorOptions) (*PublicationProcessor, error) {
	if options.Store == nil || options.Sessions == nil || options.NewCommands == nil || options.Objects == nil {
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
		newCommands: options.NewCommands, objects: options.Objects, now: options.Now,
		validate: options.Validate, maxMediaBytes: options.MaxMediaBytes,
		withPlusAccess:  options.WithPlusAccess,
		checkPlusAccess: options.CheckPlusAccess,
		observer:        options.Observer}, nil
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
	ctx = observability.WithPrivateDiagnosticContext(ctx, claim.OwnerDID, claim.ID.String())
	stage := "snapshot"
	defer func() {
		if processErr != nil {
			if diagnostic, ok := p.observer.(privateFailureObserver); ok {
				diagnostic.ObservePrivateFailure(ctx, processErr, claim.OwnerDID, claim.ID.String(), "schedule.publish", stage, "error", attempt)
			}
		}
	}()
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
	stage = "preparation"
	attempt = snapshot.AttemptCount
	ctx = context.WithValue(ctx, publicationAttemptKey{}, attempt)
	startLatency = started.UTC().Sub(snapshot.ScheduledAt)
	if startLatency < 0 {
		startLatency = 0
	}
	if p.checkPlusAccess != nil {
		paid, err := p.checkPlusAccess(ctx, claim.OwnerDID)
		if err != nil {
			return err
		}
		if !paid {
			return p.recordFailure(ctx, claim, ErrSubscriptionRequired, item.Manual)
		}
	}
	if !item.Manual && !AutomaticPublicationEligible(snapshot.ScheduledAt, p.now().UTC()) {
		return p.recordFailure(ctx, claim, ErrAutomaticCutoffExceeded, false)
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

	stage = "session"
	sessionID, err := SelectPublicationSession(ctx, p.sessions, claim.OwnerDID)
	if err != nil {
		return p.recordFailure(ctx, claim, err, item.Manual)
	}
	stage = "command_setup"
	coordinator, err := p.newCommands(ctx, claim.OwnerDID, sessionID)
	if err != nil || coordinator == nil {
		ctx = context.WithValue(ctx, publicationStageKey{}, stage)
		if err != nil {
			ctx = context.WithValue(ctx, publicationDiagnosticCauseKey{}, err)
		}
		return p.recordFailure(ctx, claim, ErrAuthUnavailable, item.Manual)
	}
	expectedOwners := []ownerlifecycle.ExpectedOwner{{
		Owner: claim.OwnerDID, Generation: claim.OwnerGeneration,
	}}
	finalized := false
	stage = "effect_acquire"
	effectErr := coordinator.WithGuardedCommands(
		ctx,
		expectedOwners,
		func(
			effectCtx context.Context,
			effects pdseffects.EffectExecutor,
			commands pdscommands.AlreadyFencedCommandExecutor,
		) (effectErr error) {
			guard, err := p.store.AcquirePublishingEffect(effectCtx, claim)
			if err != nil {
				return err
			}
			effectCtx = guard.bind(effectCtx)
			defer func() {
				releaseErr := guard.Release(effectCtx)
				if effectErr == nil && releaseErr != nil {
					stage = "effect_release"
				}
				effectErr = errors.Join(effectErr, releaseErr)
			}()
			publish := func(effectCtx context.Context) error {
				stage = "media_upload"
				effectCtx = context.WithValue(effectCtx, publicationStageKey{}, stage)
				if err := p.uploadPrivateMedia(effectCtx, effects, claim, expectedOwners, snapshot.Media); err != nil {
					return p.handleEffectFailure(effectCtx, claim, scheduledBlobEffect, err, item.Manual)
				}
				stage = "record_write"
				effectCtx = context.WithValue(effectCtx, publicationStageKey{}, stage)
				intent, err := scheduledPublicationCommandIntent(claim, recordBytes)
				if err != nil {
					return p.recordFailure(effectCtx, claim, ErrRecordConflict, item.Manual)
				}
				selectedURI := syntax.ATURI("at://" + claim.OwnerDID.String() + "/" + PostCollection + "/" + claim.Rkey.String())
				commandResult, err := commands.ExecuteAppend(
					effectCtx,
					pdscommands.FencedAppendCommandRequest{
						Command: pdscommands.AppendCommandRequest{
							Owner: claim.OwnerDID, OwnerGeneration: claim.OwnerGeneration,
							SessionID: sessionID, OperationKind: "scheduled_post_publish",
							OperationKey: scheduledPublicationCommandKey(claim),
							Collection:   syntax.NSID(PostCollection), Intent: intent,
							Blobs: scheduledPublicationBlobReferences(snapshot.Media),
							BuildRecord: func(time.Time) (json.RawMessage, error) {
								return append(json.RawMessage(nil), recordBytes...), nil
							},
							Accepted: scheduledPublicationAcceptedResult,
							Rejected: scheduledPublicationRejectedResult,
						},
						SelectedURI: selectedURI, SelectedRkey: claim.Rkey,
					},
				)
				if commandResult.DiagnosticCause != nil {
					effectCtx = context.WithValue(effectCtx, publicationDiagnosticCauseKey{}, commandResult.DiagnosticCause)
				}

				if err != nil {
					return p.handleScheduledCommandFailure(effectCtx, claim, err, item.Manual)
				}
				if commandResult.State == pdscommands.CommandAmbiguous {
					return p.handleScheduledCommandFailure(
						effectCtx, claim, pdseffects.ErrOutcomeAmbiguous, item.Manual,
					)
				}
				if commandResult.State == pdscommands.CommandRejected {
					return p.recordFailure(effectCtx, claim, ErrRecordConflict, item.Manual)
				}
				var result struct {
					URI syntax.ATURI `json:"uri"`
					CID syntax.CID   `json:"cid"`
				}
				if commandResult.State != pdscommands.CommandAccepted ||
					json.Unmarshal(commandResult.ResponseBody, &result) != nil ||
					result.URI == "" || result.CID == "" {
					return p.recordFailure(effectCtx, claim, ErrRecordConflict, item.Manual)
				}
				stage = "finalization"
				_, err = p.store.FinalizePublication(effectCtx, FinalizePublicationParams{
					Claim: claim, PublicationURI: result.URI, PublicationCID: result.CID,
					PublishedAt: p.now().UTC(),
				})
				if err == nil {
					finalized = true
				}
				return err
			}
			if p.withPlusAccess != nil {
				err = p.withPlusAccess(effectCtx, claim.OwnerDID, publish)
				if errors.Is(err, ErrSubscriptionRequired) {
					return p.recordFailure(effectCtx, claim, ErrSubscriptionRequired, item.Manual)
				}
				return err
			}
			return publish(effectCtx)
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

func scheduledPublicationCommandIntent(
	claim PublishingClaim,
	record json.RawMessage,
) (json.RawMessage, error) {
	digest := sha256.Sum256(record)
	return json.Marshal(struct {
		ScheduleID      uuid.UUID        `json:"scheduleId"`
		OwnerGeneration int64            `json:"ownerGeneration"`
		PayloadVersion  int64            `json:"payloadVersion"`
		Rkey            syntax.RecordKey `json:"rkey"`
		RecordSHA256    string           `json:"recordSha256"`
	}{
		ScheduleID: claim.ID, OwnerGeneration: claim.OwnerGeneration,
		PayloadVersion: claim.PayloadVersion, Rkey: claim.Rkey,
		RecordSHA256: hex.EncodeToString(digest[:]),
	})
}

func scheduledPublicationCommandKey(claim PublishingClaim) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(scheduledRecordEffectIdentity(claim)))
}

func scheduledPublicationBlobReferences(media []publicationMedia) []pdscommands.BlobReference {
	result := make([]pdscommands.BlobReference, len(media))
	for index, item := range media {
		result[index] = pdscommands.BlobReference{
			CID: item.BlobCID, MIMEType: item.MIMEType, Size: item.SizeBytes,
		}
	}
	return result
}

func scheduledPublicationAcceptedResult(
	record pdscommands.AuthoritativeRecord,
) (pdscommands.TerminalResult, error) {
	body, err := json.Marshal(struct {
		URI syntax.ATURI `json:"uri"`
		CID syntax.CID   `json:"cid"`
	}{URI: record.URI, CID: record.CID})
	if err != nil {
		return pdscommands.TerminalResult{}, err
	}
	return pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: 201, ResponseBody: body,
	}, nil
}

func scheduledPublicationRejectedResult(error) pdscommands.TerminalResult {
	return pdscommands.TerminalResult{
		State: pdscommands.CommandRejected, HTTPStatus: 409,
		ResponseBody: json.RawMessage(`{"error":"record_conflict"}`),
	}
}

func (p *PublicationProcessor) handleScheduledCommandFailure(
	ctx context.Context,
	claim PublishingClaim,
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
	if errors.Is(err, pdscommands.ErrIdempotencyConflict) ||
		errors.Is(err, pdscommands.ErrRecordConflict) ||
		errors.Is(err, pdscommands.ErrDispatchRejected) {
		return p.recordFailure(ctx, claim, errors.Join(ErrRecordConflict, err), manual)
	}
	return p.recordFailure(ctx, claim, classifyScheduledEffectError(scheduledRecordEffect, err), manual)
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
	if diagnostic, ok := p.observer.(privateFailureObserver); ok {
		outcome := "retry"
		if status == StatusNeedsAttention {
			outcome = "terminal"
		}
		attempt, _ := ctx.Value(publicationAttemptKey{}).(int)
		diagnosticCause, _ := ctx.Value(publicationDiagnosticCauseKey{}).(error)
		if diagnosticCause == nil {
			diagnosticCause = cause
		}
		stage, _ := ctx.Value(publicationStageKey{}).(string)
		if stage == "" {
			stage = "preparation"
		}
		diagnostic.ObservePrivateFailure(ctx, diagnosticCause, claim.OwnerDID, claim.ID.String(), "schedule.publish", stage, outcome, attempt)
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
			return errors.Join(ErrObjectUnavailable, err)
		}
		bytesValue, readErr := io.ReadAll(io.LimitReader(body, item.SizeBytes+1))
		closeErr := body.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(ErrMediaInvalid, readErr, closeErr)
		}
		if int64(len(bytesValue)) != item.SizeBytes || sha256.Sum256(bytesValue) != item.SHA256 {
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
	record := map[string]any{"$type": PostCollection, "text": payload.Text, "sponsored": payload.Sponsored, "createdAt": createdAt.UTC().Format(time.RFC3339)}
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
