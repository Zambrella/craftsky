// appview/internal/api/report_request.go
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

const maxReportDetailsCharacters = 1000

// ReportRequest is the shared body shape accepted by post/profile report
// endpoints. Details are AppView-private plain text.
type ReportRequest struct {
	ReasonType string  `json:"reasonType"`
	Details    *string `json:"details,omitempty"`
}

// DecodeReportRequest reads the shared JSON report body and rejects malformed
// JSON or unknown fields using the standard FieldError convention.
func DecodeReportRequest(body io.Reader) (ReportRequest, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return ReportRequest{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return ReportRequest{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}
	allowed := map[string]struct{}{"reasonType": {}, "details": {}}
	unknown := map[string]string{}
	for key := range rawMap {
		if _, ok := allowed[key]; !ok {
			unknown[key] = "unknown field"
		}
	}
	if len(unknown) > 0 {
		return ReportRequest{}, &FieldError{Code: "unexpected_field", Fields: unknown}
	}
	out := ReportRequest{}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&out); err != nil {
		return ReportRequest{}, &FieldError{Code: "unexpected_field", Fields: map[string]string{"_": err.Error()}}
	}
	return out, nil
}

var approvedReportReasons = map[string]struct{}{
	"harassment":             {},
	"hate":                   {},
	"spam":                   {},
	"misleading":             {},
	"suspected_ai_generated": {},
	"adult_or_graphic":       {},
	"impersonation":          {},
	"off_topic":              {},
	"intellectual_property":  {},
	"other":                  {},
}

// IsApprovedReportReason reports whether reason is in the stable MVP taxonomy.
func IsApprovedReportReason(reason string) bool {
	_, ok := approvedReportReasons[reason]
	return ok
}

// NormalizeReportDetails trims optional report details, treats omitted/empty as
// nil, and enforces the 1,000-character private-text limit.
func NormalizeReportDetails(details *string) (*string, error) {
	if details == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*details)
	if trimmed == "" {
		return nil, nil
	}
	if len([]rune(trimmed)) > maxReportDetailsCharacters {
		return nil, &FieldError{
			Code:   "validation_failed",
			Fields: map[string]string{"details": "exceeds 1000 characters"},
		}
	}
	return &trimmed, nil
}

// ValidateReportRequest validates the stable report reason taxonomy and
// normalizes optional details. The reason "other" does not require details.
func ValidateReportRequest(req ReportRequest) error {
	fields := map[string]string{}
	if !IsApprovedReportReason(req.ReasonType) {
		fields["reasonType"] = "must be an approved report reason"
	}
	if _, err := NormalizeReportDetails(req.Details); err != nil {
		var detailErr *FieldError
		if fe, ok := err.(*FieldError); ok {
			detailErr = fe
		}
		if detailErr != nil {
			for k, v := range detailErr.Fields {
				fields[k] = v
			}
		} else {
			fields["details"] = err.Error()
		}
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}
