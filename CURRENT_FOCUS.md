# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-18 (23:2x, CI all-green on the healed train — no active thread) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**None — waiting on the owner.** CI run `44397985522` on `5321c847` came back ALL GREEN: `e2e`, `build`, `test`, `mod-tidy`, every `lint (module)` leg, `module-architecture`, `security`, and `checks`. That verdict settles the stretch's last open question: the codegen gate greens on the `nix run .#gen` (generated-AND-formatted) templ form — the format war's canonical-form question is answered by evidence, and the never-bare-generate rule landed as gotcha 67. Tree clean, pushed 0/0.

The day's receipts live in [the 14-26 report](docs/status/2026-11-18_14-26_e2e-browser-cache-poisoning-heal-sse-hang-fix-migration-w13-w14_status.md) and [the 22-05 report](docs/status/2026-11-18_22-05_ci-shepherding-codegen-format-war-battery-245-green_status.md): the e2e browser-cache heal + full 245/245 both-engine battery, the dashboardui SSE-hang fix + QUERY-ladder close-out, migration W1.3/W1.4 executed, the K2 packet and go-datastar#31 filed.

## Waiting on the owner

- **§g.2 (22-05 report, carries the 14-26 one)** — do the M2M D1–D4 / adminui D1–D3 packets tick under "all recommendations GO" (converts W1/W2 + the adminui panel waves), or stay parked behind per-row ticks? One word per row converts ~2 days of work.
- **K2** — one "GO" tick on [the packet](docs/proposals/2026-11-18_k2-idempotency-default-on-owner-decision.md) executes the ~2–4 h default-mount wave.
- **§g.3** — who provisioned the poisoned chromium-1255 cache (Nov 17 ~19:25)? Attribution closes gotcha 66's loop.

Known standing reds: none local, CI green on HEAD. Quiet-window carryovers (deferred under 7 live sibling sessions, load ~52/32 cores): the erraudit advisory TOTAL re-true and one local `check-modules` composite run — CI's `checks` job carries both verdicts on HEAD meanwhile.
