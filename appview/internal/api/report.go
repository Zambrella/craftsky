// appview/internal/api/report.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// PostReportTarget is the canonical indexed post identity stored with a report.
type PostReportTarget struct {
	DID         string
	Rkey        string
	URI         string
	CIDSnapshot string
}

// AccountReportTarget is the canonical account identity stored with a report.
type AccountReportTarget struct {
	DID                     string
	SubmittedHandleSnapshot string
}

type PostReportTargetResolver interface {
	ResolvePostReportTarget(ctx context.Context, did syntax.DID, rkey syntax.RecordKey) (*PostReportTarget, error)
}

type AccountReportTargetResolver interface {
	ResolveAccountReportTarget(ctx context.Context, handleOrDID string) (*AccountReportTarget, error)
}

type ReportCreator interface {
	CreateReport(ctx context.Context, input CreateReportInput) (*ReportRow, error)
}

// ReportPostHandler serves POST /v1/posts/{did}/{rkey}/reports.
func ReportPostHandler(targets PostReportTargetResolver, reports ReportCreator, forwarder ReportForwarder, logger *slog.Logger) http.Handler {
	logger = normalizeLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		reporterDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		deviceID, _ := middleware.GetDeviceID(r.Context())
		subjectDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "not a valid DID", runID, nil)
			return
		}
		rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "not a valid record key", runID, nil)
			return
		}

		req, normalizedDetails, ok := decodeValidateReport(w, r, runID)
		if !ok {
			return
		}

		target, err := targets.ResolvePostReportTarget(r.Context(), subjectDID, rkey)
		if errors.Is(err, ErrPostNotFound) || target == nil {
			envelope.WriteError(w, http.StatusNotFound, "post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("report post: resolve target failed",
				apiLogErrorAttrs(runID, "report.post.create", "target_lookup")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report target lookup failed", runID, nil)
			return
		}
		if target.DID == reporterDID.String() {
			writeInvalidReportTarget(w, runID)
			return
		}

		collection := craftskyPostNSID
		subject := ReportSubjectSnapshot{Type: ReportSubjectPost, DID: target.DID, Collection: &collection, Rkey: &target.Rkey, URI: &target.URI, CIDSnapshot: &target.CIDSnapshot}
		metadata, err := forwarder.Prepare(r.Context(), ReportForwardingInput{ReporterDID: reporterDID.String(), Subject: subject, ReasonType: req.ReasonType, Details: normalizedDetails})
		if err != nil {
			logger.Error("report post: prepare forwarding failed",
				apiLogErrorAttrs(runID, "report.post.create", "forwarding")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report forwarding preparation failed", runID, nil)
			return
		}

		row, err := reports.CreateReport(r.Context(), CreateReportInput{
			ReporterDID:             reporterDID.String(),
			SubjectType:             ReportSubjectPost,
			SubjectDID:              target.DID,
			SubjectCollection:       &collection,
			SubjectRkey:             &target.Rkey,
			SubjectURI:              &target.URI,
			SubjectCIDSnapshot:      &target.CIDSnapshot,
			ReasonType:              req.ReasonType,
			Details:                 normalizedDetails,
			DeviceID:                optionalString(deviceID),
			ForwardingStatus:        metadata.Status,
			ForwardingSchemaVersion: metadata.SchemaVersion,
			ForwardingPreparedAt:    metadata.PreparedAt,
		})
		if err != nil {
			logger.Error("report post: create report failed",
				apiLogErrorAttrs(runID, "report.post.create", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report persistence failed", runID, nil)
			return
		}
		writeAcceptedReport(w, row.ID)
	})
}

// ReportProfileHandler serves POST /v1/profiles/{handleOrDid}/reports.
func ReportProfileHandler(targets AccountReportTargetResolver, reports ReportCreator, forwarder ReportForwarder, logger *slog.Logger) http.Handler {
	logger = normalizeLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		reporterDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		deviceID, _ := middleware.GetDeviceID(r.Context())
		rawTarget := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")

		req, normalizedDetails, ok := decodeValidateReport(w, r, runID)
		if !ok {
			return
		}

		target, err := targets.ResolveAccountReportTarget(r.Context(), rawTarget)
		if errors.Is(err, ErrProfileNotFound) || target == nil {
			envelope.WriteError(w, http.StatusNotFound, "profile_not_found", "profile not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("report profile: resolve target failed",
				apiLogErrorAttrs(runID, "report.profile.create", "target_lookup")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report target lookup failed", runID, nil)
			return
		}
		if target.DID == reporterDID.String() {
			writeInvalidReportTarget(w, runID)
			return
		}

		subject := ReportSubjectSnapshot{Type: ReportSubjectAccount, DID: target.DID, HandleSnapshot: optionalString(target.SubmittedHandleSnapshot)}
		metadata, err := forwarder.Prepare(r.Context(), ReportForwardingInput{ReporterDID: reporterDID.String(), Subject: subject, ReasonType: req.ReasonType, Details: normalizedDetails})
		if err != nil {
			logger.Error("report profile: prepare forwarding failed",
				apiLogErrorAttrs(runID, "report.profile.create", "forwarding")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report forwarding preparation failed", runID, nil)
			return
		}

		row, err := reports.CreateReport(r.Context(), CreateReportInput{
			ReporterDID:             reporterDID.String(),
			SubjectType:             ReportSubjectAccount,
			SubjectDID:              target.DID,
			SubmittedHandleSnapshot: optionalString(target.SubmittedHandleSnapshot),
			ReasonType:              req.ReasonType,
			Details:                 normalizedDetails,
			DeviceID:                optionalString(deviceID),
			ForwardingStatus:        metadata.Status,
			ForwardingSchemaVersion: metadata.SchemaVersion,
			ForwardingPreparedAt:    metadata.PreparedAt,
		})
		if err != nil {
			logger.Error("report profile: create report failed",
				apiLogErrorAttrs(runID, "report.profile.create", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "report persistence failed", runID, nil)
			return
		}
		writeAcceptedReport(w, row.ID)
	})
}

func decodeValidateReport(w http.ResponseWriter, r *http.Request, runID string) (ReportRequest, *string, bool) {
	req, err := DecodeReportRequest(r.Body)
	if err != nil {
		if fe, ok := err.(*FieldError); ok {
			status := http.StatusBadRequest
			envelope.WriteError(w, status, fe.Code, "request body rejected", runID, fe.Fields)
			return ReportRequest{}, nil, false
		}
		envelope.WriteError(w, http.StatusBadRequest, "malformed_body", "could not parse body", runID, nil)
		return ReportRequest{}, nil, false
	}
	if err := ValidateReportRequest(req); err != nil {
		if fe, ok := err.(*FieldError); ok {
			envelope.WriteError(w, http.StatusUnprocessableEntity, fe.Code, "validation failed", runID, fe.Fields)
			return ReportRequest{}, nil, false
		}
		envelope.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "validation failed", runID, nil)
		return ReportRequest{}, nil, false
	}
	normalizedDetails, err := NormalizeReportDetails(req.Details)
	if err != nil {
		envelope.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "validation failed", runID, nil)
		return ReportRequest{}, nil, false
	}
	return req, normalizedDetails, true
}

func writeAcceptedReport(w http.ResponseWriter, reportID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(NewAcceptedReportResponse(reportID))
}

func writeInvalidReportTarget(w http.ResponseWriter, runID string) {
	envelope.WriteError(w, http.StatusUnprocessableEntity, "invalid_report_target", "You cannot report your own post or profile.", runID, nil)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func normalizeLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
