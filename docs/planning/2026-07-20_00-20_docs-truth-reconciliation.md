# Docs Truth Reconciliation — Comprehensive Plan

**Created:** 2026-07-20 00:20 | **Author:** Crush session | **Status:** PLANNED — awaiting execution

> **Goal:** Fix every lie, split brain, and gap introduced or surfaced during the 2026-07-19 docs reorganization session. Make the docs worthy of trust.

---

## Context — Why This Plan Exists

On 2026-07-19, a Crush session reorganized `TODO_LIST.md` (rejected 3 items to ROADMAP Not Planned, purged ~50 `[x]` items, moved completed work to CHANGELOG). The session committed its work (`8fad996`, `26bc931`). A brutal self-review then surfaced **critical inconsistencies** that the session introduced or failed to fix:

1. **FEATURES.md "Not Planned" table has 4 rows; ROADMAP "Not Planned" has 10** — 6-item split brain.
2. **go-cqrs-lite version strings differ across docs** — ROADMAP says `v4.0.0`, TODO says `v4.0.1`, go.mod has mixed `v4.0.0`/`v4.0.1`/`v4.0.2`.
3. **TODO_LIST.md has `[x]` items again** — other agents added 3 `[x]` items (Phase 2b, Snapshot, TypedRepository) after the convention was established that `[x]` items belong in CHANGELOG, not TODO.
4. **CHANGELOG has TWO `[Unreleased]` sections** — one above `[v4.3.0]` (post-tag work: partial rendering helpers), one below it (orphaned pre-v3.3.0 work: observability wiring). Structural bug.
5. **Version headers say `v4.2.1+unreleased`** but `v4.3.0` is tagged and pushed.
6. **A deferred decision was silently dropped** — "integration test importing published version" was `[x] DEFERRED` in old TODO, got deleted without landing in ROADMAP.
7. **CHANGELOG backfill was sample-based, not exhaustive** — the session hand-waved "~15 process items removed" without verifying CHANGELOG coverage for each.

Meanwhile, other agents shipped major features (snapshots, IndexedDB Phase 2b, enum Valid(), dispatch dedup) — so the codebase has moved forward. This plan accounts for that.

---

## Pareto Analysis — Where the Leverage Is

### The 1% that delivers 51% (CRITICAL — fix the lies)

Fix the 5 split brains and structural bugs. These actively mislead every reader right now:

- **C1:** FEATURES Not Planned sync (6 items missing)
- **C2:** Version string reconciliation
- **C3:** Re-home `[x]` TODO items to CHANGELOG (restore convention)
- **C4:** Fix CHANGELOG dual `[Unreleased]` structural bug
- **C5:** Recover dropped "published version integration test" decision

### The 4% that delivers 64% (HIGH — close the verification gaps)

- **H1:** Run `nix run .#check-docs-freshness` — the tool exists for exactly this
- **H2:** Systematic CHANGELOG audit — grep every removed `[x]` item, build coverage matrix
- **H3:** Backfill any real orphans found

### The 20% that delivers 80% (MEDIUM — consistency sweep)

- **M1:** Stat reconciliation (coverage %, test counts, lint count — actual vs documented)
- **M2:** ROADMAP consolidation (merge "v4.1.0 Deferred" into Not Planned, refresh stale v3.4.0 rows)
- **M3:** ADR reference verification + CONTRIBUTING module count fix

### The other 20% (LOW — polish)

- **L1:** Status report for this recovery session
- **L2:** AGENTS.md memory note about TODO_LIST convention (prevent recurrence)
- **L3:** docs/status consistency check (old reports that reference removed items)

---

## Level 1 — Execution Tasks (30–100min each)

Sorted by impact (desc), effort (asc), customer-value (desc).

| ID        | Task                                                                                                      | Covers      | Impact                                                          | Effort | Priority | Blocked By       |
| --------- | --------------------------------------------------------------------------------------------------------- | ----------- | --------------------------------------------------------------- | ------ | -------- | ---------------- |
| **L1.1**  | Ask 3 decision questions (TOTP sub-module fate, OAuth2 admin views, version scheme)                       | D1–D3       | Unblocks 4 downstream tasks                                     | 5min   | P0       | —                |
| **L1.2**  | Fix FEATURES.md Not Planned split brain — add 6 missing rows to match ROADMAP                             | C1          | Critical — every FEATURES reader sees incomplete rejection list | 25min  | P0       | L1.1 (D1)        |
| **L1.3**  | Reconcile go-cqrs-lite version strings across all docs — ROADMAP, TODO, FEATURES, AGENTS                  | C2          | Critical — docs disagree about a core dependency                | 30min  | P0       | L1.1 (D3)        |
| **L1.4**  | Re-home `[x]` TODO items to CHANGELOG + fix dual `[Unreleased]` structural bug + recover dropped decision | C3+C4+C5+H4 | Critical — convention violation + structural bug                | 45min  | P0       | L1.1 (D3)        |
| **L1.5**  | Run `nix run .#check-docs-freshness`, triage every flag, fix all                                          | H1          | High — automated verification of everything manual might miss   | 60min  | P1       | L1.2, L1.3, L1.4 |
| **L1.6**  | CHANGELOG systematic audit + backfill — grep every prior `[x]` item, build coverage matrix, fill gaps     | H2+H3       | High — ensures no completed work is lost                        | 90min  | P1       | —                |
| **L1.7**  | Stat reconciliation — run actual coverage/tests/lint, update all doc headers to match reality             | M1          | Medium — docs cite stale numbers                                | 45min  | P2       | —                |
| **L1.8**  | ROADMAP consolidation — merge "v4.1.0 Deferred" into Not Planned, refresh stale v3.4.0 rows               | M2          | Medium — ROADMAP has redundant sections and stale statuses      | 35min  | P2       | —                |
| **L1.9**  | ADR reference verification + CONTRIBUTING module count fix                                                | M3+M5       | Medium — broken ADR refs mislead; module count wrong (11 vs 12) | 30min  | P2       | —                |
| **L1.10** | Status report + AGENTS.md convention note + docs/status consistency check                                 | L1+L2+L3    | Low — polish and recurrence prevention                          | 60min  | P3       | All above        |

**Total estimated effort:** ~7h (some parallelizable). Critical path: L1.1 → L1.4 → L1.5 → done.

---

## Level 2 — Atomic Breakdown (≤12min each)

Every L1 task decomposed into atomic, independently-executable steps. Sorted by priority then dependency order.

### P0 — Critical (L1.1–L1.4)

| ID    | Task                                                                                  | Parent | Effort | Depends On          |
| ----- | ------------------------------------------------------------------------------------- | ------ | ------ | ------------------- |
| L2.01 | Ask D1–D3 via question tool (TOTP sub-module, OAuth2 views, version scheme)           | L1.1   | 5min   | —                   |
| L2.02 | Read FEATURES.md Not Planned section (confirm 4 items)                                | L1.2   | 3min   | —                   |
| L2.03 | Read ROADMAP.md Not Planned section (confirm 10 items)                                | L1.2   | 3min   | —                   |
| L2.04 | Draft 6 new FEATURES rows matching ROADMAP rationale (shorter format for table)       | L1.2   | 8min   | L2.02, L2.03        |
| L2.05 | Edit FEATURES.md: add 6 rows to Not Planned table                                     | L1.2   | 5min   | L2.04               |
| L2.06 | Verify FEATURES item count == ROADMAP item count                                      | L1.2   | 2min   | L2.05               |
| L2.07 | Grep `go-cqrs-lite v4` across all .md files — collect every variant                   | L1.3   | 3min   | —                   |
| L2.08 | Read go.mod for actual go-cqrs-lite module versions                                   | L1.3   | 3min   | —                   |
| L2.09 | Decide canonical version representation (e.g., "v4.0.x" or "v4.0.0–v4.0.2")           | L1.3   | 5min   | L2.07, L2.08, L2.01 |
| L2.10 | Edit ROADMAP.md: fix version string                                                   | L1.3   | 5min   | L2.09               |
| L2.11 | Edit TODO_LIST.md: fix version string                                                 | L1.3   | 3min   | L2.09               |
| L2.12 | Edit FEATURES.md: fix version string (if present)                                     | L1.3   | 3min   | L2.09               |
| L2.13 | Verify version consistency via grep — all docs match                                  | L1.3   | 3min   | L2.10–L2.12         |
| L2.14 | Read TODO_LIST.md `[x]` items (Phase 2b, Snapshot, TypedRepository)                   | L1.4   | 3min   | —                   |
| L2.15 | Check if Phase 2b (IndexedDB ADR-0040) is in CHANGELOG                                | L1.4   | 5min   | L2.14               |
| L2.16 | Check if Snapshot integration is in CHANGELOG                                         | L1.4   | 5min   | L2.14               |
| L2.17 | Check if TypedRepository rejection rationale is in CHANGELOG or ROADMAP               | L1.4   | 5min   | L2.14               |
| L2.18 | Draft CHANGELOG entries for any missing items                                         | L1.4   | 8min   | L2.15–L2.17         |
| L2.19 | Add entries to CHANGELOG (appropriate version section)                                | L1.4   | 8min   | L2.18               |
| L2.20 | Remove `[x]` items from TODO_LIST — leave only `[ ]` open items                       | L1.4   | 5min   | L2.19               |
| L2.21 | Read CHANGELOG to locate both `[Unreleased]` headers                                  | L1.4   | 5min   | —                   |
| L2.22 | Rename top `[Unreleased]` → `[v4.4.0]` or decision from D3                            | L1.4   | 8min   | L2.01, L2.21        |
| L2.23 | Fix second `[Unreleased]` (old, pre-v3.3.0) → note as historical or merge into v3.3.0 | L1.4   | 8min   | L2.21               |
| L2.24 | Recover "published version integration test" deferred decision → ROADMAP Not Planned  | L1.4   | 5min   | —                   |
| L2.25 | Verify TODO_LIST has 0 `[x]` items via grep                                           | L1.4   | 3min   | L2.20               |

### P1 — High (L1.5–L1.6)

| ID    | Task                                                                                      | Parent | Effort | Depends On     |
| ----- | ----------------------------------------------------------------------------------------- | ------ | ------ | -------------- |
| L2.26 | Run `nix run .#check-docs-freshness` — capture raw output                                 | L1.5   | 3min   | L1.2–L1.4 done |
| L2.27 | Categorize flags: version strings, API refs, Go version, HTMX version, deprecated symbols | L1.5   | 10min  | L2.26          |
| L2.28 | Fix version string flags across files                                                     | L1.5   | 12min  | L2.27          |
| L2.29 | Fix API reference flags (removed/renamed symbols)                                         | L1.5   | 12min  | L2.27          |
| L2.30 | Fix Go version / HTMX version refs if stale                                               | L1.5   | 8min   | L2.27          |
| L2.31 | Re-run check-docs-freshness to verify clean                                               | L1.5   | 5min   | L2.28–L2.30    |
| L2.32 | Extract old `[x]` items from git: `git show 26bc931^:TODO_LIST.md`                        | L1.6   | 8min   | —              |
| L2.33 | Build coverage matrix: each item's key noun → grep CHANGELOG → Y/N                        | L1.6   | 12min  | L2.32          |
| L2.34 | Identify orphans (items where grep returned N)                                            | L1.6   | 10min  | L2.33          |
| L2.35 | Categorize orphans: real change / process / decision / already-in-ROADMAP                 | L1.6   | 10min  | L2.34          |
| L2.36 | Backfill real-change orphans into CHANGELOG                                               | L1.6   | 12min  | L2.35          |
| L2.37 | Move decision orphans to ROADMAP if not already there                                     | L1.6   | 8min   | L2.35          |
| L2.38 | Re-verify: no orphans remain unaccounted for                                              | L1.6   | 5min   | L2.36, L2.37   |

### P2 — Medium (L1.7–L1.9)

| ID    | Task                                                                | Parent | Effort | Depends On  |
| ----- | ------------------------------------------------------------------- | ------ | ------ | ----------- |
| L2.39 | Run `nix run .#coverage` — capture per-module numbers               | L1.7   | 10min  | —           |
| L2.40 | Count tests per module (go test -v ./... 2>&1 \| grep -c '^--- ')   | L1.7   | 12min  | —           |
| L2.41 | Run `nix run .#lint` — capture issue counts per module              | L1.7   | 10min  | —           |
| L2.42 | Update TODO_LIST.md header stats to match actuals                   | L1.7   | 5min   | L2.39–L2.41 |
| L2.43 | Update FEATURES.md metrics table to match actuals                   | L1.7   | 5min   | L2.39–L2.41 |
| L2.44 | Read ROADMAP "v4.1.0 God-Package Split (Deferred)" section          | L1.8   | 8min   | —           |
| L2.45 | Identify overlap with Not Planned section                           | L1.8   | 10min  | L2.44       |
| L2.46 | Merge unique items into Not Planned or delete duplicates            | L1.8   | 10min  | L2.45       |
| L2.47 | Read v3.4.0 table — identify items that shipped                     | L1.8   | 10min  | —           |
| L2.48 | Update v3.4.0 table statuses (Planned → Done where applicable)      | L1.8   | 8min   | L2.47       |
| L2.49 | Grep all `ADR-00` references across CHANGELOG/ROADMAP/TODO/FEATURES | L1.9   | 5min   | —           |
| L2.50 | Verify each ADR file exists in docs/adr/                            | L1.9   | 10min  | L2.49       |
| L2.51 | Fix broken ADR references                                           | L1.9   | 5min   | L2.50       |
| L2.52 | Count actual modules in go.work (confirm 12)                        | L1.9   | 3min   | —           |
| L2.53 | Edit CONTRIBUTING.md module count if wrong                          | L1.9   | 5min   | L2.52       |

### P3 — Polish (L1.10)

| ID    | Task                                                                                   | Parent | Effort | Depends On |
| ----- | -------------------------------------------------------------------------------------- | ------ | ------ | ---------- |
| L2.54 | Write status report for this recovery session                                          | L1.10  | 10min  | All above  |
| L2.55 | Save to docs/status/2026-07-20_00-20_docs-truth-reconciliation.md                      | L1.10  | 5min   | L2.54      |
| L2.56 | Draft AGENTS.md Gotchas note: "TODO_LIST contains only `[ ]` items; `[x]` → CHANGELOG" | L1.10  | 8min   | —          |
| L2.57 | Edit AGENTS.md Gotchas section                                                         | L1.10  | 5min   | L2.56      |
| L2.58 | Grep docs/status/* for references to rejected items (TOTP views, Redis, codemod)       | L1.10  | 10min  | —          |
| L2.59 | Annotate stale status reports that contradict current state (if any critical)          | L1.10  | 10min  | L2.58      |
| L2.60 | Final cross-file consistency grep — no `[x]` in TODO, versions match, links resolve    | L1.10  | 7min   | All above  |

**Total Level 2 tasks:** 60 atomic steps, all ≤12min.

---

## Decision Questions (Block L1.1 — Must Ask Before Execution)

These cannot be answered from code or docs alone. They require product direction judgment.

### D1 — TOTP sub-module fate

The TOTP admin views were rejected ("passkeys-first, not old-school"). But the `usermgmt/totp/v4` sub-module stays. Should `FEATURES.md` downgrade the TOTP MFA row from `FULLY_FUNCTIONAL` to something signaling "maintenance-only, not promoted"?

- **Option A:** Keep `FULLY_FUNCTIONAL` — the sub-module works, just isn't promoted in admin UI
- **Option B:** Add `MAINTENANCE_ONLY` status — signals "exists but not the path forward"
- **Option C:** Move TOTP MFA row to "Not Planned" — strongest signal

### D2 — OAuth2 admin link/unlink views

TODO_LIST still has `[ ] Admin UI: OAuth2 link/unlink views`. OAuth2 IS in-scope (passkeys + OAuth). But is the admin UI link/unlink management view still wanted, or is the existing OAuth2 login flow sufficient?

- **Option A:** Keep open — OAuth2 management views are still wanted
- **Option B:** Reject — login flow is enough; admin link/unlink crosses the same "not building a full IAM UI" line

### D3 — Version numbering scheme

`v4.3.0` is tagged and pushed. TODO_LIST and ROADMAP headers say `v4.2.1+unreleased`. CHANGELOG has a top `[Unreleased]` section with partial rendering helpers (post-v4.3.0 work). What should the next version be?

- **Option A:** `[v4.4.0]` — the partial rendering helpers + any other new work gets a new minor
- **Option B:** `[v4.3.1]` — treat as patch (if partial rendering helpers are additive, non-breaking)
- **Option C:** Keep `[Unreleased]` for now — don't decide version until ready to tag

---

## Execution Graph

```mermaid
graph TD
    %% Decision gate — blocks everything
    DEC{{'L1.1: Ask D1-D3<br/>5min — P0'}}
    DEC --> |'D1: TOTP fate'| F1
    DEC --> |'D3: version scheme'| F3
    DEC --> |'D3: version scheme'| F4

    %% Critical fixes (P0) — partially parallel
    subgraph CRIT['Critical — Fix Split Brains']
        F1['L1.2: FEATURES Not Planned sync<br/>25min']
        F2['L1.3: Version reconciliation<br/>30min']
        F3['L1.4: Re-home [x] items + fix Unreleased<br/>45min']
        F4['L1.4: Recover dropped decisions<br/>5min']
    end

    F1 --> GATE1
    F2 --> GATE1
    F3 --> GATE1
    F4 --> GATE1

    %% Verification (P1) — sequential after critical
    GATE1{{'All critical done'}} --> V1
    V1['L1.5: Run check-docs-freshness<br/>60min — P1'] --> V2
    V2['L1.6: CHANGELOG systematic audit<br/>90min — P1']

    %% Medium (P2) — can parallelize with verification
    V2 --> M1
    M1['L1.7: Stat reconciliation<br/>45min — P2'] --> M2
    M2['L1.8: ROADMAP consolidation<br/>35min — P2'] --> M3
    M3['L1.9: ADR + CONTRIBUTING verify<br/>30min — P2']

    %% Polish (P3) — last
    M3 --> P1
    P1['L1.10: Status report + memory note<br/>60min — P3'] --> DONE

    DONE(['✅ All docs truthful and consistent'])

    %% Styling
    style DEC fill:#ff6b6b,color:#fff
    style CRIT fill:#fff3cd
    style GATE1 fill:#d4edda
    style DONE fill:#d4edda
```

---

## Safety Constraints — Do Not Verschlimmbessern

1. **Read before write.** Every edit must be preceded by viewing the current file state. The codebase moved since the session started — stale reads will produce bad edits.
2. **Never touch other agents' uncommitted work.** As of 2026-07-20, the working tree has 14 modified files and 2 untracked files that are NOT part of this plan. They belong to other sessions. Leave them.
3. **Verify, don't trust.** Every stat claim (coverage %, test count, lint count) must be recomputed from the actual build, not copied from a doc.
4. **One logical change per edit.** Don't batch unrelated fixes into a single edit. If FEATURES.md needs both Not Planned sync AND version string fix, do them as two separate edits.
5. **Run check-docs-freshness last.** It's the safety net. Run it after all manual fixes to catch anything missed.
6. **Don't introduce new `[x]` items.** The convention is: TODO_LIST has only `[ ]` and `[~]`. Completed work goes to CHANGELOG. Deferred/rejected goes to ROADMAP. If you complete a task during this plan, it goes to CHANGELOG, not back to TODO as `[x]`.

---

## Success Criteria

- [ ] FEATURES.md "Not Planned" has the same item count as ROADMAP "Not Planned"
- [ ] go-cqrs-lite version string is consistent across all docs (and matches go.mod reality)
- [ ] TODO_LIST.md has zero `[x]` items
- [ ] CHANGELOG has exactly one `[Unreleased]` section (or zero if promoted to versioned)
- [ ] `nix run .#check-docs-freshness` exits clean
- [ ] Every previously-removed `[x]` TODO item is accounted for in CHANGELOG or ROADMAP
- [ ] Coverage/test/lint stats in docs match actual `nix run` output
- [ ] All ADR references resolve to existing files
- [ ] CONTRIBUTING.md module count matches `go.work`
- [ ] No status report in docs/status/ contradicts the current state of the docs
