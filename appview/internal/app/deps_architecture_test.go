package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewDepsDelegatesObservabilityConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newObservabilityDependencies") {
		t.Fatal("newDeps must delegate observer construction and cleanup ownership to newObservabilityDependencies")
	}
}

func TestNewDepsDelegatesContentConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newContentDependencies") {
		t.Fatal("newDeps must delegate AppView read/write model construction to newContentDependencies")
	}
}

func TestNewDepsDelegatesPDSEffectConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newPDSEffectDependencies") {
		t.Fatal("newDeps must delegate authenticated PDS capability construction to newPDSEffectDependencies")
	}
}

func TestNewDepsDelegatesPushConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newPushDependencies") {
		t.Fatal("newDeps must delegate optional Firebase and dispatcher construction to newPushDependencies")
	}
}

func TestNewDepsDelegatesAccountDeletionConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newAccountDeletionDependencies") {
		t.Fatal("newDeps must delegate deletion service, worker, and expiry construction to newAccountDeletionDependencies")
	}
}

func TestNewDepsDelegatesInstagramStorageConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newInstagramStorageDependencies") {
		t.Fatal("newDeps must delegate private Instagram storage and policy construction to newInstagramStorageDependencies")
	}
}

func TestNewDepsDelegatesTapConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newTapDependencies") {
		t.Fatal("newDeps must delegate the durable ingestion pipeline to newTapDependencies")
	}
}

func TestNewDepsDelegatesInstagramRuntimeConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newInstagramRuntimeDependencies") {
		t.Fatal("newDeps must delegate verification, Meta, import, suggestion, and reconciliation construction to newInstagramRuntimeDependencies")
	}
}

func TestNewDepsDelegatesScheduledLifecycleConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newScheduledLifecycleDependencies") {
		t.Fatal("newDeps must delegate scheduled media, cleanup, and departure construction to newScheduledLifecycleDependencies")
	}
}

func TestNewDepsDelegatesScheduledPublicationConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newScheduledPublicationDependencies") {
		t.Fatal("newDeps must delegate publication processor, manual service, and worker construction to newScheduledPublicationDependencies")
	}
}

func TestNewDepsDelegatesContentRuntimeConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newContentRuntimeDependencies") {
		t.Fatal("newDeps must delegate identity cache and relationship mutation construction to newContentRuntimeDependencies")
	}
}

func TestNewDepsDelegatesAdmissionConstruction(t *testing.T) {
	if !functionCallsIdentifier(t, "deps.go", "newDeps", "newAdmissionDependencies") {
		t.Fatal("newDeps must delegate process admission construction to newAdmissionDependencies")
	}
}

func TestNewDepsContainsNoPackageLevelFeatureConstructors(t *testing.T) {
	calls := functionSelectorCallsWithPrefix(t, "deps.go", "newDeps", "New")
	if len(calls) != 0 {
		t.Fatalf("newDeps constructs feature packages directly: %v", calls)
	}
}

func TestDependencyConstructorsNeverCloseTheSharedPoolDirectly(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(currentFile), "deps*.go"))
	if err != nil {
		t.Fatalf("list dependency sources: %v", err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Base(path), err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Close" {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if ok && receiver.Name == "pool" {
				t.Errorf("%s closes the shared pool directly; register cleanup with the root composer", filepath.Base(path))
			}
			return true
		})
	}
}

func functionCallsIdentifier(t *testing.T, filename, functionName, callee string) bool {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	path := filepath.Join(filepath.Dir(currentFile), filename)
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != functionName || function.Body == nil {
			continue
		}
		found := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if ok && identifier.Name == callee {
				found = true
				return false
			}
			return true
		})
		return found
	}
	t.Fatalf("function %s not found in %s", functionName, filename)
	return false
}

func functionSelectorCallsWithPrefix(
	t *testing.T,
	filename string,
	functionName string,
	prefix string,
) []string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	path := filepath.Join(filepath.Dir(currentFile), filename)
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != functionName || function.Body == nil {
			continue
		}
		var calls []string
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(selector.Sel.Name, prefix) {
				return true
			}
			packageName, ok := selector.X.(*ast.Ident)
			if ok {
				calls = append(calls, packageName.Name+"."+selector.Sel.Name)
			}
			return true
		})
		return calls
	}
	t.Fatalf("function %s not found in %s", functionName, filename)
	return nil
}
