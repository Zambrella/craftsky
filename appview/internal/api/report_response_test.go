// appview/internal/api/report_response_test.go
package api_test

import (
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestAcceptedReportResponse_JSONIsMinimal(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(api.AcceptedReportResponse{ReportID: "report-1"})
	if err != nil {
		t.Fatalf("marshal accepted report response: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal accepted report response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("response keys = %v, want only reportId/status", got)
	}
	if got["reportId"] != "report-1" {
		t.Fatalf("reportId = %v", got["reportId"])
	}
	if got["status"] != "accepted" {
		t.Fatalf("status = %v, want accepted", got["status"])
	}
	for _, forbidden := range []string{"details", "forwardingPayload", "moderation", "reportCount", "reasonType"} {
		if _, ok := got[forbidden]; ok {
			t.Fatalf("response leaked forbidden key %q in %v", forbidden, got)
		}
	}
}
