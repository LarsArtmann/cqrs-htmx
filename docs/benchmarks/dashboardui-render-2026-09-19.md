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
