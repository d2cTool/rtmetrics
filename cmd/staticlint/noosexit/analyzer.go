// Package noosexit запрещает прямой вызов os.Exit в функции main пакета main.
package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer сообщает о прямом os.Exit в func main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "forbid a direct os.Exit call in the main function of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok || sel.Sel.Name != "Exit" {
					return true
				}
				obj, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
				if !ok || obj.Imported().Path() != "os" {
					return true
				}
				pass.Reportf(call.Pos(), "direct os.Exit call is forbidden in main")
				return true
			})
		}
	}
	return nil, nil
}

func generated(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}
