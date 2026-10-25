# Docs-truth quartet verdicts (S22/S24/S25/S26, 2026-10-25)

The four harvested docs-truth rows, each closed with evidence. Ran as the addendum's §2 quartet (q.1–q.5) after W13/W16/W17-verdicts.

## q.1 — `VERSIONING.md` claim-by-claim (S22): HOLDS

| Claim                                                                                         | Verdict | Evidence                                                                                                                    |
| --------------------------------------------------------------------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------- |
| v4 current, module suffix convention                                                          | TRUE    | root `go.mod` = `cqrs-htmx/v4`                                                                                              |
| go.work is the authoritative module list; examples/e2e/integration_test members-not-published | TRUE    | 28 `use` entries + root = 29 modules; the three carry no published tags (treadmill's 20-repo/77-prefix world excludes them) |
| Published-module list complete (10 bullets incl. the 3 auth strategies)                       | TRUE    | 14 published gate modules, all named                                                                                        |
| "Core `usermgmt` has zero auth-related dependencies"                                          | TRUE    | `usermgmt/go.mod` has zero totp/webauthn/oauth2 requires (verified 2026-10-25)                                              |
| API doc generation lives in `go-cqrs-lite/catalog/v4`, not here                               | TRUE    | catalog v4.8.0/v4.9.0 consumed by go-appkit; no docsite in-tree                                                             |
| v3→v4 / v4→v5 migration docs exist                                                            | TRUE    | `docs/migrations/v3-to-v4.md` (+ v2-to-v3, v4-to-v5)                                                                        |

No stale claims found. No edit needed.

## q.2 — `docs/DOMAIN_LANGUAGE.md` claim-by-claim (S22): HOLDS, one precision note

| Claim                                                      | Verdict         | Evidence                                                                                                                                                                            |
| ---------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 24 events (21 lifecycle + 3 auth-audit)                    | TRUE            | Events table = 24 rows; matches identity-model payload structs                                                                                                                      |
| 20 commands (UpdateRoles legacy-removed)                   | TRUE            | 21 rows − 1 legacy = 20 active; matches `Cmd*` bijection gate                                                                                                                       |
| ActorID kinds user/bot/system/service/unknown              | TRUE            | `identity-model/id.go:119-127` constants + tests                                                                                                                                    |
| Roles super_admin/admin/user/viewer/owner                  | TRUE            | `RoleSuperAdmin`/`RoleViewer` et al. in `authz_roles.go`                                                                                                                            |
| SessionOrigin sealed interface (DirectLogin/Impersonation) | TRUE            | `isSessionOrigin()` marker methods                                                                                                                                                  |
| Email `ParseEmail`/`MustParseEmail`                        | TRUE            | `identity-model/email.go:18,40`                                                                                                                                                     |
| EventBus "default `watermill.EventBus`"                    | TRUE, IMPRECISE | it is `go-cqrs-lite/watermill/v4` (the GCL package), not upstream ThreeDotsLabs — a reader could chase the wrong library; precision note recorded here rather than churning the row |

## q.3 — dashboardui↔root co-change, measured correctly (S24): 2.6% recent / 10.1% lifetime

The 2026-10-24 measurement ("0%") was a structural artifact: a pathspec-filtered `git log -- dashboardui/` can never list root files, so co-change was invisible by construction. Correct, repeatable command (commit-marker awk over the UNFILTERED log, classifying paths):

```bash
git log --no-merges --format='COMMIT%H' --name-only | awk '
  /^COMMIT/ {if (c && hasd) {tot++; if (hasr) co++}; hasd=hasr=0; c=1; next}
  /^dashboardui\// {hasd=1}
  /^[^\/]*\.go$/ {hasr=1}
  END {if (c && hasd) {tot++; if (hasr) co++};
       printf "dashboardui-touching=%d co-changed=%d rate=%.4f\n", tot, co, (tot?co/tot:0)}'
```

Results (2026-10-25): **last-500-commits window: 78 dashboardui commits, 2 co-changed = 2.6%. Full history: 663 commits, 67 co-changed = 10.1%** (early co-evolution era included). Both windows say the same thing: the UI module already evolves independently; the recent trend is further decoupling. The [`2026-10-24_module-extraction-decision-memo.md`](../modularization/2026-10-24_module-extraction-decision-memo.md) claim was corrected with these numbers (same commit).

## q.4 — modularization proposal citation (S25): CITED, not superseded

[`docs/modularization/2026-10-18_PROPOSAL.html`](../modularization/2026-10-18_PROPOSAL.html) — "29 Modules, Direction-Neutral Re-Assessment", verdict line: **"boundaries are correct — zero merges, zero splits, zero reorganizes"**, every module row Keep. Read again 2026-10-25: still accurate against the current tree (module set unchanged; the 2.6%/10.1% co-change numbers REFRESH its coupling evidence rather than contradict it). The proposal stays the standing modularization reference; nothing supersedes it.

## q.5 — two-repo-by-audience memo (S26): ALREADY DECIDED — KEEP ONE REPO

The decision already shipped 2026-10-24 (stronger than the master plan's no-verdict default): ROADMAP "Not Planned" links [`docs/modularization/2026-10-24_module-extraction-decision-memo.md`](../modularization/2026-10-24_module-extraction-decision-memo.md), decided **KEEP ONE REPO** on (1) the direction-neutral KEEP-ALL re-assessment, (2) the co-change evidence (number corrected this session, verdict unchanged), (3) the structural cost of splitting a 29-module family-train repo. This quartet pass verified the link resolves and refreshed the evidence; no new memo needed.

---

_Quartet closed 2026-10-25 (addendum §2, q.1–q.5). Verify: `nix run .#check-docs-links` green; corpus baseline bumped in the same commit._
