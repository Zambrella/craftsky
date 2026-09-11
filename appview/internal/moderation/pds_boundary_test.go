package moderation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestModerationDomainHasNoPDSMutationSeam(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	forbiddenNames := map[string]bool{
		"ApplyWrites":          true,
		"CreateRecord":         true,
		"DeactivateAccount":    true,
		"DeleteAccount":        true,
		"DeleteRecord":         true,
		"DeleteRecordWithSwap": true,
		"DeletionPDSClient":    true,
		"PDSClient":            true,
		"PDSClientFactory":     true,
		"PutRecord":            true,
		"PutRecordWithSwap":    true,
	}
	forbiddenImports := map[string]bool{
		"github.com/bluesky-social/indigo/api/atproto": true,
		"github.com/bluesky-social/indigo/xrpc":        true,
		"net/http":                                     true,
		"social.craftsky/appview/internal/auth":        true,
		"social.craftsky/appview/internal/pdseffects":  true,
	}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(".", entry.Name())
		parsed, err := parser.ParseFile(files, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports from %s: %v", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("parse import in %s: %v", path, err)
			}
			if forbiddenImports[importPath] {
				t.Errorf("%s imports forbidden PDS or network mutation dependency %s", path, importPath)
			}
		}

		parsed, err = parser.ParseFile(files, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && (forbiddenNames[identifier.Name] || strings.Contains(strings.ToLower(identifier.Name), "pds")) {
				t.Errorf("%s exposes or invokes forbidden PDS mutation capability %s", files.Position(identifier.Pos()), identifier.Name)
			}
			return true
		})
	}
}
