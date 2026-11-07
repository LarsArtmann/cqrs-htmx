# Status — htmx "latest version" gap analysis + session self-review

**2026-11-07 18:49 CET · session scope: the "update to the LATEST htmx version" question only.**

**Scope line (verified vs not).** This report covers ONE exploration turn plus this report's own
hygiene descent. Verified with live commands this session: the embedded asset inventory (sha256 +
byte sizes of both htmx bundles and all three extensions), npm dist-tags (`htmx.org` `latest` =
**2.0.11**, published 2026-09-22; `next` = **4.0.1**), the upstream 2.0.11 changelog delta, every
`2.0.10`/`4.0.1` pin site in the repo (code, tests, comments, docs), the absence of byte/sha pins in
Go tests, and the tail/corpus ratchet state. NOT verified: the 2.0.11 asset bytes themselves (metadata
only — never downloaded), the upstream latest of the two vendored extensions, whether the browser e2e
harness loads the embedded 2.0.10 bundle, and diff-level behavior of 2.0.11 internals (risk was
inferred from changelog titles). Zero code mutations happened before this report.

## Embedded vs upstream (measured this session)

| Asset | Embedded | Upstream latest | Gap |
| --- | --- | --- | --- |
| `htmx/htmx.min.js` (default) | **2.0.10** (72,513 B, sha256 `5d13384a…`) | **2.0.11** (npm `latest`) | one patch |
| `htmx/htmx4.min.js` (`ScriptHandler4`) | **4.0.1** (37,448 B, sha256 `a0abb04f…`) | 4.0.1 (npm `next`) | none — current |
| `htmx/extensions/sse.min.js` | 2.2.4 (`7a407857…`) | **unknown — not checked** | open |
| `htmx/extensions/idiomorph-ext.min.js` | 0.7.4 (`ea820c1e…`) | **unknown — not checked** | open |
| `htmx/extensions/cqrs-query.js` | 1.0.0 (own) | n/a | — |

Upstream 2.0.11 delta (from the official CHANGELOG): `<hx-partial>` backport, `hx-push-url="false"`
ancestor fix, `hasHeader()` TypeError fix, detached-context OOB target fix, `Object.keys()` iteration.
Default-flip to 4.x stays v5-gated (ROADMAP q35 / decision D18-D1, ADR-0062 one-breaking-window).

## a) FULLY DONE

1. **The question answered with verified data, not memory**: npm registry fetched live (twice — the
   full doc truncated, re-fetched per-version for 2.0.11); dist-tags and the 2.0.11 publish metadata
   (gitHead, unpkg dist path) confirmed from the registry response.
2. **Complete pin-site map for the bump** — every site that mentions 2.0.10, enumerated by file:line:
   `htmx/embed.go:6` (the const), `htmx/serve_test.go:36-37,99,116,191,218` (version/ETag table,
   `Version()` golden, `CDNScriptTag` unpkg-URL golden), `example_app_test.go:101`,
   `example_htmx_test.go:118` (`// Output:` goldens), `adminui/assets.go:32`, `README.md:88`,
   `FEATURES.md:66` (also carries a stale "~51KB" size claim), `examples/basic/README.md:14`,
   `docs/guides/http-query-method.md:132`, SKILL.md, CHANGELOG precedent (`2.0.9 → 2.0.10` entry).
   `api_surface_compat_test.go` pins the signature only — no change needed.
3. **Confirmed no mechanical byte pins**: no test asserts size/sha256 of the 2.x bundle (only
   CHANGELOG prose does); ETag derives from the version const; `root-files.baseline` unaffected
   (in-place file update); `serve4_test.go` pins 4.0.1 and stays untouched.
4. **Upstream changelog fetched and the 2.0.11 delta enumerated** (five fixes, listed above).
5. **Policy guardrail applied**: 4.x-as-default identified as v5-gated before proposing anything that
   would violate ADR-0062's one-breaking-window rule.
6. **Skill discipline**: cqrs-htmx SKILL.md loaded before any task work.

## b) PARTIALLY DONE

1. **Extension freshness**: named the SSE/idiomorph assets as a separate concern but deferred WITHOUT
   data — their upstream latests were never fetched (the "unknown — not checked" rows above).
2. **e2e consumer check**: grepped `integration_test/` + `e2e/` for consumers (found
   `e2e/server/main.go`, two datastar tests referencing ScriptHandler) but never OPENED them to
   confirm whether a browser e2e exercises the embedded 2.0.10 bundle.
3. **Risk assessment of 2.0.11**: stated "none of the fixes touch the `ne()` inheritance resolver"
   from changelog TITLES alone — a reasonable inference, not a verified bundle diff.

## c) NOT STARTED

1. **The bump itself** — zero files changed. Awaiting owner go (see §g).
2. **Tail/corpus ratchet repair** — found red on arrival while writing this report; the descent is
   executed as part of this report's landing (3 superseded reports archived, TAIL_BASELINE lowered,
   CORPUS_BASELINE bumped with ledger row).

## d) TOTALLY FUCKED UP

1. **Nothing destroyed this session** (zero mutations before this report). But found on arrival,
   NOT authored by this session: **both docs ratchets red** — tail 16 > baseline 15 and corpus
   56,345 > 56,280 (+65), shipped by the 18:35 closeout report without its hygiene. Had the user
   said "execute the htmx bump" immediately, this session would have stacked commits onto a
   known-red tree without noticing — the red was only discovered because the user demanded this
   report.
2. **Self-critique, honestly owned**: the first answer presented the changelog-title risk inference
   with more confidence than it earned, and closed with "Want me to execute it?" without flagging
   the two unverified legs (asset bytes, extension latests) as explicit pre-execution gates.

## e) WHAT WE SHOULD IMPROVE

1. **verify-external-claims discipline at answer time**: when the deliverable is an update plan,
   download and hash the target asset in the same turn — bytes beat metadata.
2. **Answer completeness**: "latest X" questions should sweep the asset FAMILY (core + vendored
   extensions) in one pass; deferring half the family without data is a gap the reader can't see.
3. **Session-start gate check**: tail + corpus ratchets are one command each; run them at session
   START, not when a report forces it. The 18:35 session shipped red; this session nearly did.
4. **Read, don't grep**: opening `e2e/server/main.go` costs one tool call and converts "probably
   fine" into a verified claim.
5. **Confidence labeling**: inferences (changelog-title risk reads) should be marked as inferences
   in the answer itself, with the diff-verification step named as a gate.
6. **Status-report hygiene is load-bearing**: the report ritual (tail descent, corpus ledger) is the
   only thing that caught today's pre-existing red — keep treating it as mandatory, not paperwork.

## f) Next things to get done (ranked; 1–12 are the htmx thread)

1. Owner decision: execute the 2.0.10 → 2.0.11 bump now or hold for the next wave (§g q2).
2. Vendor `htmx.org@2.0.11/dist/htmx.min.js` into `htmx/htmx.min.js`; record size + sha256.
3. `htmx/embed.go:6` const bump (ETag becomes `"htmx-2.0.11"` automatically).
4. Update `htmx/serve_test.go` pins (5 sites).
5. Update the two example goldens (`example_app_test.go`, `example_htmx_test.go`).
6. Sweep the ~8 comment/doc sites incl. SKILL.md, README, FEATURES (fix the stale "~51KB" size claim
   in the same pass).
7. CHANGELOG entry (follow the 2.0.9 → 2.0.10 precedent).
8. Diff old-vs-new bundle line by line (the family-bump golden discipline) — verifies the `ne()`
   inheritance behavior the dashboardui gotcha depends on, closing the inference gap.
9. Run root + adminui + dashboardui suites (UI goldens exercise the swap engine).
10. Run `nix run .#check-modules`.
11. Open `e2e/server/main.go` + the W2.4 browser battery: confirm whether they pin or load 2.0.10.
12. Check the extensions' upstream latests (htmx-ext-sse vs 2.2.4, idiomorph vs 0.7.4); sweep or
    record a verdict.
13. Add the consumer cache note (1-year immutable → stale 2.0.10 until reload) to the bump CHANGELOG.
14. TODO_LIST row for the bump (none exists today — verified against the D18 row only).
15. ROADMAP q35: consider writing an explicit trigger for the v5 4.x-default flip (e.g. "htmx 4.x
    minimum browser matrix + `hx-partial` usage demand").
16. §f of the 18:35 closeout report: harvest any unharvested rows (it shipped red; its §f was never
    checked by this session — flagged, not verified).
17. Docs-gates battery local run (`nix run github:Artmann-Technologies/docs-gates -- run .`) after
    this descent — the audit-all pin (322) vs live 393 red predates this session (AGENTS.md P2 row).
18. Audit-all drain-or-re-pin decision (the TODO_LIST P2 top row; policy forbids raising the pin
    without an owner tick).
19. `examples/catalog-demo` workspace-red — heals when go-cqrs-lite ships a templ-era docserver
    release (A25 upstream watch, TODO row 51b); recheck the tag.
20. Go-etag v0.7.0 chore bump (the one fleet-audit lag, noticed in AGENTS.md context).
21. Verify the 18:35 report's own §f items landed in TODO_LIST (split-brain check between that
    report and the living doc).
22. erraudit advisory month: M01 verdict-5 blocking promotion still pending its green month.
23. Dependabot watch re-run after the next push (`nix run .#check-dependabot-alerts`).
24. If the bump lands: `nix run .#train-preflight` before the tag wave (mandatory since S52).
25. If the bump lands: decide ride-along train (v4.19.0) vs standalone patch tag.
26. Sweep `docs/research/2026-11-07_datastar-client-side-deep-dive.html`'s htmx 2.0.10 size table
    (72,513 B) — it is a research artifact; leave as history, but note the drift source.
27. Consider pinning the embedded bundle sha256 in a test (like serve4_test pins content markers) so
    silent asset drift fails CI — currently only the 4.x bundle has a content pin.
28. The `cqrs-query` extension's 2.0.10-coupled internals (`methodsThatUseUrlParams` proof in the
    QUERY plan §NV5) — re-verify the proof against 2.0.11 bytes when vendoring (§f item 2).
29. Re-run `nix run .#check-readme-badges` after the next root tag (badge verdicts lag tags).
30. Session-start gate check as a habit: tail + corpus + conflict-markers (one line, ~5s) before any
    mutation — propose adding to `docs/runbooks/session-start.md`.

## g) Questions I cannot figure out myself

1. **"LATEST" intent**: I read your question as the npm `latest` tag (2.0.11 patch bump of the 2.x
   default). If you meant "make htmx 4 the default", that is the v5-gated flip (D18-D1/ROADMAP q35) —
   which did you want?
2. **Timing**: execute the 2.0.11 sweep as a standalone patch now, or bundle it into the next
   v4.19.0 wave (one train, one CHANGELOG)?
3. **Scope**: sweep the two vendored extensions (SSE, idiomorph) to their latests in the same change
   (core-only is the leaner alternative), noting their latests are currently unverified?

---

*Report ritual executed with this file: 3 superseded reports archived (M2M audit + M2M design answer
→ superseded by the 21-28 M2M W0 execution report; templ-era conflict → superseded by the 05-20
v4.17.0 family-train report), TAIL_BASELINE lowered to the measured tail, CORPUS_BASELINE bumped to
the measured corpus with a ledger row in `docs/status/README.md`.*
