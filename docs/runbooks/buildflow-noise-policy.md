# BuildFlow Noise Policy — signal-only hook output

**Established:** 2026-10-01 (T24). **Scope:** the BuildFlow pre-commit/full runs in THIS repo. **Rule:** every recurring non-gating finding class gets ONE documented disposition here; anything not listed below is signal and must be triaged, not muted. Re-audit this table whenever BuildFlow's step set changes.

**Gate posture:** `fail_on: critical` (`.buildflow.yml:152`) — only critical findings block. Everything below is non-gating by design.

## Dispositioned classes (verified 2026-10-01 pre-commit run + `buildflow doctor`)

| Class | Shape | Disposition |
| --- | --- | --- |
| `cqrs-lint` ~195 findings | INFO/WARNING architecture advisories | Dispositioned per class in [`docs/research/2026-10-01_cqrs-lint-residual-triage.md`](../research/2026-10-01_cqrs-lint-residual-triage.md) (A013 accepted, V007/A012 v5-window, E005 documented FP + upstream proposal, V006 suppressed at source, B024 documented FP). Not noise — decisions. |
| erraudit `context_loss` | ERROR severity | Program closed 2026-10-01; regenerations caught same-day (see triage note §context_loss). `fail_on: critical` never saw these; treat any reappearance as signal. |
| `type-check` tsconfig | was: ERROR `moduleResolution=node10` removed in TS7 | **FIXED 2026-10-01** (commit `0d270638`). If this row ever fires again, the e2e tsconfig regressed. |
| `go-structure-linter` 5 findings | internal/ + assets/ dir advice; 3 `*_stub_test.go` testdata advice | Accepted: repo layout is deliberate (multi-module root; stub files are compile-only interface fakes). |
| `dependabot-auto-configure` 1 finding | "28 modules; generation capped at 20" | Superseded-by-design: dependabot deliberately watches root/usermgmt/integration_test only (2026-09-20 posture comment in `.github/dependabot.yml`); the rest ride the trains. |
| `nix-checker` 4 findings | pinned `hash =` info, `vendorHash` warning, "extract hash to file" suggestions | Accepted: intentional pins; hash extraction is style churn with no benefit here. |
| `pytest-test` 5 findings | "collected 0 items" warnings | Expected: no Python tests exist; the step no-ops. |
| `vulnix` N findings (was: dead-feed crash) | CVE advisories on nix store derivations (binutils/bison/coreutils/gcc) | Watch class: these are nixpkgs-level, patched by nixpkgs bumps, not by this repo. The DEAD NVD feed crash (404 on `nvdcve-2.0-modified.json.gz`) self-recovered by 2026-10-01 ~17:00 — if vulnix errors again with feed 404s, disposition = wait out the upstream feed, do not mute the step. |
| "9 tools unavailable (health check failed)" | ambient-shell runs outside the devShell | Environment-class (gotcha 8): run gates via `nix run`/devShell where the tools exist; never mute per-tool. |
| PostHog 403 "Invalid or missing CSRF token" | BuildFlow telemetry, network-blocked | External telemetry; zero repo impact. |
| timings "regression verdict" list | first run after reboot/cache-clear shows ms→seconds swings | Measurement artifact of cold caches; ignore unless reproducible warm. |
| `go-auto-upgrade` ~500 suggestion findings | full-run only (skipped in pre-commit): adopt `lo.*`, testify→stdlib suggestions | Accepted noise for THIS repo's conventions (std-lib-first is deliberate where it is; `lo.*` adoption is a ROADMAP-level style call, not a step finding). Dispositioned 2026-10-01 (T24); revisit only if the step learns a severity model. |

## Adding a new class

1. Verify the finding class is real, recurring, and non-gating (three occurrences or an explicit design decision).
2. Add a row here with the disposition AND the reason it must not be muted at the tool level.
3. If the class is actually a bug → fix it, don't document it (the tsconfig row is the precedent: fixed same day, kept as a tripwire).
