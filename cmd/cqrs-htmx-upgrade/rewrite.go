package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// fileReport is the outcome for one scanned file.
type fileReport struct {
	path    string
	edits   []plannedEdit
	manual  []manualFinding
	applied string // the rewritten source (only meaningful when edits exist)
}

type plannedEdit struct {
	Rule string
	Old  string
	New  string
}

type manualFinding struct {
	Rule string
	Note string
}

// rawBroadcasterRenames is rule R3: constructor renames inside the SAME
// package (import-neutral).
//
//nolint:gochecknoglobals // fixed lookup table
var rawBroadcasterRenames = map[string]string{
	"NewBroadcasterFromRaw": "NewBroadcasterFromHub",
}

// sourcePackages are the packages whose deprecated re-exports the codemod
// migrates; imports of these are drop-candidates once unreferenced.
//
//nolint:gochecknoglobals // fixed lookup table
var sourcePackages = map[string]bool{
	rootImportPath:     true,
	usermgmtImportPath: true,
}

// run walks the given paths, plans edits per file, and (with apply) writes
// them. Returns the number of files scanned and one report per file.
func run(paths []string, table rulesTable, apply bool) (int, []fileReport, error) {
	goFiles, err := collectGoFiles(paths)
	if err != nil {
		return 0, nil, err
	}

	reports := make([]fileReport, 0, len(goFiles))

	for _, path := range goFiles {
		rep, err := planFile(path, table)
		if err != nil {
			return len(goFiles), reports, err
		}

		if apply && len(rep.edits) > 0 {
			// Never write unparseable output: a rewrite bug must fail loudly
			// here, not corrupt the consumer tree on disk.
			if _, err := parser.ParseFile(
				token.NewFileSet(), path, []byte(rep.applied), parser.SkipObjectResolution,
			); err != nil {
				return len(
						goFiles,
					), reports, errorfamily.Wrapf(err, errorfamily.Corruption,
					"cqrshtmx.upgrade.unwritable_rewrite", "refusing to write %s: rewritten source does not parse", path)
			}

			info, err := os.Stat(path)
			if err != nil {
				return len(goFiles), reports, errorfamily.Wrapf(err, errorfamily.Infrastructure,
					"cqrshtmx.upgrade.stat_failed", "stat %s", path)
			}

			if err := os.WriteFile(path, []byte(rep.applied), info.Mode().Perm()); err != nil {
				return len(goFiles), reports, errorfamily.Wrapf(err, errorfamily.Infrastructure,
					"cqrshtmx.upgrade.write_failed", "write %s", path)
			}
		}

		reports = append(reports, rep)
	}

	return len(goFiles), reports, nil
}

// skipDirs are never scanned.
//
//nolint:gochecknoglobals // fixed lookup table
var skipDirs = map[string]bool{"vendor": true, "testdata": true, ".git": true}

// collectGoFiles expands dirs (recursively) into .go file paths, skipping
// vendor/, testdata/, and .git/ trees.
func collectGoFiles(paths []string) ([]string, error) {
	var out []string

	for _, root := range paths {
		if err := collectRoot(root, &out); err != nil {
			return nil, err
		}
	}

	sort.Strings(out)

	return out, nil
}

func collectRoot(root string, out *[]string) error {
	info, err := os.Stat(root)
	if err != nil {
		//nolint:wrapcheck // CLI boundary: errors terminate the run, not API
		return err
	}

	if !info.IsDir() {
		if strings.HasSuffix(root, ".go") {
			*out = append(*out, root)
		}

		return nil
	}

	//nolint:wrapcheck // CLI boundary
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			//nolint:wrapcheck // CLI boundary
			return err
		}

		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			*out = append(*out, path)
		}

		return nil
	})
}

// textEdit is one byte-range replacement against the ORIGINAL source (all
// edits are computed from the pristine AST and applied in one pass,
// back-to-front, so offsets never shift mid-apply).
type textEdit struct {
	offset, end int
	newText     string
}

// planFile parses one file and computes its rewritten form (without writing).
func planFile(path string, table rulesTable) (fileReport, error) {
	rep := fileReport{path: path, edits: nil, manual: nil, applied: ""}

	src, err := os.ReadFile(path)
	if err != nil {
		return rep, err //nolint:wrapcheck // CLI boundary: errors terminate the run, not API
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return rep, errorfamily.Wrapf(err, errorfamily.Rejection,
			"cqrshtmx.upgrade.parse_failed", "parse %s", path)
	}

	qualToPath := importQualifierMap(file)

	pathToQual := map[string]string{}
	for qual, p := range qualToPath {
		pathToQual[p] = qual
	}

	// defaultQual is the qualifier a rewrite introduces when the file does
	// not import the target package yet.
	targetQual := map[string]string{
		httputilImportPath: "httputil",
		gosseImportPath:    "sse",
		identityImportPath: "identitymodel",
	}

	remainingRefs := map[string]int{}
	needImport := map[string]bool{}

	edits := make([]textEdit, 0, len(file.Decls))

	ctx := &planCtx{
		fset:          fset,
		table:         table,
		qualToPath:    qualToPath,
		pathToQual:    pathToQual,
		targetQual:    targetQual,
		remainingRefs: remainingRefs,
		needImport:    needImport,
		edits:         edits,
		rep:           &rep,
	}

	ast.Inspect(file, ctx.visit)
	edits = ctx.edits

	if len(edits) == 0 && len(rep.manual) == 0 {
		return rep, nil
	}

	edits = append(edits, importEdits(fset, file, string(src), qualToPath, remainingRefs, needImport)...)

	sort.Slice(edits, func(i, j int) bool { return edits[i].offset > edits[j].offset })

	out := string(src)

	for _, e := range edits {
		if e.offset < 0 || e.end > len(out) || e.offset > e.end {
			return rep, errorfamily.Newf(errorfamily.Corruption,
				"cqrshtmx.upgrade.bad_edit_range",
				"internal: bad edit range [%d,%d) for %s (len %d)",
				e.offset,
				e.end,
				path,
				len(out),
			).WithContextAny("qual_to_path", qualToPath)
		}

		out = out[:e.offset] + e.newText + out[e.end:]
	}

	rep.applied = out

	return rep, nil
}

// reportRawCall adds an R3 manual finding for `.Raw()` calls — rewriting a
// method by name alone is type-ambiguous, so they are surfaced for manual
// Hub() migration. Package-qualified calls (`pkg.Raw(...)`) are skipped —
// they are function calls, not the broadcaster method, and the R1/R2
// tables own package selectors anyway.
func reportRawCall(fset *token.FileSet, n ast.Node, qualToPath map[string]string, rep *fileReport) {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return
	}

	method, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || method.Sel.Name != "Raw" {
		return
	}

	if ident, isIdent := method.X.(*ast.Ident); isIdent {
		if _, isPkg := qualToPath[ident.Name]; isPkg {
			return
		}
	}

	rep.manual = append(rep.manual, manualFinding{
		Rule: "R3",
		Note: fmt.Sprintf(
			"%s: .Raw() — migrate manually to .Hub() (type-ambiguous to rewrite)",
			fset.Position(call.Pos()),
		),
	})
}

// importEdits computes drop/insert edits for the import block:
//   - a source-package import whose qualifier lost every reference is
//     dropped line-wise (whole ImportSpec line, comments included);
//   - target imports that are newly referenced but not present are appended
//     inside the first import block (or a fresh block under the package
//     clause when the file imports nothing).
//
// importEdits computes drop/insert edits for the import block:
//   - a source-package import whose qualifier lost every reference is
//     dropped line-wise (whole ImportSpec line, comments included);
//   - target imports that are newly referenced but not present are inserted
//     (fused into a dropped line when they would overlap — see fuseInserts).
func importEdits(
	fset *token.FileSet,
	file *ast.File,
	src string,
	qualToPath map[string]string,
	remainingRefs map[string]int,
	needImport map[string]bool,
) []textEdit {
	inserts := missingImports(file, qualToPath, needImport)
	drops := dropImportSpecs(fset, file, src, qualToPath, remainingRefs)

	if len(inserts) > 0 {
		fuseInserts(fset, file, src, &drops, inserts)
	}

	return drops
}

// missingImports returns the import paths that needImport asks for but the
// file does not already provide (under any name).
func missingImports(file *ast.File, qualToPath map[string]string, needImport map[string]bool) []textEdit {
	var inserts []textEdit

	for path := range needImport {
		_, byQual := qualToPath[defaultQualFor(path)]

		if byQual || importedPath(file, path) {
			continue
		}

		inserts = append(inserts, textEdit{-1, -1, path})
	}

	return inserts
}

// dropImportSpecs drops source-package imports whose qualifier lost every
// remaining reference: line-wise for parenthesized blocks, whole-declaration
// for single-line imports.
func dropImportSpecs(
	fset *token.FileSet,
	file *ast.File,
	src string,
	qualToPath map[string]string,
	remainingRefs map[string]int,
) []textEdit {
	var drops []textEdit

	lineEnd := func(offset int) int {
		if idx := strings.IndexByte(src[offset:], '\n'); idx >= 0 {
			return offset + idx
		}

		return len(src)
	}

	lineStart := func(offset int) int {
		if idx := strings.LastIndexByte(src[:offset], '\n'); idx >= 0 {
			return idx + 1
		}

		return offset
	}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}

		for _, spec := range gen.Specs {
			imp, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}

			if drop, ok := dropEditForSpec(fset, src, lineEnd, lineStart, gen, imp, qualToPath, remainingRefs); ok {
				drops = append(drops, drop)
			}
		}
	}

	return drops
}

// dropEditForSpec computes the line-wise drop edit for one import spec.
// Returns ok=false when the spec must stay (still referenced, not a source
// package, or a blank/dot import — `_` registers side effects the rewrite
// must preserve, and `.` binds every exported name).
func dropEditForSpec(
	fset *token.FileSet,
	src string,
	lineEnd func(int) int,
	lineStart func(int) int,
	gen *ast.GenDecl,
	imp *ast.ImportSpec,
	qualToPath map[string]string,
	remainingRefs map[string]int,
) (textEdit, bool) {
	local := defaultPackageName(strings.Trim(imp.Path.Value, `"`))
	if imp.Name != nil {
		local = imp.Name.Name
	}

	if imp.Name != nil && (imp.Name.Name == "_" || imp.Name.Name == ".") {
		return textEdit{offset: 0, end: 0, newText: ""}, false
	}

	if remainingRefs[local] != 0 || !sourcePackages[qualToPath[local]] {
		return textEdit{offset: 0, end: 0, newText: ""}, false
	}

	off := fset.Position(imp.Pos()).Offset
	end := lineEnd(fset.Position(imp.End()).Offset)

	// Single-spec, unparenthesized decls carry the `import` keyword
	// on the same line: drop the whole declaration.
	if !gen.Lparen.IsValid() {
		off = lineStart(fset.Position(gen.Pos()).Offset)
	}

	if next := end + 1; next <= len(src) {
		end = next // consume the newline too
	}

	return textEdit{off, end, ""}, true
}

func fuseInserts(fset *token.FileSet, file *ast.File, src string, drops *[]textEdit, inserts []textEdit) {
	lineEnd := func(offset int) int {
		if idx := strings.IndexByte(src[offset:], '\n'); idx >= 0 {
			return offset + idx
		}

		return len(src)
	}

	lineStart := func(offset int) int {
		if idx := strings.LastIndexByte(src[:offset], '\n'); idx >= 0 {
			return idx + 1
		}

		return 0
	}

	var lines strings.Builder

	for _, ins := range inserts {
		lines.WriteString("\n\t")
		lines.WriteString(`"`)
		lines.WriteString(ins.newText)
		lines.WriteString(`"`)
	}

	if anchor, ok := lastImportLine(fset, file); ok {
		for i := range *drops {
			d := &(*drops)[i]

			if anchor >= d.offset && anchor <= d.end {
				// The anchor line is dropped: fuse — replace instead of
				// insert+delete.
				d.newText = fusedImportReplacement(src, *d, lines.String(), lineStart)

				return
			}
		}

		at := lineEnd(anchor) // before the newline of the last import line
		*drops = append(*drops, textEdit{at, at, lines.String()})

		return
	}

	// No import block: open one right after the package clause.
	insertAt := lineEnd(fset.Position(file.Package).Offset)
	if file.Name != nil {
		insertAt = lineEnd(fset.Position(file.Name.End()).Offset)
	}

	var block strings.Builder
	block.WriteString("\n\nimport (\n")

	for _, ins := range inserts {
		block.WriteString("\t")
		block.WriteString(`"`)
		block.WriteString(ins.newText)
		block.WriteString(`"`)
		block.WriteString("\n")
	}

	block.WriteString(")")

	*drops = append(*drops, textEdit{insertAt, insertAt, block.String()})
}

// fusedImportReplacement builds the text a fused drop is replaced with. The
// replacement must account for what the drop range already preserves: a
// parenthesized spec's range starts AFTER the line's leading indent (keep it,
// add none); a whole-declaration drop (single-line, unparenthesized
// `import "path"`) removes the `import` keyword itself, so the replacement
// has to open a fresh import block — emitting bare spec lines there produced
// unparseable output.
func fusedImportReplacement(src string, d textEdit, lines string, lineStart func(int) int) string {
	body := strings.TrimLeft(strings.TrimPrefix(lines, "\n"), "\t")

	if strings.HasPrefix(strings.TrimLeft(src[d.offset:d.end], " \t"), "import") {
		return "import (\n\t" + body + "\n)\n"
	}

	indent := ""
	if lineStart(d.offset) >= d.offset {
		indent = "\t"
	}

	return indent + body + "\n"
}

func defaultQualFor(importPath string) string {
	if qual, ok := wellKnownQualifier(importPath); ok {
		return qual
	}

	return filepath.Base(importPath)
}

// importedPath reports whether the file already imports path under ANY name.
func importedPath(file *ast.File, path string) bool {
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == path {
			return true
		}
	}

	return false
}

// lastImportLine returns the offset of the LAST import spec's end within the
// first import declaration.
func lastImportLine(fset *token.FileSet, file *ast.File) (int, bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}

		last := len(gen.Specs) - 1
		if last < 0 {
			return 0, false
		}

		imp, ok := gen.Specs[last].(*ast.ImportSpec)
		if !ok {
			return 0, false
		}

		return fset.Position(imp.End()).Offset, true
	}

	return 0, false
}

// requalify rewrites one deprecated package-qualified selector to its target
// package (import bookkeeping included) and records the planned edit.
func requalify(
	fset *token.FileSet,
	ident *ast.Ident,
	sel *ast.SelectorExpr,
	sym mappedSymbol,
	pathToQual map[string]string,
	targetQual map[string]string,
	needImport map[string]bool,
	edits *[]textEdit,
	rep *fileReport,
) {
	newQual := pathToQual[sym.targetImportPath]
	if newQual == "" {
		newQual = targetQual[sym.targetImportPath]
		needImport[sym.targetImportPath] = true
	}

	off := fset.Position(ident.Pos()).Offset
	end := fset.Position(ident.End()).Offset
	*edits = append(*edits, textEdit{off, end, newQual})

	if sym.targetName != sel.Sel.Name {
		soff := fset.Position(sel.Sel.Pos()).Offset
		send := fset.Position(sel.Sel.End()).Offset
		*edits = append(*edits, textEdit{soff, send, sym.targetName})
	}

	rep.edits = append(rep.edits, plannedEdit{
		Rule: sym.rule,
		Old:  ident.Name + "." + sel.Sel.Name,
		New:  newQual + "." + sym.targetName,
	})
}

// renameRawConstructor rewrites one NewBroadcasterFromRaw -> FromHub call
// (rule R3; the selector name alone changes, the import stays).
func renameRawConstructor(
	fset *token.FileSet,
	ident *ast.Ident,
	sel *ast.SelectorExpr,
	renamed string,
	edits *[]textEdit,
	rep *fileReport,
) {
	off := fset.Position(sel.Sel.Pos()).Offset
	end := fset.Position(sel.Sel.End()).Offset
	*edits = append(*edits, textEdit{off, end, renamed})

	rep.edits = append(rep.edits, plannedEdit{
		Rule: "R3",
		Old:  ident.Name + "." + sel.Sel.Name,
		New:  ident.Name + "." + renamed,
	})
}

// planCtx carries one file's planning state through the AST walk.
type planCtx struct {
	fset          *token.FileSet
	table         rulesTable
	qualToPath    map[string]string
	pathToQual    map[string]string
	targetQual    map[string]string
	remainingRefs map[string]int
	needImport    map[string]bool
	edits         []textEdit
	rep           *fileReport
}

// visit is the AST walk: reports .Raw() calls, renames R3 constructors, and
// requalifies deprecated re-export selectors (R1/R2/R9/R10).
func (p *planCtx) visit(n ast.Node) bool {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		reportRawCall(p.fset, n, p.qualToPath, p.rep)

		return true
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return true
	}

	srcPath, known := p.qualToPath[ident.Name]
	if !known {
		return true
	}

	p.remainingRefs[ident.Name]++

	// Rule R3: constructor renames within the same package.
	if renamed, isRaw := rawBroadcasterRenames[sel.Sel.Name]; isRaw && sourcePackages[srcPath] {
		p.remainingRefs[ident.Name]--
		renameRawConstructor(p.fset, ident, sel, renamed, &p.edits, p.rep)

		return true
	}

	// R1/R2/R9/R10: deprecated re-export requalification.
	sym, mapped := p.table[srcPath][sel.Sel.Name]
	if !mapped {
		return true
	}

	p.remainingRefs[ident.Name]--
	requalify(p.fset, ident, sel, sym, p.pathToQual, p.targetQual, p.needImport, &p.edits, p.rep)

	return true
}
