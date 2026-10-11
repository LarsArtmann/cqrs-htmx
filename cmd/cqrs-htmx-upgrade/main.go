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

	if err := runCLI(paths, *apply, *verbose); err != nil {
		fatalf("%v", err)
	}
}

// runCLI is main's body: derive the rule tables, plan every file, print the
// report, optionally write.
func runCLI(paths []string, apply, verbose bool) error {
	mappings, err := deriveMappings()
	if err != nil {
		return fmt.Errorf("derive mappings: %w", err)
	}

	_, reports, err := run(paths, mappings, apply)
	if err != nil {
		return err
	}

	printReport(reports, apply, verbose)

	return nil
}

// printReport writes the per-file findings and the summary. The CLI's
// stdout IS its product — fmt.Print* is deliberate here.
func printReport(reports []fileReport, apply, verbose bool) {
	totalEdits := 0
	totalManual := 0
	changed := 0

	for _, rep := range reports {
		totalEdits += len(rep.edits)
		totalManual += len(rep.manual)

		if len(rep.edits) > 0 {
			changed++
		}

		printFileFindings(rep)
	}

	mode := "dry run (no files written; pass -apply)"
	if apply {
		mode = "APPLY"
	}

	//nolint:forbidigo // CLI report output is the tool's product
	fmt.Printf("\n%d file(s) scanned, %d file(s) with planned edits, %d edit(s), %d manual finding(s) — %s\n",
		len(reports), changed, totalEdits, totalManual, mode)

	if !apply && totalEdits > 0 {
		//nolint:forbidigo // CLI report output is the tool's product
		fmt.Println("re-run with -apply to write these changes")
	}

	if !apply && !verbose && totalEdits == 0 && totalManual == 0 {
		//nolint:forbidigo // CLI report output is the tool's product
		fmt.Println("nothing to upgrade")
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cqrs-htmx-upgrade: "+format+"\n", args...)
	os.Exit(1)
}

// printFileFindings prints one file's edits and manual findings. The CLI's
// stdout IS its product — fmt.Print* is deliberate here.
func printFileFindings(rep fileReport) {
	if len(rep.edits) == 0 && len(rep.manual) == 0 {
		return
	}

	//nolint:forbidigo // CLI report output is the tool's product
	fmt.Printf("%s:\n", rep.path)

	for _, e := range rep.edits {
		//nolint:forbidigo // CLI report output is the tool's product
		fmt.Printf("  %s: %s -> %s\n", e.Rule, e.Old, e.New)
	}

	for _, m := range rep.manual {
		//nolint:forbidigo // CLI report output is the tool's product
		fmt.Printf("  %s (MANUAL): %s\n", m.Rule, m.Note)
	}
}
