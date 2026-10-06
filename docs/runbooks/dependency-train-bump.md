# Dependency Train Bump — Playbook

> How to sweep a family dependency (cqrs-htmx, go-cqrs-lite, templ-components, httputil, go-sse, …) to a new published version across this 28-module workspace. Companion to `release-next-train-prep.md` (cutting OUR train) — this runbook is for consuming SOMEONE ELSE'S.

**Automation:** `scripts/tools/bump-dep.sh <module-pattern> <version>` does the mechanical sweep (see below). The verification loop stays manual — it is where the mistakes hide.

## The wave order (learned the hard way)

Bump in dependency order so every hermetic build resolves:

1. Upstream-first deps (go-cqrs-lite trains, httputil, go-sse, templ-components, go-datastar).
2. cqrs-htmx family internals (identity-model → root/auth-strategies → usermgmt → UI modules → setup → systemadapter/health/auditlog).
3. Consumers (integration_test, e2e/server, examples/*) last.

## The sweep loop (per module)

```bash
source scripts/lib/go-cache-env.sh        # toolchain floor + cache guards
cd <module>
GOWORK=off go mod edit -require=<dep>@<version>   # or: go get <dep>@<version>
GOWORK=off go mod tidy && GOWORK=off go build ./... && GOWORK=off go vet ./...
```

**`go vet` is load-bearing** — `go build` does not compile `_test.go` files; test-only API drift has twice broken master while builds passed.

## Verification discipline

- **Assert absence, never sample presence.** After the sweep:
  `grep -rE '<dep>.*v<old-version>' --include=go.mod .` must print NOTHING.
  (A regex like `[a-z/-]+` that lacks digits silently misses `oauth2`-style names — write the digit-safe pattern.)
- **Validate the TARGET before aligning (R2/R18, the v1.20.0 lesson).** A tag
  whose go.mod carries placeholder pseudo-version requires
  (`vX.Y.Z-00010101000000-000000000000` — templ-components v1.20.0) is
  UNCONSUMABLE: every downstream sweep dies with a misleading `unknown
  revision`. bump-dep now pre-flights every module it is about to bump via
  `scripts/checks/check-family-release-consumable.sh` (proxy fetch, root +
  same-family submodule walk) and aborts with "upstream release … is
  unconsumable (placeholder sub-requires)" BEFORE touching anything. Run the
  checker manually for one-off checks:
  `nix run .#check-family-release-consumable -- github.com/larsartmann/templ-components v1.20.1`.
- **Capture exit codes without pipes:** `cmd > /tmp/<repo>-sweep-$$-$(date +%s).log 2>&1; echo $?` — `PIPESTATUS` is broken in mvdan/sh, and a redirect through a nonexistent dir silently skips the command and the rc lies.
- **No `set -e` in batch loops** — use explicit `rc=$?` checks after every module.
- **Mirror CI flags before pushing:** `nix run .#check-release-train -- --refresh-cache --strict-lag 0` (CI enforces strict-lag; the local advisory mode exits 0 on train lag).
- **`go work sync` policy:** NEVER run repo-wide `go work sync` during a sweep — it rewrites every member's go.mod from workspace state and can re-introduce versions you just removed, or bump unrelated requires (the "toolchain tug-of-war" class). The sweep script edits go.mod files directly; the workspace resolves via go.work `use` directives anyway. `go work sync` is only acceptable in a dedicated, tree-clean, single-purpose change where the diff is reviewed module by module.

## Failure classes seen in the wild

| Symptom                                           | Root cause                                                              | Fix                                                                                                           |
| ------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `invalid package name: ""` in unrelated modules   | VCS-cache bare repo lost its `origin` remote (GOPRIVATE git resolution) | `git -C /mnt/buildcache/go-mod/cache/vcs/<hash>/ remote add origin https://github.com/larsartmann/<repo>.git` |
| Phantom `missing go.sum entry` everywhere at once | Cache filesystem full                                                   | `df -h` the cache dir; `go-cache-env.sh` fails fast on this                                                   |
| Just-published tag reads UNPUBLISHED              | release-train gate's ls-remote tag cache TTL                            | re-run with `-- --refresh-cache`                                                                              |
| go.mod `go` directive creeps up                   | A 1.27.1-toolchain process ran `go mod tidy` (sibling session / gopls)  | restore the floor; `go-cache-env.sh` now aligns `GOTOOLCHAIN` for gate runs                                   |

## scripts/tools/bump-dep.sh

```bash
scripts/tools/bump-dep.sh 'larsartmann/go-cqrs-lite' v4.14.0          # sweep a train
scripts/tools/bump-dep.sh 'larsartmann/cqrs-htmx' v4.12.0 --dry-run   # plan only (no network)
scripts/tools/bump-dep.sh 'larsartmann/httputil$' v1.3.0 --commit      # sweep + commit as one
```

Digit-safe module matching (accepts BOTH the block-require form and the
single-line `require <mod> <ver>` form — the 9b3c2e18 gap), hermetic per-module
tidy + `go mod verify` + build + vet, absence assertion at the end, and a
per-module PASS/FAIL table. It refuses to run with a dirty tree (the daemon race
makes interleaved sweeps unreviewable). `--commit` stages + commits the sweep in
the same process so the auto-commit daemon cannot shred the sweep and its
verification into separate commits; `--no-verify` skips the `go mod verify` step.
`BUMP_DEP_ROOT` overrides the scan root for the fixture self-test
(`bash scripts/selftests/test-bump-dep.sh`). `BUMP_DEP_NO_NETWORK=1` skips the
consumability pre-flight (offline fixtures). The consumability pre-flight
(R18) validates the TARGET release before the first mutation and aborts with
the named upstream fault.

**Moving-target stop-rule.** If new family tags land WHILE you are aligning
(the upstream released mid-sweep), STOP after the current module: re-run the
pre-flight for the new tag, decide explicitly (finish the old train, or restart
on the new one), and never let one sweep mix two target versions — the absence
assertion only knows the version you started with.

**Never chain bump-dep invocations without committing between them.** MVS is
transitive: sweeping dep A can already raise a sibling dep B in the same graph,
so a second sweep begun before the first is committed can carry B's bump along
invisibly (the v4.13.x `async-startup-demo` hand-fix class). One sweep → verify →
commit → next sweep.
