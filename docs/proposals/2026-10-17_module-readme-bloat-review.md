# Module README Bloat Review — usermgmt & setup (Decision Packet, NO edits)

**Opened:** 2026-10-17 · **Task:** plan M21 (`docs/planning/2026-10-17_09-03_SUPERB-readme-followthrough-docs-truth-pareto-plan.md`) · **Mode:** measurement + verdict only — no README was modified by this packet.

**Question:** did the 1266→510 root README rewrite push bloat into the module READMEs, or leave behind module READMEs that violate the sales-page rubric?

## Rubric (sales-page)

A module README earns its bytes by answering, in order: **what is this → why would I care → how do I start → where do I go next**. Reference tables (field lists, method inventories, option catalogues) are NOT sales-page content: they belong in pkg.go.dev (godoc is generated, hence cannot drift) or a dedicated reference doc that something mechanically pins.

## Measurements (2026-10-17, live tree)

| Metric            | `usermgmt/README.md` | `setup/README.md`     |
| ----------------- | -------------------- | --------------------- |
| Lines             | 88                   | 468                   |
| `##` sections     | 5                    | 14                    |
| Headings (total)  | 12                   | 14                    |
| Table rows (`\|`) | 0                    | 79                    |
| Code fences       | ~6 (short snippets)  | ~10 (wiring + config) |
| Largest section   | Event signing (35 l) | Configuration (154 l) |

## Verdicts

### usermgmt/README.md — **KEEP** (verdict: not bloated)

88 lines is inside the rubric. The structure is description → install → opt-in signing/encryption → route posture → development → license: the signing/encryption sections are the module's differentiator (consumers must actively opt in), not reference ballast. The one honest gap is the missing `EventHandler` consumer-docs section — already its own open TODO row (opened 2026-10-17), estimated +15 lines when written, still inside budget. No restructure warranted.

### setup/README.md — **KEEP** (verdict: the "bloat" is load-bearing reference, deliberately placed)

The 468 lines split into ~90 lines of sales page (What it does / Quick start / What you get) and ~380 lines of **canonical reference**: the Configuration table (~154 lines), capability floor, troubleshooting, TLS posture. That reference is cited as canonical by AGENTS.md ("thresholds live in flake.nix… the canonical table"), by the root README, and by guides — moving it would break more links than it tidies. pkg.go.dev cannot host this content: it documents types, not the _wiring narrative_ (which dependency unlocks which route, what the TLS trade-offs are).

**Split criteria (revisit triggers, not current work):** split into a ~90-line sales README + `docs/reference/setup-config.md` ONLY if (a) the file crosses ~700 lines, (b) the config table needs its own change cadence (per-field changelog), or (c) a second module grows an equal-sized config surface and a shared reference shape emerges. Estimated effort when triggered: ~2 h + anchor stubs for external deep links.

## Adjacent rows (already tracked, not duplicated here)

- `identity-model` README completeness review — open P3 row (opened 2026-10-17).
- usermgmt `EventHandler` docs gap — open P2 row (opened 2026-10-17).
- README TOC decision — open row (revisit ~2026-11-17).

## Outcome

**No edits.** Both READMEs keep their current shape; the revisit triggers above are the only future work this packet authorizes, and they are criteria, not tasks.
