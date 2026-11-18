# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-18 (14:5x, e2e browser-cache heal + migration W1.3/W1.4 landed) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**Watch the CI run on `63ebf6ae` (the completed browser forest) and verify the healed jobs stay green.**

Landed 2026-11-18 (receipts in [docs/status/2026-11-18_14-26_e2e-browser-cache-poisoning-heal-sse-hang-fix-migration-w13-w14_status.md](docs/status/2026-11-18_14-26_e2e-browser-cache-poisoning-heal-sse-hang-fix-migration-w13-w14_status.md) + the 11-25 report):

- **dashboardui suite green ×3 + `-race`**: the SSE-hang regression fixed with a never-ending-route POLICY (stream routes skipped from live probes, every probe deadline-bounded — `routes_test.go` `routeProbeTimeout`); the a11y stale wiring pin trued; hermetic lint 12 findings → 0.
- **dashboardui QUERY fallback ladder closed out** (f.32): route-postures/permission gates green, CHANGELOG entry landed, the e2e boost-guard spec modernized (pins the new `hx-post`/`hx-query`/`hx-ext` wiring + drives a REAL filtered submit).
- **The e2e environment healed + VERIFIED**: gotcha 66 (cache collision on revision NUMBER hiding an older BUILD); the flake prefers the playwright-managed builds with the completed `playwrightBrowserLibs` forest (`cups.lib`/`expat`/`libgbm`/`udev` beyond nss/nspr); **full battery 245 passed / 0 failed / 0 not-run on both engines** incl. firefox-1572 and the modernized submit.
- **Migration W1 fully executed**: W1.3 SSE-envelope parity test green (`integration_test/sse_envelope_parity_test.go`, `-race`) + W1.4 checkpoint-key inventory in `docs/guides/declarative-projections.md`; receipts in the migration plan §11; §6 trued.
- **Pushed**: a sibling pushed the 29-commit train through `110a1619`; this session completed it with `63ebf6ae` (the WORKING forest; pre-push hook green under load 1087). K2 recommendation packet filed; go-datastar#31 filed.

## Waiting on the owner

- **§g.2 (14-26 report, carries §g.2 of 11-17)** — do the M2M D1–D4 / adminui D1–D3 packets tick under this directive ("all recommendations GO" converts W1/W2 + adminui panel waves), or stay parked behind per-row ticks (the packets' own recorded reading)?
- **K2** — the recommendation is now pre-staged (`docs/proposals/2026-11-18_k2-idempotency-default-on-owner-decision.md`): one "GO" tick executes the small wave.
- **§g.3 (14-26 report)** — who provisioned the poisoned chromium-1255 cache (Nov 17 ~19:25), so the cache-collision class gets attributed.

Known standing reds: none local (full battery green 20:0x; docs gates green). CI watch in flight on `63ebf6ae`. B18 trigger-gated.
