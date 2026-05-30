// appview/internal/api/report_response.go
package api

import "encoding/json"

// AcceptedReportResponse is the minimal user-facing report acceptance body.
// It intentionally excludes private details, forwarding payloads, moderation
// state, report counts, and reason text.
type AcceptedReportResponse struct {
	ReportID string `json:"reportId"`
	Status   string `json:"status"`
}

func NewAcceptedReportResponse(reportID string) AcceptedReportResponse {
	return AcceptedReportResponse{ReportID: reportID, Status: "accepted"}
}

func (r AcceptedReportResponse) MarshalJSON() ([]byte, error) {
	type wire struct {
		ReportID string `json:"reportId"`
		Status   string `json:"status"`
	}
	status := r.Status
	if status == "" {
		status = "accepted"
	}
	return json.Marshal(wire{ReportID: r.ReportID, Status: status})
}
