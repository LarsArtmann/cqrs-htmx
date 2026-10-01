# Proposal: cross-module command-registration awareness for cqrs-lint E005

**Status:** draft for owner review — cross-repo (go-cqrs-lite); nothing filed or implemented without owner approval.
**Author:** session 2026-10-01 (residual triage, `docs/research/2026-10-01_cqrs-lint-residual-triage.md`).
**Target:** `cmd/cqrs-lint` in `github.com/larsartmann/go-cqrs-lite`. Verified against cqrs-lint `4.13.1-0.20261001211312-b06ac8add5e8`.

## The finding

cqrs-lint's E005 (`command-without-handler`, WARNING, architecture) fires 16 times in this repo — every one on `identity-model/commands.go` command types:

> `WARNING identity-model/commands.go:12:6 Command type "RegisterUserCmd" has no registered handler — dispatching it will return ErrNoHandler` (×16: RegisterUserCmd, ChangeEmailCmd, ChangeDisplayNameCmd, AddCredentialCmd, RemoveCredentialCmd, VerifyEmailCmd, EnableTOTPCmd, DisableTOTPCmd, LinkExternalAccountCmd, UnlinkExternalAccountCmd, AddMemberCmd, UpdateMemberRolesCmd, RemoveMemberCmd, CreateTenantCmd, ReactivateTenantCmd, RegisterBotCmd)

All 16 are false positives, and the repo's architecture GUARANTEES they are:

- `identity-model` is the pure domain module by design (ADR-0111 era): it defines the 20 command types and their fold/state machinery, and deliberately owns NO handlers.
- The handlers (deciders) live one module up in `usermgmt`, registered through the typed-command registry; the system composition root (`systemadapter` → `system.New(DomainConfig(...))`) registers all 20 via `RegisterTyped` — a bijection mechanically enforced in THIS repo by `scripts/check-command-bijection.sh`.

## Why the linter can't see it

E005's collector analyzes per-module (the gate runs each module with `GOWORK=off`, loading from its own go.mod). Handler registration happens in a DIFFERENT module (usermgmt, systemadapter), so within identity-model's package graph no `Register*` call mentions these types and E005 concludes "never registered".

## Proposed fix (in cqrs-lint, not in consumers)

Teach the E005 collector a cross-module fail-open, in ONE of these shapes (best first):

1. **Workspace-aware registration scan.** When analysis runs under a go.work (or with a `--cross-module` flag), resolve `Register*`/`RegisterTyped` calls across ALL workspace members before deciding "unregistered". A command type with zero registration calls across the UNION graph is a true positive; otherwise suppress.
2. **Domain-purity exemption.** A module that DEFINES command types but imports no dispatcher/decider interfaces (import-graph fact, cheap to compute) is a *domain definition module*; E005 should not fire there — the owning composition root is where the check belongs. This generalizes: the same module-classification kills other definition-vs-usage FPs.
3. **Directive escape hatch** (`//cqrs-lint:ignore(E005) registered in usermgmt/systemadapter`) — works today but costs 16 comment lines and rots silently; listed for completeness, NOT recommended.

Shape 2 is the cheapest correct one; shape 1 is the most precise. The C040 fix (const/var-alias resolution, landed 2026-09-30 in this same collector family) is the template for how collector-side resolution beats consumer-side suppression.

## Evidence pack for the filing

- 16 findings: `cqrs-lint --strict --verbose .` in `identity-model` with the version above.
- Registration proof: `bash scripts/check-command-bijection.sh` (repo-side mechanical proof that all 20 command types are registered).
- Dispatch proof: `integration_test/actor_attribution_test.go` and the usermgmt service tests dispatch these commands end-to-end; a real `ErrNoHandler` would fail those long before the linter's warning matters.
