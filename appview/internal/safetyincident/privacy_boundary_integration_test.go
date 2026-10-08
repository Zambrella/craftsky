package safetyincident

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestrictedSafetyDataHasNoPublicPDSRecordPath(t *testing.T) {
	lexiconRoot := filepath.Join("..", "..", "..", "lexicon", "social", "craftsky")
	forbidden := []string{"safetyIncident", "restrictedEvidence", "legalHold", "safetyIntake"}
	err := filepath.WalkDir(lexiconRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, value := range forbidden {
			if strings.Contains(string(content), value) {
				t.Errorf("private safety type %q found in public lexicon %s", value, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, operation := range []string{"com.atproto.repo.createRecord", "com.atproto.repo.applyWrites"} {
			if strings.Contains(string(content), operation) {
				t.Errorf("restricted package contains PDS write operation %q in %s", operation, path)
			}
		}
	}
}
