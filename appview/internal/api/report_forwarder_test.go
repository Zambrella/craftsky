// appview/internal/api/report_forwarder_test.go
package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestPlaceholderReportForwarder_PreparesSafeMetadataOnly(t *testing.T) {
	t.Parallel()
	privateDetails := "private reporter details"
	forwarder := api.NewPlaceholderReportForwarder(func() time.Time {
		return time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	})

	metadata, err := forwarder.Prepare(context.Background(), api.ReportForwardingInput{
		ReportID:    "report-1",
		ReporterDID: "did:plc:alice",
		Subject: api.ReportSubjectSnapshot{
			Type:           api.ReportSubjectPost,
			DID:            "did:plc:bob",
			Collection:     ptrString("social.craftsky.feed.post"),
			Rkey:           ptrString("3lf2abc"),
			URI:            ptrString("at://did:plc:bob/social.craftsky.feed.post/3lf2abc"),
			CIDSnapshot:    ptrString("bafy-post-v1"),
			HandleSnapshot: ptrString("bob.craftsky.social"),
		},
		ReasonType: "spam",
		Details:    &privateDetails,
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if metadata.Status != "prepared_not_submitted" {
		t.Fatalf("status = %q, want prepared_not_submitted", metadata.Status)
	}
	if metadata.SchemaVersion == nil || *metadata.SchemaVersion != "atproto-create-report-v0" {
		t.Fatalf("schema version = %v, want atproto-create-report-v0", metadata.SchemaVersion)
	}
	if !metadata.PreparedAt.Equal(time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("prepared at = %s", metadata.PreparedAt)
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	encodedText := string(encoded)
	for _, forbidden := range []string{"private reporter details", "spam", "did:plc:alice", "did:plc:bob", "3lf2abc", "bafy-post-v1"} {
		if strings.Contains(encodedText, forbidden) {
			t.Fatalf("metadata leaked %q in %s", forbidden, encodedText)
		}
	}
}
