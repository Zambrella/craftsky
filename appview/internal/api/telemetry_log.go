package api

import (
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
		slog.String("error_category", string(observability.ClassifyPDSError(err))))
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

func apiLogErrorAttrs(runID, operation, category string) []any {
	return append(apiLogAttrs(runID, operation),
		slog.String("result", "error"),
		slog.String("error_category", category))
}

func apiLogInvalidAttrs(runID, operation string) []any {
	return append(apiLogAttrs(runID, operation), slog.String("result", "invalid"))
}

func postAuthorListOperation(label string) string {
	switch label {
	case "project list":
		return "project.author.list"
	case "comment list":
		return "comment.author.list"
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
