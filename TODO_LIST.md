# TODO List — cqrs-htmx

**Updated:** 2026-07-20 | **Coverage:** 94.0% root (incl. openapi 99.0%), 80.2% usermgmt (CI gates: root 90%, usermgmt 74% — see `nix run .#coverage-gate`) | **Lint:** 0 issues across all submodules (usermgmt, totp, webauthn, oauth2, adminui, loginpage, integration_test). Root has 75 remaining issues, all pre-existing low-severity style nits (varnamelen ×49 on short id names like `eh`/`ch`/`rw`, testpackage ×9 on internal-package tests, plus scattered ireturn/makezero/nonamedreturns/wsl_v5/tagliatelle/containedctx/testableexamples). No depguard, no err113, no mnd, no canonicalheader, no exhaustruct, no wrapcheck issues remain. | **Version:** v4.3.0+unreleased (go-cqrs-lite v4.0.x; see AGENTS.md for per-sub-module versions)

## Status Legend

- [ ] OPEN — actionable, not yet started
- [~] PARTIALLY DONE — started but incomplete

> Completed items live in [CHANGELOG.md](CHANGELOG.md). Deferred and rejected ideas live in [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## Open Items

### P3 — Technical Debt & Future

- [ ] **MySQL support** for event store (currently Postgres + SQLite only)

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
