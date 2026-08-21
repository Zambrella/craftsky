package scheduledposts

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestPrivateMetadataIsOpaqueAndDiagnosticFieldsAreSafe(t *testing.T) {
	t.Parallel()

	canaries := []string{
		"did:plc:private-owner-canary",
		"private-handle.example",
		"secret-filename.jpg",
		"private alt text canary",
		"2026-07-31T12:34:00Z",
		"oauth-token-canary",
	}
	mediaID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	objectKey, attemptID, err := NewGenerationObjectKey(
		syntax.DID("did:plc:private-owner-canary"),
		7,
		mediaID,
	)
	if err != nil {
		t.Fatalf("NewGenerationObjectKey() error = %v", err)
	}
	if objectKey != "scheduled-media/v2/7/"+attemptID.String() {
		t.Fatalf("NewGenerationObjectKey() = %q, want opaque generation path", objectKey)
	}

	fields := SafeDiagnosticFields(DiagnosticOperationPublish, DiagnosticResultFailure, "pds_unavailable")
	if got := fmt.Sprint(fields); got != "map[component:scheduled_posts errorClass:pds_unavailable operation:publish result:failure]" {
		t.Fatalf("SafeDiagnosticFields() = %s", got)
	}
	unknown := SafeDiagnosticFields(DiagnosticOperation("member-content"), DiagnosticResult("owner-id"), "raw provider response")
	if got := fmt.Sprint(unknown); got != "map[component:scheduled_posts errorClass:unknown operation:unknown result:unknown]" {
		t.Fatalf("unsafe diagnostics were not reduced: %s", got)
	}

	allOutput := objectKey + fmt.Sprint(fields) + fmt.Sprint(unknown)
	for _, canary := range canaries {
		if strings.Contains(allOutput, canary) {
			t.Fatalf("private canary %q leaked into metadata output %q", canary, allOutput)
		}
	}
}
