// appview/internal/api/report_request_test.go
package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestNormalizeReportDetails_TrimsAndOmitsEmptyPlainText(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input *string
		want  *string
	}{
		{name: "omitted"},
		{name: "empty", input: ptrString("")},
		{name: "whitespace", input: ptrString(" \n\t ")},
		{name: "trimmed", input: ptrString("  useful private context  "), want: ptrString("useful private context")},
		{name: "thousand chars", input: ptrString(strings.Repeat("a", 1000)), want: ptrString(strings.Repeat("a", 1000))},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := api.NormalizeReportDetails(tc.input)
			if err != nil {
				t.Fatalf("NormalizeReportDetails: %v", err)
			}
			assertStringPtr(t, "details", got, tc.want)
		})
	}
}

func TestNormalizeReportDetails_RejectsOverOneThousandCharacters(t *testing.T) {
	t.Parallel()
	_, err := api.NormalizeReportDetails(ptrString(strings.Repeat("a", 1001)))
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed FieldError, got %v", err)
	}
	if _, ok := fe.Fields["details"]; !ok {
		t.Fatalf("fields = %v, want details", fe.Fields)
	}
}

func TestValidateReportRequest_AllowsOtherWithoutDetails(t *testing.T) {
	t.Parallel()
	err := api.ValidateReportRequest(api.ReportRequest{ReasonType: "other"})
	if err != nil {
		t.Fatalf("ValidateReportRequest other without details: %v", err)
	}
}

func TestValidateReportRequest_ApprovedReasonTaxonomy(t *testing.T) {
	t.Parallel()
	for _, reason := range []string{
		"harassment",
		"hate",
		"spam",
		"misleading",
		"suspected_ai_generated",
		"adult_or_graphic",
		"impersonation",
		"off_topic",
		"intellectual_property",
		"other",
	} {
		reason := reason
		t.Run(reason, func(t *testing.T) {
			t.Parallel()
			if err := api.ValidateReportRequest(api.ReportRequest{ReasonType: reason}); err != nil {
				t.Fatalf("ValidateReportRequest(%q): %v", reason, err)
			}
		})
	}
}

func TestValidateReportRequest_RejectsMissingOrUnsupportedReason(t *testing.T) {
	t.Parallel()
	for _, reason := range []string{"", "not_allowed"} {
		reason := reason
		t.Run(reason, func(t *testing.T) {
			t.Parallel()
			err := api.ValidateReportRequest(api.ReportRequest{ReasonType: reason})
			var fe *api.FieldError
			if !errors.As(err, &fe) || fe.Code != "validation_failed" {
				t.Fatalf("want validation_failed FieldError, got %v", err)
			}
			if _, ok := fe.Fields["reasonType"]; !ok {
				t.Fatalf("fields = %v, want reasonType", fe.Fields)
			}
		})
	}
}
