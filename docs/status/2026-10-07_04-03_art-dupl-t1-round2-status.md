# Status — art-dupl `-t 1` dedup round 2: 113 groups triaged, 7 extracted, zero harmful duplication

> **Generated:** 2026-10-07 04:03 CEST · **Session scope:** the user-pasted `art-dupl --sort total-tokens -t 1` report (threshold dropped to the floor: 2163 detected / 113 shown / 564 tokens total) executed per the `deduplicate-code` skill — read every candidate class, triage, extract what has a domain name and real change-amplification, accept the rest with rationale.
> **Verdict up front:** 7 groups extracted (all verified), 106 accepted-by-class or accepted-with-rationale, 4 groups contain telemetry-session files (that session has since landed — triaged on the merits anyway). Post-round re-run: 2163 → 2110 detected, 113 → 91 shown; every extracted group is gone from the report. All touched modules build / vet / test (-race) / lint green in my files; `erraudit-inventory` TOTAL=0; `check-codegen` green.

---

## a) FULLY DONE — extractions

| # | Group | Extraction | Module |
|---|-------|-----------|--------|
| 1 | #110 `core/overview.go:153,178` — identical 15-line RecentEvents loops on both journal paths | `recentEventsFrom(events []event.Event) []RecentEvent` — caps at `RecentEventsLimit`, projects display shape in ONE place; both branches are now single assignments | dashboardui/core |
| 2 | #30 `session_gate.go:8` + `verification_totp_http.go:49` — twin user-or-401 guards | Unified on `currentUser` (5 callers vs 2); `sessionUserID` deleted, its doc moved over; the ADR-0055 route-wrapper gate's chokepoint markers list BOTH names, so the invariant checker is unaffected (re-ran green: 20 registrations / 13 wrapped) | usermgmt |
| 3 | #97 `es_setup.go:178` + `service_core.go:526` — twin projection-host stop blocks (9 lines, log scope + error code are the only diff) | `stopProjectionHost(host *projectionhost.Host, scope, code string)` — nil-safe, logs (never fails) so bus/store still close; both Close paths are 1-line calls | usermgmt |
| 4 | #59 `journalsse.go:137,155` — maxReplay default guard ×2 | `replayLimit()` method; the "≤0 = DefaultMaxReplay" rule is stated once. Note (accepted, not changed): the fullScan path deliberately treats `≤0` as UNLIMITED while seekable treats it as the default — a semantic divergence between paths, not a clone | root/transport |
| 5 | #64+#69 `setup/bundle.go` + `setup/sse.go` — SSE teardown pairs | `closeSSEDone(b)` (exactly-once done-channel close) + `closeSSEBroadcasters(b)` (close + nil hub pair); bundle `Close` keeps its drain BETWEEN the two, the subscribe-rollback path calls both back-to-back | setup |
| 6 | #22 (+ the identity-model half of #57) `fold.go:234,250,336` — defensive payload copies ×3 | `copySlice[T any](src []T) []T` — one generic states the "folded state never aliases payload slices" rule; used for 2× roles + 1× scopes | identity-model |
| 7 | #21 `layout.templ:37,159` — ThemeScript + stylesheet links in the page shell AND the error shell | `templ dashboardStylesheets(nonce, basePath string)` — the shared `<head>` payload; canonical `nix run .#gen`, `check-codegen` green | dashboardui |

**En-route cleanup:** the #110 extraction dropped `FetchOverview` below BOTH the gocognit and cyclop thresholds — the `//nolint:gocognit,cyclop` directive went stale in two steps and is now removed entirely (`nolintlint` confirmed both halves unused). `core` lints 0 issues with no suppression.

## b) ACCEPTED — with per-class rationale

- **Templ page-skeleton idiom (largest class: #5, #9, #12, #16, #17, #18, #42, #48, #51, #72, and most low groups):** `@emptyStatePanel(...)} else { table + pager }` per list page; the varying parts (icon, title, headers, rows fn, page path, sort) are exactly what a shared component would take as parameters — an abstraction would need ~7 params for ~6 markup lines. Rationale already lives at the round-1 sites (`emptyStatePanel`, `listNote`, `definitionList`); this round adds no new sites needing comments.
- **Overlapping-window artifacts (#12-style):** consecutive `if X != "" { @statCard }` sequences produce window-shifted clones; nothing to extract.
- **Cross-module ≤6-line twins (#24, #43, #58, #71, #107, #112, #14):** adminui/dashboardui error-page serving, BasePath defaulting, config validation, badge-default branches, the `navItem` struct, and the (now doc-comment-only) asset wrappers. Both UI modules are independently versioned; sharing any of these means a new root export (release-train cost >> duplicated lines) or a templ-components move (foreign repo). The navItem twin is 5 fields of per-module view-model plumbing; dashboardui's exported `NavLink`/`PageMeta` already own the consumer-facing half.
- **Parallel-domain guards (#62, #68, #70, #105, #108, #7, #29):** bot-vs-tenant not-founds, DisableService-vs-DisableAuth rejections, script-tag nonce variants, bundle-close cleanup, 400-on-empty-target — same SHAPE, different business rules; params are the difference.
- **Fold/read-model tails (#11/#17-style, #19, #39, #75, #80):** event-fold apply methods are parallel by domain design.
- **Idioms/artifacts (#25, #32, #33, #40, #47, #50, #57's usermgmt half, #61, #66 — which is the round-1 `closeBus` helper being called twice — #66, #74, #79, #83, #27):** mutex/map lookups, `Must` panics, generic decode prologues, select tails, switch defaults, string-slice projections in unrelated packages.

## c) FOREIGN-OWNED (touched by nobody this round)

Groups containing `dashboardui/telemetry.templ`, `telemetry_helpers.go`, `accent_color.go` (#2, #7/#8, #14, #20, #24-of-old, plus scattered occurrences). The telemetry session has since LANDED (tree clean, `check-codegen` green — their one-line generated drift self-healed). Triaged on the merits anyway: every one of these is a cell-glue/validation-loop accept-by-class; none warrants extraction. The prior round's TODO_LIST telemetry-triage follow-up is therefore RESOLVED as "no action (accept-by-class)".

## d) VERIFICATION

- Per-module battery after each extraction: `go build` + `go vet` + `go test -race` — dashboardui (incl. core), usermgmt (twice), identity-model, transport, setup: all green. `check-session-route-wrappers` green after the guard unification (20/13/7, ADR-0055 invariant holds).
- Lint: identity-model / transport / setup / usermgmt / dashboardui-core 0 issues. dashboardui module-level findings are exclusively in the telemetry session's files (`handlers_events.go` cyclop, `accent_color.go` gochecknoglobals+mnd) — not mine, untouched.
- `erraudit-inventory` TOTAL=0 · `check-codegen` PASSED · formatting `.#fmt` on all 12 touched paths: 0 changed.
- Post-round art-dupl re-run: detected 2163 → 2110, shown 113 → 91, and NONE of the 7 extracted groups appear; the top of the remaining report is exactly the accepted classes above.

## e) NEXT (unchanged from the round-1 report — user input still pending)

1. The 3 open questions from the round-1 report stand: telemetry lint findings ownership, ETag byte-identical format decision, fixture fold-in scope (Q3).
2. Release train still owed for BOTH dedup rounds (root + usermgmt + identity-model + transport + setup + dashboardui changed; all internal-only, so a single patch wave suffices once the telemetry session's train lands — coordinate to avoid double-tagging).
3. The fullScan-vs-seekable maxReplay divergence (see a-4 note) is worth an explicit decision in the v5 window: unify on `replayLimit()` semantics or document the difference.
