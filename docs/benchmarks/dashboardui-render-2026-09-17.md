# dashboardui render benchmark — hand-rolled vs hybrid library path

Recorded 2026-09-17, `go test -bench BenchmarkRenderStatCard -benchmem -count=5`
(32-core linux/amd64 machine, GOCACHE on /mnt/buildcache).

| Sub-bench        | median ns/op | B/op | allocs/op |
| ---------------- | -----------: | ---: | --------: |
| hand-rolled      |          112 |  144 |         2 |
| hybrid-library   |         2200 | 2209 |        29 |

Reading: the hybrid path costs ~2µs and ~2.2KB per stat card — templ's
component protocol (context checks, class resolution, writer plumbing)
dominates the trivial markup. At dashboard scale (a page renders dozens of
components per request, typically behind network I/O) this is noise; the
program accepted it in exchange for library-maintained markup, a11y
attributes, and CSP-safe scripts. Machine-pinned raw output:
`dashboardui-render-2026-09-17.txt` (same change).
