# Status Report — art-dupl Combined-Mode Dedup Session

> **Point-in-time snapshot:** 2026-10-18 09:48 CEST · session scope: the art-dupl
> `-t 2 --suggest-generics --type-aware` run and the dedup work executed this
> session only (per operator instruction: no unrelated research).
> Notation: ✅ fully done · 🟡 partially done · ⬜ not started · 💀 fucked up.

---

## TL;DR

| Metric | Before | After |
| --- | --- | --- |
| Art-dupl actionable clone groups | 28 | **19** (all adjudicated accepts) |
| Total clones | 70 | **43** |
| Total duplicated tokens | 180 | **104** |
| Extractions shipped | — | **9** across 6 modules |
| Behavior fixes found on sight | — | **1** (bus leak, CHANGELOG'd) |
| Modules touched | — | root (openapi + cmd), usermgmt, loginpage, dashboardui, adminui |

Every one of the 28 actionable groups now has a recorded verdict:
**9 extracted**, **19 accepted with a defensible reason**, 0 left unjudged.

---

## a) FULLY DONE ✅

1. **Report ingest + catalog.** Parsed the full 1,269-line HTML output log (the
   stdout had 336 lines omitted); extracted all 28 clone groups with category,
   priority, occurrence lists, and the tool's suggestion per group.
2. **Judgment pass.** Every group read at source and classified
   extract / accept / exclude — no group processed from the report text alone.
3. **`openapi.Spec.pathItem`** — get-or-create PathItem shared by `Path` and
   `PathOp` (builder.go:75); the "merge, never replace" contract now lives once.
4. **`wellKnownQualifier`** in `cmd/cqrs-htmx-upgrade` — the three import-path
   → package-name facts both `defaultPackageName` and `defaultQualFor` must agree
   on, in one place (derive.go:213). Behavior-identical (verified the
   `qualToPath` consumer before choosing the minimal-subset design).
5. **`projectionStatuses`** — nil-safe host-status adaptation shared by
   `EventSourcedSetup.ProjectionStatuses` and `Service.ProjectionStatuses`
   (es_projection_health.go:57), mirroring the file's existing
   `rebuildProjection` delegation pattern.
6. **`nonceAttr`** in loginpage — scriptTag/jsonScriptTag collapsed to single
   expressions; the sanitization contract (base64url, see `sanitizeNonce`) moved
   with the attribute builder (handler.go:333). Confirmed `sanitizeNonce` runs
   upstream in the handler before touching anything.
7. **`assertContains`/`assertNotContains`** test helpers in loginpage — 50
   three-line assertion blocks collapsed to one line each (39 + 11 sites).
8. **`onlyRunes`** in dashboardui accent_color.go — `isHexDigits` and
   `onlyNumericColorComponents` now one-line delegates over the shared scanner.
9. **`eventTimelineCells`** templ component (components.templ) — the three
   event-row columns shared by aggregate-timeline and time-travel tables;
   the `"2006-01-02 15:04:05"` format constant is now owned in one place.
10. **`auditedTimeTypeCells`** templ component (audit.templ) — received-at/type
    columns shared by command and query audit rows.
11. **`respondErrorHTML`** in adminui — the header-trio + HTMX-branch respond
    tail shared by `writeErrorPage` and `notFoundHandler` (errorpage.go:142).
12. **Bus-leak fix (on sight, not dedup):** `NewEventSourcedSetup` leaked the
    library-created bus when `applyBusMiddleware` failed — the only
    post-bus-creation error path that skipped `closeBus`. Fixed (es_setup.go:240)
    + CHANGELOG receipt under Unreleased → Fixed (gotcha 20: consumer-visible).
13. **Per-module verification at each step:** build + vet + `go test` green on
    usermgmt (17.6s), loginpage, adminui; root packages (openapi, cmd) green.
14. **templ regeneration:** `nix run .#gen` (module-dir canonical form),
    resulting `_templ.go` committed.
15. **Lint:** `golangci-lint run` (GOWORK=off) green on usermgmt, loginpage,
    adminui, and the root openapi/cmd packages; dashboardui findings all sit in
    the concurrent session's files (verified none in mine).
16. **Verification re-run:** art-dupl with identical flags → 70→43 clones,
    28→19 groups; the remaining set matches the accept list **exactly**.
17. **Work committed** — carried by the auto-commit daemon's heuristic commits
    (9e78659b, 8830e536, and earlier), unavoidably interleaved with the
    concurrent session's files on the shared tree.

### The 19 accepted groups (with reasons, so the next audit doesn't relitigate)

- `sql_readmodel_extra.go` find/marshal guards (×2 groups) — **documented as
  deliberate at source** (line 318-320): the file already max-extracted via
  `deleteViewOnTombstone`/`upsertView`/`queryViewByName`/`marshalViewJSON`.
- `adminui/assets.go` ↔ `dashboardui/assets.go` — the comment says it: read +
  serve logic lives in the root module; only embed decl + error presentation
  are local on purpose.
- `es_readmodel.go` `u.UpdatedAt = evt.OccurredAt()` (×2 groups) — the
  fold-handler idiom itself; extracting a `touch()` saves nothing.
- `handlers_audit.go` command/query export switch — domain symmetry
  (parallel command/query surfaces), not copy-paste.
- `aggregates/snapshots/timetravel` page-header blocks — page-specific markup
  sharing only a CSS class contract; subtitles embed components.
- `app.go` ↔ `setup/setup.go` MustNew — cross-module Go idiom.
- `decoder.go` decodeJSON/decodeForm heads — generic boilerplate; a helper
  would return an unused zero value.
- `es_setup.go` closeBus error paths (now ×3) — explicit per-path lifecycle
  cleanup; a defer would change the not-yet-created-bus path semantics.
- adminui templ relativeTime cells — shares Tailwind classes only; the
  actor-cell else-branches genuinely differ.
- tone/badge switch tails (×2 groups) — distinct domain mappings, each with
  its own severity philosophy in its doc comment.
- `setup/config.go` DisableService/DisableAuth AuthHandlerConfig rejections —
  different validators with deliberately different messages.
- `setup/run.go` ↔ `run_appkit.go` select tails — structural noise.
- config-defaults idioms (×2, cross-module) — different Config types.
- `adminui/layout.templ` 2-token param-list noise.

---

## b) PARTIALLY DONE 🟡

1. **dashboardui module-level verification.** My touched surfaces are green
   (scoped `go test -run 'Aggregate|TimeTravel|Audit|Accent|Valid|Config|Asset|
   Error|NotFound'` rc=0, build + vet rc=0), but the module's full test run has
   **4 failing tests in core/overview** — all inside the concurrent session's
   in-flight work (their daemon commits touched core/events.go +197,
   core/overview.go, handlers_events.go, overview tests + goldens). Not mine to
   fix; module cannot be declared fully green until their wave settles.
2. **Commit hygiene.** My work is spread across ≥5 heuristic daemon commits
   interleaved with the foreign session's files. Content is safe and the tree is
   green where it's mine, but there is no single reviewable "dedup pass" commit,
   and per gotcha 4 the heuristic titles stay as accepted debt while the daemon
   lives.
3. **Accept-list mechanization.** The 19 accepts are recorded in prose (this
   report + the chat summary). The skill's stronger form — a committed art-dupl
   baseline report gating the group count (the go-etag pattern) — is **not**
   implemented. Next relitigator re-reads prose instead of failing a gate.
4. **CHANGELOG granularity.** The bus-leak fix has its receipt. The two new
   templ components are lowercase (same-package only, NOT public surface) so no
   receipt is strictly required — but if any consumer-facing docs enumerate
   dashboardui internals, they now lag by two components.
5. **Concurrency rituals.** `preflight-tree-check` was never run before the
   tree-mutating batch; the concurrent session was only detected when its
   `formatLinksNav` upgrade collided with my `templ generate` run. Detection
   worked (no foreign change was reverted), but by luck-plus-diff-reading, not
   by the mechanical gate.

---

## c) NOT STARTED ⬜

1. The **2 new clone groups in handlers_audit.go** (78/170 + 21/36) — created by
   the concurrent session's mid-flight edits; theirs until their wave lands.
2. **HARVEST** — section (f) below has not been routed into TODO_LIST/ROADMAP.
3. The **medium-priority accepts I deliberately deferred** (see f-items 2-3):
   `sql_readmodel_extra` generic find-and-marshal, `setup/config.go` validator
   helper — both touch published module code and should ride a train decision.
4. **erraudit scoped step** on the touched modules (low risk — no new error
   returns were added — but never explicitly run).
5. **Workspace-wide battery** (`nix run .#test`, `check-modules`, coverage-gate,
   branching-flow re-pin check) — deferred until the tree quiets; racy now.
6. **No AGENTS.md memory update** from this session — two candidate gotchas
   identified (see e-7/e-8), not yet written.
7. **Golden page tests** for the aggregate/timetravel/audit table rows my templ
   refactor touched — if they don't exist, the cell extraction is pinned only by
   handler-level tests, not rendered-output goldens.

---

## d) TOTALLY FUCKED UP 💀 (own failures this session, all caught + repaired)

1. **The loginpage regex transform mangled 2 combined-condition assertions.**
   I predicted single-line `||` conditions would not match the pattern regex;
   they did (non-greedy `.*?` happily spanned `") || !strings.Contains(body, \"`).
   The dry-run count (39) gave me no signal because I didn't assert against an
   expected count. **Caught** by the compile failure (3-arg call), fixed
   properly by hand. Damage: zero (pre-commit), cost: one failed test cycle.
   Lesson recorded as improvement e-5.
2. **The audit.templ multiedit dropped queryRows' View-button cell and `</tr>`,
   nesting the new component inside queryRows.** Root cause: my 4th edit folded
   closing markup into `old_string` without re-including it in `new_string`, and
   the tool's whitespace-equivalent re-indent path re-indented the damage.
   **Caught** by templ compiler diagnostics, repaired in one edit.
3. **Two wasted edit-tool round trips** ("file modified since read") on
   handler_test.go — the python writer invalidated my read state; I re-read via
   sed instead of the view tool first, so the first retry also failed.
4. **Ran tree-mutating steps (`nix run .#fmt`, `.#gen`) without the preflight
   ritual during a known-busy tree.** The gen run briefly produced generated
   code inconsistent with the concurrently-edited audit.templ (stale 2-arg
   formatLinksNav calls) and captured the foreign session's mid-flight
   events.templ edit into the commit stream. No foreign change was reverted and
   a second gen re-synced everything — but the sequence was luck-tolerant, not
   designed-safe. The pre-push-grade discipline (check `git status --short` for
   foreign churn immediately before each nix run) existed and I applied it late.
5. **Nothing data-destructive happened.** No reverts of foreign work, no lost
   edits, no pushed breakage. The failures were all caught by the verification
   layers that exist for exactly this purpose.

---

## e) WHAT WE SHOULD IMPROVE 💡

1. **Batch transforms need expected-count assertions.** A python transform over
   source should carry `assert n == EXPECTED` (or a per-match structural
   validation, e.g. reject matches containing `||`), not just a printed count.
2. **Never fold closing markup into an edit's old_string** without re-including
   it in new_string; for templ blocks prefer `lsp_replace_symbol`-style
   whole-symbol replacement or smaller anchored edits.
3. **Run `preflight-tree-check` before every tree-mutating batch** — it exists
   precisely for the concurrent-session case that actually occurred this
   session; skipping it converted a designed guard into ad-hoc diff reading.
4. **Commit per module phase in one chained call** (`fmt -- <paths> && git add
   <paths> && git commit`) instead of relying on the daemon at the end — would
   have produced clean per-module commits instead of 5 interleaved heuristics.
5. **Scoped verification as the default under concurrency:** `go test -run`
   over touched surfaces while a foreign session holds the module, then the
   full battery after they land. (Done here reactively; make it proactive.)
6. **Mechanize the dedup accept-list** as a committed baseline + gate (go-etag
   pattern) if recurring audits are wanted — an enforced group count cannot be
   relitigated; prose can.
7. **AGENTS.md candidate gotcha A:** regex/batch transforms over Go test files
   — single-line `||` combined conditions defeat line-anchored patterns; always
   dry-run with a reject-filter and an expected count.
8. **AGENTS.md candidate gotcha B:** `nix run .#gen` during a concurrent
   session's templ edits bakes their half-done state into generated files and
   the commit stream — smoke-check `git status --short` for foreign `.templ`
   churn immediately before gen, exactly like the gotcha-4 build smoke-check.
9. **art-dupl invocation history:** save each run's HTML to a file
   (e.g. `docs/analysis/dupl-<date>.html`) so audits are diffable across time
   instead of living in scrollback.
10. **Threshold strategy:** `-t 2` is a noise amplifier (867 detected → 28
    actionable shown, most 2-4 tokens). For triage audits, run `-t 5` (skill
    default) first for signal, then `-t 2` for the long tail.

---

## f) Up to 50 things to get done next

*(brainstorm, not a commitment list — impact-sorted in three tiers; most of
tier 3 is ROADMAP fuel)*

**Tier 1 — direct follow-ups on this session's work**

1. Wait for the concurrent dashboardui session to land, then run the **full
   dashboardui module test** + workspace `nix run .#test` to convert my scoped
   green into a module-level green.
2. Extract the **generic find-and-marshal helper** in
   `usermgmt/sql_readmodel_extra.go` (the accepted medium clone at :180/:270) —
   closes the last medium clone in usermgmt; rides the next train.
3. Extract **`rejectAuthHandlerConfig`** for the two setup validators
   (setup/config.go:879/943) — same-file, low risk; rides the next train.
4. Decide + implement the **art-dupl accept-baseline gate** (committed baseline
   report, gate fails on group-count drift) if recurring audits are wanted.
5. Fix the **2 handlers_audit.go clone groups** left by the foreign session,
   after confirming their work is landed and owned.
6. Add the **two AGENTS.md gotchas** (e-7, e-8).
7. Add **golden page tests** for aggregate-timeline / time-travel / audit table
   rows, pinning the extracted templ cells' rendered output.
8. Run **erraudit scoped** (`buildflow -s 'erraudit@<module>'` with
   `BUILDFLOW_NO_RESULT_CACHE=1`) over the five touched modules.
9. Run **coverage-gate** — the deletions shift line counts on 15 gated modules.
10. Run **check-branching-flow** and re-pin the SARIF baseline if my
    helper-extractions moved findings (switch → delegate changes shapes).
11. Route this report's (f) list through **docs-health HARVEST** into
    TODO_LIST/ROADMAP.
12. Verify **check-css-bundle-classes** is unaffected (no Tailwind classes were
    added or changed — expect green, confirm mechanically).
13. Re-run art-dupl at **-t 5** for a strategic-signal report and file both
    runs' HTML under docs/analysis/.

**Tier 2 — adjacent dedup/quality debt noticed but not acted on**

14. `buildSQLMembershipReadModel`/`Tenant`/`Bot` constructors share the
    build-view-store shape — a bigger generic extraction than anything the
    report flagged (same file, same architecture comment governs).
15. dashboardui `handler.go` 404 path could adopt an `respondErrorHTML`
    equivalent (the cross-module half of eliminated group 13).
16. Audit adminui for remaining raw `http.Error` sites that should render the
    styled error surface (consistency with writeActionError/writeErrorPage).
17. Evaluate a **`relativeTimeCell`** templ component (group 15 accept) if the
    `px-4 py-2 text-gray-500` cell pattern spreads further.
18. Evaluate a **`pageHeader`** templ component (group 7 accept) — only if a
    fourth header appears; three sites with composite children is the ceiling.
19. decoder.go: if a third decode*Body variant appears, extract the shared
    read-and-guard head then (two sites is below the threshold).
20. `es_readmodel.go`: consider a `touchUser(u, evt)` micro-helper ONLY if the
    fold handlers grow another UpdatedAt site — currently pure idiom.
21. Consider whether the **four config-defaults idioms** (adminui/loginpage/
    dashboardui withDefaults) justify a tiny shared helper module — cross-module
    cost/benefit needs your call.
22. closeBus: evaluate the **named-return + defer cleanup** form for
    `NewEventSourcedSetup` once more after the next constructor change (my
    accept reasoned from current call-graph; re-verify if `Service`'s
    constructor ever merges).
23. Check the **loginpage combined-condition semantics change** (two assertions
    now possible where one combined error was) — confirm acceptable, then it's
    done; it is documented here only.
24. Extend assertContains-style helpers to other modules' tests **where the
    3-line pattern is ≥5 sites** (scan first; adminui/dashboardui unknown).
25. Re-check the **`openapi` golden test** still pins the PathOp merge contract
    after the pathItem extraction (tests passed; confirm the golden covers
    get-or-create explicitly).

**Tier 3 — process / platform (ROADMAP fuel)**

26. Recurring **dedup audit cadence** (quarterly?) with filed reports — the
    art-dupl accept markers show 2026-10-16 self-scan precedent.
27. Ship the **`nix run .#dupl-report`** flake app wrapping the canonical
    art-dupl invocation + output path, so audits are one command and diffable.
28. Per-module **CI test isolation (matrix)** — gotcha 40's open question; this
    session re-demonstrated the masking risk (adminui red would have hidden
    dashboardui findings).
29. **`gci` import-order lint** fired on the foreign session's handlers_audit.go
    — consider adding gci autofix to the safe BuildFlow fix list if it's a
    recurring manual annoyance.
30. Track **daemon-commit attribution debt**: a session-end ritual to amend
    heuristic titles is impossible while the daemon lives; consider a
    "train-prep" step that squashes daemon commits per module before tagging.
31. Add **art-dupl to preflight** as an advisory stage (like docs-tail-budget)
    so dedup drift surfaces per-train without blocking.
32. Evaluate `--suggest-generics` output separately from same-type clones —
    the combined mode conflates triage queues; two reports may triage better.
33. Document the **accept-reason taxonomy** (idiom / documented-at-source /
    cross-module / below-threshold) in docs/analysis/README.md so future
    audits classify consistently.
34. Consider `templ` **lint rule: no literal timestamp format strings** outside
    the two owning components — mechanically prevents format drift.
35. Sweep for other **duplicated `"2006-01-02 15:04:05"` literals** across
    dashboardui/adminui templ files and route them to the owning components.
36. loginpage: consider exporting nothing new but documenting the
    scriptTag/jsonScriptTag/nonceAttr trio in its README § Styling if that
    section enumerates internals.
37. **Skills feedback:** the deduplicate-code skill's "leave a one-line
    rationale" works well in-repo; consider adding the baseline-gate recipe as
    a first-class option with a fixture (go-etag is the validated precedent).
38. Add a **`dupl:` commit-message convention** for extraction commits to make
    future archaeology greppable.
39. Verify **pkg.go.dev rendering** of the touched packages after the next tag
    (all new identifiers are unexported; expect zero visible delta — confirm).
40. Sweep TODO_LIST for the **StreamID identity pin** and other gates this
    session's files could theoretically affect (none expected; cheap confirm).
41. When the foreign session lands, re-run **`check-modules`** end-to-end (the
    full gate bundle incl. self-tests) — the authoritative post-concurrency
    all-clear.
42. Consider **splitting the audit handler file** if command/query symmetry
    keeps generating twin clones (handlers_audit.go is the hottest clone file
    two sessions running).
43. Investigate whether **`sort total-tokens` hides high-occurrence groups** —
    group 1 (12 occ) was the largest by count; consider a secondary sort mode
    report during audits.
44. Teach the **pre-commit hook** nothing new — confirmed no hook interaction
    this session (all commits were daemon-side); keep it that way.
45. File the **`es_setup.go:241` triple** under the accept list in
    docs/analysis if the baseline gate (item 4) lands, so the gate count
    matches reality.
46. Revisit **gotcha 20 receipt boundary**: templ components are package-level
    API even when lowercase-intent; decide if new components ever warrant
    receipts (this session: no — two lowercase, same-package only).
47. Explore **`go fix`/modernize interplay**: none of the new helper literals
    are embedded-core structs, so the exhaustruct/modernize conflict class
    (gotcha 7) should not fire — verify on the next go-fix pass anyway.
48. Consider an **`art-dupl --diff`-mode audit** against the last tagged release
    to separate pre-existing from newly-introduced clones per train.
49. Evaluate whether **loginpage/dashboardui a11y tests** should share a
    rendered-page test utility module (both render full pages via httptest) —
    cross-module, needs a shared test-package decision.
50. Book the **next train checklist line**: bundle the five touched modules'
    refactors + the bus-leak fix; the fix is consumer-visible and must not be
    tag-cut without its CHANGELOG entry riding along.

---

## g) Questions I can NOT figure out myself (3)

1. **The concurrent dashboardui session** — is the in-flight core/overview +
   formatLinksNav work (its 4 failing tests, its handlers_audit clones) yours
   or another agent's, and should I hold all dashboardui follow-ups (items 1,
   5, 14, 42 above) until it lands? I cannot know its ownership or ETA from the
   tree.
2. **Gate appetite:** do you want the dedup accept-list mechanized as a
   committed art-dupl baseline + failing gate (go-etag pattern), accepting the
   maintenance cost of re-pinning on every legitimate dedup change — or stays
   prose-only?
3. **Train routing for the two published-module extractions** (usermgmt
   find-and-marshal, setup validator helper — f-items 2-3): extract now and let
   them ride the next family train, or hold entirely until that train is being
   cut so the diff stays reviewable per concern?

---

## Session facts (for the annotator)

- Art-dupl invocation: `art-dupl --sort total-tokens -t 2 --suggest-generics
  --type-aware --timing --rich-text --html` (exit 0, type-load 1m56s).
- Full before-log: `.crush/shell-output/output-275670048.log`; after-log in
  `/tmp/art-dupl-after.log` (session-local — re-file under docs/analysis/ if
  kept, per f-13).
- Commits carrying this session's work (daemon heuristic titles):
  fbf106ca, f1a2ae5e, 762a9c7b, 9e78659b, 8830e536 (+ interleaved foreign files).
- Report format note: `.md` written per explicit operator instruction; the
  status-report skill's HTML-canonical default was overridden this once.
