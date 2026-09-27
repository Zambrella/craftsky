package app

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

// UT-019 / IT-017 / REG-009: the private relationship mutation boundary must
// not regain public PDS record mutation capabilities after block migration.
func TestLegacyMutationConformanceRelationshipServiceIsPrivateOnly(t *testing.T) {
	t.Parallel()

	mutationPath := filepath.Join("..", "relationships", "mutation_service.go")
	mutationFile := parseConformanceFile(t, mutationPath)
	for _, imported := range mutationFile.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatalf("unquote import in %s: %v", mutationPath, err)
		}
		if strings.HasSuffix(path, "/internal/pdseffects") {
			t.Errorf("%s imports legacy public-record effects", mutationPath)
		}
	}
	for _, declaration := range mutationFile.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if function.Name.Name == "Block" || function.Name.Name == "Unblock" {
			t.Errorf("relationships.MutationService still exposes obsolete %s", function.Name.Name)
		}
	}

	apiPath := filepath.Join("..", "api", "relationship.go")
	apiFile := parseConformanceFile(t, apiPath)
	for _, declaration := range apiFile.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range generic.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "RelationshipMutationService" {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				t.Fatal("RelationshipMutationService is not an interface")
			}
			for _, method := range iface.Methods.List {
				for _, name := range method.Names {
					if name.Name == "Block" || name.Name == "Unblock" {
						t.Errorf("api.RelationshipMutationService still exposes obsolete %s", name.Name)
					}
				}
			}
		}
	}
}

// IT-017 / REG-009: Tap source authority is independent of the origin of a
// public record. Historical effect rows may still be purged, but cannot gate
// ingestion, projection, repair, or lifecycle transitions.
func TestLegacyMutationConformanceProductionCannotReachEffectOriginGating(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join("..", "ingestion", "store.go"),
		filepath.Join("..", "ingestion", "reconciliation.go"),
		filepath.Join("..", "ingestion", "service.go"),
		"deps_tap.go",
	} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, forbidden := range []string{
			"EffectOperationID",
			"LockedOperationID",
			"ResolvePDSRecordSourceTx",
			"LockedPDSRecordEffectTx",
			"prepareEffectSourcesForRejoinTx",
			"PDSAttemptDepartureParticipant",
		} {
			if strings.Contains(string(contents), forbidden) {
				t.Errorf("%s retains production effect-origin gate %s", path, forbidden)
			}
		}
	}
}

// UT-019 / IT-017 / REG-009: set projection must not call the legacy
// single-winner table projectors now that normalized sources own duplicates.
func TestLegacyMutationConformanceSetProjectionHasNoWinnerReplacementPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "index", "transactional_projectors.go")
	file := parseConformanceFile(t, path)
	setReceivers := map[string]bool{
		"CraftskyLike":   true,
		"CraftskyRepost": true,
		"BlueskyFollow":  true,
		"BlueskyBlock":   true,
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Project" || function.Recv == nil || len(function.Recv.List) != 1 {
			continue
		}
		receiver, ok := function.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		receiverType, ok := receiver.X.(*ast.Ident)
		if !ok || !setReceivers[receiverType.Name] {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "Handle" {
				t.Errorf("%s.Project calls legacy winner projector Handle", receiverType.Name)
			}
			return true
		})
	}

	likeRead, err := os.ReadFile(filepath.Join("..", "api", "post_interactions_store.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(likeRead), `findActiveInteraction(ctx, "craftsky_likes"`) {
		t.Error("FindActiveLike still reads the obsolete single-winner cache")
	}
}

func parseConformanceFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}
