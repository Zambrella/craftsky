package api

import (
	"context"
	"errors"
	"log/slog"

	"social.craftsky/appview/internal/observability"
)

const (
	pdsOperationProfilePutBsky     = observability.PDSOperationProfilePutBsky
	pdsOperationProfilePutCraftsky = observability.PDSOperationProfilePutCraftsky
	pdsOperationPostCreate         = observability.PDSOperationPostCreate
	pdsOperationPostDelete         = observability.PDSOperationPostDelete
	pdsOperationBlobUpload         = observability.PDSOperationBlobUpload
	pdsOperationLikeCreate         = observability.PDSOperationLikeCreate
	pdsOperationLikeDelete         = observability.PDSOperationLikeDelete
	pdsOperationRepostCreate       = observability.PDSOperationRepostCreate
	pdsOperationRepostDelete       = observability.PDSOperationRepostDelete

	pdsStageSessionResume = observability.PDSStageSessionResume
	pdsStageRequestBuild  = observability.PDSStageRequestBuild
	pdsStagePDSRequest    = observability.PDSStagePDSRequest
)

func pdsLogAttrs(runID string, operation observability.PDSOperation, stage observability.PDSStage) []any {
	attrs := []any{
		slog.String("component", "pds"),
		slog.String("operation", string(operation)),
		slog.String("stage", string(stage)),
	}
	if runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}
	return attrs
}

func pdsLogSuccessAttrs(runID string, operation observability.PDSOperation, stage observability.PDSStage) []any {
	return append(pdsLogAttrs(runID, operation, stage), slog.String("result", "success"))
}

func pdsLogErrorAttrs(runID string, operation observability.PDSOperation, stage observability.PDSStage, err error) []any {
	return append(pdsLogAttrs(runID, operation, stage),
		slog.String("result", "error"),
		slog.String("error_category", string(observability.ClassifyPDSError(err))),
		slog.Any("causes", observability.DescribeError(err, observability.EventContext{"component": "pds", "operation": string(operation), "failure_stage": string(stage)})))
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func apiLogAttrs(runID, operation string) []any {
	attrs := []any{
		slog.String("component", "api"),
		slog.String("operation", operation),
	}
	if runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}
	return attrs
}

func apiLogSuccessAttrs(runID, operation string) []any {
	return append(apiLogAttrs(runID, operation), slog.String("result", "success"))
}

func apiLogErrorAttrs(ctx context.Context, runID, operation, category string, err error) []any {
	fields := observability.EventContext{"component": "api", "operation": operation, "error_category": category, "failure_stage": category, "run_id": runID, "result": "error"}
	workflow := observability.RequestPublicWorkflow(ctx)
	observability.CaptureRequestDiagnostic(ctx, observability.DiagnosticInput{Error: err, Context: fields, Workflow: workflow})
	attrs := append(apiLogAttrs(runID, operation), slog.String("result", "error"), slog.String("error_category", category), slog.String("failure_stage", category), slog.Any("causes", observability.DescribeError(err, fields)))
	if workflow != nil {
		attrs = append(attrs, observability.DiagnosticWorkflowAttr(ctx, workflow))
	}
	return attrs
}

func requestCanceled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)
}

func postAuthorListOperation(label string) string {
	switch label {
	case "project list":
		return "project.author.list"
	case "comment list":
		return "comment.author.list"
	case "repost list":
		return "repost.author.list"
	default:
		return "post.author.list"
	}
}

func profileGraphListOperation(label string) string {
	switch label {
	case "following":
		return "profile.following.list"
	default:
		return "profile.followers.list"
	}
}
