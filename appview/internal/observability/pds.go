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
)

type PDSOperation string

const (
	PDSOperationOAuthSessionResume PDSOperation = "oauth.session_resume"
	PDSOperationProfilePutBsky     PDSOperation = "profile.put_bsky"
	PDSOperationProfilePutCraftsky PDSOperation = "profile.put_craftsky"
	PDSOperationPostCreate         PDSOperation = "post.create"
	PDSOperationPostDelete         PDSOperation = "post.delete"
	PDSOperationBlobUpload         PDSOperation = "blob.upload"
	PDSOperationFollowCreate       PDSOperation = "follow.create"
	PDSOperationFollowDelete       PDSOperation = "follow.delete"
	PDSOperationLikeCreate         PDSOperation = "like.create"
	PDSOperationLikeDelete         PDSOperation = "like.delete"
	PDSOperationRepostCreate       PDSOperation = "repost.create"
	PDSOperationRepostDelete       PDSOperation = "repost.delete"
)

var knownPDSOperations = map[PDSOperation]struct{}{
	PDSOperationOAuthSessionResume: {},
	PDSOperationProfilePutBsky:     {},
	PDSOperationProfilePutCraftsky: {},
	PDSOperationPostCreate:         {},
	PDSOperationPostDelete:         {},
	PDSOperationBlobUpload:         {},
	PDSOperationFollowCreate:       {},
	PDSOperationFollowDelete:       {},
	PDSOperationLikeCreate:         {},
	PDSOperationLikeDelete:         {},
	PDSOperationRepostCreate:       {},
	PDSOperationRepostDelete:       {},
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
		return observedPDSClient{inner: client, observer: o}, nil
	}
}

type observedPDSClient struct {
	inner    auth.PDSClient
	observer *Observer
}

func (c observedPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) (string, error) {
	return c.inner.GetRecord(ctx, repo, collection, rkey, out)
}

func (c observedPDSClient) PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error {
	operation := pdsPutOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	err := c.inner.PutRecord(operationCtx, repo, collection, rkey, record)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return err
}

func (c observedPDSClient) CreateRecord(ctx context.Context, repo syntax.DID, collection string, record any) (syntax.ATURI, syntax.CID, error) {
	operation := pdsCreateOperation(collection)
	operationCtx, finish := c.observer.startPDSOperation(ctx, operation)
	started := time.Now()
	uri, cid, err := c.inner.CreateRecord(operationCtx, repo, collection, record)
	c.observer.observePDSWrite(operationCtx, operation, PDSStagePDSRequest, err, time.Since(started))
	finish(pdsResult(err))
	return uri, cid, err
}

func (c observedPDSClient) DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error {
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
		o.CaptureError(ctx, EventContext{
			"component":      "pds",
			"operation":      string(operation),
			"failure_stage":  string(stage),
			"result":         result,
			"error_category": string(category),
			"duration":       duration.String(),
		}, err)
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
