package craftsky

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestIT008ProductionExternalAuthoringUsesGeneratedUnionTypes(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"../../api/post_create.go",
		"../../scheduledposts/publication_processor.go",
	} {
		file := parseGoFile(t, path)
		if countCalls(file, "postrecord.ExternalEmbed") == 0 {
			t.Fatalf("%s does not use the shared generated external authoring boundary", path)
		}
	}
	helper := parseGoFile(t, "../../postrecord/external.go")
	for _, generatedType := range []string{
		"craftsky.FeedPost_Embed",
		"appbsky.EmbedExternal",
		"appbsky.EmbedExternal_External",
	} {
		if countCompositeLiterals(helper, generatedType) == 0 {
			t.Fatalf("generated authoring helper does not construct %s", generatedType)
		}
	}
}

func parseGoFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

func countCalls(file *ast.File, name string) int {
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && goExpressionName(call.Fun) == name {
			count++
		}
		return true
	})
	return count
}

func countCompositeLiterals(file *ast.File, name string) int {
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if ok && goExpressionName(literal.Type) == name {
			count++
		}
		return true
	})
	return count
}

func goExpressionName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		return goExpressionName(expression.X) + "." + expression.Sel.Name
	case *ast.StarExpr:
		return goExpressionName(expression.X)
	default:
		return ""
	}
}
