package accountdeletion

import (
	"reflect"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestBlobBoundaryIsReferenceOnlyAndNeverTerminalGate(t *testing.T) {
	t.Parallel()

	capability := reflect.TypeOf((*auth.DeletionPDSClient)(nil)).Elem()
	for _, forbidden := range []string{"DeleteBlob", "UploadBlob", "DeleteAccount"} {
		if _, exists := capability.MethodByName(forbidden); exists {
			t.Fatalf("deletion PDS capability exposes forbidden %s", forbidden)
		}
	}
}
