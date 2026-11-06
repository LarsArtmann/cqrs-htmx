# ADR 0062: Direct Imports Only

**Status:** Accepted
**Date:** 2026-11-06
**Driver (owner):** "I kind of hate re-exports. People should import the code as directly as
possible. Only refactoring + deprecated aliases may temporarily be okay."
**Related:** [ADR 0047](0047-re-export-layer-retirement-plan.md) (the retirement plan this ADR
gives a policy and a gate), [v5 Removal Inventory](../guides/v5-removal-inventory.md),
[SUPERB plan](../planning/2026-11-06_07-16_SUPERB-direct-imports-re-export-eviction-and-root-burndown-pareto-plan.md),
[`reexport-baseline.tsv`](../../scripts/checks/reexport-baseline.tsv).

## Context

The 2026-11-02 subpackage extraction (`htmx`, `httperr`, `eventcatalog`) moved real code out of the
root package, then re-exported every moved symbol back into `cqrshtmx` through three alias files.
`htmx_reexport.go:15-19` documented the result as deliberate: *"The root package keeps these
aliases as its supported surface; they are not deprecated."* The compatibility bridge shipped as
the permanent API.

Measured on 2026-11-06 by `scripts/checks/reexport_scanner.go` (the tool this ADR adds):

| Metric                                       | Value |
| -------------------------------------------- | ----- |
| Top-level alias declarations, whole workspace | 418   |
| …already carrying `// Deprecated:`            | 162   |
| …carrying no deprecation marker               | 256   |
| Files containing at least one alias           | 47    |
| Root package only: alias LOC / declarations   | 718 / ~315 |

Two costs follow from keeping them un-deprecated:

1. **Two live spellings of one API forever.** Every root-level façade user and every direct-import
   user must both keep working, so the façade can never be deleted and the root package can never
   shrink. `api_surface_compat_test.go` exists solely to pin those aliases in place.
2. **A split brain caused by file placement.** `authz_contract.go` imports only `errors`, `fmt`,
   `net/http` and `errorfamily`, yet it sits at the top of the dependency stack because it lives in
   the root package. `identity-model` therefore cannot declare the permission contract, and the
   compile-time assertion that `Authz.Check` satisfies `PermissionChecker` has to live in
   `usermgmt` — a module that only sees both by accident.

A policy that is not mechanised is decoration: the same façade would be recreated by the next
extraction, because recreating it is the path of least resistance.

## Decision

1. **A cross-package alias is never the supported API.** The supported API is the package that
   owns the implementation. Consumers import that package.
2. **Every re-export declaration carries `// Deprecated:`** naming the direct import and the
   release that removes it. `// Deprecated:` is the only marker Go tooling and staticcheck honour;
   a doc sentence saying "kept for compatibility" is not a marker.
3. **No new alias file, ever.** A move ships with repointed consumers, not a shim. `*_reexport.go`
   is a closed set: it may only shrink.
4. **Exactly one breaking window per major version.** v5 is the only cut for this generation.
   Anything that cannot ride it waits for v6 — it does not get a private breaking release.
5. **Exemptions are written down, dated, and audited.** Four exemptions exist and each names why
   the alias is not a second import path:

   | Class                      | Meaning                                                                                                                       |
   | -------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
   | `exempt:unexported`        | lowercase plumbing inside one package; not API, cannot be imported, so nothing is duplicated                                   |
   | `exempt:internal-target`   | the target lives under `internal/` and is unimportable by consumers, so the alias **is** the only definition surface           |
   | `exempt:domain-vocabulary` | the declaring package owns the constructors/parse surface for the type; the alias spells a domain name, not a second import path |
   | `exempt:pending`           | a candidate re-export layer whose disposition is decided in a later wave; frozen in size, may not grow                        |

6. **Mechanisation.** `scripts/checks/check-reexport-policy.sh` enforces the policy against
   `scripts/checks/reexport-baseline.tsv`, with a fixture self-test, wired into
   `nix run .#check-modules` and CI:
   - **rule 1** — no alias file outside the baseline; no file may hold more aliases than its
     baseline row (a ratchet: aliases may shrink, never grow);
   - **rule 2** — every alias in a `reexport`-class file is `// Deprecated:` and names its target
     package;
   - **rule 3** — every file holding an alias is classified in the baseline.

## Consequences

- Root's three façade files (`htmx_reexport.go`, `httperr_reexport.go`, `eventcatalog_reexport.go`)
  are deprecated immediately (v5 removal class 1b) and become a pure delete at the v5 cut; so do
  the already-dated `httputil`/`go-sse`/`identity-model` layers (classes 1, 2, 3).
- The root package's own code stops using its own façade, so v5 deletion is a file delete plus a
  consumer repoint, not an archaeology exercise.
- `cmd/cqrs-htmx-upgrade` gains derivation rules for the three façade files, so the consumer
  migration is mechanical rather than manual.
- `api_surface_compat_test.go` loses its alias half at v5; the non-alias half (genuine root API)
  stays.
- Adding a subpackage no longer implies adding an alias file: the next extraction is a real move.
- The 47-row baseline is the audit surface. Growing it requires an ADR amendment; shrinking it is
  the normal direction of travel.

## Reopen trigger

Reopen if a consumer demonstrates a symbol that has **no** direct-import path — for example a type
needed to satisfy an interface declared in a third module that cannot import the owning subpackage.
Such a case is an argument for moving the interface, not for restoring a permanent alias.
