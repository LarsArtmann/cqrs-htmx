package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strings"
)

// rulesTable maps a SOURCE package import path to its deprecated-symbol table
// (symbol -> target). The consumer-file qualifier is resolved per file, so
// aliased consumer imports (`ch "…/v4"`) rewrite correctly too.
type rulesTable map[string]symbolMap

// symbolMap is oldSymbol -> target.
type symbolMap map[string]mappedSymbol

type mappedSymbol struct {
	targetImportPath string
	targetName       string // final selector name (usually identical)
	rule             string // R1 | R2 | R9 | R10
}

const (
	rootImportPath      = "github.com/larsartmann/cqrs-htmx/v4"
	usermgmtImportPath  = "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	datastarImportPath  = "github.com/larsartmann/cqrs-htmx/datastar/v4"
	httputilImportPath  = "github.com/larsartmann/httputil"
	gosseImportPath     = "github.com/larsartmann/go-sse"
	identityImportPath  = "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// root httputil re-export files (class 1) and go-sse shims (class 1).
var (
	rootHTTputilFiles = []string{"csrf_reexport.go", "ratelimit_reexport.go", "server_timing_reexport.go"}
	rootGoSSEFiles    = []string{"sse_event.go", "sse_store.go", "event_store_sse.go"}
)

// usermgmt identity-model alias layer (class 3): the named files plus every
// es_*.go event/command/state file.
func usermgmtAliasFiles(dir string) []string {
	named := []string{
		"authz_types.go", "credential.go", "crypto.go", "email.go", "errors.go",
		"external_account.go", "id.go", "upcaster.go", "user.go",
	}

	es, err := filepath.Glob(filepath.Join(dir, "es_*.go"))
	if err != nil {
		return named
	}

	for _, f := range es {
		named = append(named, filepath.Base(f))
	}

	return named
}

// deriveMappings builds the R1/R2/R9/R10 tables from the alias declarations
// of the cqrs-htmx version present in the consumer's module graph, so the
// codemod can never drift from the installed library.
func deriveMappings() (rulesTable, error) {
	rootDir, err := moduleDir(rootImportPath)
	if err != nil {
		return nil, err
	}

	usermgmtDir, err := moduleDir(usermgmtImportPath)
	if err != nil {
		return nil, err
	}

	httputil, err := deriveFromFiles(rootDir, rootHTTputilFiles, httputilImportPath, "R1")
	if err != nil {
		return nil, fmt.Errorf("R1 (httputil re-exports): %w", err)
	}

	gosse, err := deriveFromFiles(rootDir, rootGoSSEFiles, gosseImportPath, "R2")
	if err != nil {
		return nil, fmt.Errorf("R2 (go-sse re-exports): %w", err)
	}

	identity, err := deriveFromFiles(usermgmtDir, usermgmtAliasFiles(usermgmtDir), identityImportPath, "R9")
	if err != nil {
		return nil, fmt.Errorf("R9/R10 (usermgmt aliases): %w", err)
	}

	root := symbolMap{}
	for sym, m := range httputil {
		root[sym] = m
	}

	for sym, m := range gosse {
		if _, clash := root[sym]; clash {
			return nil, fmt.Errorf("symbol %s appears in both the httputil and go-sse re-export sets", sym)
		}

		root[sym] = m
	}

	return rulesTable{rootImportPath: root, usermgmtImportPath: identity}, nil
}

// deriveFromFiles parses alias files and collects every `X = pkg.Y` alias
// (type or var) whose target resolves to wantTargetPath. Types report as the
// given rule; value re-exports report as the same rule with a "0" suffix
// swapped to "10" when the rule is R9 (type R9 / value R10 granularity).
func deriveFromFiles(dir string, files []string, wantTargetPath, rule string) (symbolMap, error) {
	out := symbolMap{}

	fset := token.NewFileSet()

	for _, name := range files {
		path := filepath.Join(dir, name)

		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		qualToPath := importQualifierMap(file)

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok && true {
				continue
			}

			if gen.Tok != token.TYPE && gen.Tok != token.VAR {
				continue
			}

			for _, spec := range gen.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec: // X = pkg.Y (transparent alias only)
					if !s.Assign.IsValid() {
						continue
					}

					collectAlias(out, s.Name.Name, s.Type, qualToPath, wantTargetPath, ruleForTok(rule, token.TYPE))

				case *ast.ValueSpec: // X = pkg.Y (var/const re-export)
					if len(s.Names) != 1 || len(s.Values) != 1 {
						continue
					}

					collectAlias(out, s.Names[0].Name, s.Values[0], qualToPath, wantTargetPath, ruleForTok(rule, token.VAR))
				}
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no %s alias declarations found in %s (files: %v)", wantTargetPath, dir, files)
	}

	return out, nil
}

func ruleForTok(base string, tok token.Token) string {
	if base == "R9" && tok == token.VAR {
		return "R10"
	}

	return base
}

// collectAlias records sym -> target when expr is a selector onto the wanted
// target package.
func collectAlias(out symbolMap, sym string, expr ast.Expr, qualToPath map[string]string, wantTargetPath, rule string) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if qualToPath[ident.Name] != wantTargetPath {
		return
	}

	out[sym] = mappedSymbol{targetImportPath: wantTargetPath, targetName: sel.Sel.Name, rule: rule}
}

// importQualifierMap builds localQualifier -> importPath for a parsed file.
func importQualifierMap(file *ast.File) map[string]string {
	out := map[string]string{}

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		local := filepath.Base(path)
		if imp.Name != nil {
			local = imp.Name.Name
		}

		out[local] = path
	}

	return out
}

// moduleDir resolves a module's local source directory via the consumer's
// module graph (`go list -m`), so the tables match the installed version.
func moduleDir(importPath string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", importPath) //nolint:gosec // fixed argv; runs in the consumer's tree

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("locate module %s (is it in your module graph?): %w", importPath, err)
	}

	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("module %s resolved to an empty directory", importPath)
	}

	return dir, nil
}
