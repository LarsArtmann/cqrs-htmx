# Dependency Train Bump — Playbook

> How to sweep a family dependency (cqrs-htmx, go-cqrs-lite, templ-components, httputil, go-sse, …) to a new published version across this 28-module workspace. Companion to `release-next-train-prep.md` (cutting OUR train) — this runbook is for consuming SOMEONE ELSE'S.

**Automation:** `scripts/bump-dep.sh <module-pattern> <version>` does the mechanical sweep (see below). The verification loop stays manual — it is where the mistakes hide.

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
- **Capture exit codes without pipes:** `cmd > /tmp/<repo>-sweep-$$-$(date +%s).log 2>&1; echo $?` — `PIPESTATUS` is broken in mvdan/sh, and a redirect through a nonexistent dir silently skips the command and the rc lies.
- **No `set -e` in batch loops** — use explicit `rc=$?` checks after every module.
- **Mirror CI flags before pushing:** `nix run .#check-release-train -- --refresh-cache --strict-lag 0` (CI enforces strict-lag; the local advisory mode exits 0 on train lag).
- **`go work sync` policy:** NEVER run repo-wide `go work sync` during a sweep — it rewrites every member's go.mod from workspace state and can re-introduce versions you just removed, or bump unrelated requires (the "toolchain tug-of-war" class). The sweep script edits go.mod files directly; the workspace resolves via go.work `use` directives anyway. `go work sync` is only acceptable in a dedicated, tree-clean, single-purpose change where the diff is reviewed module by module.

## Failure classes seen in the wild

| Symptom | Root cause | Fix |
| --- | --- | --- |
| `invalid package name: ""` in unrelated modules | VCS-cache bare repo lost its `origin` remote (GOPRIVATE git resolution) | `git -C /mnt/buildcache/go-mod/cache/vcs/<hash>/ remote add origin https://github.com/larsartmann/<repo>.git` |
| Phantom `missing go.sum entry` everywhere at once | Cache filesystem full | `df -h` the cache dir; `go-cache-env.sh` fails fast on this |
| Just-published tag reads UNPUBLISHED | release-train gate's ls-remote tag cache TTL | re-run with `-- --refresh-cache` |
| go.mod `go` directive creeps up | A 1.27.1-toolchain process ran `go mod tidy` (sibling session / gopls) | restore the floor; `go-cache-env.sh` now aligns `GOTOOLCHAIN` for gate runs |

## scripts/bump-dep.sh

```bash
scripts/bump-dep.sh 'larsartmann/go-cqrs-lite' v4.14.0        # sweep a train
scripts/bump-dep.sh 'larsartmann/cqrs-htmx' v4.12.0 --dry-run # plan only
```

Digit-safe module matching, hermetic per-module tidy+build+vet, absence assertion at the end, and a per-module PASS/FAIL table. It refuses to run with a dirty tree (the daemon race makes interleaved sweeps unreviewable).
