//go:build ignore

// Command errorfamily_scanner checks Go source files for stdlib error
// constructors (errors.New, fmt.Errorf, errors.Join) that should use
// go-error-family constructors instead.
//
// Unlike ripgrep-based checks, this uses the Go AST via go/parser, which
// inherently ignores ALL comment types (//, /* */, inline, multi-line).
// Zero false positives from documentation or migration notes.
//
// Usage: go run scripts/errorfamily_scanner.go <dir> [<dir> ...]
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var banned = map[string]bool{
	"errors.New":   true,
	"fmt.Errorf":   true,
	"errors.Join":  true,
}

type violation struct {
	file string
	line int
	fn   string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: errorfamily_scanner <dir> [<dir> ...]")
		os.Exit(2)
	}

	var all []violation
	for _, dir := range os.Args[1:] {
		all = append(all, checkDir(dir)...)
	}

	if len(all) == 0 {
		return
	}

	for _, v := range all {
		fmt.Fprintf(os.Stderr, "FAIL: %s:%d: %s( — stdlib error constructor\n", v.file, v.line, v.fn)
	}
	os.Exit(1)
}

func checkDir(dir string) []violation {
	var violations []violation
	fset := token.NewFileSet()
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "examples" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_templ.go") {
			return nil
		}

		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil
		}

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			fn := funcName(call.Fun)
			if !banned[fn] {
				return true
			}

			pos := fset.Position(call.Pos())
			violations = append(violations, violation{
				file: pos.Filename,
				line: pos.Line,
				fn:   fn,
			})
			return true
		})
		return nil
	})
	return violations
}

func funcName(expr ast.Expr) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return pkg.Name + "." + sel.Sel.Name
}
