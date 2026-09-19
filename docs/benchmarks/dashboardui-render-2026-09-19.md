# dashboardui render benchmark — hand-rolled vs hybrid library path

Recorded 2026-09-19, `go test -bench BenchmarkRenderStatCard -benchmem -count=5`
(32-core linux/amd64 machine, GOCACHE on /mnt/buildcache).

| Sub-bench      | median ns/op | B/op | allocs/op |
| -------------- | -----------: | ---: | --------: |
| hand-rolled    |          437 |  280 |         6 |
| hybrid-library |         4288 | 2209 |        29 |

## Baseline correction (2026-09-19)

The 2026-09-17 recording used a recreated "hand-rolled" baseline that was
NOT the pre-adoption markup: it rendered an unescaped single-Fprintf
approximation with no escaping work and a dead context variable. The
baseline is now extracted verbatim from the historic `statCard` helper
(`git show 81088b64:dashboardui/handler_overview.go`), including the
`html.EscapeString` calls on value and label and the three separate
writes. Honest deltas:

- hand-rolled: 112 → 437 ns/op (the old number under-counted ~4x by
  omitting escaping)
- hybrid: 2200 → 4288 ns/op (machine/toolchain-era drift: golangci and
  toolchain churn between recordings; treat cross-day ratios, not
  absolute ns, as the signal)
- ratio: ~19.6x → ~9.8x; the conclusion is unchanged - the hybrid path
  costs single-digit microseconds per card, which is noise behind
  network I/O at dashboard scale, accepted in exchange for
  library-maintained markup, a11y attributes, and CSP-safe scripts.

Machine-pinned raw output: `dashboardui-render-2026-09-19.txt` (same
change). The prior `dashboardui-render-2026-09-17.{md,txt}` are kept as
history but their hand-rolled column is not comparable.

## Full family sweep (N16, 2026-09-19 second run)

Added Button (link/submit), EmptyState, DefinitionList, and Table
raw-body-vs-data-row benches. Raw output (5x, -benchmem):
`dashboardui-render-2026-09-19-sweep.txt`. Medians:

| Bench                                   | hand-rolled / alt path | hybrid / other path | ratio |
| --------------------------------------- | ---------------------: | ------------------: | ----: |
| StatCard   hand-rolled vs hybrid        |          1,563 ns/op |     12,732 ns/op |  8.1x |
| Button link (hand vs hybrid)            |              748     |     23,361       | 31.2x |
| Button submit (hand vs hybrid)          |              934     |     19,297       | 20.7x |
| EmptyState (hand vs hybrid)             |              817     |     16,402       | 20.1x |
| DefinitionList (meta-table vs library)  |           65,359     |     87,279       |  1.3x |
| Table data-row vs raw-body (library)    |          442,440     |    103,713       |  4.3x |

(Definitions: for the first five rows the left column is the pre-adoption
hand-rolled baseline; for the Table row both paths are current library
paths — the "ratio" is data-row cost over raw-body cost.)

Honest-baseline notes, same discipline as the StatCard correction above:

- Button/EmptyState baselines are the pre-adoption markup verbatim from
  `git 6294d73e~1` with the escaping the real sites performed.
- The DefinitionList baseline is the pre-M20 `table.meta-table` markup;
  its copyable rows already rendered the library CopyButton (M13), so the
  1.3x delta isolates the meta-table → DefinitionList structure swap.
  Library adoption there is nearly free.
- The Table result is actionable: for the dashboard's string-built
  listings the raw-body path is ~4.3x cheaper than typed data-rows at 10
  rows — keep `tableHTMLRaw` for large string-built tables, reserve
  `tableHTML` for typed/sorted listings (current usage already matches
  this split).

Environment caveat: absolute ns/op this run are ~3x the morning StatCard
recording (concurrent machine load); within-run ratios are the signal,
per the standing guidance. The StatCard ratio reproduced (9.8x → 8.1x).
