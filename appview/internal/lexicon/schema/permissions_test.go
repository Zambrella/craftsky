package schema_test

import (
	"embed"
	"encoding/json"
	"io/fs"
	"reflect"
	"slices"
	"testing"
)

//go:embed auth actor business feed project
var consentSchemas embed.FS

func TestPermissionSetCoversExactlyCraftSkyRecordCollections(t *testing.T) {
	var records, permissions []string
	err := fs.WalkDir(consentSchemas, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := consentSchemas.ReadFile(path)
		if err != nil {
			return err
		}
		var schema struct {
			ID   string `json:"id"`
			Defs map[string]struct {
				Type        string `json:"type"`
				Permissions []struct {
					Resource   string   `json:"resource"`
					Collection []string `json:"collection"`
				} `json:"permissions"`
			} `json:"defs"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			return err
		}
		main := schema.Defs["main"]
		if main.Type == "record" {
			records = append(records, schema.ID)
		}
		if schema.ID == "social.craftsky.authFull" {
			if main.Type != "permission-set" || len(main.Permissions) != 1 || main.Permissions[0].Resource != "repo" {
				t.Fatal("permission set must contain only CraftSky record authority")
			}
			permissions = main.Permissions[0].Collection
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(records)
	slices.Sort(permissions)
	if len(records) == 0 || !reflect.DeepEqual(records, permissions) {
		t.Fatalf("permission collections=%v, actual record collections=%v", permissions, records)
	}
}
