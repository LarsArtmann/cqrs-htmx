# Release v4.8.0 Push Plan — Coordinated Family Release

> Status: EXECUTED 2026-08-14 — master (`f3d285fb`) and all 10 tags are pushed
> and verified on origin; §4 items 1-6 + drift alignment are done (§3 upstream
> tags and §4 items 7-8 remain open).
> All tags are annotated (SSH-signed), point at verified-green commits, and carry
> go.mod files with ZERO local `replace` directives (filesystem replaces in
> published tags break every consumer).
>
> 2026-08-14 (later): the 17 then-local commits were rewritten (filter-branch) to
> purge the accidentally committed `examples/setup-demo/setup-demo` binary
> (27 MB). Final trees are byte-identical to before, commit messages preserved;
> the 6 in-range tags were re-pointed and re-signed (SHAs in §1 are
> post-rewrite). The 4 tags on `73ff1556` were not affected.

## 1. Tag inventory (all local, unpushed)

| Tag                        | Commit     | Notes                                                                                                                                                                                                                                          |
| -------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `v4.8.0` (root)            | `73ff1556` | BREAKING: `NewActorID`, local `ActorID`, `MetadataKeyActorID` removed (ADR-0111). Adds `ProjectionReadinessCheck`, `RecommendedSecurityMiddleware`, `openapi/`, `event.WithActor` propagation, sync handlers, httputil re-export deprecations. |
| `identity-model/v4.8.0`    | `73ff1556` | BREAKING: ActorID → `id.ActorID`; `ParseActorID` returns `(ActorID, error)`. Authz/session kind guards (security). Requires only published go-cqrs-lite (event/command v4.6.0, id v4.4.0).                                                     |
| `usermgmt/v4.8.0`          | `5be86547` | BREAKING: `ParseActorID` arity. MySQL stores, `ReadModelDialect`, `AsyncStartup`, `AuditEntry.ActorID`, cascades, state cache. Requires identity-model **v4.8.0**, root v4.7.0.                                                                |
| `usermgmt/webauthn/v4.8.0` | `73ff1556` | Deps alignment only.                                                                                                                                                                                                                           |
| `loginpage/v4.8.0`         | `73ff1556` | Deps alignment only (requires usermgmt v4.7.2, root v4.7.0 — both published; still resolvable after the family push).                                                                                                                          |
| `dashboardui/v4.8.0`       | `3383dcd6` | SSE heartbeat data-race fix, nil-safe `Close()`, Actor ID display. Requires root **v4.8.0**.                                                                                                                                                   |
| `adminui/v4.8.0`           | `548df9fd` | Direct identity-model imports (v5 prerequisite, ADR-0047); SA1019 exclusion removed. Requires root/usermgmt/identity-model **v4.8.0**.                                                                                                         |
| `setup/v4.8.0`             | `f91ee4db` | **First tag.** One-call composition root; requires whole family at v4.8.0.                                                                                                                                                                     |
| `health/v4.8.0`            | `7fce808e` | **First tag.** go-health bridge; requires root v4.7.0 (published) — no replaces.                                                                                                                                                               |
| `auditlog/v4.8.0`          | `7fce808e` | **First tag.** samber-do-auditlog bridge; no cqrs-htmx deps.                                                                                                                                                                                   |

Not in this train (intentionally):

- **totp / oauth2**: zero commits since v4.7.0 — no tags needed; consumers keep resolving v4.7.0.
- **datastar/v4.8.0**: BLOCKED upstream — `datastar/go.mod` requires `go-datastar/static v0.2.0`, but the newest tag in the go-datastar repo is `static/v0.1.0`. Tag `static/v0.2.0` (and `v0.2.0` if not pushed) there first, then cut `datastar/v4.8.0` by stripping its 3 dev replaces (`go-datastar`, `go-sse`, `go-datastar/static`).
- **systemadapter/v4.8.0**: BLOCKED upstream — requires unreleased go-cqrs-lite metaengine APIs (see §3).

## 2. Push order (strict; module graph dependencies)

```bash
git push origin master   # DONE 2026-08-14 (f3d285fb)
git push origin v4.8.0 identity-model/v4.8.0 usermgmt/v4.8.0 \
              usermgmt/webauthn/v4.8.0 loginpage/v4.8.0 dashboardui/v4.8.0 \
              adminui/v4.8.0 setup/v4.8.0 health/v4.8.0 auditlog/v4.8.0
# DONE 2026-08-14 — all 10 tags pushed and verified via ls-remote (peeled
# SHAs match §1). §4 items 1-6 + the drift-alignment pass are also done.
```

Order within the push does not matter (the proxy resolves per-module), but if
you push incrementally: identity-model → usermgmt → root → the rest.

Known interop note (no action needed, no retraction): published
`usermgmt/v4.7.2` wraps the 1-value `ParseActorID`, so it only compiles against
identity-model ≤ v4.2.0. It is self-consistent in isolation; once v4.8.0 tags
are public, MVS pulls consumers to `usermgmt/v4.8.0` which requires
`identity-model/v4.8.0` — the ecosystem heals. See failure-modes notes in the
go-release skill before re-tagging anything: **tags are immutable once pushed;
if anything is wrong, cut a new patch/minor, never re-tag.**

## 3. Upstream tag requests (other repos, your call)

### go-cqrs-lite (sibling checkout `/home/lars/projects/go-cqrs-lite`)

Needed to strip systemadapter's 2 remaining go-cqrs-lite replaces
(`EventWithID.OccurredAt` in projectionadapter; self-registering `sqlite`
driver via `register.go` in sqliteengine) and the matching one in
`examples/system-demo`:

| Tag to cut                               | Base commit                                                                              | Prep                                                                                                                                                                                                   |
| ---------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `metaengine/projectionadapter/v4 v4.5.0` | `fe017c06a` (HEAD at mapping time; any later commit containing `typed_decoder.go` works) | **Strip the workspace `replace (...)` block** from `metaengine/projectionadapter/go.mod` before tagging — its requires (event v4.6.0, id v4.4.0, metaengine v4.10.0, record v4.2.0) are all published. |
| `metaengine/sqliteengine/v4 v4.0.2`      | `fe017c06a`                                                                              | **Strip** `replace metaengine/v4 => ../` from `metaengine/sqliteengine/go.mod`; requires metaengine v4.10.0 + record v4.2.0 (published).                                                               |

Note: go-cqrs-lite working tree had uncommitted changes (`.golangci.yml`,
`docs/api_surface.txt`, `system/system_hardening_test.go` — a concurrent
session); neither of the two modules above is affected by them.

### go-datastar (sibling checkout `/home/lars/projects/go-datastar`)

- Tag `static/v0.2.0` (newest existing: `static/v0.1.0`) → unblocks
  `datastar/v4.8.0`. **DONE 2026-08-15 — user tagged and pushed `static/v0.2.0`
  (repo at `5b70bb1`, synced).**

## 4. Post-push replace-strip checklist (per module)

For each: `GOWORK=off go mod tidy && go build ./... && go vet ./...`
(`go vet` is load-bearing — tags break in `_test.go` imports that
`go build` never compiles; this exact trap bit the 18:30 session).

1. `usermgmt` — strip identity-model replace (comment documents this). **DONE 2026-08-14.**
2. `adminui` — strip root/usermgmt/identity-model replaces. **DONE.**
3. `setup` — strip all six family replaces. **DONE.**
4. `integration_test` — strip root/usermgmt/identity-model/adminui/dashboardui/loginpage (keep datastar + go-datastar/go-sse until those tags exist; the previous failed attempt at the usermgmt strip was blocked ONLY by identity-model v4.8.0 being unpublished — now resolved by this train). **DONE (health replace stripped too).**
5. `examples/setup-demo` — strip all seven (incl. setup) once pushed. **DONE.**
6. `examples/dashboard-demo` — strip dashboardui + root. **DONE.**
7. `systemadapter` + `examples/system-demo` — strip the two go-cqrs-lite metaengine replaces after §3 tags are pushed; bump requires to v4.5.0 / v4.0.2. _(Not done — blocked on §3 upstream tags; their replaces are now RELATIVE paths so the absolute-path gate leg passes.)_
8. `datastar`, `examples/datastar-demo` — strip go-datastar replaces after §3 go-datastar tags. **DONE 2026-08-15 (local): upstream replaces stripped in `datastar` (all 3; zero replaces remain), `examples/datastar-demo` (3 upstream; the `cqrs-htmx/datastar/v4 => ../../datastar` family replace stays until the tag is pushed), `integration_test` (3 upstream), `health` (static replace + its comment block) — commit `f128072d`; local signed tag `datastar/v4.8.0` cut at that commit. REMAINING: push master + tag; then strip the 2 datastar-local family replaces (`examples/datastar-demo`, `integration_test`) and verify the proxy serves the tag.**

Also after the push: re-run `nix run .#check-modules` — its
`check-version-drift --strict` leg should now pass (the known cross-tag drift
was exactly this unpublished family). **DONE — plus follow-ups the drift gate
surfaced and that are now fixed (all hermetically verified per module):**

- Local go.mods bumped to the pushed family versions: root v4.8.0 in
  usermgmt, loginpage, health, systemadapter, e2e/server, and 7 examples;
  usermgmt v4.8.0 in loginpage, admin-demo, samber-do-demo; adminui v4.8.0 +
  identity-model v4.8.0 in admin-demo/samber-do-demo; catalog v4.2.1,
  snapshot v4.3.0, metaengine v4.10.0, storage v4.6.0, templ-components
  v1.8.1, go-sse v0.5.0 stragglers aligned.
- `health` gained one documented TEMPORARY replace
  (`go-datastar/static => ../../go-datastar/static`): the published go-datastar
  v0.2.0 tag requires static v0.2.0, which is not tagged upstream — hermetic
  builds cannot resolve it otherwise. Removal condition: go-datastar
  `static/v0.2.0` tagged.
- `systemadapter` + `examples/system-demo` metaengine replaces converted from
  absolute (`/home/lars/...`) to relative (`../../go-cqrs-lite/...`) paths —
  same targets, and the check-modules absolute-path leg passes.
- `dashboardui`'s own root dev-replace was also stripped (its removal condition
  was met; it was missing from this checklist).

## 5. Why the versions are what they are

- The buildflow `gomod-check` step enforces **one coordinated family version**
  across cqrs-htmx modules (observed empirically: it rewrote every family
  require to v4.8.0 the moment a bare `v4.8.0` tag existed). Tag history
  confirms coordinated cuts (v4.7.0, v4.6.1, v4.6.0...). This train follows it.
- Root's bump is v4.7.0 → v4.8.0 (minor) because it REMOVES exported API
  (`NewActorID` et al.) — house convention ships breaking changes in minor
  bumps (go-cqrs-lite precedent: event/v4.5.0 removed `MarkTombstone`).
- The earlier plan's "identity-model v4.7.0 / usermgmt v4.7.3 / root v4.7.1"
  numbers predate the discovery of both the family-alignment tooling and the
  root breaking changes — superseded by this document.
