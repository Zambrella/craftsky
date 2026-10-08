package observability

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type PDSOperation string

const (
	PDSOperationOAuthSessionResume    PDSOperation = "oauth.session_resume"
	PDSOperationProfilePutBsky        PDSOperation = "profile.put_bsky"
	PDSOperationProfilePutCraftsky    PDSOperation = "profile.put_craftsky"
	PDSOperationPostCreate            PDSOperation = "post.create"
	PDSOperationPostDelete            PDSOperation = "post.delete"
	PDSOperationBlobUpload            PDSOperation = "blob.upload"
	PDSOperationFollowCreate          PDSOperation = "follow.create"
	PDSOperationFollowDelete          PDSOperation = "follow.delete"
	PDSOperationLikeCreate            PDSOperation = "like.create"
	PDSOperationLikeDelete            PDSOperation = "like.delete"
	PDSOperationRepostCreate          PDSOperation = "repost.create"
	PDSOperationRepostDelete          PDSOperation = "repost.delete"
	PDSOperationBusinessProfileGet    PDSOperation = "business.profile.get"
	PDSOperationBusinessProfilePut    PDSOperation = "business.profile.put"
	PDSOperationBusinessProfileDelete PDSOperation = "business.profile.delete"
	PDSOperationBusinessEventGet      PDSOperation = "business.event.get"
	PDSOperationBusinessEventPut      PDSOperation = "business.event.put"
	PDSOperationBusinessEventDelete   PDSOperation = "business.event.delete"
	PDSOperationCommandHead           PDSOperation = "command.repository_head"
	PDSOperationCommandList           PDSOperation = "command.list_records"
	PDSOperationCommandApply          PDSOperation = "command.apply_writes"
	PDSOperationCommandFallback       PDSOperation = "command.single_fallback"
)

var knownPDSOperations = map[PDSOperation]struct{}{
	PDSOperationOAuthSessionResume:    {},
	PDSOperationProfilePutBsky:        {},
	PDSOperationProfilePutCraftsky:    {},
	PDSOperationPostCreate:            {},
	PDSOperationPostDelete:            {},
	PDSOperationBlobUpload:            {},
	PDSOperationFollowCreate:          {},
	PDSOperationFollowDelete:          {},
	PDSOperationLikeCreate:            {},
	PDSOperationLikeDelete:            {},
	PDSOperationRepostCreate:          {},
	PDSOperationRepostDelete:          {},
	PDSOperationBusinessProfileGet:    {},
	PDSOperationBusinessProfilePut:    {},
	PDSOperationBusinessProfileDelete: {},
	PDSOperationBusinessEventGet:      {},
	PDSOperationBusinessEventPut:      {},
	PDSOperationBusinessEventDelete:   {},
	PDSOperationCommandHead:           {},
	PDSOperationCommandList:           {},
	PDSOperationCommandApply:          {},
	PDSOperationCommandFallback:       {},
}

func KnownPDSOperation(op PDSOperation) bool {
	_, ok := knownPDSOperations[op]
	return ok
}

type PDSStage string

const (
	PDSStageSessionResume         PDSStage = "session_resume"
	PDSStageRequestBuild          PDSStage = "request_build"
	PDSStagePDSRequest            PDSStage = "pds_request"
	PDSStagePDSResponse           PDSStage = "pds_response"
	PDSStagePostWriteIndexingWait PDSStage = "post_write_indexing_wait"
	PDSStageUnexpected            PDSStage = "unexpected"
)

type PDSCategory string

const (
	PDSCategoryNone        PDSCategory = "none"
	PDSCategoryTimeout     PDSCategory = "timeout"
	PDSCategoryNetwork     PDSCategory = "network"
	PDSCategoryAuth        PDSCategory = "auth"
	PDSCategoryRateLimited PDSCategory = "rate_limited"
	PDSCategoryValidation  PDSCategory = "validation"
	PDSCategoryNotFound    PDSCategory = "not_found"
	PDSCategoryForbidden   PDSCategory = "forbidden"
	PDSCategoryServer      PDSCategory = "server"
	PDSCategoryUnexpected  PDSCategory = "unexpected"
)

func NormalizePDSStage(stage string) PDSStage {
	switch PDSStage(stage) {
	case PDSStageSessionResume,
		PDSStageRequestBuild,
		PDSStagePDSRequest,
		PDSStagePDSResponse,
		PDSStagePostWriteIndexingWait:
		return PDSStage(stage)
	default:
		return PDSStageUnexpected
	}
}

func ClassifyPDSError(err error) PDSCategory {
	if err == nil {
		return PDSCategoryNone
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return PDSCategoryTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return PDSCategoryTimeout
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return PDSCategoryNetwork
	}
	if errors.Is(err, auth.ErrPDSSessionExpired) {
		return PDSCategoryAuth
	}
	if errors.Is(err, auth.ErrRecordNotFound) {
		return PDSCategoryNotFound
	}
	var apiErr *atclient.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return PDSCategoryValidation
		case http.StatusUnauthorized:
			return PDSCategoryAuth
		case http.StatusForbidden:
			return PDSCategoryForbidden
		case http.StatusNotFound:
			return PDSCategoryNotFound
		case http.StatusTooManyRequests:
			return PDSCategoryRateLimited
		}
		if apiErr.StatusCode >= 500 {
			return PDSCategoryServer
		}
	}
	return PDSCategoryUnexpected
}

func (o *Observer) WrapPDSFactory(factory auth.PDSClientFactory) auth.PDSClientFactory {
	if o == nil || factory == nil {
		return factory
	}
	return func(ctx context.Context, did syntax.DID, oauthSessionID string) (auth.PDSClient, error) {
		operationCtx, finish := o.startPDSOperation(ctx, PDSOperationOAuthSessionResume)
		started := time.Now()
		client, err := factory(operationCtx, did, oauthSessionID)
		o.observePDSWrite(operationCtx, PDSOperationOAuthSessionResume, PDSStageSessionResume, err, time.Since(started))
		finish(pdsResult(err))
		if err != nil || client == nil {
			return client, err
		}
		observed := observedPDSClient{inner: client, observer: o, actor: did}
		lister, hasLister := client.(auth.PDSRecordLister)
		boundary, hasBoundary := client.(auth.ActiveEffectPDSBoundary)
		switch {
		case hasLister && hasBoundary:
			return observedPDSListEffectClient{
				observedPDSEffectClient: observedPDSEffectClient{
					observedPDSClient: observed,
					boundary:          boundary,
				},
				lister: lister,
			}, nil
		case hasBoundary:
			return observedPDSEffectClient{
				observedPDSClient: observed,
				boundary:          boundary,
			}, nil
		case hasLister:
			return observedPDSListClient{observedPDSClient: observed, lister: lister}, nil
		}
		return observed, nil
	}
}

type observedPDSClient struct {
	inner    auth.PDSClient
	observer *Observer
	actor    syntax.DID
}

type observedPDSListClient struct {
	observedPDSClient
	lister auth.PDSRecordLister
}

type observedPDSEffectClient struct {
	observedPDSClient
	boundary auth.ActiveEffectPDSBoundary
}

type observedConditionalPDSClient struct {
	observedPDSClient
	deleter auth.ConditionalPDSRecordDeleter
}

type observedConditionalPutPDSClient struct {
	observedPDSClient
	putter auth.ConditionalPDSRecordPutter
}

type observedConditionalPutDeletePDSClient struct {
	observedConditionalPutPDSClient
	deleter auth.ConditionalPDSRecordDeleter
}

type observedPDSListEffectClient struct {
	observedPDSEffectClient
	lister auth.PDSRecordLister
}

type observedRepositoryCommandPDSClient struct {
	observedPDSClient
	lister  auth.PDSRecordLister
	command auth.RepositoryCommandPDSClient
}

func (c observedPDSListClient) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]auth.PDSRecord, string, error) {
	return c.observedPDSClient.observeListRecords(ctx, c.lister, repo, collection, cursor, limit)
}

func (c observedPDSListEffectClient) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]auth.PDSRecord, string, error) {
	return c.observedPDSClient.observeListRecords(ctx, c.lister, repo, collection, cursor, limit)
}

func (c observedPDSEffectClient) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	if c.boundary == nil || operation == nil {
		return errors.New("observed PDS effect boundary is unavailable")
	}
	return c.boundary.WithActiveEffects(
		ctx,
		expected,
		func(effectCtx context.Context, purposeClient auth.PDSClient) error {
			if purposeClient == nil {
				return errors.New("observed PDS effect client is unavailable")
			}
			observed := observedPDSClient{
				inner: purposeClient, observer: c.observer, actor: c.actor,
			}
			callbackClient := auth.PDSClient(observed)
			lister, hasLister := purposeClient.(auth.PDSRecordLister)
			command, hasCommand := purposeClient.(auth.RepositoryCommandPDSClient)
			if hasLister && lister != nil && hasCommand && command != nil {
				callbackClient = observedRepositoryCommandPDSClient{
					observedPDSClient: observed, lister: lister, command: command,
				}
				return operation(effectCtx, callbackClient)
			}
			putter, hasPutter := purposeClient.(auth.ConditionalPDSRecordPutter)
			deleter, hasDeleter := purposeClient.(auth.ConditionalPDSRecordDeleter)
			switch {
			case hasPutter && putter != nil && hasDeleter && deleter != nil:
				callbackClient = observedConditionalPutDeletePDSClient{
					observedConditionalPutPDSClient: observedConditionalPutPDSClient{
						observedPDSClient: observed,
						putter:            putter,
					},
					deleter: deleter,
				}
			case hasPutter && putter != nil:
				callbackClient = observedConditionalPutPDSClient{
					observedPDSClient: observed,
					putter:            putter,
				}
			case hasDeleter && deleter != nil:
				callbackClient = observedConditionalPDSClient{
					observedPDSClient: observed,
					deleter:           deleter,
				}
			}
			return operation(effectCtx, callbackClient)
		},
	)
}

func (client observedRepositoryCommandPDSClient) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]auth.PDSRecord, string, error) {
	ctx = withPDSRecordContext(ctx, client.actor, repo, syntax.NSID(collection), "")
	operationCtx, finish := client.observer.startPDSOperation(ctx, PDSOperationCommandList)
	started := time.Now()
	records, next, err := client.lister.ListRecords(operationCtx, repo, collection, cursor, limit)
	client.observer.observePDSWrite(operationCtx, PDSOperationCommandList, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return records, next, err
}

func (client observedRepositoryCommandPDSClient) LatestCommit(ctx context.Context, repo syntax.DID) (syntax.CID, error) {
	ctx = withPDSRecordContext(ctx, client.actor, repo, "", "")
	operationCtx, finish := client.observer.startPDSOperation(ctx, PDSOperationCommandHead)
	started := time.Now()
	head, err := client.command.LatestCommit(operationCtx, repo)
	client.observer.observePDSWrite(operationCtx, PDSOperationCommandHead, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return head, err
}

func (client observedRepositoryCommandPDSClient) ApplyWrites(
	ctx context.Context,
	repo syntax.DID,
	head syntax.CID,
	writes []auth.RepositoryWrite,
) error {
	ctx = withPDSRecordContext(ctx, client.actor, repo, "", "")
	operationCtx, finish := client.observer.startPDSOperation(ctx, PDSOperationCommandApply)
	started := time.Now()
	err := client.command.ApplyWrites(operationCtx, repo, head, writes)
	client.observer.observePDSWrite(operationCtx, PDSOperationCommandApply, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return err
}

func (client observedRepositoryCommandPDSClient) DeleteRecordWithRepositorySwap(
	ctx context.Context,
	repo syntax.DID,
	collection syntax.NSID,
	rkey syntax.RecordKey,
	head syntax.CID,
	record syntax.CID,
) error {
	ctx = withPDSRecordContext(ctx, client.actor, repo, collection, rkey)
	operationCtx, finish := client.observer.startPDSOperation(ctx, PDSOperationCommandFallback)
	started := time.Now()
	err := client.command.DeleteRecordWithRepositorySwap(operationCtx, repo, collection, rkey, head, record)
	client.observer.observePDSWrite(operationCtx, PDSOperationCommandFallback, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return err
}

func (c observedConditionalPutPDSClient) PutRecordWithSwap(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	record any,
	expectedCID syntax.CID,
) error {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), syntax.RecordKey(rkey))
	operation := pdsPutOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	err := c.putter.PutRecordWithSwap(
		operationCtx, repo, collection, rkey, record, expectedCID,
	)
	c.observer.observePDSWrite(
		operationCtx, operation, PDSStagePDSRequest, err, time.Since(started),
	)
	finish(pdsResult(err))
	return err
}

func (c observedConditionalPutDeletePDSClient) DeleteRecordWithSwap(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	expectedCID syntax.CID,
) error {
	return observeConditionalDelete(
		ctx, c.observer, c.deleter, repo, collection, rkey, expectedCID,
	)
}

func (c observedConditionalPDSClient) DeleteRecordWithSwap(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	expectedCID syntax.CID,
) error {
	return observeConditionalDelete(
		ctx, c.observer, c.deleter, repo, collection, rkey, expectedCID,
	)
}

func observeConditionalDelete(
	ctx context.Context,
	observer *Observer,
	deleter auth.ConditionalPDSRecordDeleter,
	repo syntax.DID,
	collection string,
	rkey string,
	expectedCID syntax.CID,
) error {
	ctx = withPDSRecordContext(ctx, repo, repo, syntax.NSID(collection), syntax.RecordKey(rkey))
	operation := pdsDeleteOperation(collection)
	operationCtx, finish := observer.startPDSOperation(ctx, operation)
	started := time.Now()
	err := deleter.DeleteRecordWithSwap(
		operationCtx, repo, collection, rkey, expectedCID,
	)
	observer.observePDSWrite(
		operationCtx, operation, PDSStagePDSRequest, err, time.Since(started),
	)
	finish(pdsResult(err))
	return err
}

func (c observedPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) (string, error) {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), syntax.RecordKey(rkey))
	operation := pdsGetOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	cid, err := c.inner.GetRecord(operationCtx, repo, collection, rkey, out)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return cid, err
}

func (c observedPDSClient) PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), syntax.RecordKey(rkey))
	operation := pdsPutOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	err := c.inner.PutRecord(operationCtx, repo, collection, rkey, record)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return err
}

func (c observedPDSClient) CreateRecord(ctx context.Context, repo syntax.DID, collection string, record any) (syntax.ATURI, syntax.CID, error) {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), syntax.RecordKey(""))
	operation := pdsCreateOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	uri, cid, err := c.inner.CreateRecord(operationCtx, repo, collection, record)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return uri, cid, err
}

func (c observedPDSClient) DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), syntax.RecordKey(rkey))
	operation := pdsDeleteOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	err := c.inner.DeleteRecord(operationCtx, repo, collection, rkey)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return err
}

func (c observedPDSClient) UploadBlob(ctx context.Context, contentType string, body []byte) (*auth.UploadedBlob, error) {
	operationCtx, finish := c.observer.startPDSOperation(ctx, PDSOperationBlobUpload)
	started := time.Now()
	blob, err := c.inner.UploadBlob(operationCtx, contentType, body)
	c.observer.observePDSWrite(operationCtx, PDSOperationBlobUpload, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return blob, err
}

func (o *Observer) startPDSOperation(ctx context.Context, operation PDSOperation) (context.Context, func(string)) {
	if o == nil {
		return ctx, func(string) {}
	}
	if !KnownPDSOperation(operation) {
		operation = "unknown"
	}
	spanCtx, span := o.StartSpan(ctx, SpanContext{Operation: string(operation), Component: "pds"})
	return spanCtx, func(result string) {
		span.Finish(result)
	}
}

func (o *Observer) observePDSWrite(ctx context.Context, operation PDSOperation, stage PDSStage, err error, duration time.Duration) {
	if o == nil {
		return
	}
	if !KnownPDSOperation(operation) {
		operation = "unknown"
	}
	result := pdsResult(err)
	category := ClassifyPDSError(err)
	stage = NormalizePDSStage(string(stage))
	o.metricRecorder.PDSOperation(ctx, string(operation), string(stage), result, string(category), duration)
	traceID, spanID := TraceIDs(ctx)
	localOnlyAttrs := []any{}
	if traceID != "" {
		localOnlyAttrs = append(localOnlyAttrs, slog.String("sentry_trace_id", traceID))
	}
	if spanID != "" {
		localOnlyAttrs = append(localOnlyAttrs, slog.String("sentry_span_id", spanID))
	}
	level := slog.LevelInfo
	if err != nil {
		level = slog.LevelWarn
		if pdsCategoryCaptured(category) {
			level = slog.LevelError
		}
		if workflow := pdsRecordWorkflow(ctx, err); workflow != nil {
			localOnlyAttrs = append(localOnlyAttrs, DiagnosticWorkflowAttr(ctx, workflow))
		}
		localOnlyAttrs = append(localOnlyAttrs, slog.Any("causes", DescribeError(err, EventContext{"component": "pds", "operation": string(operation), "failure_stage": string(stage)})))
	}
	o.Log(ctx, level, "pds write completed", EventContext{
		"component":      "pds",
		"operation":      string(operation),
		"failure_stage":  string(stage),
		"result":         result,
		"error_category": string(category),
		"duration":       duration.String(),
	}, localOnlyAttrs...)
	if err != nil {
		if !pdsCategoryCaptured(category) {
			MarkCaptured(ctx)
			return
		}
		o.CaptureDiagnostic(ctx, DiagnosticInput{Error: err, Workflow: pdsRecordWorkflow(ctx, err), Context: EventContext{
			"component":      "pds",
			"operation":      string(operation),
			"failure_stage":  string(stage),
			"result":         result,
			"error_category": string(category),
			"duration":       duration.String(),
		}})
	}
}

func pdsResult(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

func pdsCategoryCaptured(category PDSCategory) bool {
	switch category {
	case PDSCategoryTimeout, PDSCategoryNetwork, PDSCategoryServer, PDSCategoryUnexpected:
		return true
	default:
		return false
	}
}

func pdsPutOperation(collection string) PDSOperation {
	switch collection {
	case "app.bsky.actor.profile":
		return PDSOperationProfilePutBsky
	case "social.craftsky.actor.profile":
		return PDSOperationProfilePutCraftsky
	case "social.craftsky.business.profile":
		return PDSOperationBusinessProfilePut
	case "social.craftsky.business.event":
		return PDSOperationBusinessEventPut
	default:
		return "unknown"
	}
}

func pdsGetOperation(collection string) PDSOperation {
	switch collection {
	case "app.bsky.actor.profile":
		return PDSOperationProfilePutBsky
	case "social.craftsky.actor.profile":
		return PDSOperationProfilePutCraftsky
	case "social.craftsky.business.profile":
		return PDSOperationBusinessProfileGet
	case "social.craftsky.business.event":
		return PDSOperationBusinessEventGet
	default:
		return "unknown"
	}
}

func pdsCreateOperation(collection string) PDSOperation {
	switch collection {
	case "social.craftsky.feed.post":
		return PDSOperationPostCreate
	case "app.bsky.graph.follow":
		return PDSOperationFollowCreate
	case "social.craftsky.feed.like":
		return PDSOperationLikeCreate
	case "social.craftsky.feed.repost":
		return PDSOperationRepostCreate
	default:
		return "unknown"
	}
}

func pdsDeleteOperation(collection string) PDSOperation {
	switch collection {
	case "social.craftsky.business.profile":
		return PDSOperationBusinessProfileDelete
	case "social.craftsky.business.event":
		return PDSOperationBusinessEventDelete
	case "social.craftsky.feed.post":
		return PDSOperationPostDelete
	case "app.bsky.graph.follow":
		return PDSOperationFollowDelete
	case "social.craftsky.feed.like":
		return PDSOperationLikeDelete
	case "social.craftsky.feed.repost":
		return PDSOperationRepostDelete
	default:
		return "unknown"
	}
}

// Source identifiers are already parsed by their operation boundary. This never
// reads the record body, blob, cursor, repository writes or OAuth session ID.
type pdsRecordContextKey struct{}

func withPDSRecordContext(ctx context.Context, actor, target syntax.DID, nsid syntax.NSID, rkey syntax.RecordKey) context.Context {
	if actor == "" {
		actor = target
	}
	record := PublicRecordContext{ActorDID: actor, TargetDID: target, NSID: nsid, RecordKey: rkey}
	if target != "" && nsid != "" && rkey != "" {
		record.URI = syntax.ATURI("at://" + target.String() + "/" + nsid.String() + "/" + rkey.String())
	}
	return context.WithValue(ctx, pdsRecordContextKey{}, record)
}
func pdsRecordWorkflow(ctx context.Context, cause error) WorkflowContext {
	if cause == nil {
		return nil
	}
	if private, ok := ctx.Value(privateDiagnosticContextKey{}).(privateFailureContext); ok {
		return private
	}
	// A record's destination alone is not public-workflow provenance.
	request, classified := ctx.Value(requestDiagnosticContextKey{}).(RequestDiagnosticContext)
	if !classified || !request.PublicTargets {
		return RequestFailureWorkflow(ctx, cause)
	}

	if record, ok := ctx.Value(pdsRecordContextKey{}).(PublicRecordContext); ok {
		return workflowForRequest(ctx, record)
	}
	return RequestPublicWorkflow(ctx)
}

func (c observedPDSClient) observeListRecords(ctx context.Context, lister auth.PDSRecordLister, repo syntax.DID, collection, cursor string, limit int) ([]auth.PDSRecord, string, error) {
	ctx = withPDSRecordContext(ctx, c.actor, repo, syntax.NSID(collection), "")
	operationCtx, finish := c.observer.startPDSOperation(ctx, PDSOperationCommandList)
	started := time.Now()
	records, next, err := lister.ListRecords(operationCtx, repo, collection, cursor, limit)
	c.observer.observePDSWrite(operationCtx, PDSOperationCommandList, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return records, next, err
}
