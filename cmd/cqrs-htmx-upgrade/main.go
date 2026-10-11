// Command cqrs-htmx-upgrade rewrites consumer code off APIs scheduled for
// removal in cqrs-htmx v5 (see docs/guides/v5-removal-inventory.md).
//
// Track A rules (mechanical-safe, applied automatically):
//
//	R1  root httputil re-exports (CSRF, rate limiting, server timing, security
//	    headers) → the github.com/larsartmann/httputil package
//	R2  root go-sse re-export shims (SSE event/store types) → github.com/larsartmann/go-sse
//	R3  raw-broadcaster constructors → the Hub vocabulary
//	    (NewBroadcasterFromRaw → NewBroadcasterFromHub); remaining `.Raw()` call
//	    sites are REPORTED for manual migration (type-ambiguous to rewrite safely)
//	R9  usermgmt deprecated type aliases → identity-model
//	R10 usermgmt deprecated var/const re-exports → identity-model
//
// The R1/R2/R9/R10 symbol tables are NOT hardcoded: they are derived at run
// time from the alias declarations of the cqrs-htmx version present in the
// consumer's module graph, so the codemod can never drift from the library.
//
// Usage:
//
//	cqrs-htmx-upgrade [flags] [paths...]   # paths default to "."
//
// Flags:
//
//	-apply    write the rewritten files (default: dry-run report only)
//	-v        print every planned edit
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	apply := flag.Bool("apply", false, "write rewritten files (default: dry-run report)")
	verbose := flag.Bool("v", false, "print every planned edit")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	mappings, err := deriveMappings()
	if err != nil {
		fatal("derive mappings: %v", err)
	}

	files, reports, err := run(paths, mappings, *apply)
	if err != nil {
		fatal("%v", err)
	}

	totalEdits := 0
	totalManual := 0

	for _, rep := range reports {
		totalEdits += len(rep.edits)
		totalManual += len(rep.manual)

		if len(rep.edits) == 0 && len(rep.manual) == 0 {
			continue
		}

		fmt.Printf("%s:\n", rep.path)

		for _, e := range rep.edits {
			fmt.Printf("  %s: %s -> %s\n", e.Rule, e.Old, e.New)
		}

		for _, m := range rep.manual {
			fmt.Printf("  %s (MANUAL): %s\n", m.Rule, m.Note)
		}
	}

	mode := "dry run (no files written; pass -apply)"
	if *apply {
		mode = "APPLY"
	}

	fmt.Printf("\n%d file(s) scanned, %d file(s) with planned edits, %d edit(s), %d manual finding(s) — %s\n",
		files, countChanged(reports), totalEdits, totalManual, mode)

	if !*apply && totalEdits > 0 {
		fmt.Println("re-run with -apply to write these changes")
	}

	if !*apply && !*verbose && totalEdits == 0 && totalManual == 0 {
		fmt.Println("nothing to upgrade")
	}
}

func countChanged(reports []fileReport) int {
	n := 0

	for _, rep := range reports {
		if len(rep.edits) > 0 {
			n++
		}
	}

	return n
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cqrs-htmx-upgrade: "+format+"\n", args...)
	os.Exit(1)
}
