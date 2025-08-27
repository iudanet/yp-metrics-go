// Package exitcheck implements a static analyzer that prevents direct usage of os.Exit in main function.
package exitcheck

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the analyzer that checks for direct os.Exit usage in main function of main package.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check for direct os.Exit calls in main function of main package",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Filter for function declarations only
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl := node.(*ast.FuncDecl)

		// Check if this is the main function in the main package
		if funcDecl.Name.Name == "main" && pass.Pkg.Name() == "main" {
			// Skip checking build cache files
			file := pass.Fset.File(node.Pos())
			if file != nil && isBuildCacheFile(file.Name()) {
				return
			}

			// Inspect the function body for os.Exit calls
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				// Check for call expressions
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Check if the call is to os.Exit
				selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				xIdent, ok := selectorExpr.X.(*ast.Ident)
				if !ok {
					return true
				}

				if xIdent.Name == "os" && selectorExpr.Sel.Name == "Exit" {
					pass.Reportf(callExpr.Pos(), "direct call to os.Exit is not allowed in main function")
				}

				return true
			})
		}

	})

	return nil, nil
}

// isBuildCacheFile checks if the file is from the Go build cache
func isBuildCacheFile(filename string) bool {
	return strings.Contains(filename, "/.cache/go-build/") ||
		strings.Contains(filename, "\\.cache\\go-build\\")
}
