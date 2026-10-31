# Upstream draft: go-cqrs-lite — journal newest-N reads (ReadLastN)

Target: github.com/LarsArtmann/go-cqrs-lite (event journal / storage read
APIs). Status: S55 packet U2 draft, refreshed 2026-10-31.

## Finding

`ReadFrom(zero, n)` returns the OLDEST n events (head semantics), so a
"show me the latest events" view that reads from a zero cursor gets the
WRONG window entirely. cqrs-htmx's dashboardui hit this twice: the
overview's recent-events panel had to grow a rolling-tail walker
(`core.LoadTailEvents`) that pages the journal keeping a running tail, and
every hand-rolled "newest-N" consumer since re-implements the same loop.

## Ask

`ReadLastN(ctx, streamRef, n)` (and/or a documented `ReadTail` helper) that
returns the NEWEST n events in stream order — one primitive instead of every
consumer re-deriving tail paging, and one place to keep the cursor semantics
honest (the in-repo walker stall-guards with `*_cursor_stalled`; gotcha 42's
class).

## Consumer cost today

Every newest-N surface pages backwards manually; a naive zero-cursor read
silently shows the oldest data (the 2026-10-18 overview-blanking-adjacent
class).
