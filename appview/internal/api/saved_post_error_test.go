package api

import (
	"net/http"
	"testing"
)

func TestSavedPostFolderErrorMappingIsOperationSpecificAndIndistinguishable(t *testing.T) {
	for _, operation := range []savedPostFolderOperation{
		savedPostFolderAssignment,
		savedPostFolderRename,
		savedPostFolderScopedList,
	} {
		status, code, handled := mapSavedPostFolderError(operation, ErrSavedPostFolderNotFound)
		if !handled || status != http.StatusNotFound || code != "saved_post_folder_not_found" {
			t.Fatalf("operation %q mapped to %d/%q/%v", operation, status, code, handled)
		}
	}

	status, code, handled := mapSavedPostFolderError(savedPostFolderDelete, ErrSavedPostFolderNotFound)
	if !handled || status != http.StatusNoContent || code != "" {
		t.Fatalf("delete mapped to %d/%q/%v", status, code, handled)
	}
	if _, _, handled := mapSavedPostFolderError(savedPostFolderRename, errSavedPostTestSentinel{}); handled {
		t.Fatal("unrelated store error was handled as folder not found")
	}
}

type errSavedPostTestSentinel struct{}

func (errSavedPostTestSentinel) Error() string { return "sentinel" }
