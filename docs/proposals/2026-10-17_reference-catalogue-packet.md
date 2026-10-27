# Reference Catalogue Decision Packet — where does reference live after the README rewrite?

**Opened:** 2026-10-17 · **Tasks:** plan M12 (catalogue GO/NO-GO, owner Q3) + M13 (old-anchor deep-link decision, owner) + M14 (demo asset, owner Q1) · **Mode:** decision packet — builds stay owner-gated (plan guardrail 4).

**Erratum (plan correction, verified 2026-10-17):** the plan says to inventory `git show 8d62074d^:README.md`. That parent already carries the REWRITTEN 506-line README — the rewrite landed as daemon-captured `554c6d0f`, so the true pre-rewrite document is **`554c6d0f^:README.md` (1266 lines)**. All inventory below comes from that commit.

## 1. What the old README carried (and where it went)

| Old section (line span)                     | ~Lines | Nature    | New home after the rewrite                                        |
| ------------------------------------------- | ------ | --------- | ----------------------------------------------------------------- |
| Handler Options (179–286)                   | 108    | reference | pkg.go.dev godoc on `handler_options*.go` + README "What you get" |
| HTMXRequest struct + accessors (287–326)    | 40     | reference | pkg.go.dev (`htmx_context.go`)                                    |
| Partial rendering (327–385)                 | 59     | how-to    | README bullet + `TemplComponent` godoc                            |
| HTMX Response Builder (386–424)             | 39     | reference | pkg.go.dev (`response builder` godoc)                             |
| SSE API (471–510)                           | 40     | reference | `docs/guides/sse-and-datastar.md` + transport godoc               |
| Embedded htmx.js/extensions (510–548)       | 39     | reference | README Realtime section + godoc                                   |
| Error mapping + handlers (609–657)          | 49     | reference | README Error handling section + godoc                             |
| Middleware: CSRF fields (706–782)           | 77     | reference | `docs/guides/csrf-trusted-proxies.md` + httputil re-export note   |
| Middleware: rate limit (782–813)            | 32     | reference | `docs/guides/leveraging-httputil.md`                              |
| Security headers (813–836)                  | 24     | reference | same guide                                                        |
| usermgmt: provider injection (866–921)      | 56     | how-to    | `usermgmt/README.md` + auth sub-module READMEs (new, M05)         |
| usermgmt: service methods (941–982)         | 42     | reference | pkg.go.dev (`Service` godoc)                                      |
| usermgmt: domain events (982–1032)          | 51     | reference | Event catalog (`docs/guides/event-catalog-guide.md`) + godoc      |
| usermgmt: lockout/sessions/CSRF (1032–1103) | 72     | how-to    | `usermgmt/README.md` route posture section                        |
| Architecture tree (1103–1207)               | 105    | reference | README "The modules" (verified claims) + AGENTS                   |
| OpenAPI generation (1209–1232)              | 24     | how-to    | README openapi section + godoc                                    |

Nothing was deleted silently: every row either moved to generated godoc (cannot drift), an existing guide (curated), or the rewritten README's verified summary. The dropped tables were the README's known drift surface — the "27 factories" claim lived in exactly such a table.

## 2. M12 — `docs/reference/handler-options.md`: GO/NO-GO criteria (owner Q3)

**GO only if ALL of:**

1. **Demand:** ≥ 3 independent consumers ask for a single printable option catalogue (issue, Discord, support thread — not hypothetical).
2. **Godoc insufficiency proven:** the ask is for something pkg.go.dev cannot render — e.g. an option INTERACTION matrix (which options compose, which are mutually exclusive), not a per-option list.
3. **Drift tax funded:** a maintainer accepts that every `HandlerOption` change now touches two places. The M02 quick-start gate does NOT generalize here: options are not extracted-and-compiled, so a hand catalogue has no mechanical truth anchor.

**NO-GO while (current state):**

- pkg.go.dev renders the full option set with doc comments (verified live 2026-10-17 for v4.14.1 — report 10-41 §a-12).
- The cross-cutting concerns have curated guides (CSRF, rate limiting, SSE, error mapping).
- The historical precedent is squarely against hand tables: the old README's tables drifted repeatedly ("27 factories" being the wave's exhibit A).

**Recommendation: NO-GO today.** Revisit on the first real consumer request. If the interaction-matrix ask ever materializes, prefer a **generated** table (source-derived, `check-domain-counts`-style) over a hand-written one — generation removes criterion 3 from the equation entirely.

## 3. M13 — old-anchor deep links: stubs vs accept-the-break (owner)

Externally-plausible old headings (from the pre-rewrite structure, ranked by link likelihood):

1. `#handler-options` (the biggest reference section)
2. `#csrf-protection`
3. `#rate-limiting`
4. `#sse-server-sent-events`
5. `#htmx-response-builder`
6. `#user-management-usermgmt`
7. `#architecture`
8. `#error-mapping`
9. `#partial-rendering-htmx`
10. `#middleware`

**Recommendation: ACCEPT-THE-BREAK.** Rationale: (a) README anchors are not a released API — the rewrite existed precisely because the old page had become a drift liability; (b) GitHub has never redirected renamed anchors, so the ecosystem's baseline expectation for README deep links is fragility; (c) the correct durable link targets now exist as guides (`csrf-trusted-proxies.md`, `sse-and-datastar.md`, …) — if anything deserves stability guarantees, it is those paths, which are unchanged; (d) `<a id>` stubs would re-attach ~10 dead weights to a 510-line page and themselves rot. If the owner overrides: one `<a id="handler-options">`-style block under the License section, ~30 min, listed in this packet's amendment log.

## 4. M14 — README demo asset (owner Q1)

**Recommendation: CODE-ONLY.** The hero code fence IS the demo: one endpoint, five concerns, in declarative order — and since M02 it is **compile-gated**, so unlike a screenshot or GIF it cannot silently lie about the API. A visual asset (screenshot/GIF of the admin panel) drifts on every UI/module train, adds a binary to the repo, and sells the READY-MADE UIs rather than the library's core value (declarative wiring). If UI visuals are ever wanted, they belong in `adminui/README.md` / `dashboardui/README.md`, not the root hero.

If the owner answers YES later: source = `examples/setup-demo` admin panel (the one-call story), capture ≤ 1 MB, commit under `docs/assets/`, embed with alt text; effort ~90 min (plan F62–F64).

## Outcome

| Lane          | Recommendation                                       | Blocks on |
| ------------- | ---------------------------------------------------- | --------- |
| M12 catalogue | NO-GO today (generated-table path if demand appears) | owner Q3  |
| M13 anchors   | accept-the-break (stub block only on owner override) | owner     |
| M14 asset     | code-only (per-module READMEs if UI visuals wanted)  | owner Q1  |
