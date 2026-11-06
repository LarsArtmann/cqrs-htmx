//go:build ignore

// Command reexport_scanner inventories cross-package alias declarations in the
// workspace so the re-export policy has a machine-readable baseline.
//
// An "alias" here is a top-level declaration whose value is a package
// selector: a type alias (`type X = pkg.Y`), a var (`var X = pkg.Y`) or a
// const (`const X = pkg.Y`). Functions that merely delegate (constructor
// wrappers) are NOT aliases and are not counted.
//
// Because an unqualified identifier can only refer to the declaring package,
// any selector-valued declaration is by definition a cross-package alias.
//
// Usage:
//
//	go run scripts/checks/reexport_scanner.go            # TSV to stdout
//	go run scripts/checks/reexport_scanner.go --files    # per-file summary
//	go run scripts/checks/reexport_scanner.go --dir DIR  # scan one module tree
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// aliasDecl is one top-level selector-valued declaration.
type aliasDecl struct {
	Module      string
	File        string
	Name        string
	Kind        string
	TargetPkg   string
	Deprecated  bool
	NamesTarget bool
	Doc         string
}

// aliasFile groups the aliases of one file.
type aliasFile struct {
	Module    string
	Path      string
	Aliases   []aliasDecl
	TopLevel  int
	PureAlias bool
}

const (
	kindType  = "type"
	kindVar   = "var"
	kindConst = "const"
)

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"testdata":     true,
	"vendor":       true,
	"archived":     true,
}

func main() {
	var (
		dirFlag  = flag.String("dir", ".", "module tree to scan")
		filesOut = flag.Bool("files", false, "emit per-file summary instead of per-declaration rows")
	)
	flag.Parse()

	root, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reexport_scanner: resolve root: %v\n", err)
		os.Exit(2)
	}

	files, err := scanTree(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reexport_scanner: %v\n", err)
		os.Exit(2)
	}

	if *filesOut {
		writeFileSummary(root, files)
		return
	}

	writeDeclRows(root, files)
}

// scanTree collects every alias file under root, grouped by owning module.
func scanTree(root string) ([]aliasFile, error) {
	modules, err := moduleDirs(root)
	if err != nil {
		return nil, err
	}

	var out []aliasFile

	for _, modDir := range modules {
		modName := relPath(root, modDir)

		err := filepath.WalkDir(modDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				if skipDirs[entry.Name()] {
					return filepath.SkipDir
				}

				if path != modDir && isModuleDir(path) {
					return filepath.SkipDir
				}

				return nil
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}

			if strings.HasSuffix(name, ".gen.go") || strings.HasSuffix(name, "_templ.go") {
				return nil
			}

			file, parseErr := parseAliasFile(modName, path)
			if parseErr != nil {
				return parseErr
			}

			if file != nil {
				out = append(out, *file)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })

	return out, nil
}

// moduleDirs returns every directory holding a go.mod, root itself first.
func moduleDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() {
			if entry.Name() == "go.mod" {
				dirs = append(dirs, filepath.Dir(path))
			}

			return nil
		}

		if skipDirs[entry.Name()] {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(dirs)

	return dirs, nil
}

func isModuleDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))

	return err == nil
}

// parseAliasFile returns nil when the file declares no alias at all.
func parseAliasFile(module, path string) (*aliasFile, error) {
	fset := token.NewFileSet()

	parsed, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reexport_scanner: parse %s: %v\n", path, err)
		return nil, err
	}

	file := &aliasFile{Module: module, Path: path}
	imports := importNames(parsed)

	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		file.TopLevel += len(gen.Specs)
		file.Aliases = append(file.Aliases, genAliases(module, path, imports, gen)...)
	}

	if len(file.Aliases) == 0 {
		return nil, nil
	}

	file.PureAlias = len(file.Aliases) == file.TopLevel

	return file, nil
}

func genAliases(module, path string, imports map[string]string, gen *ast.GenDecl) []aliasDecl {
	var out []aliasDecl

	switch gen.Tok {
	case token.TYPE:
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !typeSpec.Assign.IsValid() {
				continue
			}

			if decl, ok := makeAlias(
				module,
				path,
				imports,
				kindType,
				typeSpec.Name.Name,
				typeSpec.Type,
				gen.Doc,
				typeSpec.Doc,
				typeSpec.Comment,
			); ok {
				out = append(out, decl)
			}
		}
	case token.VAR, token.CONST:
		kind := kindVar
		if gen.Tok == token.CONST {
			kind = kindConst
		}

		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, value := range valueSpec.Values {
				name := "?"
				if i < len(valueSpec.Names) {
					name = valueSpec.Names[i].Name
				}

				if decl, ok := makeAlias(
					module,
					path,
					imports,
					kind,
					name,
					value,
					gen.Doc,
					valueSpec.Doc,
					valueSpec.Comment,
				); ok {
					out = append(out, decl)
				}
			}
		}
	}

	return out
}

func makeAlias(
	module, path string,
	imports map[string]string,
	kind, name string,
	expr ast.Expr,
	genDoc, specDoc, lineDoc *ast.CommentGroup,
) (aliasDecl, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return aliasDecl{}, false
	}

	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return aliasDecl{}, false
	}

	doc := joinDocs(genDoc, specDoc, lineDoc)

	return aliasDecl{
		Module:      module,
		File:        path,
		Name:        name,
		Kind:        kind,
		TargetPkg:   pkgIdent.Name,
		Deprecated:  hasDeprecatedParagraph(doc),
		NamesTarget: namesTarget(doc, pkgIdent.Name, imports[pkgIdent.Name]),
		Doc:         doc,
	}, true
}

func joinDocs(groups ...*ast.CommentGroup) string {
	var parts []string

	for _, group := range groups {
		if group == nil {
			continue
		}

		parts = append(parts, group.Text())
	}

	return strings.Join(parts, "\n")
}

// hasDeprecatedParagraph reports whether the doc carries a "Deprecated:"
// paragraph, which is the only form go doc and staticcheck recognise.
func hasDeprecatedParagraph(doc string) bool {
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Deprecated:") {
			return true
		}
	}

	return false
}

// importNames maps the identifier a file uses for an import (explicit alias or
// the path's last element) to that import's full path.
func importNames(file *ast.File) map[string]string {
	names := make(map[string]string, len(file.Imports))

	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}

		local := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			local = spec.Name.Name
		}

		names[local] = path
	}

	return names
}

// namesTarget reports whether the deprecation text points the reader at the
// package that actually owns the symbol. Every spelling a consumer might see is
// accepted: the qualified form ("identitymodel.User"), the import path
// ("github.com/larsartmann/cqrs-htmx/identity-model/v4") and the bare path
// segment ("import .../identity-model/v4 directly").
func namesTarget(doc, pkg, importPath string) bool {
	if strings.Contains(doc, pkg+".") || strings.Contains(doc, importPath) {
		return true
	}

	if importPath == "" {
		return false
	}

	segment := importPath[strings.LastIndex(importPath, "/")+1:]
	if strings.Contains(doc, segment) {
		return true
	}

	return false
}

func writeDeclRows(root string, files []aliasFile) {
	fmt.Println("# module\tfile\tname\tkind\ttarget_pkg\tdeprecated\tnames_target\tpure_alias_file")

	for _, file := range files {
		for _, alias := range file.Aliases {
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				clean(file.Module),
				clean(relPath(root, file.Path)),
				clean(alias.Name),
				alias.Kind,
				clean(alias.TargetPkg),
				strconv.FormatBool(alias.Deprecated),
				strconv.FormatBool(alias.NamesTarget),
				strconv.FormatBool(file.PureAlias),
			)
		}
	}
}

func writeFileSummary(root string, files []aliasFile) {
	fmt.Println("# module\tfile\taliases\ttop_level\tpure_alias_file\tdeprecated\tall_deprecated")

	for _, file := range files {
		deprecated := 0

		for _, alias := range file.Aliases {
			if alias.Deprecated {
				deprecated++
			}
		}

		fmt.Printf("%s\t%s\t%d\t%d\t%s\t%d\t%s\n",
			clean(file.Module),
			clean(relPath(root, file.Path)),
			len(file.Aliases),
			file.TopLevel,
			strconv.FormatBool(file.PureAlias),
			deprecated,
			strconv.FormatBool(deprecated == len(file.Aliases)),
		)
	}
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}

	return filepath.ToSlash(rel)
}

func clean(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ").Replace(s)
}
