# Release Runbook — the v5 Cut

**Status:** skeleton (2026-10-05, plan A10/M15). The wave mapping and the
pre-cut gates are decided; per-class consumer notes are at inventory depth
and grow as the cut approaches.
**Precondition:** ROADMAP OQ11 (v5 timeline) is an owner decision — this
runbook executes once that lands, not before.
**Inputs:** [v5-removal-inventory](../guides/v5-removal-inventory.md) (the
per-item source of truth — this file never duplicates its content, only
sequences it), [release-playbook §3a](../guides/release-playbook.md)
(wave choreography), [release-next-train-prep](release-next-train-prep.md)
(the standing train script).

## 1. Wave-ordered deletion plan

Tag waves follow playbook §3a order; each class lands in the wave of the
module that owns it. Within a wave, deletions are one commit per module so
`verify-tag` can verify each tree independently.

| Wave | Modules                                                              | Removal work (inventory class)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1    | auth strategies (totp/webauthn/oauth2)                               | nothing — ride-only, re-tag after their deps move                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| 2    | identity-model                                                       | class 4: delete `NewUserID` (SHA-256-hashing constructor)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| 3    | root + usermgmt                                                      | class 1: delete `*_reexport.go`/`sse_*.go`/`event_store_sse.go`/`security.go` shim (root) · class 2: delete `Raw`/`NewBroadcasterFromRaw`/`RawBroadcaster` (root) · class 3: delete the ~161-symbol alias layer, 22 files (usermgmt — constructor WRAPPERS stay) · class 4: SQL view-API migration at the same cut + idempotency-default decision (usermgmt/root) · class 5b: V007 stack surface — `es_materialize_adapter.go`, `buildStackRepositories`, `Bundle` field, the three `//go:build ignore` SQL templates + `sql_setup_shared.go` (usermgmt; cluster-1 SQL read models ONLY if the ADR-0051 upstream AND-gate went green — otherwise they ride v4 trains unchanged, §5c) |
| 4    | adminui/dashboardui/loginpage/datastar/health/auditlog/systemadapter | class 2: delete `datastar/broadcaster.go` facade + drop the `go-datastar/broadcast` require (datastar, budget 6→5) · class 4: delete `NewProjectionLayer` (systemadapter — only with no consumer examples left on the legacy path; `examples/system-demo` migrates this wave) · class 5 (if decided): cut renamed module paths `identityadmin`/`esdashboard` via verify-tag; old paths become doc-only shims for ONE cycle                                                                                                                                                                                                                                                           |
| 5    | setup                                                                | re-tag after renames (its requires name the UI module paths); no deletions                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| 6    | examples + e2e + integration_test                                    | sweep last consumer references; `examples/system-demo` off `NewProjectionLayer`; docs sweep                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |

Module-renames caveat (class 5): if either rename is rejected, the docs
mitigation (cross-linked contrast sections, landed 2026-09-17) becomes the
permanent fix and wave 4 carries only the facade/ProjectionLayer work.

## 2. Consumer migration notes per class

Per class, what a downstream consumer does instead (expand inline per class
as the cut approaches; the detailed tables already live in the linked docs):

1. **Root httputil/SSE re-exports** — import `github.com/larsartmann/httputil`
   and `github.com/larsartmann/go-sse` directly; migration tables:
   [leveraging-httputil.md](../guides/leveraging-httputil.md).
2. **Raw-named broadcaster API** — use `Hub()` + `NewBroadcasterFromHub`, or
   pass the `*sse.Broadcaster[sse.Event]` hub directly; both adapters embed
   it so `Subscribe`/`SubscribeFilter`/`Close`/`OnSubscribe` promote. DataStar
   consumers import `github.com/larsartmann/go-datastar/broadcast` directly.
3. **usermgmt alias layer** — import
   `github.com/larsartmann/cqrs-htmx/identity-model/v4` directly; the aliases
   were transparent type aliases, so this is an import-line change.
   `usermgmt.NewExternalAccount` and the other constructor WRAPPERS remain.
4. **Behavior-bearing constructors** — `NewUserID(s)` → `ParseUserID`
   (strict), `SyntheticUserID` (explicit stable hash), or `GenerateUserID`
   (fresh ULID); SQL view-API call sites move to the record/listing view APIs
   (the ADR-0123 nolint comments name the sites); `NewProjectionLayer` → the
   declarative path (`DomainConfig.Projections` +
   `ProjectionTypeDecoder`,
   [declarative-projections.md](../guides/declarative-projections.md)) —
   consumers needing durable checkpoint/DLQ stores wait for the upstream
   `system.New` option (§Accepted limitation) and the migration guide carries
   the caveat if the cut proceeds first.
5. **Module renames** — change the import path; a one-cycle deprecated shim
   keeps the old path compiling (doc-only, no forwarding surface).
   5b. **V007 stack surface** — migrate to `NewEventSourcedSetup` (raw store+bus)
   or the systemadapter declarative path.

## 3. Pre-cut gate checklist (ALL green before wave 1)

Standing train gates (every wave, per playbook §3a):

- [ ] `nix run .#preflight-tree-check` + `.#wait-tree-quiet` before the train
- [ ] per-wave `scripts/tools/verify-tag.sh <module-dir> <version>` — never raw `git tag`
- [ ] per-wave `bash scripts/checks/check-release-train.sh` strict (0 unpublished / 0 lag) BEFORE the next wave
- [ ] version-drift strict at every push (pre-push hook enforces; fresh-tag TTL ghost → `-- --refresh-cache`)
- [ ] proxy smoke (scratch `go get` of each new tag) after the push

v5-specific gates:

- [ ] `nix run .#build` + `.#test` + `.#lint` workspace-wide after ALL deletions (gotcha 2: root `./...` is a false green)
- [ ] `bash scripts/checks/check-module-isolation.sh` per touched module (hermetic, GOWORK=off)
- [ ] docs sweep greps return ZERO: `cqrshtmx.CSRF|cqrshtmx.SecurityHeaders|Broadcaster.Raw|NewBroadcasterFromRaw|usermgmt.User.*Payload|NewUserID\(` across README/AGENTS/guides/examples — then `nix run .#check-docs-links`
- [ ] the `cqrs-htmx-upgrade` codemod (Track A, TODO 58) round-trips its golden consumer fixture against the post-cut tree
- [ ] Track B data-compat goldens (TODO 58) still decode real v4.13 journals on the v5 tree (the backwards-compat contract)
- [ ] CHANGELOG version headers cut BEFORE tagging (train step 3.7 convention); every class gets a `### Removed` entry with its migration note
- [ ] OQ11 decided and recorded in ROADMAP before wave 1 (this runbook does not start without it)

Post-cut: annotate the removal inventory (class 5 shims: schedule the
one-cycle-later shim deletion NOW, in the v5.x train notes, or it leaks into
v6).

## See also

- [v5-removal-inventory](../guides/v5-removal-inventory.md) — per-item truth
- [ADR-0047](../adr/0047-re-export-layer-retirement-plan.md),
  [ADR-0046](../adr/0046-drop-websocket-sse-only.md),
  [ADR-0051](../adr/) — decision records behind the classes
- [declarative-projections.md](../guides/declarative-projections.md) — the
  NewProjectionLayer replacement + checkpoint caveat
