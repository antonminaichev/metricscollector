// Package noexitanalyzer реализует собственный статический анализатор,
// запрещающий использование os.Exit в функции main пакета main.
//
// Это правило помогает повысить тестируемость и корректность завершения программы.
// Вместо прямого вызова os.Exit рекомендуется использовать возврат ошибки из main()
// или log.Fatal, если это необходимо.
//
// Анализатор проверяет, что:
//
//   - анализируется именно главный пакет (path == "main");
//   - в функции main не вызывается os.Exit.
//
// Пример нарушающего кода:
//
//	package main
//
//	import "os"
//
//	func main() {
//	    os.Exit(1) // ← будет ошибка: os.Exit in main function is not allowed
//	}
//
// Пример корректного:
//
//	func main() {
//	    err := run()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	}
package noexitanalyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "Запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Path() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if pass.Pkg.Name() != "main" {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				pkgIdent, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}

				if selector.Sel.Name == "Exit" && pkgIdent.Name == "os" {
					obj := pass.TypesInfo.Uses[pkgIdent]
					if pkgName, ok := obj.(*types.PkgName); ok && pkgName.Imported().Path() == "os" {
						pass.Reportf(call.Lparen, "os.Exit in main function is not allowed")
					}
				}

				return true
			})
		}
	}
	return nil, nil
}
